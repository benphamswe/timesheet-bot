package application

import (
	"errors"
	"fmt"
	"strings"

	alloc "timesheet-bot/internal/allocation/domain"
	"timesheet-bot/internal/shared/kernel"
)

// MavenlinkRule sends the time of matching entries to one Mavenlink project
// (workspace) and, optionally, one task (story). jira_keys mode matches
// IssueKey and JiraProject; report mode matches Project. config.Load rejects a
// rule that sets Project together with IssueKey or JiraProject, and two rules
// with the same IssueKey or the same normalized Project.
type MavenlinkRule struct {
	IssueKey    kernel.IssueKey // one issue, such as GCM-12, any case
	JiraProject string          // a Jira project key, such as GCM: the key's prefix before the first '-'
	Project     string          // report mode: a project name as written in the report, compared after alloc.NormalizeReportText
	WorkspaceID int64
	StoryID     int64 // 0: the default task in the default project, or no task
	RoleID      int64 // 0: the default role, or no role
	Billable    *bool // nil: Mavenlink's default
}

// MavenlinkPlacement is where in Mavenlink one entry is logged.
type MavenlinkPlacement struct {
	WorkspaceID int64
	StoryID     int64 // 0: no task
	RoleID      int64 // 0: no role
	Billable    *bool // nil: Mavenlink's default
	Fallback    bool  // no rule matched; the default project was used
}

// PlacementResolver decides where in Mavenlink an entry is logged. An entry
// has either a Jira issue key (jira_keys mode) or a report project name
// (report mode); the other argument is empty.
type PlacementResolver interface {
	Resolve(key kernel.IssueKey, project string) (MavenlinkPlacement, error)
}

// ErrNoMavenlinkPlacement is wrapped in the error a [PlacementResolver]
// returns when no rule and no default places the entry.
var ErrNoMavenlinkPlacement = errors.New("no Mavenlink mapping")

// MavenlinkMapping is the config's `mavenlink` block: an ordered list of rules
// and a fallback project, task and role.
type MavenlinkMapping struct {
	Rules              []MavenlinkRule
	DefaultWorkspaceID int64 // 0: no fallback project
	DefaultStoryID     int64 // used only in the default project
	DefaultRoleID      int64 // 0: no fallback role
}

var _ PlacementResolver = MavenlinkMapping{}

// Rule specificities, most specific highest.
const (
	noMatch = iota
	byJiraProject
	byProject
	byIssueKey
)

// Resolve returns where the time of one entry goes in Mavenlink. Exactly one
// of key (jira_keys mode) and project (report mode) is set.
//
// Matching:
//   - Project rules compare both names after alloc.NormalizeReportText: case
//     ignored, ends trimmed, inner white space collapsed. Why: the same
//     function decides which report lines are duplicates, so a project that
//     dedups as one name also maps as one name.
//   - IssueKey rules compare the whole key, case ignored. Why: an issue is
//     exact; only the case of a hand-typed key may differ.
//   - JiraProject rules compare the key's prefix before the first '-', case
//     ignored. Why: that prefix is the Jira project, so one rule covers all of
//     its issues.
//
// Precedence: the most specific matching rule wins, IssueKey over Project over
// JiraProject; among rules of the same kind, the first in config order wins.
// Why: a narrow rule is a deliberate exception to a broad one, whatever its
// position, and config.Load rejects the same issue key or project listed
// twice, so "first wins" only orders rules that do not conflict.
//
// No match: with DefaultWorkspaceID set, the default project is returned with
// Fallback set, so plan can warn. Why: nothing is left behind, but the owner
// sees every entry that landed on a guess before submitting. Without it, the
// error names the project or issue key (never notes). Why: logging time on a
// wrong project is worse than stopping.
//
// Task, role, billable:
//   - RoleID is the rule's when set, else DefaultRoleID, else 0 (omitted).
//     Why: a role is about the person, the same on every project.
//   - StoryID is the rule's when set, else DefaultStoryID only when the
//     resolved project is the default project, else 0 (omitted). Why: a task
//     belongs to one project; the default task exists only in the default
//     project.
//   - Billable is the rule's, else nil, leaving Mavenlink's own default.
func (m MavenlinkMapping) Resolve(key kernel.IssueKey, project string) (MavenlinkPlacement, error) {
	best, bestRank := -1, noMatch
	for i, r := range m.Rules {
		// Strictly greater: an equal rank later in the list never replaces the first.
		if rank := r.specificity(key, project); rank > bestRank {
			best, bestRank = i, rank
		}
	}
	if best >= 0 {
		r := m.Rules[best]
		return m.place(r.WorkspaceID, r.StoryID, r.RoleID, r.Billable, false), nil
	}
	if m.DefaultWorkspaceID > 0 {
		return m.place(m.DefaultWorkspaceID, 0, 0, nil, true), nil
	}
	if key != "" {
		return MavenlinkPlacement{}, fmt.Errorf("%w for issue %q: add a mavenlink.mappings rule or set default_workspace_id", ErrNoMavenlinkPlacement, key)
	}
	return MavenlinkPlacement{}, fmt.Errorf("%w for project %q: add a mavenlink.mappings rule or set default_workspace_id", ErrNoMavenlinkPlacement, project)
}

func (m MavenlinkMapping) place(workspace, story, role int64, billable *bool, fallback bool) MavenlinkPlacement {
	if role <= 0 {
		role = m.DefaultRoleID
	}
	if story <= 0 && workspace == m.DefaultWorkspaceID {
		story = m.DefaultStoryID
	}
	return MavenlinkPlacement{WorkspaceID: workspace, StoryID: story, RoleID: role, Billable: billable, Fallback: fallback}
}

// specificity returns how specifically r matches the entry, or noMatch. A rule
// counts as its most specific field, and every field it sets must match.
func (r MavenlinkRule) specificity(key kernel.IssueKey, project string) int {
	switch {
	case r.IssueKey != "":
		if key == "" || !strings.EqualFold(string(r.IssueKey), string(key)) ||
			(r.JiraProject != "" && !strings.EqualFold(r.JiraProject, jiraProject(key))) {
			return noMatch
		}
		return byIssueKey
	case r.Project != "":
		if project == "" || alloc.NormalizeReportText(r.Project) != alloc.NormalizeReportText(project) {
			return noMatch
		}
		return byProject
	case r.JiraProject != "":
		if key == "" || !strings.EqualFold(r.JiraProject, jiraProject(key)) {
			return noMatch
		}
		return byJiraProject
	}
	return noMatch
}

// jiraProject returns the part of key before the first '-'.
func jiraProject(key kernel.IssueKey) string {
	p, _, _ := strings.Cut(string(key), "-")
	return p
}
