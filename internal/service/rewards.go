package service

import (
	"context"

	"github.com/el/family-habit-tracker/internal/domain"
	"github.com/el/family-habit-tracker/internal/store"
)

// SetGoal records a new goal for a person. Only one goal may be in flight at a
// time, so every tick has an unambiguous destination.
//
// It returns the goal as achieved when the person's points already cover it:
// picking a goal you can afford earns it on the spot, rather than waiting for
// the next unrelated tick to notice.
func (s *Service) SetGoal(ctx context.Context, personID int64, name string, cost int) (domain.Reward, *domain.Reward, error) {
	person, err := s.livePerson(ctx, personID)
	if err != nil {
		return domain.Reward{}, nil, err
	}

	validator := &domain.Validator{}
	cleanName := validator.Name("name", name, domain.MaxRewardNameLen)
	validator.IntRange("cost", cost, 1, domain.MaxRewardCost)

	if err := validator.Err(); err != nil {
		return domain.Reward{}, nil, err
	}

	var (
		goal     domain.Reward
		achieved *domain.Reward
	)

	err = s.store.Tx(ctx, func(tx *store.Store) error {
		existing, err := tx.ActiveRewardForPerson(ctx, person.ID)
		if err != nil {
			return err
		}
		if existing != nil {
			return domain.Invalid("name", person.Name+" is already saving for "+existing.Name+".")
		}

		if goal, err = tx.CreateReward(ctx, person.ID, cleanName, cost); err != nil {
			return err
		}

		achieved, err = s.bind(tx).detectAchievement(ctx, person.ID)

		return err
	})
	if err != nil {
		return domain.Reward{}, nil, err
	}

	return goal, achieved, nil
}

// UpdateReward edits a goal.
//
// An achieved goal is part of the historical record: its name may be corrected,
// but its cost is frozen, because changing it would retroactively rewrite the
// balance it was bought with. Lowering the cost of an in-flight goal re-checks
// achievement immediately, so a goal made cheaper is earned on the spot.
func (s *Service) UpdateReward(ctx context.Context, rewardID int64, name string, cost int) (domain.Reward, *domain.Reward, error) {
	reward, err := s.store.Reward(ctx, rewardID)
	if err != nil {
		return domain.Reward{}, nil, err
	}

	validator := &domain.Validator{}
	cleanName := validator.Name("name", name, domain.MaxRewardNameLen)

	if reward.IsAchieved() {
		if cost != reward.Cost {
			validator.Add("cost", "Cannot edit cost on an achieved reward.")
		}
	} else {
		validator.IntRange("cost", cost, 1, domain.MaxRewardCost)
	}

	if err := validator.Err(); err != nil {
		return domain.Reward{}, nil, err
	}

	if reward.IsAchieved() {
		if err := s.store.UpdateReward(ctx, reward.ID, cleanName, reward.Cost); err != nil {
			return domain.Reward{}, nil, err
		}

		updated, err := s.store.Reward(ctx, reward.ID)

		return updated, nil, err
	}

	var achieved *domain.Reward

	err = s.store.Tx(ctx, func(tx *store.Store) error {
		if err := tx.UpdateReward(ctx, reward.ID, cleanName, cost); err != nil {
			return err
		}

		achieved, err = s.bind(tx).detectAchievement(ctx, reward.PersonID)

		return err
	})
	if err != nil {
		return domain.Reward{}, nil, err
	}

	updated, err := s.store.Reward(ctx, reward.ID)

	return updated, achieved, err
}

// DeleteReward drops a goal that was never earned. Achieved goals are never
// deletable: they are the record of what the family actually celebrated.
func (s *Service) DeleteReward(ctx context.Context, rewardID int64) (domain.Reward, error) {
	reward, err := s.store.Reward(ctx, rewardID)
	if err != nil {
		return domain.Reward{}, err
	}

	if reward.IsAchieved() {
		return domain.Reward{}, domain.Invalid("reward", "Cannot delete an achieved reward.")
	}

	return reward, s.store.DeleteReward(ctx, reward.ID)
}
