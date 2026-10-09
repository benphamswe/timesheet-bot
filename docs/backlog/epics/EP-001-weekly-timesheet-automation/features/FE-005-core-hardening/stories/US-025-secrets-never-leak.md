---
id: US-025
title: Tests prove secrets never leak
status: Active
parent: FE-005
points: 2
sprint: SP-01
github:
fr: [NFR-6]
---

# US-025: Tests prove secrets never leak

As the timesheet owner, I want tests that prove no token ever appears in an error, an output line or the database, so that a log or bug report I share never exposes my credentials.

Finding: test review (security tests for NFR-6).

## Acceptance criteria

```gherkin
# AC-025.1
Scenario: Adapter errors carry no secret
  Given every secret variable set to a unique sentinel value
  When each adapter meets a 401, a 500, a timeout and an unreadable body
  Then no returned error contains a sentinel

# AC-025.2
Scenario: Config errors name the variable, not the value
  Given JIRA_BASE_URL is set to a value that is not a URL
  When the config is loaded
  Then the error names JIRA_BASE_URL and does not contain its value

# AC-025.3
Scenario: A full run stores and prints no secret
  Given sentinel secrets
  When the end-to-end run of US-022 completes
  Then stdout, stderr and every database row contain no sentinel
```

## Dependencies

- US-022 for AC-025.3.

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
| US-025-T1 | Sentinel helper and adapter error-path tests | - | go-coder | 2 | To Do |
| US-025-T2 | Config and end-to-end leak checks | - | go-coder | 1 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-025.1 | planned | |
| AC-025.2 | planned | |
| AC-025.3 | planned | |
