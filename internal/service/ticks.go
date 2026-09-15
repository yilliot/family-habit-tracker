package service

import (
	"context"
	"time"

	"github.com/el/family-habit-tracker/internal/domain"
	"github.com/el/family-habit-tracker/internal/store"
)

// ToggleResult reports the outcome of tapping a grid cell.
type ToggleResult struct {
	// Ticked is the cell's state after the toggle.
	Ticked bool
	// Balance is the participant's balance after the toggle.
	Balance int
	// Participant is the enrolment the habit belongs to.
	Participant domain.SprintParticipant
	// Person owns the habit.
	Person domain.Person
	// ActiveReward is the goal being earned toward, if any.
	ActiveReward *domain.Reward
	// JustAchieved is set when this tick is the one that met the goal.
	JustAchieved *domain.Reward
}

// ToggleTick flips a single (habit, day) cell and reports the resulting state.
//
// The write is idempotent by construction: a unique index on (habit_id,
// tick_date) means a double-tap can never bank two points for one day.
func (s *Service) ToggleTick(ctx context.Context, habitID int64, day time.Time) (ToggleResult, error) {
	day = startOfDay(day)

	habit, err := s.store.Habit(ctx, habitID)
	if err != nil {
		if err == store.ErrNotFound {
			return ToggleResult{}, domain.Invalid("habit_id", "That habit no longer exists.")
		}

		return ToggleResult{}, err
	}

	if habit.SprintParticipantID == nil {
		return ToggleResult{}, domain.Invalid("habit_id", "That habit is no longer part of a sprint.")
	}

	participant, err := s.store.Participant(ctx, *habit.SprintParticipantID)
	if err != nil {
		return ToggleResult{}, err
	}

	sprint, err := s.sprintForParticipant(ctx, participant)
	if err != nil {
		return ToggleResult{}, err
	}

	if err := requireActiveSprint(sprint, "tick"); err != nil {
		return ToggleResult{}, err
	}

	if day.After(s.store.Today()) {
		return ToggleResult{}, domain.Invalid("tick_date", "Cannot tick a future day.")
	}

	if day.Before(sprint.StartDate) || day.After(sprint.EndDate) {
		return ToggleResult{}, domain.Invalid("tick_date", "Tick date is outside the sprint range.")
	}

	var result ToggleResult

	err = s.store.Tx(ctx, func(tx *store.Store) error {
		txService := s.bind(tx)

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
			achieved, err := txService.detectAchievement(ctx, sprint, participant)
			if err != nil {
				return err
			}

			result.JustAchieved = achieved
		}

		refreshed, err := tx.Participant(ctx, participant.ID)
		if err != nil {
			return err
		}

		balance, err := txService.points.SprintBalance(ctx, sprint, refreshed)
		if err != nil {
			return err
		}

		result.Participant = refreshed
		result.Balance = balance

		return nil
	})
	if err != nil {
		return ToggleResult{}, err
	}

	if result.Person, err = s.store.PersonWithTrashed(ctx, participant.PersonID); err != nil {
		return ToggleResult{}, err
	}

	if result.ActiveReward, err = s.store.RewardByIDPtr(ctx, result.Participant.ActiveRewardID); err != nil {
		return ToggleResult{}, err
	}

	return result, nil
}

// detectAchievement flips the participant's active reward to achieved when
// their balance has reached its cost, returning the reward only at the moment
// it is earned so the caller can celebrate exactly once.
func (s *Service) detectAchievement(ctx context.Context, sprint domain.Sprint, participant domain.SprintParticipant) (*domain.Reward, error) {
	reward, err := s.store.RewardByIDPtr(ctx, participant.ActiveRewardID)
	if err != nil || reward == nil || reward.IsAchieved() {
		return nil, err
	}

	balance, err := s.points.SprintBalance(ctx, sprint, participant)
	if err != nil {
		return nil, err
	}

	if balance < reward.Cost {
		return nil, nil
	}

	if err := s.store.MarkRewardAchieved(ctx, reward.ID); err != nil {
		return nil, err
	}

	achieved, err := s.store.Reward(ctx, reward.ID)
	if err != nil {
		return nil, err
	}

	return &achieved, nil
}

// CreateAdjustment records a manual, signed correction to a participant's
// balance. Balances are allowed to go negative: a deduction is a real
// consequence, not something to silently floor at zero.
func (s *Service) CreateAdjustment(ctx context.Context, participantID int64, amount int, reason *string) (domain.PointAdjustment, error) {
	participant, err := s.store.Participant(ctx, participantID)
	if err != nil {
		if err == store.ErrNotFound {
			return domain.PointAdjustment{}, domain.Invalid("participant", "That participant no longer exists.")
		}

		return domain.PointAdjustment{}, err
	}

	sprint, err := s.sprintForParticipant(ctx, participant)
	if err != nil {
		return domain.PointAdjustment{}, err
	}

	if err := requireActiveSprint(sprint, "adjust points"); err != nil {
		return domain.PointAdjustment{}, err
	}

	if amount == 0 {
		return domain.PointAdjustment{}, domain.Invalid("points", "Adjustment amount cannot be zero.")
	}

	return s.store.CreateAdjustment(ctx, participantID, amount, reason)
}

// startOfDay reduces an instant to midnight UTC, the resolution ticks use.
func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
