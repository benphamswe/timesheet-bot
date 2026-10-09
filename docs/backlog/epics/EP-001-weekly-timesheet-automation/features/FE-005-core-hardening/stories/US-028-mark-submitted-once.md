---
id: US-028
title: A submitted entry keeps its Tempo worklog id
status: Active
parent: FE-005
points: 1
sprint: SP-01
github:
fr: [FR-SUB-2, NFR-1]
---

# US-028: A submitted entry keeps its Tempo worklog id

As the timesheet owner, I want an entry that already has its Tempo worklog id to keep it, so that a repeated or wrong update can never point the entry at a different worklog.

Finding: test review Q2. A second `MarkEntrySubmitted` call overwrites the Tempo worklog id.

## Acceptance criteria

```gherkin
# AC-028.1
Scenario: The same id again changes nothing
  Given GCM-1 on Monday is submitted with worklog id 9001
  When it is marked submitted again with worklog id 9001
  Then nothing changes
  And no event is recorded

# AC-028.2
Scenario: A different id is refused
  Given GCM-1 on Monday is submitted with worklog id 9001
  When it is marked submitted with worklog id 9002
  Then it fails with an already-submitted error
  And the entry keeps worklog id 9001
```

## Dependencies

- Naming: the test review calls the error `ErrEntryAlreadySubmitted`; the draft plan adds `ErrEntrySubmitted` for `BeginEntrySubmit`. Use one sentinel; technical-architect picks the name.

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
| US-028-T1 | Guard in MarkEntrySubmitted with tests | - | go-coder | 1 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-028.1 | planned | |
| AC-028.2 | `TestMarkEntrySubmittedKeepsFirstWorklogID` | skipped (BUG exactly-once; un-skip with this story) |
