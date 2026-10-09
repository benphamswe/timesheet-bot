package application_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	alloc "timesheet-bot/internal/allocation/domain"
	"timesheet-bot/internal/shared/kernel"
	"timesheet-bot/internal/timesheet/application"
	"timesheet-bot/internal/timesheet/domain"
	"timesheet-bot/internal/timesheet/infrastructure/memory"
)

var (
	week = kernel.WeekID{Year: 2026, Week: 41}
	mon  = week.Days()[0]
	tue  = week.Days()[1]

	errDB    = errors.New("db down")
	errSlack = errors.New("slack down")
	errJira  = errors.New("jira 503")
)

type fixedClock struct{}

func (fixedClock) Now() time.Time { return time.Date(2026, 10, 9, 16, 0, 0, 0, time.UTC) }

// fakeSource returns acts, or err when set.
type fakeSource struct {
	acts []kernel.WorkActivity
	err  error
}

func (f fakeSource) Collect(context.Context, kernel.WeekID) ([]kernel.WorkActivity, error) {
	return f.acts, f.err
}

// fakeIssues resolves the keys in ids. A key in errFor fails with that error;
// any other key is ErrIssueNotFound.
type fakeIssues struct {
	ids    map[kernel.IssueKey]int64
	errFor map[kernel.IssueKey]error
}

func (f fakeIssues) Resolve(_ context.Context, k kernel.IssueKey) (application.Issue, error) {
	if err := f.errFor[k]; err != nil {
		return application.Issue{}, err
	}
	id, ok := f.ids[k]
	if !ok {
		return application.Issue{}, fmt.Errorf("%w: %s", application.ErrIssueNotFound, k)
	}
	return application.Issue{ID: id, Key: k}, nil
}

var knownIssues = fakeIssues{ids: map[kernel.IssueKey]int64{"GCM-1": 101, "GCM-2": 102}}

// fakePolicy returns prop and err, recording every input it was given.
type fakePolicy struct {
	name string
	prop alloc.Proposal
	err  error
	seen []alloc.Input
}

func (p *fakePolicy) Name() string { return p.name }
func (p *fakePolicy) Allocate(_ context.Context, in alloc.Input) (alloc.Proposal, error) {
	p.seen = append(p.seen, in)
	return p.prop, p.err
}

// fakeGateway records created worklogs and fails for issue ids in failFor.
type fakeGateway struct {
	failFor map[int64]bool
	created []application.Worklog
}

func (g *fakeGateway) Create(_ context.Context, w application.Worklog) (string, error) {
	if g.failFor[w.IssueID] {
		return "", errors.New("tempo 500")
	}
	g.created = append(g.created, w)
	return fmt.Sprintf("w-%d", len(g.created)), nil
}

// failingRepo is a memory.Repository whose methods fail with the set error.
type failingRepo struct {
	*memory.Repository
	getErr, saveErr, deleteErr error
}

func (r *failingRepo) Get(ctx context.Context, user string, w kernel.WeekID) (*domain.Timesheet, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	return r.Repository.Get(ctx, user, w)
}

func (r *failingRepo) Save(ctx context.Context, ts *domain.Timesheet) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	return r.Repository.Save(ctx, ts)
}

func (r *failingRepo) Delete(ctx context.Context, id string) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	return r.Repository.Delete(ctx, id)
}

func activity(day kernel.WorkDay, text string) kernel.WorkActivity {
	return kernel.WorkActivity{Day: day, ChannelName: "dev", Text: text, IssueKeys: kernel.ExtractIssueKeys(text)}
}

func newPropose(repo domain.Repository, src application.ActivitySource, policies ...alloc.Policy) *application.ProposeHandler {
	return &application.ProposeHandler{
		Repo: repo, Source: src, Issues: knownIssues,
		Policies: policies, Clock: fixedClock{}, UserAccountID: "acc-1",
		DailyCap: kernel.MustDuration(480), PerKey: kernel.MustDuration(45),
	}
}

func newSubmit(repo domain.Repository, gw application.WorklogGateway) *application.SubmitHandler {
	return &application.SubmitHandler{Repo: repo, Gateway: gw, Clock: fixedClock{}, UserAccountID: "acc-1"}
}

