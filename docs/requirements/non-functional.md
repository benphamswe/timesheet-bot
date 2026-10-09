# Non-functional requirements

| ID | Attribute | Requirement | How it is met |
|---|---|---|---|
| NFR-1 | Correctness | No worklog is ever written to Tempo twice for the same entry | Per-entry `tempo_worklog_id` saved after each call ([ADR-0002](../adr/0002-postgres-with-transactional-outbox.md)) |
| NFR-2 | Correctness | Every logged duration is a positive multiple of 15m; no day exceeds the cap | Kernel `Duration` constructor, aggregate invariants |
| NFR-3 | Reliability | A missed Friday run (laptop off) is caught up without user action | [planned: P2] hourly catch-up, idempotent propose |
| NFR-4 | Reliability | Transient API failures (429, 5xx) do not fail a run | `httpx.RetryTransport`, max 5 attempts, Retry-After honoured |
| NFR-5 | Consistency | State changes and their events are stored atomically | Outbox written in the save transaction |
| NFR-6 | Security | Secrets exist only as environment variables; never in files, logs, errors or the database | `config.Env`, compose pass-through, project deny rules |
| NFR-7 | Security | AI output and Slack content cannot bypass domain validation or write to external systems | [ADR-0004](../adr/0004-allocation-policy-fallback-chain.md), [threat model](../security/threat-model.md) |
| NFR-8 | Cost | One agent run per week stays within a configured token budget | [planned: P3] `agent.yaml` budget, `agent_runs` table |
| NFR-9 | Operability | Every write path can be previewed without side effects | `plan --dry-run` |
| NFR-10 | Portability | Runs as one container plus Postgres on a laptop (WSL Docker) | distroless image, embedded tzdata and migrations |
| NFR-11 | Maintainability | Domain code has no infrastructure dependencies | Package-by-context, [ADR-0001](../adr/0001-ddd-package-by-bounded-context.md) |
| NFR-12 | Scalability (owner scoping) | Every timesheet, entry, event and outbox row is keyed by its owner id, and no operation reads or changes another owner's data | [ADR-0006](../adr/0006-personal-tool-scale-ready-aws-later.md) seam 1. [built] timesheets keyed by `user_account_id`; [planned: P1] events, outbox rows and an owner-scoped replace (US-011) |
