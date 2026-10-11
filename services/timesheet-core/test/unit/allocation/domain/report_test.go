package domain_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"timesheet-bot/internal/allocation/domain"
	"timesheet-bot/internal/shared/kernel"
)

var errParse = errors.New("cannot read line 3")

// stubParser returns the lines registered for each message text, and fails
// for the text in failOn. It records the time zone it was given.
type stubParser struct {
	lines  map[string][]domain.ReportLine
	failOn string
	tz     *time.Location
}

func (p *stubParser) Parse(msg kernel.WorkActivity, tz *time.Location) ([]domain.ReportLine, error) {
	p.tz = tz
	if msg.Text == p.failOn {
		return nil, errParse
	}
	return p.lines[msg.Text], nil
}

func reportInput(texts ...string) domain.Input {
	in := input()
	for _, t := range texts {
		in.Activities = append(in.Activities, kernel.WorkActivity{Day: mon, ChannelName: "timesheet", Text: t, Permalink: "https://x/" + t})
	}
	return in
}

func line(date, project string, minutes int, notes string) domain.ReportLine {
	return domain.ReportLine{Date: date, Project: project, Minutes: minutes, Notes: notes}
}

type entryView struct {
	Day      string
	Project  string
	Minutes  int
	Notes    string
	IssueKey kernel.IssueKey
}

func views(p domain.Proposal) []entryView {
	var out []entryView
	for _, e := range p.Entries {
		out = append(out, entryView{e.Day.String(), e.Project, e.Duration.Minutes(), e.Description, e.IssueKey})
	}
	return out
}

func TestReportPolicyOneEntryPerLine(t *testing.T) {
	t.Parallel()
	loc := time.FixedZone("ICT", 7*3600)
	parser := &stubParser{lines: map[string][]domain.ReportLine{
		"mon report": {line("2026-10-05", "Atlas", 240, "API review"), line("2026-10-05", " Atlas ", 60, "deploy")},
		"tue report": {line("2026-10-06", "Beacon", 90, "")},
	}}

	p, err := domain.ReportPolicy{Parser: parser, Location: loc}.Allocate(context.Background(), reportInput("mon report", "tue report", "chatter"))
	if err != nil {
		t.Fatalf("ReportPolicy.Allocate() error = %v, want nil", err)
	}
	want := []entryView{
		{Day: "2026-10-05", Project: "Atlas", Minutes: 240, Notes: "API review"},
		{Day: "2026-10-05", Project: "Atlas", Minutes: 60, Notes: "deploy"},
		{Day: "2026-10-06", Project: "Beacon", Minutes: 90},
	}
	if got := views(p); !slices.Equal(got, want) {
		t.Errorf("Allocate() entries = %+v, want %+v (one per line, no 45m rule, no channel default)", got, want)
	}
	if p.Policy != "report" || p.Reasoning["2026-10-05"] != "2 report line(s)" || len(p.Warnings) != 0 {
		t.Errorf("Allocate() = policy %q, reasoning %v, warnings %v, want report, 2 lines on Monday, no warnings", p.Policy, p.Reasoning, p.Warnings)
	}
	if parser.tz != loc {
		t.Errorf("parser got time zone %v, want the policy's %v", parser.tz, loc)
	}
}

// The owner posts the same summary in the report channel and the self-DM.
func TestReportPolicyDropsLinesPostedTwice(t *testing.T) {
	t.Parallel()
	parser := &stubParser{lines: map[string][]domain.ReportLine{
		"channel": {line("2026-10-05", "Atlas", 240, "API review"), line("2026-10-05", "Beacon", 60, "standup")},
		"self-dm": {line("2026-10-05", "atlas", 240, "API  Review "), line("2026-10-05", "Beacon", 60, "standup")},
	}}

	p, err := domain.ReportPolicy{Parser: parser}.Allocate(context.Background(), reportInput("channel", "self-dm"))
	if err != nil {
		t.Fatalf("ReportPolicy.Allocate() error = %v, want nil", err)
	}
	want := []entryView{
		{Day: "2026-10-05", Project: "Atlas", Minutes: 240, Notes: "API review"},
		{Day: "2026-10-05", Project: "Beacon", Minutes: 60, Notes: "standup"},
	}
	if got := views(p); !slices.Equal(got, want) {
		t.Errorf("Allocate() entries = %+v, want %+v (duplicates dropped, first kept)", got, want)
	}
}

