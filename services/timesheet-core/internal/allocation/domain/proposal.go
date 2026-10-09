package domain

import "timesheet-bot/internal/shared/kernel"

// ProposedEntry is one suggested worklog line.
type ProposedEntry struct {
	IssueKey    kernel.IssueKey
	Day         kernel.WorkDay
	Duration    kernel.Duration
	Description string
}

// Proposal is the output of an allocation policy. It is not yet validated by the
// Timesheet aggregate; the timesheet application layer does that.
type Proposal struct {
	Policy    string
	Entries   []ProposedEntry
	Reasoning map[string]string // keyed by day (YYYY-MM-DD)
}
