# timesheet-bot

Fills Jira Tempo timesheets from your Slack activity. Each week it collects your Slack messages, allocates time per Jira issue, lets you approve, then writes worklogs to Tempo.

Design: `docs/superpowers/specs/2026-10-09-timesheet-bot-design.md`. Phase 1 plan: `docs/superpowers/plans/2026-10-09-phase1-core.md`.

## Status

Phase 1 (this code): CLI only, rule-based allocation (45 minutes per Jira key per day, channel defaults for keyless messages).

Not yet built: Slack approval DM and scheduler (Phase 2), Claude agent allocation and skills (Phase 3), learning from feedback (Phase 4).

## Setup

1. Create a Slack app with a **user token** (`xoxp-`) with scopes `search:read`, `channels:history`, `groups:history`, `im:history`.
2. Create a Tempo API token (Tempo > Settings > API integration) and a Jira API token (id.atlassian.com > Security > API tokens).
3. Put the tokens in the Go service's `.env` (owner-only, gitignored) or `~/.zshrc`. For `.env`, run `cp services/timesheet-core/.env.example services/timesheet-core/.env` and `chmod 600 services/timesheet-core/.env` at the repository root, then fill in the empty token lines yourself. Each service has its own `.env` ([ADR-0013](docs/adr/0013-per-service-env-files.md)). For `~/.zshrc`, export them:

   ```bash
   export SLACK_USER_TOKEN=...
   export TEMPO_API_TOKEN=...
   export JIRA_BASE_URL=https://<site>.atlassian.net
   export JIRA_EMAIL=...
   export JIRA_API_TOKEN=...
   ```

4. `cp services/timesheet-core/config/config.example.yaml services/timesheet-core/config/config.yaml` and fill in `jira_account_id`, `slack_user_id`, `channel_defaults`, and the Tempo region URL.

## Run with Docker

```bash
docker compose up -d db
docker compose run --rm app migrate up
docker compose run --rm app plan --dry-run            # preview this week, nothing saved
docker compose run --rm app plan                      # save the proposal
docker compose run --rm app approve
docker compose run --rm app submit                    # write to Tempo; rerun to retry failures
```

Options: `--week 2026-W41`, `plan --force` (replace a not yet submitted proposal), `reject --reason "..."`.

## Run locally

```bash
cd services/timesheet-core
make build
export DATABASE_URL=postgres://timesheet:timesheet@localhost:5432/timesheet?sslmode=disable
./bin/timesheet migrate up
./bin/timesheet plan --dry-run
```

## Tests

```bash
make test        # unit tests
make test-int    # Postgres integration tests (needs `docker compose up -d db`)
make cover       # coverage of production code (-coverpkg=./internal/...,./cmd/...)
```

Run them at the root (every service) or in `services/timesheet-core/`. Every Go test lives in `services/timesheet-core/test/`, not beside the code (ADR-0009); see `docs/architecture/testing.md`.

## Layout

Two services in one repository (ADR-0008). Full annotated tree: `docs/architecture/project-structure.md`.

```text
services/timesheet-core/   Go service: Slack collection, timesheet, Tempo submission (built)
services/allocation-agent/ Python + LangGraph AI allocation service (planned: Phase 3)
api/proto/                 gRPC contracts between services (planned)
docs/                      product and engineering docs
docker-compose.yml         local runtime for all services
Makefile                   runs a target in every service (make ci)
```
