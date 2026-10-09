---
id: US-018
title: Database calls are bounded, consistent and clearly reported
status: Active
parent: FE-005
points: 3
sprint: SP-01
github:
fr: [FR-TS-1, NFR-5]
---

# US-018: Database calls are bounded, consistent and clearly reported

As the timesheet owner, I want every database call to finish in bounded time, read one consistent state, and report what went wrong, so that a stuck or busy database never hangs the tool or shows me half-saved data.

Findings: best-practice review F6 (error mapping), F8 (statement timeouts), F9 (consistent `Get`).

## Acceptance criteria

```gherkin
# AC-018.1
Scenario: A database outage is reported as an outage
  Given the database is unreachable
  When I run approve for 2026-W41
  Then the error says the database is unavailable
  And it is not reported as not found or already exists

# AC-018.2
Scenario: A slow query is stopped
  Given a store query that runs longer than the configured statement timeout
  When any store operation runs it
  Then it fails with a timeout error within the timeout plus one second
  And nothing from that operation is stored

# AC-018.3
Scenario: A read returns one committed state
  Given a timesheet for 2026-W41 that is saved repeatedly with a changing set of entries
  When it is read at the same time
  Then every read returns the entries and the version of one committed save
```

## Dependencies

- US-010 maps the unique violation; this story maps the other database errors.
- The statement timeout value is set in config with a default chosen in the plan.

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
| US-018-T1 | Map pgx errors (no rows, unavailable, others wrapped with the operation) (F6) | - | go-coder | 1 | To Do |
| US-018-T2 | Statement timeout on the pool and per query (F8) | - | go-coder | 1.5 | To Do |
| US-018-T3 | `Get` reads timesheet and entries in one transaction (F9) | - | go-coder | 1.5 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-018.1 | planned | |
| AC-018.2 | planned | |
| AC-018.3 | planned | |
