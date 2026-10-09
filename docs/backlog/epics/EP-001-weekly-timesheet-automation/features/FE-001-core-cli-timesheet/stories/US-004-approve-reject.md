---
id: US-004
title: Approve or reject a timesheet
status: Resolved
parent: FE-001
points: 2
sprint: SP-01
github:
fr: [FR-TS-3]
---

# US-004: Approve or reject a timesheet

As the timesheet owner, I want to approve or reject the proposal with a reason, so that nothing reaches Tempo without my decision.

## Acceptance criteria

```gherkin
# AC-004.1
Scenario: Only a proposal awaiting approval can be approved
  Given a draft timesheet
  When I approve it
  Then it fails with an invalid transition error

# AC-004.2
Scenario: A rejection keeps the reason
  Given a proposal awaiting approval
  When I reject it with reason "GCM-1 was 2h"
  Then it is rejected and a TimesheetRejected event carries the reason
```

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-004.1 | `TestIllegalTransitions` | pass |
| AC-004.2 | `TestRejectRecordsReason` | pass |
