package file_test

// Fixtures in testdata/<case>/2026-W41.json follow the activity file contract
// the allocation-agent writes. Their ts and posted_at describe the same
// instant, so the Slack collector can be fed the same messages for parity.

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/slack-go/slack"

	fileactivity "timesheet-bot/internal/activity/infrastructure/file"
	slackactivity "timesheet-bot/internal/activity/infrastructure/slack"
	"timesheet-bot/internal/shared/kernel"
)

const userID = "U0123ABCD"

var week41 = kernel.WeekID{Year: 2026, Week: 41}

func hoChiMinh(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatalf("LoadLocation(Asia/Ho_Chi_Minh): %v", err)
	}
	return loc
}

func day(t *testing.T, s string) kernel.WorkDay {
	t.Helper()
	d, err := kernel.ParseWorkDay(s)
	if err != nil {
		t.Fatalf("ParseWorkDay(%s): %v", s, err)
	}
	return d
}

func collect(t *testing.T, fixture string) ([]kernel.WorkActivity, error) {
	t.Helper()
	c := fileactivity.NewCollector(filepath.Join("testdata", fixture), userID, hoChiMinh(t))
	return c.Collect(t.Context(), week41)
}

func TestCollectReadsWeekFile(t *testing.T) {
	t.Parallel()
	got, err := collect(t, "happy")
	if err != nil {
		t.Fatalf("Collect(%v) error = %v, want nil", week41, err)
	}
	const parent = "1791252000.000100"
	want := []kernel.WorkActivity{
		{
			// 00:30 on Monday in Ho Chi Minh City is still Sunday in UTC.
			Day: day(t, "2026-10-05"), ChannelID: "C01", ChannelName: "proj-x", Text: "standup GCM-1",
			Permalink: "https://example.slack.com/archives/C01/p1791135000000100",
			IssueKeys: []kernel.IssueKey{"GCM-1"},
		},
		{
			// A thread parent: thread_ts equals its own ts, so it is not a reply.
			Day: day(t, "2026-10-06"), ChannelID: "C02", ChannelName: "proj-hub", Text: "kickoff HUB-2",
			Permalink: "https://example.slack.com/archives/C02/p1791252000000100",
			IssueKeys: []kernel.IssueKey{"HUB-2"},
		},
		{
			Day: day(t, "2026-10-06"), ChannelID: "C02", ChannelName: "proj-hub", Text: "worked on HUB-2 and GCM-1",
			Permalink: "https://example.slack.com/archives/C02/p1791256500000100?thread_ts=" + parent + "&cid=C02",
			ThreadTS:  parent, IssueKeys: []kernel.IssueKey{"HUB-2", "GCM-1"},
		},
		{
			Day: day(t, "2026-10-07"), ChannelID: "C03", ChannelName: "random", Text: "lunch",
			Permalink: "https://example.slack.com/archives/C03/p1791356400000100",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Collect(%v) =\n%+v\nwant\n%+v", week41, got, want)
	}
}

func TestCollectSkipsMessagesOutsideTheWeek(t *testing.T) {
	t.Parallel()
	got, err := collect(t, "happy")
	if err != nil {
		t.Fatalf("Collect(%v) error = %v, want nil", week41, err)
	}
	for _, a := range got {
		// GCM-9 was posted on Sunday; GCM-8 at 23:30 UTC Friday, which is Saturday in Ho Chi Minh City.
		if strings.Contains(a.Text, "GCM-9") || strings.Contains(a.Text, "GCM-8") {
			t.Errorf("Collect(%v) kept %q, want messages outside the working week skipped", week41, a.Text)
		}
	}
}

func TestCollectMissingFileNamesPathAndCollector(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	c := fileactivity.NewCollector(dir, userID, hoChiMinh(t))

	_, err := c.Collect(t.Context(), week41)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Collect() error = %v, want fs.ErrNotExist", err)
	}
	want := filepath.Join(dir, "2026-W41.json")
	if c.Path(week41) != want || !strings.Contains(err.Error(), want) {
		t.Errorf("Collect() error = %q, want it to name %s", err, want)
	}
	if !strings.Contains(err.Error(), "run the allocation-agent collector") {
		t.Errorf("Collect() error = %q, want it to say how to create the file", err)
	}
}

func TestCollectRejectsInvalidFiles(t *testing.T) {
	t.Parallel()
	tests := []struct {
		fixture string
		want    string // part of the error, after the file's path
	}{
		{fixture: "wrong-week", want: `week "2026-W40", want "2026-W41"`},
		{fixture: "wrong-user", want: `slack_user_id "U9999", want slack_user_id "U0123ABCD"`},
		{fixture: "schema-v2", want: "schema_version 2, want 1"},
		{fixture: "bad-json", want: "parse activity file"},
		{fixture: "bad-posted-at", want: "messages[0] (ts 1791135000.000100): posted_at"},
	}
	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			t.Parallel()
			got, err := collect(t, tt.fixture)
			if err == nil {
				t.Fatalf("Collect(%s) = %d activities, want an error", tt.fixture, len(got))
			}
			path := filepath.Join("testdata", tt.fixture, "2026-W41.json")
			if !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Collect(%s) error = %q, want it to name %s and say %q", tt.fixture, err, path, tt.want)
			}
		})
	}
}

func TestCollectHonorsCanceledContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	c := fileactivity.NewCollector(filepath.Join("testdata", "happy"), userID, hoChiMinh(t))
	if _, err := c.Collect(ctx, week41); !errors.Is(err, context.Canceled) {
		t.Errorf("Collect(canceled ctx) error = %v, want context.Canceled", err)
	}
}

// fakeSearch serves one page of search.messages results.
type fakeSearch struct{ matches []slack.SearchMessage }

func (f fakeSearch) SearchMessagesContext(context.Context, string, slack.SearchParameters) (*slack.SearchMessages, error) {
	return &slack.SearchMessages{Matches: f.matches, Paging: slack.Paging{Pages: 1}}, nil
}

// The same messages must become the same activities whichever source read
// them, so allocation does not depend on activity.source.
func TestCollectMatchesSlackCollector(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(filepath.Join("testdata", "happy", "2026-W41.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var f struct {
		Messages []struct {
			TS          string `json:"ts"`
			ChannelID   string `json:"channel_id"`
			ChannelName string `json:"channel_name"`
			Text        string `json:"text"`
			Permalink   string `json:"permalink"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	var matches []slack.SearchMessage
	for _, m := range f.Messages {
		sm := slack.SearchMessage{Timestamp: m.TS, Text: m.Text, Permalink: m.Permalink}
		sm.Channel.ID, sm.Channel.Name = m.ChannelID, m.ChannelName
		matches = append(matches, sm)
	}

	fromSlack, err := slackactivity.NewCollectorWithSearcher(fakeSearch{matches: matches}, userID, hoChiMinh(t)).Collect(t.Context(), week41)
	if err != nil {
		t.Fatalf("slack Collect(%v) error = %v, want nil", week41, err)
	}
	fromFile, err := collect(t, "happy")
	if err != nil {
		t.Fatalf("file Collect(%v) error = %v, want nil", week41, err)
	}
	if len(fromFile) == 0 || !reflect.DeepEqual(fromFile, fromSlack) {
		t.Errorf("file Collect(%v) =\n%+v\nwant the Slack collector's\n%+v", week41, fromFile, fromSlack)
	}
}
