package application_test

import (
	"errors"
	"strings"
	"testing"

	"timesheet-bot/internal/shared/kernel"
	"timesheet-bot/internal/timesheet/application"
)

func TestMavenlinkMappingResolve(t *testing.T) {
	t.Parallel()
	billable := false
	rules := []application.MavenlinkRule{
		0: {JiraProject: "GCM", WorkspaceID: 7, StoryID: 70},
		1: {IssueKey: "GCM-12", WorkspaceID: 9, RoleID: 5, Billable: &billable},
		2: {JiraProject: "HUB", WorkspaceID: 8},
		3: {Project: "Atlas", WorkspaceID: 11, StoryID: 110, RoleID: 6},
		4: {Project: "Beacon  Mobile", WorkspaceID: 12},
		5: {JiraProject: "GCM", WorkspaceID: 99}, // same kind as rule 0, later: never wins
		6: {Project: "Delta", WorkspaceID: 1},    // the default project; no task of its own
	}
	plain := application.MavenlinkMapping{Rules: rules}
	withDefault := application.MavenlinkMapping{Rules: rules, DefaultWorkspaceID: 1, DefaultStoryID: 10, DefaultRoleID: 2}
	tests := []struct {
		name    string
		mapping application.MavenlinkMapping
		key     kernel.IssueKey
		project string
		want    application.MavenlinkPlacement
		wantErr string // part of the error; empty means no error
	}{
		{name: "jira_project matches the key prefix", mapping: plain, key: "GCM-3",
			want: application.MavenlinkPlacement{WorkspaceID: 7, StoryID: 70}},
		{name: "issue_key beats jira_project wherever it is", mapping: plain, key: "GCM-12",
			want: application.MavenlinkPlacement{WorkspaceID: 9, RoleID: 5, Billable: &billable}},
		{name: "issue_key ignores case", mapping: plain, key: "gcm-12",
			want: application.MavenlinkPlacement{WorkspaceID: 9, RoleID: 5, Billable: &billable}},
		{name: "first jira_project rule wins over a later one", mapping: plain, key: "GCM-40",
			want: application.MavenlinkPlacement{WorkspaceID: 7, StoryID: 70}},
		{name: "jira_project needs the whole prefix", mapping: plain, key: "GCMX-1",
			wantErr: `no Mavenlink mapping for issue "GCMX-1": add a mavenlink.mappings rule or set default_workspace_id`},
		{name: "project, same case", mapping: plain, project: "Atlas",
			want: application.MavenlinkPlacement{WorkspaceID: 11, StoryID: 110, RoleID: 6}},
		{name: "project ignores case and spaces", mapping: plain, project: "  beacon MOBILE ",
			want: application.MavenlinkPlacement{WorkspaceID: 12}},
		{name: "a project name never matches a Jira rule", mapping: plain, project: "GCM",
			wantErr: `no Mavenlink mapping for project "GCM"`},
		{name: "no match and no default names the project", mapping: plain, project: "Zephyr",
			wantErr: `no Mavenlink mapping for project "Zephyr": add a mavenlink.mappings rule or set default_workspace_id`},
		{name: "no match falls back to the default with the flag", mapping: withDefault, project: "Zephyr",
			want: application.MavenlinkPlacement{WorkspaceID: 1, StoryID: 10, RoleID: 2, Fallback: true}},
		{name: "default role fills a rule without one; default task stays in the default project", mapping: withDefault, project: "Beacon Mobile",
			want: application.MavenlinkPlacement{WorkspaceID: 12, RoleID: 2}},
		{name: "a rule's own task and role win over the defaults", mapping: withDefault, project: "Atlas",
			want: application.MavenlinkPlacement{WorkspaceID: 11, StoryID: 110, RoleID: 6}},
		{name: "a rule pointing at the default project gets the default task", mapping: withDefault, project: "delta",
			want: application.MavenlinkPlacement{WorkspaceID: 1, StoryID: 10, RoleID: 2}},
		{name: "a matched rule is not a fallback even with a default", mapping: withDefault, key: "HUB-1",
			want: application.MavenlinkPlacement{WorkspaceID: 8, RoleID: 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.mapping.Resolve(tt.key, tt.project)
			if tt.wantErr != "" {
				if !errors.Is(err, application.ErrNoMavenlinkPlacement) || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Resolve(%q, %q) error = %v, want ErrNoMavenlinkPlacement saying %q", tt.key, tt.project, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve(%q, %q) error = %v, want nil", tt.key, tt.project, err)
			}
			if got.WorkspaceID != tt.want.WorkspaceID || got.StoryID != tt.want.StoryID || got.RoleID != tt.want.RoleID ||
				got.Fallback != tt.want.Fallback || !sameBillable(got.Billable, tt.want.Billable) {
				t.Errorf("Resolve(%q, %q) = %+v, want %+v", tt.key, tt.project, got, tt.want)
			}
		})
	}
}

func sameBillable(a, b *bool) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
