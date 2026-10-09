---
id: US-003
title: Propose and store the week's timesheet
status: Resolved
parent: FE-001
points: 5
sprint: SP-01
github:
fr: [FR-ALC-5, FR-TS-1, FR-TS-2]
---

# US-003: Propose and store the week's timesheet

As the timesheet owner, I want one validated proposal per week stored safely, so that I review exactly what will be logged.

## Acceptance criteria

```gherkin
# AC-003.1
Scenario: A proposal drops unknown issue keys
  Given my activity mentions GCM-1, GCM-2 and UNKNOWN-9
  And Jira knows only GCM-1 and GCM-2
  When I run plan for 2026-W41
  Then the timesheet awaits approval with entries for GCM-1 and GCM-2 only

# AC-003.2
Scenario: A week is proposed only once unless forced
  Given a proposal exists for 2026-W41
  When I run plan again without --force
  Then it fails with "already exists"
  And with --force a new proposal replaces it

# AC-003.3
Scenario: An invalid policy proposal falls back to the rule policy
  Given the first policy proposes 600 minutes on Monday
  When the proposal is validated against the 8h cap
  Then the rule policy's proposal is used and the rejection reason is reported

# AC-003.4
Scenario: Concurrent edits do not overwrite each other
  Given two copies of the same stored timesheet
  When both are changed and saved
  Then the second save fails with a concurrent modification error
```

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-003.1 | `TestProposeWithRulePolicy` | pass |
| AC-003.2 | `TestProposeAlreadyProposedAndForce` | pass |
| AC-003.3 | `TestProposeFallsBackWhenProposalInvalid` | pass |
| AC-003.4 | `TestRoundTripConflictAndOutbox` (integration) | pass |
