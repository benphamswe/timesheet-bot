package domain_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"timesheet-bot/internal/shared/kernel"
	"timesheet-bot/internal/timesheet/domain"
)

var (
	week   = kernel.WeekID{Year: 2026, Week: 41}
	mon    = week.Days()[0]
	tue    = week.Days()[1]
	wed    = week.Days()[2]
	now    = time.Date(2026, 10, 9, 16, 0, 0, 0, time.UTC)
	cap8h  = kernel.MustDuration(480)
	min45  = kernel.MustDuration(45)
	hours4 = kernel.MustDuration(240)

	tempoOnly = []domain.Target{domain.TargetTempo}
	both      = []domain.Target{domain.TargetTempo, domain.TargetMavenlink}
)

func newDraft(t *testing.T) *domain.Timesheet {
	t.Helper()
	return domain.New("ts-1", "acc-1", week, cap8h, "rule", now)
}

func addEntry(t *testing.T, ts *domain.Timesheet, key string, day kernel.WorkDay, d kernel.Duration) {
	t.Helper()
	if err := ts.AddEntry(kernel.IssueKey(key), 1000, day, d, "work on "+key); err != nil {
		t.Fatalf("AddEntry(%s, %s, %s) error = %v, want nil", key, day, d, err)
	}
}

func mustOK(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("setup step error = %v, want nil", err)
	}
}

func eventNames(evs []kernel.DomainEvent) []string {
	names := make([]string, len(evs))
	for i, e := range evs {
		names[i] = e.EventName()
	}
	return names
}

// inStatus returns a timesheet with GCM-1 (Monday) and HUB-2 (Tuesday) that
// reached status through the aggregate's own methods, with its events drained.
func inStatus(t *testing.T, status domain.Status) *domain.Timesheet {
	t.Helper()
	ts := newDraft(t)
	addEntry(t, ts, "GCM-1", mon, min45)
	addEntry(t, ts, "HUB-2", tue, min45)
	switch status {
	case domain.StatusDraft:
	case domain.StatusAwaitingApproval:
		mustOK(t, ts.Propose(now))
	case domain.StatusApproved:
		mustOK(t, ts.Propose(now))
		mustOK(t, ts.Approve(now))
	case domain.StatusRejected:
		mustOK(t, ts.Propose(now))
		mustOK(t, ts.Reject("wrong hours", now))
	case domain.StatusSubmitted:
		mustOK(t, ts.Propose(now))
		mustOK(t, ts.Approve(now))
		for _, e := range ts.PendingEntries(domain.TargetTempo) {
			mustOK(t, ts.MarkEntrySubmitted(e.ID, domain.TargetTempo, "w-"+e.ID, now))
		}
		mustOK(t, ts.CompleteSubmission(tempoOnly, now))
	default:
		t.Fatalf("inStatus(%q): unknown status", status)
	}
	if got := ts.Status(); got != status {
		t.Fatalf("inStatus(%q) built status %q", status, got)
	}
	ts.PullEvents()
	return ts
}

func TestAddEntryInvariants(t *testing.T) {
	t.Parallel()
	twoWeeksAgo := week.Prev().Prev().Days()[0] // outside the week
	tests := []struct {
		name string
		day  kernel.WorkDay
		key  string
		dur  kernel.Duration
		want error
	}{
		{name: "outside week", day: twoWeeksAgo, key: "GCM-2", dur: min45, want: domain.ErrDayOutsideWeek},
		{name: "duplicate issue and day", day: mon, key: "GCM-1", dur: min45, want: domain.ErrDuplicateEntry},
		{name: "one step over cap", day: mon, key: "GCM-3", dur: kernel.MustDuration(255), want: domain.ErrDayOverCap},
		{name: "exactly cap ok", day: mon, key: "GCM-4", dur: hours4, want: nil},
		{name: "same issue other day ok", day: tue, key: "GCM-1", dur: min45, want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ts := newDraft(t)
			addEntry(t, ts, "GCM-1", mon, hours4)
			err := ts.AddEntry(kernel.IssueKey(tt.key), 1, tt.day, tt.dur, "")
			if !errors.Is(err, tt.want) {
				t.Errorf("AddEntry(%s, %s, %s) error = %v, want %v", tt.key, tt.day, tt.dur, err, tt.want)
			}
		})
	}
}

