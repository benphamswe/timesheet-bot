package application_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	alloc "timesheet-bot/internal/allocation/domain"
	"timesheet-bot/internal/shared/kernel"
	"timesheet-bot/internal/timesheet/application"
	"timesheet-bot/internal/timesheet/domain"
	"timesheet-bot/internal/timesheet/infrastructure/memory"
)

// lineParser turns every message into the report lines registered for its text.
type lineParser map[string][]alloc.ReportLine

func (p lineParser) Parse(msg kernel.WorkActivity, _ *time.Location) ([]alloc.ReportLine, error) {
	return p[msg.Text], nil
}

// The same summary arrives from the report channel and from the self-DM.
var weekReport = lineParser{
	"summary": {
		{Date: "2026-10-05", Project: "Atlas", Minutes: 240, Notes: "API review"},
		{Date: "2026-10-05", Project: "Atlas", Minutes: 300, Notes: "release"},
		{Date: "2026-10-06", Project: "Beacon", Minutes: 60, Notes: "standup"},
	},
}

func newReportPropose(repo domain.Repository) *application.ProposeHandler {
	return &application.ProposeHandler{
		Repo: repo,
		Source: fakeSource{acts: []kernel.WorkActivity{
			{Day: mon, ChannelName: "timesheet", Text: "summary"},
			{Day: mon, ChannelID: "D01", Text: "summary"},
		}},
		Policies:      []alloc.Policy{alloc.ReportPolicy{Parser: weekReport, Location: time.UTC}},
		Clock:         fixedClock{},
		UserAccountID: "U1",
		DailyCap:      kernel.MustDuration(480),
	}
}

func TestProposeReportModeWithoutJira(t *testing.T) {
	t.Parallel()
	repo := memory.NewRepository()

	res, err := newReportPropose(repo).Handle(t.Context(), application.ProposeCommand{Week: week})
	if err != nil {
		t.Fatalf("Handle(%v) error = %v, want nil", week, err)
	}
	var got []string
	for _, e := range res.Timesheet.Entries() {
		got = append(got, e.Day.String()+" "+e.Project+" "+e.Duration.String()+" "+e.Description)
	}
	want := []string{"2026-10-05 Atlas 4h API review", "2026-10-05 Atlas 5h release", "2026-10-06 Beacon 1h standup"}
	if !slices.Equal(got, want) {
		t.Errorf("Handle(%v) entries = %q, want %q (duplicates from the self-DM dropped)", week, got, want)
	}
	if res.Timesheet.Policy() != "report" || res.Timesheet.Status() != domain.StatusAwaitingApproval {
		t.Errorf("Handle(%v) policy, status = %s, %s, want report, %s", week, res.Timesheet.Policy(), res.Timesheet.Status(), domain.StatusAwaitingApproval)
	}
	if w := res.Proposal.Warnings; len(w) != 1 || !strings.Contains(w[0], "2026-10-05: 9h reported, over the daily cap of 8h") {
		t.Errorf("Handle(%v) warnings = %q, want the Monday over-cap warning", week, w)
	}
	if _, err := repo.Get(t.Context(), "U1", week); err != nil {
		t.Errorf("repo.Get(U1, %v) error = %v, want the proposal saved", week, err)
	}
}

func TestProposeReportModeDryRunKeepsWarnings(t *testing.T) {
	t.Parallel()
	res, err := newReportPropose(memory.NewRepository()).Handle(t.Context(), application.ProposeCommand{Week: week, DryRun: true})
	if err != nil {
		t.Fatalf("Handle(%v, dry run) error = %v, want nil", week, err)
	}
	if len(res.Proposal.Warnings) != 1 {
		t.Errorf("Handle(%v, dry run) warnings = %q, want the over-cap warning", week, res.Proposal.Warnings)
	}
}

// approveReportWeek stores the report proposal, approved.
func approveReportWeek(t *testing.T, repo domain.Repository) {
	t.Helper()
	if _, err := newReportPropose(repo).Handle(t.Context(), application.ProposeCommand{Week: week}); err != nil {
		t.Fatalf("propose %v: %v", week, err)
	}
	if err := (&application.TransitionHandler{Repo: repo, Clock: fixedClock{}, UserAccountID: "U1"}).Approve(t.Context(), week); err != nil {
		t.Fatalf("approve %v: %v", week, err)
	}
}

