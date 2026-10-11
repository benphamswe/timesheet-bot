# Development setup

First-time setup only. Every command you run after this, grouped by purpose, is in [commands](../commands/README.md).

## Prerequisites

- Go 1.26 (or any Go with `GOTOOLCHAIN=auto`, which downloads the pinned `go1.26.9` on first use inside `services/timesheet-core/`)
- Docker with Compose 2.24.0 or later (`env_file` with `required: false`; tested with v5.0.2)
- `jq`, used by the Claude Code hooks
- Optional: `golangci-lint` v2.14.0 and `govulncheck` v1.8.0 for `make lint` and `make vuln`

## First run

```bash
# 1. Env files, one per service (ADR-0013). The Go CLI's holds the local DATABASE_URL and
#    empty token lines; the collector's is optional.
cp services/timesheet-core/.env.example services/timesheet-core/.env
chmod 600 services/timesheet-core/.env
cp services/allocation-agent/.env.example services/allocation-agent/.env    # optional
chmod 600 services/allocation-agent/.env

# 2. Tokens (names in configuration). Put each in one of two places:
#    - fill the empty token lines in the service .env yourself (gitignored, never committed), or
#    - export them in ~/.zshrc, then open a new shell.
#    If a key is set in both, the shell wins on the host.
#    Never set ANTHROPIC_API_KEY, ANTHROPIC_AUTH_TOKEN or CLAUDE_CODE_OAUTH_TOKEN for `collect`.

# 3. Non-secret config
cp services/timesheet-core/config/config.example.yaml services/timesheet-core/config/config.yaml
#    fill jira_account_id, slack_user_id, channel_defaults, tempo_base_url

# 4. Local database
docker compose up -d db

# 5. Migrate, then preview this week (nothing saved)
make -C services/timesheet-core run ARGS="migrate up"
make -C services/timesheet-core run ARGS="plan --dry-run"
```

`make run` loads `services/timesheet-core/.env` (for `DATABASE_URL`, and the tokens if you put them there) and keeps every variable your shell already exports, so a token from `~/.zshrc` always wins over the same key in `.env`. It never prints a value. `make collect` loads `services/allocation-agent/.env` the same way.

The same steps in the container: `docker compose run --rm app migrate up`, then `docker compose run --rm app plan --dry-run`. The `app` service loads `services/timesheet-core/.env` with `env_file` (optional). A token exported only in your shell does not reach the container unless you pass it by name: `docker compose run --rm -e SLACK_USER_TOKEN app plan --dry-run`.

If you used the old repository-root `.env`, move its values into `services/timesheet-core/.env`, then delete it ([ADR-0013](../adr/0013-per-service-env-files.md)).

Go itself never reads `.env`. If you run `go run ./cmd/timesheet` or `./bin/timesheet` directly, export `DATABASE_URL` in your shell first, or use [direnv](../operations/configuration.md#optional-direnv) (optional, no `.envrc` committed).

How the variables are read, the precedence rules, and the comparison with Spring profiles: [configuration](../operations/configuration.md).

## Check the setup

```bash
make ci     # from the repository root
```

What `make ci` runs, what it does not run, and its current state: [CI](../commands/ci.md). Every other command: [commands](../commands/README.md).
