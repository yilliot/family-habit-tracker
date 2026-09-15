// Package points derives every balance in the application. Nothing here is
// persisted: a balance is always recomputed from the ticks, rewards and manual
// adjustments that produced it.
package points

import (
	"context"
	"time"

	"github.com/el/family-habit-tracker/internal/domain"
	"github.com/el/family-habit-tracker/internal/store"
)

// Calculator computes lifetime and sprint-scoped balances.
type Calculator struct {
	store *store.Store
}

// New builds a Calculator over a store.
func New(s *store.Store) *Calculator { return &Calculator{store: s} }

// For returns a Calculator bound to a different store, used to run a
// calculation inside an open transaction so it sees that transaction's writes.
func (c *Calculator) For(s *store.Store) *Calculator { return &Calculator{store: s} }

// LifetimeBalance is a person's balance across their entire history:
//
//	every point ever earned
//	  − the cost of every reward ever achieved
//	  + every manual adjustment
//
// It seeds the opening balance of a new sprint enrolment, which is why a big
// goal can be saved toward across as many sprints as it takes.
func (c *Calculator) LifetimeBalance(ctx context.Context, personID int64) (int, error) {
	earned, err := c.store.TicksEarnedByPerson(ctx, personID)
	if err != nil {
		return 0, err
	}

	spent, err := c.store.SpentByPerson(ctx, personID)
	if err != nil {
		return 0, err
	}

	adjusted, err := c.store.AdjustmentTotalByPerson(ctx, personID)
	if err != nil {
		return 0, err
	}

	return earned - spent + adjusted, nil
}

// SprintBalance is a participant's balance within one sprint:
//
//	the balance they carried in
//	  + points earned on this sprint's habits
//	  − rewards achieved at or after this sprint began
//	  + this enrolment's manual adjustments
//
// Only rewards achieved since the sprint started are subtracted, because
// anything achieved earlier was already accounted for when the carried-in
// balance was computed.
func (c *Calculator) SprintBalance(ctx context.Context, sprint domain.Sprint, participant domain.SprintParticipant) (int, error) {
	earned, err := c.store.TicksEarnedByParticipant(ctx, participant.ID)
	if err != nil {
		return 0, err
	}

	spent, err := c.store.SpentByPersonSince(ctx, participant.PersonID, SprintStartInstant(sprint))
	if err != nil {
		return 0, err
	}

	adjusted, err := c.store.AdjustmentTotalByParticipant(ctx, participant.ID)
	if err != nil {
		return 0, err
	}

	return participant.CarryForwardBalance + earned - spent + adjusted, nil
}

// SprintStartInstant is the moment a sprint began, used as the cut-off for
// "rewards achieved during this sprint". It prefers the recorded start, falls
// back to when the row was created, and finally to midnight on the start date,
// so sprints created by any earlier version of the app still compare correctly.
func SprintStartInstant(sprint domain.Sprint) time.Time {
	switch {
	case sprint.StartedAt != nil:
		return *sprint.StartedAt
	case sprint.CreatedAt != nil:
		return *sprint.CreatedAt
	default:
		return sprint.StartDate
	}
}

// SprintEndInstant is the moment a sprint stopped accepting writes, used to
// decide which achievements belong to it in the history listing.
func SprintEndInstant(sprint domain.Sprint) time.Time {
	if sprint.EndedAt != nil {
		return *sprint.EndedAt
	}

	return sprint.EndDate.Add(24*time.Hour - time.Second)
}