func newTransition(repo domain.Repository) *application.TransitionHandler {
	return &application.TransitionHandler{Repo: repo, Clock: fixedClock{}, UserAccountID: "acc-1"}
}

// proposeWeek stores a proposal awaiting approval: GCM-1 and GCM-2 on Monday
// and GCM-1 on Tuesday.
func proposeWeek(t *testing.T, repo domain.Repository) *domain.Timesheet {
	t.Helper()
	h := newPropose(repo, fakeSource{acts: []kernel.WorkActivity{
		activity(mon, "GCM-1 GCM-2"), activity(tue, "GCM-1"),
	}}, alloc.RulePolicy{})
	res, err := h.Handle(t.Context(), application.ProposeCommand{Week: week})
	if err != nil {
		t.Fatalf("propose %v: %v", week, err)
	}
	return res.Timesheet
}

// approveWeek stores the proposeWeek timesheet, approved.
func approveWeek(t *testing.T, repo domain.Repository) {
	t.Helper()
	proposeWeek(t, repo)
	if err := newTransition(repo).Approve(t.Context(), week); err != nil {
		t.Fatalf("approve %v: %v", week, err)
	}
}

// submitWeek stores the proposeWeek timesheet, fully submitted.
func submitWeek(t *testing.T, repo domain.Repository) {
	t.Helper()
	approveWeek(t, repo)
	if _, err := newSubmit(repo, &fakeGateway{}).Handle(t.Context(), week); err != nil {
		t.Fatalf("submit %v: %v", week, err)
	}
}

func eventNames(evs []kernel.DomainEvent) []string {
	names := make([]string, len(evs))
	for i, e := range evs {
		names[i] = e.EventName()
	}
	return names
}

// AC-003.1
func TestProposeWithRulePolicy(t *testing.T) {
	t.Parallel()
	repo := memory.NewRepository()
	h := newPropose(repo, fakeSource{acts: []kernel.WorkActivity{
		activity(mon, "GCM-1 and GCM-2"),
		activity(tue, "GCM-1 plus UNKNOWN-9"),
	}}, alloc.RulePolicy{})

	res, err := h.Handle(t.Context(), application.ProposeCommand{Week: week})
	if err != nil {
		t.Fatalf("Handle(%v) error = %v, want nil", week, err)
	}
	ts := res.Timesheet
	if ts.Status() != domain.StatusAwaitingApproval || ts.Policy() != "rule" {
		t.Errorf("Handle(%v) timesheet status, policy = %s, %s, want %s, rule", week, ts.Status(), ts.Policy(), domain.StatusAwaitingApproval)
	}
	var got []string
	for _, e := range ts.Entries() {
		got = append(got, fmt.Sprintf("%s %s %d", e.Day, e.IssueKey, e.IssueID))
	}
	want := []string{"2026-10-05 GCM-1 101", "2026-10-05 GCM-2 102", "2026-10-06 GCM-1 101"} // UNKNOWN-9 dropped
	if !slices.Equal(got, want) {
		t.Errorf("Handle(%v) entries = %v, want %v", week, got, want)
	}
	if len(res.Failures) != 0 {
		t.Errorf("Handle(%v) failures = %+v, want none", week, res.Failures)
	}
	if _, err := repo.Get(t.Context(), "acc-1", week); err != nil {
		t.Errorf("repo.Get(acc-1, %v) error = %v, want the proposal saved", week, err)
	}
	if got, want := eventNames(repo.Events), []string{domain.EventProposed}; !slices.Equal(got, want) {
		t.Errorf("saved events = %v, want %v", got, want)
	}
}

