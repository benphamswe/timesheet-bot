package reportparser_test

import (
	"slices"
	"testing"
	"time"

	alloc "timesheet-bot/internal/allocation/domain"
	"timesheet-bot/internal/allocation/infrastructure/reportparser"
	"timesheet-bot/internal/shared/kernel"
)

func TestParseNotImplementedYet(t *testing.T) {
	t.Parallel()
	_, err := reportparser.New().Parse(kernel.WorkActivity{Text: "Atlas - 2h"}, time.UTC)
	if err == nil {
		t.Errorf("Parse() error = nil, want ErrNotImplemented until the owner writes it")
	}
}

// Owned by the owner (reportparser.Parser.Parse is TODO(owner)); the body is
// not edited. The cases are examples of decisions the doc comment lists;
// change them to the report format you choose, then remove the skip.
func TestParse(t *testing.T) {
	t.Skip("TODO(owner): reportparser.Parser.Parse not implemented yet; remove this skip when it is")
	t.Parallel()
	mon, err := kernel.ParseWorkDay("2026-10-05")
	if err != nil {
		t.Fatalf("ParseWorkDay: %v", err)
	}
	tz, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	tests := []struct {
		name    string
		text    string
		want    []alloc.ReportLine
		wantErr bool
	}{
		{name: "daily message, dated by the message", text: "Atlas - 4h - API review\nBeacon - 30m - standup",
			want: []alloc.ReportLine{
				{Date: "2026-10-05", Project: "Atlas", Minutes: 240, Notes: "API review"},
				{Date: "2026-10-05", Project: "Beacon", Minutes: 30, Notes: "standup"},
			}},
		{name: "decimal hours", text: "Atlas - 2.5h - design", want: []alloc.ReportLine{
			{Date: "2026-10-05", Project: "Atlas", Minutes: 150, Notes: "design"}}},
		{name: "hours and minutes", text: "Atlas - 1h30 - design", want: []alloc.ReportLine{
			{Date: "2026-10-05", Project: "Atlas", Minutes: 90, Notes: "design"}}},
		{name: "colon separator and bullet", text: "- Atlas: 1h review", want: []alloc.ReportLine{
			{Date: "2026-10-05", Project: "Atlas", Minutes: 60, Notes: "review"}}},
		{name: "weekly message with day headings", text: "Tue 06/10\nAtlas - 2h - release\nWed 07/10\nBeacon - 1h - fix",
			want: []alloc.ReportLine{
				{Date: "2026-10-06", Project: "Atlas", Minutes: 120, Notes: "release"},
				{Date: "2026-10-07", Project: "Beacon", Minutes: 60, Notes: "fix"},
			}},
		{name: "not a report", text: "lunch?", want: nil},
		{name: "project line without hours", text: "Atlas - API review", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := reportparser.New().Parse(kernel.WorkActivity{Day: mon, Text: tt.text}, tz)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse(%q) error = %v, want error %v", tt.text, err, tt.wantErr)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("Parse(%q) = %+v, want %+v", tt.text, got, tt.want)
			}
		})
	}
}
