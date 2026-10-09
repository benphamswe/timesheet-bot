# Configuration

Configuration has three sources:

| Source | Holds | Committed |
|---|---|---|
| `.env` at the repository root, copied from `.env.example` | Local settings such as the local `DATABASE_URL`, and optionally the tokens (owner-only) | `.env.example` only, with empty token keys; `.env` is gitignored |
| `~/.zshrc` | Tokens as exported environment variables, as an alternative to `.env` | no |
| `services/timesheet-core/config/config.yaml`, copied from `config.example.yaml` | Non-secret application settings | `config.example.yaml` only |

## How the Go code reads the environment

- The binary reads variables with `os.Getenv`, in `internal/config.Env` (called from `cmd/timesheet/wire.go`). An empty value counts as missing.
- Go has no built-in `.env` loading, and this project uses no dotenv library. The binary sees only what is already in its process environment.
- Something outside the binary therefore loads `.env`: `make run` / `make test-int` on the host, or `docker compose` for the container.
- Comparison with Java/Spring Boot:

  | Spring Boot | timesheet-bot |
  |---|---|
  | `application.yml` | `config/config.yaml` |
  | `application-{profile}.yml` for the active profile, "always overriding the non-specific ones" | no profiles; one `.env` per developer machine plays the `local` profile's role |
  | Relaxed binding maps `SPRING_CONFIG_NAME` onto `spring.config.name`; OS environment variables override config files | no binding; the code reads fixed names (`SLACK_USER_TOKEN`, ...) with `os.Getenv`, and those names never come from YAML |
  | Profile files are read by the framework at startup | `.env` is read by `make` or compose, never by the binary |

  Quotes from <https://docs.spring.io/spring-boot/reference/features/external-config.html>.

## Environment variables

Names only. This page, like every committed file, never holds a value.

### Where tokens can live

You choose one of two places for the tokens below:

- **`.env` at the repository root**: owner-only, gitignored, never committed. Copy it from `.env.example` and fill in the empty token lines yourself. Keep it at `chmod 600`. Agents never read, write or print it.
- **`~/.zshrc`**: export each token, then open a new shell.

