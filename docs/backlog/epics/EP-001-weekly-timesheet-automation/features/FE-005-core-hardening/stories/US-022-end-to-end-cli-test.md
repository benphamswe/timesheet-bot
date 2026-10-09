---
id: US-022
title: An end-to-end test runs the whole week from the CLI
status: Active
parent: FE-005
points: 5
sprint: SP-01
github:
fr: [FR-ACT-1, FR-TS-1, FR-TS-3, FR-SUB-1, FR-SUB-2, NFR-1]
---

# US-022: An end-to-end test runs the whole week from the CLI

As the timesheet owner, I want one automated test that runs plan, approve and submit through the real CLI, so that a change that breaks the week from Slack to Tempo fails the build.

Finding: test review. Today each layer is tested alone; nothing runs the built commands end to end.

## Acceptance criteria

```gherkin
# AC-022.1
Scenario: A week goes from Slack to Tempo
  Given fake Slack, Jira and Tempo servers and an empty database
  When the CLI runs migrate up, then plan, approve and submit for 2026-W41
  Then Tempo holds one worklog per entry
  And the timesheet for 2026-W41 is submitted

# AC-022.2
Scenario: A second submit sends nothing
  Given the week of AC-022.1 is submitted
  When the CLI runs submit for 2026-W41 again
  Then Tempo receives no create request

# AC-022.3
Scenario: A server error mid-run does not duplicate
  Given Tempo stores the second worklog but answers 502
  When the CLI runs submit for 2026-W41 twice
  Then Tempo holds exactly one worklog per entry

# AC-022.4
Scenario: A rejected week can be planned again
  Given a proposal for 2026-W41
  When the CLI runs reject with reason "GCM-1 was 2h" and then plan --force
  Then a new proposal for 2026-W41 awaits approval
```

## Dependencies

- US-007 for AC-022.3.
- Runs against the compose database under the integration build tag.

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
| US-022-T1 | Fake Slack, Jira and Tempo servers driven by the US-023 fixtures | - | go-coder | 2 | To Do |
| US-022-T2 | CLI harness: build the binary, run commands, read exit codes and output | - | go-coder | 2 | To Do |
| US-022-T3 | Scenarios AC-022.1 to AC-022.4 | - | go-coder | 3 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-022.1 | planned | |
| AC-022.2 | planned | |
| AC-022.3 | planned | |
| AC-022.4 | planned | |
