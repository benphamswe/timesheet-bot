# Deployment (laptop, Docker on WSL)

## Install / upgrade

The `app` container reads its tokens from `services/timesheet-core/.env` (`env_file`, optional; owner-only, `chmod 600`). A token kept only in `~/.zshrc` reaches it only with `docker compose run -e KEY` ([configuration](configuration.md#precedence), [ADR-0013](../adr/0013-per-service-env-files.md)).

```bash
docker compose build app
docker compose up -d db
docker compose run --rm app migrate up
```

Migrations are embedded in the image, so each upgrade runs `migrate up` once.

## Weekly operation (Phase 1, manual)

```bash
docker compose run --rm app plan --dry-run   # review
docker compose run --rm app plan
docker compose run --rm app approve
docker compose run --rm app submit
```

From Phase 2, `docker compose up -d` runs `timesheet serve` with `restart: unless-stopped`; enable "Start Docker Desktop when you sign in" so the Friday job and catch-up run.

## Slack activity collection

Collect on the host with `make collect WEEK=<week>` ([activity collection](../commands/activity-collection.md)). The `collector` compose service (profile `collect`) builds, but has no claude.ai login inside the container until the owner picks a credential option ([ADR-0013](../adr/0013-per-service-env-files.md#collector-credentials-in-a-container)).

## Backup

```bash
docker compose exec db pg_dump -U timesheet timesheet > backup-$(date +%F).sql
```

## Rollback

Rebuild the previous image tag and run `migrate down` only if the newer migration must be reverted; check the migration's Down section first.
