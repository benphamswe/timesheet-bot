package domain

import "timesheet-bot/internal/shared/kernel"

// Entry is one worklog line: time spent on an issue on a day.
type Entry struct {
	ID               string
	IssueKey         kernel.IssueKey
	IssueID          int64 // Jira numeric id, required by Tempo
	Day              kernel.WorkDay
	Duration         kernel.Duration
	Description      string
	TempoWorklogID   string // empty until created in Tempo
	MavenlinkEntryID string // empty until created in Mavenlink
	SubmitAttempts   int    // failed attempts, summed over every target
	LastError        string
}

// SubmittedTo reports whether e has been created in target, that is, whether
// it has that target's id. It is false for an unknown target.
func (e Entry) SubmittedTo(target Target) bool {
	switch target {
	case TargetTempo:
		return e.TempoWorklogID != ""
	case TargetMavenlink:
		return e.MavenlinkEntryID != ""
	}
	return false
}
