package service

import (
	"context"

	"github.com/el/family-habit-tracker/internal/domain"
	"github.com/el/family-habit-tracker/internal/store"
)

// CreateReward records a new goal for a person. Only one goal may be in flight
// at a time, so that every tick has an unambiguous destination.
func (s *Service) CreateReward(ctx context.Context, personID int64, name string, cost int) (domain.Reward, error) {
	person, err := s.store.Person(ctx, personID)
	if err != nil {
		return domain.Reward{}, err
	}

	validator := &domain.Validator{}
	cleanName := validator.Name("name", name, domain.MaxRewardNameLen)
	validator.IntRange("cost", cost, 1, domain.MaxRewardCost)

	if err := validator.Err(); err != nil {
		return domain.Reward{}, err
	}

	existing, err := s.store.ActiveRewardForPerson(ctx, person.ID)
	if err != nil {
		return domain.Reward{}, err
	}
	if existing != nil {
		return domain.Reward{}, domain.Invalid("name", "This person already has an active reward.")
	}

	return s.store.CreateReward(ctx, person.ID, cleanName, cost)
}

// UpdateReward edits a goal.
//
// An achieved goal is part of the historical record: its name may be corrected,
// but its cost is frozen, because changing it would retroactively rewrite the
// balance it was bought with. Lowering the cost of an in-flight goal re-checks
// achievement immediately, so a goal made cheaper at a family meeting is earned
// on the spot rather than on the next unrelated tick.
func (s *Service) UpdateReward(ctx context.Context, rewardID int64, name string, cost int) (domain.Reward, error) {
	reward, err := s.store.Reward(ctx, rewardID)
	if err != nil {
		return domain.Reward{}, err
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
		return domain.Reward{}, err
	}

	if reward.IsAchieved() {
		if err := s.store.UpdateReward(ctx, reward.ID, cleanName, reward.Cost); err != nil {
			return domain.Reward{}, err
		}

		return s.store.Reward(ctx, reward.ID)
	}

	err = s.store.Tx(ctx, func(tx *store.Store) error {
		txService := s.bind(tx)

		if err := tx.UpdateReward(ctx, reward.ID, cleanName, cost); err != nil {
			return err
		}

		sprint, err := tx.ActiveSprint(ctx)
		if err != nil || sprint == nil {
			return err
		}

		participant, err := tx.ParticipantWithActiveReward(ctx, sprint.ID, reward.ID)
		if err != nil || participant == nil {
			return err
		}

		_, err = txService.detectAchievement(ctx, *sprint, *participant)

		return err
	})
	if err != nil {
		return domain.Reward{}, err
	}

	return s.store.Reward(ctx, reward.ID)
}

// DeleteReward permanently removes a goal that was never earned and that nobody
// is currently working toward. Achieved goals are never deletable: they are the
// record of what the family actually celebrated.
func (s *Service) DeleteReward(ctx context.Context, rewardID int64) error {
	reward, err := s.store.Reward(ctx, rewardID)
	if err != nil {
		return err
	}

	if reward.IsAchieved() {
		return domain.Invalid("reward", "Cannot delete an achieved reward.")
	}

	sprint, err := s.store.ActiveSprint(ctx)
	if err != nil {
		return err
	}

	if sprint != nil {
		linked, err := s.store.ParticipantWithActiveReward(ctx, sprint.ID, reward.ID)
		if err != nil {
			return err
		}

		if linked != nil {
			return domain.Invalid("reward", "Cannot delete a reward currently linked to an active sprint participant.")
		}
	}

	return s.store.DeleteReward(ctx, reward.ID)
}
