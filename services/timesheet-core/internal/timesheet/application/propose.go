package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	alloc "timesheet-bot/internal/allocation/domain"
	"timesheet-bot/internal/shared/kernel"
	"timesheet-bot/internal/timesheet/domain"
)

// Errors returned by [ProposeHandler.Handle].
var (
	// ErrAlreadyProposed means the week already has a timesheet and the
	// command did not set Force.
	ErrAlreadyProposed = errors.New("timesheet already exists for this week (use force to replace)")
	// ErrAlreadySubmitted means the week's timesheet is already submitted, so
	// it cannot be replaced.
	ErrAlreadySubmitted = errors.New("timesheet for this week is already submitted")
	// ErrNoValidProposal means every policy failed or produced a proposal
	// that the aggregate rejected.
	ErrNoValidProposal = errors.New("no allocation policy produced a valid timesheet")
)

// ProposeCommand asks [ProposeHandler] to build the timesheet for one week.
type ProposeCommand struct {
	Week   kernel.WeekID
	Force  bool // replace an existing, not yet submitted timesheet
	DryRun bool // build and validate only; no repository access
}

// PolicyFailure records why a policy's output was not used.
type PolicyFailure struct {
	Policy string
	Err    error
}

// ProposeResult is the outcome of [ProposeHandler.Handle]: the accepted
// timesheet and proposal, and the policies that failed before it.
type ProposeResult struct {
	Timesheet  *domain.Timesheet
	Proposal   alloc.Proposal
	Failures   []PolicyFailure
	Placements map[string]MavenlinkPlacement // by entry ID; nil unless Mavenlink is a submit target
}

// ProposeHandler builds a week's timesheet from activity. Policies are tried in
// order; the first proposal the aggregate accepts wins. Errors from a rejected
// proposal are passed to the next policy as feedback.
type ProposeHandler struct {
	Repo            domain.Repository
	Source          ActivitySource
	Reports         ReportLineSource  // report.parser: llm only; fills alloc.Input.ReportLines
	Issues          IssueDirectory    // nil in report mode, which has no Jira issues
	Placements      PlacementResolver // set when Mavenlink is a submit target: plan shows where time goes
	Policies        []alloc.Policy
	Clock           kernel.Clock
	UserAccountID   string
	DailyCap        kernel.Duration
	PerKey          kernel.Duration
	ChannelDefaults map[string]kernel.IssueKey
}

// Handle builds the timesheet for cmd.Week and, unless cmd.DryRun is set,
// saves it. An existing timesheet for the week is replaced only when
// cmd.Force is set and it is not yet submitted. If no policy produces a valid
// timesheet, Handle returns the failures in the result and an error wrapping
// [ErrNoValidProposal].
func (h *ProposeHandler) Handle(ctx context.Context, cmd ProposeCommand) (ProposeResult, error) {
	var replace *domain.Timesheet
	if !cmd.DryRun {
		existing, err := h.Repo.Get(ctx, h.UserAccountID, cmd.Week)
		switch {
		case err == nil && existing.Status() == domain.StatusSubmitted:
			return ProposeResult{}, ErrAlreadySubmitted
		case err == nil && !cmd.Force:
			return ProposeResult{}, fmt.Errorf("%w: %s is %s", ErrAlreadyProposed, cmd.Week, existing.Status())
		case err == nil:
			replace = existing
		case !errors.Is(err, domain.ErrNotFound):
			return ProposeResult{}, err
		}
	}

	acts, err := h.Source.Collect(ctx, cmd.Week)
	if err != nil {
		return ProposeResult{}, fmt.Errorf("collect activity: %w", err)
	}
	issues := newIssueCache(h.Issues)
	if h.Issues != nil {
		if acts, err = issues.dropUnknownKeys(ctx, acts); err != nil {
			return ProposeResult{}, err
		}
	}

	in := alloc.Input{
		Week: cmd.Week, Activities: acts, ChannelDefaults: h.ChannelDefaults,
		DailyCap: h.DailyCap, PerKey: h.PerKey,
	}
	if h.Reports != nil {
		if in.ReportLines, err = h.Reports.ReportLines(ctx, cmd.Week); err != nil {
			return ProposeResult{}, fmt.Errorf("collect report lines: %w", err)
		}
	}
	var res ProposeResult
	for _, p := range h.Policies {
		prop, err := p.Allocate(ctx, in)
		if err == nil {
			var ts *domain.Timesheet
			if ts, err = h.build(ctx, cmd.Week, p.Name(), prop, issues); err == nil {
				res.Timesheet, res.Proposal = ts, prop
				break
			}
			in.Feedback = append(in.Feedback, err.Error())
		}
		res.Failures = append(res.Failures, PolicyFailure{Policy: p.Name(), Err: err})
	}
	if res.Timesheet == nil {
		return res, fmt.Errorf("%w: %v", ErrNoValidProposal, res.Failures)
	}
	if h.Placements != nil {
		if err := h.place(&res); err != nil {
			return res, err
		}
	}

	if cmd.DryRun {
		return res, nil
	}
	if replace != nil {
		if err := h.Repo.Delete(ctx, replace.ID()); err != nil {
			return res, err
		}
	}
	return res, h.Repo.Save(ctx, res.Timesheet)
}

