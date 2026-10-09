package domain

// Status is a Timesheet's lifecycle state. The legal transitions are
// draft -> awaiting_approval -> approved -> submitted, and
// awaiting_approval -> rejected. The string values are persisted.
type Status string

// Timesheet lifecycle states.
const (
	StatusDraft            Status = "draft"
	StatusAwaitingApproval Status = "awaiting_approval"
	StatusApproved         Status = "approved"
	StatusSubmitted        Status = "submitted"
	StatusRejected         Status = "rejected"
)