// AC-003.2
func TestProposeAlreadyProposedAndForce(t *testing.T) {
	t.Parallel()
	repo := memory.NewRepository()
	h := newPropose(repo, fakeSource{acts: []kernel.WorkActivity{activity(mon, "GCM-1")}}, alloc.RulePolicy{})
	first, err := h.Handle(t.Context(), application.ProposeCommand{Week: week})
	if err != nil {
		t.Fatalf("first Handle(%v) error = %v, want nil", week, err)
	}
	if _, err := h.Handle(t.Context(), application.ProposeCommand{Week: week}); !errors.Is(err, application.ErrAlreadyProposed) {
		t.Errorf("second Handle(%v) error = %v, want ErrAlreadyProposed", week, err)
	}
	second, err := h.Handle(t.Context(), application.ProposeCommand{Week: week, Force: true})
	if err != nil {
		t.Fatalf("Handle(%v, force) error = %v, want nil", week, err)
	}
	if second.Timesheet.ID() == first.Timesheet.ID() {
		t.Errorf("Handle(%v, force) kept id %s, want a new timesheet", week, first.Timesheet.ID())
	}
	stored, err := repo.Get(t.Context(), "acc-1", week)
	if err != nil {
		t.Fatalf("repo.Get(acc-1, %v) error = %v, want nil", week, err)
	}
	if stored.ID() != second.Timesheet.ID() {
		t.Errorf("stored timesheet = %s, want the forced replacement %s", stored.ID(), second.Timesheet.ID())
	}
}

func TestProposeDryRunDoesNotTouchRepository(t *testing.T) {
	t.Parallel()
	// Every repository call fails, so any access during a dry run shows up as an error.
	repo := &failingRepo{Repository: memory.NewRepository(), getErr: errDB, saveErr: errDB, deleteErr: errDB}
	h := newPropose(repo, fakeSource{acts: []kernel.WorkActivity{activity(mon, "GCM-1")}}, alloc.RulePolicy{})

	res, err := h.Handle(t.Context(), application.ProposeCommand{Week: week, DryRun: true})
	if err != nil {
		t.Fatalf("Handle(%v, dry run) error = %v, want nil", week, err)
	}
	if res.Timesheet == nil || len(res.Timesheet.Entries()) != 1 {
		t.Errorf("Handle(%v, dry run) timesheet = %+v, want one built entry", week, res.Timesheet)
	}
	if _, err := repo.Repository.Get(t.Context(), "acc-1", week); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("repo.Get(acc-1, %v) after dry run error = %v, want ErrNotFound", week, err)
	}
}

// AC-003.3
func TestProposeFallsBackWhenProposalInvalid(t *testing.T) {
	t.Parallel()
	bad := &fakePolicy{name: "agent", prop: alloc.Proposal{Policy: "agent", Entries: []alloc.ProposedEntry{
		{IssueKey: "GCM-1", Day: mon, Duration: kernel.MustDuration(300)},
		{IssueKey: "GCM-2", Day: mon, Duration: kernel.MustDuration(300)}, // 600m > cap
	}}}
	h := newPropose(memory.NewRepository(), fakeSource{acts: []kernel.WorkActivity{activity(mon, "GCM-1")}}, bad, alloc.RulePolicy{})

	res, err := h.Handle(t.Context(), application.ProposeCommand{Week: week})
	if err != nil {
		t.Fatalf("Handle(%v) error = %v, want nil", week, err)
	}
	if got := res.Timesheet.Policy(); got != "rule" {
		t.Errorf("Handle(%v) policy = %s, want fallback to rule", week, got)
	}
	if len(res.Failures) != 1 || res.Failures[0].Policy != "agent" || !errors.Is(res.Failures[0].Err, domain.ErrDayOverCap) {
		t.Errorf("Handle(%v) failures = %+v, want one agent failure with ErrDayOverCap", week, res.Failures)
	}
}

func TestProposePassesFeedbackToNextPolicy(t *testing.T) {
	t.Parallel()
	bad := &fakePolicy{name: "agent", prop: alloc.Proposal{Entries: []alloc.ProposedEntry{
		{IssueKey: "GCM-1", Day: week.Prev().Days()[0], Duration: kernel.MustDuration(45)}, // outside the week
	}}}
	next := &fakePolicy{name: "agent-retry", prop: alloc.Proposal{Entries: []alloc.ProposedEntry{
		{IssueKey: "GCM-1", Day: mon, Duration: kernel.MustDuration(45)},
	}}}
	h := newPropose(memory.NewRepository(), fakeSource{acts: []kernel.WorkActivity{activity(mon, "GCM-1")}}, bad, next)

	res, err := h.Handle(t.Context(), application.ProposeCommand{Week: week})
	if err != nil {
		t.Fatalf("Handle(%v) error = %v, want nil", week, err)
	}
	if got := res.Timesheet.Policy(); got != "agent-retry" {
		t.Errorf("Handle(%v) policy = %s, want agent-retry", week, got)
	}
	if len(bad.seen) != 1 || len(bad.seen[0].Feedback) != 0 {
		t.Errorf("first policy inputs = %+v, want one input with no feedback", bad.seen)
	}
	if len(next.seen) != 1 || len(next.seen[0].Feedback) != 1 {
		t.Errorf("next policy inputs = %+v, want one input carrying the first policy's error", next.seen)
	}
}

