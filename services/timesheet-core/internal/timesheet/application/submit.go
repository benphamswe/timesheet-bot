package application

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"timesheet-bot/internal/shared/kernel"
	"timesheet-bot/internal/timesheet/domain"
)

// DayStartMinute is when the first worklog of a day starts (09:00).
const DayStartMinute = 9 * 60

// SubmitResult counts the entries sent and failed in one submit run, over
// every target, the Mavenlink timesheet submissions sent for approval, and
// gives the timesheet's status afterwards.
type SubmitResult struct {
	Submitted   int
	Failed      int
	Submissions int
	Status      domain.Status
}

// MavenlinkSubmit is how a [SubmitHandler] writes to Mavenlink.
type MavenlinkSubmit struct {
	Gateway    TimeEntryGateway
	Placements PlacementResolver
	// SubmitForApproval sends one timesheet submission per Mavenlink project
	// once every entry is created.
	SubmitForApproval bool
}

// SubmitHandler sends approved entries to every configured target: Tempo when
// Gateway is set, Mavenlink when Mavenlink is set. It saves after every entry
// so a crash or retry never sends the same entry twice, and an entry that
// fails in one target is never marked as sent to another.
type SubmitHandler struct {
	Repo          domain.Repository
	Gateway       WorklogGateway   // Tempo; nil skips Tempo
	Mavenlink     *MavenlinkSubmit // nil skips Mavenlink
	Clock         kernel.Clock
	UserAccountID string
}

// errNoTargets is returned when neither Tempo nor Mavenlink is configured.
var errNoTargets = errors.New("no submit target configured")

// Handle sends every not yet submitted entry of an approved timesheet for week
// to each target, saving the timesheet after each entry. A failed entry is
// recorded on the timesheet and counted in the result; it does not stop the
// run. A rejected credential ([ErrUnauthorized]) stops the run. The timesheet
// becomes submitted once every entry is in every target and, for Mavenlink,
// its submissions were sent.
func (h *SubmitHandler) Handle(ctx context.Context, week kernel.WeekID) (SubmitResult, error) {
	ts, err := h.Repo.Get(ctx, h.UserAccountID, week)
	if err != nil {
		return SubmitResult{}, err
	}
	if ts.Status() != domain.StatusApproved {
		return SubmitResult{Status: ts.Status()}, fmt.Errorf("%w: %s is %s, want approved",
			domain.ErrInvalidTransition, week, ts.Status())
	}

	var targets []domain.Target
	if h.Gateway != nil {
		targets = append(targets, domain.TargetTempo)
	}
	if h.Mavenlink != nil {
		targets = append(targets, domain.TargetMavenlink)
	}
	if len(targets) == 0 {
		return SubmitResult{Status: ts.Status()}, errNoTargets
	}

	var res SubmitResult
	if h.Gateway != nil {
		if err := h.submitTempo(ctx, ts, &res); err != nil {
			res.Status = ts.Status()
			return res, err
		}
	}
	if h.Mavenlink != nil {
		if err := h.submitMavenlink(ctx, ts, &res); err != nil {
			res.Status = ts.Status()
			return res, err
		}
	}

	switch err := ts.CompleteSubmission(targets, h.Clock.Now()); {
	case errors.Is(err, domain.ErrEntriesPending):
	case err != nil:
		return res, err
	default:
		if err := h.Repo.Save(ctx, ts); err != nil {
			return res, fmt.Errorf("save after submit: %w", err)
		}
	}
	res.Status = ts.Status()
	return res, nil
}

