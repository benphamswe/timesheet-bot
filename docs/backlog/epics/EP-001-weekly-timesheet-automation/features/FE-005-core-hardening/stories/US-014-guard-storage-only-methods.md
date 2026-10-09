---
id: US-014
title: Only the stores use the storage-only timesheet methods
status: Active
parent: FE-005
points: 2
sprint: SP-01
github:
fr: [NFR-5, NFR-11]
---

# US-014: Only the stores use the storage-only timesheet methods

As the timesheet owner, I want the storage-only operations on a timesheet usable only by the stores, so that no command can change a timesheet's version or drop its pending events without the store knowing.

Finding: N7. `Rehydrate`, `Snapshot`, `SetVersion` and `PullEvents` are exported for the stores, and nothing stops other code from calling them.

## Acceptance criteria

```gherkin
# AC-014.1
Scenario: A call from outside the stores is caught
  Given non-test code under an application or interfaces package that calls Rehydrate, Snapshot, SetVersion or PullEvents
  When the architecture tests run
  Then they fail and name the call site

# AC-014.2
Scenario: The stores may call them
  Given the current code, where only the memory and Postgres stores call these methods
  When the architecture tests run
  Then they pass
```

## Dependencies

- US-012 (architecture test framework). If the plan instead makes these methods unreachable by design, AC-014.1 is proven by a build failure fixture.

## Plan

`docs/superpowers/plans/2026-10-09-sp01-hardening.md` (status: draft).

Draft plan coverage, read 2026-10-09: Partly covered: WP-1 removes `SetVersion` and `PullEvents` (replaced by `PendingEvents` and `MarkPersisted`). The call-site guard is not in the draft yet. The guarded set becomes `Rehydrate`, `Snapshot`, `PendingEvents` and `MarkPersisted`.

## Definition of Ready

The Definition of Ready and story points are team conventions, not Scrum Guide 2020 rules (see `docs/backlog/README.md`).

- [x] Story form, Gherkin AC with a failure path, at most 8 points, dependencies named, linked to its FR ids

## Tasks

Provisional. `technical-architect` aligns them with the plan's work packages.

| Id | Task | WP | Owner agent | Estimate (h) | Status |
|---|---|---|---|---|---|
| US-014-T1 | Call-site check for the storage-only methods, with a violation fixture | - | go-coder | 2 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-014.1 | planned | |
| AC-014.2 | planned | |