func TestProposeErrorPaths(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		// setup returns the handler under test and the repository to inspect afterwards.
		setup func(t *testing.T) (*application.ProposeHandler, *memory.Repository)
		cmd   application.ProposeCommand
		want  error
	}{
		{
			name: "already submitted, even with force",
			setup: func(t *testing.T) (*application.ProposeHandler, *memory.Repository) {
				repo := memory.NewRepository()
				submitWeek(t, repo)
				return newPropose(repo, fakeSource{}, alloc.RulePolicy{}), repo
			},
			cmd:  application.ProposeCommand{Week: week, Force: true},
			want: application.ErrAlreadySubmitted,
		},
		{
			name: "repository get fails",
			setup: func(*testing.T) (*application.ProposeHandler, *memory.Repository) {
				repo := &failingRepo{Repository: memory.NewRepository(), getErr: errDB}
				return newPropose(repo, fakeSource{}, alloc.RulePolicy{}), repo.Repository
			},
			cmd:  application.ProposeCommand{Week: week},
			want: errDB,
		},
		{
			name: "collect fails",
			setup: func(*testing.T) (*application.ProposeHandler, *memory.Repository) {
				repo := memory.NewRepository()
				return newPropose(repo, fakeSource{err: errSlack}, alloc.RulePolicy{}), repo
			},
			cmd:  application.ProposeCommand{Week: week},
			want: errSlack,
		},
		{
			name: "jira error other than not found aborts",
			setup: func(*testing.T) (*application.ProposeHandler, *memory.Repository) {
				repo := memory.NewRepository()
				h := newPropose(repo, fakeSource{acts: []kernel.WorkActivity{activity(mon, "GCM-1")}}, alloc.RulePolicy{})
				h.Issues = fakeIssues{ids: knownIssues.ids, errFor: map[kernel.IssueKey]error{"GCM-1": errJira}}
				return h, repo
			},
			cmd:  application.ProposeCommand{Week: week},
			want: errJira,
		},
		{
			name: "every policy fails",
			setup: func(*testing.T) (*application.ProposeHandler, *memory.Repository) {
				repo := memory.NewRepository()
				broken := &fakePolicy{name: "agent", err: errors.New("anthropic down")}
				return newPropose(repo, fakeSource{acts: []kernel.WorkActivity{activity(mon, "GCM-1")}}, broken), repo
			},
			cmd:  application.ProposeCommand{Week: week},
			want: application.ErrNoValidProposal,
		},
		{
			name: "save fails",
			setup: func(*testing.T) (*application.ProposeHandler, *memory.Repository) {
				repo := &failingRepo{Repository: memory.NewRepository(), saveErr: errDB}
				return newPropose(repo, fakeSource{acts: []kernel.WorkActivity{activity(mon, "GCM-1")}}, alloc.RulePolicy{}), repo.Repository
			},
			cmd:  application.ProposeCommand{Week: week},
			want: errDB,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h, repo := tt.setup(t)
			before := len(repo.Events)
			if _, err := h.Handle(t.Context(), tt.cmd); !errors.Is(err, tt.want) {
				t.Errorf("Handle(%+v) error = %v, want %v", tt.cmd, err, tt.want)
			}
			if got := repo.Events[before:]; len(got) != 0 {
				t.Errorf("Handle(%+v) saved events %v, want none on error", tt.cmd, eventNames(got))
			}
		})
	}
}

