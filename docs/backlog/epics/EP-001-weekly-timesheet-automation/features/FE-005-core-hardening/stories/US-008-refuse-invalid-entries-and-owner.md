---
id: US-008
title: Refuse timesheet entries and owners that cannot be valid
status: Active
parent: FE-005
points: 2
sprint: SP-01
github:
fr: [FR-TS-1, FR-ALC-5, NFR-2]
---

# US-008: Refuse timesheet entries and owners that cannot be valid

As the timesheet owner, I want the timesheet to refuse a zero-length entry, an entry without an issue key, or a timesheet without an owner, so that nothing invalid can be stored or sent to Tempo, whichever policy produced it.

Finding: N4. The zero value of a duration or an issue key skips its constructor, and the timesheet does not check its owner.

## Acceptance criteria

```gherkin
# AC-008.1
Scenario: A valid entry is accepted
  Given a draft timesheet for 2026-W41
  When an entry for GCM-1 on Monday of 45 minutes is added
  Then the timesheet has one entry, GCM-1 on Monday for 45m

# AC-008.2
Scenario: A zero-length entry is refused
  Given a draft timesheet for 2026-W41
  When an entry for GCM-1 on Monday of 0 minutes is added
  Then it fails with an invalid duration error
  And the timesheet has no entries

# AC-008.3
Scenario: An entry without an issue key is refused
  Given a draft timesheet for 2026-W41
  When an entry with an empty issue key on Monday of 45 minutes is added
  Then it fails with an invalid issue key error
  And the timesheet has no entries

# AC-008.4
Scenario: A timesheet without an owner is refused
  When a timesheet for 2026-W41 is created with an empty owner
  Then it fails with a missing owner error
  And no timesheet is returned
```

## Dependencies

- AC-008.4 changes the timesheet constructor to return an error. US-016 renames it to `NewTimesheet`. Make both changes in the same work package so the call sites change once.

## Plan

`docs/superpowers/plans/2026-10-09-sp01-hardening.md` (status: draft).

Draft plan coverage, read 2026-10-09: Not covered by the draft yet.

## Definition of Ready

The Definition of Ready and story points are team conventions, not Scrum Guide 2020 rules (see `docs/backlog/README.md`).

- [x] Story form, Gherkin AC with a failure path, at most 8 points, dependencies named, linked to its FR ids

## Tasks

Provisional. `technical-architect` aligns them with the plan's work packages.

| Id | Task | WP | Owner agent | Estimate (h) | Status |
|---|---|---|---|---|---|
| US-008-T1 | Guard zero duration and empty issue key in AddEntry | - | go-coder | 1 | To Do |
| US-008-T2 | Constructor refuses an empty owner and returns an error | - | go-coder | 1 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-008.1 | planned | |
| AC-008.2 | planned | |
| AC-008.3 | planned | |
| AC-008.4 | planned | |
