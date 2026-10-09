---
id: US-010
title: Report conflicting saves the same way in every store
status: Active
parent: FE-005
points: 2
sprint: SP-01
github:
fr: [FR-TS-1, NFR-9]
---

# US-010: Report conflicting saves the same way in every store

As the timesheet owner, I want the dry-run store and the database to treat conflicting saves the same way, so that what a dry run or a test shows is what the real run does.

Finding: L10. When a second new timesheet is saved for a week that already has one, the memory store returns the concurrent modification error, but Postgres returns a wrapped unique-key error. The test review adds a shared repository contract, `domaintest.RunRepositoryContract`, so every store is held to the same behaviour.

## Acceptance criteria

```gherkin
# AC-010.1
Scenario: A second new timesheet for the same week is refused alike
  Given a stored timesheet for 2026-W41
  When a different new timesheet for the same owner and 2026-W41 is saved
  Then the save fails with a concurrent modification error
  And this holds for both the memory store and the Postgres store

# AC-010.2
Scenario: A stale copy is refused alike
  Given two copies of the same stored timesheet for 2026-W41
  When both are changed and saved
  Then the second save fails with a concurrent modification error
  And this holds for both the memory store and the Postgres store

# AC-010.3
Scenario: Every store passes one shared repository contract
  Given the shared repository contract domaintest.RunRepositoryContract
  When it runs against the memory store and against the Postgres store
  Then both pass every scenario: get, save, replace, conflicting saves, owner scoping and events kept until commit

# AC-010.4
Scenario: A store that breaks the contract is caught
  Given a store that accepts a stale copy without an error
  When the repository contract runs against it
  Then the contract fails and names the conflicting-save scenario
```

## Dependencies

- Owner decision 2026-10-09 ("fix everything"): Postgres maps a duplicate (SQLSTATE 23505) to `ErrConcurrentModification`, the same error the memory store returns.
- AC-010.3 covers the scenarios of US-006, US-009 and US-011, so the contract grows with them.
- `domaintest` is a non-test package under `timesheet/domain` that imports `testing`. After US-012, the archtest domain rule must allow it.

## Plan

`docs/superpowers/plans/2026-10-09-sp01-hardening.md` (status: draft).

Draft plan coverage, read 2026-10-09: WP-7 (Task 7.2: SQLSTATE 23505 mapped to `ErrConcurrentModification`). `domaintest.RunRepositoryContract` is not in the draft yet.

## Definition of Ready

The Definition of Ready and story points are team conventions, not Scrum Guide 2020 rules (see `docs/backlog/README.md`).

- [x] Story form, Gherkin AC with a failure path, at most 8 points, dependencies named, linked to its FR ids

## Tasks

Provisional. `technical-architect` aligns them with the plan's work packages.

| Id | Task | WP | Owner agent | Estimate (h) | Status |
|---|---|---|---|---|---|
| US-010-T1 | Map the Postgres unique violation to `ErrConcurrentModification` | - | go-coder | 1 | To Do |
| US-010-T2 | `domaintest.RunRepositoryContract`, run from the memory tests and the Postgres integration tests | - | go-coder | 3 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-010.1 | planned | |
| AC-010.2 | planned | |
| AC-010.3 | planned | |
| AC-010.4 | planned | |
