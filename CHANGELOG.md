# Changelog

All notable changes to this project are documented in this file. The format is based on [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- A `collector` compose service (profile `collect`) and `services/allocation-agent/Dockerfile` for the Slack activity collector; not usable for collection until the owner picks how the container gets a claude.ai login (ADR-0013).
- Collect the week's Slack activity per local working day (US-001).
- Rule-based allocation of 45 minutes per issue key per day, with channel defaults (US-002).
- Propose, store, approve and reject weekly timesheets from the CLI (US-003, US-004).
- Submit approved entries to Tempo with per-entry idempotency (US-005).
- Architecture fitness tests, GitHub Actions CI, Dependabot.
- Report mode (`allocation.mode: report`): each line of the timesheet summaries you write in Slack (report channel and self-DM) becomes one Mavenlink entry, without Jira or Tempo; exact duplicates are dropped and days over the cap are warned about, not scaled; migration `00003`. Report parsing is `TODO(owner)` (ADR-0012).
- `report.parser: llm`: use the `report_lines` an LLM in the allocation-agent collector parsed (activity file version 2), validated again in Go, instead of the hand-written parser. Version 1 files still work for `manual` (ADR-0012).
- Read the week's Slack messages from `data/activity/<ISO week>.json`, written by the allocation-agent collector, instead of Slack search (`activity.source: file`); no Slack app or `SLACK_USER_TOKEN` needed. Both sources share one message mapping (ADR-0011).
- Submit to Mavenlink (Kantata OX) as well as, or instead of, Tempo (`submit_targets`): time entries tagged with an external reference so retries never duplicate them, one timesheet submission per project, and bearer-token or browser-session-cookie auth; migration `00002` (ADR-0010). Mapping Jira issues to Mavenlink projects is `TODO(owner)`.

### Changed

- One `.env` per service instead of one at the repository root: `services/timesheet-core/.env` (Go CLI, `make run`, `make test-int`, compose `app`) and `services/allocation-agent/.env` (`make collect`). Compose loads each with `env_file`; tokens exported only in the shell reach a container with `docker compose run -e KEY` (ADR-0013).
- `MavenlinkMapping.Resolve` is implemented (owner's policy, ADR-0010): most specific rule wins, duplicate rules are a config error, unmapped entries fall back to `default_workspace_id` with a warning or fail `plan`; `plan` shows a `MAVENLINK` column.
- `MavenlinkMapping.Resolve` takes the report project as well as the issue key; mapping rules gain `project` (ADR-0012).
- `MarkEntrySubmitted` takes a submit target and no longer completes the timesheet; `CompleteSubmission` does, once every entry is in every target (ADR-0010).
- Every Go test moved to `services/timesheet-core/test/`, black-box only; `make cover` measures production code with `-coverpkg=./internal/...,./cmd/...` (ADR-0009).

### Fixed

- `make lint` passes with no findings, test files included (US-015).

### Security

- Compose no longer interpolates secrets into `environment:`, where an unset `${KEY:-}` would blank the `env_file` token. `collect` exits 2 at startup if `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN` or `CLAUDE_CODE_OAUTH_TOKEN` is set, naming the variables only (ADR-0013).
- Pin the Go toolchain to go1.26.9; govulncheck reports no vulnerabilities.
