---
name: review-checklist
description: timesheet-bot code review checklist with stable rule IDs - boundaries, invariants, idempotency, error handling, secrets, AI trust boundary, tests. Preloaded into ddd-reviewer.
user-invocable: false
---

# Review checklist

Go paths (`internal/`, `cmd/`, `migrations/`) are relative to `services/timesheet-core/`. Go tools run there, or from the repo root with `go -C services/timesheet-core`.

Cite findings by ID. Every finding needs a concrete failure scenario. Style preferences are not findings.

## B - Boundaries

- **B1** A `internal/*/domain` package imports another context's domain, infrastructure, an SDK or `database/sql`. Prove it with `go list -deps`.
- **B2** An application package imports `infrastructure/` or `interfaces/`.
- **B3** A concrete adapter is constructed outside `cmd/timesheet/wire.go`.
- **B4** A port is defined by its implementation instead of its consumer.

## I - Invariants and state

- **I1** A state change bypasses an aggregate method (direct field writes, a `Snapshot` edited and rehydrated outside a repository).
- **I2** A transition is missing its status check, or allows a path not in the lifecycle `draft -> awaiting_approval -> approved -> submitted`, `awaiting_approval -> rejected`.
- **I3** A value object is created without its constructor in non-test code (struct literal, `Must*` outside tests and constants).
- **I4** A transition does not record its event, or an event is recorded without a state change.
- **I5** `time.Now()` is called in domain or application code.

## X - External writes and persistence

- **X1** A retry can resend an already-successful external write (the external id is not persisted before the next call).
- **X2** Events and state are saved in separate transactions (the outbox is broken).
- **X3** The optimistic `version` check is missing on update, or the version is not bumped.
- **X4** An adapter uses a bare `http.Client` instead of `httpx.NewClient`, losing retry on 429.
- **X5** An API field name or type differs from the verified facts in `docs/api/external-integrations.md` (Tempo `issueId` int64, `tempoWorklogId`, Jira `id` string, Slack exclusive dates).

## E - Errors

- **E1** An error is swallowed, logged and dropped, or replaced with a zero value.
- **E2** An error is compared with `==` instead of `errors.Is`, or a sentinel is not wrapped with `%w`.
- **E3** A not-found result from an external API is not distinguished from other failures where the caller treats them differently.

## S - Secrets and trust

- **S1** A secret value can reach a log, an error message, stdout, a fixture or the database.
- **S2** A secret is read other than via `config.Env`, or is hard-coded.
- **S3** AI or Slack content reaches the aggregate or `agent/memory/` without passing domain validation, or a lesson is learned from raw Slack text instead of user feedback (memory poisoning).
- **S4** Agent tools include a write to Tempo, Slack or the workspace.

## T - Tests

- **T1** A changed behaviour has no test that would fail if the behaviour broke.
- **T2** A test depends on the wall clock, network, or test order.
- **T3** An integration test lacks the `//go:build integration` tag, or skips without a `TEST_DATABASE_URL` check.