func TestAddEntryRecordsEntry(t *testing.T) {
	t.Parallel()
	ts := newDraft(t)
	mustOK(t, ts.AddEntry("GCM-1", 101, mon, min45, "work on GCM-1"))
	want := []domain.Entry{{
		ID: "ts-1/2026-10-05/GCM-1", IssueKey: "GCM-1", IssueID: 101, Day: mon, Duration: min45, Description: "work on GCM-1",
	}}
	if got := ts.Entries(); !slices.Equal(got, want) {
		t.Errorf("Entries() = %+v, want %+v", got, want)
	}
}

func TestProposeRequiresEntries(t *testing.T) {
	t.Parallel()
	ts := newDraft(t)
	if err := ts.Propose(now); !errors.Is(err, domain.ErrEmptyTimesheet) {
		t.Errorf("Propose() on empty draft error = %v, want ErrEmptyTimesheet", err)
	}
	if got := ts.Status(); got != domain.StatusDraft {
		t.Errorf("Status() after failed Propose = %s, want %s", got, domain.StatusDraft)
	}
}

func TestProposeBumpsRevision(t *testing.T) {
	t.Parallel()
	ts := newDraft(t)
	addEntry(t, ts, "GCM-1", mon, min45)
	if got := ts.Revision(); got != 0 {
		t.Fatalf("Revision() of a new draft = %d, want 0", got)
	}
	mustOK(t, ts.Propose(now))
	if got := ts.Revision(); got != 1 {
		t.Errorf("Revision() after Propose = %d, want 1", got)
	}
	evs := ts.PullEvents()
	if len(evs) != 1 {
		t.Fatalf("PullEvents() = %v, want one TimesheetProposed event", eventNames(evs))
	}
	ev, ok := evs[0].(domain.Event)
	if !ok {
		t.Fatalf("PullEvents()[0] is %T, want domain.Event", evs[0])
	}
	want := domain.Event{Name: domain.EventProposed, TimesheetID: "ts-1", Week: "2026-W41", Revision: 1, At: now}
	if ev != want {
		t.Errorf("proposed event = %+v, want %+v", ev, want)
	}
}

func TestLifecycleProposeApproveSubmitEmitsEvents(t *testing.T) {
	t.Parallel()
	ts := newDraft(t)
	addEntry(t, ts, "GCM-1", mon, min45)
	addEntry(t, ts, "HUB-2", tue, min45)

	mustOK(t, ts.Propose(now))
	mustOK(t, ts.Approve(now))
	pending := ts.PendingEntries(domain.TargetTempo)
	if len(pending) != 2 {
		t.Fatalf("PendingEntries() = %d entries, want 2", len(pending))
	}
	mustOK(t, ts.MarkEntrySubmitted(pending[0].ID, domain.TargetTempo, "w-1", now))
	if got := ts.Status(); got != domain.StatusApproved {
		t.Errorf("Status() after 1 of 2 submitted = %s, want %s", got, domain.StatusApproved)
	}
	mustOK(t, ts.MarkEntrySubmitted(pending[1].ID, domain.TargetTempo, "w-2", now))
	if got := ts.Status(); got != domain.StatusApproved {
		t.Errorf("Status() after 2 of 2 submitted = %s, want %s until CompleteSubmission", got, domain.StatusApproved)
	}
	mustOK(t, ts.CompleteSubmission(tempoOnly, now))
	if got := ts.Status(); got != domain.StatusSubmitted {
		t.Errorf("Status() after 2 of 2 submitted = %s, want %s", got, domain.StatusSubmitted)
	}
	if got := ts.PendingEntries(domain.TargetTempo); len(got) != 0 {
		t.Errorf("PendingEntries() after submit = %+v, want none", got)
	}

	want := []string{domain.EventProposed, domain.EventApproved, domain.EventSubmitted}
	if got := eventNames(ts.PullEvents()); !slices.Equal(got, want) {
		t.Errorf("PullEvents() = %v, want %v", got, want)
	}
	if got := ts.PullEvents(); len(got) != 0 {
		t.Errorf("second PullEvents() = %v, want none (drained)", eventNames(got))
	}
}