// place resolves the Mavenlink project of every entry, so the owner sees
// where time will go before submitting. A failure fails the plan; a project
// that fell back to the default adds one warning.
func (h *ProposeHandler) place(res *ProposeResult) error {
	res.Placements = map[string]MavenlinkPlacement{}
	warned := map[string]bool{}
	var errs []error
	for _, e := range res.Timesheet.Entries() {
		p, err := h.Placements.Resolve(e.IssueKey, e.Project)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		res.Placements[e.ID] = p
		subject := e.Project
		if e.IssueKey != "" {
			subject = e.IssueKey.String()
		}
		if p.Fallback && !warned[subject] {
			warned[subject] = true
			res.Proposal.Warnings = append(res.Proposal.Warnings, fmt.Sprintf(
				"%q has no mavenlink.mappings rule; its time goes to default_workspace_id %d", subject, p.WorkspaceID))
		}
	}
	return errors.Join(errs...)
}

// build runs a proposal through the aggregate; every invariant violation is reported.
func (h *ProposeHandler) build(ctx context.Context, week kernel.WeekID, policy string, p alloc.Proposal, issues *issueCache) (*domain.Timesheet, error) {
	ts := domain.New(uuid.NewString(), h.UserAccountID, week, h.DailyCap, policy, h.Clock.Now())
	var errs []error
	for _, e := range p.Entries {
		if e.Project != "" {
			if err := ts.AddReportEntry(e.Project, e.Day, e.Duration, e.Description); err != nil {
				errs = append(errs, err)
			}
			continue
		}
		issue, err := issues.resolve(ctx, e.IssueKey)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := ts.AddEntry(e.IssueKey, issue.ID, e.Day, e.Duration, e.Description); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	if err := ts.Propose(h.Clock.Now()); err != nil {
		return nil, err
	}
	return ts, nil
}

// issueCache memoizes lookups for one run, including misses.
type issueCache struct {
	dir   IssueDirectory
	found map[kernel.IssueKey]Issue
	miss  map[kernel.IssueKey]error
}

func newIssueCache(dir IssueDirectory) *issueCache {
	return &issueCache{dir: dir, found: map[kernel.IssueKey]Issue{}, miss: map[kernel.IssueKey]error{}}
}

func (c *issueCache) resolve(ctx context.Context, k kernel.IssueKey) (Issue, error) {
	if c.dir == nil {
		return Issue{}, fmt.Errorf("%w: %s (no Jira in report mode)", ErrIssueNotFound, k)
	}
	if i, ok := c.found[k]; ok {
		return i, nil
	}
	if err, ok := c.miss[k]; ok {
		return Issue{}, err
	}
	i, err := c.dir.Resolve(ctx, k)
	if errors.Is(err, ErrIssueNotFound) {
		c.miss[k] = err
		return Issue{}, err
	}
	if err != nil {
		return Issue{}, err
	}
	c.found[k] = i
	return i, nil
}

// dropUnknownKeys removes issue keys that do not exist in Jira (e.g. "UTF-8").
// Lookup failures other than not-found abort the run.
func (c *issueCache) dropUnknownKeys(ctx context.Context, acts []kernel.WorkActivity) ([]kernel.WorkActivity, error) {
	out := make([]kernel.WorkActivity, len(acts))
	for i, a := range acts {
		var keep []kernel.IssueKey
		for _, k := range a.IssueKeys {
			_, err := c.resolve(ctx, k)
			switch {
			case err == nil:
				keep = append(keep, k)
			case !errors.Is(err, ErrIssueNotFound):
				return nil, fmt.Errorf("resolve %s: %w", k, err)
			}
		}
		a.IssueKeys = keep
		out[i] = a
	}
	return out, nil
}
