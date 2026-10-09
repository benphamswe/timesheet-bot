package application

import (
	"errors"

	"timesheet-bot/internal/shared/kernel"
)

// MavenlinkRule sends the time of matching Jira issues to one Mavenlink
// project (workspace) and, optionally, one task (story).
type MavenlinkRule struct {
	IssueKey    kernel.IssueKey // one issue, such as GCM-12; empty matches any issue
	JiraProject string          // a Jira project key, such as GCM; empty matches any project
	WorkspaceID int64
	StoryID     int64 // 0: no task
	RoleID      int64 // 0: no role
	Billable    *bool // nil: Mavenlink's default
}

// MavenlinkPlacement is where in Mavenlink one entry is logged.
type MavenlinkPlacement struct {
	WorkspaceID int64
	StoryID     int64 // 0: no task
	RoleID      int64 // 0: no role
	Billable    *bool // nil: Mavenlink's default
}

// PlacementResolver decides where in Mavenlink an entry is logged.
type PlacementResolver interface {
	Resolve(key kernel.IssueKey) (MavenlinkPlacement, error)
}

// ErrNoMavenlinkPlacement is returned by a [PlacementResolver] when no rule and
// no default places the issue.
var ErrNoMavenlinkPlacement = errors.New("no mavenlink project for issue")

var errResolveNotImplemented = errors.New("mavenlink mapping resolution is not implemented yet (TODO(owner) in MavenlinkMapping.Resolve)")

// MavenlinkMapping is the config's `mavenlink` block: an ordered list of rules
// and a fallback project and task.
type MavenlinkMapping struct {
	Rules              []MavenlinkRule
	DefaultWorkspaceID int64 // 0: no fallback project
	DefaultStoryID     int64 // 0: no fallback task
	DefaultRoleID      int64 // 0: no fallback role
}

var _ PlacementResolver = MavenlinkMapping{}

// Resolve returns where the time logged on key goes in Mavenlink.
//
// Inputs: key is a valid Jira issue key; its project is the part before the
// last hyphen (GCM for GCM-12). A rule matches when each of its non-empty
// fields matches: IssueKey equal to key, JiraProject equal to key's project.
//
// Decisions this function owns:
//   - Rule choice: first match in config order (simple, the owner orders the
//     list) or most specific (an IssueKey rule beats a JiraProject rule, which
//     beats a rule with neither, whatever the order).
//   - No match: fall back to DefaultWorkspaceID, DefaultStoryID and DefaultRoleID (no entry
//     is ever left behind, but time may land on the wrong project), or return
//     [ErrNoMavenlinkPlacement] (the entry fails and is retried after you fix
//     the config, so nothing is logged on a guess). With no default set
//     (DefaultWorkspaceID 0), it must return the error.
//
// An error marks only this entry as failed in the submit run; the others are
// still sent.
func (m MavenlinkMapping) Resolve(key kernel.IssueKey) (MavenlinkPlacement, error) {
	// TODO(owner): implement mapping resolution.
	_ = key
	return MavenlinkPlacement{}, errResolveNotImplemented
}
