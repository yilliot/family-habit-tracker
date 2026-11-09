package service_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/el/family-habit-tracker/internal/db"
	"github.com/el/family-habit-tracker/internal/domain"
	"github.com/el/family-habit-tracker/internal/service"
	"github.com/el/family-habit-tracker/internal/store"
)

// clock is a movable "now", so a test can add a habit, let days pass, and tick
// on each of them.
type clock struct{ now time.Time }

func (c *clock) at(y int, m time.Month, d int) { c.now = time.Date(y, m, d, 9, 30, 0, 0, time.UTC) }

// newTestService builds a service over a fresh database whose clock starts on
// Monday 9 November 2026.
func newTestService(t *testing.T) (*service.Service, *clock, context.Context) {
	t.Helper()

	sqlDB, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	t.Cleanup(func() { sqlDB.Close() })

	c := &clock{}
	c.at(2026, 11, 9)

	dataStore := store.New(sqlDB).WithClock(func() time.Time { return c.now })

	return service.New(dataStore), c, t.Context()
}

// day builds a UTC day.
func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func mustPerson(t *testing.T, svc *service.Service, ctx context.Context, name string) domain.Person {
	t.Helper()

	person, err := svc.CreatePerson(ctx, name, nil)
	if err != nil {
		t.Fatalf("create person %q: %v", name, err)
	}

	return person
}

func mustHabit(t *testing.T, svc *service.Service, ctx context.Context, person domain.Person, name string) domain.Habit {
	t.Helper()

	habit, err := svc.AddHabit(ctx, person.ID, name)
	if err != nil {
		t.Fatalf("add habit %q: %v", name, err)
	}

	return habit
}

func mustTick(t *testing.T, svc *service.Service, ctx context.Context, habit domain.Habit, on time.Time) service.ToggleResult {
	t.Helper()

	result, err := svc.ToggleTick(ctx, habit.ID, on)
	if err != nil {
		t.Fatalf("tick %q on %s: %v", habit.Name, on.Format("2006-01-02"), err)
	}

	return result
}

func balanceOf(t *testing.T, svc *service.Service, ctx context.Context, person domain.Person) int {
	t.Helper()

	balance, err := svc.Points().Balance(ctx, person.ID)
	if err != nil {
		t.Fatalf("balance: %v", err)
	}

	return balance
}

func dayView(t *testing.T, svc *service.Service, ctx context.Context, person domain.Person, on time.Time) service.DayView {
	t.Helper()

	view, err := svc.Day(ctx, person.ID, on)
	if err != nil {
		t.Fatalf("day view: %v", err)
	}

	return view
}

func names(rows []service.DayHabit) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Habit.Name)
	}

	return out
}

// requireValidation asserts that err is a validation failure naming field.
func requireValidation(t *testing.T, err error, field string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected a validation error for %q, got nil", field)
	}

	validation, ok := service.AsValidation(err)
	if !ok {
		t.Fatalf("expected a validation error for %q, got %T: %v", field, err, err)
	}

	if _, present := validation.Fields[field]; !present {
		t.Fatalf("expected field %q to be rejected, got %v", field, validation.Fields)
	}
}