func TestDedupReportLines(t *testing.T) {
	t.Parallel()
	base := line("2026-10-05", "Atlas", 60, "API review")
	tests := []struct {
		name  string
		other domain.ReportLine
		kept  int
	}{
		{name: "identical", other: base, kept: 1},
		{name: "project case and spaces", other: line("2026-10-05", "  ATLAS ", 60, "API review"), kept: 1},
		{name: "notes case and spaces", other: line("2026-10-05", "Atlas", 60, "api\treview "), kept: 1},
		{name: "other minutes", other: line("2026-10-05", "Atlas", 45, "API review"), kept: 2},
		{name: "other notes", other: line("2026-10-05", "Atlas", 60, "deploy"), kept: 2},
		{name: "other date", other: line("2026-10-06", "Atlas", 60, "API review"), kept: 2},
		{name: "other project", other: line("2026-10-05", "Atlas Mobile", 60, "API review"), kept: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := domain.DedupReportLines([]domain.ReportLine{base, tt.other})
			if len(got) != tt.kept || got[0] != base {
				t.Errorf("DedupReportLines(%+v, %+v) = %+v, want %d line(s) starting with the first", base, tt.other, got, tt.kept)
			}
		})
	}
}

func TestReportPolicyWarnsOverCapWithoutScaling(t *testing.T) {
	t.Parallel()
	parser := &stubParser{lines: map[string][]domain.ReportLine{
		"long day": {line("2026-10-05", "Atlas", 360, "release"), line("2026-10-05", "Beacon", 180, "hotfix")},
	}}

	p, err := domain.ReportPolicy{Parser: parser}.Allocate(context.Background(), reportInput("long day"))
	if err != nil {
		t.Fatalf("ReportPolicy.Allocate() error = %v, want nil", err)
	}
	if got := views(p); len(got) != 2 || got[0].Minutes != 360 || got[1].Minutes != 180 {
		t.Errorf("Allocate() entries = %+v, want 360m and 180m kept as reported", got)
	}
	if len(p.Warnings) != 1 || !strings.Contains(p.Warnings[0], "2026-10-05: 9h reported, over the daily cap of 8h") {
		t.Errorf("Allocate() warnings = %q, want one over-cap warning for 2026-10-05", p.Warnings)
	}
}

func TestReportPolicySkipsLinesOutsideTheWeekWithWarning(t *testing.T) {
	t.Parallel()
	parser := &stubParser{lines: map[string][]domain.ReportLine{
		"report": {
			line("2026-10-04", "Atlas", 60, "Sunday deploy"),
			line("2026-10-12", "Atlas", 60, "next Monday"),
			line("2026-10-07", "Atlas", 60, "kept"),
		},
	}}

	p, err := domain.ReportPolicy{Parser: parser}.Allocate(context.Background(), reportInput("report"))
	if err != nil {
		t.Fatalf("ReportPolicy.Allocate() error = %v, want nil", err)
	}
	if got := views(p); len(got) != 1 || got[0].Notes != "kept" {
		t.Errorf("Allocate() entries = %+v, want only the Wednesday line", got)
	}
	if len(p.Warnings) != 2 || !strings.Contains(p.Warnings[0], "weekend") || !strings.Contains(p.Warnings[1], "outside 2026-W41") {
		t.Errorf("Allocate() warnings = %q, want a weekend warning and an outside-the-week warning", p.Warnings)
	}
}

