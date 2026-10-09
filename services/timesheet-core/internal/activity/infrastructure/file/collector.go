// Package file collects the user's Slack messages from the JSON files the
// allocation-agent collector writes, one per ISO week, for users who cannot
// create a Slack app. The file format is a contract shared with that service.
package file

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"timesheet-bot/internal/activity/domain"
	"timesheet-bot/internal/shared/kernel"
)

// SchemaVersion is the only activity file version this collector reads.
const SchemaVersion = 1

// Collector is a [domain.Source] that reads <dir>/<ISO week>.json.
type Collector struct {
	dir      string
	userID   string         // Slack member id the file must belong to
	location *time.Location // turns posted_at into calendar days
}

// NewCollector returns a Collector that reads the week files in dir, accepts
// only files for the Slack member userID, and assigns each message to a
// calendar day in loc.
func NewCollector(dir, userID string, loc *time.Location) *Collector {
	return &Collector{dir: dir, userID: userID, location: loc}
}

var _ domain.Source = (*Collector)(nil)

// Path returns the file that holds week's messages.
func (c *Collector) Path(week kernel.WeekID) string {
	return filepath.Join(c.dir, week.String()+".json")
}

type weekFile struct {
	SchemaVersion int       `json:"schema_version"`
	Week          string    `json:"week"`
	SlackUserID   string    `json:"slack_user_id"`
	Messages      []message `json:"messages"`
}

type message struct {
	TS          string  `json:"ts"`
	ChannelID   string  `json:"channel_id"`
	ChannelName string  `json:"channel_name"`
	PostedAt    string  `json:"posted_at"`
	Text        string  `json:"text"`
	Permalink   string  `json:"permalink"`
	ThreadTS    *string `json:"thread_ts"`
}

// Collect returns the messages in week's file that were posted on the
// working days of week in the collector's location. It fails when the file
// is missing (the error wraps [fs.ErrNotExist]), is not valid JSON, has
// another schema_version, or belongs to another week or user.
func (c *Collector) Collect(ctx context.Context, week kernel.WeekID) ([]kernel.WorkActivity, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path := c.Path(week)
	raw, err := os.ReadFile(path) //nolint:gosec // G304: the path is built from the operator's own activity.file_dir and a parsed week
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("activity file %s not found: run the allocation-agent collector for %s first: %w", path, week, err)
	}
	if err != nil {
		return nil, fmt.Errorf("read activity file %s: %w", path, err)
	}
	var f weekFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse activity file %s: %w", path, err)
	}
	switch {
	case f.SchemaVersion != SchemaVersion:
		return nil, fmt.Errorf("activity file %s: schema_version %d, want %d", path, f.SchemaVersion, SchemaVersion)
	case f.Week != week.String():
		return nil, fmt.Errorf("activity file %s: week %q, want %q", path, f.Week, week)
	case f.SlackUserID != c.userID:
		return nil, fmt.Errorf("activity file %s: slack_user_id %q, want slack_user_id %q from config", path, f.SlackUserID, c.userID)
	}

	var out []kernel.WorkActivity
	for i, m := range f.Messages {
		posted, err := time.Parse(time.RFC3339, m.PostedAt)
		if err != nil {
			return nil, fmt.Errorf("activity file %s: messages[%d] (ts %s): posted_at: %w", path, i, m.TS, err)
		}
		if a, ok := domain.ToActivity(domain.Message{
			PostedAt:    posted,
			ChannelID:   m.ChannelID,
			ChannelName: m.ChannelName,
			Text:        m.Text,
			Permalink:   m.Permalink,
			ThreadTS:    replyThreadTS(m),
		}, week, c.location); ok {
			out = append(out, a)
		}
	}
	return out, nil
}

// replyThreadTS returns thread_ts only for a thread reply. Slack sets
// thread_ts on a thread's parent too, equal to its own ts; search.messages
// gives the parent no thread_ts, so neither does this collector.
func replyThreadTS(m message) string {
	if m.ThreadTS == nil || *m.ThreadTS == m.TS {
		return ""
	}
	return *m.ThreadTS
}
