# Deployment (laptop, Docker on WSL)

## Install / upgrade

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

## Backup

```bash
docker compose exec db pg_dump -U timesheet timesheet > backup-$(date +%F).sql
```

## Rollback

Rebuild the previous image tag and run `migrate down` only if the newer migration must be reverted; check the migration's Down section first.
