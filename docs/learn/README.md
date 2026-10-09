# Learn

Ask anything about this codebase or the ideas behind it:

```text
/learn what is an aggregate and why does Timesheet have one?
/learn why do retries need jitter? --feature FE-001
```

`learning-coach` answers from this repository's code and verified official sources, then saves the answer here so the material grows with every question.

| Folder | Contents |
|---|---|
| `concepts/` | One page per concept, beginner level, with exercises and self-check questions |
| `features/<FE-id>/guide.md` | A guided tour of a feature's code path, written when the feature is delivered |
| `features/<FE-id>/qa.md` | Your questions about that feature and their answers |

## Index

### Learning path

Read the concept pages in this order. Each page lists what to read first, explains one idea using this repository's code, shows a real mistake found in review, and ends with self-check questions.

| # | Page | Read first | Review findings used |
|---|---|---|---|
| 1 | [Clean Architecture dependency rule and architecture fitness functions](concepts/clean-architecture-dependency-rule.md) | - | M4, N1, N2, N3, N5 |
| 2 | [Ports and adapters, and consumer-defined interfaces in Go](concepts/ports-and-adapters.md) | 1 | L6, L7 |
| 3 | [Aggregates, invariants, and how Go zero values can bypass a constructor](concepts/aggregates-invariants-and-zero-values.md) | 1 | N4, L9 |
| 4 | [Optimistic locking](concepts/optimistic-locking.md) | 3 | M5, L10, N7 |
| 5 | [Transactional outbox](concepts/transactional-outbox.md) | 3, 4 | M5 |
| 6 | [Idempotency and safe retries](concepts/idempotency-and-safe-retries.md) | 2 | H2 |

Finding ids refer to the 2026-10-09 conformance review in [project-structure.md](../architecture/project-structure.md#conformance-review-2026-10-09). The fixes are planned in the [SP-01 hardening plan](../superpowers/plans/2026-10-09-sp01-hardening.md) (`status: draft` on 2026-10-09). The pages describe the code as it was on that date.

### Concepts

- [aggregates-invariants-and-zero-values](concepts/aggregates-invariants-and-zero-values.md): DDD aggregates and invariants; Go zero values and composite literals (2026-10-09)
- [clean-architecture-dependency-rule](concepts/clean-architecture-dependency-rule.md): the dependency rule; `test/architecture` (archtest) as a fitness function (2026-10-09)
- [idempotency-and-safe-retries](concepts/idempotency-and-safe-retries.md): RFC 9110 idempotent methods; why the retrying transport can duplicate Tempo worklogs (2026-10-09)
- [optimistic-locking](concepts/optimistic-locking.md): version column, `SetVersion`, `ErrConcurrentModification`, PostgreSQL Read Committed (2026-10-09)
- [ports-and-adapters](concepts/ports-and-adapters.md): hexagonal architecture; Go interfaces belong to the consumer (2026-10-09)
- [transactional-outbox](concepts/transactional-outbox.md): `outbox` table, `Save` in one transaction, the planned relay (2026-10-09)

### Features

(Feature guides are added here when a feature is delivered.)