func TestSubmitReportEntriesToMavenlinkByProject(t *testing.T) {
	t.Parallel()
	repo := memory.NewRepository()
	approveReportWeek(t, repo)
	var asked []string
	ml := &fakeTimeEntries{}
	h := &application.SubmitHandler{
		Repo: repo, Clock: fixedClock{}, UserAccountID: "U1",
		Mavenlink: &application.MavenlinkSubmit{Gateway: ml, Placements: stubPlacements{projects: &asked}, SubmitForApproval: true},
	}

	res, err := h.Handle(t.Context(), week)
	if err != nil {
		t.Fatalf("Handle(%v) error = %v, want nil", week, err)
	}
	if !slices.Equal(asked, []string{"Atlas", "Atlas", "Beacon"}) {
		t.Errorf("resolver got projects %q, want Atlas, Atlas, Beacon", asked)
	}
	// stubPlacements places Atlas only, so Beacon fails and holds the submission.
	if res.Submitted != 2 || res.Failed != 1 || res.Status != domain.StatusApproved {
		t.Errorf("Handle(%v) = %+v, want 2 submitted, 1 failed, approved", week, res)
	}
	if len(ml.created) != 2 || ml.created[0].Notes != "API review" || ml.created[0].Minutes != 240 || ml.created[0].WorkspaceID != 7 {
		t.Errorf("Mavenlink time entries = %+v, want the two Atlas lines in project 7", ml.created)
	}
}

func TestSubmitReportEntryToTempoFails(t *testing.T) {
	t.Parallel()
	repo := memory.NewRepository()
	approveReportWeek(t, repo)
	gw := &fakeGateway{}
	h := &application.SubmitHandler{Repo: repo, Gateway: gw, Clock: fixedClock{}, UserAccountID: "U1"}

	res, err := h.Handle(t.Context(), week)
	if err != nil {
		t.Fatalf("Handle(%v) error = %v, want nil", week, err)
	}
	if len(gw.created) != 0 || res.Failed != 3 || res.Status != domain.StatusApproved {
		t.Errorf("Handle(%v) = %+v with %d Tempo worklogs, want every report entry failed and none sent", week, res, len(gw.created))
	}
}

func TestProposeJiraKeyWithoutJiraFails(t *testing.T) {
	t.Parallel()
	h := newPropose(memory.NewRepository(), fakeSource{acts: []kernel.WorkActivity{activity(mon, "GCM-1")}}, alloc.RulePolicy{})
	h.Issues = nil
	_, err := h.Handle(context.Background(), application.ProposeCommand{Week: week, DryRun: true})
	if !errors.Is(err, application.ErrNoValidProposal) {
		t.Errorf("Handle() with an issue key and no Jira error = %v, want ErrNoValidProposal", err)
	}
}

// fakeReportLines returns lines, or err when set, as the activity file adapter does.
type fakeReportLines struct {
	lines []kernel.ReportLine
	err   error
}

func (f fakeReportLines) ReportLines(context.Context, kernel.WeekID) ([]kernel.ReportLine, error) {
	return f.lines, f.err
}

func newLLMPropose(repo domain.Repository, reports application.ReportLineSource) *application.ProposeHandler {
	h := newReportPropose(repo)
	h.Reports = reports
	h.Policies = []alloc.Policy{alloc.ReportPolicy{Precomputed: true, Location: time.UTC}}
	return h
}

