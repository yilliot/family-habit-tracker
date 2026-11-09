// Package service holds the application's business rules. Handlers validate
// shape and delegate here; this package owns every rule about what the family
// ledger will and will not accept.
package service

import (
	"context"
	"errors"
	"time"

	"github.com/el/family-habit-tracker/internal/domain"
	"github.com/el/family-habit-tracker/internal/points"
	"github.com/el/family-habit-tracker/internal/store"
)

// EditableDays is how many days, counting today, can still be ticked. A
// forgotten evening or a weekend away can be caught up; anything older is
// history and stays exactly as it was recorded.
const EditableDays = 7

// Service applies the habit-tracker rules over a store.
type Service struct {
	store  *store.Store
	points *points.Calculator
}

// New builds a Service.
func New(s *store.Store) *Service {
	return &Service{store: s, points: points.New(s)}
}

// Store exposes the underlying store for read-only page rendering.
func (s *Service) Store() *store.Store { return s.store }

// Points exposes the balance calculator for read-only page rendering.
func (s *Service) Points() *points.Calculator { return s.points }

// Today is the family's current day.
func (s *Service) Today() time.Time { return s.store.Today() }

// IsEditable reports whether a day can still be ticked.
func (s *Service) IsEditable(day time.Time) bool { return isEditable(day, s.store.Today()) }

func isEditable(day, today time.Time) bool {
	return !day.After(today) && !day.Before(today.AddDate(0, 0, -(EditableDays-1)))
}

// bind returns a Service operating inside an open transaction, so rules that
// read their own writes (an achievement check that must see the tick that just
// landed) observe a consistent view.
func (s *Service) bind(txStore *store.Store) *Service {
	return &Service{store: txStore, points: s.points.For(txStore)}
}

// livePerson loads a family member who has not been archived, reporting a
// missing one as a validation failure the page can show.
func (s *Service) livePerson(ctx context.Context, id int64) (domain.Person, error) {
	person, err := s.store.Person(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return domain.Person{}, domain.Invalid("person", "That person is not in the family list.")
	}

	return person, err
}

// habit loads a habit, reporting a missing one as a validation failure.
func (s *Service) habit(ctx context.Context, id int64) (domain.Habit, error) {
	habit, err := s.store.Habit(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return domain.Habit{}, domain.Invalid("habit", "That habit no longer exists.")
	}

	return habit, err
}

// startOfDay reduces an instant to midnight UTC, the resolution ticks use.
func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// AsValidation extracts a ValidationError from err, reporting whether err was
// one. Handlers use it to decide between re-rendering a form and returning a
// server error.
func AsValidation(err error) (*domain.ValidationError, bool) {
	var v *domain.ValidationError
	if errors.As(err, &v) {
		return v, true
	}

	return nil, false
}
