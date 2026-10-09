-- +goose Up
CREATE TABLE timesheets (
    id                text PRIMARY KEY,
    user_account_id   text        NOT NULL,
    week_id           text        NOT NULL,
    status            text        NOT NULL,
    policy            text        NOT NULL,
    daily_cap_minutes int         NOT NULL,
    version           int         NOT NULL,
    proposal_revision int         NOT NULL DEFAULT 0,
    slack_message_ts  text,
    created_at        timestamptz NOT NULL,
    updated_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_account_id, week_id)
);

CREATE TABLE timesheet_entries (
    id               text PRIMARY KEY,
    timesheet_id     text   NOT NULL REFERENCES timesheets (id) ON DELETE CASCADE,
    issue_key        text   NOT NULL,
    issue_id         bigint NOT NULL,
    day              date   NOT NULL,
    minutes          int    NOT NULL CHECK (minutes > 0 AND minutes % 15 = 0),
    description      text   NOT NULL DEFAULT '',
    tempo_worklog_id text,
    submit_attempts  int    NOT NULL DEFAULT 0,
    last_error       text   NOT NULL DEFAULT '',
    UNIQUE (timesheet_id, issue_key, day)
);

CREATE TABLE outbox (
    id           bigserial PRIMARY KEY,
    aggregate_id text        NOT NULL,
    event_type   text        NOT NULL,
    payload      jsonb       NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    processed_at timestamptz,
    attempts     int         NOT NULL DEFAULT 0
);

CREATE INDEX outbox_unprocessed ON outbox (id) WHERE processed_at IS NULL;

-- +goose Down
DROP TABLE outbox;
DROP TABLE timesheet_entries;
DROP TABLE timesheets;
