---
id: US-029
title: Clean up the low review findings
status: Active
parent: FE-005
points: 5
sprint: SP-01
github:
fr: [NFR-11]
---

# US-029: Clean up the low review findings

As the timesheet owner, I want the small design flaws from the conformance review fixed, so that each rule lives in one place and later phases do not copy them.

Findings: conformance review 2026-10-09, L6 (slot rule and Tempo formats in the application port), L7 (unused `activity/domain.Source`), L8 (double issue cache), L9 (value objects can bypass their constructors), L11 (`domain` package alias convention: keep the aliases and document them).

## Acceptance criteria

```gherkin
# AC-029.1
Scenario: Start times are a timesheet rule (L6)
  Given an approved timesheet with GCM-1 and GCM-2 on Monday and GCM-1 on Tuesday
  When the timesheet is asked for each entry's start time
  Then the times are 09:00 and 09:45 on Monday and 09:00 on Tuesday
  And only the Tempo adapter formats times as text

# AC-029.2
Scenario: The unused activity port is gone (L7)
  Given the code base
  When it builds
  Then no package internal/activity/domain exists

# AC-029.3
Scenario: An issue is looked up once per run (L8)
  Given activity mentioning GCM-1 on three days
  When I run plan for 2026-W41
  Then Jira is asked about GCM-1 once

# AC-029.4
Scenario: Zero-value value objects are refused (L9)
  Given the zero value of WeekID, WorkDay, IssueKey or Duration
  When it is given to the timesheet
  Then the timesheet refuses it with a validation error

# AC-029.5
Scenario: Domain package aliases follow one convention (L11)
  Given conventions.md documents the alias for each context's domain package
  When the architecture tests run
  Then an import of another context's domain package without its documented alias fails, naming the file
```

## Dependencies

- L7 is done by the draft plan's WP-2 (Task 2.2); AC-029.2 records it.
- L9 overlaps US-008, which covers Duration and IssueKey inside AddEntry; this story covers every value object.
- AC-029.5 uses the US-012 checks.

## Plan

`docs/superpowers/plans/2026-10-09-sp01-hardening.md` (status: draft).

Draft plan coverage, read 2026-10-09: L7 in WP-2 (Task 2.2). The draft defers L6, L8, L9 and L11 (section 8, "Deferred").

## Definition of Ready

The Definition of Ready and story points are team conventions, not Scrum Guide 2020 rules (see `docs/backlog/README.md`).

- [x] Story form, Gherkin AC with a failure path, at most 8 points, dependencies named, linked to its FR ids

## Tasks

Provisional. `technical-architect` aligns them with the plan's work packages.

| Id | Task | WP | Owner agent | Estimate (h) | Status |
|---|---|---|---|---|---|
| US-029-T1 | L6: slot rule into the domain, time format into the Tempo adapter | - | go-coder | 3 | To Do |
| US-029-T2 | L8: one issue cache | - | go-coder | 1 | To Do |
| US-029-T3 | L9: zero-value guards for kernel value objects | - | go-coder | 3 | To Do |
| US-029-T4 | L11: alias convention in conventions.md and an archtest check | - | go-coder | 1.5 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-029.1 | planned | |
| AC-029.2 | planned | |
| AC-029.3 | planned | |
| AC-029.4 | planned | |
| AC-029.5 | planned | |
