# ADR-0009: Keep every Go test in one `test/` tree, outside the package directories

- Status: Accepted (2026-10-09, owner)
- Date: 2026-10-09
- Deciders: owner
- Related: [ADR-0001](0001-ddd-package-by-bounded-context.md) (package by bounded context, enforced by the architecture tests), [ADR-0008](0008-monorepo-service-layout.md) (each service owns its tooling; Go paths are relative to `services/timesheet-core/`)

**This decision deviates from the official Go convention by owner choice.** The owner chose it after being told that the Go convention is a `_test.go` file beside the code it tests.

## Context

The official Go convention puts a package's tests in the package's own directory:

- `go test` "recompiles each package along with any files with names matching the file pattern `*_test.go`". Test files that declare a package with the suffix `_test` "will be compiled as a separate package". The go tool ignores directories named `testdata`. (`go help test`, Go 1.26.9)
- A test file in the same package "may refer to unexported identifiers within the package". A file in a separate `_test` package must import the package under test, and "only its exported identifiers may be used"; this is black-box testing. Fuzz seed inputs live in `testdata/fuzz/<Name>` and run as regular tests when fuzzing is off. ([pkg.go.dev/testing](https://pkg.go.dev/testing))
- The official module-layout guide shows test files next to their code: `modname_test.go` beside `modname.go`, `auth_test.go` beside `auth.go`, and `token_test.go` inside `auth/token/`. ([Organizing a Go module](https://go.dev/doc/modules/layout))
- The Go repository's own top-level `test/` directory "contains tests of the Go tool chain and runtime", and "Standard library tests should be written as regular Go tests in the appropriate package." ([golang/go test/README.md](https://github.com/golang/go/blob/master/test/README.md))
- By default each test measures coverage only for the package it tests. `-coverpkg` applies coverage analysis "to packages whose import paths match the patterns". (`go help testflag`)

All four sources were read on 2026-10-09.

The owner wants every test in one place, separate from production code, so that a package directory holds only production code and the test suite can be read as one tree. Before this ADR the service had 12 `*_test.go` files in 11 package directories: three packages were tested in-package (`slack`, `postgres`, `archtest`), and `httpx` had an `export_test.go` bridge.

## Decision

1. Every Go test of `services/timesheet-core` lives under `services/timesheet-core/test/`. No `*_test.go` file stays under `internal/` or `cmd/`.
2. Every test package is black-box: it declares `package <name>_test` and imports the production package. No test uses an unexported identifier, and there is no `export_test.go` bridge.
3. The tree:

   | Path | Holds | Build tag |
   |---|---|---|
   | `test/README.md` | The layout and the mapping rule below | - |
   | `test/unit/kernel/` | `internal/shared/kernel` | none |
   | `test/unit/allocation/domain/` | `internal/allocation/domain` | none |
   | `test/unit/timesheet/domain/` | `internal/timesheet/domain` | none |
   | `test/unit/timesheet/application/` | `internal/timesheet/application` | none |
   | `test/unit/config/` | `internal/config` | none |
   | `test/unit/httpx/` | `internal/shared/httpx` | none |
   | `test/unit/adapters/jira/` | `internal/timesheet/infrastructure/jira` | none |
   | `test/unit/adapters/tempo/` | `internal/timesheet/infrastructure/tempo` | none |
   | `test/unit/adapters/slack/` | `internal/activity/infrastructure/slack` | none |
   | `test/integration/postgres/` | `internal/timesheet/infrastructure/postgres` | `//go:build integration` |
   | `test/architecture/` | Architecture fitness tests (was `internal/archtest`); they scan `internal/` and `cmd/` | none |
   | `test/contract/` [planned] | Contract tests: adapters against official API specs, and the repository port contract | none (`integration` for Postgres) |
   | `test/e2e/` [planned] | End-to-end CLI tests of this service | `e2e` |

4. Mapping rule for a new test package: `test/<kind>/<area>/`, where `<kind>` is `unit`, `integration`, `contract`, `architecture` or `e2e`, and `<area>` is the package's path under `internal/`, shortened:
   - domain and application code keep `<context>/<layer>` (`timesheet/domain`, `timesheet/application`);
   - infrastructure adapters become `adapters/<adapter>` (`adapters/tempo`, `adapters/memory`); integration tests use the adapter name alone (`integration/postgres`);
   - `shared/` is dropped (`kernel`, `httpx`);
   - every other package keeps its path (`config`, `interfaces/cli`, `platform/db`).

   End-to-end tests of `cmd/timesheet` go in `test/e2e/`.
5. Cross-service end-to-end tests stay planned at the repo root in `tests/e2e/` for Phase 3. They are not part of this tree.
6. Three production seams replace the in-package access the old tests had:
   - `internal/activity/infrastructure/slack` exports a `Searcher` interface, the one Slack Web API method the collector calls.
   - It also exports `NewCollectorWithSearcher`, so a test can build a `Collector` around a fake `Searcher`.
   - `internal/shared/httpx` exports `ErrNoRewind`, so a test can match it with `errors.Is`.

   The Postgres integration test opens its own `pgxpool` pool (through `db.Open`) to read the outbox, instead of reading the repository's unexported `pool` field. Production code does not change for it.
7. The architecture tests gain a rule that production code (`internal/...`, `cmd/...`) never imports `timesheet-bot/test/...`.
8. Coverage is measured with `-coverpkg=./internal/...,./cmd/...`, because a test package in `test/` would otherwise measure only itself.

## Alternatives considered

- **Tests beside the code (the Go standard).** This is what `go help test`, `pkg.go.dev/testing`, the module-layout guide and the Go repository itself do. Coverage works per package with no flags, in-package tests can reach unexported names, and a package's tests move with it. Rejected by the owner, who prefers one test tree separate from production code.
- **Hybrid: black-box tests in `test/`, in-package tests beside the code.** This keeps unexported access where a test needs it (`slack`, `postgres`, `httpx`) with no new production seams. Rejected by the owner because tests would then live in two places, which is the problem this ADR removes.

## Consequences

- Positive: package directories hold only production code; the whole suite is one tree to read and review; every test exercises only the exported API, which matches "test via public APIs" ([Software Engineering at Google, ch. 12](https://abseil.io/resources/swe-book/html/ch12.html#test_via_public_apis)).
- Negative / accepted risk:
  - Only the exported API can be tested. A behaviour that cannot be reached through the exported API needs a new exported seam or goes untested.
  - Three production seams exist mainly for tests: `slack.Searcher`, `slack.NewCollectorWithSearcher` and `httpx.ErrNoRewind`. Each widens its package's exported API, and each must keep a doc comment and stay stable like any other export.
  - Coverage needs `-coverpkg=./internal/...,./cmd/...`. Without it, each test package measures only itself; it holds no production code, so `go test -cover` prints `coverage: [no statements]`. The total also changes, because `-coverpkg` counts every package that matches the pattern, including packages no test reaches.
  - A package's tests no longer travel with the package. When a bounded context is extracted into its own service (ADR-0006, ADR-0007), its `internal/<context>/` directory and its `test/*/<context>/` directories (and its adapters under `test/unit/adapters/`) must move together in the same change.
  - Fuzz seed corpora live beside the fuzz test, under `test/<kind>/<area>/testdata/fuzz/<FuzzName>/`, not beside the parser.
  - Contributors and tools that expect the Go convention (`go test ./internal/timesheet/domain/` now finds no tests) need the commands in `docs/commands/build-and-test.md`.
- Result on 2026-10-09, after the move: 84 top-level tests (80 pass, 4 skip, the same skips as before), `make ci` green, and `make cover` reports a total of 65.1% with `-coverpkg` (61.5% before the move, without it). The two totals measure different package sets, so they are not comparable.
- Follow-up: the service `Makefile` (`cover`, `arch`) and CI use the new paths and `-coverpkg`; `docs/architecture/testing.md`, `docs/architecture/project-structure.md`, the command docs, `.claude/rules/tests.md` and `.claude/skills/go-testing-conventions/SKILL.md` describe this layout; `.claude/skills/coding-standards/sources.md` records this ADR as a known deviation.
