---
id: US-024
title: A smoke test proves the container starts and migrates
status: Active
parent: FE-005
points: 2
sprint: SP-01
github:
fr: [NFR-10]
---

# US-024: A smoke test proves the container starts and migrates

As the timesheet owner, I want CI to start the built image against a database and check it migrates and runs, so that a broken image never reaches my laptop.

Finding: test review (Docker smoke test).

## Acceptance criteria

```gherkin
# AC-024.1
Scenario: The image migrates an empty database
  Given the built image and the compose database
  When the container runs migrate up
  Then it exits 0
  And the schema is at the latest version

# AC-024.2
Scenario: The image runs as a non-root user
  Given the built image
  When the container runs --help
  Then it exits 0 as a non-root user

# AC-024.3
Scenario: A missing database URL fails clearly
  Given DATABASE_URL is not set
  When the container runs migrate up
  Then it exits non-zero with an error naming DATABASE_URL
```

## Dependencies

- US-021 (pinned images).

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
| US-024-T1 | Smoke script and make target; CI job | - | go-coder | 2 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-024.1 | planned | |
| AC-024.2 | planned | |
| AC-024.3 | planned | |
