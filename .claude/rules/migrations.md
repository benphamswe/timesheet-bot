---
paths:
  - "services/timesheet-core/migrations/**"
---

# Database migrations

Applied migrations are history; changing one corrupts every existing database.

## Never

- Edit or rename a migration that exists on `main` or has been run anywhere.
- Store secrets or environment-specific values in SQL.

## Always

- Add a new file `NNNNN_<snake_case_purpose>.sql` with the next number, containing `-- +goose Up` and a working `-- +goose Down`.
- Name identifiers unquoted `snake_case`, at most 63 bytes. Name indexes `<table>_<purpose>`.
- Use `timestamptz` for times and `text` for strings, and add `CHECK` constraints that mirror domain invariants.
- Run `make test-int` against a fresh database (`up`, then `down` once, then `up`).
