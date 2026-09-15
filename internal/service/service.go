// Package service holds the application's business rules. Handlers validate
// shape and delegate here; this package owns every rule about what the family
// ledger will and will not accept.
package service

import (
	"context"
	"errors"

	"github.com/el/family-habit-tracker/internal/domain"
	"github.com/el/family-habit-tracker/internal/points"
	"github.com/el/family-habit-tracker/internal/store"
)

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

// bind returns a Service operating inside an open transaction, so rules that
// read their own writes (an achievement check that must see the tick that just
// landed) observe a consistent view.
func (s *Service) bind(txStore *store.Store) *Service {
	return &Service{store: txStore, points: s.points.For(txStore)}
}

// requireActiveSprint rejects any write aimed at a sprint that has been
// archived. Archived sprints are a permanent record and never change.
func requireActiveSprint(sprint domain.Sprint, what string) error {
	if sprint.IsArchived() {
		return domain.Invalid("sprint", "Cannot "+what+" on an archived sprint.")
	}

	return nil
}

// sprintForParticipant loads the sprint an enrolment belongs to.
func (s *Service) sprintForParticipant(ctx context.Context, participant domain.SprintParticipant) (domain.Sprint, error) {
	return s.store.Sprint(ctx, participant.SprintID)
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
