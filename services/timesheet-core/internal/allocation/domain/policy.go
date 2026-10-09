// Package domain is the Allocation context's domain: it turns a week of work
// activity into a proposed set of worklog entries. A [Policy] produces a
// [Proposal]; the proposal is not validated here, because the Timesheet
// aggregate replays and accepts or rejects it. [RulePolicy] is the local,
// deterministic policy that needs no network calls.
package domain

import (
	"context"

	"timesheet-bot/internal/shared/kernel"
)

// Input is everything a policy may use to allocate a week.
type Input struct {
	Week            kernel.WeekID
	Activities      []kernel.WorkActivity
	ChannelDefaults map[string]kernel.IssueKey // channel name -> issue for keyless activity
	DailyCap        kernel.Duration
	PerKey          kernel.Duration // time credited per issue key per day
	Feedback        []string        // validation errors or user corrections from a previous attempt
}

// Policy turns a week of activity into a proposal.
type Policy interface {
	Name() string
	Allocate(ctx context.Context, in Input) (Proposal, error)
}
