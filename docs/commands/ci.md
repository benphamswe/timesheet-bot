# CI

What `.github/workflows/ci.yml` runs, and how to reproduce each job on your machine.

## What triggers it

- Every push to `main` and every pull request.
- Permissions are `contents: read`. A new run on the same ref cancels the previous one (`concurrency: ci-<ref>`).
- Every `run:` step uses `working-directory: services/timesheet-core`, except in the `proto` job, which runs from the repository root.
- Actions are pinned by commit SHA with the tag in a comment. To check a pin, see [resolve a GitHub Action tag](security-and-dependencies.md#resolve-a-github-action-tag-to-a-commit-sha).
- Dependabot (`.github/dependabot.yml`) opens update PRs, which run the same workflow.

You cannot trigger CI from the command line without pushing, and pushing needs the owner's approval. Reproduce jobs locally instead.

## Jobs and their local equivalents

Run the local commands from `services/timesheet-core/` unless the row says otherwise.

| Job | What CI runs | Reproduce locally |
|---|---|---|
| `proto` | Only when `api/proto` has `.proto` files: installs buf 1.73.0, fetches `main` on pull requests (for `buf breaking`), then `make proto-lint` from the root. Without `.proto` files every step is skipped and the job passes. | `make proto-lint` (from the root) |
| `lint` | `make -s print-GOLANGCI_LINT_VERSION` to read the pinned version (`v2.14.0`), then `golangci/golangci-lint-action` with that version in `services/timesheet-core` | `make lint` |
| `test` | Postgres 17 service container, `TEST_DATABASE_URL` set to the local DSN, then `make test-race` (every package under `test/`), `make test-int` (`test/integration/...`), `make cover` (with `-coverpkg=./internal/...,./cmd/...`) | `docker compose up -d db` (from the root), then `make test-race`, `make test-int`, `make cover` |
| `security` | `make vuln` (`go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...`) | `make vuln` |
| `build` | `docker build -t timesheet-bot:ci .` | `docker build -t timesheet-bot:ci .` |

The `proto` gate is a step-level `if: hashFiles('/api/proto/**/*.proto') != ''`, not a job-level one, because `hashFiles` is not available in a job `if`. The comment in `ci.yml` says so.

Each job's details, expected output and common failures are in [build and test](build-and-test.md), [lint and format](lint-and-format.md), [security and dependencies](security-and-dependencies.md) and [docker and runtime](docker-and-runtime.md).

## `make ci` is close to CI, not identical

```bash
make ci
```

- **Run from**: the repository root (all services) or `services/timesheet-core/`.
- **What it runs**: from the root, `proto-lint` first, then `lint`, `test`, `test-race`, `vuln`, `build` in each service. A bare `make` at the root does the same, because `ci` is the default goal (see [build and test](build-and-test.md#root-ci-target-and-default-goal)). Inside the service, the same list without `proto-lint`.
- **What it does not run**: the integration tests (`make test-int`) and coverage (`make cover`), which the CI `test` job runs, and the Docker build, which the CI `build` job runs.
- **What CI does not run**: `make test` (CI runs `make test-race`, which covers the same tests), `make build` (CI builds the Docker image instead) and `make lint-version` (the lint action installs the pinned version itself). When there are no `.proto` files, `make ci` still calls `proto-lint`, which prints `no proto files yet`; CI skips those steps.
- **Full local reproduction** of every CI job:

  ```bash
  # from the repository root
  docker compose up -d db
  make ci
  make test-int
  make cover
  docker build -t timesheet-bot:ci services/timesheet-core
  ```

- **Expected**: everything passes. Confirmed 2026-10-09: `make ci` printed `no proto files yet` and is fully green; lint reports 0 issues, and 5 tests are skipped (see [known skipped tests](../architecture/testing.md#known-skipped-tests)). `make cover` prints its total line (65.1% on 2026-10-09, measured with `-coverpkg=./internal/...,./cmd/...` because the tests live in `test/`; see [build and test](build-and-test.md#make-cover)). `make test-int` (the Postgres test passes) and the Docker build were confirmed separately.

## Version pins that must match

The service `Makefile` is the single source for the golangci-lint and govulncheck versions; `ci.yml` reads them from there, so CI and local runs cannot drift.

| Tool | CI | Local |
|---|---|---|
| golangci-lint | read with `make -s print-GOLANGCI_LINT_VERSION` | `GOLANGCI_LINT_VERSION` in the service `Makefile`, checked by `make lint-version` |
| govulncheck | `make vuln` | `GOVULNCHECK_VERSION` in the service `Makefile`, run by `make vuln` |
| Go | `go-version-file: services/timesheet-core/go.mod` | `toolchain go1.26.9` in `go.mod`, with `GOTOOLCHAIN=auto` |
| buf | `version: 1.73.0` in `ci.yml` (`proto` job) | not installed yet; see [contracts](contracts.md) |
| Postgres | `postgres:17-alpine` service | `postgres:17-alpine` in `docker-compose.yml` |

Check your local tools:

```bash
make -s -C services/timesheet-core print-GOLANGCI_LINT_VERSION   # the pinned version: v2.14.0
make lint-version                                               # fails if your golangci-lint differs
go -C services/timesheet-core run golang.org/x/vuln/cmd/govulncheck@v1.8.0 -version
```

- **Run from**: the repository root.
- **Expected**: `v2.14.0`; no error from `lint-version`; `Go: go1.26.9` and `Scanner: govulncheck@v1.8.0`. All confirmed 2026-10-09. Do not use `govulncheck -C services/timesheet-core`: it scans the system Go (`go1.22.10`) and reports false vulnerabilities (see [security and dependencies](security-and-dependencies.md#from-the-repository-root)).

## Reading a failed run

Not run while documenting, because it needs the owner's GitHub login:

```bash
gh run list --workflow ci.yml --limit 5
gh run view <run-id> --log-failed
```

These are reads; `--log-failed` prints only the failing steps' logs.
