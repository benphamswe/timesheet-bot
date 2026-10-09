---
status: draft
tech-review: pending
sprint: SP-01
stories: [US-005]
findings: [H1, H2, R3, M3, M4, M5, L7, L10, N1-N7, N-01, N-02, N-03, N-07, N-11, Q1, Q2, F1-F20, test-review 3-9]
---

# SP-01 Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Coders execute this plan only after its frontmatter says `status: final` (docs/development/conventions.md, "Plans").

**Goal:** Close the SP-01 open items so US-005 can be resolved. After this plan, `plan --force` never discards work already sent to Tempo, a Tempo create is never blindly re-sent or lost on Ctrl-C, persistence is owner-keyed and loses no events, the architecture tests close their gaps, the binary logs, and the CLI has an end-to-end test. US-002 is then blocked only by the owner's `scaleDay`.

**Architecture:** No new layers and no new modules. The domain holds the invariants: the discard guard, the submit attempt marker, the rejected-entry state, and events kept until commit. The application reconciles against Tempo before any re-post. The HTTP transport retries only safe cases. Postgres replaces a timesheet in one transaction. `cmd/timesheet` builds one `*slog.Logger` and injects it.

**Tech stack:** Unchanged. Go 1.26 (toolchain go1.26.9), pgx v5.11.0, goose v3.28.0, slack-go v0.30.1, cobra v1.10.2, golangci-lint v2.14.0, Postgres 17. `go.mod` does not change; `log/slog` and `math/rand/v2` are stdlib.

All Go paths are relative to `services/timesheet-core/`, and every command runs from that folder unless it starts with `../../`. Integration tests need `docker compose up -d db` from the repository root.

Commits: the repository owner commits. Each task ends with a verify checkpoint, not a commit step.

## Preconditions (stage 0, outside this plan)

- Lint cleared on 2026-10-09 outside this plan.
- The test-file rewrite by the test agent is merged. It adds a skipped domain test for Q2 and a skipped AC-005.3 test in the `tempo` package, and puts the owner-approved `t.Skip("TODO(owner): ...")` on `TestRulePolicyScalesOverCapDay`. The test-first steps below add cases on top of the rewritten files; they do not rewrite them.
- Stage 1 starts only when `go test ./...` and `golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 ./...` are both clean on `main`.

## Scope

| Finding | What | Work package |
|---|---|---|
| H1 | `plan --force` discards a partly submitted timesheet (`propose.go:59-64,104-109`); delete and re-save are two transactions (`postgres/repository.go:137-140`) | WP-1, WP-7 |
| N4 | `AddEntry` accepts a zero `Duration` and an empty `IssueKey`; `New` accepts an empty owner | WP-1 (with H1) |
| H2, R3 | Tempo creates are not idempotent (`httpx/retry.go:47,68-70`; `tempo/client.go:58,65`; `submit.go:48-67`) | WP-1 (domain), WP-2 (transport), WP-5 (submit), WP-6 (adapters) |
| Q2 | `MarkEntrySubmitted` overwrites an existing `TempoWorklogID` (`timesheet.go:167-179`) | WP-1, H2 domain task |
| Q1 | Skipped AC-005.3 test in `tempo` | WP-6 removes the skip |
| F1 | Save after a successful POST uses the cancelled ctx (`submit.go:48-68`) | WP-5 |
| F2, F3, F4, F13, F14 | Retry sleep ignores ctx; `Retry-After` not capped by deadline; no jitter, 5 attempts; HTTP-date `Retry-After`; retries any 5xx | WP-2 |
| F5, F11, F12 | Typed `*APIError`; permanent 4xx not re-posted; short `last_error`; bounded decode and drain | WP-1 (domain state), WP-5, WP-6 |
| M3 | Owner id missing from outbox and events; `Delete` not owner-scoped | WP-1 (port, event), WP-7 (migration, SQL) |
| M5 | Postgres `Save` drains events and bumps the version before commit (`repository.go:84,122,132`) | WP-1 (domain API), WP-7 |
| L10 | Memory and Postgres disagree on conflicts | WP-1 (`domaintest` contract), WP-7 (23505 mapping) |
| F6, F8, F9 | pgx errors wrapped with `%w`; no session timeouts; `Get` not in one snapshot | WP-7 |
| M4, N1, N2, N3, N7 | Archtest gaps, regexes, missing denies, vacuous rules, wall clock, repository-only methods, `*test` imports | WP-3 |
| N5 | `application` imports `google/uuid` | WP-1 (inject `NewID`), WP-8 (wire) |
| N6 | `cli/root.go:69` calls `time.Now()` | WP-4, WP-8 (wire) |
| F7, F15, F16, F18 | Duplicate failure output; second Ctrl-C; https base URLs; env vars by `Needs` | WP-4, WP-8 |
| F10 | No logging | WP-8 (after every WP that edits the same files) |
| F19, F20, test-review 6 | Digest pins, Dependabot, BuildKit cache, Docker smoke | WP-9 |
| L7 | Unused `activity/domain.Source` | WP-2 |
| N-01, N-11 | `domain.New` to `NewTimesheet`; `EventEntrySubmitFail` to `EventEntrySubmissionFailed` | WP-1 (it edits every caller already) |
| N-02 | Package `slack` to `slacksearch` | WP-2 (it owns the package), WP-8 (wire import) |
| N-03 | One `doc.go` for archtest | WP-3 |
| N-07 | `migrations/` to `internal/migrations/` | WP-7 |
| test-review 3 | E2E CLI test | WP-11 |
| test-review 4 | Integration: submit state, owner isolation, NotFound, Delete, outbox rollback, migrate down and up | WP-1 (`domaintest`), WP-7 |
| test-review 5 | Contract fixtures (Tempo must, Jira and Slack should) | WP-6 (Tempo, Jira), WP-2 (Slack) |
| test-review 7 | Secrets never in errors (NFR-6); Slack text fuzz (NFR-7) | WP-6, WP-7, WP-2 |
| test-review 8 | Fuzz targets; `make fuzz` reports what it ran | WP-10, WP-2, WP-4 |
| test-review 9 | Coverage gate at 85% per domain and application package | WP-11 (after coverage is reached) |
| F17, OTel note | YAML config deviation; observability readiness in `scalability.md` | WP-11 |

Out of scope:

- `scaleDay` in `internal/allocation/domain/rule.go` (`TODO(owner)`). No task edits `rule.go` or `rule_test.go`.
- L6, L8, L9, L11 (see "Deferred").
- N-09 and N-10 test renames: done by the test agent in the stage 0 rewrite.

---

## Verified facts

Verified on 2026-10-09 against primary sources by the technical architect directly. The `official-source-verifier` agent was not available in this session; re-run these claims through it before `status: final` (Q5).

| Claim | Status | Source |
|---|---|---|
| Tempo `POST /4/worklogs` (`createWorklog`) takes **no** idempotency key. It has no header parameters, and `WorklogInput` has only `attributes, authorAccountId, billableSeconds, bypassPeriodClosuresAndApprovals, description, issueId, remainingEstimateSeconds, startDate, startTime, timeSpentSeconds`. The string "idempoten" occurs nowhere in the spec. Documented responses: `200`, `400`. | VERIFIED (absent) | https://apidocs.tempo.io/#tag/Worklogs/operation/createWorklog ; spec https://apidocs.tempo.io/tempo-openapi.yaml (`info.version: 4.0`, Last-Modified 2026-10-07 22:26:27 GMT) |
| Tempo `GET /4/worklogs/user/{accountId}` (`getWorklogsByUser`): query `from`, `to` ("up to and including this date"), `updatedFrom`, `offset` (default 0), `limit` (default 50). Returns `PageableWorklog{metadata{count, limit, offset, next, previous}, results[], self}`. | VERIFIED | https://apidocs.tempo.io/#tag/Worklogs/operation/getWorklogsByUser |
| Tempo `Worklog`: `tempoWorklogId` (int64), `issue.id` (int64), `author.accountId`, `startDate` (date), `startTime` (string; example elsewhere in the spec `"16:00:00"`), `timeSpentSeconds`, `description`, `createdAt`, `updatedAt`. The spec has **no response examples** for `createWorklog` or `getWorklogsByUser`, so fixtures are built from these schemas. | VERIFIED | same spec, `components.schemas.Worklog`, `Issue`, `User`, `PageableWorklog` |
| Tempo `POST /4/worklogs/search` takes `WorklogSearchInput{authorIds, issueIds, projectIds, from, to, updatedFrom, orderBy}`. | VERIFIED | https://apidocs.tempo.io/#tag/Worklogs/operation/searchWorklogs |
| Tempo answers excess traffic with `429`; no `Retry-After` is documented. | VERIFIED | spec `info.description`, "Rate limiting" |
| RFC 9110 9.2.2: PUT, DELETE and the safe methods are idempotent. A client "SHOULD NOT automatically retry a request with a non-idempotent method unless it has some means to know that the request semantics are actually idempotent"; it "can repeat a POST request automatically if it knows ... that the request is safe for that resource". | VERIFIED | https://www.rfc-editor.org/rfc/rfc9110#section-9.2.2 |
| `ALTER TABLE ... ADD CONSTRAINT ... CHECK (...) NOT VALID` skips the scan of existing rows and is "still enforced against subsequent inserts or updates". `ADD COLUMN` without a default needs no table rewrite. | VERIFIED | https://www.postgresql.org/docs/17/sql-altertable.html |
| `statement_timeout` aborts a statement running longer than the value; `idle_in_transaction_session_timeout` terminates a session idle inside an open transaction. Both default to milliseconds when given without units. | VERIFIED | https://www.postgresql.org/docs/17/runtime-config-client.html |
| SQLSTATE `23505` is `unique_violation`. | VERIFIED | https://www.postgresql.org/docs/17/errcodes-appendix.html |
| pgx v5.11.0: `BeginFunc` commits only when `fn` returns nil and returns the commit error; `BeginTxFunc` takes `pgx.TxOptions{IsoLevel, AccessMode}` with `pgx.RepeatableRead` and `pgx.ReadOnly`; `pgxpool.Config.ConnConfig` carries `RuntimeParams map[string]string`; `pgconn.PgError.Code`. | VERIFIED | `go doc github.com/jackc/pgx/v5.BeginFunc`, `.BeginTxFunc`, `.TxOptions`, `go doc github.com/jackc/pgx/v5/pgxpool.Config`, `go doc github.com/jackc/pgx/v5/pgconn.Config` |
| slack-go v0.30.1 sends `search.messages` as POST (`search.go:117` `postMethod`, `misc.go:507` `http.MethodPost`), and has `OptionAPIURL(string)`. | VERIFIED (source) | module cache `github.com/slack-go/slack@v0.30.1`; `go doc github.com/slack-go/slack.OptionAPIURL` |
| Go 1.26 stdlib: `net/http.ParseTime`, `context.WithoutCancel`, `log/slog.DiscardHandler`. `go:embed` patterns "are interpreted relative to the package directory" and "may not contain '.' or '..'", so `migrations/` can move to `internal/migrations/` together with its `embed.go`. | VERIFIED | `go doc net/http.ParseTime`, `go doc context.WithoutCancel`, `go doc log/slog.DiscardHandler`, `go doc embed` |

