# Runbook: submit reported failed entries

**Symptom:** `timesheet submit` prints `N submitted, M failed, status approved` and exits non-zero.

## Diagnose

1. Inspect the failures:
   ```bash
   docker compose exec db psql -U timesheet -c \
     "SELECT e.issue_key, e.day, e.submit_attempts, e.last_error
        FROM timesheet_entries e JOIN timesheets t ON t.id = e.timesheet_id
       WHERE t.week_id = '2026-W41' AND e.tempo_worklog_id IS NULL;"
   ```
2. Match `last_error`:
   - `401` / `403`: token expired or wrong Tempo region -> [token rotation](token-rotation.md), check `tempo_base_url`.
   - `400 ... period closed` / approval locked: the Tempo period is closed; ask the Tempo admin to reopen, or log manually.
   - `400 ... issue`: the issue was deleted or moved; re-plan with `plan --force` if nothing was submitted yet.
   - `429` / `5xx`: transient; retry.

## Fix

```bash
docker compose run --rm app submit --week 2026-W41
```

Only entries without `tempo_worklog_id` are sent, in their original time slots. Never insert worklogs manually for entries the bot still tracks as pending - the next submit would duplicate them.
