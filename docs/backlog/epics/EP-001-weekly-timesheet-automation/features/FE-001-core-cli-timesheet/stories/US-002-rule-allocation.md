---
id: US-002
title: Allocate time with the 45-minute rule
status: Active
parent: FE-001
points: 5
sprint: SP-01
github:
fr: [FR-ALC-1, FR-ALC-2, FR-ALC-3]
---

# US-002: Allocate time with the 45-minute rule

As the timesheet owner, I want each issue I touched on a day credited with 45 minutes, so that a reasonable timesheet exists even without AI.

## Acceptance criteria

```gherkin
# AC-002.1
Scenario: One entry per issue per day
  Given I mentioned GCM-12 twice and GCM-13 once on Monday
  When the rule policy allocates the week
  Then Monday has GCM-12 for 45m and GCM-13 for 45m

# AC-002.2
Scenario: Keyless messages use the channel default
  Given channel proj-hub defaults to HUB-1
  And I posted in proj-hub without an issue key on Monday
  When the rule policy allocates the week
  Then Monday has HUB-1 for 45m

# AC-002.3
Scenario: An over-cap day is scaled down without dropping entries
  Given I mentioned 13 issues on Monday and the daily cap is 8h
  When the rule policy allocates the week
  Then Monday totals at most 8h and at least 7h
  And every entry is a positive multiple of 15m
```

## Tasks

| Id | Task | WP | Owner agent | Estimate (h) | Status |
|---|---|---|---|---|---|
| US-002-T1 | Rule policy and proposal types | Task 4 | go-coder | 3 | Done |
| US-002-T2 | `scaleDay` scaling strategy | Task 4b | owner | 1 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-002.1 | `TestRulePolicyOneEntryPerKeyPerDay` | pass |
| AC-002.2 | `TestRulePolicyChannelDefault` | pass |
| AC-002.3 | `TestRulePolicyScalesOverCapDay` | skipped (owner task) |

`TestRulePolicyScalesOverCapDay` is skipped with `t.Skip("TODO(owner): ...")` until the owner implements `scaleDay` and removes the skip (owner decision, 2026-10-09). US-002-T2 stays the owner's task.
