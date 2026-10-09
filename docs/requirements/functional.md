# Functional requirements

Source: [design spec](../superpowers/specs/2026-10-09-timesheet-bot-design.md).

## Collection (ACT)

- **FR-ACT-1** [built] Collect the user's own Slack messages for an ISO week (Mon-Fri) via `search.messages`.
  - Acceptance: all result pages are read; messages are assigned to the calendar day in the configured timezone; weekend messages are dropped.
- **FR-ACT-2** [built] Extract Jira issue keys (`^[A-Z][A-Z0-9]+-\d+$`) from message text and drop keys Jira does not know.
- **FR-ACT-3** [planned: P3] Fetch full thread context on demand for the agent (`conversations.replies`).

## Allocation (ALC)

- **FR-ALC-1** [built] Rule policy: 45 minutes (configurable) per distinct issue key per day.
- **FR-ALC-2** [built] Keyless activity in a configured channel counts for that channel's default issue.
- **FR-ALC-3** [planned: P1, `scaleDay` is `TODO(owner)`, US-002-T2; `TestRulePolicyScalesOverCapDay` skipped until then] Days over the daily cap are scaled down to at most the cap in 15-minute steps without dropping entries.
- **FR-ALC-4** [planned: P3] Claude agent policy decides issues and hours from activity, using read-only tools.
- **FR-ALC-5** [built] Policies run as a fallback chain; the first proposal that passes domain validation wins and validation errors feed the next policy.

## Timesheet lifecycle (TS)

- **FR-TS-1** [built] One timesheet per user per ISO week; lifecycle draft, awaiting_approval, approved, submitted, or rejected.
- **FR-TS-2** [built; refusing a partly submitted week and an atomic replace: planned P1, US-006] `plan --force` replaces a not yet submitted timesheet; a submitted week cannot be replaced.
- **FR-TS-3** [built] Approve and reject from the CLI; reject accepts a reason.
- **FR-TS-4** [planned: P2] Approval DM in Slack with Approve and Reject buttons (Socket Mode) and a weekly table.
- **FR-TS-5** [planned: P2] Thread reply in the DM revises the proposal.

## Submission (SUB)

- **FR-SUB-1** [built] Approved entries are written to Tempo v4 as worklogs, start times stacked from 09:00 per day.
- **FR-SUB-2** [built; recovery after an unknown Tempo outcome: planned P1, US-007] Each successful worklog id is persisted before the next call; a retry sends only unsent entries and keeps their time slots.
- **FR-SUB-3** [planned: P2] After 3 failed attempts for an entry the user gets a DM listing the failures.

## Scheduling (SCH)

- **FR-SCH-1** [planned: P2] Propose automatically every Friday 16:00 local time.
- **FR-SCH-2** [planned: P2] Hourly catch-up proposes any past week without a timesheet (laptop was off).

## Learning (LRN)

- **FR-LRN-1** [planned: P4] Capture feedback from reject reasons, thread corrections and Tempo edits after submission.
- **FR-LRN-2** [planned: P4] Distill lessons (active immediately) and draft skills (inactive until approved in Slack).
