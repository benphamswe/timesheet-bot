---
id: US-020
title: Every run leaves structured, secret-free logs
status: Active
parent: FE-005
points: 3
sprint: SP-01
github:
fr: [NFR-6, NFR-9]
---

# US-020: Every run leaves structured, secret-free logs

As the timesheet owner, I want each run to write structured log records of what it did and what failed, so that I can see why a Friday run went wrong without re-running it.

Finding: best-practice review F10 (high): the tool has no logging; it prints only the final result or error. Logging uses the standard library `log/slog`.

## Acceptance criteria

```gherkin
# AC-020.1
Scenario: A run is logged with its key facts
  Given an approved timesheet for 2026-W41 with 3 entries
  When I submit
  Then the log has one record per Tempo call with the command, the week, the entry id and the HTTP status
  And a final record with the counts submitted, reconciled and failed

# AC-020.2
Scenario: A failure is logged at error level
  Given Tempo answers 400 for GCM-2
  When I submit
  Then the log has an ERROR record with the entry id of GCM-2 and status 400

# AC-020.3
Scenario: Logs never carry a secret
  Given every secret variable is set to a unique sentinel value
  When plan, approve and submit run at debug level
  Then no log record contains a sentinel value
```

## Dependencies

- US-025 reuses the sentinel check for errors and the database.
- Assumption: the log format (JSON or text) and the default level are config settings chosen in the plan.

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
| US-020-T1 | slog setup in cmd/timesheet, level and format from config | - | go-coder | 1 | To Do |
| US-020-T2 | Log records in handlers and adapters | - | go-coder | 3 | To Do |
| US-020-T3 | Secret-free check with sentinel values | - | go-coder | 1 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-020.1 | planned | |
| AC-020.2 | planned | |
| AC-020.3 | planned | |