**H2 decision:** Tempo offers no idempotency key. The plan follows coding-standards K2: persist an attempt marker before the POST (K1), treat 5xx and timeouts as indeterminate, and reconcile through `GET /4/worklogs/user/{accountId}` before posting again.

---

## 0. Technology decisions

No new module. These rows cover new uses of existing APIs, stdlib packages and protocol behaviour.

| Tech | Version | Purpose | Alternatives considered | P1..P12 evidence | Security | Networking | SRE | SA fit | Verdict |
|---|---|---|---|---|---|---|---|---|---|
| Tempo `GET /4/worklogs/user/{accountId}` | API v4 (spec 4.0, 2026-10-07) | Reconcile before re-posting | `POST /4/worklogs/search` (a POST, so never retried on 5xx); a work attribute marker (needs an admin-defined attribute); a `description` marker (visible to the user's manager); an idempotency key (not offered) | Same host and token as create (P6, P9). One GET per run, only when an entry has an attempt (P10). Paged by `offset`/`limit=50`, capped at 20 pages. | pending | pending | pending | pending | pending |
| Retry policy: RFC 9110 9.2.2 plus R2/R3 | RFC 9110 | Retry 429 for every method; retry 502, 503 and 504 only for idempotent methods or a ctx marked `WithReadOnly`; full jitter; 3 attempts; `Retry-After` in seconds or HTTP-date, capped by the ctx deadline | A second `http.Client` without retries for Tempo (two retry layers, against R4); honouring `Idempotency-Key` (no vendor here supports it) | stdlib only (`math/rand/v2`, `net/http.ParseTime`) | pending | pending | pending | pending | pending |
| `log/slog` JSON handler on stderr | Go 1.26 stdlib | F10: one logger built in `main` and injected | A third-party logger (adds a dependency, against P8) | stdlib; sloglint is already enabled (static messages, snake_case keys) | pending | pending | pending | pending | pending |
| Postgres `CHECK ... NOT VALID` on `outbox.user_account_id` | Postgres 17 | Owner on every new outbox row without failing on legacy rows | `SET NOT NULL` after backfill (fails on orphan rows); deleting orphans (destroys history) | ALTER TABLE docs | pending | pending | pending | pending | pending |
| Session `statement_timeout=5s`, `idle_in_transaction_session_timeout=10s` through `RuntimeParams` | Postgres 17, pgx v5.11.0 | F8: bound a stuck query or transaction | Per-query `context.WithTimeout` only (no server-side guard against an abandoned transaction) | runtime-config-client docs | pending | pending | pending | pending | pending |
| Image digests for `golang:1.26.9-alpine`, `gcr.io/distroless/static-debian12:nonroot`, `postgres:17` (compose) and `postgres:17-alpine` (CI), kept current by Dependabot | as resolved by `docker buildx imagetools inspect` | F19: reproducible and tamper-evident base images | Tags only (mutable) | Digests resolved read-only at execution time (Task 9.1) | pending | pending | pending | pending | pending |

---

## 1. Work packages and stages

Layer order: domain, then application, then infrastructure, then interfaces and wiring. Within a stage, WPs own disjoint files and run at the same time in separate worktrees.

| Stage | WP | Owns | Depends on |
|---|---|---|---|
| 1 | WP-1 Timesheet domain, ports, propose (H1, N4, H2 domain, Q2, M3, M5, L10 contract, N5, N-01, N-11) | `internal/timesheet/domain/**` (with new `domaintest/`); `internal/timesheet/infrastructure/memory/**`; `internal/timesheet/application/{ports.go,propose.go,handlers_test.go}` | stage 0 |
| 1 | WP-2 Transport and Slack (H2 transport, F2-F4, F13, F14, L7, N-02, Slack fixtures and fuzz) | `internal/shared/httpx/**`; `internal/activity/**` (renamed package; `activity/domain` deleted) | stage 0 |
| 1 | WP-4 CLI, main, config (N6, F7, F15, F16 config part, F18 `Needs`, config fuzz) | `internal/interfaces/cli/**`; `cmd/timesheet/main.go`; `internal/config/**` | stage 0 |
| 1 | WP-9 Container and CI (F19, F20, Docker smoke) | `Dockerfile`; `../../docker-compose.yml`; `../../.github/workflows/ci.yml`; `../../.github/dependabot.yml` | stage 0 |
| 1 | WP-10 Test infrastructure (fuzz reporting, kernel fuzz, smoke target) | `Makefile`; new `internal/shared/kernel/fuzz_test.go` | stage 0 |
| 2 | WP-3 Archtest (M4, N1, N2, N3, N7, `*test` rule, N-03) | `internal/archtest/**` | WP-1, WP-2, WP-4 |
| 2 | WP-5 Submit (H2 application, F1, F5, F11) | `internal/timesheet/application/{submit.go,transition.go}`, new `submit_test.go` | WP-1 |
| 2 | WP-6 Tempo and Jira adapters (H2 infrastructure, Q1, F5, F11, F12, NFR-6 tests, fixtures) | `internal/timesheet/infrastructure/{tempo,jira}/**` | WP-1, WP-2 |
| 2 | WP-7 Postgres, db, migrations (H1, M3, M5, L10, F6, F8, F9, N-07, integration) | `internal/timesheet/infrastructure/postgres/**`; `internal/platform/db/**`; `migrations/` moved to `internal/migrations/**` | WP-1 |
| 3 | WP-8 Composition root and observability (F10, F16 wiring, F18 wiring, N5, N6, N-02 wiring) | `cmd/timesheet/{main.go,wire.go}`; `internal/interfaces/cli/commands.go`; new `internal/config/secret.go`; logger fields in `httpx/retry.go`, `application/{propose,submit}.go`, `tempo/client.go` | WP-1..WP-7 |
| 4 | WP-11 E2E, coverage gate, whole-tree gate, docs, backlog | new `cmd/timesheet/e2e_test.go`; `Makefile` (`cover`); `docs/**` | WP-8, WP-9, WP-10 |

Dependency graph:

```text
WP-1  owns timesheet/domain (+domaintest), timesheet/infrastructure/memory, application/{ports,propose,handlers_test}.go; depends on stage 0
WP-2  owns shared/httpx, activity (slack -> slacksearch, domain deleted); depends on stage 0
WP-4  owns interfaces/cli, cmd/timesheet/main.go, config; depends on stage 0
WP-9  owns Dockerfile, docker-compose.yml, .github/workflows/ci.yml, .github/dependabot.yml; depends on stage 0
WP-10 owns Makefile (fuzz, smoke), shared/kernel/fuzz_test.go; depends on stage 0
WP-3  owns archtest; depends on WP-1, WP-2, WP-4
WP-5  owns application/{submit,transition,submit_test}.go; depends on WP-1
WP-6  owns timesheet/infrastructure/{tempo,jira}; depends on WP-1, WP-2
WP-7  owns timesheet/infrastructure/postgres, platform/db, internal/migrations; depends on WP-1
WP-8  owns cmd/timesheet/{main,wire}.go, cli/commands.go, config/secret.go, logger fields; depends on WP-1..WP-7
WP-11 owns cmd/timesheet/e2e_test.go, Makefile cover target, docs; depends on WP-8, WP-9, WP-10
```

**Build state between stages.** After stage 1, `tempo` (missing `ListByAuthor`), `postgres` (removed `PullEvents`, `SetVersion`, old `Delete`), `platform/db` (moved `migrations` import, fixed by WP-7) and `cmd/timesheet` (renamed Slack package, new fields) do not compile. This is expected: stage 1 and stage 2 WPs verify with package-scoped commands, and `go build ./...` is gated in WP-8 and WP-11. WP-1 keeps `internal/timesheet/application` compiling, because `tempo` and `jira` import it.

**Owner action outside the agents:** N-07 moves the migrations folder, so the path glob in `.claude/rules/migrations.md` (`services/timesheet-core/migrations/**`) must become `services/timesheet-core/internal/migrations/**`. `.claude/` is owner-maintained (Q6).

---

## 2. Contracts (WP-1 defines; later WPs code against them)

### `internal/timesheet/domain`

```go
// errors.go: added
var (
	// ErrInvalidTimesheet rejects NewTimesheet without an id, an owner or a positive daily cap (N4).
	ErrInvalidTimesheet = errors.New("timesheet needs an id, an owner and a positive daily cap")
	// ErrInvalidEntry rejects AddEntry without a valid issue key, a positive issue id or a positive duration (N4).
	ErrInvalidEntry = errors.New("entry needs a valid issue key, an issue id and a positive duration")
	// ErrHasSubmissions rejects Discard or Delete when an entry is sent or may have been sent to Tempo.
	ErrHasSubmissions = errors.New("timesheet has entries sent or being sent to Tempo")
	// ErrNotDiscarded rejects Repository.Replace when the previous timesheet was not discarded.
	ErrNotDiscarded = errors.New("timesheet must be discarded before it is replaced")
	// ErrReplaceMismatch rejects Repository.Replace across owners or weeks.
	ErrReplaceMismatch = errors.New("replacement timesheet has a different owner or week")
	// ErrEntryAlreadySubmitted rejects a submit attempt, or a different Tempo id, for an entry that already has one (Q2).
	ErrEntryAlreadySubmitted = errors.New("entry is already submitted")
)

// MaxReasonLen caps LastError and Event.Reason, so no response body reaches storage (F11).
const MaxReasonLen = 200

// timesheet.go
// NewTimesheet replaces New (N-01) and validates its input (N4).
func NewTimesheet(id, userAccountID string, week kernel.WeekID, dailyCap kernel.Duration, policy string, now time.Time) (*Timesheet, error)
// AddEntry also checks kernel.NewIssueKey(string(key)), issueID > 0 and d.Minutes() > 0, wrapping ErrInvalidEntry.
func (t *Timesheet) AddEntry(key kernel.IssueKey, issueID int64, day kernel.WorkDay, d kernel.Duration, description string) error

// Discard marks the timesheet for replacement and records TimesheetDiscarded.
// ErrHasSubmissions when the status is submitted, or an entry is submitted, or an
// entry has SubmitAttempts > 0 and is not SubmitRejected. A second call is a no-op.
func (t *Timesheet) Discard(reason string, now time.Time) error
func (t *Timesheet) Discarded() bool

// BeginEntrySubmit records an attempt before the call to Tempo (K1).
// Requires approved (ErrInvalidTransition), an unsubmitted entry (ErrEntryAlreadySubmitted),
// and an entry that is not SubmitRejected (ErrInvalidTransition).
func (t *Timesheet) BeginEntrySubmit(entryID string, now time.Time) error
// MarkEntrySubmitted (Q2 default): the same id again is a no-op returning nil; a different
// id on a submitted entry returns ErrEntryAlreadySubmitted and changes nothing.
func (t *Timesheet) MarkEntrySubmitted(entryID, tempoWorklogID string, now time.Time) error
// MarkEntryFailed no longer increments SubmitAttempts. It stores the cause truncated to
// MaxReasonLen and records EventEntrySubmissionFailed when SubmitAttempts == MaxSubmitAttempts.
func (t *Timesheet) MarkEntryFailed(entryID string, cause error, now time.Time) error
// MarkEntryRejected records a permanent refusal (F5): SubmitRejected = true, LastError = reason
// (truncated), and EventEntrySubmissionFailed. A rejected entry is never posted until reopened.
func (t *Timesheet) MarkEntryRejected(entryID, reason string, now time.Time) error
// ReopenRejected clears SubmitRejected on every entry (approved only) and returns how many it cleared.
func (t *Timesheet) ReopenRejected(now time.Time) (int, error)

// PendingEvents returns a copy of the events recorded since the last MarkPersisted.
func (t *Timesheet) PendingEvents() []kernel.DomainEvent
// MarkPersisted sets the stored version and clears the pending events; repositories call it after commit (M5).
func (t *Timesheet) MarkPersisted(version int)
// Removed: New, SetVersion, PullEvents.

// entry.go
type Entry struct {
	// existing fields ...
	SubmitRejected bool // permanent 4xx from Tempo; skipped until ReopenRejected
}
// NeedsReconcile reports an entry whose earlier attempt may have reached Tempo.
func (e Entry) NeedsReconcile() bool { return !e.Submitted() && e.SubmitAttempts > 0 && !e.SubmitRejected }

// events.go
const (
	EventDiscarded             = "TimesheetDiscarded"
	EventEntrySubmissionFailed = "EntrySubmissionFailed" // renamed constant (N-11); wire value unchanged
)
type Event struct {
	Name          string    `json:"name"`
	TimesheetID   string    `json:"timesheet_id"`
	UserAccountID string    `json:"user_account_id"` // M3, ADR-0006 seam 1
	Week          string    `json:"week"`
	Revision      int       `json:"revision"`
	EntryID       string    `json:"entry_id,omitempty"`
	Reason        string    `json:"reason,omitempty"` // truncated to MaxReasonLen
	At            time.Time `json:"at"`
}

// repository.go
type Repository interface {
	Get(ctx context.Context, userAccountID string, week kernel.WeekID) (*Timesheet, error)
	// Save: optimistic version check, entries rewritten, pending events to the outbox, one
	// transaction, then t.MarkPersisted. A second timesheet for the same owner and week gives
	// ErrConcurrentModification (L10).
	Save(ctx context.Context, t *Timesheet) error
	// Replace deletes previous (id, owner and version must match, else ErrConcurrentModification)
	// and inserts next (version 0), writing both aggregates' pending events, in one transaction.
	// previous must be Discarded (ErrNotDiscarded); owner and week must match (ErrReplaceMismatch).
	Replace(ctx context.Context, previous, next *Timesheet) error
	// Delete removes the owner's timesheet (M3). ErrNotFound when the owner has no timesheet
	// with that id; ErrHasSubmissions under the same rule as Discard.
	Delete(ctx context.Context, userAccountID, id string) error
}
```

### `internal/timesheet/domain/domaintest`

```go
// Package domaintest holds test helpers for the Timesheet domain. Only _test.go files may import it (archtest).
package domaintest

// RunRepositoryContract runs the behaviour every domain.Repository must share. Owner ids are
// random per subtest (crypto/rand), so runs against a persistent database never collide. It uses
// a fixed time (2026-10-09 16:00 UTC) and week 2026-W41; it never reads the wall clock.
func RunRepositoryContract(t *testing.T, newRepo func(*testing.T) domain.Repository)
```

Subtests: `GetNotFound`, `SaveGetRoundTrip`, `VersionConflict`, `DuplicateOwnerWeek` (L10), `OwnerIsolation` (owner B's `Get` of owner A's week gives `ErrNotFound`), `OwnerScopedDelete` (wrong owner gives `ErrNotFound` and the row stays; right owner deletes, then `Get` gives `ErrNotFound`), `DeleteRefusesSubmissions`, `SubmitStateRoundTrip` (`SubmitAttempts`, `LastError`, `TempoWorklogID` and `SubmitRejected` survive `Save` and `Get`), `ReplaceSwaps`, `ReplaceStaleWritesNothing`, `ReplaceNotDiscarded`, `ReplaceMismatch`, `EventsClearedAfterSave` (`PendingEvents()` empty and `Version()` bumped).

Imports: stdlib (`context`, `crypto/rand`, `encoding/hex`, `errors`, `testing`, `time`), `shared/kernel` and `timesheet/domain` only. This keeps it inside the domain rule (N1 matches `domain/` subpackages).

### `internal/timesheet/application/ports.go`

```go
var (
	// ErrCreateIndeterminate: outcome unknown (transport error, timeout, 408, 5xx, unreadable 2xx). Reconcile before posting again (K2).
	ErrCreateIndeterminate = errors.New("worklog create outcome unknown")
	// ErrCreateRejected: the system refused the worklog permanently (a 4xx other than 401, 403, 408 and 429). Not created; do not re-post (F5).
	ErrCreateRejected = errors.New("worklog create rejected")
	// ErrUnauthorized: credentials refused (401 or 403). Stop the run.
	ErrUnauthorized = errors.New("external system refused the credentials")
)

// RecordedWorklog is a worklog that already exists in Tempo.
type RecordedWorklog struct {
	ID string // tempoWorklogId
	Worklog
}

type WorklogGateway interface {
	Create(ctx context.Context, w Worklog) (worklogID string, err error)
	// ListByAuthor returns the author's weekday worklogs from..to inclusive, StartTime normalized
	// to HH:MM:SS; worklogs without a start time are omitted.
	ListByAuthor(ctx context.Context, authorAccountID string, from, to kernel.WorkDay) ([]RecordedWorklog, error)
}
```

`ProposeHandler` gains `NewID func() string` (N5); `uuid` leaves `application`.

### `internal/shared/httpx` (WP-2)

```go
// WithReadOnly marks requests made with ctx as safe to retry on 502/503/504 even when the method
// is not idempotent (RFC 9110 9.2.2). Only for calls that cannot change anything, such as Slack
// search.messages, which slack-go sends as POST.
func WithReadOnly(ctx context.Context) context.Context
func IsReadOnly(ctx context.Context) bool

type RetryTransport struct {
	Base        http.RoundTripper                           // defaults to http.DefaultTransport
	MaxAttempts int                                         // total attempts; default 3 (F4)
	BaseDelay   time.Duration                               // default 1s
	MaxDelay    time.Duration                               // default 30s
	Wait        func(ctx context.Context, d time.Duration) error // default: timer + select on ctx.Done (F2)
	Jitter      func(d time.Duration) time.Duration         // default: full jitter, rand.N(d+1) (F4)
	Now         func() time.Time                            // default time.Now; for HTTP-date Retry-After (F13)
	// Logger is added by WP-8 (F10).
}
```

Rule: retry `429` for every method; retry `502`, `503` and `504` (F14) only when the method is GET, HEAD, OPTIONS, TRACE, PUT, DELETE or empty, or `IsReadOnly(ctx)`; never retry a transport error. Wait is `Retry-After` (seconds, or HTTP-date through `http.ParseTime` minus `Now()`, F13) when present, else `Jitter(min(MaxDelay, BaseDelay*2^(attempt-1)))`. When the request ctx has a deadline and `wait` would pass it, return the 429/503 response as-is, without sleeping (F3).

---

## 3. Migrations (WP-7)

Files move to `internal/migrations/` (N-07), and `embed.go` moves with them. `00001_init.sql` moves byte-for-byte (goose tracks versions by number, not path).

`internal/migrations/00002_outbox_owner.sql`:

```sql
-- +goose Up
ALTER TABLE outbox ADD COLUMN user_account_id text;

UPDATE outbox o
SET user_account_id = t.user_account_id
FROM timesheets t
WHERE t.id = o.aggregate_id
  AND o.user_account_id IS NULL;

-- Rows whose timesheet an earlier `plan --force` deleted stay NULL. Every new row must carry the owner.
ALTER TABLE outbox
    ADD CONSTRAINT outbox_user_account_id_set CHECK (user_account_id IS NOT NULL) NOT VALID;

-- +goose Down
ALTER TABLE outbox DROP CONSTRAINT outbox_user_account_id_set;
ALTER TABLE outbox DROP COLUMN user_account_id;
```

`internal/migrations/00003_entry_submit_rejected.sql`:

```sql
-- +goose Up
ALTER TABLE timesheet_entries ADD COLUMN submit_rejected boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE timesheet_entries DROP COLUMN submit_rejected;
```

---

## 4. Tasks

### WP-1: Timesheet domain, ports, propose (stage 1)

Owner agent: go-coder.

#### Task 1.1: H1 and N4 domain invariants, contract helper (with N-01, N-11)

**Closes:** US-005, proposed AC-005.4 (Q1).

**Files:** `internal/timesheet/domain/{errors.go,events.go,entry.go,timesheet.go,repository.go,timesheet_test.go}`; new `internal/timesheet/domain/domaintest/contract.go`.

- [ ] **Test first** (`timesheet_test.go`, table-driven, `got X, want Y`):
  - `TestNewTimesheetValidates`: empty id, empty owner, and zero cap each give `ErrInvalidTimesheet`.
  - `TestAddEntryRejectsInvalid`: `kernel.Duration{}`, `IssueKey("")`, `IssueKey("lower-1")` and issue id 0 each give `ErrInvalidEntry`, and no entry is added.
  - `TestDiscardGuardsSubmissions`: awaiting, rejected, and approved with nothing sent give nil. Approved with a submitted entry, approved with an attempt, and submitted give `ErrHasSubmissions`. Approved where the only attempted entry is `SubmitRejected` gives nil. On success, `Discarded()` is true and one `TimesheetDiscarded` event carries the reason and owner.
  - `TestDiscardTwiceIsNoOp`.
  - `TestPendingEventsKeptUntilMarkPersisted`; `TestEventsCarryOwner`; `TestEventReasonTruncated` (a 1000-rune reason is stored at 200).
  - Replace `New(` with `NewTimesheet(` and `PullEvents()` with `PendingEvents()` plus `MarkPersisted` in the existing tests.
- [ ] Run `go test ./internal/timesheet/domain/...`. Expect FAIL.
- [ ] **Change:** section 2 for `NewTimesheet`, `AddEntry`, `Discard`, `Discarded`, `PendingEvents`, `MarkPersisted`, the event field and constants, `Repository`, and `domaintest.RunRepositoryContract` with every subtest listed in section 2. Remove `New`, `SetVersion` and `PullEvents`. Every new exported identifier has a doc comment.
- [ ] **Verify:** `go test ./internal/timesheet/domain/... && go list -deps ./internal/timesheet/domain/... | grep timesheet-bot && golangci-lint run ./internal/timesheet/domain/...`. The deps are only `timesheet/domain`, `timesheet/domain/domaintest` and `shared/kernel`.

#### Task 1.2: H2 domain: attempt marker, Q2, rejected state

**Closes:** US-005 AC-005.3.

**Files:** `internal/timesheet/domain/{timesheet.go,entry.go,timesheet_test.go}`.

- [ ] **Test first:**
  - Un-skip the Q2 test added in stage 0, and make it assert the Q2 default: the same id is a no-op; a different id gives `ErrEntryAlreadySubmitted` with the original id kept.
  - `TestBeginEntrySubmitCountsAttempt`: attempts go 0 to 1, and `NeedsReconcile()` is true. A submitted entry gives `ErrEntryAlreadySubmitted`; awaiting gives `ErrInvalidTransition`; an unknown id gives `ErrEntryNotFound`; a rejected entry gives `ErrInvalidTransition`.
  - `TestMarkEntryFailedEventAtMaxAttempts`: three Begin+Fail cycles give exactly one `EntrySubmissionFailed`.
  - `TestMarkEntryRejectedAndReopen`: rejected gives `NeedsReconcile()` false and one event; `ReopenRejected` returns 1 and clears the flag; status stays approved.
- [ ] Run, expect FAIL. **Change:** section 2. **Verify:** as in Task 1.1.

#### Task 1.3: Memory repository passes the contract

**Closes:** US-005, proposed AC-005.4 (L10 reference behaviour).

**Files:** `internal/timesheet/infrastructure/memory/repository.go`; new `internal/timesheet/infrastructure/memory/repository_test.go`.

- [ ] **Test first:** `TestRepositoryContract(t) { domaintest.RunRepositoryContract(t, func(*testing.T) domain.Repository { return NewRepository() }) }`, plus `TestReplaceAppendsBothEventStreams` (`repo.Events` ends with `TimesheetDiscarded`, then `TimesheetProposed`).
- [ ] **Change:** `Save` appends `PendingEvents()`, then calls `MarkPersisted`. `Replace` and owner-scoped `Delete` follow the section 2 contract under the one mutex. Keep `Events` for the existing tests.
- [ ] **Verify:** `go test ./internal/timesheet/infrastructure/memory/... && golangci-lint run ./internal/timesheet/infrastructure/memory/...`

#### Task 1.4: Ports, H1 in propose, `NewID` (N5)

**Closes:** US-005, proposed AC-005.4.

**Files:** `internal/timesheet/application/{ports.go,propose.go,handlers_test.go}`.

- [ ] **Test first** (`handlers_test.go`): `newPropose` sets `NewID` to a counter (`ts-1`, `ts-2`, ...).
  - `TestProposeForceRefusesPartlySubmitted`: one entry `BeginEntrySubmit` + `MarkEntrySubmitted`, saved; force gives `domain.ErrHasSubmissions`; the original is kept with its Tempo id; a counting source records 0 `Collect` calls.
  - `TestProposeForceRefusesAttemptInFlight`.
  - `TestProposeForceReplacesAtomically`: `TimesheetDiscarded` (reason `replaced by plan --force`), then `TimesheetProposed` for `ts-2`.
  - `TestProposeInvalidOwnerFailsFast`: `UserAccountID: ""` gives `domain.ErrInvalidTimesheet`, with no policy failures recorded.
  - Add `ListByAuthor` to `fakeGateway` (returns created worklogs) so the package compiles; WP-5 moves the fake.
- [ ] Run, expect FAIL.
- [ ] **Change:**
  - `ports.go`: section 2.
  - `propose.go`: in the force branch, `existing.Discard("replaced by plan --force", h.Clock.Now())` before `Collect`, returning `fmt.Errorf("%s: %w", cmd.Week, err)`. At the end, `Replace` when replacing, else `Save`. `build` uses `h.NewID()` and `domain.NewTimesheet`; when `errors.Is(err, domain.ErrInvalidTimesheet)`, `Handle` returns at once instead of trying the next policy. Remove the `uuid` import.
- [ ] **Verify:** `go test ./internal/timesheet/application/... && golangci-lint run ./internal/timesheet/application/... && go list -deps ./internal/timesheet/application | grep -c google/uuid` prints `0`.

### WP-2: Transport and Slack (stage 1)

Owner agent: go-coder.

#### Task 2.1: Retry policy (R3, F2, F3, F4, F13, F14)

**Closes:** US-005 AC-005.3.

**Files:** `internal/shared/httpx/{retry.go,retry_test.go}`.

- [ ] **Test first** (fake `Wait` and `Jitter` record their arguments; `Jitter` returns its input):
  - `TestRetryPolicyByMethodAndStatus`: a table of method by status (500, 502, 503, 504, 429) and calls with `MaxAttempts: 3`. GET 502/503/504 gives 3 calls; GET 500 gives 1; POST 502 gives 1 (response returned); POST 429 gives 3; PUT 503 gives 3; POST with `WithReadOnly` and 503 gives 3.
  - `TestRetryDefaultsToThreeAttempts`.
  - `TestRetryAfterSecondsAndHTTPDate`: `Retry-After: 7` waits 7s; `Retry-After: <Now()+5s in http.TimeFormat>` waits 5s.
  - `TestRetryAfterBeyondDeadlineReturnsResponse`: ctx deadline 2s, `Retry-After: 10`; one call, the 429 is returned and its body is readable, and `Wait` is never called.
  - `TestRetryStopsWhenContextCancelled`: the default `Wait` with a cancelled ctx returns `context.Canceled` without waiting.
  - `TestFullJitterBounds`: the default `Jitter` over 1000 draws stays within `[0, d]`.
  - `TestRetryPostOn429ResendsBody` (existing body-resend test, kept).
- [ ] Run, expect FAIL. **Change:** section 2. Drain discarded responses with `io.Copy(io.Discard, io.LimitReader(body, 64<<10))`, then `Close`.
- [ ] **Verify:** `go test -race ./internal/shared/httpx/... && golangci-lint run ./internal/shared/httpx/...`

#### Task 2.2: Slack opt-in, L7, N-02, fixtures, fuzz (NFR-7)

**Closes:** US-005 AC-005.3 (reads keep their retries while POST retries stop).

**Files:** move `internal/activity/infrastructure/slack/` to `internal/activity/infrastructure/slacksearch/` (package `slacksearch`); delete `internal/activity/domain/`; new `slacksearch/testdata/{search_messages_page1.json,search_messages_page2.json,SOURCES.md}`; new `slacksearch/fuzz_test.go`; new `slacksearch/contract_test.go`.

- [ ] **Test first:**
  - `TestCollectMarksSearchReadOnly`: the fake records `httpx.IsReadOnly(ctx)`; all calls are true.
  - `TestCollectParsesOfficialPayload` (`contract_test.go`): an `httptest` server on `/search.messages` serves the fixtures; the client is `slack.New("xoxp-test", slack.OptionAPIURL(srv.URL+"/"))`. Assert the day, channel, permalink and keys of each activity.
  - `FuzzParseTS`: never panics; on success, the result is within ±1s of the integer seconds part.
  - `FuzzToActivityText` (NFR-7): for arbitrary message text, every returned `IssueKey` passes `kernel.NewIssueKey`, and no panic.
- [ ] **Change:** at the top of `Collect`, add `ctx = httpx.WithReadOnly(ctx)`; remove `var _ domain.Source`; rename the package. `SOURCES.md` lists each fixture with its source URL (https://api.slack.com/methods/search.messages, the response example) and the date checked. The fixture is copied from that example, with only ids, text and timestamps changed to week 2026-W41.
- [ ] **Verify:** `go test ./internal/activity/... && go test -run '^$' -fuzz FuzzParseTS -fuzztime 10s ./internal/activity/infrastructure/slacksearch/ && test ! -d internal/activity/domain && golangci-lint run ./internal/activity/...`

### WP-4: CLI, main, config (stage 1)

Owner agent: go-coder.

#### Task 4.1: N6, F7, F18 `Needs`

**Closes:** US-005 (no AC; it enables the WP-11 E2E test, which covers AC-005.1 to AC-005.3 end to end).

**Files:** `internal/interfaces/cli/{root.go,commands.go}`, new `internal/interfaces/cli/cli_test.go`.

- [ ] **Test first:**
  - `TestResolveWeekUsesClock`: an empty `--week` with a fixed clock at 2026-10-09 gives `2026-W41`.
  - `TestPlanPrintsFailuresOnlyOnFallback`: the factory returns an App whose propose fails with `ErrNoValidProposal`; stderr has no `policy ... rejected` lines, and the returned error contains them. On a successful fallback, stderr has one line per failure.
  - `TestCommandsRequestOnlyWhatTheyNeed`: a recording factory. `plan` gives `Needs{DB: true, Activity: true}`; `plan --dry-run` gives `{Activity: true}`; `approve` and `reject` give `{DB: true}`; `submit` gives `{DB: true, Tempo: true}`.
- [ ] **Change:** `App.Clock kernel.Clock`; `resolveWeek(s string, clock kernel.Clock)`; `Needs.Activity bool`; the F7 print rule; no `time.Now` in the package.
- [ ] **Verify:** `go test ./internal/interfaces/cli/... && golangci-lint run ./internal/interfaces/cli/... && ! grep -rn 'time\.\(Now\|Since\|Until\)' internal/interfaces --include='*.go' | grep -v _test.go`

#### Task 4.2: F15 and F16 config, config fuzz

**Closes:** US-005 (no AC; F15, F16).

**Files:** `cmd/timesheet/main.go`; `internal/config/{config.go,config_test.go}`; new `internal/config/fuzz_test.go`.

- [ ] **Test first:**
  - `TestTempoBaseURLMustBeHTTPS`: `http://api.tempo.io/4`, `https://` (no host) and `ftp://x` each give an error naming `tempo_base_url`; empty (default) and `https://api.eu.tempo.io/4` pass.
  - `TestHTTPSURL`: the exported `config.HTTPSURL(name, raw string) (string, error)` used for `JIRA_BASE_URL` in WP-8. The error names the variable and never contains its value.
  - `FuzzLoad`: writes the fuzz bytes to `t.TempDir()/c.yaml`; `Load` never panics.
- [ ] **Change:** the validation in `resolve`. In `main.go`, after `signal.NotifyContext`, add `go func() { <-ctx.Done(); stop() }()`, so a second Ctrl-C ends the process.
- [ ] **Verify:** `go test ./internal/config/... && go test -run '^$' -fuzz FuzzLoad -fuzztime 10s ./internal/config/ && golangci-lint run ./internal/config/...`

### WP-9: Container and CI (stage 1)

Owner agent: devops-engineer.

#### Task 9.1: Digests, Dependabot, cache mounts, smoke in CI (F19, F20, test-review 6)

**Closes:** US-005 (no AC; NFR container hardening).

**Files:** `Dockerfile`, `../../docker-compose.yml`, `../../.github/workflows/ci.yml`, `../../.github/dependabot.yml`.

- [ ] Resolve digests read-only: `docker buildx imagetools inspect golang:1.26.9-alpine`, `docker buildx imagetools inspect gcr.io/distroless/static-debian12:nonroot`, `docker buildx imagetools inspect postgres:17`, `docker buildx imagetools inspect postgres:17-alpine`. Write each as `image:tag@sha256:<index digest>`, with a comment giving the resolve date.
- [ ] `dependabot.yml`: a `docker` ecosystem entry for `/services/timesheet-core` and `/`, plus `github-actions` (keep the existing entries).
- [ ] F20: `RUN --mount=type=cache,target=/go/pkg/mod go mod download` and `RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build ...`.
- [ ] `ci.yml` `build` job: after `docker build`, add `run: make smoke IMAGE=timesheet-bot:ci` (the target is added by WP-10).
- [ ] **Verify:** `docker build -t timesheet-bot:local .` succeeds once WP-8 has made the tree compile. Until then, check syntax with `docker buildx build --check .`.

### WP-10: Test infrastructure (stage 1)

Owner agent: test-engineer.

#### Task 10.1: `make fuzz` reports, kernel fuzz, `make smoke`

**Closes:** US-005 (no AC; test-review 6 and 8).

**Files:** `Makefile`; new `internal/shared/kernel/fuzz_test.go`.

- [ ] **Test first:** `FuzzParseWeekID` (round trip: `ParseWeekID(w.String()) == w` whenever parsing succeeds); `FuzzNewIssueKey` (accepted keys match `^[A-Z][A-Z0-9]+-\d+$`); `FuzzExtractIssueKeys` (every result passes `NewIssueKey`, with no duplicates).
- [ ] **Change:** the `fuzz` target counts targets, prints `fuzz: ran N targets`, and exits 1 when N is 0. New target `smoke` (variable `IMAGE ?= timesheet-bot:local`) that:
  1. runs `docker run --rm $(IMAGE) --help` and expects exit 0;
  2. checks that `docker inspect -f '{{.Config.User}}' $(IMAGE)` equals `nonroot:nonroot`;
  3. runs `docker run --rm -v $(CURDIR)/config/config.example.yaml:/app/config/config.yaml:ro $(IMAGE) approve --week 2026-W41`, expecting a non-zero exit, stderr containing `DATABASE_URL`, and stderr not containing `unknown time zone`. Config loading resolves `Asia/Ho_Chi_Minh` before the environment check, so this proves the embedded tzdata works.
  Add both targets to `.PHONY`.
- [ ] **Verify:** `make fuzz FUZZTIME=5s` prints a non-zero count. `make smoke` is verified in WP-11.

### WP-3: Archtest (stage 2, after WP-1, WP-2, WP-4)

Owner agent: go-coder.

#### Task 3.1: Rules, self-test, wall clock, repository-only methods, `*test` imports (M4, N1, N2, N3, N7, N-03)

**Closes:** US-005 (no AC; M4 and N1-N7 from the FE-001 open items).

**Files:** `internal/archtest/{archtest_test.go,read.go}`; new `internal/archtest/doc.go` (the only package comment, N-03).

- [ ] **Test first:** extract `violations(r rule, byDir map[string][]string) []string`, then add `TestRulesCatchKnownViolations`, table-driven over synthetic maps:
  - `internal/timesheet/application` importing `net/http`, `database/sql`, `os/exec`, `timesheet-bot/internal/config`, `timesheet-bot/internal/shared/httpx`, `github.com/google/uuid`, `github.com/jackc/pgx/v5`, or `timesheet-bot/internal/activity/application`: each violates.
  - `internal/timesheet/application/sub` (N1) importing `net/http` violates; `internal/b2b/domain` (N1, digits) importing `net/http` violates.
  - `internal/interfaces/cli` importing `timesheet-bot/internal/config` or `timesheet-bot/internal/platform/db` violates (N2).
  - `internal/timesheet/infrastructure/tempo` importing `timesheet-bot/internal/config`, `timesheet-bot/internal/platform/db` or `timesheet-bot/internal/activity/infrastructure/slacksearch` violates (N2).
  - `internal/platform/db` importing `timesheet-bot/internal/timesheet/domain` violates (N2).
  - Any non-test, non-helper package importing `timesheet-bot/internal/timesheet/domain/domaintest`, `net/http/httptest` or `testing` violates (the `*test` rule).
  - Allowed: application importing `context`, `log/slog`, `timesheet-bot/internal/allocation/domain` or `timesheet-bot/internal/shared/kernel`; `domaintest` importing `testing`.
  - `TestEveryRuleMatchesAPackage` (N3): each rule matches at least one real directory, otherwise it fails with the rule name.
- [ ] Run, expect FAIL.
- [ ] **Change:**
  - Domain `from`: `^internal/(shared/kernel|[a-z0-9]+/domain)(/.*)?$`. Application `from`: `^internal/[a-z0-9]+/application(/.*)?$`. Infrastructure `from`: `^internal/[a-z0-9]+/infrastructure(/.*)?$`.
  - Application `allow`: `^timesheet-bot/internal/(shared/kernel|[a-z0-9]+/domain(/.*)?)$`, own-context `application(/.*)?` via a layer-aware `ownContext(dir, imp string) bool` (domain dirs: own `domain/...`; application dirs: own `application/...` and own `domain/...`). Add a deny for `^(net(/|$)|database/|os/exec$)`.
  - Interfaces deny: `/infrastructure(/|$)`, `^timesheet-bot/internal/(platform|config)(/|$)`, `^timesheet-bot/cmd(/|$)`, `^(net(/|$)|database/)`.
  - Infrastructure deny: `^timesheet-bot/internal/(interfaces|config|platform)(/|$)`, `^timesheet-bot/cmd(/|$)`, and other contexts' `infrastructure` (computed from the dir's context).
  - Platform: `^internal/platform(/.*)?$` denies `^timesheet-bot/internal/[a-z0-9]+/(domain|application|infrastructure)(/|$)` and `^timesheet-bot/internal/interfaces(/|$)`.
  - `*test` rule: every non-test package whose directory base does not end in `test` denies imports whose last element ends in `test`, and `^testing(/|$)`.
  - Wall clock (N3): regex `\btime\.(Now|Since|Until)\b`, scope `^internal/[a-z0-9]+/(domain|application)(/.*)?/` and `^internal/interfaces/`; fail when the walker visited 0 files.
  - N7: parse non-test files fully (`go/parser`, `ast.Inspect`). A call whose selector is `Rehydrate`, `Snapshot`, `SetVersion`, `MarkPersisted` or `PendingEvents` is allowed only under `^internal/timesheet/(domain|infrastructure)(/.*)?$`.
- [ ] **Verify:** `make arch && golangci-lint run ./internal/archtest/...`

### WP-5: Submit (stage 2, after WP-1)

Owner agent: go-coder.

#### Task 5.1: Attempt marker, reconcile, Ctrl-C safety, permanent 4xx (H2, F1, F5, F11)

**Closes:** US-005 AC-005.3; keeps AC-005.1 and AC-005.2 green.

**Files:** `internal/timesheet/application/{submit.go,transition.go,handlers_test.go}`; new `submit_test.go`.

- [ ] **Test first:** move `fakeGateway`, `approvedTimesheet` and the two submit tests to `submit_test.go`. The fake gains `stored []RecordedWorklog`, `creates map[int64]int`, `failFor map[int64]error`, `storeThenFail map[int64]bool` (store, then return `fmt.Errorf("%w: tempo 502", ErrCreateIndeterminate)`), `listErr`, and `cancelAfterCreate context.CancelFunc`. Add `saveFailRepo` (fails the n-th `Save`) and `ctxCheckingRepo` (fails a `Save` whose ctx is already cancelled).
  - `TestSubmitServerErrorIsNotResent` (AC-005.3): run 1 gives `Indeterminate == 1` and `creates[102] == 1`; run 2 gives `creates[102] == 1`, `Reconciled == 1`, status submitted, and `len(stored) == 3`.
  - `TestSubmitCrashAfterTempoSuccessReconciles`: the second `Save` fails; run 2 makes no new POST.
  - `TestSubmitCtrlCAfterCreateStillSaves` (F1): the fake cancels the ctx right after storing GCM-1. With `ctxCheckingRepo`, the post-create save succeeds (it runs on `WithoutCancel`), `Handle` returns `context.Canceled` before the next `Create`, and the stored entry has its Tempo id.
  - `TestSubmitDoesNotPostWhenReconcileLookupFails`.
  - `TestSubmitRejectedIsNotReposted` (F5): `failFor{102: fmt.Errorf("%w: tempo 400 Bad Request: period closed", ErrCreateRejected)}`. Run 1 marks the entry rejected (`Rejected == 1`); run 2 makes no POST for 102 (`creates[102] == 1`), and the status stays approved. After `TransitionHandler.ReopenRejected`, run 3 posts it at `09:45:00`.
  - `TestSubmitUnauthorizedStopsRun`: the first create gives `ErrUnauthorized`; `Handle` returns it, and `creates` has 1 call in total.
  - `TestSubmitStoresShortReason` (F11): a 5000-byte error message is stored as `LastError` of at most 200 runes.
  - `TestSubmitSkipsLookupWhenNoAttempts`.
- [ ] Run, expect FAIL.
- [ ] **Change** in `submit.go`:
  - `SubmitResult` gains `Reconciled`, `Indeterminate` and `Rejected`.
  - Saves use `saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)` (F1).
  - Algorithm per run:
    1. Get; require approved.
    2. If any entry `NeedsReconcile()`, call `ListByAuthor` for Monday to Friday. An error ends the run before any POST.
    3. Walk `Entries()` with the existing slot offsets. Skip submitted entries. Skip `SubmitRejected` entries and count them in `Rejected`.
    4. Reconcile: match the first unclaimed `RecordedWorklog` with equal `IssueID`, `Day`, `StartTime`, `Seconds` and `AuthorAccountID`. On a match: `MarkEntrySubmitted`, save, continue.
    5. Before each POST, if `err := ctx.Err(); err != nil`, return it (F1). Then `BeginEntrySubmit`, then save.
    6. `Create(ctx, ...)`. On `ErrUnauthorized`: `MarkEntryFailed`, save, return the error. On `ErrCreateRejected`: `MarkEntryRejected(id, err.Error())`. On any other error: `MarkEntryFailed`, and count `Indeterminate` when it is `ErrCreateIndeterminate`. On success: `MarkEntrySubmitted`. Then save.
  - `transition.go`: `func (h *TransitionHandler) ReopenRejected(ctx context.Context, week kernel.WeekID) (int, error)`.
- [ ] **Verify:** `go test -race ./internal/timesheet/application/... && golangci-lint run ./internal/timesheet/application/...`

### WP-6: Tempo and Jira adapters (stage 2, after WP-1, WP-2)

Owner agent: go-coder.

#### Task 6.1: Tempo create classification, `ListByAuthor`, fixtures, Q1

**Closes:** US-005 AC-005.3.

**Files:** `internal/timesheet/infrastructure/tempo/{client.go,client_test.go}`; new `tempo/testdata/{create_worklog_200.json,worklogs_user_page1.json,worklogs_user_page2.json,error_400.json,SOURCES.md}`; new `tempo/contract_test.go`, `tempo/security_test.go`.

- [ ] **Fixtures (must):** built from the `Worklog`, `PageableWorklog` and `PageableMetadata` schemas at https://apidocs.tempo.io/tempo-openapi.yaml (the spec has no examples for these operations; see Verified facts). `SOURCES.md` lists, for each file, the operation (`#tag/Worklogs/operation/createWorklog`, `.../getWorklogsByUser`), the schema names, the spec Last-Modified date and the date checked.
- [ ] **Test first:**
  - Un-skip the stage 0 AC-005.3 test, and make it pass with `httpx.RetryTransport` in the client: the server stores the worklog and returns 502; expect `errors.Is(err, application.ErrCreateIndeterminate)` and exactly 1 POST.
  - `TestCreateClassification`, a table: transport error gives indeterminate; 408 gives indeterminate; 500, 502 and 503 give indeterminate; 200 with `{}` gives indeterminate; 401 and 403 give `ErrUnauthorized`; 400, 404 and 422 give `ErrCreateRejected`; 429 (after the transport's attempts) gives a plain `*APIError`, neither rejected nor indeterminate. Every error is an `*APIError` (via `errors.As`) with `StatusCode` and `Status` set, except transport errors.
  - `TestCreateParsesOfficialPayload` and `TestListByAuthorPaginates` (`contract_test.go`, fixtures): the path is `/4/worklogs/user/acc-1`, the query is `from=2026-10-05&to=2026-10-09&offset=0&limit=50` then `offset=50`, and 51 results are mapped.
  - `TestListByAuthorNormalizesAndSkips`: `9:45:00` becomes `09:45:00`; no `startTime` is omitted; Saturday is omitted. `TestListByAuthorEscapesAccountID`: `a/b` gives `a%2Fb`. `TestListByAuthorStopsAfter20Pages`.
  - `TestErrorsNeverContainSecrets` (`security_test.go`, NFR-6): token `tok-SECRET-123`. The server echoes the `Authorization` header and the token inside `errors[0].message`, for 400 and 500. `err.Error()` contains neither `tok-SECRET-123` nor `Bearer`.
  - `TestDecodeIsBounded` (F12): a 5 MB success body gives indeterminate, and the server sees the connection drained (the handler's write completes).
- [ ] **Change:** `type APIError struct{ StatusCode int; Status string; Reason string; kind error }` with `Error()` = `"tempo: " + Status + ": " + Reason` and `Unwrap() error { return kind }`. `Reason` is `errors[0].message` from at most 4 KB of body, with the token redacted and the text truncated to 200 runes (F11). Decode success bodies through `io.LimitReader(resp.Body, 1<<20)`, then drain to EOF (64 KB cap) and close (F12). `ListByAuthor`: `GET {base}/worklogs/user/{url.PathEscape(accountID)}?from=YYYY-MM-DD&to=YYYY-MM-DD&offset=N&limit=50`, decoding `{metadata:{next}, results:[{tempoWorklogId, issue:{id}, startDate, startTime, timeSpentSeconds, description, author:{accountId}}]}`. Stop when `metadata.next` is empty or a page is empty; after 20 pages return `tempo list worklogs: more than 1000 results`. Skip a result whose `startDate` fails `kernel.ParseWorkDay` with `kernel.ErrWeekend`, or whose `startTime` is empty. Normalize `startTime` with `time.Parse("15:04:05", s)` and re-format it as `"15:04:05"`. A non-2xx response is classified as for `Create`.
- [ ] **Verify:** `go test ./internal/timesheet/infrastructure/tempo/... && golangci-lint run ./internal/timesheet/infrastructure/tempo/...`. No `t.Skip` remains in the package (`! grep -rn 't.Skip' internal/timesheet/infrastructure/tempo`).

#### Task 6.2: Jira `*APIError`, F12, fixtures, NFR-6

**Closes:** US-005 (no AC; F5, F12, NFR-6).

**Files:** `internal/timesheet/infrastructure/jira/{client.go,client_test.go}`; new `jira/testdata/{issue_200.json,SOURCES.md}`, `jira/contract_test.go`, `jira/security_test.go`.

- [ ] **Fixtures (should):** `issue_200.json` follows the `GET /rest/api/3/issue/{issueIdOrKey}` response in the Jira Cloud platform REST v3 OpenAPI (https://developer.atlassian.com/cloud/jira/platform/swagger-v3.v3.json). `SOURCES.md` records the URL, the operation `getIssue` and the date checked.
- [ ] **Test first:** `TestResolveParsesOfficialPayload` (`id` is a string, parsed to int64); `TestErrorsAreAPIError` (500 gives `*APIError` with the status; 404 still gives `ErrIssueNotFound` via `Unwrap`; 401 gives `ErrUnauthorized`); `TestErrorsNeverContainSecrets` (the server echoes the Basic header and the token; the error contains neither the token, `Basic ` nor the base64 credentials).
- [ ] **Change:** as in Task 6.1. Remove the raw 512-byte body from the error.
- [ ] **Verify:** `go test ./internal/timesheet/infrastructure/jira/... && golangci-lint run ./internal/timesheet/infrastructure/jira/...`

### WP-7: Postgres, db, migrations (stage 2, after WP-1)

Owner agent: go-coder.

#### Task 7.1: Move migrations (N-07), add 00002 and 00003

**Closes:** US-005 (no AC; M3; supports F5 persistence).

**Files:** `git mv migrations internal/migrations` (the owner runs the `git` command, or the coder moves the files); `internal/migrations/{embed.go,00002_outbox_owner.sql,00003_entry_submit_rejected.sql}`; the import in `internal/platform/db/db.go`.

- [ ] **Test first:** `internal/platform/db/migrate_test.go` (`//go:build integration`), `TestMigrateDownUpOnFreshDatabase`. Using `TEST_DATABASE_URL`, run `CREATE DATABASE tsb_mig_<8 random hex>`; then `Migrate up` (3 versions), `down` 3 times, and `up`. Check `information_schema.columns` for `outbox.user_account_id` and `timesheet_entries.submit_rejected` after each `up`. Finally `DROP DATABASE`. It runs on its own database, so it cannot disturb the postgres package tests running in parallel.
- [ ] **Change:** section 3; `embed.go` keeps `//go:embed *.sql`, with a doc comment on `FS`.
- [ ] **Verify:** `TEST_DATABASE_URL=postgres://timesheet:timesheet@localhost:5432/timesheet?sslmode=disable go test -tags=integration ./internal/platform/db/...`

#### Task 7.2: Postgres repository (H1, M3, M5, L10, F6, F9)

**Closes:** US-005, proposed AC-005.4; keeps AC-005.2 durable.

**Files:** `internal/timesheet/infrastructure/postgres/repository.go`, `repository_test.go`; new `postgres/persistence_test.go` (integration tag).

- [ ] **Test first:**
  - `repository_test.go`: `TestRepositoryContract`, which calls `domaintest.RunRepositoryContract(t, func(t *testing.T) domain.Repository { return setup(t) })`.
  - `persistence_test.go`:
    - `TestOutboxRowsCarryOwner`: the `user_account_id` column and `payload->>'user_account_id'` equal the owner.
    - `TestReplaceWritesBothEventStreams`.
    - `TestReplaceInsertFailureRollsBackDelete`: next reuses the id of the owner's W42 timesheet; the error is returned, and A survives with no `TimesheetDiscarded` outbox row.
    - `TestSaveFailureKeepsEventsAndVersion` (M5): a duplicate owner and week gives `ErrConcurrentModification`, `Version() == 0`, one pending event, and no outbox rows.
    - `TestErrorsAreDomainSentinelsOnly` (F6): the conflict error is not `errors.As`-able to `*pgconn.PgError`.
    - `TestGetUsesOneSnapshot` (F9): with a stubbed hook, the transaction starts with `pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}`. Test it through an unexported `getTxOptions` variable that the test reads.
- [ ] Run `TEST_DATABASE_URL=... go test -tags=integration ./internal/timesheet/infrastructure/postgres/...`. Expect FAIL.
- [ ] **Change:**
  - Extract `writeTimesheet(ctx, tx, snapshot, next)` and `appendOutbox(ctx, tx, owner, events)`. The outbox insert carries `user_account_id`.
  - `Save`: `BeginFunc`, then `MarkPersisted(next)` only after commit.
  - `Replace` and `Delete` per section 2. `Delete` runs in one transaction: `SELECT 1 FROM timesheets WHERE id=$1 AND user_account_id=$2 FOR UPDATE` (no row gives `ErrNotFound`); `SELECT EXISTS (SELECT 1 FROM timesheet_entries WHERE timesheet_id=$1 AND (tempo_worklog_id IS NOT NULL OR (submit_attempts > 0 AND NOT submit_rejected)))` (true gives `ErrHasSubmissions`); then `DELETE`.
  - `Get`: `BeginTxFunc` with RepeatableRead and ReadOnly (F9). Reads and writes `submit_rejected`.
  - F6: map `23505` to `fmt.Errorf("%w: %s", domain.ErrConcurrentModification, week)`. Every other driver error becomes `fmt.Errorf("postgres %s: %v", op, err)`, with `%v` and never `%w`.
- [ ] **Verify:** `go vet -tags=integration ./internal/timesheet/infrastructure/postgres/... && TEST_DATABASE_URL=postgres://timesheet:timesheet@localhost:5432/timesheet?sslmode=disable go test -tags=integration ./internal/timesheet/infrastructure/postgres/... && golangci-lint run ./internal/timesheet/infrastructure/postgres/...`

#### Task 7.3: Session timeouts (F8) and DSN redaction (NFR-6)

**Closes:** US-005 (no AC; F8, NFR-6).

**Files:** `internal/platform/db/db.go`; new `internal/platform/db/db_test.go`.

- [ ] **Test first:**
  - `TestOpenErrorHidesPassword` (no database needed): `Open(ctx, "postgres://u:hunter2-SECRET@127.0.0.1:1/db?connect_timeout=1")` and `Open(ctx, "postgres://u:hunter2-SECRET@[bad")` both return errors without `hunter2-SECRET`. `Migrate` with the same DSNs behaves the same.
  - `TestRuntimeParams`: the exported `db.Config(dsn string) (*pgxpool.Config, error)` sets `statement_timeout` to `5000` and `idle_in_transaction_session_timeout` to `10000`.
  - Integration: `TestSessionTimeoutsApplied` checks that `SHOW statement_timeout` returns `5s`.
- [ ] **Change:** `Config` parses the DSN with `pgxpool.ParseConfig` and sets `cfg.ConnConfig.RuntimeParams`. On a parse error it returns `errors.New("DATABASE_URL is not a valid connection string")`, without the underlying text. `Open` uses `pgxpool.NewWithConfig`. Ping and Migrate errors pass through `redact(err, password)` (`strings.ReplaceAll` of the parsed password with `[redacted]`).
- [ ] **Verify:** `go test ./internal/platform/db/... && TEST_DATABASE_URL=... go test -tags=integration ./internal/platform/db/...`

### WP-8: Composition root and observability (stage 3, sequential)

Owner agent: go-coder. It runs after every WP that edits `wire.go`'s inputs, `retry.go`, `submit.go`, `propose.go` or `tempo/client.go`.

#### Task 8.1: Wiring (N5, N6, N-02, F16, F18) and logging (F10)

**Closes:** US-005 AC-005.3 operationally (the operator sees retries and final failures).

**Files:** `cmd/timesheet/{main.go,wire.go}`; `internal/interfaces/cli/commands.go` (submit output, `--retry-rejected`); new `internal/config/secret.go`; the `Logger *slog.Logger` field in `httpx.RetryTransport`, `application.ProposeHandler`, `application.SubmitHandler` and `tempo.NewClient(..., logger)`.

- [ ] **Test first:**
  - `TestSecretLogValueRedacts` (config): `slog` JSON output of `config.Secret("tok-SECRET")` is `"[redacted]"`, and `fmt.Sprint` gives `[redacted]`.
  - `TestRetryLogsWarn` (httpx): with a JSON handler into a buffer, one Warn record has message `http retry` and keys `method`, `host`, `status`, `attempt`, `wait_ms`, and no URL query or header.
  - `TestSubmitLogsFinalFailureAtError` (application): one Error record `tempo create failed` with `entry_id`, `outcome` (`indeterminate`, `rejected`, `unauthorized` or `failed`) and `status`.
  - `TestSubmitCommandOutput` (cli): `1 submitted, 1 reconciled, 1 rejected, 1 failed (1 unknown outcome, re-run submit to reconcile)`; `--retry-rejected` calls `ReopenRejected` before `Handle`.
- [ ] **Change:**
  - `main`: `logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))`; `cli.NewRoot(func(ctx context.Context, p string, n cli.Needs) (*cli.App, error) { return buildApp(ctx, p, n, logger) }, migrate)`.
  - `wire.go`:
    - Required env from `Needs` (F18): `Activity` needs Slack and the three Jira variables; `DB` needs `DATABASE_URL`; `Tempo` needs `TEMPO_API_TOKEN`.
    - `config.HTTPSURL(config.EnvJiraBaseURL, ...)` (F16).
    - `NewID: uuid.NewString` (N5); `app.Clock = clock` (N6); the import is `slacksearch` (N-02).
    - The Propose handler is built only when `needs.Activity` is set.
    - `config.Env` returns `map[string]config.Secret`, and secrets are revealed only at the constructor call sites.
  - Nil loggers default to `slog.New(slog.DiscardHandler)`.
- [ ] **Verify:** `go build ./... && go test ./... && golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 ./... && make arch`

### WP-11: E2E, coverage gate, whole-tree gate, docs, backlog (stage 4, sequential)

#### Task 11.1: E2E CLI test (test-review 3)

Owner agent: test-engineer. **Closes:** US-005 AC-005.1, AC-005.2 and AC-005.3 end to end; proposed AC-005.4.

**Files:** new `cmd/timesheet/e2e_test.go` (package `main`).

- [ ] Use one shared `memory.Repository` and fixed clock 2026-10-09 (W41). The httptest servers are:
  - Slack `/search.messages`, serving the WP-2 fixtures;
  - Jira `/rest/api/3/issue/{key}`, serving the WP-6 fixture with ids 101 and 102;
  - a stateful Tempo fake (`POST /4/worklogs`, `GET /4/worklogs/user/{id}`), which can return 502 after storing.
  
  The factory builds the App from the real adapters pointed at those servers (`slack.OptionAPIURL`, `jira.NewClient(srv.URL, ...)`, `tempo.NewClient(srv.URL+"/4", ...)`), with `httpx.RetryTransport` using a no-op `Wait`. Commands run through `cli.NewRoot(factory, nil)` with `SetArgs`, `SetOut`, `SetErr` and `ExecuteContext`.
- [ ] Cases:
  - `plan --dry-run` prints the table, and the repo is empty.
  - `plan` saves; `plan` again fails with `already exists`.
  - `approve` succeeds.
  - `submit`, with Tempo returning 502-after-store for issue 102, reports `1 unknown outcome`. A second `submit` gives Tempo 3 worklogs and status submitted, with starts `09:00:00`, `09:45:00` and `09:00:00`.
  - `plan --force` after submit fails.
  - `--week 2026-W40` touches only W40.
  - `plan --force` on an approved, unsent W40 replaces it.
- [ ] **Verify:** `go test -race ./cmd/timesheet/...`

#### Task 11.2: Coverage gate (test-review 9) and whole-tree gate

Owner agent: test-engineer. **Closes:** US-005 (no AC; quality gate).

**Files:** `Makefile` (`cover`).

- [ ] **Change:** `cover` runs `go test -coverprofile` per package matching `./internal/*/domain/...` and `./internal/*/application/...`, prints each package's percentage, and exits 1 when any is below 85.0. It excludes `domaintest`, which has no tests of its own. CI already runs `make cover`.
- [ ] **Verify, in order:**
  - `make ci`
  - `make cover` (every domain and application package at 85% or more)
  - `make arch`
  - `docker compose up -d db` (repo root), then `make test-int`
  - `make fuzz FUZZTIME=5s`
  - `make build && docker build -t timesheet-bot:local . && make smoke`
  - `golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 ./...` prints `0 issues.`

#### Task 11.3: Docs

Owner agent: tech-writer, plus standards-curator for `sources.md`. **Closes:** FE-001 "Open items before Closed".

- [ ] `docs/architecture/code-map.md`: the Submit flow (attempt marker, reconcile, Ctrl-C safety, rejected entries and `--retry-rejected`) and the Persistence flow (`Replace`, owner-scoped `Delete`, events after commit, REPEATABLE READ `Get`). Remove "does not yet guarantee exactly-once".
- [ ] `docs/api/external-integrations.md`: Tempo `GET /4/worklogs/user/{accountId}` as `[built]`, "no idempotency key on `POST /worklogs` (verified 2026-10-09)", the status classification table, and the fixture locations.
- [ ] `docs/architecture/network.md`: the new retry rule, `WithReadOnly`, jitter, the deadline cap and the Postgres session timeouts.
- [ ] `docs/architecture/project-structure.md`: `internal/migrations/`, `slacksearch/`, `domaintest/` and `doc.go`; `activity/domain/` removed. Correct the "`migrations/` at the root (required by `go:embed`)" sentence (N-07). Mark the resolved findings.
- [ ] `docs/architecture/scalability.md`: add "Observability readiness `[planned: P2]`": slog today; an OpenTelemetry traces and metrics exporter behind the logger seam when the service gets a scheduler.
- [ ] `docs/development/conventions.md` and `.claude/skills/coding-standards/sources.md` (standards-curator): record the YAML non-secret config file as a documented deviation (F17).
- [ ] `docs/commands/build-and-test.md` and `docs/commands/docker-and-runtime.md`: `make fuzz` (now reports its count), `make cover` (85% gate), `make smoke`, `timesheet submit --retry-rejected`, and the `docker buildx imagetools inspect` digest refresh (CLAUDE.md: new commands are documented in the same change).
- [ ] ADR-0002 is accepted and is not edited. If the reviewers want `Replace` and the rejected state recorded, write a new ADR.

#### Task 11.4: Backlog

Owner agent: backlog-manager. After `status: final`, write section 7 into the story task tables, add AC-005.4 if Q1 is approved, fill the traceability tables, and update SP-01 progress and the FE-001 open items. Leave US-002 Active: AC-002.3 waits for the owner's `scaleDay`.

---

## 5. Test plan and AC traceability

| AC | Tests | Tag |
|---|---|---|
| AC-002.1, AC-002.2 | unchanged (`allocation/domain`) | - |
| AC-002.3 | owner task, out of scope | - |
| AC-005.1 | `TestSubmitStaggersStartTimes`, E2E `submit` starts | - |
| AC-005.2 | `TestSubmitPartialFailureThenRetry`, `TestSubmitRejectedIsNotReposted` (reopen path), contract `SubmitStateRoundTrip` | contract on Postgres: `integration` |
| AC-005.3 | `TestSubmitServerErrorIsNotResent`, `TestSubmitCrashAfterTempoSuccessReconciles`, `TestSubmitCtrlCAfterCreateStillSaves`, `TestSubmitDoesNotPostWhenReconcileLookupFails`, the un-skipped tempo AC-005.3 test, `TestCreateClassification`, `TestRetryPolicyByMethodAndStatus`, E2E 502-after-store | - |
| AC-005.4 (proposed) | `TestDiscardGuardsSubmissions`, `TestProposeForceRefusesPartlySubmitted`, `TestProposeForceRefusesAttemptInFlight`, `TestProposeForceReplacesAtomically`, contract `Replace*`, `TestReplaceInsertFailureRollsBackDelete`, E2E force cases | Postgres: `integration` |
| M3 | `TestEventsCarryOwner`, contract `OwnerScopedDelete`, `OwnerIsolation`, `TestOutboxRowsCarryOwner` | Postgres: `integration` |
| M5 | `TestPendingEventsKeptUntilMarkPersisted`, `TestSaveFailureKeepsEventsAndVersion` | Postgres: `integration` |
| L10 | contract `DuplicateOwnerWeek` on memory and Postgres | Postgres: `integration` |
| N4 | `TestNewTimesheetValidates`, `TestAddEntryRejectsInvalid`, `TestProposeInvalidOwnerFailsFast` | - |
| Q2 | the un-skipped stage 0 domain test | - |
| M4, N1-N3, N7 | `TestRulesCatchKnownViolations`, `TestEveryRuleMatchesAPackage`, `TestDependencyRules`, `TestDomainHasNoWallClock` | - |
| NFR-6 | the tempo, jira and db `*ErrorsNeverContainSecrets`/`HidesPassword` tests, `TestSecretLogValueRedacts` | - |
| NFR-7 | `FuzzToActivityText`, `FuzzExtractIssueKeys` | fuzz |

Proposed AC-005.4 (Q1):

```gherkin
# AC-005.4
Scenario: Re-planning never discards work already sent to Tempo
  Given an approved timesheet with GCM-1 already sent to Tempo
  When I run plan --force
  Then the command fails and the timesheet and its Tempo worklog id are kept
```

## 6. Lint

Lint cleared on 2026-10-09 outside this plan.

## 7. Backlog tasks

All rows go to US-005. This plan adds nothing to US-002, which waits for the owner's `scaleDay`.

| Id | Task | WP | Owner agent | Estimate (h) | Status |
|---|---|---|---|---|---|
| US-005-T2 | Retry policy (R3, F2-F4, F13, F14); Slack opt-in, rename, fixtures, fuzz; delete `activity/domain` | WP-2 | go-coder | 3 | To Do |
| US-005-T3 | Domain: H1, N4, H2 attempt marker, Q2, rejected state, events until commit, owner in events; `domaintest` contract; memory repo; propose with `NewID` | WP-1 | go-coder | 6 | To Do |
| US-005-T4 | Submit: reconcile, Ctrl-C safety, permanent 4xx, short reasons, `ReopenRejected` | WP-5 | go-coder | 3.5 | To Do |
| US-005-T5 | Tempo and Jira: `*APIError`, classification, `ListByAuthor`, bounded decode, fixtures, NFR-6 tests, Q1 un-skip | WP-6 | go-coder | 4 | To Do |
| US-005-T6 | Postgres and db: migrations move, 00002, 00003, `Replace`, owner-scoped `Delete`, `MarkPersisted` after commit, 23505, REPEATABLE READ `Get`, session timeouts, DSN redaction, integration tests | WP-7 | go-coder | 5 | To Do |
| US-005-T7 | Archtest: M4, N1, N2, N3, N7, `*test` rule, `doc.go` | WP-3 | go-coder | 2.5 | To Do |
| US-005-T8 | CLI clock, failure output, `Needs`, second Ctrl-C, https URLs, config fuzz | WP-4 | go-coder | 2 | To Do |
| US-005-T9 | Container digests, Dependabot, cache mounts, CI smoke | WP-9 | devops-engineer | 1.5 | To Do |
| US-005-T10 | `make fuzz` reporting, kernel fuzz, `make smoke` | WP-10 | test-engineer | 1 | To Do |
| US-005-T11 | Composition root, slog logging, `Secret` redaction, submit output and `--retry-rejected` | WP-8 | go-coder | 3 | To Do |
| US-005-T12 | E2E CLI test, coverage gate, whole-tree gate | WP-11 | test-engineer | 3 | To Do |
| US-005-T13 | Docs and backlog updates | WP-11 | tech-writer, standards-curator, backlog-manager | 2 | To Do |

Total about 36.5 hours. Critical path: WP-1 (6h), WP-7 (5h), WP-8 (3h), WP-11 (5h), about 19 hours.

## 8. Deferred

| Id | Why |
|---|---|
| L6 | Moving the slot rule into the domain changes the `Worklog` port alongside H2; a pure refactor with no AC. After SP-01, behind the H2 tests. |
| L8 | Removing the Jira per-process cache has no AC. |
| L9 | Unforgeable `IssueKey` and `WeekID` touch every context. N4 already validates keys at `AddEntry`. Plan before Phase 3. |
| L11 | Docs-only; goes with the next conventions update. |

## 9. Risks the reviews did not cover

- **Content match, not identity.** If the user edits or deletes the worklog in Tempo between a crash and the next run, the bot posts again; a hand-logged identical slot is adopted instead. SRE to confirm (Q4).
- **A 429 is retried for POST** (R3 allows it). If Tempo ever stored a worklog and still returned 429, the retry would duplicate it unseen. Tempo does not document that case.
- **Legacy outbox rows keep a NULL owner** under the `NOT VALID` check; the Phase 2 relay must tolerate them (Q2).
- **Two extra DB transactions per entry** (the attempt marker, and saves on `WithoutCancel`), for about 25 entries a week.
- **Sprint capacity.** About 36 hours of work, with SP-01 ending on 2026-10-16, plus the four-reviewer gate before coding. Expect SP-01 to close only if WP-9, WP-10 and the "should" items (Jira and Slack fixtures, smoke, coverage gate) slip to SP-02 (Q7).
- **The lint baseline.** The default `golangci-lint` caps showed 68 findings; the uncapped run on 2026-10-09 showed 88 (revive 70). If the lint cleanup fixed only the 68 shown, 20 revive findings remain, and the stage 0 precondition fails.

## 10. Open questions for the owner

- **Q1.** Approve AC-005.4 for US-005, or record H1 as a technical task with no AC?
- **Q2.** Legacy outbox rows without an owner: keep them NULL (default), or delete them in 00002? Also open: **MarkEntrySubmitted overwrite (test-review Q2).** Default in this plan: the same id is an idempotent no-op; a different id returns `ErrEntryAlreadySubmitted`. Alternative: always reject a second call.
- **Q3.** Permanent 4xx (F5): the default marks the entry rejected and never re-posts it until `submit --retry-rejected`. Accept, or prefer a one-shot block that the next `submit` clears automatically?
- **Q4.** Reconcile matches issue, day, start, duration and author. Also require an equal `description`?
- **Q5.** These facts were verified directly, because `official-source-verifier` was not available here. Re-run them through it before `status: final`?
- **Q6.** N-07 needs `.claude/rules/migrations.md` to glob `services/timesheet-core/internal/migrations/**`. That file is owner-maintained; will you change it with WP-7?
- **Q7.** If the sprint cannot hold about 36 hours, move WP-9, WP-10, the Jira and Slack fixtures and the coverage gate to SP-02, and close SP-01 on the "must" items?
