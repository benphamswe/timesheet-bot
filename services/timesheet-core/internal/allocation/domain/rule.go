package domain

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"timesheet-bot/internal/shared/kernel"
)

// RulePolicy credits PerKey (default 45m) for each distinct issue key per day.
// Keyless activity in a channel with a configured default issue counts for that issue.
type RulePolicy struct{}

// Name returns "rule", the identifier recorded in [Proposal.Policy].
func (RulePolicy) Name() string { return "rule" }

// Allocate proposes one entry of in.PerKey per distinct issue key per day,
// for activity inside in.Week only. An activity with no issue key uses the
// default issue for its channel, if in.ChannelDefaults has one, and is
// otherwise ignored. Entries are ordered by day, then by issue key, and each
// day's reasoning is keyed by its YYYY-MM-DD date. A day whose total exceeds
// in.DailyCap is passed to scaleDay. Allocate ignores ctx and in.Feedback
// and never returns an error.
func (RulePolicy) Allocate(_ context.Context, in Input) (Proposal, error) {
	type slot struct {
		day kernel.WorkDay
		key kernel.IssueKey
	}
	channels := map[slot][]string{}
	var order []slot
	for _, a := range in.Activities {
		if !in.Week.Contains(a.Day) {
			continue
		}
		keys := a.IssueKeys
		if len(keys) == 0 {
			if def, ok := in.ChannelDefaults[a.ChannelName]; ok {
				keys = []kernel.IssueKey{def}
			}
		}
		for _, k := range keys {
			s := slot{a.Day, k}
			if _, ok := channels[s]; !ok {
				order = append(order, s)
			}
			channels[s] = appendUnique(channels[s], a.ChannelName)
		}
	}

	byDay := map[kernel.WorkDay][]ProposedEntry{}
	for _, s := range order {
		byDay[s.day] = append(byDay[s.day], ProposedEntry{
			IssueKey:    s.key,
			Day:         s.day,
			Duration:    in.PerKey,
			Description: fmt.Sprintf("Work on %s (Slack: #%s)", s.key, strings.Join(channels[s], ", #")),
		})
	}

	p := Proposal{Policy: "rule", Reasoning: map[string]string{}}
	for _, day := range in.Week.Days() {
		entries := byDay[day]
		if len(entries) == 0 {
			continue
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].IssueKey < entries[j].IssueKey })
		total := len(entries) * in.PerKey.Minutes()
		reason := fmt.Sprintf("%d issue(s) x %s", len(entries), in.PerKey)
		if total > in.DailyCap.Minutes() {
			entries = scaleDay(entries, in.DailyCap.Minutes())
			reason += fmt.Sprintf(", scaled from %dm to the %s cap", total, in.DailyCap)
		}
		p.Entries = append(p.Entries, entries...)
		p.Reasoning[day.String()] = reason
	}
	return p, nil
}

// scaleDay shrinks one day's entries so their total fits within capMin.
//
// Contract (checked by TestRulePolicyScalesOverCapDay):
//   - the returned total is <= capMin
//   - every Duration is a positive multiple of kernel.Granularity (15m)
//   - no entry is dropped, and none goes below 15m
//   - entries keep their order, IssueKey, Day and Description
//
// Input: every entry has the same Duration (PerKey), and the sum is > capMin.
func scaleDay(entries []ProposedEntry, capMin int) []ProposedEntry {
	// TODO(owner): implement the scaling strategy.
	return entries
}

func appendUnique(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}
