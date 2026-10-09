// Package postgres persists Timesheet aggregates and their outbox events.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"timesheet-bot/internal/shared/kernel"
	"timesheet-bot/internal/timesheet/domain"
)

// Repository is a [domain.Repository] backed by PostgreSQL.
type Repository struct{ pool *pgxpool.Pool }

// NewRepository returns a Repository that uses pool.
func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// Get loads the timesheet for user and week with its entries, or returns
// [domain.ErrNotFound].
func (r *Repository) Get(ctx context.Context, user string, week kernel.WeekID) (*domain.Timesheet, error) {
	var (
		s      domain.Snapshot
		weekID string
		capMin int
		status string
	)
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_account_id, week_id, status, policy, daily_cap_minutes, version, proposal_revision, created_at
		FROM timesheets WHERE user_account_id = $1 AND week_id = $2`, user, week.String()).
		Scan(&s.ID, &s.UserAccountID, &weekID, &status, &s.Policy, &capMin, &s.Version, &s.Revision, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	s.Status = domain.Status(status)
	if s.Week, err = kernel.ParseWeekID(weekID); err != nil {
		return nil, err
	}
	if s.DailyCap, err = kernel.NewDuration(capMin); err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, issue_key, issue_id, day, minutes, description, coalesce(tempo_worklog_id, ''), coalesce(mavenlink_entry_id, ''),
		       submit_attempts, last_error
		FROM timesheet_entries WHERE timesheet_id = $1`, s.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			e       domain.Entry
			key     string
			day     time.Time
			minutes int
		)
		if err := rows.Scan(&e.ID, &key, &e.IssueID, &day, &minutes, &e.Description, &e.TempoWorklogID, &e.MavenlinkEntryID, &e.SubmitAttempts, &e.LastError); err != nil {
			return nil, err
		}
		if e.IssueKey, err = kernel.NewIssueKey(key); err != nil {
			return nil, err
		}
		if e.Day, err = kernel.NewWorkDay(day); err != nil {
			return nil, err
		}
		if e.Duration, err = kernel.NewDuration(minutes); err != nil {
			return nil, err
		}
		s.Entries = append(s.Entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return domain.Rehydrate(s), nil
}

// Save writes the aggregate, replaces its entries and appends pending events to
// the outbox in one transaction. Version 0 means insert.
func (r *Repository) Save(ctx context.Context, t *domain.Timesheet) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		s := t.Snapshot()
		next := s.Version + 1
		if s.Version == 0 {
			_, err := tx.Exec(ctx, `
				INSERT INTO timesheets (id, user_account_id, week_id, status, policy, daily_cap_minutes, version, proposal_revision, created_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
				s.ID, s.UserAccountID, s.Week.String(), string(s.Status), s.Policy, s.DailyCap.Minutes(), next, s.Revision, s.CreatedAt)
			if err != nil {
				return fmt.Errorf("insert timesheet: %w", err)
			}
		} else {
			tag, err := tx.Exec(ctx, `
				UPDATE timesheets SET status = $1, policy = $2, version = $3, proposal_revision = $4, updated_at = now()
				WHERE id = $5 AND version = $6`,
				string(s.Status), s.Policy, next, s.Revision, s.ID, s.Version)
			if err != nil {
				return fmt.Errorf("update timesheet: %w", err)
			}
			if tag.RowsAffected() == 0 {
				return domain.ErrConcurrentModification
			}
		}

		if _, err := tx.Exec(ctx, `DELETE FROM timesheet_entries WHERE timesheet_id = $1`, s.ID); err != nil {
			return err
		}
		for _, e := range s.Entries {
			_, err := tx.Exec(ctx, `
				INSERT INTO timesheet_entries (id, timesheet_id, issue_key, issue_id, day, minutes, description,
					tempo_worklog_id, mavenlink_entry_id, submit_attempts, last_error)
				VALUES ($1, $2, $3, $4, $5, $6, $7, nullif($8, ''), nullif($9, ''), $10, $11)`,
				e.ID, s.ID, e.IssueKey.String(), e.IssueID, e.Day.Time(), e.Duration.Minutes(), e.Description,
				e.TempoWorklogID, e.MavenlinkEntryID, e.SubmitAttempts, e.LastError)
			if err != nil {
				return fmt.Errorf("insert entry %s: %w", e.ID, err)
			}
		}

		for _, ev := range t.PullEvents() {
			payload, err := json.Marshal(ev)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO outbox (aggregate_id, event_type, payload) VALUES ($1, $2, $3)`,
				ev.AggregateID(), ev.EventName(), payload); err != nil {
				return fmt.Errorf("insert outbox: %w", err)
			}
		}
		t.SetVersion(next)
		return nil
	})
}

// Delete removes the timesheet with id. Deleting an unknown id is not an error.
func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM timesheets WHERE id = $1`, id)
	return err
}
