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

// fixedToday is the day every test pretends it is, so date-boundary rules are
// deterministic rather than dependent on when the suite runs.
var fixedToday = time.Date(2026, 5, 1, 9, 30, 0, 0, time.UTC)

// newTestService builds a service over a fresh database with a pinned clock.
func newTestService(t *testing.T) (*service.Service, *store.Store, context.Context) {
	t.Helper()

	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "test.sqlite")

	sqlDB, err := db.Open(ctx, path)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	t.Cleanup(func() { sqlDB.Close() })

	dataStore := store.New(sqlDB).WithClock(func() time.Time { return fixedToday })

	return service.New(dataStore), dataStore, ctx
}

// day builds a UTC day for the test's fixed calendar.
func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// mustPerson creates a family member or fails the test.
func mustPerson(t *testing.T, svc *service.Service, ctx context.Context, name string) domain.Person {
	t.Helper()

	person, err := svc.CreatePerson(ctx, name, nil)
	if err != nil {
		t.Fatalf("create person %q: %v", name, err)
	}

	return person
}

// startSprint opens a sprint running across the fixed today, with the given
// habits for one person.
func startSprint(t *testing.T, svc *service.Service, ctx context.Context, person domain.Person, habits ...string) (domain.Sprint, domain.SprintParticipant) {
	t.Helper()

	inputs := make([]service.HabitInput, 0, len(habits))
	for index, name := range habits {
		inputs = append(inputs, service.HabitInput{Name: name, DisplayOrder: index + 1})
	}

	sprint, err := svc.StartSprint(ctx, day(2026, 4, 19), day(2026, 5, 16), []service.ParticipantInput{
		{PersonID: person.ID, Habits: inputs, Reward: service.RewardChoice{Mode: service.RewardNone}},
	})
	if err != nil {
		t.Fatalf("start sprint: %v", err)
	}

	participant := participantFor(t, svc, ctx, sprint, person)

	return sprint, participant
}

// participantFor looks up a person's enrolment in a sprint.
func participantFor(t *testing.T, svc *service.Service, ctx context.Context, sprint domain.Sprint, person domain.Person) domain.SprintParticipant {
	t.Helper()

	participant, err := svc.Store().ParticipantForPerson(ctx, sprint.ID, person.ID)
	if err != nil || participant == nil {
		t.Fatalf("participant for person %d: %v", person.ID, err)
	}

	return *participant
}

// habitsOf returns a participant's habits in display order.
func habitsOf(t *testing.T, svc *service.Service, ctx context.Context, participant domain.SprintParticipant) []domain.Habit {
	t.Helper()

	habits, err := svc.Store().HabitsByParticipant(ctx, participant.ID)
	if err != nil {
		t.Fatalf("habits for participant %d: %v", participant.ID, err)
	}

	return habits
}

// balanceOf returns a participant's current sprint balance.
func balanceOf(t *testing.T, svc *service.Service, ctx context.Context, sprint domain.Sprint, participant domain.SprintParticipant) int {
	t.Helper()

	refreshed, err := svc.Store().Participant(ctx, participant.ID)
	if err != nil {
		t.Fatalf("reload participant: %v", err)
	}

	balance, err := svc.Points().SprintBalance(ctx, sprint, refreshed)
	if err != nil {
		t.Fatalf("sprint balance: %v", err)
	}

	return balance
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
