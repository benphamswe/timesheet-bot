package application_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"timesheet-bot/internal/shared/kernel"
	"timesheet-bot/internal/timesheet/application"
	"timesheet-bot/internal/timesheet/domain"
	"timesheet-bot/internal/timesheet/infrastructure/memory"
)

var errMavenlink = errors.New("mavenlink 500")

// stubPlacements places GCM-1 and report project "Atlas" in project 7 (task
// 70, role 5) and GCM-2 in project 8; anything else, or a key in fail, has no
// placement. It records every report project it is asked about.
type stubPlacements struct {
	fail     map[kernel.IssueKey]bool
	projects *[]string
}

func (s stubPlacements) Resolve(k kernel.IssueKey, project string) (application.MavenlinkPlacement, error) {
	if s.projects != nil && project != "" {
		*s.projects = append(*s.projects, project)
	}
	switch {
	case s.fail[k]:
	case k == "GCM-1", k == "" && strings.EqualFold(project, "atlas"):
		return application.MavenlinkPlacement{WorkspaceID: 7, StoryID: 70, RoleID: 5}, nil
	case k == "GCM-2":
		return application.MavenlinkPlacement{WorkspaceID: 8}, nil
	}
	return application.MavenlinkPlacement{}, fmt.Errorf("%w: %s%s", application.ErrNoMavenlinkPlacement, k, project)
}

type submission struct {
	workspaceID int64
	title       string
	ids         []string
}

// fakeTimeEntries is a stateful Mavenlink: Create stores an entry that
// Created lists afterwards, and SubmitForApproval marks its entries as in a
// submission. The err fields make the matching call fail.
type fakeTimeEntries struct {
	remote      map[string]application.RemoteTimeEntry
	created     []application.TimeEntry
	submissions []submission

	listErr   error
	createErr error
	submitErr map[int64]error
}

func (f *fakeTimeEntries) Created(context.Context, kernel.WeekID) (map[string]application.RemoteTimeEntry, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := map[string]application.RemoteTimeEntry{}
	for k, v := range f.remote {
		out[k] = v
	}
	return out, nil
}

func (f *fakeTimeEntries) Create(_ context.Context, e application.TimeEntry) (string, error) {
	if f.createErr != nil {
		return "", f.createErr
	}
	f.created = append(f.created, e)
	id := fmt.Sprintf("m-%d", len(f.created))
	if f.remote == nil {
		f.remote = map[string]application.RemoteTimeEntry{}
	}
	f.remote[e.EntryID] = application.RemoteTimeEntry{ID: id, WorkspaceID: e.WorkspaceID}
	return id, nil
}

func (f *fakeTimeEntries) SubmitForApproval(_ context.Context, ws int64, title string, ids []string) error {
	if err := f.submitErr[ws]; err != nil {
		return err
	}
	f.submissions = append(f.submissions, submission{workspaceID: ws, title: title, ids: ids})
	for k, v := range f.remote {
		if slices.Contains(ids, v.ID) {
			v.InSubmission = true
			f.remote[k] = v
		}
	}
	return nil
}

func newMavenlinkSubmit(repo domain.Repository, tempoGW application.WorklogGateway, ml *fakeTimeEntries, placements application.PlacementResolver) *application.SubmitHandler {
	h := &application.SubmitHandler{
		Repo: repo, Clock: fixedClock{}, UserAccountID: "acc-1",
		Mavenlink: &application.MavenlinkSubmit{Gateway: ml, Placements: placements, SubmitForApproval: true},
	}
	if tempoGW != nil {
		h.Gateway = tempoGW
	}
	return h
}

// entryIDs returns the approved week's entry IDs in Entries order: Monday
// GCM-1, Monday GCM-2, Tuesday GCM-1.
func entryIDs(t *testing.T, repo domain.Repository) []string {
	t.Helper()
	ts, err := repo.Get(t.Context(), "acc-1", week)
	if err != nil {
		t.Fatalf("repo.Get(acc-1, %v) error = %v", week, err)
	}
	var ids []string
	for _, e := range ts.Entries() {
		ids = append(ids, e.ID)
	}
	return ids
}

