// Package domain is the Timesheet context's domain: the [Timesheet] aggregate,
// its [Entry] lines, [Status] lifecycle and [Event]s, and the [Repository]
// port. The aggregate enforces the invariants every proposal must pass: one
// entry per issue and day, every day inside the timesheet's ISO week, a daily
// cap, and changes to entries only while the timesheet is a draft.
package domain

import (
	"fmt"
	"sort"
	"time"

	"timesheet-bot/internal/shared/kernel"
)

// MaxSubmitAttempts is how many failed submit attempts, over every target, an
// entry gets before the user is told.
const MaxSubmitAttempts = 3

// Timesheet is the aggregate root: one user's worklogs for one ISO week.
type Timesheet struct {
	id            string
	userAccountID string
	week          kernel.WeekID
	dailyCap      kernel.Duration
	policy        string
	status        Status
	entries       []Entry
	version       int // persisted version, used for optimistic locking
	revision      int // bumps on every proposal
	createdAt     time.Time
	events        []kernel.DomainEvent
}

// New returns an empty draft timesheet for userAccountID's week. dailyCap is
// the most time [Timesheet.AddEntry] accepts on any one day, policy names the
// allocation policy that fills the timesheet, and now becomes its creation
// time. The timesheet starts at version 0 and revision 0.
func New(id, userAccountID string, week kernel.WeekID, dailyCap kernel.Duration, policy string, now time.Time) *Timesheet {
	return &Timesheet{
		id: id, userAccountID: userAccountID, week: week, dailyCap: dailyCap,
		policy: policy, status: StatusDraft, createdAt: now,
	}
}

// Snapshot is the persisted state, used by repositories only.
type Snapshot struct {
	ID            string
	UserAccountID string
	Week          kernel.WeekID
	DailyCap      kernel.Duration
	Policy        string
	Status        Status
	Entries       []Entry
	Version       int
	Revision      int
	CreatedAt     time.Time
}

// Rehydrate rebuilds a Timesheet from persisted state, for repositories only.
// It copies s.Entries, does not re-check the invariants, and records no
// events.
func Rehydrate(s Snapshot) *Timesheet {
	return &Timesheet{
		id: s.ID, userAccountID: s.UserAccountID, week: s.Week, dailyCap: s.DailyCap,
		policy: s.Policy, status: s.Status, entries: append([]Entry(nil), s.Entries...),
		version: s.Version, revision: s.Revision, createdAt: s.CreatedAt,
	}
}

// Snapshot returns t's persisted state, with entries copied and sorted as by
// [Timesheet.Entries]. Recorded events are not included; use
// [Timesheet.PullEvents].
func (t *Timesheet) Snapshot() Snapshot {
	return Snapshot{
		ID: t.id, UserAccountID: t.userAccountID, Week: t.week, DailyCap: t.dailyCap,
		Policy: t.policy, Status: t.status, Entries: t.Entries(),
		Version: t.version, Revision: t.revision, CreatedAt: t.createdAt,
	}
}

// ID returns the timesheet's identifier, the prefix of every entry ID.
func (t *Timesheet) ID() string { return t.id }

// UserAccountID returns the account ID of the user who owns the timesheet.
func (t *Timesheet) UserAccountID() string { return t.userAccountID }

// Week returns the ISO week the timesheet covers.
func (t *Timesheet) Week() kernel.WeekID { return t.week }

// Policy returns the name of the allocation policy the timesheet was created
// with.
func (t *Timesheet) Policy() string { return t.policy }

// Status returns the timesheet's current lifecycle state.
func (t *Timesheet) Status() Status { return t.status }

// Version returns the persisted version used for optimistic locking. It is
// set only by [Timesheet.SetVersion] and [Rehydrate].
func (t *Timesheet) Version() int { return t.version }

// Revision returns how many times the timesheet has been proposed.
// [Timesheet.Propose] increments it, and it is stamped on every event.
func (t *Timesheet) Revision() int { return t.revision }

