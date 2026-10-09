---
id: US-019
title: The CLI and config fail once, clearly and safely
status: Active
parent: FE-005
points: 3
sprint: SP-01
github:
fr: [FR-TS-3, NFR-6]
---

# US-019: The CLI and config fail once, clearly and safely

As the timesheet owner, I want each error printed once, insecure base URLs refused, and each command to ask only for the secrets it uses, so that I can read failures at a glance, never send a token over plain HTTP, and preview a week without every token set.

Findings: best-practice review F7 (double error reporting), F15 and F16 (https-only base URLs), F18 (environment variables checked per command).

## Acceptance criteria

```gherkin
# AC-019.1
Scenario: An error is printed once
  Given a proposal exists for 2026-W41
  When I run plan for 2026-W41 without --force
  Then the error is printed exactly once on stderr
  And the exit code is 1

# AC-019.2
Scenario: A plain-HTTP base URL is refused
  Given the config sets the Tempo base URL to http://api.tempo.io/4
  When any command starts
  Then it fails with a config error naming tempo_base_url
  And no request is sent

# AC-019.3
Scenario: A command needs only its own secrets
  Given TEMPO_API_TOKEN is not set and every other variable is set
  When I run plan --dry-run for 2026-W41
  Then it completes

# AC-019.4
Scenario: A missing secret is named before any work
  Given TEMPO_API_TOKEN is not set
  When I run submit for 2026-W41
  Then it fails before any request with an error naming TEMPO_API_TOKEN
```

## Dependencies

- Assumption: tests reach their fake servers through adapters built directly, or through TLS test servers, so no plain-HTTP exception is needed in config.

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
| US-019-T1 | One error report path in main and cobra (F7) | - | go-coder | 1 | To Do |
| US-019-T2 | https-only validation for every base URL (F15, F16) | - | go-coder | 1 | To Do |
| US-019-T3 | Per-command environment requirements (F18) | - | go-coder | 2 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-019.1 | planned | |
| AC-019.2 | planned | |
| AC-019.3 | planned | |
| AC-019.4 | planned | |
