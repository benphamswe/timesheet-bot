---
id: US-016
title: Names say what the code holds and checks
status: Active
parent: FE-005
points: 8
sprint: SP-01
github:
fr: [NFR-11]
---

# US-016: Names say what the code holds and checks

As the timesheet owner, I want package, constructor, folder and test names that say what they hold or check, so that I can find code and read a test report without opening the code.

## Naming items

| Id | Change | Decision |
|---|---|---|
| N-01 | `domain.New` becomes `domain.NewTimesheet` | Rename |
| N-02 | Package `slack` (`internal/activity/infrastructure/slack`) becomes `slacksearch` | Rename |
| N-03 | The architecture tests have one package comment and no non-test file. Met by the ADR-0009 move on 2026-10-09: `internal/archtest` became `test/architecture`, a single file `archtest_test.go` (`package architecture_test`) with the only package comment; `doc.go` and `read.go` are gone | Change (done) |
| N-04 | The `platform/db` package name shadows the common local variable name `db` | Rename; the plan picks the new name |
| N-05 | `Event.EventName()` becomes `Name()`, and the `Event.Name` field becomes `Type` | Rename; the JSON key `name` stays, so stored outbox payloads still decode |
| N-06 | `IssueDirectory` becomes `IssueResolver`; `WorklogGateway` becomes `WorklogCreator`, or both are documented as domain-noun ports | See below |
| N-07 | `migrations/` moves to `internal/migrations/` | Move (owner decision 2026-10-09: fix everything) |
| N-08 | `slack.Query` is unexported (`query`) | Change |
| N-09 | Test helpers: `approvedTimesheet` becomes `approveTimesheet`, `act` becomes `activity` | Rename |
| N-10 | Test names describe behaviour, for example `TestCreate`, `TestResolve`, `TestDuration`, `TestQuery`, `TestIssueKey` | Rename, plus a check (AC-016.4) |
| N-11 | `EventEntrySubmitFail` becomes `EventEntrySubmissionFailed`; its stored value `EntrySubmissionFailed` does not change | Rename |

N-06 decision (owner decision 2026-10-09, "fix everything", with the recommended default): rename `IssueDirectory` to `IssueResolver`, because its only method is `Resolve`. Keep `WorklogGateway` and document it as a domain-noun port in `docs/development/conventions.md`. The draft plan gives it a second method, `ListByAuthor` (US-007), so `WorklogCreator` would no longer describe it. Coding standard N6 allows domain nouns for ports.

## Acceptance criteria

```gherkin
# AC-016.1
Scenario: The timesheet constructor is named for what it builds (N-01)
  Given the code base
  When it builds
  Then timesheets are created with domain.NewTimesheet
  And no exported function named New remains in timesheet/domain

# AC-016.2
Scenario: The Slack collector package is named for what it does (N-02, N-08)
  Given the code base
  When it builds
  Then the Slack collector is package slacksearch under internal/activity/infrastructure/slacksearch
  And no package named slack remains under internal
  And the package exports no Query function

# AC-016.3
Scenario: Migrations load from their new folder (N-07)
  Given an empty database
  When the app migrates it
  Then the migrations embedded from internal/migrations/ apply and the schema is at the latest version
  And no migrations/ folder remains at the service root

# AC-016.4
Scenario: A test named only after a function fails the check (N-10)
  Given a test whose name is Test followed only by an identifier declared in its package, such as TestNew or TestCreate
  When the architecture tests run
  Then they fail and name the test

# AC-016.5
Scenario: Renamed tests stay traceable (N-10)
  Given the renamed tests
  When every test named in a story's traceability table is looked up in the code base
  Then each one exists under that name

# AC-016.6
Scenario: Events stored before the rename still decode (N-05, N-11)
  Given an outbox payload stored before the rename, with key "name" set to "EntrySubmissionFailed"
  When it is decoded into an Event
  Then its Type is EventEntrySubmissionFailed

# AC-016.7
Scenario: No old name remains (N-03, N-04, N-05, N-06, N-09, N-11)
  Given the code base after the renames
  When it is searched for EventName, EventEntrySubmitFail, IssueDirectory, approvedTimesheet, the test helper act and the package name db under internal/platform
  Then none is found
  And test/architecture has exactly one package comment and no non-test file
  And conventions.md documents WorklogGateway as a domain-noun port
```

## Dependencies

- After US-015 (clean lint baseline) and together with US-008 (constructor signature, N-01).
- N-07 after the draft plan's WP-7, which adds `migrations/00002_outbox_owner.sql` at the old path, or WP-7 writes the file to `internal/migrations/` directly.
- N-07 is a reversal: the conformance review of 2026-10-09 listed `migrations/` at the root as "deliberately fine". The owner chose to move it (2026-10-09, "fix everything"). `go:embed` only needs `embed.go` beside the SQL files, which is still true at `internal/migrations/`. `docs/architecture/project-structure.md` needs the same update (tech-writer).

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
| US-016-T1 | Production renames N-01, N-02, N-04, N-05, N-06, N-08, N-11 | - | go-coder | 4 | To Do |
| US-016-T2 | N-07: move migrations to internal/migrations and update the import in platform/db | - | go-coder | 1 | To Do |
| US-016-T3 | N-03: one package comment and no non-test file in the architecture tests (met by the ADR-0009 move to `test/architecture`) | - | go-coder | 0.5 | Done |
| US-016-T4 | N-09, N-10: test helper and test renames, the test-name check, traceability updates | - | go-coder | 3 | To Do |
| US-016-T5 | conventions.md: domain-noun ports (N-06) | - | tech-writer | 0.5 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-016.1 | planned | |
| AC-016.2 | planned | |
| AC-016.3 | planned | |
| AC-016.4 | planned | |
| AC-016.5 | planned | |
| AC-016.6 | planned | |
| AC-016.7 | planned | |