// AC-004.1
func TestIllegalTransitions(t *testing.T) {
	t.Parallel()
	type command struct {
		name string
		run  func(ts *domain.Timesheet) error
	}
	propose := command{"Propose", func(ts *domain.Timesheet) error { return ts.Propose(now) }}
	approve := command{"Approve", func(ts *domain.Timesheet) error { return ts.Approve(now) }}
	reject := command{"Reject", func(ts *domain.Timesheet) error { return ts.Reject("no", now) }}
	add := command{"AddEntry", func(ts *domain.Timesheet) error { return ts.AddEntry("GCM-9", 9, wed, min45, "") }}
	submit := command{"MarkEntrySubmitted", func(ts *domain.Timesheet) error {
		return ts.MarkEntrySubmitted(ts.Entries()[0].ID, domain.TargetTempo, "w-9", now)
	}}
	fail := command{"MarkEntryFailed", func(ts *domain.Timesheet) error {
		return ts.MarkEntryFailed(ts.Entries()[0].ID, errors.New("tempo 500"), now)
	}}
	// complete finds every entry pending, so from approved it is refused with ErrEntriesPending.
	complete := command{"CompleteSubmission", func(ts *domain.Timesheet) error {
		return ts.CompleteSubmission(tempoOnly, now)
	}}

	tests := []struct {
		from domain.Status
		cmd  command
		want error
	}{
		{domain.StatusDraft, add, nil},
		{domain.StatusDraft, propose, nil},
		{domain.StatusDraft, approve, domain.ErrInvalidTransition},
		{domain.StatusDraft, reject, domain.ErrInvalidTransition},
		{domain.StatusDraft, submit, domain.ErrInvalidTransition},
		{domain.StatusDraft, fail, domain.ErrInvalidTransition},
		{domain.StatusDraft, complete, domain.ErrInvalidTransition},

		{domain.StatusAwaitingApproval, add, domain.ErrNotDraft},
		{domain.StatusAwaitingApproval, propose, domain.ErrInvalidTransition},
		{domain.StatusAwaitingApproval, approve, nil},
		{domain.StatusAwaitingApproval, reject, nil},
		{domain.StatusAwaitingApproval, submit, domain.ErrInvalidTransition},
		{domain.StatusAwaitingApproval, fail, domain.ErrInvalidTransition},
		{domain.StatusAwaitingApproval, complete, domain.ErrInvalidTransition},

		{domain.StatusApproved, add, domain.ErrNotDraft},
		{domain.StatusApproved, propose, domain.ErrInvalidTransition},
		{domain.StatusApproved, approve, domain.ErrInvalidTransition},
		{domain.StatusApproved, reject, domain.ErrInvalidTransition},
		{domain.StatusApproved, submit, nil},
		{domain.StatusApproved, fail, nil},
		{domain.StatusApproved, complete, domain.ErrEntriesPending},

		{domain.StatusRejected, add, domain.ErrNotDraft},
		{domain.StatusRejected, propose, domain.ErrInvalidTransition},
		{domain.StatusRejected, approve, domain.ErrInvalidTransition},
		{domain.StatusRejected, reject, domain.ErrInvalidTransition},
		{domain.StatusRejected, submit, domain.ErrInvalidTransition},
		{domain.StatusRejected, fail, domain.ErrInvalidTransition},
		{domain.StatusRejected, complete, domain.ErrInvalidTransition},

		{domain.StatusSubmitted, add, domain.ErrNotDraft},
		{domain.StatusSubmitted, propose, domain.ErrInvalidTransition},
		{domain.StatusSubmitted, approve, domain.ErrInvalidTransition},
		{domain.StatusSubmitted, reject, domain.ErrInvalidTransition},
		{domain.StatusSubmitted, submit, domain.ErrInvalidTransition},
		{domain.StatusSubmitted, fail, domain.ErrInvalidTransition},
		{domain.StatusSubmitted, complete, domain.ErrInvalidTransition},
	}
	for _, tt := range tests {
		t.Run(string(tt.from)+"/"+tt.cmd.name, func(t *testing.T) {
			t.Parallel()
			ts := inStatus(t, tt.from)
			if err := tt.cmd.run(ts); !errors.Is(err, tt.want) {
				t.Errorf("%s from %s error = %v, want %v", tt.cmd.name, tt.from, err, tt.want)
			}
		})
	}
}

