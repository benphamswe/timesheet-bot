---
id: US-013
title: Take ids and the current time from injected sources
status: Active
parent: FE-005
points: 2
sprint: SP-01
github:
fr: [FR-TS-1, NFR-11]
---

# US-013: Take ids and the current time from injected sources

As the timesheet owner, I want the default week and new timesheet ids to come from the configured clock and id source, so that the week picked near midnight is always the one in my timezone and every run can be reproduced in a test.

Findings: N5 (`timesheet/application/propose.go` calls `uuid.NewString()`), N6 (`interfaces/cli/root.go` calls `time.Now()` for the default week).

## Acceptance criteria

```gherkin
# AC-013.1
Scenario: The default week follows the configured clock and timezone
  Given the clock reads Sunday 2026-10-11 23:30 UTC
  And the timezone is Asia/Ho_Chi_Minh
  When I run plan without a week
  Then the week is 2026-W42

# AC-013.2
Scenario: A new timesheet takes its id from the id source
  Given an id source that returns "ts-1"
  When plan stores a new proposal for 2026-W41
  Then the stored timesheet's id is "ts-1"

# AC-013.3
Scenario: The application layer cannot generate ids itself
  Given application code that imports github.com/google/uuid
  When the architecture tests run
  Then they fail and name the import

# AC-013.4
Scenario: The CLI cannot read the wall clock itself
  Given code under internal/interfaces that calls time.Now
  When the architecture tests run
  Then they fail and name the file and line
```

## Dependencies

- AC-013.3 and AC-013.4 use the checks of US-012 (application allow rule, wall-clock check extended to `internal/interfaces`).

## Plan

`docs/superpowers/plans/2026-10-09-sp01-hardening.md` (status: draft).

Draft plan coverage, read 2026-10-09: Not covered by the draft yet. It conflicts with WP-3's uuid allowance (see US-012).

## Definition of Ready

The Definition of Ready and story points are team conventions, not Scrum Guide 2020 rules (see `docs/backlog/README.md`).

- [x] Story form, Gherkin AC with a failure path, at most 8 points, dependencies named, linked to its FR ids

## Tasks

Provisional. `technical-architect` aligns them with the plan's work packages.

| Id | Task | WP | Owner agent | Estimate (h) | Status |
|---|---|---|---|---|---|
| US-013-T1 | Id source port; wire uuid in cmd/timesheet | - | go-coder | 1 | To Do |
| US-013-T2 | CLI default week from the injected clock | - | go-coder | 1 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-013.1 | planned | |
| AC-013.2 | planned | |
| AC-013.3 | planned | |
| AC-013.4 | planned | |
