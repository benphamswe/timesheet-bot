---
id: US-009
title: Keep events until the save is committed
status: Active
parent: FE-005
points: 2
sprint: SP-01
github:
fr: [NFR-5]
---

# US-009: Keep events until the save is committed

As the timesheet owner, I want a change's events kept until the change is safely stored, so that a failed save never loses the notice that my timesheet was approved, rejected or submitted.

Finding: M5. The Postgres store takes the events off the timesheet and raises its version inside the transaction, before the commit. If the commit fails in a long-running process (the Phase 2 bot), the events are gone and the version is wrong.

## Acceptance criteria

```gherkin
# AC-009.1
Scenario: A failed commit keeps the events and the version
  Given an approved timesheet for 2026-W41 at version 2
  And its TimesheetApproved event is not yet stored
  When the save fails at commit
  Then the timesheet still holds the TimesheetApproved event
  And its version is still 2

# AC-009.2
Scenario: A retried save stores the events exactly once
  Given the save of AC-009.1 failed at commit
  When the timesheet is saved again and the commit succeeds
  Then the outbox holds the TimesheetApproved event exactly once
  And the timesheet holds no unstored events
  And its version is 3
```

## Dependencies

- None. Run the same scenarios against the memory store (see US-010).

## Plan

`docs/superpowers/plans/2026-10-09-sp01-hardening.md` (status: draft).

Draft plan coverage, read 2026-10-09: WP-1 (`PendingEvents`, `MarkPersisted`) and WP-7 (Task 7.2: `Save` after commit). The draft files this work as US-005-T3 and T6.

## Definition of Ready

The Definition of Ready and story points are team conventions, not Scrum Guide 2020 rules (see `docs/backlog/README.md`).

- [x] Story form, Gherkin AC with a failure path, at most 8 points, dependencies named, linked to its FR ids

## Tasks

Provisional. `technical-architect` aligns them with the plan's work packages.

| Id | Task | WP | Owner agent | Estimate (h) | Status |
|---|---|---|---|---|---|
| US-009-T1 | Clear events and set the version only after commit (Postgres and memory) | - | go-coder | 2 | To Do |
| US-009-T2 | Integration test with a commit failure | - | go-coder | 2 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-009.1 | planned | |
| AC-009.2 | planned | |