func TestRejectedCommandLeavesStateUnchanged(t *testing.T) {
	t.Parallel()
	ts := inStatus(t, domain.StatusDraft)
	before := ts.Snapshot()
	if err := ts.Approve(now); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("Approve() from draft error = %v, want ErrInvalidTransition", err)
	}
	if after := ts.Snapshot(); !reflect.DeepEqual(after, before) {
		t.Errorf("Snapshot() after rejected Approve = %+v, want %+v", after, before)
	}
	if got := ts.PullEvents(); len(got) != 0 {
		t.Errorf("PullEvents() after rejected Approve = %v, want none", eventNames(got))
	}
}

func TestRejectRecordsReasonEvent(t *testing.T) {
	t.Parallel()
	ts := inStatus(t, domain.StatusAwaitingApproval)
	mustOK(t, ts.Reject("GCM-1 was 2h", now))
	evs := ts.PullEvents()
	if len(evs) != 1 {
		t.Fatalf("PullEvents() = %v, want one TimesheetRejected event", eventNames(evs))
	}
	ev, ok := evs[0].(domain.Event)
	if !ok {
		t.Fatalf("PullEvents()[0] is %T, want domain.Event", evs[0])
	}
	want := domain.Event{Name: domain.EventRejected, TimesheetID: "ts-1", Week: "2026-W41", Revision: 1, Reason: "GCM-1 was 2h", At: now}
	if ev != want {
		t.Errorf("rejected event = %+v, want %+v", ev, want)
	}
}

func TestMarkEntryFailedKeepsPending(t *testing.T) {
	t.Parallel()
	ts := inStatus(t, domain.StatusApproved)
	id := ts.Entries()[0].ID
	mustOK(t, ts.MarkEntryFailed(id, errors.New("tempo 500"), now))

	pending := ts.PendingEntries(domain.TargetTempo)
	if len(pending) != 2 {
		t.Fatalf("PendingEntries() = %d entries, want 2 (failed entry stays pending)", len(pending))
	}
	if got := pending[0]; got.ID != id || got.SubmitAttempts != 1 || got.LastError != "tempo 500" {
		t.Errorf("failed entry = %+v, want ID %s, SubmitAttempts 1, LastError %q", got, id, "tempo 500")
	}
	if got := ts.Status(); got != domain.StatusApproved {
		t.Errorf("Status() after a failure = %s, want %s", got, domain.StatusApproved)
	}
}

func TestMarkEntryUnknownIDIsNotFound(t *testing.T) {
	t.Parallel()
	ts := inStatus(t, domain.StatusApproved)
	if err := ts.MarkEntryFailed("nope", errors.New("x"), now); !errors.Is(err, domain.ErrEntryNotFound) {
		t.Errorf("MarkEntryFailed(%q) error = %v, want ErrEntryNotFound", "nope", err)
	}
	if err := ts.MarkEntrySubmitted("nope", domain.TargetTempo, "w-1", now); !errors.Is(err, domain.ErrEntryNotFound) {
		t.Errorf("MarkEntrySubmitted(%q) error = %v, want ErrEntryNotFound", "nope", err)
	}
}

func TestMarkEntryFailedEmitsEventAtMaxAttempts(t *testing.T) {
	t.Parallel()
	ts := inStatus(t, domain.StatusApproved)
	id := ts.Entries()[0].ID
	for attempt := 1; attempt < domain.MaxSubmitAttempts; attempt++ {
		mustOK(t, ts.MarkEntryFailed(id, errors.New("tempo 500"), now))
		if got := ts.PullEvents(); len(got) != 0 {
			t.Fatalf("PullEvents() after attempt %d = %v, want none", attempt, eventNames(got))
		}
	}

	mustOK(t, ts.MarkEntryFailed(id, errors.New("tempo 503"), now))
	evs := ts.PullEvents()
	if len(evs) != 1 {
		t.Fatalf("PullEvents() after attempt %d = %v, want one %s", domain.MaxSubmitAttempts, eventNames(evs), domain.EventEntrySubmitFail)
	}
	ev, ok := evs[0].(domain.Event)
	if !ok {
		t.Fatalf("PullEvents()[0] is %T, want domain.Event", evs[0])
	}
	want := domain.Event{
		Name: domain.EventEntrySubmitFail, TimesheetID: "ts-1", Week: "2026-W41", Revision: 1,
		EntryID: id, Reason: "tempo 503", At: now,
	}
	if ev != want {
		t.Errorf("submit-failed event = %+v, want %+v", ev, want)
	}

	mustOK(t, ts.MarkEntryFailed(id, errors.New("tempo 500"), now))
	if got := ts.PullEvents(); len(got) != 0 {
		t.Errorf("PullEvents() after attempt %d = %v, want none (event fires once)", domain.MaxSubmitAttempts+1, eventNames(got))
	}
}

