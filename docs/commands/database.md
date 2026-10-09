# Database

The local Postgres 17 container from `docker-compose.yml`, the schema migrations embedded in the binary (`services/timesheet-core/migrations/`, run by goose), and how to look inside.

## Local DSN

```bash
export DATABASE_URL='postgres://timesheet:timesheet@localhost:5432/timesheet?sslmode=disable'
```

- This is a **non-secret local default**: the compose database listens only on `127.0.0.1:5432`, and `timesheet:timesheet` is the documented local-only credential (the `guard-secrets` hook allows exactly this DSN).
- Inside compose, the `app` service uses host `db` instead of `localhost`; `docker-compose.yml` sets that for you.
- Integration tests read `TEST_DATABASE_URL`, which defaults to the same DSN in `make test-int`.
- Any other database URL is a secret: keep it in `.env` (owner-only, gitignored) or `~/.zshrc` and refer to it as `${DATABASE_URL}`.

## Start the database

```bash
docker compose up -d db
docker compose ps
```

- **Run from**: the repository root.
- **What it does**: starts `postgres:17-alpine` with the named volume `pgdata`, and waits on nothing; the healthcheck (`pg_isready`) runs every 5 seconds.
- **When to use it**: before `migrate`, `make test-int`, or any non-dry-run `plan`.
- **Expected**: `docker compose ps` shows `timesheet-bot-db-1 ... Up ... (healthy) 127.0.0.1:5432->5432/tcp`. Confirmed 2026-10-09 (healthy after about 6 seconds).
- **Common failure**: `port is already allocated` means another Postgres is on 5432. Stop it, or change the host port in `docker-compose.yml` and in your DSN.

## Stop the database

```bash
docker compose stop db     # keep the container and data
docker compose down        # remove containers and network, keep the pgdata volume
docker compose down -v     # also delete the pgdata volume: all local data is lost
```

- **Run from**: the repository root.
- **When to use it**: `down -v` only when you want a clean database; it deletes every stored timesheet.
- **Expected**: `Volume timesheet-bot_pgdata Removed` for `down -v`. Confirmed 2026-10-09 on a database created only for this check.

## Migrations

### Apply all pending migrations

```bash
./bin/timesheet migrate up                    # from services/timesheet-core, after make build
docker compose run --rm app migrate up        # from the root, inside the container
```

- **What it does**: applies every pending goose migration. It needs only `DATABASE_URL`.
- **When to use it**: on first setup and after every upgrade (migrations ship inside the binary and the image).
- **Expected**: `migrate up: ok`. Running it again is a no-op with the same output. Both forms confirmed 2026-10-09.
- **Common failure**: `missing environment variables: DATABASE_URL` means the variable is not exported in this shell. Export the local DSN above.

### Roll back one migration

```bash
./bin/timesheet migrate down
```

- **Run from**: `services/timesheet-core/`.
- **What it does**: rolls back only the most recent migration (goose `Down`).
- **When to use it**: during a rollback, and only when the newer migration must be reverted. Read its `-- +goose Down` section first; it can drop tables and data. See [deployment](../operations/deployment.md).
- **Expected**: `migrate down: ok`. Confirmed 2026-10-09 on a throwaway database.

### Migration status

There is no `migrate status` subcommand. `migrate` accepts only `up` or `down`:

```text
$ ./bin/timesheet migrate status
error: invalid argument "status" for "timesheet migrate"
```

To see which migrations are applied, query goose's table with `psql` (next section).

## Look inside with `psql`

`psql` is not needed on your machine; use the one in the container.

```bash
docker compose exec db psql -U timesheet -d timesheet                    # interactive
docker compose exec -T db psql -U timesheet -d timesheet -c '\dt'        # list tables
docker compose exec -T db psql -U timesheet -d timesheet \
  -c 'SELECT version_id, is_applied FROM goose_db_version ORDER BY id;'  # applied migrations
```

- **Run from**: the repository root, with the database running.
- **What it does**: opens `psql` inside the `db` container. `-T` disables the TTY, which you need when you pass `-c` from a script or from Claude Code.
- **Expected** after `migrate up` (confirmed 2026-10-09): tables `goose_db_version`, `outbox`, `timesheet_entries`, `timesheets`; versions `0`, `1` and `2` with `is_applied = t` (`2` is `00002_mavenlink_entry_id`).
- **Common failure**: `service "db" is not running` means you need `docker compose up -d db` first.

## Throwaway database for migration and integration checks

Checks a new migration (`up`, `down` once, `up`) and runs `make test-int` without touching the compose volume `pgdata`.

```bash
docker run --rm -d --name tsbot-it-pg -e POSTGRES_USER=timesheet -e POSTGRES_PASSWORD=timesheet \
  -e POSTGRES_DB=timesheet -p 55432:5432 postgres:17-alpine
docker exec tsbot-it-pg pg_isready -U timesheet            # repeat until it prints "accepting connections"
export DATABASE_URL='postgres://timesheet:timesheet@localhost:55432/timesheet?sslmode=disable'
go run ./cmd/timesheet migrate up && go run ./cmd/timesheet migrate down && go run ./cmd/timesheet migrate up
docker exec tsbot-it-pg psql -U timesheet -tAc \
  "SELECT column_name FROM information_schema.columns WHERE table_name = 'timesheet_entries' ORDER BY 1;"
TEST_DATABASE_URL=$DATABASE_URL make test-int
docker stop tsbot-it-pg                                      # --rm deletes the container and its data
```

- **Run from**: `services/timesheet-core` (the `docker` lines work anywhere).
- **What it does**: starts a disposable Postgres 17 on host port 55432 with the same local-only credentials, applies the migrations up, down one step and up again, lists the entry columns, runs the integration tests, and removes the container.
- **When to use it**: after adding a migration (`.claude/rules/migrations.md`), or when the compose database holds data you want to keep.
- **Expected** (confirmed 2026-10-09): `migrate up: ok`, `migrate down: ok`, `migrate up: ok`; the columns include `mavenlink_entry_id` and `tempo_worklog_id`; `ok  timesheet-bot/test/integration/postgres`.
- **Common failure**: `port is already allocated` means 55432 is taken; pick another host port in `-p` and in the DSN.

## Back up

```bash
docker compose exec db pg_dump -U timesheet timesheet > backup-$(date +%F).sql
```

- **Run from**: the repository root.
- **What it does**: dumps the whole database to a dated SQL file on the host. Keep backups out of the repository.
- **When to use it**: before `migrate down`, `down -v`, or an upgrade.
- **Not run while documenting** (the check database held no data worth backing up).
