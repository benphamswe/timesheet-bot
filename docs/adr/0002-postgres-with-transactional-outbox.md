# ADR-0002: Use Postgres with a transactional outbox and per-entry submit idempotency

- Status: Accepted
- Date: 2026-10-09

## Context

Approval and submission cross process and network boundaries: a Slack click [P2] must lead to exactly one set of Tempo worklogs even if the laptop sleeps, the process crashes, or Tempo fails part-way (NFR-1, NFR-5).

## Decision

- Postgres 17 is the source of truth (the user chose it over SQLite).
- `Repository.Save` performs, in one transaction: a version-checked update of `timesheets`, a rewrite of its entries, and inserts of drained domain events into `outbox`.
- `SubmitHandler` saves after every Tempo call, storing `tempo_worklog_id`; entries with an id are skipped on retry.
- An outbox relay [P2] delivers events at least once; every handler is idempotent.

## Alternatives considered

- SQLite: simpler to run, but the user preferred Postgres for later growth.
- Calling Slack/Tempo directly inside the command handler: loses events on crash.

## Consequences

- Positive: no lost or duplicated work; restart-safe.
- Negative: a Postgres container is required even for a single user.