func TestMarkEntrySubmittedKeepsFirstWorklogID(t *testing.T) {
	t.Skip("BUG: exactly-once invariant; owner to decide rule")
	t.Parallel()
	ts := inStatus(t, domain.StatusApproved)
	id := ts.Entries()[0].ID
	mustOK(t, ts.MarkEntrySubmitted(id, domain.TargetTempo, "w-1", now))

	err := ts.MarkEntrySubmitted(id, domain.TargetTempo, "w-9", now)
	if err == nil {
		t.Errorf("second MarkEntrySubmitted(%s) error = nil, want an error", id)
	}
	if got := ts.Entries()[0].TempoWorklogID; got != "w-1" {
		t.Errorf("TempoWorklogID after second MarkEntrySubmitted = %q, want %q (first id kept)", got, "w-1")
	}
}

func TestDayTotalSumsOnlyThatDay(t *testing.T) {
	t.Parallel()
	ts := newDraft(t)
	addEntry(t, ts, "GCM-1", mon, min45)
	addEntry(t, ts, "GCM-2", mon, kernel.MustDuration(90))
	addEntry(t, ts, "GCM-1", tue, min45)
	tests := []struct {
		day  kernel.WorkDay
		want int
	}{
		{day: mon, want: 135},
		{day: tue, want: 45},
		{day: wed, want: 0},
	}
	for _, tt := range tests {
		if got := ts.DayTotal(tt.day); got != tt.want {
			t.Errorf("DayTotal(%s) = %d, want %d", tt.day, got, tt.want)
		}
	}
}

func TestEntriesSortedByDayThenKey(t *testing.T) {
	t.Parallel()
	ts := newDraft(t)
	addEntry(t, ts, "GCM-1", tue, min45)
	addEntry(t, ts, "HUB-2", mon, min45)
	addEntry(t, ts, "GCM-3", mon, min45)
	var got []string
	for _, e := range ts.Entries() {
		got = append(got, e.Day.String()+" "+e.IssueKey.String())
	}
	want := []string{"2026-10-05 GCM-3", "2026-10-05 HUB-2", "2026-10-06 GCM-1"}
	if !slices.Equal(got, want) {
		t.Errorf("Entries() order = %v, want %v", got, want)
	}
}

func TestEntriesReturnsDefensiveCopy(t *testing.T) {
	t.Parallel()
	ts := inStatus(t, domain.StatusDraft)
	entries := ts.Entries()
	entries[0].Description = "changed by caller"
	entries[0].TempoWorklogID = "w-forged"
	if got := ts.Entries()[0]; got.Description != "work on GCM-1" || got.TempoWorklogID != "" {
		t.Errorf("Entries()[0] after caller edit = %+v, want the aggregate unchanged", got)
	}
}

