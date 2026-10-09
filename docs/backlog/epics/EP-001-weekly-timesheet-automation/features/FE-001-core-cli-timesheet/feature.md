---
id: FE-001
title: Core CLI timesheet
status: Resolved
parent: EP-001
github:
fr: [FR-ACT-1, FR-ACT-2, FR-ALC-1, FR-ALC-2, FR-ALC-3, FR-ALC-5, FR-TS-1, FR-TS-2, FR-TS-3, FR-SUB-1, FR-SUB-2]
---

# FE-001: Core CLI timesheet

## Problem and value

Logging time in Tempo by hand every week is slow and easy to forget. This feature builds the week's timesheet from the owner's Slack messages and writes it to Tempo after approval, from the command line.

## Scope

- In scope: Slack collection, 45-minute rule allocation, Postgres storage, CLI plan, approve, reject and submit, Docker.
- Out of scope: Slack buttons, scheduling, AI allocation (FE-002, FE-003).

## Stories

| Id | Story | Points | Status | Sprint |
|---|---|---|---|---|
| US-001 | Collect my week's Slack activity | 3 | Resolved | SP-01 |
| US-002 | Allocate time with the 45-minute rule | 5 | Active (`scaleDay` pending owner) | SP-01 |
| US-003 | Propose and store the week's timesheet | 5 | Resolved | SP-01 |
| US-004 | Approve or reject a timesheet | 2 | Resolved | SP-01 |
| US-005 | Submit approved worklogs to Tempo exactly once | 5 | Active (retry R3 fix open) | SP-01 |

## Design links

- ADRs: 0001, 0002, 0004, 0005
- Plan: `docs/superpowers/plans/2026-10-09-phase1-core.md`
- Feature doc: `docs/features/FE-001-core-cli-timesheet.md` (to be written by tech-writer at Closed)
- Learning guide: `docs/learn/features/FE-001/guide.md` (to be written by learning-coach)

## Open items before Closed

- `scaleDay` (owner, US-002-T2); `TestRulePolicyScalesOverCapDay` is skipped with a `TODO(owner)` reason until then.
- Retry transport must not resend non-idempotent POSTs on 5xx (coding-standards R3): US-005-T2, AC-005.3.
- The review findings of 2026-10-09 are tracked in FE-005 (Core hardening), not here; all of FE-005 is in SP-01 (owner decision 2026-10-09: fix everything). FE-001 cannot be `Closed` until FE-005's US-006 (H1), US-007 (H2, reconcile on the next run) and US-015 (lint) are `Closed`.
- Lint: 68 found (capped; 88 uncapped) on 2026-10-09; all cleared 2026-10-09 (US-015).
