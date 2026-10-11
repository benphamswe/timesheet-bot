# Test architecture

Go paths below are relative to `services/timesheet-core/`. `api/`, `services/` and `tests/` paths are relative to the repo root.

## Where tests live, and why

Every Go test lives in one tree, `services/timesheet-core/test/`, outside the package directories. Every test package is black-box: it declares `package <name>_test`, imports the production package, and uses only exported identifiers. The owner decided this on 2026-10-09 in [ADR-0009](../adr/0009-tests-in-separate-tree.md).

This is a deliberate deviation from the official Go convention, which puts `*_test.go` files beside the code they test:

- `go test` builds tests from files matching `*_test.go` in each package directory; files whose package name ends in `_test` are compiled as a separate, black-box package. (`go help test`)
- The Go tutorial places `greetings_test.go` beside `greetings.go`, and the official module-layout guide shows `modname_test.go` and `auth_test.go` next to their code. [Add a test](https://go.dev/doc/tutorial/add-a-test), [Organizing a Go module](https://go.dev/doc/modules/layout)
- Go's own top-level `test/` holds only toolchain and runtime tests; standard library tests stay in their packages ([test/README.md](https://github.com/golang/go/blob/master/test/README.md)).

What the separate tree costs, and how the project pays for it (ADR-0009, Consequences):

- A test in another directory is another package, so it can use only exported identifiers ([testing](https://pkg.go.dev/testing)). Three exported seams exist for that reason: `slack.Searcher`, `slack.NewCollectorWithSearcher` and `httpx.ErrNoRewind`. The Postgres test opens its own pool instead of reading the repository's.
- A test measures coverage only for its own package unless you pass `-coverpkg` (`go help testflag`). `make cover` therefore runs with `-coverpkg=./internal/...,./cmd/...`.
- A package's tests no longer sit with the package. When a bounded context is extracted into its own service, move its `internal/<context>/` code and its folders under `test/` in the same change.
- `test/` sits inside the module root `services/timesheet-core/`, which is the parent of `internal/`, so the Go `internal` rule lets the tests import every production package ([Organizing a Go module](https://go.dev/doc/modules/layout)).

Clean Architecture is not affected. Test files are compiled only by `go test`, never into the binary. The architecture tests in `test/architecture/` scan only `internal/` and `cmd/`, and they fail if production code imports `timesheet-bot/test/...`. A test may import an in-memory adapter (`timesheet/application` tests use `infrastructure/memory`) because that edge exists only at test time.

`golang-standards/project-layout` is not an official Go standard: its [README](https://github.com/golang-standards/project-layout) says so, and Russ Cox says so in [issue 117](https://github.com/golang-standards/project-layout/issues/117).

## Layout by test type

Paths are relative to `services/timesheet-core/`. Folder names follow the mapping rule in [ADR-0009](../adr/0009-tests-in-separate-tree.md#decision): `test/<kind>/<area>/`, where `<kind>` is `unit`, `integration`, `contract`, `architecture` or `e2e`. `<area>` is the package path under `internal/`, shortened: `<context>/<layer>` for domain and application code, `adapters/<adapter>` for adapters (`integration/<adapter>` for integration tests), `shared/` dropped (`kernel`, `httpx`), and every other package by its path (`config`, `interfaces/cli`, `platform/db`).

| Type | Location | Tests | Build tag | Command |
|---|---|---|---|---|
| Unit: kernel | `test/unit/kernel/` | `internal/shared/kernel` | none | `make test` |
| Unit: domain | `test/unit/allocation/domain/`, `test/unit/timesheet/domain/` | `internal/allocation/domain`, `internal/timesheet/domain` | none | `make test` |
| Application handlers | `test/unit/timesheet/application/`; fakes stay private to the test file until they are shared | `internal/timesheet/application` | none | `make test` |
| Unit: config, `httpx` | `test/unit/config/`, `test/unit/httpx/` | `internal/config`, `internal/shared/httpx` | none | `make test` |
| Adapter, `httptest` servers or a fake `Searcher` | `test/unit/adapters/{jira,tempo,slack}/` | `internal/timesheet/infrastructure/{jira,tempo}`, `internal/activity/infrastructure/slack` | none | `make test` |
| API contract [planned] | `test/contract/<adapter>/`, fixtures in its `testdata/` citing the official spec and verified date | the adapter against the official API spec | none | `make test` |
| Vendor API replay [planned] | `test/unit/adapters/<adapter>/`, payloads in its `testdata/` | the adapter | none | `make test` |
| Port contract [planned] | `test/contract/`: `RunRepositoryContract` in a helper package, called for the memory and Postgres repositories; the exact folder is set by the US-010 plan | `timesheet/domain.Repository` implementations | `integration` for Postgres | `make test` / `make test-int` |
| gRPC contract [planned] | `api/proto` (`buf lint`, `buf breaking`); `bufconn` tests in `test/unit/adapters/grpcpolicy/`; Python tests in `services/allocation-agent/tests/` | `grpcpolicy`, `allocation-agent` | none | `make proto-lint`, `make test` |
| Integration | `test/integration/postgres/` | `internal/timesheet/infrastructure/postgres` | `integration` | `make test-int` |
| End to end [planned] | `test/e2e/`; builds and runs the CLI against fake servers, because `package main` cannot be imported | `cmd/timesheet` | `e2e` | `make test-e2e` [planned], not `make test` |
| Fuzz [planned] | `*_fuzz_test.go` in the unit folder of the package that owns the parser; seed corpus in that folder's `testdata/fuzz/<FuzzName>/` | the parser | none | `make fuzz`; seeds also run under `make test` |
| Architecture fitness | `test/architecture/` (was `internal/archtest`) | every production package under `internal/` and `cmd/` | none | `make arch`; also part of `make test` |

Every test package is external (`package <name>_test`). There are no same-package tests and no `export_test.go` bridges. If a test needs something that is not exported, add a documented exported seam in the production package (as `NewCollectorWithSearcher` does), or test the behaviour through the exported API.

Notes on the table:

- Integration tests skip when `TEST_DATABASE_URL` is unset. CI sets it and runs them against a Postgres service container.
- [Testing](https://pkg.go.dev/testing) documents the fuzz seed corpus location (`testdata/fuzz/<Name>` in the fuzz test's package directory), and that seeds run as regular tests when fuzzing is off. Under ADR-0009 that directory is under `test/`, not beside the parser.
- Cross-service end-to-end tests (Phase 3) go at the repo root in `tests/e2e/`. They are separate from `services/timesheet-core/test/e2e/`, which tests this service alone.

## Helpers

- A helper that takes `t` calls `t.Helper()` and fails with `t.Fatal` ([Google Go Style best practices](https://google.github.io/styleguide/go/best-practices)).
- Shared fakes go in a `<pkg>test` package, such as `domaintest` ([Google Go Style best practices](https://google.github.io/styleguide/go/best-practices)).
- Never use `t.Parallel` together with `t.Setenv` ([testing](https://pkg.go.dev/testing)).
- Failure messages read `Func(in) = got, want want` ([Google Go Style decisions](https://google.github.io/styleguide/go/decisions); [useful test failures](https://go.dev/wiki/CodeReviewComments#useful-test-failures)).
- Python tests [built: the P1.5 activity collector; P3 agent planned] live in `services/allocation-agent/tests/`, with a `src` layout and the `importlib` import mode ([pytest good practices](https://docs.pytest.org/en/stable/explanation/goodpractices.html)).

## Known skipped tests

Five tests are skipped on 2026-10-10. Each skip message gives the reason, and the skip is removed when the fix lands.

| Test | Location | Reason |
|---|---|---|
| `TestRulePolicyScalesOverCapDay` | `test/unit/allocation/domain/rule_test.go` | `TODO(owner)`: `scaleDay` is the owner's task (AC-002.3, US-002-T2). Only the owner removes this skip. |
| `TestParse` | `test/unit/adapters/reportparser/parser_test.go` | `TODO(owner)`: `reportparser.Parser.Parse` is the owner's task ([ADR-0012](../adr/0012-report-mode-slack-summaries-to-mavenlink-without-jira.md)). Only the owner removes this skip; change the cases to the report format chosen. |
| `TestCreateDoesNotResendWorklogOn5xx` | `test/unit/adapters/tempo/client_test.go` | BUG H2: the retry transport re-sends a Tempo POST on 5xx (AC-005.3). Fixed by US-005-T2. |
| `TestMarkEntrySubmittedKeepsFirstWorklogID` | `test/unit/timesheet/domain/timesheet_test.go` | BUG exactly-once: a second `MarkEntrySubmitted` with a different worklog id is accepted instead of refused. Fixed by US-028; the owner decided the rule: the same id is a no-op, a different id returns `ErrEntryAlreadySubmitted`. |
| `TestRetryReplaysNoBodyRequest` | `test/unit/httpx/retry_test.go` | BUG NoBody: `RetryTransport` rejects a request whose body is `http.NoBody` (it has no `GetBody`) with `ErrNoRewind` instead of retrying it (AC-017.5). |

## Scaling the test suite

- Each new bounded context brings its own test folders under `test/` (one package per folder), so suites grow per package and run in parallel under `go test ./...`. When a context is extracted into its own service, its test folders move with it (ADR-0009).
- Slow tests are opt-in by build tag (`integration`) and run in CI against a Postgres service container.
- Acceptance criteria ids (`AC-nnn.m`) in test names give traceability from the backlog to tests.