func TestProposeForceKeepsOldTimesheetWhenDeleteFails(t *testing.T) {
	t.Parallel()
	mem := memory.NewRepository()
	old := proposeWeek(t, mem)
	repo := &failingRepo{Repository: mem, deleteErr: errDB}
	h := newPropose(repo, fakeSource{acts: []kernel.WorkActivity{activity(mon, "GCM-1")}}, alloc.RulePolicy{})

	if _, err := h.Handle(t.Context(), application.ProposeCommand{Week: week, Force: true}); !errors.Is(err, errDB) {
		t.Errorf("Handle(%v, force) error = %v, want %v", week, err, errDB)
	}
	stored, err := mem.Get(t.Context(), "acc-1", week)
	if err != nil {
		t.Fatalf("repo.Get(acc-1, %v) error = %v, want the old timesheet kept", week, err)
	}
	if stored.ID() != old.ID() {
		t.Errorf("stored timesheet = %s, want the old one %s", stored.ID(), old.ID())
	}
}

// AC-005.1
func TestSubmitStaggersStartTimes(t *testing.T) {
	t.Parallel()
	repo := memory.NewRepository()
	approveWeek(t, repo)
	gw := &fakeGateway{}

	res, err := newSubmit(repo, gw).Handle(t.Context(), week)
	if err != nil {
		t.Fatalf("Handle(%v) error = %v, want nil", week, err)
	}
	if want := (application.SubmitResult{Submitted: 3, Status: domain.StatusSubmitted}); res != want {
		t.Errorf("Handle(%v) = %+v, want %+v", week, res, want)
	}
	want := []application.Worklog{
		{IssueID: 101, Day: mon, StartTime: "09:00:00", Seconds: 2700, Description: "Work on GCM-1 (Slack: #dev)", AuthorAccountID: "acc-1"},
		{IssueID: 102, Day: mon, StartTime: "09:45:00", Seconds: 2700, Description: "Work on GCM-2 (Slack: #dev)", AuthorAccountID: "acc-1"},
		{IssueID: 101, Day: tue, StartTime: "09:00:00", Seconds: 2700, Description: "Work on GCM-1 (Slack: #dev)", AuthorAccountID: "acc-1"},
	}
	if !slices.Equal(gw.created, want) {
		t.Errorf("Tempo worklogs = %+v, want %+v", gw.created, want)
	}
}

// AC-005.2
func TestSubmitPartialFailureThenRetry(t *testing.T) {
	t.Parallel()
	repo := memory.NewRepository()
	approveWeek(t, repo)
	gw := &fakeGateway{failFor: map[int64]bool{102: true}}
	h := newSubmit(repo, gw)

	res, err := h.Handle(t.Context(), week)
	if err != nil {
		t.Fatalf("first Handle(%v) error = %v, want nil", week, err)
	}
	if want := (application.SubmitResult{Submitted: 2, Failed: 1, Status: domain.StatusApproved}); res != want {
		t.Fatalf("first Handle(%v) = %+v, want %+v", week, res, want)
	}

	gw.failFor = nil
	res, err = h.Handle(t.Context(), week)
	if err != nil {
		t.Fatalf("retry Handle(%v) error = %v, want nil", week, err)
	}
	if want := (application.SubmitResult{Submitted: 1, Status: domain.StatusSubmitted}); res != want {
		t.Errorf("retry Handle(%v) = %+v, want %+v", week, res, want)
	}
	if len(gw.created) != 3 {
		t.Fatalf("Tempo got %d worklogs, want 3 (no duplicates)", len(gw.created))
	}
	if got := gw.created[2]; got.IssueID != 102 || got.StartTime != "09:45:00" {
		t.Errorf("retried worklog = %+v, want issue 102 kept in its 09:45:00 slot", got)
	}
}

