---
paths:
  - "services/timesheet-core/test/**/*_test.go"
---

# Test files

Every Go test lives in `services/timesheet-core/test/<kind>/<area>/`, outside `internal/` and `cmd/`, and is black-box. This deviates from the Go convention (tests beside the code) by owner choice: [ADR-0009](../../docs/adr/0009-tests-in-separate-tree.md). Layout and folder mapping: `docs/architecture/testing.md`.

## Never

- Put a `*_test.go` file under `internal/` or `cmd/`, or add an `export_test.go` bridge.
- Use an unexported identifier, or declare a package without the `_test` suffix. If a test needs something that is not exported, test it through the exported API, or report the seam it needs (as `slack.NewCollectorWithSearcher` and `httpx.ErrNoRewind` were added).
- Import `timesheet-bot/test/...` from production code. `test/architecture` fails the build if you do.
- Use `time.Now()`, the network, or real tokens in assertions. Use week `2026-W41`, `fixedClock`, and fake values (`tok`, `me@example.com`).
- Add an assertion library. Use the standard `testing` package.
- Edit production code to make a new test pass from a test-only task. Mark it `t.Skip("BUG: ...")` and report it.

## Always

- Name the package `<name>_test` and import the package under test from `timesheet-bot/internal/...`.
- Put the test in the folder that matches the package: `test/unit/<context>/<layer>/`, `test/unit/adapters/<adapter>/`, `test/unit/{kernel,httpx,config}/`, `test/integration/<adapter>/`, `test/contract/`, `test/architecture/`, or `test/e2e/`.
- Put fuzz seed corpora in the test folder's `testdata/fuzz/<FuzzName>/`, not beside the parser.
- Put the integration build tag `//go:build integration` on any test that needs Postgres, and skip when `TEST_DATABASE_URL` is unset.
- Write failure messages as `got X, want Y` and compare errors with `errors.Is`.
- Use hand-written fakes that record calls and can be told to fail.
- Measure coverage with `make cover` (`-coverpkg=./internal/...,./cmd/...`). Plain `go test -cover` on a test folder prints `coverage: [no statements]`, because the test package has no production code.
