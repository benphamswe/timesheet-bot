-- +goose Up
-- Report mode (ADR-0012): an entry names either a Jira issue or a project from the owner's
-- report, and one project may have several entries on one day.
ALTER TABLE timesheet_entries ADD COLUMN project text NOT NULL DEFAULT '';
ALTER TABLE timesheet_entries DROP CONSTRAINT timesheet_entries_timesheet_id_issue_key_day_key;
CREATE UNIQUE INDEX timesheet_entries_issue_day ON timesheet_entries (timesheet_id, issue_key, day)
    WHERE issue_key <> '';
ALTER TABLE timesheet_entries ADD CONSTRAINT timesheet_entries_issue_or_project
    CHECK ((issue_key <> '') <> (project <> ''));

-- +goose Down
-- Report entries cannot satisfy the old one-issue-per-day constraint, so they are removed.
DELETE FROM timesheet_entries WHERE project <> '';
ALTER TABLE timesheet_entries DROP CONSTRAINT timesheet_entries_issue_or_project;
DROP INDEX timesheet_entries_issue_day;
ALTER TABLE timesheet_entries ADD CONSTRAINT timesheet_entries_timesheet_id_issue_key_day_key
    UNIQUE (timesheet_id, issue_key, day);
ALTER TABLE timesheet_entries DROP COLUMN project;
