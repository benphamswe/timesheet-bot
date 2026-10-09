package application_test

import (
	"errors"
	"testing"

	"timesheet-bot/internal/shared/kernel"
	"timesheet-bot/internal/timesheet/application"
)

// Owned by the owner (MavenlinkMapping.Resolve is TODO(owner)); the body is
// not edited. The cases assume most-specific-wins and an error when nothing
// matches and there is no default; adjust them to the rule you choose.
func TestMavenlinkMappingResolve(t *testing.T) {
	t.Skip("TODO(owner): MavenlinkMapping.Resolve not implemented yet; remove this skip when it is")
	t.Parallel()
	billable := false
	rules := []application.MavenlinkRule{
		{JiraProject: "GCM", WorkspaceID: 7, StoryID: 70},
		{IssueKey: "GCM-12", WorkspaceID: 9, RoleID: 5, Billable: &billable},
		{JiraProject: "HUB", WorkspaceID: 8},
	}
	tests := []struct {
		name    string
		mapping application.MavenlinkMapping
		key     kernel.IssueKey
		want    application.MavenlinkPlacement
		wantErr error
	}{
		{name: "project rule", mapping: application.MavenlinkMapping{Rules: rules}, key: "GCM-3",
			want: application.MavenlinkPlacement{WorkspaceID: 7, StoryID: 70}},
		{name: "issue rule beats project rule", mapping: application.MavenlinkMapping{Rules: rules}, key: "GCM-12",
			want: application.MavenlinkPlacement{WorkspaceID: 9, RoleID: 5, Billable: &billable}},
		{name: "default when nothing matches",
			mapping: application.MavenlinkMapping{Rules: rules, DefaultWorkspaceID: 1, DefaultStoryID: 10, DefaultRoleID: 2},
			key:     "OPS-1", want: application.MavenlinkPlacement{WorkspaceID: 1, StoryID: 10, RoleID: 2}},
		{name: "no match and no default", mapping: application.MavenlinkMapping{Rules: rules}, key: "OPS-1",
			wantErr: application.ErrNoMavenlinkPlacement},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.mapping.Resolve(tt.key)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Resolve(%s) error = %v, want %v", tt.key, err, tt.wantErr)
			}
			if got.WorkspaceID != tt.want.WorkspaceID || got.StoryID != tt.want.StoryID || got.RoleID != tt.want.RoleID ||
				(got.Billable == nil) != (tt.want.Billable == nil) {
				t.Errorf("Resolve(%s) = %+v, want %+v", tt.key, got, tt.want)
			}
		})
	}
}
