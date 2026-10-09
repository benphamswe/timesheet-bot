# Runbook: a week was not proposed

**Symptom:** no proposal for a past week (laptop off on Friday, or Phase 1 manual run forgotten).

1. Check: `docker compose run --rm app plan --week 2026-W40 --dry-run`.
2. If the preview is right: `docker compose run --rm app plan --week 2026-W40`, then `approve` and `submit` with the same `--week`.
3. From Phase 2 the hourly catch-up does this automatically; if it did not, check that the `app` container is running (`docker compose ps`) and its logs (`docker compose logs app`).

Slack search covers history the user can see, so older weeks can be planned the same way.
