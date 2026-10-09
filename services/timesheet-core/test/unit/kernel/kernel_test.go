package kernel_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"timesheet-bot/internal/shared/kernel"
)

// ict is a fixed UTC+7 zone (Asia/Ho_Chi_Minh has no DST), so the tests do
// not depend on the host's tzdata.
var ict = time.FixedZone("ICT", 7*60*60)

func date(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		t.Fatalf("time.Parse(%q): %v", s, err)
	}
	return d
}

func parseWeek(t *testing.T, s string) kernel.WeekID {
	t.Helper()
	w, err := kernel.ParseWeekID(s)
	if err != nil {
		t.Fatalf("ParseWeekID(%q): %v", s, err)
	}
	return w
}

func dayStrings(days []kernel.WorkDay) []string {
	out := make([]string, len(days))
	for i, d := range days {
		out[i] = d.String()
	}
	return out
}

func TestParseWeekIDRoundTrip(t *testing.T) {
	t.Parallel()
	for _, in := range []string{"2026-W41", "2026-W01", "2026-W53"} {
		t.Run(in, func(t *testing.T) {
			t.Parallel()
			w, err := kernel.ParseWeekID(in)
			if err != nil {
				t.Fatalf("ParseWeekID(%q) error = %v, want nil", in, err)
			}
			if got := w.String(); got != in {
				t.Errorf("ParseWeekID(%q).String() = %q, want %q", in, got, in)
			}
		})
	}
}

func TestParseWeekIDRejectsInvalid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
	}{
		{name: "empty", in: ""},
		{name: "missing W", in: "2026-41"},
		{name: "week zero", in: "2026-W00"},
		{name: "week 54", in: "2026-W54"},
		{name: "week 53 in a 52-week year", in: "2025-W53"},
		{name: "non-numeric year", in: "abcd-W10"},
		{name: "one-digit week", in: "2026-W4"},
		{name: "trailing text", in: "2026-W41x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, err := kernel.ParseWeekID(tt.in); err == nil {
				t.Errorf("ParseWeekID(%q) = %v, nil, want an error", tt.in, got)
			}
		})
	}
}

