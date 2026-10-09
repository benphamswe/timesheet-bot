package domain

import "errors"

// Sentinel errors returned by the Timesheet aggregate and its [Repository].
// Callers match them with [errors.Is]; most are wrapped with detail.
var (
	// ErrNotDraft is returned when entries are added to a timesheet that is
	// no longer in StatusDraft.
	ErrNotDraft = errors.New("timesheet is not a draft")
	// ErrInvalidTransition is returned when a lifecycle method is called from
	// a status that does not allow it.
	ErrInvalidTransition = errors.New("invalid status transition")
	// ErrDayOutsideWeek is returned for an entry dated outside the
	// timesheet's ISO week.
	ErrDayOutsideWeek = errors.New("day is outside the timesheet week")
	// ErrDayOverCap is returned when an entry would push its day's total
	// above the daily cap.
	ErrDayOverCap = errors.New("day total exceeds daily cap")
	// ErrDuplicateEntry is returned when the timesheet already has an entry
	// for the same issue on the same day.
	ErrDuplicateEntry = errors.New("entry for this issue and day already exists")
	// ErrEmptyTimesheet is returned when a timesheet with no entries is
	// proposed.
	ErrEmptyTimesheet = errors.New("timesheet has no entries")
	// ErrEntryNotFound is returned when no entry has the given entry ID.
	ErrEntryNotFound = errors.New("entry not found")
	// ErrUnknownTarget is returned for a submit target the timesheet does
	// not know.
	ErrUnknownTarget = errors.New("unknown submit target")
	// ErrEntriesPending is returned when a timesheet is completed while an
	// entry is not yet in every submit target.
	ErrEntriesPending = errors.New("entries are not yet in every submit target")
	// ErrNotFound is returned by a Repository when the requested timesheet
	// does not exist.
	ErrNotFound = errors.New("timesheet not found")
	// ErrConcurrentModification is returned by a Repository when the stored
	// version no longer matches the version the timesheet was loaded with.
	ErrConcurrentModification = errors.New("timesheet was modified concurrently")
)
