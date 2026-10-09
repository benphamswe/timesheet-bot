-- +goose Up
ALTER TABLE timesheet_entries ADD COLUMN mavenlink_entry_id text;

-- +goose Down
ALTER TABLE timesheet_entries DROP COLUMN mavenlink_entry_id;
