---
id: US-023
title: Adapters are tested against recorded API responses
status: Active
parent: FE-005
points: 3
sprint: SP-01
github:
fr: [FR-ACT-1, FR-ACT-2, FR-SUB-1]
---

# US-023: Adapters are tested against recorded API responses

As the timesheet owner, I want each adapter tested against recorded Slack, Jira and Tempo responses, so that the tool keeps reading and writing these services correctly when its code changes.

Finding: test review (contract fixtures for Tempo, Jira and Slack). The current adapter tests use hand-written JSON.

## Acceptance criteria

```gherkin
# AC-023.1
Scenario: Every field the adapter uses is mapped
  Given recorded responses for Tempo create and list-by-user, Jira issue, and Slack search.messages over two pages
  And each fixture names its source document and date
  When the adapters decode them
  Then every field each adapter uses is mapped to the expected value

# AC-023.2
Scenario: A missing required field is an error
  Given a Tempo create response without tempoWorklogId
  When the adapter decodes it
  Then it returns an error instead of a zero worklog id

# AC-023.3
Scenario: Requests match the recorded request
  Given the recorded request for each fixture
  When the adapter sends the same call
  Then the method, path, query and body fields match
```

## Dependencies

- Fixtures are built from the vendors' published examples and schemas, verified through official-source-verifier; they hold no real account ids or tokens.

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
| US-023-T1 | Fixture files with source and date under each adapter's testdata | - | go-coder | 1.5 | To Do |
| US-023-T2 | Decode and request-shape tests per adapter | - | go-coder | 3 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-023.1 | planned | |
| AC-023.2 | planned | |
| AC-023.3 | planned | |
