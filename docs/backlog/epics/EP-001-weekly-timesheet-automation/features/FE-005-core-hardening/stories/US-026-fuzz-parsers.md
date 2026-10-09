---
id: US-026
title: Parsers survive any input
status: Active
parent: FE-005
points: 2
sprint: SP-01
github:
fr: [NFR-2]
---

# US-026: Parsers survive any input

As the timesheet owner, I want every parser of outside input fuzzed, so that a strange Slack message or API response cannot crash a run.

Finding: test review (fuzz targets).

## Acceptance criteria

```gherkin
# AC-026.1
Scenario: Kernel parsers never panic and round-trip
  Given fuzz targets for ParseWeekID, ParseWorkDay, ExtractIssueKeys, NewIssueKey and NewDuration
  When each runs for 30 seconds
  Then none panics
  And every accepted value formats back to text that parses to the same value

# AC-026.2
Scenario: Response decoders never panic
  Given fuzz targets for the Tempo and Slack response decoders
  When each runs for 30 seconds on arbitrary bytes
  Then none panics

# AC-026.3
Scenario: A crash found once stays tested
  Given a fuzz input that once made a target fail
  When go test runs
  Then that input runs as a regression case from testdata/fuzz
```

## Dependencies

- Fuzz runs are a separate make target; go test runs only the seed corpus.

## Plan

`docs/superpowers/plans/2026-10-09-sp01-hardening.md` (status: draft).

Draft plan coverage, read 2026-10-09: not covered yet.

## Definition of Ready

The Definition of Ready and story points are team conventions, not Scrum Guide 2020 rules (see `docs/backlog/README.md`).

- [x] Story form, Gherkin AC with a failure path, at most 8 points, dependencies named, linked to its FR ids

## Tasks

Provisional. `technical-architect` aligns them with the plan's work packages.

| Id | Task | WP | Owner agent | Estimate (h) | Status |
|---|---|---|---|---|---|
| US-026-T1 | Kernel fuzz targets | - | go-coder | 1.5 | To Do |
| US-026-T2 | Decoder fuzz targets and make target | - | go-coder | 1.5 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-026.1 | planned | |
| AC-026.2 | planned | |
| AC-026.3 | planned | |
