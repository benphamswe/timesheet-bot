// Package memory is an in-process Repository for tests and dry runs.
package memory

import (
	"context"
	"sync"

	"timesheet-bot/internal/shared/kernel"
	"timesheet-bot/internal/timesheet/domain"
)

// Repository is an in-memory [domain.Repository] keyed by user and week. Events
// holds every event drained by Save, in order. Repository is safe for
// concurrent use, except that Events must not be read while Save may run.
type Repository struct {
	mu     sync.Mutex
	byKey  map[string]domain.Snapshot
	Events []kernel.DomainEvent
}

// NewRepository returns an empty Repository.
func NewRepository() *Repository { return &Repository{byKey: map[string]domain.Snapshot{}} }

func key(user string, week kernel.WeekID) string { return user + "|" + week.String() }

// Get returns the timesheet for user and week, or [domain.ErrNotFound].
func (r *Repository) Get(_ context.Context, user string, week kernel.WeekID) (*domain.Timesheet, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.byKey[key(user, week)]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return domain.Rehydrate(s), nil
}

// Save stores t, increments its version and appends its pending events to
// Events. It returns [domain.ErrConcurrentModification] if the stored timesheet
// for t's user and week has a different id or version, or if none is stored
// and t's version is not 0.
func (r *Repository) Save(_ context.Context, t *domain.Timesheet) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := key(t.UserAccountID(), t.Week())
	if cur, ok := r.byKey[k]; ok && (cur.ID != t.ID() || cur.Version != t.Version()) {
		return domain.ErrConcurrentModification
	} else if !ok && t.Version() != 0 {
		return domain.ErrConcurrentModification
	}
	t.SetVersion(t.Version() + 1)
	r.byKey[k] = t.Snapshot()
	r.Events = append(r.Events, t.PullEvents()...)
	return nil
}

// Delete removes the timesheet with id. Deleting an unknown id is not an error.
func (r *Repository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, s := range r.byKey {
		if s.ID == id {
			delete(r.byKey, k)
		}
	}
	return nil
}
