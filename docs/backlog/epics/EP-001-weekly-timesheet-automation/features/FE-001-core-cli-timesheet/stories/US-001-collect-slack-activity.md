---
id: US-001
title: Collect my week's Slack activity
status: Resolved
parent: FE-001
points: 3
sprint: SP-01
github:
fr: [FR-ACT-1, FR-ACT-2]
---

# US-001: Collect my week's Slack activity

As the timesheet owner, I want my own Slack messages for a week collected per working day, so that the timesheet reflects what I actually worked on.

## Acceptance criteria

```gherkin
# AC-001.1
Scenario: Messages are assigned to local working days across pages
  Given Slack search returns my messages for 2026-W41 over two result pages
  When the week's activity is collected in timezone Asia/Ho_Chi_Minh
  Then every page is read
  And a message sent Sunday 23:00 UTC counts for Monday
  And messages on Saturday are dropped

# AC-001.2
Scenario: Issue keys are extracted once per message
  Given a message "done GCM-12, reviewing HUB-3 and GCM-12 again"
  When issue keys are extracted
  Then the keys are GCM-12 and HUB-3 in that order
```

## Tasks

| Id | Task | WP | Owner agent | Estimate (h) | Status |
|---|---|---|---|---|---|
| US-001-T1 | Kernel week, day and issue-key value objects | Task 2 | go-coder | 2 | Done |
| US-001-T2 | Slack collector with pagination and local days | Task 5 | go-coder | 3 | Done |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-001.1 | `TestCollectPagesAndLocalDays` | pass |
| AC-001.2 | `TestExtractIssueKeys` | pass |