func TestSnapshotRehydrateRoundTrip(t *testing.T) {
	t.Parallel()
	ts := inStatus(t, domain.StatusApproved)
	ts.SetVersion(3)
	mustOK(t, ts.MarkEntryFailed(ts.Entries()[0].ID, errors.New("tempo 500"), now))
	snap := ts.Snapshot()

	got := domain.Rehydrate(snap)
	if gotSnap := got.Snapshot(); !reflect.DeepEqual(gotSnap, snap) {
		t.Errorf("Rehydrate(s).Snapshot() = %+v, want %+v", gotSnap, snap)
	}
	if got.ID() != "ts-1" || got.UserAccountID() != "acc-1" || got.Week() != week || got.Policy() != "rule" ||
		got.Status() != domain.StatusApproved || got.Version() != 3 || got.Revision() != 1 {
		t.Errorf("Rehydrate(s) accessors = %s %s %v %s %s v%d r%d, want ts-1 acc-1 %v rule %s v3 r1",
			got.ID(), got.UserAccountID(), got.Week(), got.Policy(), got.Status(), got.Version(), got.Revision(),
			week, domain.StatusApproved)
	}
	if evs := got.PullEvents(); len(evs) != 0 {
		t.Errorf("Rehydrate(s).PullEvents() = %v, want none (events are not persisted state)", eventNames(evs))
	}

	snap.Entries[0].Description = "edited after rehydrate"
	if d := got.Entries()[0].Description; d == "edited after rehydrate" {
		t.Errorf("Rehydrate(s) shares the entries slice with s, want a copy")
	}
}

func TestPendingEntriesArePerTarget(t *testing.T) {
	t.Parallel()
	ts := inStatus(t, domain.StatusApproved)
	id := ts.Entries()[0].ID
	mustOK(t, ts.MarkEntrySubmitted(id, domain.TargetMavenlink, "m-1", now))

	if got := len(ts.PendingEntries(domain.TargetMavenlink)); got != 1 {
		t.Errorf("PendingEntries(mavenlink) = %d entries, want 1", got)
	}
	if got := len(ts.PendingEntries(domain.TargetTempo)); got != 2 {
		t.Errorf("PendingEntries(tempo) = %d entries, want 2 (a Mavenlink id is not a Tempo id)", got)
	}
	e := ts.Entries()[0]
	if !e.SubmittedTo(domain.TargetMavenlink) || e.SubmittedTo(domain.TargetTempo) || e.MavenlinkEntryID != "m-1" {
		t.Errorf("entry = %+v, want only MavenlinkEntryID m-1", e)
	}
}

func TestCompleteSubmissionNeedsEveryTarget(t *testing.T) {
	t.Parallel()
	ts := inStatus(t, domain.StatusApproved)
	for _, e := range ts.Entries() {
		mustOK(t, ts.MarkEntrySubmitted(e.ID, domain.TargetTempo, "w-"+e.ID, now))
	}
	if err := ts.CompleteSubmission(both, now); !errors.Is(err, domain.ErrEntriesPending) {
		t.Fatalf("CompleteSubmission(tempo, mavenlink) with Mavenlink pending error = %v, want ErrEntriesPending", err)
	}
	if got := ts.PullEvents(); len(got) != 0 {
		t.Errorf("PullEvents() after a refused CompleteSubmission = %v, want none", eventNames(got))
	}
	for _, e := range ts.Entries() {
		mustOK(t, ts.MarkEntrySubmitted(e.ID, domain.TargetMavenlink, "m-"+e.ID, now))
	}
	mustOK(t, ts.CompleteSubmission(both, now))
	if got := ts.Status(); got != domain.StatusSubmitted {
		t.Errorf("Status() after CompleteSubmission = %s, want %s", got, domain.StatusSubmitted)
	}
	if got, want := eventNames(ts.PullEvents()), []string{domain.EventSubmitted}; !slices.Equal(got, want) {
		t.Errorf("PullEvents() = %v, want %v", got, want)
	}
}

func TestCompleteSubmissionWithoutTargetsIsInvalid(t *testing.T) {
	t.Parallel()
	ts := inStatus(t, domain.StatusApproved)
	if err := ts.CompleteSubmission(nil, now); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Errorf("CompleteSubmission(nil) error = %v, want ErrInvalidTransition", err)
	}
}

func TestMarkEntrySubmittedUnknownTarget(t *testing.T) {
	t.Parallel()
	ts := inStatus(t, domain.StatusApproved)
	before := ts.Snapshot()
	if err := ts.MarkEntrySubmitted(ts.Entries()[0].ID, "jira", "x-1", now); !errors.Is(err, domain.ErrUnknownTarget) {
		t.Errorf("MarkEntrySubmitted(target jira) error = %v, want ErrUnknownTarget", err)
	}
	if after := ts.Snapshot(); !reflect.DeepEqual(after, before) {
		t.Errorf("Snapshot() after unknown target = %+v, want %+v", after, before)
	}
}