// SetVersion is called by repositories after a successful save.
func (t *Timesheet) SetVersion(v int) { t.version = v }

// Entries returns a copy sorted by day, then issue key, then ID; report
// entries, which have no issue key, keep the order they were added in
// (report-01, report-02, ...).
func (t *Timesheet) Entries() []Entry {
	out := append([]Entry(nil), t.entries...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Day != out[j].Day {
			return out[i].Day.Before(out[j].Day)
		}
		if out[i].IssueKey != out[j].IssueKey {
			return out[i].IssueKey < out[j].IssueKey
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// DayTotal returns minutes logged on day.
func (t *Timesheet) DayTotal(day kernel.WorkDay) int {
	total := 0
	for _, e := range t.entries {
		if e.Day == day {
			total += e.Duration.Minutes()
		}
	}
	return total
}

// AddEntry adds a worklog line while the timesheet is a draft.
func (t *Timesheet) AddEntry(key kernel.IssueKey, issueID int64, day kernel.WorkDay, d kernel.Duration, description string) error {
	if t.status != StatusDraft {
		return ErrNotDraft
	}
	if !t.week.Contains(day) {
		return fmt.Errorf("%w: %s not in %s", ErrDayOutsideWeek, day, t.week)
	}
	for _, e := range t.entries {
		if e.IssueKey == key && e.Day == day {
			return fmt.Errorf("%w: %s on %s", ErrDuplicateEntry, key, day)
		}
	}
	if total := t.DayTotal(day) + d.Minutes(); total > t.dailyCap.Minutes() {
		return fmt.Errorf("%w: %s would be %dm, cap %dm", ErrDayOverCap, day, total, t.dailyCap.Minutes())
	}
	t.entries = append(t.entries, Entry{
		ID:       fmt.Sprintf("%s/%s/%s", t.id, day, key),
		IssueKey: key, IssueID: issueID, Day: day, Duration: d, Description: description,
	})
	return nil
}

// AddReportEntry adds a report mode line, time on project as the owner wrote
// it, while the timesheet is a draft. Unlike [Timesheet.AddEntry] it accepts
// several entries for one project on one day and a day over the daily cap:
// the owner's report is the source of truth, and plan warns about the cap
// instead (ADR-0012). Entry IDs are <timesheet>/<day>/report-NN, numbered per
// day in the order added.
func (t *Timesheet) AddReportEntry(project string, day kernel.WorkDay, d kernel.Duration, notes string) error {
	if t.status != StatusDraft {
		return ErrNotDraft
	}
	if !t.week.Contains(day) {
		return fmt.Errorf("%w: %s not in %s", ErrDayOutsideWeek, day, t.week)
	}
	if project == "" {
		return ErrNoProject
	}
	n := 1
	for _, e := range t.entries {
		if e.Day == day && e.IssueKey == "" {
			n++
		}
	}
	t.entries = append(t.entries, Entry{
		ID:      fmt.Sprintf("%s/%s/report-%02d", t.id, day, n),
		Project: project, Day: day, Duration: d, Description: notes,
	})
	return nil
}

// Propose sends the draft for approval.
func (t *Timesheet) Propose(now time.Time) error {
	if t.status != StatusDraft {
		return fmt.Errorf("%w: propose from %s", ErrInvalidTransition, t.status)
	}
	if len(t.entries) == 0 {
		return ErrEmptyTimesheet
	}
	t.status = StatusAwaitingApproval
	t.revision++
	t.record(EventProposed, "", "", now)
	return nil
}

// Approve moves an awaiting_approval timesheet to approved and records
// [EventApproved] at now. From any other status it returns
// [ErrInvalidTransition].
func (t *Timesheet) Approve(now time.Time) error {
	if t.status != StatusAwaitingApproval {
		return fmt.Errorf("%w: approve from %s", ErrInvalidTransition, t.status)
	}
	t.status = StatusApproved
	t.record(EventApproved, "", "", now)
	return nil
}

// Reject moves an awaiting_approval timesheet to rejected and records
// [EventRejected] with reason at now. From any other status it returns
// [ErrInvalidTransition].
func (t *Timesheet) Reject(reason string, now time.Time) error {
	if t.status != StatusAwaitingApproval {
		return fmt.Errorf("%w: reject from %s", ErrInvalidTransition, t.status)
	}
	t.status = StatusRejected
	t.record(EventRejected, "", reason, now)
	return nil
}

// PendingEntries returns the entries not yet created in target.
func (t *Timesheet) PendingEntries(target Target) []Entry {
	var out []Entry
	for _, e := range t.Entries() {
		if !e.SubmittedTo(target) {
			out = append(out, e)
		}
	}
	return out
}

// MarkEntrySubmitted records externalID, the id target gave the entry, and
// clears the entry's last error. It does not change the status; see
// [Timesheet.CompleteSubmission]. It returns [ErrInvalidTransition] unless the
// timesheet is approved, [ErrEntryNotFound] for an unknown entry, and
// [ErrUnknownTarget] for an unknown target.
func (t *Timesheet) MarkEntrySubmitted(entryID string, target Target, externalID string, now time.Time) error {
	e, err := t.approvedEntry(entryID)
	if err != nil {
		return err
	}
	switch target {
	case TargetTempo:
		e.TempoWorklogID = externalID
	case TargetMavenlink:
		e.MavenlinkEntryID = externalID
	default:
		return fmt.Errorf("%w: %q", ErrUnknownTarget, target)
	}
	e.LastError = ""
	return nil
}

// CompleteSubmission moves an approved timesheet to submitted and records
// [EventSubmitted] at now, once every entry is in every target in targets. It
// returns [ErrInvalidTransition] unless the timesheet is approved or when
// targets is empty, and [ErrEntriesPending] while an entry is missing from
// one of them.
func (t *Timesheet) CompleteSubmission(targets []Target, now time.Time) error {
	if t.status != StatusApproved {
		return fmt.Errorf("%w: submit from %s", ErrInvalidTransition, t.status)
	}
	if len(targets) == 0 {
		return fmt.Errorf("%w: no submit targets", ErrInvalidTransition)
	}
	for _, target := range targets {
		if n := len(t.PendingEntries(target)); n > 0 {
			return fmt.Errorf("%w: %d not in %s", ErrEntriesPending, n, target)
		}
	}
	t.status = StatusSubmitted
	t.record(EventSubmitted, "", "", now)
	return nil
}

// MarkEntryFailed records a failed submit attempt.
func (t *Timesheet) MarkEntryFailed(entryID string, cause error, now time.Time) error {
	e, err := t.approvedEntry(entryID)
	if err != nil {
		return err
	}
	e.SubmitAttempts++
	e.LastError = cause.Error()
	if e.SubmitAttempts == MaxSubmitAttempts {
		t.record(EventEntrySubmitFail, entryID, e.LastError, now)
	}
	return nil
}

func (t *Timesheet) approvedEntry(entryID string) (*Entry, error) {
	if t.status != StatusApproved {
		return nil, fmt.Errorf("%w: submit from %s", ErrInvalidTransition, t.status)
	}
	for i := range t.entries {
		if t.entries[i].ID == entryID {
			return &t.entries[i], nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrEntryNotFound, entryID)
}

// PullEvents returns and clears recorded events.
func (t *Timesheet) PullEvents() []kernel.DomainEvent {
	evs := t.events
	t.events = nil
	return evs
}

func (t *Timesheet) record(name, entryID, reason string, now time.Time) {
	t.events = append(t.events, Event{
		Name: name, TimesheetID: t.id, Week: t.week.String(), Revision: t.revision,
		EntryID: entryID, Reason: reason, At: now,
	})
}