func TestReportPolicyRejectsBadLines(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		line domain.ReportLine
		want string
	}{
		{name: "malformed date", line: line("05/10/2026", "Atlas", 60, ""), want: `date "05/10/2026"`},
		{name: "minutes not a multiple of 15", line: line("2026-10-05", "Atlas", 20, ""), want: "2026-10-05"},
		{name: "zero minutes", line: line("2026-10-05", "Atlas", 0, ""), want: "2026-10-05"},
		{name: "no project", line: line("2026-10-05", "  ", 60, ""), want: "no project"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			parser := &stubParser{lines: map[string][]domain.ReportLine{"report": {tt.line}}}
			_, err := domain.ReportPolicy{Parser: parser}.Allocate(context.Background(), reportInput("report"))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Allocate(%+v) error = %v, want one mentioning %q", tt.line, err, tt.want)
			}
		})
	}
}

func TestReportPolicyNamesMessageThatFailedToParse(t *testing.T) {
	t.Parallel()
	parser := &stubParser{failOn: "broken"}
	_, err := domain.ReportPolicy{Parser: parser}.Allocate(context.Background(), reportInput("broken"))
	if !errors.Is(err, errParse) || !strings.Contains(err.Error(), "https://x/broken") {
		t.Errorf("Allocate() error = %v, want the parse error naming the message permalink", err)
	}
}

func TestNormalizeReportText(t *testing.T) {
	t.Parallel()
	if got := domain.NormalizeReportText("  API\t Review\n"); got != "api review" {
		t.Errorf("NormalizeReportText() = %q, want %q", got, "api review")
	}
}

// panicParser fails the test if the policy calls it.
type panicParser struct{ t *testing.T }

func (p panicParser) Parse(kernel.WorkActivity, *time.Location) ([]domain.ReportLine, error) {
	p.t.Errorf("Parse called, want precomputed lines used instead")
	return nil, nil
}

// report.parser: llm gives the same proposal as the manual parser when both
// produce the same lines: dedup, cap warning and entries are shared.
func TestReportPolicyPrecomputedMatchesParsedLines(t *testing.T) {
	t.Parallel()
	lines := []domain.ReportLine{
		line("2026-10-05", "Atlas", 360, "release"),
		line("2026-10-05", "atlas", 360, "Release "), // the self-DM copy
		line("2026-10-05", "Beacon", 180, "hotfix"),
		line("2026-10-06", "Beacon", 60, "standup"),
	}
	manual, err := domain.ReportPolicy{Parser: &stubParser{lines: map[string][]domain.ReportLine{"week": lines}}}.
		Allocate(context.Background(), reportInput("week"))
	if err != nil {
		t.Fatalf("manual Allocate() error = %v, want nil", err)
	}
	in := reportInput("week")
	in.ReportLines = lines
	llm, err := domain.ReportPolicy{Parser: panicParser{t}, Precomputed: true}.Allocate(context.Background(), in)
	if err != nil {
		t.Fatalf("precomputed Allocate() error = %v, want nil", err)
	}
	if !reflect.DeepEqual(llm, manual) {
		t.Errorf("precomputed Allocate() =\n%+v\nwant the manual parser's\n%+v", llm, manual)
	}
	if len(llm.Entries) != 3 || len(llm.Warnings) != 1 {
		t.Errorf("precomputed Allocate() = %d entries, warnings %q, want 3 entries and the Monday over-cap warning", len(llm.Entries), llm.Warnings)
	}
}

func TestReportDedupIgnoresSourceMessage(t *testing.T) {
	t.Parallel()
	a := domain.ReportLine{Date: "2026-10-05", Project: "Atlas", Minutes: 60, Notes: "x", SourceTS: "1.1", SourceChannelID: "C01"}
	b := a
	b.SourceTS, b.SourceChannelID = "2.2", "D01"
	if got := domain.DedupReportLines([]domain.ReportLine{a, b}); len(got) != 1 || got[0] != a {
		t.Errorf("DedupReportLines(channel copy, self-DM copy) = %+v, want only the first", got)
	}
}
