package domain

import (
	"time"

	"timesheet-bot/internal/shared/kernel"
)

// Event names, stored in [Event.Name]. They are persisted, so changing a
// value breaks stored events.
const (
	// EventProposed is recorded when a draft is sent for approval.
	EventProposed = "TimesheetProposed"
	// EventApproved is recorded when a proposed timesheet is approved.
	EventApproved = "TimesheetApproved"
	// EventRejected is recorded when a proposed timesheet is rejected; the
	// event's Reason holds the rejection reason.
	EventRejected = "TimesheetRejected"
	// EventSubmitted is recorded when every entry is in every submit target
	// and the timesheet is completed.
	EventSubmitted = "TimesheetSubmitted"
	// EventEntrySubmitFail is recorded when an entry fails its final
	// (MaxSubmitAttempts-th) submit attempt; the event's EntryID and Reason
	// identify the entry and its last error.
	EventEntrySubmitFail = "EntrySubmissionFailed"
)

// Event is the single event shape emitted by the Timesheet aggregate.
type Event struct {
	Name        string    `json:"name"`
	TimesheetID string    `json:"timesheet_id"`
	Week        string    `json:"week"`
	Revision    int       `json:"revision"`
	EntryID     string    `json:"entry_id,omitempty"`
	Reason      string    `json:"reason,omitempty"`
	At          time.Time `json:"at"`
}

// EventName returns e.Name, one of the Event* constants.
func (e Event) EventName() string { return e.Name }

// AggregateID returns the ID of the timesheet that recorded e.
func (e Event) AggregateID() string { return e.TimesheetID }

// OccurredAt returns the time passed to the method that recorded e.
func (e Event) OccurredAt() time.Time { return e.At }

var _ kernel.DomainEvent = Event{}
