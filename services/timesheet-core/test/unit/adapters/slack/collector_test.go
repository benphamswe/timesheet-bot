package slack_test

// NewCollector takes a real *slack.Client, so these tests build the Collector
// with NewCollectorWithSearcher around a fake Searcher.

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/slack-go/slack"

	slackactivity "timesheet-bot/internal/activity/infrastructure/slack"
	"timesheet-bot/internal/shared/kernel"
)

var week41 = kernel.WeekID{Year: 2026, Week: 41}

// pageSize is the search.messages page size the Collector must request: 100,
// the most Slack returns per page.
const pageSize = 100

// fakeSearch serves pages in order and records every call. A non-nil err is
// returned from every call.
type fakeSearch struct {
	pages   [][]slack.SearchMessage
	err     error
	queries []string
	params  []slack.SearchParameters
}

func (f *fakeSearch) SearchMessagesContext(_ context.Context, q string, p slack.SearchParameters) (*slack.SearchMessages, error) {
	f.queries = append(f.queries, q)
	f.params = append(f.params, p)
	if f.err != nil {
		return nil, f.err
	}
	res := &slack.SearchMessages{Matches: f.pages[p.Page-1]}
	res.Pages = len(f.pages)
	return res, nil
}

func searchMessage(at time.Time, channel, text, permalink string) slack.SearchMessage {
	m := slack.SearchMessage{Timestamp: fmt.Sprintf("%d.000100", at.Unix()), Text: text, Permalink: permalink}
	m.Channel.ID, m.Channel.Name = "C"+channel, channel
	return m
}

func hoChiMinh(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatalf("LoadLocation(Asia/Ho_Chi_Minh): %v", err)
	}
	return loc
}

func workDay(t *testing.T, s string) kernel.WorkDay {
	t.Helper()
	d, err := kernel.ParseWorkDay(s)
	if err != nil {
		t.Fatalf("ParseWorkDay(%q): %v", s, err)
	}
	return d
}

func TestQueryCoversSundayToSaturdayExclusive(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		week kernel.WeekID
		want string
	}{
		{name: "mid year", week: week41, want: "from:<@U1> after:2026-10-04 before:2026-10-10"},
		{name: "week 1 starts in the previous year", week: kernel.WeekID{Year: 2026, Week: 1}, want: "from:<@U1> after:2025-12-28 before:2026-01-03"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := slackactivity.Query("U1", tt.week); got != tt.want {
				t.Errorf("Query(U1, %v) = %q, want %q", tt.week, got, tt.want)
			}
		})
	}
}

// AC-001.1
func TestCollectPagesAndLocalDays(t *testing.T) {
	t.Parallel()
	hcm := hoChiMinh(t)
	f := &fakeSearch{pages: [][]slack.SearchMessage{
		{
			// Monday 23:30 local is Monday 16:30 UTC.
			searchMessage(time.Date(2026, 10, 5, 23, 30, 0, 0, hcm), "proj-gcm", "GCM-12 fixed", "https://x.slack.com/archives/C/p1"),
			// Sunday 23:00 UTC is already Monday 06:00 local.
			searchMessage(time.Date(2026, 10, 4, 23, 0, 0, 0, time.UTC), "dev", "early start on HUB-3", ""),
		},
		{
			// Saturday local: dropped.
			searchMessage(time.Date(2026, 10, 10, 10, 0, 0, 0, hcm), "dev", "GCM-99 weekend", ""),
			searchMessage(time.Date(2026, 10, 9, 10, 0, 0, 0, hcm), "proj-hub", "reviewed",
				"https://x.slack.com/archives/C/p2?thread_ts=1728450000.000100&cid=C"),
		},
	}}
	c := slackactivity.NewCollectorWithSearcher(f, "U1", hcm)

	acts, err := c.Collect(t.Context(), week41)
	if err != nil {
		t.Fatalf("Collect(%v) error = %v, want nil", week41, err)
	}

	wantQuery := slackactivity.Query("U1", week41)
	if len(f.params) != 2 {
		t.Fatalf("Collect(%v) made %d search calls, want 2 (one per page)", week41, len(f.params))
	}
	for i, p := range f.params {
		if f.queries[i] != wantQuery || p.Page != i+1 || p.Count != pageSize || p.Sort != "timestamp" || p.SortDirection != "asc" {
			t.Errorf("search call %d = %q %+v, want %q page %d count %d sorted by timestamp asc",
				i+1, f.queries[i], p, wantQuery, i+1, pageSize)
		}
	}

	want := []kernel.WorkActivity{
		{
			Day: workDay(t, "2026-10-05"), ChannelID: "Cproj-gcm", ChannelName: "proj-gcm", Text: "GCM-12 fixed",
			Permalink: "https://x.slack.com/archives/C/p1", IssueKeys: []kernel.IssueKey{"GCM-12"},
		},
		{
			Day: workDay(t, "2026-10-05"), ChannelID: "Cdev", ChannelName: "dev", Text: "early start on HUB-3",
			IssueKeys: []kernel.IssueKey{"HUB-3"},
		},
		{
			Day: workDay(t, "2026-10-09"), ChannelID: "Cproj-hub", ChannelName: "proj-hub", Text: "reviewed",
			Permalink: "https://x.slack.com/archives/C/p2?thread_ts=1728450000.000100&cid=C", ThreadTS: "1728450000.000100",
		},
	}
	if !reflect.DeepEqual(acts, want) {
		t.Errorf("Collect(%v) =\n%+v\nwant\n%+v", week41, acts, want)
	}
}

func TestCollectSearchErrorIsWrapped(t *testing.T) {
	t.Parallel()
	errSlack := errors.New("ratelimited")
	c := slackactivity.NewCollectorWithSearcher(&fakeSearch{err: errSlack}, "U1", time.UTC)
	if got, err := c.Collect(t.Context(), week41); !errors.Is(err, errSlack) {
		t.Errorf("Collect(%v) = %v, %v, want an error wrapping %v", week41, got, err, errSlack)
	}
}

func TestCollectRejectsMalformedTimestamp(t *testing.T) {
	t.Parallel()
	bad := slack.SearchMessage{Timestamp: "not-a-ts", Text: "GCM-1"}
	c := slackactivity.NewCollectorWithSearcher(&fakeSearch{pages: [][]slack.SearchMessage{{bad}}}, "U1", time.UTC)
	if got, err := c.Collect(t.Context(), week41); !errors.Is(err, strconv.ErrSyntax) {
		t.Errorf("Collect(%v) = %v, %v, want an error wrapping strconv.ErrSyntax", week41, got, err)
	}
}

func TestCollectEmptyResult(t *testing.T) {
	t.Parallel()
	f := &fakeSearch{pages: [][]slack.SearchMessage{{}}}
	c := slackactivity.NewCollectorWithSearcher(f, "U1", time.UTC)
	acts, err := c.Collect(t.Context(), week41)
	if err != nil || len(acts) != 0 {
		t.Errorf("Collect(%v) = %v, %v, want no activities and nil", week41, acts, err)
	}
	if len(f.params) != 1 {
		t.Errorf("Collect(%v) made %d search calls, want 1", week41, len(f.params))
	}
}
