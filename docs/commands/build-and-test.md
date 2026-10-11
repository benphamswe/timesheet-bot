# Build and test

Make targets and Go commands that build the service and run its tests. Targets are defined in the root `Makefile` and in `services/timesheet-core/Makefile`.

Every Go test lives under `services/timesheet-core/test/`, not beside the code it tests ([ADR-0009](../adr/0009-tests-in-separate-tree.md); layout in [testing](../architecture/testing.md#layout-by-test-type)). So `go test ./internal/...` finds no tests: point `go test` at `./test/...`, and measure coverage with `make cover`, which adds `-coverpkg=./internal/...,./cmd/...`.

## The owner-only test

`TestRulePolicyScalesOverCapDay` (in `test/unit/allocation/domain/rule_test.go`) covers `scaleDay` in `internal/allocation/domain/rule.go`, which is marked `TODO(owner)`. With the owner's approval (2026-10-09) the test is skipped with a `TODO(owner)` message, and the owner removes the skip when `scaleDay` lands. Never implement `scaleDay` or remove the skip yourself; see [conventions](../development/conventions.md).

`TestParse` (in `test/unit/adapters/reportparser/parser_test.go`) covers `reportparser.Parser.Parse`, also `TODO(owner)` ([ADR-0012](../adr/0012-report-mode-slack-summaries-to-mavenlink-without-jira.md)); change its cases to your report format, then remove the skip.

See the skip:

```bash
go -C services/timesheet-core test -v ./test/unit/allocation/domain/
```

- **Run from**: the repository root (or drop `-C services/timesheet-core` inside the service).
- **Expected**: `--- SKIP: TestRulePolicyScalesOverCapDay` with the `TODO(owner)` message, and `ok` for the package.
- **Before the skip lands**: the test failed with `rule_test.go:97: total = 585m, want <= 480` (the line number of the file's old location, `internal/allocation/domain/`), and every target that runs the whole suite (`test`, `test-race`, `test-int`, `cover`, `ci`) exits non-zero with only that failure. That was the state when this page was first checked on 2026-10-09.

## From the repository root

Go tools pick the toolchain from the `go.mod` in the current folder, so outside `services/timesheet-core/` they fall back to the system Go. Use these root-safe forms when you are at the root:

```bash
make <target>                                   # runs the target inside each service
go -C services/timesheet-core test ./...        # -C changes folder first, so the pinned toolchain is used
go -C services/timesheet-core build ./...
```

- **Expected**: the same results as inside the service. `go -C services/timesheet-core version` prints `go1.26.9`. Confirmed 2026-10-09.

The root-safe forms for `govulncheck` and `golangci-lint` are in [security and dependencies](security-and-dependencies.md) and [lint and format](lint-and-format.md); both have a false-result trap at the root.

## Root Makefile

### Run a target in every service

```bash
make <target>        # for example: make test, make vuln
```

- **Run from**: the repository root.
- **What it does**: runs `make <target>` in each `services/*/` that has a `Makefile` (today only `services/timesheet-core`), printing `== services/timesheet-core: <target>` first. It stops at the first service that fails. `ci` and `proto-lint` are root targets of their own (see [root `ci` target and default goal](#root-ci-target-and-default-goal)).
- **When to use it**: to run one service target in all services at once.
- **Expected**: the same output as running the target inside the service.

### Preview what `make ci` runs

```bash
make -n ci
```

- **Run from**: the repository root.
- **What it does**: prints the commands without running them. The sub-make also runs in dry-run mode, so you see each service's commands.
- **When to use it**: before a long run, or to check the order CI-equivalent steps run in.
- **Expected** (run 2026-10-09): the `proto-lint` shell step, the service loop, then the service `ci` steps. The `v=$(golangci-lint version --short ...)` block is `lint-version`, which `lint` runs first.

  ```text
  if find api/proto -name '*.proto' | grep -q .; then \
    buf lint api/proto && buf breaking api/proto --against '.git#branch=main,subdir=api/proto'; \
  else echo "no proto files yet"; fi
  for s in services/timesheet-core; do echo "== $s: ci"; /usr/bin/make --no-print-directory -C $s ci || exit 1; done
  == services/timesheet-core: ci
  v=$(golangci-lint version --short 2>/dev/null); \
  if [ "${v#v}" != "2.14.0" ]; then \
    echo "golangci-lint '$v' found, v2.14.0 required (GOLANGCI_LINT_VERSION)" >&2; exit 1; \
  fi
  golangci-lint run ./...
  go test ./...
  go test -race ./...
  go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...
  CGO_ENABLED=0 go build -o bin/timesheet ./cmd/timesheet
  ```

### Root `ci` target and default goal

```bash
make          # same as make ci
make ci
```

- **Run from**: the repository root.
- **What it does**: the root `ci` target runs `proto-lint` first, then `make ci` in each service. `.DEFAULT_GOAL` is `ci`, so a bare `make` runs the same thing. Every other target still goes through the pattern rule to each service.
- **When to use it**: before asking the owner to review a change.
- **Expected**: `no proto files yet` (there are no `.proto` files), then `== services/timesheet-core: ci` and the service steps, all green. Confirmed 2026-10-09: `make ci` is fully green, lint reports 0 issues, and 5 tests are skipped (see [known skipped tests](../architecture/testing.md#known-skipped-tests)). Re-run on 2026-10-10 as `GOTOOLCHAIN=auto make ci` after ADR-0013: exit 0; allocation-agent `All checks passed!` and `132 passed`; timesheet-core `0 issues.` and `No vulnerabilities found.`

`make proto-lint` is defined only at the root; see [contracts](contracts.md).

## Service Makefile targets

Run all of these from `services/timesheet-core/`.

### `make build`

```bash
make build
```

- **What it does**: `CGO_ENABLED=0 go build -o bin/timesheet ./cmd/timesheet`. `bin/` is gitignored.
- **When to use it**: before running the CLI locally (`./bin/timesheet ...`).
- **Expected**: no output, exit 0, a static `bin/timesheet` (about 17 MB). Confirmed 2026-10-09.

### `make test`

```bash
make test
```

- **What it does**: `go test ./...`, which runs every test package under `test/`: unit and architecture tests. Integration files are excluded without the build tag.
- **When to use it**: after every change.
- **Expected**: `ok` for every `timesheet-bot/test/...` package, and `no test files` for the production packages. On 2026-10-09: 84 top-level tests, 80 pass and 4 skip: `TestRulePolicyScalesOverCapDay` ([the owner-only test](#the-owner-only-test)) and three BUG tests ([known skipped tests](../architecture/testing.md#known-skipped-tests)).

### `make test-race`

```bash
make test-race
```

- **What it does**: `go test -race ./...`, the unit suite with the race detector.
- **When to use it**: after touching goroutines, the retry client, or anything shared across calls.
- **Expected**: no `WARNING: DATA RACE`, every package `ok`. Confirmed 2026-10-09: no races.

### `make test-int`

```bash
docker compose up -d db      # from the repository root, first
make test-int
```

- **What it does**: `go test -tags=integration ./test/integration/...` with `TEST_DATABASE_URL` defaulting to the local compose database (`postgres://timesheet:timesheet@localhost:5432/timesheet?sslmode=disable`, a non-secret local default). Set `TEST_DATABASE_URL` to point elsewhere. The recipe first loads `services/timesheet-core/.env` if present (the same `LOAD_DOTENV` macro as [`make run`](docker-and-runtime.md#make-run)), so the order is: your shell, then that `.env`, then the default. Without `.env`, as in CI, it behaves as before.
- **When to use it**: after changing `internal/timesheet/infrastructure/postgres` or a migration.
- **Expected**: `ok timesheet-bot/test/integration/postgres`; `TestRoundTripConflictAndOutbox` passes. Confirmed 2026-10-09 against a fresh container, before the move to `test/`.
- **Common failure**: `connect: connection refused` means the database is not up. Run `docker compose up -d db` from the root and wait until `docker compose ps` shows `(healthy)`. See [database](database.md).

### `make cover`

```bash
make cover
```

- **What it does**: `go test -coverpkg=./internal/...,./cmd/... -coverprofile=coverage.out ./...`, then `go tool cover -func=coverage.out | tail -1` for the total. The tests live in `test/`, so `-coverpkg` is required: without it each test package measures only itself (`go help testflag`). Domain and application packages must stay at or above 85% (qa-engineer).
- **When to use it**: before a review, to check the coverage gate.
- **Expected**: one line `total: (statements) NN.N%`. On 2026-10-09 it printed a total of 65.1%. The total counts every package under `internal/` and `cmd/`, including packages no test reaches, so it is lower than the per-package figures (domain 97.7%, application 95.3%).
- **Per package**: `go tool cover -func=coverage.out` without `tail` lists every function; `go test -coverpkg=./internal/timesheet/domain -cover ./test/unit/timesheet/domain/` prints the coverage of one package.
- **Common failure**: if any test fails, `&&` stops before the summary and you see only `FAIL`. Fix the failing test first. `go test -cover ./internal/...` is not a substitute: it finds no tests there.

### `make fuzz`

```bash
make fuzz                 # 20s per target
make fuzz FUZZTIME=5s     # shorter
```

- **What it does**: finds every `Fuzz*` function in the `./test/...` packages with `go test -list '^Fuzz'` and runs each with `-fuzz` for `FUZZTIME`. Seed corpora live in `test/<kind>/<area>/testdata/fuzz/<FuzzName>/`.
- **When to use it**: after changing parsing code (week ids, durations, Slack message text).
- **Expected**: one `fuzz <pkg> <FuzzName>` line per target. Confirmed 2026-10-09: there are no `Fuzz` targets yet, so it prints nothing and exits 0.

### `make arch`

```bash
make arch
```

- **What it does**: `go test ./test/architecture/...`, the architecture fitness tests. They read the source of `internal/` and `cmd/`: domain imports only the stdlib and the kernel, application does not import infrastructure, no production code imports `timesheet-bot/test/...`, and domain and application code never call the wall clock.
- **When to use it**: after adding an import across packages.
- **Expected**: `ok timesheet-bot/test/architecture`. Confirmed 2026-10-09.

### `make ci`

```bash
make ci
```

- **What it does**: `lint test test-race vuln build`, in that order. `lint` runs `lint-version` first, so a wrong golangci-lint version stops the run before any linting. `vuln` runs the pinned govulncheck through `go run`. It stops at the first failure. CI runs the same targets in separate jobs; the differences are in [CI](ci.md#make-ci-is-close-to-ci-not-identical).
- **When to use it**: before asking the owner to review a change.
- **Expected**: every step passes. Confirmed 2026-10-09: `make ci` is fully green; `lint` reports 0 issues (see [lint and format](lint-and-format.md)), and `test` and `test-race` pass with 5 tests skipped (see [known skipped tests](../architecture/testing.md#known-skipped-tests)).

`make lint`, `make lint-version`, `make fix` and `make print-<VARIABLE>` are described in [lint and format](lint-and-format.md); `make vuln` is in [security and dependencies](security-and-dependencies.md).

## Go commands

Run all of these from `services/timesheet-core/`.

### Build everything

```bash
go build ./...
```

- **What it does**: compiles every package without writing a binary.
- **When to use it**: a fast compile check. The `verify-build` hook runs it when `go-coder`, `test-engineer` or `qa-engineer` finishes.
- **Expected**: no output. Confirmed 2026-10-09 (`go version` inside the service: `go1.26.9`).
- **Common failure**: `go.mod requires go >= 1.26.0` means you are outside the service or `GOTOOLCHAIN` is not `auto`. Check with `go env GOTOOLCHAIN`.

### Run one test

```bash
go test ./test/unit/timesheet/application/ -run TestSubmitPartialFailureThenRetry -v
```

- **What it does**: runs only the tests whose name matches the `-run` regular expression, in one package. Use `-run 'TestA|TestB'` for several, or `-run 'TestX/subtest'` for a subtest.
- **When to use it**: while working on one behaviour.
- **Expected**: `--- PASS: TestSubmitPartialFailureThenRetry`. Confirmed 2026-10-09.
- **Common failure**: `testing: warning: no tests to run` means the name or the package path is wrong. `no test files` means you pointed at a production package such as `./internal/timesheet/application/`; its tests are in `./test/unit/timesheet/application/`.

### Skip a test by name

```bash
go test ./... -skip TestRulePolicyScalesOverCapDay
```

- **What it does**: runs the whole suite except the named test.
- **When to use it**: to run the suite without one known-failing test while you work. Do not add `-skip` to the Makefile or CI; a permanent skip belongs in the test with a reason, as for the owner-only test.
- **Expected**: every package `ok`. Confirmed 2026-10-09, before the move to `test/`.

### Integration tests directly

```bash
TEST_DATABASE_URL='postgres://timesheet:timesheet@localhost:5432/timesheet?sslmode=disable' \
  go test -tags=integration -count=1 ./test/integration/postgres/
```

- **What it does**: same as `make test-int`, for one package. `-count=1` bypasses the test cache, which otherwise reports `(cached)` even when the database changed.
- **Expected**: `ok timesheet-bot/test/integration/postgres`. Without `TEST_DATABASE_URL` the test skips with `TEST_DATABASE_URL not set`.

### Raw test output when the token filter hides failures

```bash
rtk proxy go test ./...
rtk proxy go test ./test/unit/allocation/domain/ -v
```

- **What it does**: runs the command through RTK without its output filter. In Claude Code sessions, `go test` is rewritten to a summarised form that can drop the `--- FAIL:` lines and the assertion message.
- **When to use it**: when a test run says it failed but you cannot see which test or why.
- **Expected**: the normal `go test` output, for example `--- SKIP: TestRulePolicyScalesOverCapDay` with its `TODO(owner)` message.

### Count passed and skipped tests

```bash
go -C services/timesheet-core test -count=1 -json ./... > /tmp/gotest.json
grep '"Action":"skip"' /tmp/gotest.json | grep -c '"Test":'
grep '"Action":"pass"' /tmp/gotest.json | grep -c '"Test":'
```

- **Run from**: the repository root.
- **What it does**: runs every Go test without the cache and counts the `skip` and `pass` events that name a test (subtests included).
- **When to use it**: to report test counts after `make ci`, whose plain output only lists packages.
- **Expected** (2026-10-10): 357 passed (163 top-level), 5 skipped, 0 failed.

## Python service (allocation-agent)

Run these from `services/allocation-agent/`, or from the root with `make -C services/allocation-agent <target>`. uv is pinned in `pyproject.toml` (`[tool.uv] required-version = "==0.12.21"`); another uv version stops with a version error. Tests live in `tests/` with a `src` layout and the `importlib` import mode ([testing](../architecture/testing.md)); they fake the SDK client and never call Claude or Slack.

### `uv sync --locked`

```bash
uv sync --locked          # or: make sync
```

- **Run from**: `services/allocation-agent/`.
- **What it does**: creates `.venv/` (Python 3.12 from `.python-version`; uv downloads it if missing) and installs exactly what `uv.lock` pins. Every other Python target runs it first.
- **When to use it**: after a clone or a lockfile change.
- **Expected**: `Resolved 41 packages` and an install list on the first run; later runs print only the resolve and audit lines. Confirmed 2026-10-10. If `uv.lock` is stale it fails with an error saying the lockfile needs to be updated; run `uv lock` (below).

### `uv lock`

```bash
uv lock                   # or: make lock
```

- **Run from**: `services/allocation-agent/`.
- **What it does**: re-resolves `uv.lock` (every transitive dependency with hashes) from `pyproject.toml`.
- **When to use it**: after editing dependencies in `pyproject.toml`.
- **Expected**: `Resolved 41 packages in ...` (2026-10-10, with `claude-agent-sdk==0.2.165`); `Resolved 44 packages` after `httpx` was added.

### `uv add` and `uv lock --check`

```bash
uv add 'httpx>=0.28,<1'   # adds a runtime dependency to pyproject.toml and re-locks
uv lock --check           # fails if uv.lock does not match pyproject.toml; changes nothing
```

- **Run from**: `services/allocation-agent/`.
- **What it does**: `uv add` writes the requirement into `[project] dependencies`, re-resolves `uv.lock` and installs it into `.venv/`. `uv lock --check` only verifies the lockfile.
- **When to use it**: `uv add` when code needs a new library (after the tech-review rules for new dependencies). `uv lock --check` before a commit that touches `pyproject.toml`.
- **Expected** (2026-10-10): `uv add 'httpx>=0.28,<1'` installed `certifi==2026.7.22`, `httpcore==1.0.9` and `httpx==0.28.1` (used by the DeepSeek adapter, [ADR-0014](../adr/0014-llm-provider-port-with-deepseek-for-report-parsing.md)). `uv lock --check` prints `Resolved 44 packages in ...`.

### `make test` (Python)

```bash
make test                          # uv sync --locked, then uv run --locked pytest
uv run --locked pytest -q          # the same without make
uv run --locked pytest tests/unit/activity/test_week.py -k boundaries   # one file or test
```

- **Run from**: `services/allocation-agent/`.
- **What it does**: runs every test under `tests/`.
- **Expected** (2026-10-10): `259 passed`, with no skips. `tests/unit/activity/test_normalize.py` runs, because `normalize_messages` is implemented. The LLM tests (`tests/unit/llm/`, `tests/unit/report/`, `test_cli_llm.py`) use `FakeLLMClient` or `httpx.MockTransport`. An autouse fixture in `tests/conftest.py` fails any test that tries a real HTTP request.

### `make ci` (Python)

```bash
make ci                   # lint, then test
```

- **Run from**: `services/allocation-agent/`. The root `make ci` runs it too, after `proto-lint` and before `services/timesheet-core`.
- **Expected** (2026-10-10): ruff `All checks passed!`, `44 files already formatted`, then `259 passed` (re-confirmed 2026-10-10 with the LLM parser). `test-race` runs `test`; `test-int`, `cover`, `vuln`, `build`, `fuzz`, `arch` and `run` print `allocation-agent: no '<target>' step for the Python service yet` and pass, so root fan-out targets such as `make run ARGS=...` still reach the Go service.
