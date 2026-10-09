// Package slack collects the user's own messages via Slack search.messages.
// search.messages only works with a user token (xoxp-) with the search:read scope.
package slack

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"

	"timesheet-bot/internal/activity/domain"
	"timesheet-bot/internal/shared/kernel"
)

const pageSize = 100

// Searcher is the one Slack Web API method the Collector calls. *slack.Client
// implements it.
type Searcher interface {
	SearchMessagesContext(ctx context.Context, query string, params slack.SearchParameters) (*slack.SearchMessages, error)
}

// Collector is a [domain.Source] that finds the user's own Slack messages for a
// week with search.messages.
type Collector struct {
	api      Searcher
	userID   string         // Slack member id, e.g. U012ABC
	location *time.Location // turns message timestamps into calendar days
}

// NewCollector returns a Collector that searches as api for messages from the
// Slack member userID and assigns each message to a calendar day in loc.
func NewCollector(api *slack.Client, userID string, loc *time.Location) *Collector {
	return NewCollectorWithSearcher(api, userID, loc)
}

// NewCollectorWithSearcher is [NewCollector] for any [Searcher], such as a
// fake that serves canned search results.
func NewCollectorWithSearcher(s Searcher, userID string, loc *time.Location) *Collector {
	return &Collector{api: s, userID: userID, location: loc}
}

var _ domain.Source = (*Collector)(nil)

// Query builds the search for one week. Slack's after/before are exclusive dates.
func Query(userID string, week kernel.WeekID) string {
	days := week.Days()
	sunday := days[0].Time().AddDate(0, 0, -1)
	saturday := days[4].Time().AddDate(0, 0, 1)
	return fmt.Sprintf("from:<@%s> after:%s before:%s", userID, sunday.Format(time.DateOnly), saturday.Format(time.DateOnly))
}

// Collect returns the user's messages from the working days of week, reading
// every page of search results. Messages that fall outside the week in the
// collector's location are skipped.
func (c *Collector) Collect(ctx context.Context, week kernel.WeekID) ([]kernel.WorkActivity, error) {
	params := slack.NewSearchParameters()
	params.Sort = "timestamp"
	params.SortDirection = "asc"
	params.Count = pageSize

	var out []kernel.WorkActivity
	for page := 1; ; page++ {
		params.Page = page
		res, err := c.api.SearchMessagesContext(ctx, Query(c.userID, week), params)
		if err != nil {
			return nil, fmt.Errorf("slack search page %d: %w", page, err)
		}
		for _, m := range res.Matches {
			a, ok, err := c.toActivity(m, week)
			if err != nil {
				return nil, err
			}
			if ok {
				out = append(out, a)
			}
		}
		if page >= res.Pages {
			return out, nil
		}
	}
}

func (c *Collector) toActivity(m slack.SearchMessage, week kernel.WeekID) (kernel.WorkActivity, bool, error) {
	ts, err := parseTS(m.Timestamp)
	if err != nil {
		return kernel.WorkActivity{}, false, err
	}
	a, ok := domain.ToActivity(domain.Message{
		PostedAt:    ts,
		ChannelID:   m.Channel.ID,
		ChannelName: m.Channel.Name,
		Text:        m.Text,
		Permalink:   m.Permalink,
	}, week, c.location)
	return a, ok, nil
}

// parseTS converts a Slack "1728450000.123456" timestamp.
func parseTS(s string) (time.Time, error) {
	secs, _, _ := strings.Cut(s, ".")
	n, err := strconv.ParseInt(secs, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("bad slack ts %q: %w", s, err)
	}
	return time.Unix(n, 0), nil
}
