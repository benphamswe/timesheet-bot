---
id: US-027
title: Each package keeps at least 85% test coverage
status: Active
parent: FE-005
points: 2
sprint: SP-01
github:
fr: [NFR-11]
---

# US-027: Each package keeps at least 85% test coverage

As the timesheet owner, I want CI to fail when any package drops below 85% statement coverage, so that new code cannot land untested.

Finding: test review (per-package coverage gate of 85%).

## Acceptance criteria

```gherkin
# AC-027.1
Scenario: A package below the gate fails CI
  Given a package with statement coverage below 85%
  When make ci runs
  Then it fails and names the package and its coverage

# AC-027.2
Scenario: Covered packages pass
  Given every package at 85% or more
  When make ci runs
  Then the coverage step passes

# AC-027.3
Scenario: Exclusions are explicit
  Given a package listed in the coverage exclusion file with a reason
  When make ci runs
  Then the coverage step reports it as excluded with that reason
  And a package not on the list cannot be skipped
```

## Dependencies

- Coverage includes the integration tests, so Postgres code is measured; CI runs them with the compose database.
- Candidates for exclusion, such as cmd/timesheet and test-only packages, are decided in the plan.

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
| US-027-T1 | Coverage script, exclusion file, make ci step | - | go-coder | 2 | To Do |
| US-027-T2 | Raise packages below 85% | - | go-coder | 3 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-027.1 | planned | |
| AC-027.2 | planned | |
| AC-027.3 | planned | |
