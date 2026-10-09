---
id: US-011
title: Tie every stored event and delete to its owner
status: Active
parent: FE-005
points: 3
sprint: SP-01
github:
fr: [FR-TS-1, NFR-12]
---

# US-011: Tie every stored event and delete to its owner

As the timesheet owner, I want every timesheet event, outbox record and delete tied to my account, so that the tool can later serve more people without one person's data reaching another's.

Finding: M3, ADR-0006 seam 1 ("every aggregate and table is keyed by an owner id"). Events and outbox rows carry no owner id, and `Delete` takes only a timesheet id.

## Acceptance criteria

```gherkin
# AC-011.1
Scenario: Events and outbox rows carry the owner
  Given owner U1 has an awaiting-approval timesheet for 2026-W41
  When U1 approves it and it is saved
  Then the TimesheetApproved event carries owner U1
  And the outbox row for that event carries owner U1

# AC-011.2
Scenario: One owner cannot remove another owner's timesheet
  Given owner U1 has a timesheet T1 for 2026-W41
  When a delete or replace of T1 is requested for owner U2
  Then T1 is still stored for U1, unchanged

# AC-011.3
Scenario: Existing outbox rows get their owner
  Given outbox rows written before this change, without an owner
  When the database is migrated to the latest version
  Then every outbox row carries the owner of its timesheet
```

## Dependencies

- US-006: AC-011.2 applies to the replace operation US-006 introduces, if it replaces `Delete`.
- The conformance review names the migration `00002_outbox_owner.sql`. If US-016 has moved the migrations folder, the file goes in `internal/migrations/`.
- NFR-12 (owner scoping) records ADR-0006 seam 1 as a requirement.

## Plan

`docs/superpowers/plans/2026-10-09-sp01-hardening.md` (status: draft).

Draft plan coverage, read 2026-10-09: WP-1 (owner on events) and WP-7 (Task 7.1: migration `00002_outbox_owner.sql`; Task 7.2: owner-scoped `Replace`, `Delete` removed). The draft files this work as US-005-T3 and T6.

## Definition of Ready

The Definition of Ready and story points are team conventions, not Scrum Guide 2020 rules (see `docs/backlog/README.md`).

- [x] Story form, Gherkin AC with a failure path, at most 8 points, dependencies named, linked to its FR ids

## Tasks

Provisional. `technical-architect` aligns them with the plan's work packages.

| Id | Task | WP | Owner agent | Estimate (h) | Status |
|---|---|---|---|---|---|
| US-011-T1 | Owner id on events; migration adds and backfills the outbox owner column | - | go-coder | 3 | To Do |
| US-011-T2 | Owner-scoped delete or replace in both stores | - | go-coder | 2 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-011.1 | planned | |
| AC-011.2 | planned | |
| AC-011.3 | planned | |
