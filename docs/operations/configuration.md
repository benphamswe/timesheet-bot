# Configuration

Configuration has three sources:

| Source | Holds | Committed |
|---|---|---|
| One `.env` per service, copied from that service's `.env.example` ([ADR-0013](../adr/0013-per-service-env-files.md)): `services/timesheet-core/.env` and `services/allocation-agent/.env` | Local settings such as the local `DATABASE_URL`, and optionally that service's tokens (owner-only) | `.env.example` only, with empty token keys; every `.env` is gitignored |
| `~/.zshrc` | Tokens as exported environment variables, as an alternative to `.env` | no |
| `services/timesheet-core/config/config.yaml`, copied from `config.example.yaml` | Non-secret application settings, read by both services | `config.example.yaml` only |

The repository-root `.env` is no longer read by any `make` target or by `docker-compose.yml`. If you still have one, move its values into `services/timesheet-core/.env`, then delete it.

## How the Go code reads the environment

- The binary reads variables with `os.Getenv`, in `internal/config.Env` (called from `cmd/timesheet/wire.go`). An empty value counts as missing.
- Go has no built-in `.env` loading, and this project uses no dotenv library. The binary sees only what is already in its process environment.
- Something outside the binary therefore loads `services/timesheet-core/.env`: `make run` / `make test-int` on the host, or `docker compose` (`env_file`) for the container.
- Comparison with Java/Spring Boot:

  | Spring Boot | timesheet-bot |
  |---|---|
  | `application.yml` | `config/config.yaml` |
  | `application-{profile}.yml` for the active profile, "always overriding the non-specific ones" | no profiles; one `.env` per service on each developer machine plays the `local` profile's role |
  | Relaxed binding maps `SPRING_CONFIG_NAME` onto `spring.config.name`; OS environment variables override config files | no binding; the code reads fixed names (`SLACK_USER_TOKEN`, ...) with `os.Getenv`, and those names never come from YAML |
  | Profile files are read by the framework at startup | `.env` is read by `make` or compose, never by the binary |

  Quotes from <https://docs.spring.io/spring-boot/reference/features/external-config.html>.

## Environment variables

Names only. This page, like every committed file, never holds a value.

### Where tokens can live

You choose one of two places for each token below:

- **The service's `.env`**: `services/timesheet-core/.env` for the Go CLI's tokens, `services/allocation-agent/.env` for the collector's. Owner-only, gitignored, never committed. Copy each from the `.env.example` next to it and fill in the empty token lines yourself. Keep both at `chmod 600`. Agents never read, write or print them.
- **`~/.zshrc`**: export each token, then open a new shell.

