---
id: EP-001
title: Weekly Tempo timesheet filled automatically from Slack
status: Active
github:
---

# EP-001: Weekly Tempo timesheet filled automatically from Slack

## Outcome

Every Friday the owner gets a proposed timesheet built from their Slack activity, approves it in one step, and it lands in Tempo without duplicates. Over time the proposals need fewer corrections.

## Success measures

- A normal week needs no manual Tempo entry.
- Zero duplicate worklogs and zero lost approved entries (target to be recorded in `docs/operations/slo.md`, not yet written).
- Corrections per week trend down after the learning feature (FE-004).

## Features

| Id | Feature | Status | Sprint(s) |
|---|---|---|---|
| FE-001 | Core CLI timesheet (Phase 1) | Resolved | SP-01 |
| FE-002 | Slack approval DM, outbox relay and Friday scheduler (Phase 2) | New | |
| FE-003 | Claude allocation agent with skills (Phase 3) | New | |
| FE-004 | Learning from owner feedback (Phase 4) | New | |
| FE-005 | Core hardening (review findings 2026-10-09) | Active | SP-01 |

## Out of scope

- Multiple users, a web UI, editing worklogs the bot did not create.
