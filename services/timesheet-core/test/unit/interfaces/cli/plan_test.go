package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	alloc "timesheet-bot/internal/allocation/domain"
	"timesheet-bot/internal/interfaces/cli"
	"timesheet-bot/internal/shared/kernel"
	"timesheet-bot/internal/timesheet/application"
	"timesheet-bot/internal/timesheet/infrastructure/memory"
)

type fixedClock struct{}

func (fixedClock) Now() time.Time { return time.Date(2026, 10, 9, 16, 0, 0, 0, time.UTC) }

type noActivity struct{}

func (noActivity) Collect(context.Context, kernel.WeekID) ([]kernel.WorkActivity, error) {
	return nil, nil
}

type lines []kernel.ReportLine

func (l lines) ReportLines(context.Context, kernel.WeekID) ([]kernel.ReportLine, error) {
	return l, nil
}

// planApp is report mode with LLM-parsed lines and Mavenlink as the target:
// Atlas has a rule, Beacon falls back to the default project.
func planApp() *cli.App {
	return &cli.App{
		Location: time.UTC,
		Close:    func() {},
		Propose: &application.ProposeHandler{
			Repo:   memory.NewRepository(),
			Source: noActivity{},
			Reports: lines{
				{Date: "2026-10-05", Project: "Atlas", Minutes: 240, Notes: "API review"},
				{Date: "2026-10-05", Project: "Beacon", Minutes: 300, Notes: "hotfix"},
			},
			Policies:      []alloc.Policy{alloc.ReportPolicy{Precomputed: true, Location: time.UTC}},
			Clock:         fixedClock{},
			UserAccountID: "U1",
			DailyCap:      kernel.MustDuration(480),
			Placements: application.MavenlinkMapping{
				Rules:              []application.MavenlinkRule{{Project: "Atlas", WorkspaceID: 7, StoryID: 70}},
				DefaultWorkspaceID: 1,
			},
		},
	}
}

func TestPlanDryRunShowsMavenlinkProjectsAndWarnings(t *testing.T) {
	t.Parallel()
	root := cli.NewRoot(func(context.Context, string, cli.Needs) (*cli.App, error) { return planApp(), nil }, nil)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"plan", "--dry-run", "--week", "2026-W41"})

	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatalf("plan --dry-run error = %v, want nil; output:\n%s", err, out.String())
	}
	got := out.String()
	for _, want := range []string{
		"MAVENLINK",
		"Atlas          4h    project 7 task 70",
		"Beacon         5h    project 1 (default)",
		`warning: "Beacon" has no mavenlink.mappings rule; its time goes to default_workspace_id 1`,
		"warning: 2026-10-05: 9h reported, over the daily cap of 8h; kept as reported, not scaled",
		"dry run: nothing saved",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("plan --dry-run output is missing %q; got:\n%s", want, got)
		}
	}
}

func TestPlanWithoutMavenlinkHasNoColumn(t *testing.T) {
	t.Parallel()
	app := planApp()
	app.Propose.Placements = nil
	root := cli.NewRoot(func(context.Context, string, cli.Needs) (*cli.App, error) { return app, nil }, nil)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"plan", "--dry-run", "--week", "2026-W41"})

	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatalf("plan --dry-run error = %v, want nil", err)
	}
	if strings.Contains(out.String(), "MAVENLINK") || strings.Contains(out.String(), "default_workspace_id") {
		t.Errorf("plan output without Mavenlink = \n%s\nwant no MAVENLINK column and no fallback warning", out.String())
	}
}