func storedEntries(t *testing.T, repo domain.Repository) []domain.Entry {
	t.Helper()
	ts, err := repo.Get(t.Context(), "acc-1", week)
	if err != nil {
		t.Fatalf("repo.Get(acc-1, %v) error = %v", week, err)
	}
	return ts.Entries()
}

func TestSubmitMavenlinkCreatesEntriesAndSubmitsPerProject(t *testing.T) {
	t.Parallel()
	repo := memory.NewRepository()
	approveWeek(t, repo)
	ids := entryIDs(t, repo)
	ml := &fakeTimeEntries{}

	res, err := newMavenlinkSubmit(repo, nil, ml, stubPlacements{}).Handle(t.Context(), week)
	if err != nil {
		t.Fatalf("Handle(%v) error = %v, want nil", week, err)
	}
	if want := (application.SubmitResult{Submitted: 3, Submissions: 2, Status: domain.StatusSubmitted}); res != want {
		t.Errorf("Handle(%v) = %+v, want %+v", week, res, want)
	}
	want := []application.TimeEntry{
		{EntryID: ids[0], WorkspaceID: 7, StoryID: 70, RoleID: 5, Day: mon, Minutes: 45, Notes: "Work on GCM-1 (Slack: #dev)"},
		{EntryID: ids[1], WorkspaceID: 8, Day: mon, Minutes: 45, Notes: "Work on GCM-2 (Slack: #dev)"},
		{EntryID: ids[2], WorkspaceID: 7, StoryID: 70, RoleID: 5, Day: tue, Minutes: 45, Notes: "Work on GCM-1 (Slack: #dev)"},
	}
	if !slices.Equal(ml.created, want) {
		t.Errorf("Mavenlink time entries = %+v, want %+v", ml.created, want)
	}
	wantSubs := []submission{
		{workspaceID: 7, title: "Week 2026-W41", ids: []string{"m-1", "m-3"}},
		{workspaceID: 8, title: "Week 2026-W41", ids: []string{"m-2"}},
	}
	if !slices.EqualFunc(ml.submissions, wantSubs, func(a, b submission) bool {
		return a.workspaceID == b.workspaceID && a.title == b.title && slices.Equal(a.ids, b.ids)
	}) {
		t.Errorf("submissions = %+v, want %+v", ml.submissions, wantSubs)
	}
	if got := storedEntries(t, repo)[1].MavenlinkEntryID; got != "m-2" {
		t.Errorf("stored MavenlinkEntryID of %s = %q, want m-2", ids[1], got)
	}
}

// An earlier run created the entry in Mavenlink but stopped before saving.
func TestSubmitMavenlinkAdoptsEntryFromEarlierRun(t *testing.T) {
	t.Parallel()
	repo := memory.NewRepository()
	approveWeek(t, repo)
	ids := entryIDs(t, repo)
	ml := &fakeTimeEntries{remote: map[string]application.RemoteTimeEntry{
		ids[0]: {ID: "m-old", WorkspaceID: 7},
	}}

	if _, err := newMavenlinkSubmit(repo, nil, ml, stubPlacements{}).Handle(t.Context(), week); err != nil {
		t.Fatalf("Handle(%v) error = %v, want nil", week, err)
	}
	if len(ml.created) != 2 {
		t.Errorf("Mavenlink got %d new time entries, want 2 (the earlier one adopted, not duplicated)", len(ml.created))
	}
	if got := storedEntries(t, repo)[0].MavenlinkEntryID; got != "m-old" {
		t.Errorf("stored MavenlinkEntryID of %s = %q, want m-old", ids[0], got)
	}
	if len(ml.submissions) == 0 || !slices.Contains(ml.submissions[0].ids, "m-old") {
		t.Errorf("submissions = %+v, want project 7's to include m-old", ml.submissions)
	}
}

