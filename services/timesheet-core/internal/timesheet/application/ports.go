// Package application holds the Timesheet context's command handlers.
package application

import (
	"context"
	"errors"

	"timesheet-bot/internal/shared/kernel"
)

// ActivitySource provides the week's work evidence (Slack in production).
type ActivitySource interface {
	Collect(ctx context.Context, week kernel.WeekID) ([]kernel.WorkActivity, error)
}

// ReportLineSource provides the week's report lines that the collector's LLM
// already parsed (report.parser: llm; the activity file adapter in
// production). It fails when the source has no parsed lines for the week.
type ReportLineSource interface {
	ReportLines(ctx context.Context, week kernel.WeekID) ([]kernel.ReportLine, error)
}

// Issue is the subset of a Jira issue the timesheet needs.
type Issue struct {
	ID      int64
	Key     kernel.IssueKey
	Summary string
}

// ErrIssueNotFound is returned by an [IssueDirectory] when the issue key does
// not exist.
var ErrIssueNotFound = errors.New("issue not found")

// IssueDirectory resolves issue keys to Jira issues.
type IssueDirectory interface {
	Resolve(ctx context.Context, key kernel.IssueKey) (Issue, error)
}

// Worklog is what gets written to Tempo for one entry.
type Worklog struct {
	IssueID         int64
	Day             kernel.WorkDay
	StartTime       string // HH:MM:SS
	Seconds         int
	Description     string
	AuthorAccountID string
}

// WorklogGateway writes worklogs to the time tracking system (Tempo).
type WorklogGateway interface {
	Create(ctx context.Context, w Worklog) (worklogID string, err error)
}

// ErrUnauthorized is returned by a gateway when the time tracking system
// rejects its credentials (an expired session or a revoked token). A submit
// run stops at the first one, without counting it against the entry.
var ErrUnauthorized = errors.New("time tracking system rejected the credentials")

// TimeEntry is what gets written to Mavenlink for one entry.
type TimeEntry struct {
	EntryID     string // the timesheet entry's ID, stored as the external reference
	WorkspaceID int64
	StoryID     int64 // 0: no task
	RoleID      int64 // 0: no role
	Billable    *bool // nil: Mavenlink's default
	Day         kernel.WorkDay
	Minutes     int
	Notes       string
}

// RemoteTimeEntry is a Mavenlink time entry that a submit run created.
type RemoteTimeEntry struct {
	ID           string
	WorkspaceID  int64
	InSubmission bool // already part of an active timesheet submission
}

// TimeEntryGateway writes time entries and timesheet submissions to Mavenlink.
type TimeEntryGateway interface {
	// Created returns the time entries on week's days that a submit run
	// created, keyed by [TimeEntry.EntryID].
	Created(ctx context.Context, week kernel.WeekID) (map[string]RemoteTimeEntry, error)
	Create(ctx context.Context, e TimeEntry) (timeEntryID string, err error)
	// SubmitForApproval sends the time entries of one workspace for approval.
	SubmitForApproval(ctx context.Context, workspaceID int64, title string, timeEntryIDs []string) error
}
