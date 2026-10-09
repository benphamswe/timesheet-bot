---
id: FE-005
title: Core hardening
status: Active
parent: EP-001
github:
fr: [FR-ACT-1, FR-ACT-2, FR-TS-1, FR-TS-2, FR-TS-3, FR-SUB-1, FR-SUB-2, NFR-1, NFR-2, NFR-4, NFR-5, NFR-6, NFR-9, NFR-10, NFR-11, NFR-12]
---

# FE-005: Core hardening

## Problem and value

The Phase 1 CLI works end to end, but the reviews of 2026-10-09 found ways it can post the same work to Tempo twice (H1, H2) and accept entries that can never be valid (N4). They also found ways it can lose events (M5) or later mix up owners (M3). The architecture tests have gaps (M4, N1-N3, N5-N7), and there is a naming backlog (N-01 to N-11). Lint: 68 found (capped; 88 uncapped) on 2026-10-09; all cleared 2026-10-09. Later reviews added:
- the test review: end-to-end, contract, smoke, security, fuzz and coverage tests, and Q2;
- the best-practice review: retries, cancel safety, typed errors, database bounds, CLI and config, logging, and image pins (F1 to F20);
- the low findings L6 to L11.

This feature makes the core safe to trust with real Tempo data and keeps it safe as Phases 2 to 4 add code.

## Why a new feature instead of FE-001

- FE-001 is a shippable capability, "the core CLI timesheet", and it is already `Resolved`. Adding these stories would reopen it and blur what it delivered and what its release notes say.
- Most of this work is not FE-001's capability. Owner scoping (ADR-0006 seam 1) gets the outbox ready for FE-002. The architecture tests, the naming rules, the test gates and logging guard every later feature.
- A feature may span sprints; a story may not. FE-005 started with an SP-01 and SP-02 split.
- FE-001 keeps what finishes its own stories: US-005-T2 (no blind retry of a Tempo create on 5xx, AC-005.3) stays in US-005. US-007 covers the next run after an unknown outcome. FE-001 cannot be `Closed` until US-006, US-007 and US-015 are `Closed`.

## Owner decisions (2026-10-09)

Each of these is recorded as "owner decision 2026-10-09: fix everything".

- **SP-01 scope change:** approved. SP-01 first went to 31 of 20 points. It then took every FE-005 story, at 93 of 20 points, as an over-commitment the owner accepted. See `docs/backlog/sprints/SP-01.md`.
- **`migrations/` folder:** moves to `internal/migrations/` (US-016, N-07). This reverses the conformance review's "deliberately fine" note. `go:embed` only needs `embed.go` beside the SQL files.
- **Postgres duplicate:** a duplicate (SQLSTATE 23505) maps to `domain.ErrConcurrentModification`, the same error the memory store returns (US-010).
- **Ports (N-06), recommended default applied:** rename `IssueDirectory` to `IssueResolver`, and keep `WorklogGateway` as a documented domain-noun port (US-016).

## Scope

- In scope: H1, H2, N4, M5, L10, M3, M4, N1-N3, N5-N7, lint, N-01 to N-11, the test review (end-to-end CLI, contract fixtures, Docker smoke, NFR-6 security tests, fuzz, 85% coverage gate, Q2), the best-practice review (F1-F20 as grouped below), L6, L7, L8, L9, L11, and `domaintest.RunRepositoryContract`.
- Out of scope: the transport change that stops re-sending a create on 5xx (US-005-T2, FE-001). `scaleDay` stays the owner's task under US-002. No circuit breaker, broker or CQRS (ADR-0006).
- Assumptions:
  - Tempo v4 accepts no idempotency key, so recovery is a lookup (coding-standards K2; verified in the draft plan).
  - The best-practice review is not in the repository. Its F ids are grouped by the themes the coordinator gave.

## Stories

Story points are a team convention, not a Scrum Guide 2020 rule (see `docs/backlog/README.md`).

