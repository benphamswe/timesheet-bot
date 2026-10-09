# Changelog

All notable changes to this project are documented in this file. The format is based on [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Collect the week's Slack activity per local working day (US-001).
- Rule-based allocation of 45 minutes per issue key per day, with channel defaults (US-002).
- Propose, store, approve and reject weekly timesheets from the CLI (US-003, US-004).
- Submit approved entries to Tempo with per-entry idempotency (US-005).
- Architecture fitness tests, GitHub Actions CI, Dependabot.
- Read the week's Slack messages from `data/activity/<ISO week>.json`, written by the allocation-agent collector, instead of Slack search (`activity.source: file`); no Slack app or `SLACK_USER_TOKEN` needed. Both sources share one message mapping (ADR-0011).
- Submit to Mavenlink (Kantata OX) as well as, or instead of, Tempo (`submit_targets`): time entries tagged with an external reference so retries never duplicate them, one timesheet submission per project, and bearer-token or browser-session-cookie auth; migration `00002` (ADR-0010). Mapping Jira issues to Mavenlink projects is `TODO(owner)`.

### Changed

- `MarkEntrySubmitted` takes a submit target and no longer completes the timesheet; `CompleteSubmission` does, once every entry is in every target (ADR-0010).
- Every Go test moved to `services/timesheet-core/test/`, black-box only; `make cover` measures production code with `-coverpkg=./internal/...,./cmd/...` (ADR-0009).

### Fixed

- `make lint` passes with no findings, test files included (US-015).

### Security

- Pin the Go toolchain to go1.26.9; govulncheck reports no vulnerabilities.
