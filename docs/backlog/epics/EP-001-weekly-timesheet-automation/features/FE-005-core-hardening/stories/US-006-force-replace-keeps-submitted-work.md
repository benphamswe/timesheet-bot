---
id: US-006
title: Replace a week's proposal without losing submitted work
status: Active
parent: FE-005
points: 3
sprint: SP-01
github:
fr: [FR-TS-2, NFR-1]
---

# US-006: Replace a week's proposal without losing submitted work

As the timesheet owner, I want `plan --force` to refuse a week that already has entries in Tempo, and to replace any other proposal in one step, so that forcing a new proposal never posts the same work to Tempo twice and never leaves the week with no timesheet.

Finding: H1 (conformance review 2026-10-09). Today `plan --force` only refuses a fully `submitted` week, and the delete and the save run in two separate transactions.

## Acceptance criteria

```gherkin
# AC-006.1
Scenario: A partly submitted week cannot be replaced
  Given an approved timesheet for 2026-W41
  And GCM-1 on Monday is already in Tempo with worklog id 9001
  And GCM-2 on Monday is not yet sent
  When I run plan for 2026-W41 with --force
  Then it fails with "partly submitted"
  And the stored timesheet still holds worklog id 9001 for GCM-1
  And no request is sent to Tempo

# AC-006.2
Scenario: A proposal with nothing in Tempo is replaced in one step
  Given a timesheet awaiting approval for 2026-W41 with no entry in Tempo
  When I run plan for 2026-W41 with --force
  Then exactly one timesheet is stored for 2026-W41
  And it is the new proposal, awaiting approval

# AC-006.3
Scenario: A failed replacement keeps the previous proposal
  Given a timesheet awaiting approval for 2026-W41
  When I run plan for 2026-W41 with --force
  And storing the new proposal fails
  Then plan reports the storage error
  And the previous proposal is still stored, unchanged
```

## Dependencies

- None blocking. US-011 (owner-scoped delete) builds on the replace operation this story introduces.

## Plan

`docs/superpowers/plans/2026-10-09-sp01-hardening.md` (status: draft).

Draft plan coverage, read 2026-10-09: WP-1 (Tasks 1.1 to 1.3: `Discard`, `Replace` port, H1 in propose) and WP-7 (Task 7.2: Postgres `Replace`). The draft files this work as US-005-T3 and T6 and proposes AC-005.4, which duplicates AC-006.1.

## Definition of Ready

The Definition of Ready and story points are team conventions, not Scrum Guide 2020 rules (see `docs/backlog/README.md`).

- [x] Story form, Gherkin AC with a failure path, at most 8 points, dependencies named, linked to its FR ids

## Tasks

Provisional. `technical-architect` aligns them with the plan's work packages.

| Id | Task | WP | Owner agent | Estimate (h) | Status |
|---|---|---|---|---|---|
| US-006-T1 | Refuse force-replace when any entry has a Tempo worklog id | - | go-coder | 1 | To Do |
| US-006-T2 | Atomic replace in both stores (one transaction in Postgres) | - | go-coder | 3 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-006.1 | planned | |
| AC-006.2 | planned | |
| AC-006.3 | planned | |
