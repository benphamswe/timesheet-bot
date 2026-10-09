package application

import (
	"context"
	"time"

	"timesheet-bot/internal/shared/kernel"
	"timesheet-bot/internal/timesheet/domain"
)

// TransitionHandler applies approval decisions.
type TransitionHandler struct {
	Repo          domain.Repository
	Clock         kernel.Clock
	UserAccountID string
}

// Approve approves the timesheet for week and saves it.
func (h *TransitionHandler) Approve(ctx context.Context, week kernel.WeekID) error {
	return h.apply(ctx, week, func(ts *domain.Timesheet, now time.Time) error { return ts.Approve(now) })
}

// Reject rejects the timesheet for week with reason and saves it.
func (h *TransitionHandler) Reject(ctx context.Context, week kernel.WeekID, reason string) error {
	return h.apply(ctx, week, func(ts *domain.Timesheet, now time.Time) error { return ts.Reject(reason, now) })
}

func (h *TransitionHandler) apply(ctx context.Context, week kernel.WeekID, fn func(*domain.Timesheet, time.Time) error) error {
	ts, err := h.Repo.Get(ctx, h.UserAccountID, week)
	if err != nil {
		return err
	}
	if err := fn(ts, h.Clock.Now()); err != nil {
		return err
	}
	return h.Repo.Save(ctx, ts)
}