func TestSubmitRequiresApprovedTimesheet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func(t *testing.T, repo *memory.Repository)
		want  error
	}{
		{name: "no timesheet", setup: func(*testing.T, *memory.Repository) {}, want: domain.ErrNotFound},
		{name: "awaiting approval", setup: func(t *testing.T, r *memory.Repository) { proposeWeek(t, r) }, want: domain.ErrInvalidTransition},
		{name: "already submitted", setup: func(t *testing.T, r *memory.Repository) { submitWeek(t, r) }, want: domain.ErrInvalidTransition},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := memory.NewRepository()
			tt.setup(t, repo)
			gw := &fakeGateway{}
			if _, err := newSubmit(repo, gw).Handle(t.Context(), week); !errors.Is(err, tt.want) {
				t.Errorf("Handle(%v) error = %v, want %v", week, err, tt.want)
			}
			if len(gw.created) != 0 {
				t.Errorf("Tempo got %d worklogs, want none", len(gw.created))
			}
		})
	}
}

func TestSubmitStopsWhenSaveFailsAfterTempoWrite(t *testing.T) {
	t.Parallel()
	mem := memory.NewRepository()
	approveWeek(t, mem)
	repo := &failingRepo{Repository: mem, saveErr: errDB}
	gw := &fakeGateway{}

	res, err := newSubmit(repo, gw).Handle(t.Context(), week)
	if !errors.Is(err, errDB) {
		t.Errorf("Handle(%v) error = %v, want %v", week, err, errDB)
	}
	if res.Submitted != 1 {
		t.Errorf("Handle(%v) submitted = %d, want 1", week, res.Submitted)
	}
	// The run must stop at the first unsaved write, or a retry would re-send more entries.
	if len(gw.created) != 1 {
		t.Errorf("Tempo got %d worklogs, want 1 (stop after the save fails)", len(gw.created))
	}
}

// AC-004.2
func TestRejectRecordsReason(t *testing.T) {
	t.Parallel()
	repo := memory.NewRepository()
	proposeWeek(t, repo)
	if err := newTransition(repo).Reject(t.Context(), week, "GCM-1 was 2h"); err != nil {
		t.Fatalf("Reject(%v) error = %v, want nil", week, err)
	}
	last, ok := repo.Events[len(repo.Events)-1].(domain.Event)
	if !ok {
		t.Fatalf("last event is %T, want domain.Event", repo.Events[len(repo.Events)-1])
	}
	if last.Name != domain.EventRejected || last.Reason != "GCM-1 was 2h" {
		t.Errorf("last event = %+v, want %s with reason %q", last, domain.EventRejected, "GCM-1 was 2h")
	}
	stored, err := repo.Get(t.Context(), "acc-1", week)
	if err != nil {
		t.Fatalf("repo.Get(acc-1, %v) error = %v, want nil", week, err)
	}
	if stored.Status() != domain.StatusRejected {
		t.Errorf("stored status = %s, want %s", stored.Status(), domain.StatusRejected)
	}
}

func TestTransitionErrorPaths(t *testing.T) {
	t.Parallel()
	approve := func(ctx context.Context, h *application.TransitionHandler) error { return h.Approve(ctx, week) }
	reject := func(ctx context.Context, h *application.TransitionHandler) error { return h.Reject(ctx, week, "no") }
	tests := []struct {
		name  string
		setup func(t *testing.T) domain.Repository
		run   func(context.Context, *application.TransitionHandler) error
		want  error
	}{
		{
			name:  "approve without a timesheet",
			setup: func(*testing.T) domain.Repository { return memory.NewRepository() },
			run:   approve,
			want:  domain.ErrNotFound,
		},
		{
			name:  "reject without a timesheet",
			setup: func(*testing.T) domain.Repository { return memory.NewRepository() },
			run:   reject,
			want:  domain.ErrNotFound,
		},
		{
			name: "approve twice",
			setup: func(t *testing.T) domain.Repository {
				repo := memory.NewRepository()
				approveWeek(t, repo)
				return repo
			},
			run:  approve,
			want: domain.ErrInvalidTransition,
		},
		{
			name: "save fails",
			setup: func(t *testing.T) domain.Repository {
				mem := memory.NewRepository()
				proposeWeek(t, mem)
				return &failingRepo{Repository: mem, saveErr: errDB}
			},
			run:  approve,
			want: errDB,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newTransition(tt.setup(t))
			if err := tt.run(t.Context(), h); !errors.Is(err, tt.want) {
				t.Errorf("%s: error = %v, want %v", tt.name, err, tt.want)
			}
		})
	}
}
