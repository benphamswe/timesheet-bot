---
id: US-007
title: Recover a worklog whose Tempo outcome is unknown
status: Active
parent: FE-005
points: 5
sprint: SP-01
github:
fr: [FR-SUB-2, NFR-1]
---

# US-007: Recover a worklog whose Tempo outcome is unknown

As the timesheet owner, I want submit to check Tempo before it re-sends an entry whose last attempt has an unknown outcome, so that a crash or a server error never leaves a duplicate worklog in my Tempo week.

Finding: H2 (conformance review 2026-10-09), coding standards R3 and K2. This extends US-005. US-005-T2 and AC-005.3 cover the run in which Tempo answers 5xx: the create is not re-sent. This story covers the next run: the entry has no worklog id, but Tempo may already hold its worklog.

## Acceptance criteria

```gherkin
# AC-007.1
Scenario: A worklog stored before a crash is recovered, not re-posted
  Given an approved timesheet for 2026-W41 with GCM-1 on Monday at 09:00
  And Tempo stored the worklog for GCM-1, but the run stopped before its worklog id was saved
  When I submit again
  Then no new worklog is created in Tempo
  And the GCM-1 entry records the existing Tempo worklog id

# AC-007.2
Scenario: A worklog stored behind a server error is recovered on the next run
  Given an approved timesheet for 2026-W41 with GCM-2 on Monday at 09:45
  And Tempo stored the worklog for GCM-2 but answered 502
  When I submit again
  Then Tempo holds exactly one worklog for GCM-2 on Monday
  And the GCM-2 entry records that worklog's id

# AC-007.3
Scenario: A worklog Tempo never stored is sent once in its slot
  Given the last attempt for GCM-2 on Monday had an unknown outcome
  And Tempo holds no worklog that the bot created for that entry
  When I submit again
  Then GCM-2 is sent once, starting at 09:45 on Monday

# AC-007.4
Scenario: A failed check never falls back to posting
  Given the last attempt for GCM-2 on Monday had an unknown outcome
  And looking the worklog up in Tempo fails with 503
  When I submit
  Then GCM-2 is not sent
  And GCM-2 stays unsent with the lookup error recorded
  And submit exits with an error that names GCM-2
```

## Dependencies

- US-005-T2: the transport no longer re-sends a create on 5xx or a timeout.
- Assumption: Tempo v4 takes no idempotency key, so recovery is a lookup (K2). The plan decides how a worklog is recognised as the one the bot created for an entry (for example author, issue, date, start time and a marker). The marker never contains personal data (K3).

## Plan

`docs/superpowers/plans/2026-10-09-sp01-hardening.md` (status: draft).

Draft plan coverage, read 2026-10-09: WP-1 (attempt marker), WP-5 (Task 5.1: reconcile before re-posting) and WP-6 (Task 6.1: `ErrCreateIndeterminate`, `ListByAuthor`). The draft files this work as US-005-T3 to T5. US-005-T2 is WP-2.

## Definition of Ready

The Definition of Ready and story points are team conventions, not Scrum Guide 2020 rules (see `docs/backlog/README.md`).

- [x] Story form, Gherkin AC with a failure path, at most 8 points, dependencies named, linked to its FR ids

## Tasks

Provisional. `technical-architect` aligns them with the plan's work packages.

| Id | Task | WP | Owner agent | Estimate (h) | Status |
|---|---|---|---|---|---|
| US-007-T1 | Record an attempt as outcome-unknown before the create is sent | - | go-coder | 2 | To Do |
| US-007-T2 | Tempo lookup for the bot's worklog of an entry (gateway port and adapter) | - | go-coder | 3 | To Do |
| US-007-T3 | Submit reconciles outcome-unknown entries before sending | - | go-coder | 3 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-007.1 | planned | |
| AC-007.2 | planned | |
| AC-007.3 | planned | |
| AC-007.4 | planned | |