If a key is set in both, the shell wins over `.env`, in `make run` and in compose alike ([precedence](#precedence)). Optional direnv is the one exception ([direnv](#optional-direnv)).

If a token leaks (into a commit, a log, a chat, or any file other than `.env`), rotate it at its provider and replace the old value.

### Secrets

Set them in `.env` or export them in `~/.zshrc` ([where tokens can live](#where-tokens-can-live)). `.env.example` lists them with empty values.

| Variable | Used by | Needed for |
|---|---|---|
| `SLACK_USER_TOKEN` | Slack collector | every command except `migrate`, when `activity.source` is `slack` (the default); not needed with `file` |
| `JIRA_BASE_URL`, `JIRA_EMAIL`, `JIRA_API_TOKEN` | Jira issue lookup | every `plan` |
| `TEMPO_API_TOKEN` | Tempo gateway | `submit`, when `submit_targets` has `tempo` |
| `MAVENLINK_API_TOKEN` | Mavenlink gateway, `bearer` auth | `submit`, when `submit_targets` has `mavenlink` and `mavenlink.auth` is `bearer` |
| `MAVENLINK_SESSION_COOKIE` | Mavenlink gateway, `cookie` auth: the full raw `Cookie` header from DevTools; expires with the SSO session | `submit`, when `submit_targets` has `mavenlink` and `mavenlink.auth` is `cookie` |
| `SLACK_BOT_TOKEN`, `SLACK_APP_TOKEN` | Slack bot | [P2] `serve` |
| `ANTHROPIC_API_KEY` | agent policy | [P3] |

`JIRA_BASE_URL` (for example `https://<site>.atlassian.net`) and `JIRA_EMAIL` are not secret, but they sit with the Jira token, in `.env` or `~/.zshrc`, so the three stay together.

### Local settings (`.env`)

| Variable | Used by | Value in `.env.example` |
|---|---|---|
| `DATABASE_URL` | the CLI on the host (`make run`); everything except `plan --dry-run` | the local compose database on `localhost:5432` (user, password and database `timesheet`) |
| `TEST_DATABASE_URL` | integration tests (`make test-int`) | same local database |

Inside compose, the `app` service sets its own `DATABASE_URL` (host `db`), which overrides the `.env` value. The `db` service keeps its local defaults (`POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` = `timesheet`) in `docker-compose.yml`. They are not read from `.env`, because the Postgres image applies them only when the volume is first created, and both URLs above would have to change with them.

The app reads no other runtime settings from the environment. Everything else is in `config.yaml` (below).

A missing variable fails the command with the variable names only.

## Precedence

The rule in every path: **a variable exported in your shell wins over `.env`**, and `.env` only fills in what the shell does not set. An empty `KEY=` line in `.env` never clears a token exported in `~/.zshrc`, and a token exported in your shell overrides the same key filled in `.env`. Confirmed 2026-10-09 with dummy values in a scratch copy of the compose file and the Makefile.

| Path | Order (highest first) |
|---|---|
| `make run`, `make test-int` | shell environment, then `.env`, then the Makefile default (`test-int` only) |
| `docker compose`, secrets | `environment:` holds `KEY: ${KEY}`; `${KEY}` comes from the shell first, then from `.env`. Only `run -e` beats it. The `env_file` copy of `.env` has lower precedence and never wins |
| `docker compose`, `DATABASE_URL` | the fixed value in `environment:` beats the `.env` value loaded by `env_file` |

How compose does it, from the official Compose documentation:

- `env_file` on the `app` service points at `.env` with `required: false`. "The `required` attribute defaults to `true`. When `required` is set to `false` and the `.env` file is missing, Compose silently ignores the entry." `required` needs Docker Compose 2.24.0 or later (local: v5.0.2). "Relative paths are resolved from the Compose file's parent folder." Source: <https://docs.docker.com/reference/compose-file/services/#env_file>
- `environment:` beats `env_file`: "Environment variables declared in the `environment` section override these values. This holds true even if those values are empty or undefined." Source: same page.
- The precedence list puts "Set with either the `environment` or `env_file` attribute but with the value interpolated from your shell or an environment file" (2) above "Set using just the `environment` attribute" (3) and "Use of the `env_file` attribute" (4). Source: <https://docs.docker.com/compose/how-tos/environment-variables/envvars-precedence/>
- `.env` in the project directory is also read for `${VAR}` interpolation in `docker-compose.yml`, and the shell wins: "1. Variables from your shell environment 2. Variables from a file set by `--env-file` 3. If `--env-file` is not set, variables from an `.env` file in the project directory". Source: <https://docs.docker.com/compose/how-tos/environment-variables/variable-interpolation/>
- If a secret is set nowhere (neither the shell nor `.env` has the key), compose warns `The "SLACK_USER_TOKEN" variable is not set. Defaulting to a blank string.` The warning names the variable, never a value. Set it in `.env` or export it in `~/.zshrc`, or ignore it for commands that do not need it (`docker compose up -d db`).
- `.env` never enters an image: the build context is `services/timesheet-core`, so the root `.env` is outside it, and `services/timesheet-core/.dockerignore` also excludes `.env` and `.env.*` as defence in depth. Tokens reach the container only at run time, through `environment:`.

How `make run` does it: the `LOAD_DOTENV` macro in `services/timesheet-core/Makefile` saves the shell's exported variables, sources `../../.env` with `set -a`, then restores the saved variables. It prints nothing.

## Workflow

```bash
cp .env.example .env                    # once, at the repository root; then chmod 600 .env
# tokens: fill them in .env yourself, or export them in ~/.zshrc (open a new shell after editing it)
docker compose up -d db
make -C services/timesheet-core run ARGS="migrate up"
make -C services/timesheet-core run ARGS="plan --dry-run"
# or, in the container:
docker compose run --rm app plan --dry-run
```

Never run a bare `docker compose config` or `docker compose config --environment`: both print resolved values, including exported tokens. Use `docker compose config -q` or `--no-interpolate`.

### Optional: direnv

[direnv](https://direnv.net/) can load `.env` automatically when you `cd` into the repository, so plain `go run` and `./bin/timesheet` see it too. It is optional and nothing in the repository depends on it.

- Keep your `.envrc` local. It is gitignored and the repository ships none.
- A minimal `.envrc` is the single line `dotenv_if_exists`, which "Loads a .env file into the current environment, but only if it exists" (<https://direnv.net/man/direnv-stdlib.1.html>).
- Unlike `make run`, direnv does not keep your shell's values: its stdlib evaluates the exports produced by `direnv dotenv bash` for every key in the file (<https://github.com/direnv/direnv/blob/master/stdlib.sh>). An empty `SLACK_USER_TOKEN=` line in `.env` would then hide the token from `~/.zshrc`, and a filled line would override it. If you use direnv, keep each token in one place only: fill it in `.env`, or delete its line from `.env` and export it in `~/.zshrc`.

## services/timesheet-core/config/config.yaml

| Key | Default | Meaning |
|---|---|---|
| `timezone` | `Asia/Ho_Chi_Minh` | Day boundaries for Slack messages and the default week |
| `jira_account_id` | required | Tempo `authorAccountId` and timesheet owner |
| `slack_user_id` | required | `from:<@...>` in the search query |
| `daily_cap_minutes` | 480 | Maximum logged per day |
| `per_key_minutes` | 45 | Rule policy credit per issue key per day |
| `tempo_base_url` | `https://api.tempo.io/4` | Regional Tempo endpoint |
| `channel_defaults` | empty | Channel name to issue key for keyless messages |
| `activity.source` | `slack` | Where the week's messages come from: `slack` (search.messages, needs `SLACK_USER_TOKEN`) or `file` (week files from the allocation-agent collector, no Slack token) |
| `activity.file_dir` | `../../../data/activity` | `file` only: the folder of `<ISO week>.json` files. Absolute, or relative to the folder of the config file ([below](#activity-week-files)) |
| `submit_targets` | `[tempo]` | Where `submit` writes: `tempo`, `mavenlink`, or both |
| `mavenlink.auth` | `bearer` | `bearer` (`MAVENLINK_API_TOKEN`) or `cookie` (`MAVENLINK_SESSION_COOKIE`, undocumented; [ADR-0010](../adr/0010-add-mavenlink-as-a-second-submit-target.md)) |
| `mavenlink.base_url` | `https://api.mavenlink.com/api/v1` for `bearer`; `https://27global.mavenlink.com/api/v1` for `cookie` | API endpoint |
| `mavenlink.csrf_page_url` | `https://27global.mavenlink.com/timesheets` | `cookie` only: the page whose `csrf-token` meta tag is sent as `X-CSRF-Token` |
| `mavenlink.submit_for_approval` | `true` | Send one timesheet submission per Mavenlink project once every entry is created |
| `mavenlink.mappings` | empty | Rules: `issue_key` and/or `jira_project` to `workspace_id`, optional `story_id`, `role_id`, `billable` |
| `mavenlink.default_workspace_id`, `default_story_id`, `default_role_id` | 0 (none) | Fallback for issues no rule matches |

The `mavenlink` block is validated only when `submit_targets` has `mavenlink`. How a rule is chosen is decided in `MavenlinkMapping.Resolve` (`TODO(owner)`); until it is written, every Mavenlink entry fails with "mavenlink mapping resolution is not implemented yet".

Template: `services/timesheet-core/config/config.example.yaml`. The real file is gitignored.

## Activity week files

With `activity.source: file`, `plan` reads `<activity.file_dir>/<ISO week>.json` (for example `2026-W41.json`) instead of calling Slack. The allocation-agent collector writes these files; the format is in [external integrations](../api/external-integrations.md#activity-week-file-built).

How `activity.file_dir` is resolved:

- An absolute path is used as it is.
- A relative path is joined to the folder of the config file passed with `--config`, not to the working directory, then made absolute. Errors show the absolute path.
- The default `../../../data/activity` therefore means the same folder in every supported way of running:

  | How you run | Config file | Week files |
  |---|---|---|
  | `make -C services/timesheet-core run`, or `go run` inside `services/timesheet-core` | `services/timesheet-core/config/config.yaml` | `data/activity/` at the repository root |
  | `docker compose run --rm app` | `/app/config/config.yaml` (mounted) | `/data/activity`, where compose mounts the root `data/activity/` read-only |

- If you keep your config elsewhere, set an absolute `file_dir`.

The week files hold your message text. `data/activity/*.json` is gitignored; only `data/activity/.gitkeep` is committed. A missing file fails `plan` with `activity file <path> not found: run the allocation-agent collector for <week> first`.
