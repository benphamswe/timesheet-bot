# Lint and format

Static checks and formatting. The configuration is `services/timesheet-core/.golangci.yml` (golangci-lint v2, pinned to `v2.14.0` in the service `Makefile` and in CI). Rule rationale is in `.claude/skills/coding-standards/SKILL.md`.

Run every command on this page from `services/timesheet-core/`, unless it is marked as a root form.

## Warning: a false clean from the repository root

```text
$ golangci-lint run ./...          # from the repository root: wrong folder
level=error msg="[linters_context] typechecking error: pattern ./...: directory prefix . does not contain main module or its selected dependencies"
0 issues.
$ echo $?
7
```

From the root there is no `go.mod`, so golangci-lint loads no packages, prints `0 issues.`, and exits 7. It looks clean but checked nothing (confirmed 2026-10-09, while the service still had 68 findings). Always check the exit code, and lint from the service folder or with a root form:

```bash
make lint                                                     # root form: runs inside each service
golangci-lint config verify -c services/timesheet-core/.golangci.yml   # root form: validates the config only
```

Both root forms confirmed 2026-10-09. There is no root form of `golangci-lint run` itself; it has no `-C` flag.

## Pinned golangci-lint version

`GOLANGCI_LINT_VERSION` in `services/timesheet-core/Makefile` is the single source for the version. CI reads it from there, and `make lint` and `make fix` check it before they run.

### `make print-GOLANGCI_LINT_VERSION`

```bash
make -s -C services/timesheet-core print-GOLANGCI_LINT_VERSION    # root form
make -s print-GOLANGCI_LINT_VERSION                               # inside the service
```

- **What it does**: prints the value of a Makefile variable through the `print-%` rule. `-s` hides the make banner, so the output is only the value. The CI `lint` job runs the inside-the-service form to pick the golangci-lint version. `print-GOVULNCHECK_VERSION` works the same way.
- **When to use it**: to see which version CI will install, or before upgrading golangci-lint.
- **Expected**: `v2.14.0`. Confirmed 2026-10-09.

### `make lint-version`

```bash
make lint-version
```

- **Run from**: `services/timesheet-core/`, or the repository root (runs in each service).
- **What it does**: compares `golangci-lint version --short` with `GOLANGCI_LINT_VERSION` and fails when they differ, or when golangci-lint is not installed. `make lint` and `make fix` run it first.
- **When to use it**: after installing or upgrading golangci-lint, or when local lint results differ from CI.
- **Expected**: no output and exit 0 (from the root you see only `== services/timesheet-core: lint-version`). Confirmed 2026-10-09 with golangci-lint 2.14.0.
- **Common failure**: `golangci-lint '2.13.0' found, v2.14.0 required (GOLANGCI_LINT_VERSION)` and exit 2. Install the version the message names. The failure path was confirmed 2026-10-09 by overriding the pin (`make lint-version GOLANGCI_LINT_VERSION=v2.13.0` printed `golangci-lint '2.14.0' found, v2.13.0 required`).

## `golangci-lint run`

```bash
make lint                      # make lint-version, then golangci-lint run ./...
golangci-lint run ./...
golangci-lint run ./internal/timesheet/...    # one subtree
```

- **What it does**: runs the enabled linters and the `gofmt` and `goimports` formatter checks. `.golangci.yml` sets `run.build-tags: [integration]`, so integration test files are linted too. `make lint` checks the installed version first (see [`make lint-version`](#make-lint-version)); the plain command does not.
- **When to use it**: before every review; it is the first step of `make ci` and the CI `lint` job.
- **Expected**: `0 issues.`
- **Current state** (run 2026-10-09): `0 issues.`, exit 0, also uncapped (`--max-issues-per-linter=0 --max-same-issues=0`). The first run that day found 68 (capped; 88 uncapped), most of them missing doc comments; all were cleared on 2026-10-09 (US-015). The tests in `test/` are linted with the rest, and `.golangci.yml` relaxes `gosec`, `unparam` and `noctx` for every `_test.go` path.
- **Common failure**: in Claude Code the RTK filter truncates long lint output. Run `rtk proxy golangci-lint run ./...` to see the per-linter summary at the end.

## `golangci-lint run --fix`

```bash
make fix
```

- **What it does**: `make lint-version`, then `golangci-lint run --fix ./...`, which applies the fixes that linters can make automatically. It edits files.
- **When to use it**: after `make lint`, to clear mechanical findings before fixing the rest by hand.
- **Expected**: fewer issues on the next `make lint`. Not run while documenting, because it edits Go code.

## `golangci-lint fmt`

```bash
golangci-lint fmt --diff ./...     # show what would change; edits nothing
golangci-lint fmt ./...            # rewrite files
golangci-lint fmt path/to/file.go  # one file
```

- **What it does**: runs the formatters from `.golangci.yml` (`gofmt`, and `goimports` with local prefix `timesheet-bot`). The `format-go` hook runs it on every Go file Claude edits.
- **When to use it**: `--diff` to check, without `--diff` to fix.
- **Expected**: `--diff` prints nothing and exits 0.
- **Current state** (run 2026-10-09): the earlier `--diff` change (the alignment of `deny:` in the architecture test, then at `internal/archtest/archtest_test.go`) was fixed with the lint cleanup; lint now reports 0 issues, `gofmt` included. The rewrite form was not run while documenting, because it edits Go code.

## `golangci-lint config verify`

```bash
golangci-lint config verify
```

- **What it does**: validates `.golangci.yml` against the golangci-lint v2 schema.
- **When to use it**: after editing `.golangci.yml`, before running the linters.
- **Expected**: no output, exit 0. Confirmed 2026-10-09.
- **Common failure**: an unknown or misplaced key fails with a schema error that names it. In v2, linter settings live under `linters.settings` and formatter settings under `formatters.settings`. (The failure case was not reproduced while documenting.)

## `go vet`

```bash
go vet ./...
go vet -tags=integration ./...
```

- **What it does**: runs the Go vet analyzers. Integration test files only compile with the tag, so run both. The `verify-build` hook runs both.
- **When to use it**: a fast check when you do not need the full linter set.
- **Expected**: no output. Both confirmed 2026-10-09.

## `gofmt`

```bash
gofmt -l .        # list unformatted files; must print nothing
gofmt -d <file>   # show the diff for one file
```

- **What it does**: lists (or diffs) files that `gofmt` would change. It edits nothing without `-w`.
- **When to use it**: a quick format check without golangci-lint installed.
- **Expected**: no output.
- **Current state** (2026-10-09): the one file it listed earlier (the architecture test, now `test/architecture/archtest_test.go`) is formatted; lint reports 0 issues, `gofmt` included.

## Python: ruff

```bash
make lint                                  # uv sync --locked, ruff check ., ruff format --check .
make fix                                   # ruff check --fix ., ruff format .
uv run --locked ruff check .
uv run --locked ruff format --check .
uv run --locked ruff check src/allocation_agent/activity/normalize.py tests/unit/activity/test_normalize.py   # only the files you changed
```

- **Run from**: `services/allocation-agent/`. The rules are in `pyproject.toml` (`[tool.ruff]`: line length 100, rule sets `E F W I B UP S SIM RUF`).
- **What it does**: lints and checks formatting of `src/` and `tests/`. `make fix` rewrites files.
- **When to use it**: before every review of Python changes.
- **Expected** (2026-10-10): `All checks passed!` and `23 files already formatted`.
