package domain

import (
	"context"

	"timesheet-bot/internal/shared/kernel"
)

// Repository persists Timesheet aggregates. Save also stores pulled domain events.
type Repository interface {
	Get(ctx context.Context, userAccountID string, week kernel.WeekID) (*Timesheet, error)
	Save(ctx context.Context, t *Timesheet) error
	Delete(ctx context.Context, id string) error
}