| Id | Story | Findings | Points | Status | Sprint |
|---|---|---|---|---|---|
| US-006 | Replace a week's proposal without losing submitted work | H1 | 3 | Active | SP-01 |
| US-007 | Recover a worklog whose Tempo outcome is unknown | H2 | 5 | Active | SP-01 |
| US-008 | Refuse timesheet entries and owners that cannot be valid | N4 | 2 | Active | SP-01 |
| US-009 | Keep events until the save is committed | M5 | 2 | Active | SP-01 |
| US-010 | Report conflicting saves the same way in every store | L10, repository contract | 2 | Active | SP-01 |
| US-011 | Tie every stored event and delete to its owner | M3 | 3 | Active | SP-01 |
| US-012 | Architecture tests catch every layer violation | M4, N1-N3 | 3 | Active | SP-01 |
| US-013 | Take ids and the current time from injected sources | N5, N6 | 2 | Active | SP-01 |
| US-014 | Only the stores use the storage-only timesheet methods | N7 | 2 | Active | SP-01 |
| US-015 | Lint passes with no findings | lint | 3 | Resolved | SP-01 |
| US-016 | Names say what the code holds and checks | N-01 to N-11 | 8 | Active | SP-01 |
| US-017 | API calls retry, stop and report safely | F1-F5, F11-F14 | 5 | Active | SP-01 |
| US-018 | Database calls are bounded, consistent and clearly reported | F6, F8, F9 | 3 | Active | SP-01 |
| US-019 | The CLI and config fail once, clearly and safely | F7, F15, F16, F18 | 3 | Active | SP-01 |
| US-020 | Every run leaves structured, secret-free logs | F10 | 3 | Active | SP-01 |
| US-021 | Container images are pinned by digest | F19, F20 | 2 | Active | SP-01 |
| US-022 | An end-to-end test runs the whole week from the CLI | test review | 5 | Active | SP-01 |
| US-023 | Adapters are tested against recorded API responses | test review | 3 | Active | SP-01 |
| US-024 | A smoke test proves the container starts and migrates | test review | 2 | Active | SP-01 |
| US-025 | Tests prove secrets never leak | test review, NFR-6 | 2 | Active | SP-01 |
| US-026 | Parsers survive any input | test review | 2 | Active | SP-01 |
| US-027 | Each package keeps at least 85% test coverage | test review | 2 | Active | SP-01 |
| US-028 | A submitted entry keeps its Tempo worklog id | test review Q2 | 1 | Active | SP-01 |
| US-029 | Clean up the low review findings | L6, L7, L8, L9, L11 | 5 | Active | SP-01 |

Total: 73 points, all in SP-01. Status `Active` means committed in a sprint (`docs/backlog/README.md`, "States"). It does not mean work has started. Only US-015 has work in progress.

## Alignment with the draft plan

The plan (`docs/superpowers/plans/2026-10-09-sp01-hardening.md`, status: draft) was written before these stories existed. It is not edited here; `technical-architect` reconciles it before `status: final`.

| Point | Plan says | Backlog says |
|---|---|---|
| Story mapping | H1, H2, M3, M5, L10, M4 and lint are tasks US-005-T2 to T9 (section 7) | US-005 keeps only T2 (WP-2). The rest maps to US-006, US-007, US-009, US-010, US-011, US-012 and US-015. Each story names its WPs. |
| H1 criterion | Proposed AC-005.4 (Q1) | AC-006.1 covers it; AC-005.4 is not needed |
| uuid in application | WP-3 allowlists `github.com/google/uuid` | US-013 (N5) removes it; the allowlist entry goes |
| Migration path | WP-7 writes `migrations/00002_outbox_owner.sql` | N-07 moves the folder; order the two, or write to `internal/migrations/` |
| Error name | `ErrEntrySubmitted` (for `BeginEntrySubmit`) | The test review names `ErrEntryAlreadySubmitted` (US-028); use one sentinel |
| `WorklogGateway` | Gains `ListByAuthor` | Kept as a domain-noun port (N-06), not renamed `WorklogCreator` |
| Deferred | L6, L8, L9, L11 deferred | In US-029, SP-01 (owner decision: fix everything) |
| Lint count | 88 uncapped | 68 found (capped; 88 uncapped) on 2026-10-09; all cleared 2026-10-09 |

## Design links

- ADRs: 0001, 0002, 0006
- Reviews: `docs/architecture/project-structure.md#conformance-review-2026-10-09`, `docs/commands/lint-and-format.md`; the test review and the best-practice review are not in the repository
- Plan: `docs/superpowers/plans/2026-10-09-sp01-hardening.md` (status: draft)
- Feature doc: `docs/features/FE-005-core-hardening.md` (written by tech-writer at Done)
- Learning guide: `docs/learn/features/FE-005/guide.md`

## Release notes

One paragraph for the CHANGELOG, written at Done.