func (h *SubmitHandler) submitTempo(ctx context.Context, ts *domain.Timesheet, res *SubmitResult) error {
	offset := map[kernel.WorkDay]int{}
	// Walk all entries, not only pending ones, so a retried entry keeps its time slot.
	for _, e := range ts.Entries() {
		start := DayStartMinute + offset[e.Day]
		offset[e.Day] += e.Duration.Minutes()
		if e.SubmittedTo(domain.TargetTempo) {
			continue
		}
		id, err := h.Gateway.Create(ctx, Worklog{
			IssueID:         e.IssueID,
			Day:             e.Day,
			StartTime:       fmt.Sprintf("%02d:%02d:00", start/60, start%60),
			Seconds:         e.Duration.Seconds(),
			Description:     e.Description,
			AuthorAccountID: ts.UserAccountID(),
		})
		if err := h.record(ctx, ts, e.ID, domain.TargetTempo, id, err, res); err != nil {
			return err
		}
	}
	return nil
}

func (h *SubmitHandler) submitMavenlink(ctx context.Context, ts *domain.Timesheet, res *SubmitResult) error {
	m := h.Mavenlink
	// Entries created by an earlier run that stopped before saving are
	// adopted, never created twice.
	remote, err := m.Gateway.Created(ctx, ts.Week())
	if err != nil {
		return fmt.Errorf("mavenlink: %w", err)
	}
	for _, e := range ts.Entries() {
		if e.SubmittedTo(domain.TargetMavenlink) {
			continue
		}
		r, ok := remote[e.ID]
		if !ok {
			r, err = h.createMavenlink(ctx, e)
			if err == nil {
				remote[e.ID] = r
			}
		}
		if err := h.record(ctx, ts, e.ID, domain.TargetMavenlink, r.ID, err, res); err != nil {
			return err
		}
	}
	if !m.SubmitForApproval || len(ts.PendingEntries(domain.TargetMavenlink)) > 0 {
		return nil
	}

	byWorkspace := map[int64][]string{}
	for _, e := range ts.Entries() {
		if r, ok := remote[e.ID]; ok && !r.InSubmission {
			byWorkspace[r.WorkspaceID] = append(byWorkspace[r.WorkspaceID], r.ID)
		}
	}
	title := "Week " + ts.Week().String()
	for _, ws := range slices.Sorted(maps.Keys(byWorkspace)) {
		if err := m.Gateway.SubmitForApproval(ctx, ws, title, byWorkspace[ws]); err != nil {
			return fmt.Errorf("mavenlink: submit project %d for approval: %w", ws, err)
		}
		res.Submissions++
	}
	return nil
}

func (h *SubmitHandler) createMavenlink(ctx context.Context, e domain.Entry) (RemoteTimeEntry, error) {
	p, err := h.Mavenlink.Placements.Resolve(e.IssueKey)
	if err != nil {
		return RemoteTimeEntry{}, err
	}
	id, err := h.Mavenlink.Gateway.Create(ctx, TimeEntry{
		EntryID:     e.ID,
		WorkspaceID: p.WorkspaceID,
		StoryID:     p.StoryID,
		RoleID:      p.RoleID,
		Billable:    p.Billable,
		Day:         e.Day,
		Minutes:     e.Duration.Minutes(),
		Notes:       e.Description,
	})
	return RemoteTimeEntry{ID: id, WorkspaceID: p.WorkspaceID}, err
}

// record marks entry entryID as sent to target with id, or as failed with
// sendErr, and saves. A rejected credential stops the run without touching
// the entry.
func (h *SubmitHandler) record(ctx context.Context, ts *domain.Timesheet, entryID string, target domain.Target, id string, sendErr error, res *SubmitResult) error {
	if errors.Is(sendErr, ErrUnauthorized) {
		return fmt.Errorf("%s: %w", target, sendErr)
	}
	now := h.Clock.Now()
	var err error
	if sendErr != nil {
		res.Failed++
		if target != domain.TargetTempo {
			sendErr = fmt.Errorf("%s: %w", target, sendErr)
		}
		err = ts.MarkEntryFailed(entryID, sendErr, now)
	} else {
		res.Submitted++
		err = ts.MarkEntrySubmitted(entryID, target, id, now)
	}
	if err != nil {
		return err
	}
	if err := h.Repo.Save(ctx, ts); err != nil {
		return fmt.Errorf("save after %s: %w", entryID, err)
	}
	return nil
}