// The collector's LLM delivered the same lines the manual parser would have
// produced (both copies of the summary), so the timesheet is the same.
func TestProposeLLMLinesMatchManualParser(t *testing.T) {
	t.Parallel()
	var lines []kernel.ReportLine
	for _, l := range append(weekReport["summary"], weekReport["summary"]...) {
		l.SourceTS = "1791135000.000100"
		lines = append(lines, l)
	}
	manual, err := newReportPropose(memory.NewRepository()).Handle(t.Context(), application.ProposeCommand{Week: week, DryRun: true})
	if err != nil {
		t.Fatalf("manual Handle() error = %v, want nil", err)
	}
	llm, err := newLLMPropose(memory.NewRepository(), fakeReportLines{lines: lines}).Handle(t.Context(), application.ProposeCommand{Week: week, DryRun: true})
	if err != nil {
		t.Fatalf("llm Handle() error = %v, want nil", err)
	}
	describe := func(ts *domain.Timesheet) []string {
		var out []string
		for _, e := range ts.Entries() {
			out = append(out, e.Day.String()+" "+e.Project+" "+e.Duration.String()+" "+e.Description)
		}
		return out
	}
	if got, want := describe(llm.Timesheet), describe(manual.Timesheet); !slices.Equal(got, want) {
		t.Errorf("llm entries = %q, want the manual parser's %q", got, want)
	}
	if !slices.Equal(llm.Proposal.Warnings, manual.Proposal.Warnings) {
		t.Errorf("llm warnings = %q, want the manual parser's %q", llm.Proposal.Warnings, manual.Proposal.Warnings)
	}
}

func TestProposeLLMFailsWithoutReportLines(t *testing.T) {
	t.Parallel()
	errNoLines := errors.New("activity file has no report_lines: re-run the collector with report.parser: llm")
	repo := memory.NewRepository()
	_, err := newLLMPropose(repo, fakeReportLines{err: errNoLines}).Handle(t.Context(), application.ProposeCommand{Week: week})
	if !errors.Is(err, errNoLines) {
		t.Errorf("Handle() error = %v, want the report lines error", err)
	}
	if _, err := repo.Get(t.Context(), "U1", week); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("repo.Get() after a failed llm propose error = %v, want ErrNotFound (nothing saved)", err)
	}
}

func TestProposeShowsMavenlinkPlacements(t *testing.T) {
	t.Parallel()
	h := newReportPropose(memory.NewRepository())
	h.Placements = application.MavenlinkMapping{
		Rules:              []application.MavenlinkRule{{Project: "atlas", WorkspaceID: 7, StoryID: 70}},
		DefaultWorkspaceID: 1,
	}

	res, err := h.Handle(t.Context(), application.ProposeCommand{Week: week, DryRun: true})
	if err != nil {
		t.Fatalf("Handle(%v, dry run) error = %v, want nil", week, err)
	}
	for _, e := range res.Timesheet.Entries() {
		p := res.Placements[e.ID]
		wantWS, wantFallback := int64(7), false
		if e.Project == "Beacon" {
			wantWS, wantFallback = 1, true
		}
		if p.WorkspaceID != wantWS || p.Fallback != wantFallback {
			t.Errorf("Placements[%s] (%s) = %+v, want workspace %d, fallback %v", e.ID, e.Project, p, wantWS, wantFallback)
		}
	}
	var fallbackWarnings []string
	for _, w := range res.Proposal.Warnings {
		if strings.Contains(w, "no mavenlink.mappings rule") {
			fallbackWarnings = append(fallbackWarnings, w)
		}
	}
	if want := []string{`"Beacon" has no mavenlink.mappings rule; its time goes to default_workspace_id 1`}; !slices.Equal(fallbackWarnings, want) {
		t.Errorf("fallback warnings = %q, want %q (one per project)", fallbackWarnings, want)
	}
}

func TestProposeFailsWhenAnEntryHasNoMavenlinkProject(t *testing.T) {
	t.Parallel()
	repo := memory.NewRepository()
	h := newReportPropose(repo)
	h.Placements = application.MavenlinkMapping{Rules: []application.MavenlinkRule{{Project: "Atlas", WorkspaceID: 7}}}

	_, err := h.Handle(t.Context(), application.ProposeCommand{Week: week})
	if !errors.Is(err, application.ErrNoMavenlinkPlacement) || !strings.Contains(err.Error(), `project "Beacon"`) {
		t.Fatalf("Handle() error = %v, want the no-mapping error for Beacon", err)
	}
	if strings.Contains(err.Error(), "standup") {
		t.Errorf("Handle() error = %q, want it to omit the entry's notes", err)
	}
	if _, err := repo.Get(t.Context(), "U1", week); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("repo.Get() after the failed plan error = %v, want ErrNotFound (nothing saved)", err)
	}
}
