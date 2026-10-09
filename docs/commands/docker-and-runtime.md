# Docker and runtime

Building the image, checking the compose file, and running the `timesheet` CLI, locally or in the `app` container. The CLI is defined with cobra in `services/timesheet-core/internal/interfaces/cli/` and wired in `cmd/timesheet/`.

## Images and compose

### Build the service image

```bash
docker build -t timesheet-bot:local .
```

- **Run from**: `services/timesheet-core/` (the build context; CI runs the same command there with tag `timesheet-bot:ci`).
- **What it does**: a two-stage build: `golang:1.26.9-alpine` compiles a static, trimmed binary, then `gcr.io/distroless/static-debian12:nonroot` runs it as `nonroot` with `ENTRYPOINT ["/usr/local/bin/timesheet"]` and `CMD ["--help"]`.
- **When to use it**: after changing the `Dockerfile`, `.dockerignore`, or `go.mod`, and before a release.
- **Expected**: ends with `naming to docker.io/library/timesheet-bot:local done`. `docker run --rm timesheet-bot:local` prints the CLI help. Confirmed 2026-10-09.

### Build the compose `app` image

```bash
docker compose build app
```

- **Run from**: the repository root.
- **What it does**: builds `./services/timesheet-core` as image `timesheet-bot-app`, the one `docker compose run` uses.
- **When to use it**: after any code change, before `docker compose run`. `run` does not rebuild an existing image on its own.
- **Expected**: exit 0. Confirmed 2026-10-09.

### Validate the compose file

```bash
docker compose config -q
docker compose --profile cli config -q
```

- **Run from**: the repository root.
- **What it does**: parses and validates `docker-compose.yml` and prints nothing on success. The `app` service sits in profile `cli`, so the second form also checks it.
- **When to use it**: after editing `docker-compose.yml`.
- **Expected**: no output, exit 0. Both confirmed 2026-10-09.
- **Warnings**: if a secret is set neither in your shell nor in `.env`, each command warns `The "SLACK_USER_TOKEN" variable is not set. Defaulting to a blank string.` (one line per variable, names only). Set it in `.env` (owner-only, gitignored) or `~/.zshrc`; commands that do not need it, such as `docker compose up -d db`, still work. With a `.env` copied from `.env.example`, the empty lines silence the warning.
- **Note**: never run a bare `docker compose config` or `docker compose config --environment`; both print resolved values, which include the exported values of `SLACK_USER_TOKEN`, `TEMPO_API_TOKEN` and `JIRA_API_TOKEN`. To read the file's structure, use `docker compose config --no-interpolate`, which leaves variables unresolved and prints the token entries as `${SLACK_USER_TOKEN}` and so on, and the `env_file` entry as `path: <repo>/.env, required: false` (confirmed 2026-10-09). The project permissions allow `docker compose config -q`, `--quiet` and `--no-interpolate`, plus `docker compose --profile cli config -q`, without a prompt; any other form prompts.

## Local env file

```bash
cp .env.example .env
```

