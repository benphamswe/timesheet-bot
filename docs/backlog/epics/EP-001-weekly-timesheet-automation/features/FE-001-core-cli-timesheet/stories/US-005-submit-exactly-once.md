---
id: US-005
title: Submit approved worklogs to Tempo exactly once
status: Active
parent: FE-001
points: 5
sprint: SP-01
github:
fr: [FR-SUB-1, FR-SUB-2]
---

# US-005: Submit approved worklogs to Tempo exactly once

As the timesheet owner, I want approved entries written to Tempo once each, in sensible time slots, so that my Tempo week is correct even when Tempo fails part-way.

## Acceptance criteria

```gherkin
# AC-005.1
Scenario: Entries stack from 09:00 each day
  Given an approved timesheet with GCM-1 and GCM-2 on Monday and GCM-1 on Tuesday
  When I submit
  Then the start times are 09:00 and 09:45 on Monday and 09:00 on Tuesday

# AC-005.2
Scenario: A partial failure retries only the failed entry in its slot
  Given Tempo fails for GCM-2 on the first submit
  When I submit again
  Then only GCM-2 is sent, at 09:45, and Tempo has three worklogs in total

# AC-005.3
Scenario: A server error on create is never blindly re-sent
  Given Tempo stores a worklog but returns 502
  When the submit runs
  Then the worklog is not created a second time
```

## Tasks

| Id | Task | WP | Owner agent | Estimate (h) | Status |
|---|---|---|---|---|---|
| US-005-T1 | Tempo gateway and submit handler | Task 7-8 | go-coder | 4 | Done |
| US-005-T2 | Stop retrying non-idempotent POSTs on 5xx or a timeout (R3); reconciling on the next run is US-007 (K2) | - | go-coder | 3 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-005.1 | `TestSubmitStaggersStartTimes` | pass |
| AC-005.2 | `TestSubmitPartialFailureThenRetry` | pass |
| AC-005.3 | `TestCreateDoesNotResendWorklogOn5xx` | skipped (BUG H2; un-skip with US-005-T2) |
