package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"timesheet-bot/internal/shared/kernel"
)

// ReportLine is one line of a timesheet summary the owner wrote in Slack. It
// is the shared kernel type, so a collector can deliver lines already parsed.
type ReportLine = kernel.ReportLine

// ReportParser reads the timesheet summary lines in one message. msg is the
// message as the Activity context delivers it: msg.Day is the date it was
// posted in tz, and msg.Text its text. A message with no report lines gives
// none and no error.
type ReportParser interface {
	Parse(msg kernel.WorkActivity, tz *time.Location) ([]ReportLine, error)
}

// ReportPolicy is the report mode policy: every line of the owner's own
// summaries becomes one entry, as written. It applies no per-key credit, no
// channel defaults and no scaling (ADR-0012).
//
// The lines come from one of two places. With report.parser: manual, Parser
// reads each activity. With report.parser: llm, Precomputed is set and the
// lines are in.ReportLines, parsed by the collector's LLM and validated by the
// file adapter; Parser is not called. Everything after that is shared.
type ReportPolicy struct {
	Parser      ReportParser
	Precomputed bool
	Location    *time.Location
}

// Name returns "report", the identifier recorded in [Proposal.Policy].
func (ReportPolicy) Name() string { return "report" }

// Allocate parses every activity in in, drops duplicate lines with
// [DedupReportLines], and proposes one entry per remaining line, in report
// order. A line dated on a weekend or outside in.Week is skipped with a
// warning; a day whose total exceeds in.DailyCap is kept as reported with a
// warning. A parse error, a malformed date, an empty project, or minutes that
// are not a positive multiple of 15 fail the proposal.
func (p ReportPolicy) Allocate(_ context.Context, in Input) (Proposal, error) {
	lines, err := p.lines(in)
	if err != nil {
		return Proposal{}, err
	}
	lines = DedupReportLines(lines)

	prop := Proposal{Policy: p.Name(), Reasoning: map[string]string{}}
	var errs []error
	totals := map[kernel.WorkDay]int{}
	counts := map[kernel.WorkDay]int{}
	for _, l := range lines {
		day, err := kernel.ParseWorkDay(l.Date)
		switch {
		case errors.Is(err, kernel.ErrWeekend):
			prop.Warnings = append(prop.Warnings, fmt.Sprintf("%s: %s %s is on a weekend; skipped", l.Date, l.Project, minutesText(l.Minutes)))
			continue
		case err != nil:
			errs = append(errs, fmt.Errorf("report line %q: date %q: %w", l.Project, l.Date, err))
			continue
		case !in.Week.Contains(day):
			prop.Warnings = append(prop.Warnings, fmt.Sprintf("%s: %s %s is outside %s; skipped", l.Date, l.Project, minutesText(l.Minutes), in.Week))
			continue
		}
		d, err := kernel.NewDuration(l.Minutes)
		if err != nil {
			errs = append(errs, fmt.Errorf("report line %s %q: %w", l.Date, l.Project, err))
			continue
		}
		if strings.TrimSpace(l.Project) == "" {
			errs = append(errs, fmt.Errorf("report line %s %s: no project", l.Date, d))
			continue
		}
		prop.Entries = append(prop.Entries, ProposedEntry{Project: strings.TrimSpace(l.Project), Day: day, Duration: d, Description: l.Notes})
		totals[day] += d.Minutes()
		counts[day]++
	}
	if err := errors.Join(errs...); err != nil {
		return Proposal{}, err
	}
	for _, day := range in.Week.Days() {
		if counts[day] == 0 {
			continue
		}
		prop.Reasoning[day.String()] = fmt.Sprintf("%d report line(s)", counts[day])
		if total := totals[day]; total > in.DailyCap.Minutes() {
			prop.Warnings = append(prop.Warnings, fmt.Sprintf("%s: %s reported, over the daily cap of %s; kept as reported, not scaled",
				day, minutesText(total), in.DailyCap))
		}
	}
	return prop, nil
}

func (p ReportPolicy) lines(in Input) ([]ReportLine, error) {
	if p.Precomputed {
		return in.ReportLines, nil
	}
	var lines []ReportLine
	for _, a := range in.Activities {
		got, err := p.Parser.Parse(a, p.Location)
		if err != nil {
			return nil, fmt.Errorf("parse report message of %s (%s): %w", a.Day, a.Permalink, err)
		}
		lines = append(lines, got...)
	}
	return lines, nil
}

// DedupReportLines drops exact duplicates and keeps the first of each, in
// order. The owner posts the same summary in the report channel and in their
// self-DM, so the same line arrives twice. Two lines are duplicates when
// [ReportDedupKey] is equal: same date, same project and same notes after
// [NormalizeReportText], and same minutes. Lines that differ in any of these,
// such as two separate 1h blocks on one project with different notes, are
// both kept. Change this function, and ADR-0012, to change the policy.
func DedupReportLines(lines []ReportLine) []ReportLine {
	seen := map[string]bool{}
	var out []ReportLine
	for _, l := range lines {
		k := ReportDedupKey(l)
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, l)
	}
	return out
}

// ReportDedupKey is the identity [DedupReportLines] compares. The source
// message fields are left out: the same line from two messages is a duplicate.
func ReportDedupKey(l ReportLine) string {
	return fmt.Sprintf("%s\x00%s\x00%d\x00%s", l.Date, NormalizeReportText(l.Project), l.Minutes, NormalizeReportText(l.Notes))
}

// NormalizeReportText lowercases s and collapses every run of white space to
// one space, trimming both ends.
func NormalizeReportText(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

func minutesText(m int) string {
	if d, err := kernel.NewDuration(m); err == nil {
		return d.String()
	}
	return fmt.Sprintf("%dm", m)
}