func TestWeekOfReturnsISOWeek(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   time.Time
		want string
	}{
		{name: "friday", in: date(t, "2026-10-09"), want: "2026-W41"},
		{name: "monday", in: date(t, "2026-10-05"), want: "2026-W41"},
		{name: "sunday ends the week", in: date(t, "2026-10-11"), want: "2026-W41"},
		{name: "december 31 in week 53", in: date(t, "2026-12-31"), want: "2026-W53"},
		{name: "january 1 in previous year's week 53", in: date(t, "2027-01-01"), want: "2026-W53"},
		// Sunday 23:00 UTC is Monday 06:00 in UTC+7, so the local week is next week.
		{name: "uses the time's location", in: time.Date(2026, 10, 11, 23, 0, 0, 0, time.UTC).In(ict), want: "2026-W42"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := kernel.WeekOf(tt.in).String(); got != tt.want {
				t.Errorf("WeekOf(%v) = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}

func TestWeekIDDays(t *testing.T) {
	t.Parallel()
	w := parseWeek(t, "2026-W41")
	got := dayStrings(w.Days())
	want := []string{"2026-10-05", "2026-10-06", "2026-10-07", "2026-10-08", "2026-10-09"}
	if !slices.Equal(got, want) {
		t.Errorf("%v.Days() = %v, want %v", w, got, want)
	}
}

func TestWeekIDContains(t *testing.T) {
	t.Parallel()
	w := parseWeek(t, "2026-W41")
	tests := []struct {
		name string
		day  kernel.WorkDay
		want bool
	}{
		{name: "wednesday of the week", day: w.Days()[2], want: true},
		{name: "friday of the previous week", day: w.Prev().Days()[4], want: false},
		{name: "monday of a later week", day: parseWeek(t, "2026-W42").Days()[0], want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := w.Contains(tt.day); got != tt.want {
				t.Errorf("%v.Contains(%v) = %v, want %v", w, tt.day, got, tt.want)
			}
		})
	}
}

func TestWeekIDPrev(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want string
	}{
		{in: "2026-W41", want: "2026-W40"},
		{in: "2026-W01", want: "2025-W52"},
		{in: "2027-W01", want: "2026-W53"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			if got := parseWeek(t, tt.in).Prev().String(); got != tt.want {
				t.Errorf("%s.Prev() = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}

func TestNewWorkDayRejectsWeekend(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   time.Time
	}{
		{name: "saturday", in: date(t, "2026-10-10")},
		{name: "sunday", in: date(t, "2026-10-11")},
		// Friday 22:00 UTC is already Saturday 05:00 in UTC+7.
		{name: "saturday in the time's location", in: time.Date(2026, 10, 9, 22, 0, 0, 0, time.UTC).In(ict)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, err := kernel.NewWorkDay(tt.in); !errors.Is(err, kernel.ErrWeekend) {
				t.Errorf("NewWorkDay(%v) = %v, %v, want ErrWeekend", tt.in, got, err)
			}
		})
	}
}

func TestNewWorkDayKeepsCalendarDate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   time.Time
		want string
	}{
		{name: "friday afternoon UTC", in: date(t, "2026-10-09").Add(15 * time.Hour), want: "2026-10-09"},
		// Sunday 23:00 UTC is Monday 06:00 in UTC+7.
		{name: "monday in the time's location", in: time.Date(2026, 10, 4, 23, 0, 0, 0, time.UTC).In(ict), want: "2026-10-05"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d, err := kernel.NewWorkDay(tt.in)
			if err != nil {
				t.Fatalf("NewWorkDay(%v) error = %v, want nil", tt.in, err)
			}
			if got := d.String(); got != tt.want {
				t.Errorf("NewWorkDay(%v) = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseWorkDay(t *testing.T) {
	t.Parallel()
	d, err := kernel.ParseWorkDay("2026-10-05")
	if err != nil {
		t.Fatalf("ParseWorkDay(%q) error = %v, want nil", "2026-10-05", err)
	}
	if got, want := d.Weekday(), time.Monday; got != want {
		t.Errorf("ParseWorkDay(%q).Weekday() = %v, want %v", "2026-10-05", got, want)
	}
	if got, err := kernel.ParseWorkDay("2026-10-10"); !errors.Is(err, kernel.ErrWeekend) {
		t.Errorf("ParseWorkDay(%q) = %v, %v, want ErrWeekend", "2026-10-10", got, err)
	}
	if got, err := kernel.ParseWorkDay("05/10/2026"); err == nil {
		t.Errorf("ParseWorkDay(%q) = %v, nil, want a parse error", "05/10/2026", got)
	}
}

func TestNewIssueKeyAcceptsValidKeys(t *testing.T) {
	t.Parallel()
	for _, in := range []string{"GCM-1", "AB2-123", "XY-9"} {
		t.Run(in, func(t *testing.T) {
			t.Parallel()
			got, err := kernel.NewIssueKey(in)
			if err != nil {
				t.Fatalf("NewIssueKey(%q) error = %v, want nil", in, err)
			}
			if got.String() != in {
				t.Errorf("NewIssueKey(%q) = %q, want %q", in, got, in)
			}
		})
	}
}

func TestNewIssueKeyRejectsInvalidKeys(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
	}{
		{name: "lowercase project", in: "gcm-1"},
		{name: "missing number", in: "GCM-"},
		{name: "missing project", in: "-1"},
		{name: "missing dash", in: "GCM1"},
		{name: "leading digit", in: "1GC-2"},
		{name: "one-letter project", in: "X-9"},
		{name: "empty", in: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, err := kernel.NewIssueKey(tt.in); err == nil {
				t.Errorf("NewIssueKey(%q) = %q, nil, want an error", tt.in, got)
			}
		})
	}
}

// AC-001.2
func TestExtractIssueKeys(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		text string
		want []kernel.IssueKey
	}{
		{
			name: "distinct keys in order of first appearance",
			text: "done GCM-12, reviewing HUB-3 and GCM-12 again; not utf-8 or abc-1",
			want: []kernel.IssueKey{"GCM-12", "HUB-3"},
		},
		{name: "no keys", text: "lunch?", want: nil},
		// Key-shaped words are extracted; the application drops the ones Jira does not know.
		{name: "key-shaped word", text: "UTF-8 encoding", want: []kernel.IssueKey{"UTF-8"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := kernel.ExtractIssueKeys(tt.text); !slices.Equal(got, tt.want) {
				t.Errorf("ExtractIssueKeys(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestNewDurationRejectsNonPositiveOrOffGrid(t *testing.T) {
	t.Parallel()
	for _, minutes := range []int{0, -15, 10, 50} {
		if got, err := kernel.NewDuration(minutes); err == nil {
			t.Errorf("NewDuration(%d) = %v, nil, want an error", minutes, got)
		}
	}
}

func TestDurationUnitsAndString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		minutes     int
		wantSeconds int
		wantString  string
	}{
		{minutes: 45, wantSeconds: 2700, wantString: "45m"},
		{minutes: 60, wantSeconds: 3600, wantString: "1h"},
		{minutes: 90, wantSeconds: 5400, wantString: "1h30m"},
	}
	for _, tt := range tests {
		t.Run(tt.wantString, func(t *testing.T) {
			t.Parallel()
			d, err := kernel.NewDuration(tt.minutes)
			if err != nil {
				t.Fatalf("NewDuration(%d) error = %v, want nil", tt.minutes, err)
			}
			if got := d.Minutes(); got != tt.minutes {
				t.Errorf("NewDuration(%d).Minutes() = %d, want %d", tt.minutes, got, tt.minutes)
			}
			if got := d.Seconds(); got != tt.wantSeconds {
				t.Errorf("NewDuration(%d).Seconds() = %d, want %d", tt.minutes, got, tt.wantSeconds)
			}
			if got := d.String(); got != tt.wantString {
				t.Errorf("NewDuration(%d).String() = %q, want %q", tt.minutes, got, tt.wantString)
			}
		})
	}
}

func TestMustDurationPanicsOnInvalid(t *testing.T) {
	t.Parallel()
	defer func() {
		if recover() == nil {
			t.Errorf("MustDuration(10) did not panic, want a panic")
		}
	}()
	kernel.MustDuration(10)
}

func TestSystemClockUsesLocation(t *testing.T) {
	t.Parallel()
	// Only the location is asserted; the wall-clock value is never compared.
	if got := (kernel.SystemClock{Location: ict}).Now().Location(); got != ict {
		t.Errorf("SystemClock{ict}.Now().Location() = %v, want %v", got, ict)
	}
}
