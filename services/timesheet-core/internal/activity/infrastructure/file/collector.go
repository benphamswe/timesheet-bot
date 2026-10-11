// Package file collects the user's Slack messages, and in version 2 the
// report lines an LLM parsed from them, from the JSON files the
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
	"strings"
	"time"

	"timesheet-bot/internal/activity/domain"
	"timesheet-bot/internal/shared/kernel"
)

// Activity file versions this collector reads. Version 2 is version 1 plus
// `parser` and `report_lines`, the timesheet summary lines the collector's
// LLM parsed (report.parser: llm).
const (
	SchemaVersionMessages    = 1
	SchemaVersionReportLines = 2
)

// errNoReportLines is wrapped by [Collector.ReportLines] when the file has no
// parsed lines.
var errNoReportLines = errors.New("activity file has no report_lines: re-run the collector with report.parser: llm")

// Collector is a [domain.Source] that reads <dir>/<ISO week>.json. It also
// provides the file's precomputed report lines (version 2).
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
	SchemaVersion int           `json:"schema_version"`
	Week          string        `json:"week"`
	SlackUserID   string        `json:"slack_user_id"`
	Messages      []message     `json:"messages"`
	Parser        *parserInfo   `json:"parser"`       // version 2
	ReportLines   *[]reportLine `json:"report_lines"` // version 2; nil when absent
}

// parserInfo says how report_lines were made. It is read for completeness;
// nothing depends on it.
type parserInfo struct {
	Kind     string `json:"kind"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type reportLine struct {
	Date            string `json:"date"`
	Project         string `json:"project"`
	Minutes         int    `json:"minutes"`
	Notes           string `json:"notes"`
	SourceTS        string `json:"source_ts"`
	SourceChannelID string `json:"source_channel_id"`
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
// is missing (the error wraps [fs.ErrNotExist]), is not valid JSON, has a
// schema_version other than 1 or 2, or belongs to another week or user.
func (c *Collector) Collect(ctx context.Context, week kernel.WeekID) ([]kernel.WorkActivity, error) {
	f, path, err := c.load(ctx, week)
	if err != nil {
		return nil, err
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

// ReportLines returns the report lines the collector's LLM parsed for week
// (version 2), in file order. It fails like [Collector.Collect], and also
// when the file is version 1 or has no report_lines, or at the first line
// whose minutes are not a positive multiple of 15, whose date is not a
// Monday-Friday of week, or whose project is empty. Errors name the line's
// source_ts, never its notes.
func (c *Collector) ReportLines(ctx context.Context, week kernel.WeekID) ([]kernel.ReportLine, error) {
	f, path, err := c.load(ctx, week)
	if err != nil {
		return nil, err
	}
	if f.SchemaVersion < SchemaVersionReportLines || f.ReportLines == nil {
		return nil, fmt.Errorf("%s (version %d): %w", path, f.SchemaVersion, errNoReportLines)
	}
	out := make([]kernel.ReportLine, 0, len(*f.ReportLines))
	for i, l := range *f.ReportLines {
		if err := validLine(l, week); err != nil {
			return nil, fmt.Errorf("activity file %s: report_lines[%d] (source_ts %s): %w", path, i, l.SourceTS, err)
		}
		out = append(out, kernel.ReportLine{
			Date: l.Date, Project: strings.TrimSpace(l.Project), Minutes: l.Minutes, Notes: l.Notes,
			SourceTS: l.SourceTS, SourceChannelID: l.SourceChannelID,
		})
	}
	return out, nil
}

// validLine checks a parsed line again; the collector checks the same rules
// before it writes the file. The error never includes the notes.
func validLine(l reportLine, week kernel.WeekID) error {
	if _, err := kernel.NewDuration(l.Minutes); err != nil {
		return fmt.Errorf("minutes %d: %w", l.Minutes, err)
	}
	day, err := kernel.ParseWorkDay(l.Date)
	if err != nil {
		return fmt.Errorf("date %q: %w", l.Date, err)
	}
	if !week.Contains(day) {
		return fmt.Errorf("date %s is outside %s", l.Date, week)
	}
	if strings.TrimSpace(l.Project) == "" {
		return errors.New("empty project")
	}
	return nil
}

// load reads, decodes and checks week's file.
func (c *Collector) load(ctx context.Context, week kernel.WeekID) (weekFile, string, error) {
	if err := ctx.Err(); err != nil {
		return weekFile{}, "", err
	}
	path := c.Path(week)
	raw, err := os.ReadFile(path) //nolint:gosec // G304: the path is built from the operator's own activity.file_dir and a parsed week
	if errors.Is(err, fs.ErrNotExist) {
		return weekFile{}, path, fmt.Errorf("activity file %s not found: run the allocation-agent collector for %s first: %w", path, week, err)
	}
	if err != nil {
		return weekFile{}, path, fmt.Errorf("read activity file %s: %w", path, err)
	}
	var f weekFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return weekFile{}, path, fmt.Errorf("parse activity file %s: %w", path, err)
	}
	switch {
	case f.SchemaVersion != SchemaVersionMessages && f.SchemaVersion != SchemaVersionReportLines:
		return weekFile{}, path, fmt.Errorf("activity file %s: schema_version %d, want %d or %d",
			path, f.SchemaVersion, SchemaVersionMessages, SchemaVersionReportLines)
	case f.Week != week.String():
		return weekFile{}, path, fmt.Errorf("activity file %s: week %q, want %q", path, f.Week, week)
	case f.SlackUserID != c.userID:
		return weekFile{}, path, fmt.Errorf("activity file %s: slack_user_id %q, want slack_user_id %q from config", path, f.SlackUserID, c.userID)
	}
	return f, path, nil
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
