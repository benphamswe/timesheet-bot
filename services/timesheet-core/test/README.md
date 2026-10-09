# test

Every test for timesheet-core lives in this tree, not beside the code it tests (ADR-0009).

Every package here is a black-box test package (`package <name>_test`). It imports the code it tests from `timesheet-bot/internal/...`. Go allows this because `test/` is inside the module root, the parent of `internal/` (`go help gopath`, the internal rule). Production code never imports `test/`, and `make arch` enforces that.

## Layout

| Directory | Tests | Package under test |
| --- | --- | --- |
| `unit/kernel/` | value objects: duration, issue key, week, work day, clock | `internal/shared/kernel` |
| `unit/allocation/domain/` | rule-based allocation policy | `internal/allocation/domain` |
| `unit/timesheet/domain/` | Timesheet aggregate invariants and transitions | `internal/timesheet/domain` |
| `unit/timesheet/application/` | command handlers, with fakes for every port | `internal/timesheet/application` |
| `unit/config/` | config file and environment loading | `internal/config` |
| `unit/httpx/` | retrying HTTP transport | `internal/shared/httpx` |
| `unit/adapters/jira/` | Jira client against `httptest` | `internal/timesheet/infrastructure/jira` |
| `unit/adapters/tempo/` | Tempo client against `httptest` | `internal/timesheet/infrastructure/tempo` |
| `unit/adapters/slack/` | Slack collector with a fake `Searcher` | `internal/activity/infrastructure/slack` |
| `unit/adapters/file/` | Week-file collector against JSON fixtures in `testdata/<case>/2026-W41.json`, plus parity with the Slack collector | `internal/activity/infrastructure/file` |
| `unit/adapters/mavenlink/` | Mavenlink client and its bearer and cookie auth against `httptest` | `internal/timesheet/infrastructure/mavenlink` |
| `integration/postgres/` | repository round trip, optimistic locking, outbox (`//go:build integration`) | `internal/timesheet/infrastructure/postgres` |
| `architecture/` | dependency rules and the no-wall-clock rule, read from the source of `internal/` and `cmd/` | all production packages |

## Running

Run these from `services/timesheet-core`.

| Command | Runs |
| --- | --- |
| `make test` | `go test ./...`: unit and architecture tests |
| `make test-race` | the same tests with the race detector |
| `make test-int` | `test/integration/...` with `-tags=integration`. It needs Postgres, for example `docker compose up -d db`. It uses `TEST_DATABASE_URL`, and the test skips when that is unset. |
| `make arch` | `test/architecture/...` only |
| `make cover` | every test, with coverage measured on `internal/...` and `cmd/...` |
| `make fuzz` | every `Fuzz` target under `test/...` for `FUZZTIME` each |

To run one package, use `go test ./test/unit/timesheet/domain/`. Add `-run '^TestName$'` to run one test.

## Coverage

A test package here is a different package from the code it covers. Plain `go test -cover ./...` would report 0% for every production package. `make cover` passes `-coverpkg=./internal/...,./cmd/...` so that coverage is counted in the production packages.
