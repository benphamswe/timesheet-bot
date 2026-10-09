// Package domain is the Activity context: where evidence of work comes from.
package domain

import (
	"context"

	"timesheet-bot/internal/shared/kernel"
)

// Source collects the user's work activity for a week.
type Source interface {
	Collect(ctx context.Context, week kernel.WeekID) ([]kernel.WorkActivity, error)
}
