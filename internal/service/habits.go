package service

import (
	"context"

	"github.com/el/family-habit-tracker/internal/domain"
	"github.com/el/family-habit-tracker/internal/store"
)

// CreateHabit adds a habit to a participant in the active sprint.
func (s *Service) CreateHabit(ctx context.Context, participantID int64, name string, displayOrder *int) (domain.Habit, error) {
	participant, err := s.store.Participant(ctx, participantID)
	if err != nil {
		return domain.Habit{}, err
	}

	sprint, err := s.sprintForParticipant(ctx, participant)
	if err != nil {
		return domain.Habit{}, err
	}

	if err := requireActiveSprint(sprint, "add habits"); err != nil {
		return domain.Habit{}, err
	}

	validator := &domain.Validator{}
	cleanName := validator.Name("name", name, domain.MaxHabitNameLen)

	if err := validator.Err(); err != nil {
		return domain.Habit{}, err
	}

	order := 0
	if displayOrder != nil {
		order = *displayOrder
	} else {
		maxOrder, err := s.store.MaxHabitDisplayOrder(ctx, participant.ID)
		if err != nil {
			return domain.Habit{}, err
		}

		order = maxOrder + 1
	}

	return s.store.CreateHabit(ctx, participant.ID, cleanName, order)
}

// UpdateHabit renames a habit inside the active sprint.
func (s *Service) UpdateHabit(ctx context.Context, habitID int64, name string, displayOrder int) (domain.Habit, error) {
	habit, sprint, err := s.habitWithSprint(ctx, habitID)
	if err != nil {
		return domain.Habit{}, err
	}

	if err := requireActiveSprint(sprint, "edit habits"); err != nil {
		return domain.Habit{}, err
	}

	validator := &domain.Validator{}
	cleanName := validator.Name("name", name, domain.MaxHabitNameLen)

	if displayOrder < 0 {
		validator.Add("display_order", "Order cannot be negative.")
	}

	if err := validator.Err(); err != nil {
		return domain.Habit{}, err
	}

	if err := s.store.UpdateHabit(ctx, habit.ID, cleanName, displayOrder); err != nil {
		return domain.Habit{}, err
	}

	return s.store.Habit(ctx, habit.ID)
}

// ArchiveHabit drops a habit from the grid. Its ticks are kept, so the points
// already earned on it stay in the person's totals.
func (s *Service) ArchiveHabit(ctx context.Context, habitID int64) error {
	habit, sprint, err := s.habitWithSprint(ctx, habitID)
	if err != nil {
		return err
	}

	if err := requireActiveSprint(sprint, "delete habits"); err != nil {
		return err
	}

	return s.store.SoftDeleteHabit(ctx, habit.ID)
}

// habitWithSprint loads a habit together with the sprint that governs it.
func (s *Service) habitWithSprint(ctx context.Context, habitID int64) (domain.Habit, domain.Sprint, error) {
	habit, err := s.store.Habit(ctx, habitID)
	if err != nil {
		return domain.Habit{}, domain.Sprint{}, err
	}

	if habit.SprintParticipantID == nil {
		return domain.Habit{}, domain.Sprint{}, domain.Invalid("habit", "That habit is no longer part of a sprint.")
	}

	participant, err := s.store.Participant(ctx, *habit.SprintParticipantID)
	if err != nil {
		if err == store.ErrNotFound {
			return domain.Habit{}, domain.Sprint{}, domain.Invalid("habit", "That habit is no longer part of a sprint.")
		}

		return domain.Habit{}, domain.Sprint{}, err
	}

	sprint, err := s.sprintForParticipant(ctx, participant)
	if err != nil {
		return domain.Habit{}, domain.Sprint{}, err
	}

	return habit, sprint, nil
}
