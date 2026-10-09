---
id: US-015
title: Lint passes with no findings
status: Resolved
parent: FE-005
points: 3
sprint: SP-01
github:
fr: [NFR-11]
---

# US-015: Lint passes with no findings

As the timesheet owner, I want `make lint` to pass with no findings, so that `make ci` can gate every story's Definition of Done.

Finding: the first `golangci-lint` run on 2026-10-09 reported 68 issues (`revive` 50, `errcheck` 5, `gosec` 5, `staticcheck` 2, `usestdlibvars` 2, `gocritic` 1, `gofmt` 1, `predeclared` 1, `unparam` 1; `docs/commands/lint-and-format.md`). The `gocritic` finding is a real defect: `os.Exit` after `defer` in `main.go` skips the deferred cleanup. The draft plan's uncapped run (`--max-issues-per-linter=0 --max-same-issues=0`) shows 88 findings; the default caps hid 20.

State: 68 found (capped; 88 uncapped) on 2026-10-09; all cleared 2026-10-09. `golangci-lint` now reports 0 issues uncapped.

## Acceptance criteria

```gherkin
# AC-015.1
Scenario: The current tree is lint clean
  Given the code with every finding of the 2026-10-09 run fixed, test files included
  When golangci-lint runs with --max-issues-per-linter=0 and --max-same-issues=0
  Then it exits 0 with no findings

# AC-015.2
Scenario: A new undocumented export fails lint
  Given a new exported identifier without a doc comment
  When make lint runs
  Then it fails and names that identifier

# AC-015.3
Scenario: Every suppression is specific and explained
  Given the gosec G101 false positives on environment variable names are suppressed
  When make lint runs
  Then every nolint directive names its linter and gives a reason
  And a nolint directive without a linter name or a reason fails lint
```

## Dependencies

- Do this before US-016, so the renames start from a clean lint baseline.

## Plan

`docs/superpowers/plans/2026-10-09-sp01-hardening.md` (status: draft).

Draft plan coverage, read 2026-10-09: Lint batches in WP-1 to WP-7 (section 6, "Lint inventory").

## Definition of Ready

The Definition of Ready and story points are team conventions, not Scrum Guide 2020 rules (see `docs/backlog/README.md`).

- [x] Story form, Gherkin AC with a failure path, at most 8 points, dependencies named, linked to its FR ids

## Tasks

Provisional. `technical-architect` aligns them with the plan's work packages.

| Id | Task | WP | Owner agent | Estimate (h) | Status |
|---|---|---|---|---|---|
| US-015-T1 | Lint findings in non-test files | WP-1 to WP-7 | go-coder | 4 | Done |
| US-015-T2 | Lint findings in test files (9 left on 2026-10-09, all cleared the same day) | - | go-coder | 1 | Done |
| US-015-T3 | Explained, specific suppressions for gosec G101 false positives | WP-4 | go-coder | 0.5 | Done |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-015.1 | `golangci-lint run --max-issues-per-linter=0 --max-same-issues=0` (`make lint`): 0 issues | pass |
| AC-015.2 | planned | |
| AC-015.3 | planned | |
