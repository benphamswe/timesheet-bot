# Data model

Migrations: `migrations/*.sql` (goose, embedded in the binary).

## Built (00001_init)

| Table | Purpose | Key constraints |
|---|---|---|
| `timesheets` | Aggregate root row | PK `id`; UNIQUE (`user_account_id`, `week_id`); `version` for optimistic locking |
| `timesheet_entries` | One worklog line | FK `timesheet_id` ON DELETE CASCADE; UNIQUE (`timesheet_id`, `issue_key`, `day`); CHECK `minutes > 0 AND minutes % 15 = 0` |
| `outbox` | Domain events awaiting relay | partial index on unprocessed rows |

Entry ids are deterministic: `<timesheet_id>/<YYYY-MM-DD>/<ISSUE-KEY>`.

Migration `00002_mavenlink_entry_id` adds the nullable `timesheet_entries.mavenlink_entry_id` (text), the Mavenlink time entry id, next to `tempo_worklog_id`. An entry is in a target when that target's column is not null ([ADR-0010](../adr/0010-add-mavenlink-as-a-second-submit-target.md)).

`Save` rewrites the entry rows of a timesheet inside the same transaction as the version-checked update and the outbox insert.

## Planned

| Table | Phase | Purpose |
|---|---|---|
| `feedback` | P4 | Reject reasons, thread corrections, Tempo edits |
| `lessons` | P4 | Learned rules, status active or retired |
| `skill_proposals` | P4 | Agent-drafted skills awaiting approval |
| `agent_runs` | P3 | Model, tokens, tool turns, outcome per run |
