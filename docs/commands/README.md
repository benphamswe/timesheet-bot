# Commands

Every command used on this project, grouped by purpose. This is a living reference: when you use a command that is not here, add it to the file for its purpose in the same change (see [conventions](../development/conventions.md)).

First-time setup lives in [development setup](../development/setup.md). Everything you run after that lives here.

## Files

| File | Purpose |
|---|---|
| [build-and-test.md](build-and-test.md) | `make` targets, `go build`, `go test` (unit, race, integration, cover, fuzz, one test), raw test output |
| [lint-and-format.md](lint-and-format.md) | `golangci-lint run`, `fmt`, `config verify`, `go vet`, `gofmt` |
| [security-and-dependencies.md](security-and-dependencies.md) | `govulncheck`, `go mod tidy`, `go list`, read-only version and supply-chain lookups (proxy, PyPI, deps.dev, GitHub advisories) |
| [database.md](database.md) | Local Postgres container, `migrate up` and `down`, `psql`, backup |
| [docker-and-runtime.md](docker-and-runtime.md) | Image build, compose checks, the local `.env` file, `make run` (CLI via `go run`), and the `timesheet` CLI subcommands and flags |
| [ci.md](ci.md) | What `.github/workflows/ci.yml` runs and how to reproduce each job locally |
| [contracts.md](contracts.md) | `make proto-lint`, `buf lint`, `buf breaking` [planned: P3] |
| [claude-code-tooling.md](claude-code-tooling.md) | Testing hooks, `bash -n`, and the slash commands |
| [activity-collection.md](activity-collection.md) | Slack activity collector: connector check, `make collect`, contract checks |
| [git-and-github.md](git-and-github.md) | Read-only git, the owner's review before a commit, `gh` for the backlog mirror |

## How each entry is written

Each command has:

- the command in a code block;
- **Run from**: the folder to run it in;
- **What it does** and **When to use it**;
- **Expected**: the result you should see, or a common failure and its fix.

Commands that write to an external service, push, or need secrets are marked **Not run while documenting**. Everything else was run on 2026-10-09 and the result is recorded.

## Rules that apply to every file

- Run Go and service `make` commands inside `services/timesheet-core/`. From the repository root, `make <target>` runs the target in every service.
- `make ci` is green (confirmed 2026-10-09): lint reports 0 issues, 5 tests are skipped (see [known skipped tests](../architecture/testing.md#known-skipped-tests)), and `make cover` prints a total. Every Go test lives in `services/timesheet-core/test/` ([ADR-0009](../adr/0009-tests-in-separate-tree.md)); see [build and test](build-and-test.md) and [lint and format](lint-and-format.md).
- Refer to secrets by environment variable name only, for example `${SLACK_USER_TOKEN}`. They live in `.env` (owner-only, gitignored) or `~/.zshrc` (see [configuration](../operations/configuration.md#where-tokens-can-live)).
- Any command that commits, pushes, or writes to GitHub needs the owner's approval of the exact content first.

## From the repository root

Use these forms when you are at the root instead of inside `services/timesheet-core/`. All were confirmed on 2026-10-09.

```bash
make <target>                                                        # any service target, in every service
go -C services/timesheet-core test ./...                             # also build, vet, list, version
go -C services/timesheet-core run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...    # or: make vuln
golangci-lint config verify -c services/timesheet-core/.golangci.yml
make lint                                                            # golangci-lint run has no root form
```

Two traps:

- `golangci-lint run ./...` from the root prints `0 issues.` and exits 7. That is a false clean: it loaded no packages. See [lint and format](lint-and-format.md#warning-a-false-clean-from-the-repository-root).
- `govulncheck -C services/timesheet-core ./...` scans the system Go's standard library and can report vulnerabilities the project does not have. See [security and dependencies](security-and-dependencies.md#from-the-repository-root).

## Shell gotchas (zsh)

The owner's shell is zsh. These differ from bash:

- **`echo` turns `\n` into a newline.** JSON piped from `echo '{"prompt":"a\nb"}'` reaches `jq` with a raw newline inside a string and fails with `parse error: Invalid string: control characters from U+0000 through U+001F must be escaped`. Use `printf '%s' '<json>'` instead.
- **Unquoted globs fail when nothing matches.** `grep -rn TODO --include=*.md docs` fails with `no matches found: --include=*.md`, because zsh expands the glob before `grep` sees it. Quote it: `--include='*.md'`.
- **A word that starts with `=` is expanded to a command path.** `echo ===` fails with `==: not found`. Quote it: `echo "==="`.
- **`path` is tied to `PATH`.** In zsh the lowercase array `path` mirrors `PATH`, so `for path in a b; do ...; done` overwrites your `PATH`, and every command after it in the loop fails with `command not found`. Use another loop variable, for example `for p in a b`.
- **An unquoted variable that holds a command is not split into words.** `G="python3 -I x.py"; $G` looks for a program literally named `python3 -I x.py` and fails with `command not found: python3 -I x.py` (or `no such file or directory` when the value contains a `/`). Write the command out, use an array (`G=(python3 -I x.py); $G`), or define a function.
- **Go commands must run inside `services/timesheet-core/`.** Outside it there is no `go.mod`, so `GOTOOLCHAIN=auto` does not switch to the pinned `go1.26.9`, and `go version` shows the system Go instead.