func TestSubmitMavenlinkSkipsEntriesAlreadyInSubmission(t *testing.T) {
	t.Parallel()
	repo := memory.NewRepository()
	approveWeek(t, repo)
	ids := entryIDs(t, repo)
	ml := &fakeTimeEntries{remote: map[string]application.RemoteTimeEntry{
		ids[0]: {ID: "m-a", WorkspaceID: 7, InSubmission: true},
		ids[1]: {ID: "m-b", WorkspaceID: 8, InSubmission: true},
		ids[2]: {ID: "m-c", WorkspaceID: 7, InSubmission: true},
	}}

	res, err := newMavenlinkSubmit(repo, nil, ml, stubPlacements{}).Handle(t.Context(), week)
	if err != nil {
		t.Fatalf("Handle(%v) error = %v, want nil", week, err)
	}
	if want := (application.SubmitResult{Submitted: 3, Status: domain.StatusSubmitted}); res != want {
		t.Errorf("Handle(%v) = %+v, want %+v", week, res, want)
	}
	if len(ml.created) != 0 || len(ml.submissions) != 0 {
		t.Errorf("Mavenlink got %d entries and %d submissions, want none", len(ml.created), len(ml.submissions))
	}
}

func TestSubmitMavenlinkUnplacedEntryFailsAndHoldsSubmission(t *testing.T) {
	t.Parallel()
	repo := memory.NewRepository()
	approveWeek(t, repo)
	ml := &fakeTimeEntries{}

	res, err := newMavenlinkSubmit(repo, nil, ml, stubPlacements{fail: map[kernel.IssueKey]bool{"GCM-2": true}}).Handle(t.Context(), week)
	if err != nil {
		t.Fatalf("Handle(%v) error = %v, want nil", week, err)
	}
	if want := (application.SubmitResult{Submitted: 2, Failed: 1, Status: domain.StatusApproved}); res != want {
		t.Errorf("Handle(%v) = %+v, want %+v", week, res, want)
	}
	if len(ml.submissions) != 0 {
		t.Errorf("submissions = %+v, want none while an entry is missing", ml.submissions)
	}
	failed := storedEntries(t, repo)[1]
	if failed.SubmitAttempts != 1 || !strings.HasPrefix(failed.LastError, "mavenlink: ") {
		t.Errorf("failed entry = %+v, want SubmitAttempts 1 and LastError starting with %q", failed, "mavenlink: ")
	}
}

// AC: one target failing never marks the entry as sent to the other.
func TestSubmitBothTargetsKeepsFailuresPerTarget(t *testing.T) {
	t.Parallel()
	repo := memory.NewRepository()
	approveWeek(t, repo)
	tempoGW := &fakeGateway{failFor: map[int64]bool{102: true}}
	ml := &fakeTimeEntries{}
	h := newMavenlinkSubmit(repo, tempoGW, ml, stubPlacements{})

	res, err := h.Handle(t.Context(), week)
	if err != nil {
		t.Fatalf("first Handle(%v) error = %v, want nil", week, err)
	}
	if want := (application.SubmitResult{Submitted: 5, Failed: 1, Submissions: 2, Status: domain.StatusApproved}); res != want {
		t.Fatalf("first Handle(%v) = %+v, want %+v", week, res, want)
	}
	gcm2 := storedEntries(t, repo)[1]
	if gcm2.SubmittedTo(domain.TargetTempo) || !gcm2.SubmittedTo(domain.TargetMavenlink) {
		t.Errorf("GCM-2 entry = %+v, want in Mavenlink only", gcm2)
	}

	tempoGW.failFor = nil
	res, err = h.Handle(t.Context(), week)
	if err != nil {
		t.Fatalf("retry Handle(%v) error = %v, want nil", week, err)
	}
	if want := (application.SubmitResult{Submitted: 1, Status: domain.StatusSubmitted}); res != want {
		t.Errorf("retry Handle(%v) = %+v, want %+v", week, res, want)
	}
	if len(ml.created) != 3 || len(ml.submissions) != 2 {
		t.Errorf("Mavenlink got %d entries and %d submissions, want 3 and 2 (no duplicates on retry)", len(ml.created), len(ml.submissions))
	}
}

