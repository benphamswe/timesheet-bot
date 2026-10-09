//go:build integration

package postgres_test

// The test counts outbox rows through its own pool on TEST_DATABASE_URL, so it
// does not reach into the Repository.

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"timesheet-bot/internal/platform/db"
	"timesheet-bot/internal/shared/kernel"
	"timesheet-bot/internal/timesheet/domain"
	"timesheet-bot/internal/timesheet/infrastructure/postgres"
)

// fixedNow is 2026-10-09 16:00 UTC, the suite's fixed clock.
var fixedNow = time.Date(2026, 10, 9, 16, 0, 0, 0, time.UTC)

// newTestRepository migrates TEST_DATABASE_URL and returns a repository on it,
// or skips the test when the variable is not set.
func newTestRepository(t *testing.T) *postgres.Repository {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := t.Context()
	if err := db.Migrate(ctx, dsn, "up"); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	return postgres.NewRepository(openPool(t, dsn))
}

// openPool opens a pool on dsn that is closed when the test ends.
func openPool(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()
	pool, err := db.Open(t.Context(), dsn)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// AC-003.4
func TestRoundTripConflictAndOutbox(t *testing.T) {
	repo := newTestRepository(t)
	// A second pool, separate from the repository's, reads the outbox.
	outbox := openPool(t, os.Getenv("TEST_DATABASE_URL"))
	ctx := t.Context()
	user := "it-" + uuid.NewString() // isolate from other runs
	week := kernel.WeekID{Year: 2026, Week: 41}
	mon := week.Days()[0]

	ts := domain.New(uuid.NewString(), user, week, kernel.MustDuration(480), "rule", fixedNow)
	// Registered before pool.Close runs (cleanups are LIFO). t.Context is
	// already canceled during cleanup, so use a fresh context.
	t.Cleanup(func() {
		if err := repo.Delete(context.Background(), ts.ID()); err != nil {
			t.Errorf("cleanup Delete(%s): %v", ts.ID(), err)
		}
	})
	if err := ts.AddEntry("GCM-1", 101, mon, kernel.MustDuration(45), "work"); err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	if err := ts.Propose(fixedNow); err != nil {
		t.Fatalf("Propose: %v", err)
	}
	if err := repo.Save(ctx, ts); err != nil {
		t.Fatalf("Save(new) error = %v, want nil", err)
	}

	got, err := repo.Get(ctx, user, week)
	if err != nil {
		t.Fatalf("Get(%s, %v) error = %v, want nil", user, week, err)
	}
	if got.Status() != domain.StatusAwaitingApproval || len(got.Entries()) != 1 || got.Version() != 1 {
		t.Fatalf("Get() = status %s, %d entries, version %d, want %s, 1, 1",
			got.Status(), len(got.Entries()), got.Version(), domain.StatusAwaitingApproval)
	}
	if e := got.Entries()[0]; e.IssueKey != "GCM-1" || e.Day != mon || e.Duration.Minutes() != 45 {
		t.Errorf("Get().Entries()[0] = %+v, want GCM-1 on %s for 45m", e, mon)
	}

	stale, err := repo.Get(ctx, user, week)
	if err != nil {
		t.Fatalf("Get(stale copy) error = %v, want nil", err)
	}
	if err := got.Approve(fixedNow); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if err := repo.Save(ctx, got); err != nil {
		t.Fatalf("Save(approved) error = %v, want nil", err)
	}
	if err := stale.Reject("late", fixedNow); err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if err := repo.Save(ctx, stale); !errors.Is(err, domain.ErrConcurrentModification) {
		t.Errorf("Save(stale) error = %v, want ErrConcurrentModification", err)
	}

	var n int
	if err := outbox.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE aggregate_id = $1`, ts.ID()).Scan(&n); err != nil {
		t.Fatalf("count outbox rows: %v", err)
	}
	if n != 2 {
		t.Errorf("outbox rows = %d, want 2 (proposed, approved)", n)
	}
}

func TestRoundTripKeepsIDsPerTarget(t *testing.T) {
	repo := newTestRepository(t)
	ctx := t.Context()
	user := "it-" + uuid.NewString() // isolate from other runs
	week := kernel.WeekID{Year: 2026, Week: 41}
	mon, tue := week.Days()[0], week.Days()[1]

	ts := domain.New(uuid.NewString(), user, week, kernel.MustDuration(480), "rule", fixedNow)
	t.Cleanup(func() {
		if err := repo.Delete(context.Background(), ts.ID()); err != nil {
			t.Errorf("cleanup Delete(%s): %v", ts.ID(), err)
		}
	})
	for _, step := range []func() error{
		func() error { return ts.AddEntry("GCM-1", 101, mon, kernel.MustDuration(45), "work") },
		func() error { return ts.AddEntry("GCM-2", 102, tue, kernel.MustDuration(45), "work") },
		func() error { return ts.Propose(fixedNow) },
		func() error { return ts.Approve(fixedNow) },
	} {
		if err := step(); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}
	first, second := ts.Entries()[0].ID, ts.Entries()[1].ID
	if err := ts.MarkEntrySubmitted(first, domain.TargetTempo, "w-1", fixedNow); err != nil {
		t.Fatalf("MarkEntrySubmitted(tempo): %v", err)
	}
	if err := ts.MarkEntrySubmitted(first, domain.TargetMavenlink, "m-1", fixedNow); err != nil {
		t.Fatalf("MarkEntrySubmitted(mavenlink): %v", err)
	}
	if err := ts.MarkEntrySubmitted(second, domain.TargetMavenlink, "m-2", fixedNow); err != nil {
		t.Fatalf("MarkEntrySubmitted(mavenlink): %v", err)
	}
	if err := repo.Save(ctx, ts); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	got, err := repo.Get(ctx, user, week)
	if err != nil {
		t.Fatalf("Get(%s, %v) error = %v, want nil", user, week, err)
	}
	entries := got.Entries()
	if e := entries[0]; e.TempoWorklogID != "w-1" || e.MavenlinkEntryID != "m-1" {
		t.Errorf("Get().Entries()[0] = %+v, want TempoWorklogID w-1 and MavenlinkEntryID m-1", e)
	}
	if e := entries[1]; e.TempoWorklogID != "" || e.MavenlinkEntryID != "m-2" {
		t.Errorf("Get().Entries()[1] = %+v, want no TempoWorklogID and MavenlinkEntryID m-2", e)
	}
}
