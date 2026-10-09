# Development setup

First-time setup only. Every command you run after this, grouped by purpose, is in [commands](../commands/README.md).

## Prerequisites

- Go 1.26 (or any Go with `GOTOOLCHAIN=auto`, which downloads the pinned `go1.26.9` on first use inside `services/timesheet-core/`)
- Docker with Compose 2.24.0 or later (`env_file` with `required: false`; tested with v5.0.2)
- `jq`, used by the Claude Code hooks
- Optional: `golangci-lint` v2.14.0 and `govulncheck` v1.8.0 for `make lint` and `make vuln`

## First run

```bash
# 1. Local env file (repository root). Holds the local DATABASE_URL and empty token lines.
cp .env.example .env
chmod 600 .env

# 2. Tokens (names in configuration). Put them in one of two places:
#    - fill the empty token lines in .env yourself (gitignored, never committed), or
#    - export them in ~/.zshrc, then open a new shell.
#    If a key is set in both, the shell wins.

# 3. Non-secret config
cp services/timesheet-core/config/config.example.yaml services/timesheet-core/config/config.yaml
#    fill jira_account_id, slack_user_id, channel_defaults, tempo_base_url

# 4. Local database
docker compose up -d db

# 5. Migrate, then preview this week (nothing saved)
make -C services/timesheet-core run ARGS="migrate up"
make -C services/timesheet-core run ARGS="plan --dry-run"
```

`make run` loads the repository-root `.env` (for `DATABASE_URL`, and the tokens if you put them there) and keeps every variable your shell already exports, so a token from `~/.zshrc` always wins over the same key in `.env`. It never prints a value.

The same steps in the container: `docker compose run --rm app migrate up`, then `docker compose run --rm app plan --dry-run`. The `app` service loads `.env` with `env_file` (optional) and takes the tokens from your shell first, then from `.env`.

Go itself never reads `.env`. If you run `go run ./cmd/timesheet` or `./bin/timesheet` directly, export `DATABASE_URL` in your shell first, or use [direnv](../operations/configuration.md#optional-direnv) (optional, no `.envrc` committed).

How the variables are read, the precedence rules, and the comparison with Spring profiles: [configuration](../operations/configuration.md).

## Check the setup

```bash
make ci     # from the repository root
```

What `make ci` runs, what it does not run, and its current state: [CI](../commands/ci.md). Every other command: [commands](../commands/README.md).
