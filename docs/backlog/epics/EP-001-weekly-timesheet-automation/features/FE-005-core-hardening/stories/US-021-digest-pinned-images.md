---
id: US-021
title: Container images are pinned by digest
status: Active
parent: FE-005
points: 2
sprint: SP-01
github:
fr: [NFR-10]
---

# US-021: Container images are pinned by digest

As the timesheet owner, I want every container image pinned to an exact digest, so that a rebuild uses exactly the images that were reviewed and an upstream tag change cannot alter what runs on my laptop.

Findings: best-practice review F19 (`Dockerfile`: `golang:1.26.9-alpine` and `distroless/static-debian12:nonroot` by tag) and F20 (`docker-compose.yml`: `postgres:17-alpine` by tag).

## Acceptance criteria

```gherkin
# AC-021.1
Scenario: The Dockerfile pins every base image
  Given services/timesheet-core/Dockerfile
  When its FROM lines are read
  Then each names its image with a sha256 digest

# AC-021.2
Scenario: Compose pins every image
  Given docker-compose.yml
  When its image lines are read
  Then each names its image with a sha256 digest

# AC-021.3
Scenario: An unpinned image fails CI
  Given a FROM line or a compose image without a digest
  When CI runs
  Then it fails and names the file and line

# AC-021.4
Scenario: Pins stay current
  Given .github/dependabot.yml
  When it is read
  Then the docker ecosystem covers services/timesheet-core and the docker-compose ecosystem covers the repository root
```

## Dependencies

- AC-021.4 already holds today (checked 2026-10-09); it guards against removing the Dependabot entries.

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
| US-021-T1 | Pin digests in Dockerfile and compose (F19, F20) | - | go-coder | 0.5 | To Do |
| US-021-T2 | CI check for unpinned images | - | go-coder | 1 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-021.1 | planned | |
| AC-021.2 | planned | |
| AC-021.3 | planned | |
| AC-021.4 | planned | |
