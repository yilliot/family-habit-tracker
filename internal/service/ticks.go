package service

import (
	"context"
	"time"

	"github.com/el/family-habit-tracker/internal/domain"
	"github.com/el/family-habit-tracker/internal/store"
)

// ToggleResult reports the outcome of tapping a habit.
type ToggleResult struct {
	// Ticked is the habit's state on that day after the toggle.
	Ticked bool
	// Person owns the habit.
	Person domain.Person
	// JustAchieved is set when this tick is the one that met the goal.
	JustAchieved *domain.Reward
}

// ToggleTick flips one habit on one day and reports the resulting state.
//
// The write is idempotent by construction: a unique index on (habit_id,
// tick_date) means a double-tap can never bank two points for one day.
func (s *Service) ToggleTick(ctx context.Context, habitID int64, day time.Time) (ToggleResult, error) {
	day = startOfDay(day)
	today := s.store.Today()

	habit, err := s.habit(ctx, habitID)
	if err != nil {
		return ToggleResult{}, err
	}

	person, err := s.livePerson(ctx, habit.PersonID)
	if err != nil {
		return ToggleResult{}, err
	}

	switch {
	case day.After(today):
		return ToggleResult{}, domain.Invalid("tick_date", "Cannot tick a future day.")
	case !isEditable(day, today):
		return ToggleResult{}, domain.Invalid("tick_date", "Days more than a week ago are kept as history and can't be changed.")
	case !habit.LiveOn(day):
		return ToggleResult{}, domain.Invalid("tick_date", "That habit wasn't running on that day.")
	}

	result := ToggleResult{Person: person}

	err = s.store.Tx(ctx, func(tx *store.Store) error {
		existed, err := tx.HasTick(ctx, habit.ID, day)
		if err != nil {
			return err
		}

		if existed {
			if err := tx.DeleteTick(ctx, habit.ID, day); err != nil {
				return err
			}
		} else if err := tx.InsertTick(ctx, habit.ID, day); err != nil {
			return err
		}

		result.Ticked = !existed

		// Reaching a goal is only ever checked on the way up. Un-ticking a day
		// never revokes a reward that has already been celebrated.
		if result.Ticked {
			result.JustAchieved, err = s.bind(tx).detectAchievement(ctx, person.ID)
		}

		return err
	})
	if err != nil {
		return ToggleResult{}, err
	}

	return result, nil
}

// detectAchievement marks the person's goal achieved when their balance covers
// its cost, returning the goal only at the moment it is earned so the caller
// can celebrate exactly once.
func (s *Service) detectAchievement(ctx context.Context, personID int64) (*domain.Reward, error) {
	goal, err := s.store.ActiveRewardForPerson(ctx, personID)
	if err != nil || goal == nil {
		return nil, err
	}

	balance, err := s.points.Balance(ctx, personID)
	if err != nil || balance < goal.Cost {
		return nil, err
	}

	if err := s.store.MarkRewardAchieved(ctx, goal.ID); err != nil {
		return nil, err
	}

	achieved, err := s.store.Reward(ctx, goal.ID)
	if err != nil {
		return nil, err
	}

	return &achieved, nil
}

// CreateAdjustment records a manual, signed correction to a person's balance.
// Balances are allowed to go negative: a deduction is a real consequence, not
// something to silently floor at zero.
func (s *Service) CreateAdjustment(ctx context.Context, personID int64, amount int, reason *string) (domain.PointAdjustment, error) {
	if _, err := s.livePerson(ctx, personID); err != nil {
		return domain.PointAdjustment{}, err
	}

	if amount == 0 {
		return domain.PointAdjustment{}, domain.Invalid("points", "Adjustment amount cannot be zero.")
	}

	return s.store.CreateAdjustment(ctx, personID, amount, reason)
}