- **Run from**: the repository root, once.
- **What it does**: creates your local, gitignored `.env` with the local `DATABASE_URL` and `TEST_DATABASE_URL` and empty token lines.
- **Tokens**: fill the token lines in `.env` yourself (owner-only, never committed; run `chmod 600 .env`), or leave them empty and export the tokens in `~/.zshrc`. Agents never read, write or print `.env`. See [where tokens can live](../operations/configuration.md#where-tokens-can-live).
- **Who reads it**: `make run` and `make test-int` (on the host), and `docker compose`, which reads it for `${VAR}` interpolation and loads it into the `app` service with `env_file` (`required: false`, so compose works without it; needs Compose 2.24.0 or later).
- **Precedence**: your shell always wins, so a token exported in `~/.zshrc` overrides the same key in `.env`. In compose, `environment:` beats `env_file`, and the shell beats `.env` for interpolation; `make run` restores your exported variables after loading `.env`. Details and official sources: [configuration](../operations/configuration.md#precedence).
- **Never in an image**: the build context is `services/timesheet-core`, outside which the root `.env` sits, and that folder's `.dockerignore` also excludes `.env` and `.env.*`.
- **Optional**: direnv can load it on `cd`; see [configuration](../operations/configuration.md#optional-direnv). No `.envrc` is committed.

## Running the CLI

You can run the CLI three ways:

| Form | Run from | Needs |
|---|---|---|
| `make -C services/timesheet-core run ARGS="<command>"` | the repository root | secrets from `.env` (owner-only, gitignored) or `~/.zshrc`; `DATABASE_URL` from `.env` (or your shell) for anything except `plan --dry-run` |
| `./bin/timesheet <command>` | `services/timesheet-core/` | `make build`; secrets and `DATABASE_URL` exported in your shell, from `~/.zshrc` (the binary never reads `.env`, so tokens kept only in `.env` need `make run`, compose or direnv) |
| `docker compose run --rm app <command>` | the repository root | `docker compose build app`; secrets from `.env` (owner-only, gitignored) or `~/.zshrc` (compose interpolates them into `environment:`, shell first); compose sets `DATABASE_URL` and starts `db` |

### `make run`

```bash
make -C services/timesheet-core run ARGS="--help"
make -C services/timesheet-core run ARGS="plan --dry-run"
make -C services/timesheet-core run ARGS="plan --week 2026-W41"
```

- **Run from**: the repository root (or `make run ARGS="..."` inside `services/timesheet-core/`).
- **What it does**: exports the repository-root `.env` if it exists, restores every variable your shell already exported (so `~/.zshrc` wins over `.env`), then runs `go run ./cmd/timesheet $(ARGS)`. The recipe is silent and never prints a value.
- **When to use it**: day-to-day local runs after a code change, without `make build` or an image rebuild.
- **Expected**: `make -n -C services/timesheet-core run ARGS="--help"` prints the recipe with no values (confirmed 2026-10-09). The commands themselves need secrets and were not run while documenting.
- **Quoting**: put the whole command in `ARGS="..."`; inner quotes pass through, for example `ARGS='reject --reason "missed Monday"'`.

Both read `config/config.yaml` (in the container, `./services/timesheet-core/config` is mounted read-only at `/app/config`). Override with the global flag `--config <path>`.

Which environment variables each command needs (names only; values come from `.env` (owner-only, gitignored) or `~/.zshrc`):

| Command | Needs |
|---|---|
| `plan --dry-run` | `${JIRA_BASE_URL}`, `${JIRA_EMAIL}`, `${JIRA_API_TOKEN}`, and `${SLACK_USER_TOKEN}` only when `activity.source` is `slack` (the default) |
| `plan`, `approve`, `reject` | the same, plus `${DATABASE_URL}` |
| `submit` | the same, `${DATABASE_URL}`, and the secret of each submit target: `${TEMPO_API_TOKEN}`, `${MAVENLINK_API_TOKEN}` or `${MAVENLINK_SESSION_COOKIE}` |
| `migrate up`, `migrate down` | `${DATABASE_URL}` only |

The commands that call Slack, Jira or Tempo need secrets and talk to external services, so they were **not run while documenting**. Their flags were confirmed from `--help` and the cobra code on 2026-10-09.

### Help

```bash
./bin/timesheet --help
./bin/timesheet <command> --help
docker compose run --rm app plan --help
```

- **What it does**: lists the commands (`plan`, `approve`, `reject`, `submit`, `migrate`, plus cobra's `completion` and `help`) or one command's flags.
- **Expected**: confirmed 2026-10-09 for every command.

### `plan`

```bash
docker compose run --rm app plan --dry-run                 # preview this week; nothing saved, no database
docker compose run --rm app plan --dry-run --week 2026-W41 # preview another week
docker compose run --rm app plan                           # save the proposal
docker compose run --rm app plan --week 2026-W41 --force   # replace a proposal that is not yet submitted
```

- **What it does**: collects your Slack activity for the week (from Slack, or from `<activity.file_dir>/<week>.json` when `activity.source` is `file`), allocates time per Jira issue, and prints the proposed timesheet. Without `--dry-run` it saves the proposal and prints `saved. approve with: timesheet approve --week <week>`.
- **Flags**: `--week` is an ISO week such as `2026-W41`; the default is the current week in the configured timezone. `--dry-run` prints without saving and needs no database. `--force` replaces an existing proposal that has not been submitted.
- **When to use it**: once a week, after the week's work is done. Always preview with `--dry-run` first.
- **Not run while documenting** (needs secrets, calls Slack and Jira).
- **Common failures** (both confirmed 2026-10-09 without secrets):
  - `error: open config/config.yaml: no such file or directory`: copy `config/config.example.yaml` to `config/config.yaml` and fill it in.
  - `error: missing environment variables: SLACK_USER_TOKEN, JIRA_BASE_URL, JIRA_EMAIL, JIRA_API_TOKEN (export them in ~/.zshrc)`: set them in `.env` (owner-only, gitignored) or export them in `~/.zshrc` and open a new shell. The message text still mentions only `~/.zshrc`. It names variables, never values.
  - `--week: ...`: the week is not in `YYYY-Www` form.
  - `error: collect activity: activity file <path> not found: run the allocation-agent collector for <week> first`: with `activity.source: file`, the week's file is missing. Run the collector for that week; the path shown is where it must be ([configuration](../operations/configuration.md#activity-week-files)).
  - `activity file <path>: week ...`, `slack_user_id ...` or `schema_version ...`: the file is for another week or user, or in a format this version does not read.

### Check the week-file source without Slack or Jira

```bash
# inside services/timesheet-core
printf 'jira_account_id: acc-1\nslack_user_id: U1\nactivity:\n  source: file\n' > config/tmp-activity-check.yaml
JIRA_BASE_URL=x JIRA_EMAIL=x JIRA_API_TOKEN=x go run ./cmd/timesheet --config config/tmp-activity-check.yaml plan --dry-run --week 2026-W40
rm -f config/tmp-activity-check.yaml
```

- **What it does**: loads a throwaway config (fake, non-secret values) next to the real one, so the default `file_dir` resolves exactly as it does for `config/config.yaml`, and shows where `plan` looks for the week file. `SLACK_USER_TOKEN` is not needed.
- **Expected** (confirmed 2026-10-10): `error: collect activity: activity file <repo>/data/activity/2026-W40.json not found: run the allocation-agent collector for 2026-W40 first: ...`.

### Check that the container reads the mounted week files

```bash
# from the repository root, after docker compose build app
T=$(mktemp -d) && chmod 755 $T
printf 'jira_account_id: acc-1\nslack_user_id: U0123ABCD\nactivity:\n  source: file\n' > $T/config.yaml && chmod 644 $T/config.yaml
cp services/timesheet-core/test/unit/adapters/file/testdata/happy/2026-W41.json data/activity/
docker run --rm -v $T:/app/config:ro -v $PWD/data/activity:/data/activity:ro \
  -e JIRA_BASE_URL=http://127.0.0.1:9 -e JIRA_EMAIL=x -e JIRA_API_TOKEN=x \
  timesheet-bot-app plan --dry-run --week 2026-W41
rm -f data/activity/2026-W41.json; rm -rf $T
```

- **What it does**: runs the built image with the same two mounts as `docker-compose.yml`, a throwaway config and the fake test fixture as the week file. Jira points at a closed local port, so nothing leaves the machine.
- **Expected** (confirmed 2026-10-10): `error: resolve GCM-1: Get "http://127.0.0.1:9/...": connection refused`. Reaching Jira proves the file was found at `/data/activity` and parsed. With `--week 2026-W40` it fails earlier with `activity file /data/activity/2026-W40.json not found`.

### `approve`

```bash
docker compose run --rm app approve
docker compose run --rm app approve --week 2026-W41
```

- **What it does**: moves the week's proposed timesheet to approved and prints `<week> approved. send it with: timesheet submit --week <week>`.
- **Flags**: `--week` (default: current week).
- **When to use it**: after reviewing the `plan` output.
- **Not run while documenting** (needs secrets and a saved proposal).

### `reject`

```bash
docker compose run --rm app reject --reason "too much time on GCM-1"
docker compose run --rm app reject --week 2026-W41 --reason "missed Monday"
```

- **What it does**: marks the proposal rejected and prints `<week> rejected. re-plan with: timesheet plan --week <week> --force`.
- **Flags**: `--week` (default: current week); `--reason` is free text kept for learning later [planned: P4].
- **When to use it**: when the proposal is wrong; then re-plan with `--force`.
- **Not run while documenting** (needs secrets and a saved proposal).

### `submit`

```bash
docker compose run --rm app submit
docker compose run --rm app submit --week 2026-W41
```

- **What it does**: writes the approved timesheet to every target in `submit_targets` (`config.yaml`): Tempo worklogs, Mavenlink time entries, or both. It prints `<week>: N submitted, M failed, status <status>`, and, when Mavenlink submissions were sent for approval, `mavenlink: K timesheet submissions sent for approval`. Each entry is idempotent per target, so running it again retries only what failed; Mavenlink entries from an interrupted run are found by their external reference, not created twice ([ADR-0010](../adr/0010-add-mavenlink-as-a-second-submit-target.md)).
- **Flags**: `--week` (default: current week).
- **When to use it**: after `approve`.
- **Not run while documenting**: it writes to Tempo and Mavenlink, external services.
- **Common failures**:
  - `N entries failed; run submit again to retry only those`. Rerun `submit`; if the same entries keep failing, check the Tempo token and `tempo_base_url` region, or the `mavenlink.mappings` rules, in [configuration](../operations/configuration.md).
  - `missing environment variables: MAVENLINK_API_TOKEN` or `MAVENLINK_SESSION_COOKIE`: the variable for your `mavenlink.auth` mode is not set. See [Mavenlink credentials](../operations/runbooks/mavenlink-credentials.md).
  - `mavenlink session cookie expired: copy a fresh Cookie header into MAVENLINK_SESSION_COOKIE`: the browser session ended. Nothing was marked; refresh the cookie and rerun ([runbook](../operations/runbooks/mavenlink-credentials.md#when-the-session-expires)).
  - `mavenlink mapping resolution is not implemented yet`: `MavenlinkMapping.Resolve` is still `TODO(owner)`.

### Check which secrets `submit` needs, without sending anything

```bash
T=$(mktemp -d)
printf 'jira_account_id: acc-1\nslack_user_id: U1\nsubmit_targets: [tempo, mavenlink]\nmavenlink:\n  auth: cookie\n' > $T/c.yaml
SLACK_USER_TOKEN=x JIRA_BASE_URL=x JIRA_EMAIL=x JIRA_API_TOKEN=x DATABASE_URL=x \
  go run ./cmd/timesheet --config $T/c.yaml submit
rm -rf $T
```

- **Run from**: `services/timesheet-core`.
- **What it does**: loads a throwaway config with fake, non-secret placeholder values, so the command stops at the secret check before it opens the database or calls any API.
- **When to use it**: after changing `submit_targets` or `mavenlink.auth`, to see which variables you must set.
- **Expected** (confirmed 2026-10-09): `error: missing environment variables: TEMPO_API_TOKEN, MAVENLINK_SESSION_COOKIE`; with `auth: bearer`, `MAVENLINK_API_TOKEN` instead.

### `migrate`

```bash
docker compose run --rm app migrate up
```

See [database](database.md#migrations). Confirmed 2026-10-09.

## Weekly flow

```bash
docker compose up -d db
docker compose run --rm app plan --dry-run
docker compose run --rm app plan
docker compose run --rm app approve
docker compose run --rm app submit
```

The same flow with deployment notes is in [deployment](../operations/deployment.md). From Phase 2, `timesheet serve` runs it on a schedule [planned: P2].