func TestSubmitStopsWhenCredentialsRejected(t *testing.T) {
	t.Parallel()
	repo := memory.NewRepository()
	approveWeek(t, repo)
	ml := &fakeTimeEntries{createErr: fmt.Errorf("%w: session cookie expired", application.ErrUnauthorized)}

	res, err := newMavenlinkSubmit(repo, nil, ml, stubPlacements{}).Handle(t.Context(), week)
	if !errors.Is(err, application.ErrUnauthorized) {
		t.Fatalf("Handle(%v) error = %v, want ErrUnauthorized", week, err)
	}
	if res.Submitted != 0 || res.Failed != 0 || res.Status != domain.StatusApproved {
		t.Errorf("Handle(%v) = %+v, want nothing counted and status approved", week, res)
	}
	for _, e := range storedEntries(t, repo) {
		if e.SubmittedTo(domain.TargetMavenlink) || e.SubmitAttempts != 0 {
			t.Errorf("entry %s = %+v, want untouched (not submitted, no attempt counted)", e.ID, e)
		}
	}
}

func TestSubmitForApprovalFailureRetriesOnlyThatProject(t *testing.T) {
	t.Parallel()
	repo := memory.NewRepository()
	approveWeek(t, repo)
	ml := &fakeTimeEntries{submitErr: map[int64]error{8: errMavenlink}}
	h := newMavenlinkSubmit(repo, nil, ml, stubPlacements{})

	res, err := h.Handle(t.Context(), week)
	if !errors.Is(err, errMavenlink) {
		t.Fatalf("first Handle(%v) error = %v, want %v", week, err, errMavenlink)
	}
	if res.Status != domain.StatusApproved || res.Submissions != 1 {
		t.Errorf("first Handle(%v) = %+v, want status approved and 1 submission", week, res)
	}

	ml.submitErr = nil
	res, err = h.Handle(t.Context(), week)
	if err != nil {
		t.Fatalf("retry Handle(%v) error = %v, want nil", week, err)
	}
	if want := (application.SubmitResult{Submissions: 1, Status: domain.StatusSubmitted}); res != want {
		t.Errorf("retry Handle(%v) = %+v, want %+v", week, res, want)
	}
	if got := len(ml.submissions); got != 2 || ml.submissions[1].workspaceID != 8 {
		t.Errorf("submissions = %+v, want project 7 then project 8, once each", ml.submissions)
	}
}

func TestSubmitMavenlinkWithoutApproval(t *testing.T) {
	t.Parallel()
	repo := memory.NewRepository()
	approveWeek(t, repo)
	ml := &fakeTimeEntries{}
	h := newMavenlinkSubmit(repo, nil, ml, stubPlacements{})
	h.Mavenlink.SubmitForApproval = false

	res, err := h.Handle(t.Context(), week)
	if err != nil {
		t.Fatalf("Handle(%v) error = %v, want nil", week, err)
	}
	if want := (application.SubmitResult{Submitted: 3, Status: domain.StatusSubmitted}); res != want || len(ml.submissions) != 0 {
		t.Errorf("Handle(%v) = %+v with %d submissions, want %+v and none", week, res, len(ml.submissions), want)
	}
}

func TestSubmitMavenlinkListFailureTouchesNothing(t *testing.T) {
	t.Parallel()
	repo := memory.NewRepository()
	approveWeek(t, repo)
	ml := &fakeTimeEntries{listErr: errMavenlink}

	if _, err := newMavenlinkSubmit(repo, nil, ml, stubPlacements{}).Handle(t.Context(), week); !errors.Is(err, errMavenlink) {
		t.Errorf("Handle(%v) error = %v, want %v", week, err, errMavenlink)
	}
	if len(ml.created) != 0 {
		t.Errorf("Mavenlink got %d time entries, want none when the listing fails", len(ml.created))
	}
}

func TestSubmitWithoutTargetsFails(t *testing.T) {
	t.Parallel()
	repo := memory.NewRepository()
	approveWeek(t, repo)
	h := &application.SubmitHandler{Repo: repo, Clock: fixedClock{}, UserAccountID: "acc-1"}
	if _, err := h.Handle(t.Context(), week); err == nil {
		t.Errorf("Handle(%v) with no target error = nil, want an error", week)
	}
}