If a key is set in both, the shell wins over `.env` in `make run`, `make test-int` and `make collect` ([precedence](#precedence)). In compose, the container gets the service `.env` only; pass a shell token with `docker compose run -e KEY` ([precedence](#precedence)). Optional direnv is the one exception ([direnv](#optional-direnv)).

If a token leaks (into a commit, a log, a chat, or any file other than a service `.env`), rotate it at its provider and replace the old value.

### Secrets

Set them in the service's `.env` or export them in `~/.zshrc` ([where tokens can live](#where-tokens-can-live)). Each service's `.env.example` lists its keys with empty values.

| Variable | Service `.env` | Used by | Needed for |
|---|---|---|---|
| `SLACK_USER_TOKEN` | `timesheet-core` | Slack collector (Go) | every command except `migrate`, when `activity.source` is `slack` (the default); not needed with `file` |
| `JIRA_BASE_URL`, `JIRA_EMAIL`, `JIRA_API_TOKEN` | `timesheet-core` | Jira issue lookup | every command except `migrate`, in `jira_keys` mode only; not needed with `allocation.mode: report` |
| `TEMPO_API_TOKEN` | `timesheet-core` | Tempo gateway | `submit`, when `submit_targets` has `tempo` |
| `MAVENLINK_API_TOKEN` | `timesheet-core` | Mavenlink gateway, `bearer` auth | `submit`, when `submit_targets` has `mavenlink` and `mavenlink.auth` is `bearer` |
| `MAVENLINK_SESSION_COOKIE` | `timesheet-core` | Mavenlink gateway, `cookie` auth: the full raw `Cookie` header from DevTools; expires with the SSO session | `submit`, when `submit_targets` has `mavenlink` and `mavenlink.auth` is `cookie` |
| `SLACK_BOT_TOKEN`, `SLACK_APP_TOKEN` | `timesheet-core` | Slack bot | [P2] `serve` |
| `DEEPSEEK_API_KEY` | `allocation-agent` (`services/allocation-agent/.env`) | the collector's LLM report parser; sends summaries to DeepSeek, a third party (confirm company policy) | `collect`, only when `report.parser` is `llm` |
| `ANTHROPIC_API_KEY` | not `allocation-agent/.env` while `collect` reads it ([ADR-0013](../adr/0013-per-service-env-files.md)) | agent policy | [P3] |

`JIRA_BASE_URL` (for example `https://<site>.atlassian.net`) and `JIRA_EMAIL` are not secret, but they sit with the Jira token, in `services/timesheet-core/.env` or `~/.zshrc`, so the three stay together.

`collect` needs no Claude credential: it runs on your Claude Code claude.ai login. It exits 2 at startup if `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN` or `CLAUDE_CODE_OAUTH_TOKEN` is set, from the shell or `services/allocation-agent/.env`, because each one hides the claude.ai Slack connector ([activity collection](../commands/activity-collection.md)).

### Local settings

`services/timesheet-core/.env`:

| Variable | Used by | Value in `.env.example` |
|---|---|---|
| `DATABASE_URL` | the CLI on the host (`make run`); everything except `plan --dry-run` | the local compose database on `localhost:5432` (user, password and database `timesheet`) |
| `TEST_DATABASE_URL` | integration tests (`make test-int`) | same local database |

`services/allocation-agent/.env` (all optional, none secret; they override `config.yaml`, and the `collect` flags override them):

| Variable | Overrides |
|---|---|
| `SLACK_USER_ID` | `slack_user_id` |
| `TIMEZONE` | `timezone` |
| `REPORT_CHANNEL` | `report.channel` |

Inside compose, the `app` service sets its own `DATABASE_URL` (host `db`), which overrides the `.env` value. The `db` service keeps its local defaults (`POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` = `timesheet`) in `docker-compose.yml`. They are not read from any `.env` and nothing in `db` is interpolated, because the Postgres image applies them only when the volume is first created, and both URLs above would have to change with them.

The app reads no other runtime settings from the environment. Everything else is in `config.yaml` (below).

A missing variable fails the command with the variable names only.

## Precedence

The rule on the host: **a variable exported in your shell wins over the service `.env`**, and `.env` only fills in what the shell does not set. An empty `KEY=` line in `.env` never clears a token exported in `~/.zshrc`, and a token exported in your shell overrides the same key filled in `.env`.

| Path | Order (highest first) |
|---|---|
| `make run`, `make test-int` (timesheet-core) | shell environment, then `services/timesheet-core/.env`, then the Makefile default (`test-int` only) |
| `make collect` (allocation-agent) | flags, then shell environment, then `services/allocation-agent/.env`, then `config.yaml` |
| `docker compose`, `app` secrets | `docker compose run -e KEY` with `KEY` exported in your shell, then `services/timesheet-core/.env` (`env_file`). `environment:` lists no secret |
| `docker compose`, `DATABASE_URL` | the fixed value in `environment:` beats the `.env` value loaded by `env_file` |
| `docker compose`, `collector` secrets | `services/allocation-agent/.env` (`env_file`) only |

How compose does it, from the official Compose documentation:

- `env_file` on each service points at that service's `.env` with `required: false`. "The `required` attribute defaults to `true`. When `required` is set to `false` and the `.env` file is missing, Compose silently ignores the entry." `required` needs Docker Compose 2.24.0 or later (local: v5.0.2). "Relative paths are resolved from the Compose file's parent folder." Source: <https://docs.docker.com/reference/compose-file/services/#env_file>
- `environment:` beats `env_file`: "Environment variables declared in the `environment` section override these values. This holds true even if those values are empty or undefined." Source: same page.
- `${VAR}` interpolation in `docker-compose.yml` reads only "1. Variables from your shell environment 2. Variables from a file set by `--env-file` 3. If `--env-file` is not set, variables from an `.env` file in the project directory" (<https://docs.docker.com/compose/how-tos/environment-variables/variable-interpolation/>). A service's `env_file` is never used for interpolation.
- So no secret is listed in `environment:`, in any form. Checked on 2026-10-10 with dummy values in a scratch compose file: `KEY: ${KEY:-}` sets the container's `KEY` to an empty string, and a bare `KEY:` unsets it, both hiding the value the `env_file` loaded. Details: [ADR-0013](../adr/0013-per-service-env-files.md#compose-interpolation-checked).
- `docker compose run -e KEY` (no value) passes your shell's `KEY` for that run and wins over the `env_file`. If `KEY` is unset in your shell, the `env_file` value stays. Checked the same day.
- `.env` never enters an image: `services/timesheet-core/.dockerignore` and `services/allocation-agent/.dockerignore` exclude `.env` and `.env.*`. Each service `.env` sits inside its build context, so this exclusion is the control that keeps it out. Tokens reach a container only at run time, through `env_file` or `run -e`.

How `make` does it: the `LOAD_DOTENV` macro in `services/timesheet-core/Makefile` and `services/allocation-agent/Makefile` saves the shell's exported variables, sources `./.env` (the service's own file) with `set -a`, then restores the saved variables. It prints nothing. From the root, `make run` and `make collect` run the recipe inside the service folder (`make -C`), so `./.env` is always the service's file.

## Workflow

```bash
cp services/timesheet-core/.env.example services/timesheet-core/.env      # once; then chmod 600
cp services/allocation-agent/.env.example services/allocation-agent/.env  # once, optional; then chmod 600
# tokens: fill them in the service .env yourself, or export them in ~/.zshrc (open a new shell after editing it)
docker compose up -d db
make -C services/timesheet-core run ARGS="migrate up"
make -C services/timesheet-core run ARGS="plan --dry-run"
# or, in the container (tokens from services/timesheet-core/.env; add -e KEY for a shell-only token):
docker compose run --rm app plan --dry-run
```

Never run a bare `docker compose config` or `docker compose config --environment`: both print resolved values, including every `env_file` value and exported token. Use `docker compose config -q` or `--no-interpolate`.

### Optional: direnv

[direnv](https://direnv.net/) can load a service's `.env` automatically when you `cd` into that service folder, so plain `go run` and `./bin/timesheet` see it too. It is optional and nothing in the repository depends on it.

- Keep your `.envrc` local, in the service folder (for example `services/timesheet-core/.envrc`). It is gitignored and the repository ships none.
- A minimal `.envrc` is the single line `dotenv_if_exists`, which "Loads a .env file into the current environment, but only if it exists" (<https://direnv.net/man/direnv-stdlib.1.html>).
- Unlike `make run`, direnv does not keep your shell's values: its stdlib evaluates the exports produced by `direnv dotenv bash` for every key in the file (<https://github.com/direnv/direnv/blob/master/stdlib.sh>). An empty `SLACK_USER_TOKEN=` line in `.env` would then hide the token from `~/.zshrc`, and a filled line would override it. If you use direnv, keep each token in one place only: fill it in `.env`, or delete its line from `.env` and export it in `~/.zshrc`.
- Never use direnv in `services/allocation-agent/` with an `.envrc` that exports an Anthropic credential: `collect` refuses to start.

## services/timesheet-core/config/config.yaml

| Key | Default | Meaning |
|---|---|---|
| `timezone` | `Asia/Ho_Chi_Minh` | Day boundaries for Slack messages and the default week |
| `jira_account_id` | required in `jira_keys` mode | Tempo `authorAccountId` and timesheet owner; not used in `report` mode, where `slack_user_id` owns the timesheets |
| `slack_user_id` | required | `from:<@...>` in the search query |
| `daily_cap_minutes` | 480 | Maximum logged per day |
| `per_key_minutes` | 45 | Rule policy credit per issue key per day |
| `tempo_base_url` | `https://api.tempo.io/4` | Regional Tempo endpoint |
| `channel_defaults` | empty | Channel name to issue key for keyless messages |
| `allocation.mode` | `jira_keys` | `jira_keys`: 45m per Jira key per day (needs Jira). `report`: each line of your Slack timesheet summaries becomes one entry; no Jira, Mavenlink only ([report mode](#report-mode)) |
| `report.channel` | empty | `report` mode: the summary channel, without `#` (a leading `#` is stripped). Must match `[a-z0-9._-]+` |
| `report.include_self_dm` | `false` when absent | `report` mode: also read your self-DM. The example config sets it to `true` |
| `report.parser` | `manual` | `report` mode: `manual` runs Go's hand-written parser (`reportparser.Parser.Parse`, `TODO(owner)`); `llm` uses the `report_lines` the allocation-agent collector's LLM wrote (activity file version 2) and validates them again in Go |
| `llm.provider`, `llm.model`, `llm.base_url`, `llm.timeout_seconds`, `llm.max_retries` | collector defaults | `report.parser: llm` only. Read by the allocation-agent collector; Go ignores the block |
| `activity.source` | `slack` | Where the week's messages come from: `slack` (search.messages, needs `SLACK_USER_TOKEN`) or `file` (week files from the allocation-agent collector, no Slack token) |
| `activity.file_dir` | `../../../data/activity` | `file` only: the folder of `<ISO week>.json` files. Absolute, or relative to the folder of the config file ([below](#activity-week-files)) |
| `submit_targets` | `[tempo]` | Where `submit` writes: `tempo`, `mavenlink`, or both |
| `mavenlink.auth` | `bearer` | `bearer` (`MAVENLINK_API_TOKEN`) or `cookie` (`MAVENLINK_SESSION_COOKIE`, undocumented; [ADR-0010](../adr/0010-add-mavenlink-as-a-second-submit-target.md)) |
| `mavenlink.base_url` | `https://api.mavenlink.com/api/v1` for `bearer`; `https://27global.mavenlink.com/api/v1` for `cookie` | API endpoint |
| `mavenlink.csrf_page_url` | `https://27global.mavenlink.com/timesheets` | `cookie` only: the page whose `csrf-token` meta tag is sent as `X-CSRF-Token` |
| `mavenlink.submit_for_approval` | `true` | Send one timesheet submission per Mavenlink project once every entry is created |
| `mavenlink.mappings` | empty | Rules mapping entries to a Mavenlink project (`workspace_id`), with optional `story_id`, `role_id`, `billable`. Set `issue_key` and/or `jira_project` (jira_keys mode) or `project` (report mode); `project` cannot be combined with the other two |
| `mavenlink.default_workspace_id`, `default_story_id`, `default_role_id` | 0 (none) | Fallback for entries no rule matches. `default_story_id` applies only to entries that land in the default project; `default_role_id` fills any rule without a `role_id` |

The `mavenlink` block is validated only when `submit_targets` has `mavenlink`. How a rule is chosen is decided in `MavenlinkMapping.Resolve` ([below](#how-an-entry-finds-its-mavenlink-project)).

Template: `services/timesheet-core/config/config.example.yaml`. The real file is gitignored.

### How an entry finds its Mavenlink project

- Matching: `issue_key` compares the whole key ignoring case; `jira_project` compares the key's prefix before the first `-`; `project` compares names ignoring case and extra spaces (the same rule that drops duplicate report lines).
- The most specific rule wins: `issue_key`, then `project`, then `jira_project`. Among rules of the same kind, the first one in the list wins.
- Two rules with the same `issue_key`, or the same `project` after normalization, are a config error that names both indexes.
- No rule matches: with `default_workspace_id` set, the entry goes there and `plan` prints `warning: "<project>" has no mavenlink.mappings rule; its time goes to default_workspace_id <id>`. Without it, `plan` fails with `no Mavenlink mapping for project "<project>": add a mavenlink.mappings rule or set default_workspace_id`.
- Task: the rule's `story_id`, else `default_story_id` only in the default project, else none. Role: the rule's `role_id`, else `default_role_id`, else none. Billable: the rule's flag, else Mavenlink's default.
- When `submit_targets` has `mavenlink`, `plan` and `plan --dry-run` show each entry's project and task in a `MAVENLINK` column, so you can check where time goes before `submit`.

## Report mode

`allocation.mode: report` is for owners without Jira or Tempo ([ADR-0012](../adr/0012-report-mode-slack-summaries-to-mavenlink-without-jira.md)). You write timesheet summaries in Slack, in `report.channel`, in your self-DM, or both; the allocation-agent collector writes them to the week file, and each summary line becomes one entry.

- **Requirements, checked when the config is loaded:**
  - `activity.source: file`;
  - `submit_targets: [mavenlink]` (Tempo needs Jira issues);
  - `report.channel`, `report.include_self_dm: true`, or both;
  - `mavenlink.mappings` rules with `project`, or `mavenlink.default_workspace_id` as a fallback (plan warns for each project that uses it).
- `JIRA_*` and `jira_account_id` are not needed. `slack_user_id` owns the timesheets, so weeks planned in the other mode are not visible.
- Lines that appear in both places are counted once: exact duplicates of date, project, minutes and notes are dropped, ignoring case and extra spaces.
- A day over `daily_cap_minutes` is kept as reported, and `plan` prints `warning: <date>: <total> reported, over the daily cap of <cap>; kept as reported, not scaled`.
- Each line's minutes must be a positive multiple of 15.
- How a summary is read depends on `report.parser`. With `manual` (the default), Go parses each message (`reportparser.Parser.Parse`, `TODO(owner)`) and `plan` fails with `report parsing is not implemented yet` until it is written. With `llm`, the collector's LLM writes `report_lines` into the week file (version 2); Go re-checks every line (minutes, date in the week, project) and fails at the first invalid one, naming its `source_ts`. A file without `report_lines` fails with `activity file has no report_lines: re-run the collector with report.parser: llm`.

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
