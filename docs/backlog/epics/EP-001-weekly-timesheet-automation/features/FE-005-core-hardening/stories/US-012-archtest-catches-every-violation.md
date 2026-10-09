---
id: US-012
title: Architecture tests catch every layer violation
status: Active
parent: FE-005
points: 3
sprint: SP-01
github:
fr: [NFR-11]
---

# US-012: Architecture tests catch every layer violation

As the timesheet owner, I want the architecture tests to fail on any forbidden import or wall-clock use, including in sub-packages, so that `make ci` catches layering mistakes before review has to.

Findings: M4 (no deny rules for `application` importing `config`, `httpx`, `net/http` or SDKs), N1 (the application rule matches only `internal/<ctx>/application`, not its sub-packages), N2 (a rule that matches no package passes silently; only the total is checked), N3 (the wall-clock check matches only the call `time.Now()`).

## Acceptance criteria

```gherkin
# AC-012.1
Scenario: A violation in a sub-package is caught
  Given a package internal/timesheet/application/submit that imports an infrastructure package
  When the architecture tests run
  Then they fail and name that package and the import

# AC-012.2
Scenario: Application code cannot reach config, transport or SDKs
  Given application code that imports internal/config, internal/shared/httpx, net/http or a third-party module
  When the architecture tests run
  Then they fail and name the import

# AC-012.3
Scenario: A rule that checks nothing fails
  Given a rule whose package pattern matches no package
  When the architecture tests run
  Then they fail and name that rule

# AC-012.4
Scenario: The wall clock cannot be referenced indirectly
  Given domain or application code that refers to time.Now without calling it, or through an import alias
  When the architecture tests run
  Then they fail and name the file and line

# AC-012.5
Scenario: The tree conforms
  Given the current code, after US-013 removes uuid from the application layer
  When the architecture tests run
  Then they pass
```

## Dependencies

- AC-012.5 needs US-013 (the application layer imports `github.com/google/uuid` today).
- US-013 and US-014 add their checks to the framework this story fixes.

## Plan

`docs/superpowers/plans/2026-10-09-sp01-hardening.md` (status: draft).

Draft plan coverage, read 2026-10-09: WP-3 (Task 3.1) covers M4 only. N1 (sub-packages), N2 (empty rules) and N3 (wall-clock regex) are not in the draft yet. WP-3's allowlist permits `github.com/google/uuid` in the application layer, which US-013 removes.

## Definition of Ready

The Definition of Ready and story points are team conventions, not Scrum Guide 2020 rules (see `docs/backlog/README.md`).

- [x] Story form, Gherkin AC with a failure path, at most 8 points, dependencies named, linked to its FR ids

## Tasks

Provisional. `technical-architect` aligns them with the plan's work packages.

| Id | Task | WP | Owner agent | Estimate (h) | Status |
|---|---|---|---|---|---|
| US-012-T1 | Sub-package patterns, application deny and allow rules, per-rule empty check | - | go-coder | 2 | To Do |
| US-012-T2 | Wall-clock check on the parsed syntax tree instead of a regex | - | go-coder | 2 | To Do |
| US-012-T3 | Violation fixtures under testdata for each rule | - | go-coder | 2 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-012.1 | planned | |
| AC-012.2 | planned | |
| AC-012.3 | planned | |
| AC-012.4 | planned | |
| AC-012.5 | planned | |
