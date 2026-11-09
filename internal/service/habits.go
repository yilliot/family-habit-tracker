package service

import (
	"context"
	"strings"

	"github.com/el/family-habit-tracker/internal/domain"
	"github.com/el/family-habit-tracker/internal/store"
)

// AddHabit starts a new habit for a person today. Anyone can add one at any
// time; it runs until someone stops it.
func (s *Service) AddHabit(ctx context.Context, personID int64, name string) (domain.Habit, error) {
	person, err := s.livePerson(ctx, personID)
	if err != nil {
		return domain.Habit{}, err
	}

	validator := &domain.Validator{}
	cleanName := validator.Name("name", name, domain.MaxHabitNameLen)

	if err := validator.Err(); err != nil {
		return domain.Habit{}, err
	}

	habits, err := s.store.HabitsForPerson(ctx, person.ID)
	if err != nil {
		return domain.Habit{}, err
	}

	if running := runningNamed(habits, cleanName, 0); running != nil {
		return domain.Habit{}, domain.Invalid("name", person.Name+" already has “"+running.Name+"”.")
	}

	maxOrder, err := s.store.MaxHabitDisplayOrder(ctx, person.ID)
	if err != nil {
		return domain.Habit{}, err
	}

	return s.store.CreateHabit(ctx, person.ID, cleanName, maxOrder+1, s.store.Today())
}

// RenameHabit corrects a running habit's name. Its ticks stay with it.
func (s *Service) RenameHabit(ctx context.Context, habitID int64, name string) (domain.Habit, error) {
	habit, err := s.runningHabit(ctx, habitID)
	if err != nil {
		return domain.Habit{}, err
	}

	validator := &domain.Validator{}
	cleanName := validator.Name("name", name, domain.MaxHabitNameLen)

	if err := validator.Err(); err != nil {
		return domain.Habit{}, err
	}

	habits, err := s.store.HabitsForPerson(ctx, habit.PersonID)
	if err != nil {
		return domain.Habit{}, err
	}

	if other := runningNamed(habits, cleanName, habit.ID); other != nil {
		return domain.Habit{}, domain.Invalid("name", "There is already a habit called “"+other.Name+"”.")
	}

	if err := s.store.RenameHabit(ctx, habit.ID, cleanName); err != nil {
		return domain.Habit{}, err
	}

	return s.store.Habit(ctx, habit.ID)
}

// MoveHabit moves a running habit one place up (delta −1) or down (+1) its
// person's list. Positions are renumbered 1..n as it goes, so orders inherited
// from the sprint era, which repeat across rows, never leave a tie.
func (s *Service) MoveHabit(ctx context.Context, habitID int64, delta int) error {
	habit, err := s.runningHabit(ctx, habitID)
	if err != nil {
		return err
	}

	habits, err := s.store.HabitsForPerson(ctx, habit.PersonID)
	if err != nil {
		return err
	}

	running := make([]domain.Habit, 0, len(habits))
	for _, h := range habits {
		if h.IsRunning() {
			running = append(running, h)
		}
	}

	return s.store.Tx(ctx, func(tx *store.Store) error {
		for index, h := range reorder(running, habit.ID, delta, func(h domain.Habit) int64 { return h.ID }) {
			if h.DisplayOrder == index+1 {
				continue
			}

			if err := tx.SetHabitOrder(ctx, h.ID, index+1); err != nil {
				return err
			}
		}

		return nil
	})
}

// StopHabit takes a habit off its person's list. Its ticks and the points they
// earned stay. It still shows today if it was ticked today; otherwise it ends
// yesterday. One added today by mistake, with nothing ticked, simply goes.
func (s *Service) StopHabit(ctx context.Context, habitID int64) (domain.Habit, error) {
	habit, err := s.runningHabit(ctx, habitID)
	if err != nil {
		return domain.Habit{}, err
	}

	today := s.store.Today()

	tickedToday, err := s.store.HasTick(ctx, habit.ID, today)
	if err != nil {
		return domain.Habit{}, err
	}

	if !habit.StartDate.Before(today) && !tickedToday {
		return habit, s.store.DeleteHabit(ctx, habit.ID)
	}

	end := today.AddDate(0, 0, -1)
	if tickedToday {
		end = today
	}

	if err := s.store.StopHabit(ctx, habit.ID, end); err != nil {
		return domain.Habit{}, err
	}

	return s.store.Habit(ctx, habit.ID)
}

// RestartHabit brings a stopped habit back from today. One stopped only today
// or yesterday is simply resumed, so a mistaken stop leaves no gap; an older one
// starts a new run under the same name, which history reads as one habit.
func (s *Service) RestartHabit(ctx context.Context, habitID int64) (domain.Habit, error) {
	habit, err := s.habit(ctx, habitID)
	if err != nil {
		return domain.Habit{}, err
	}

	if habit.IsRunning() {
		return domain.Habit{}, domain.Invalid("habit", "That habit is already running.")
	}

	person, err := s.livePerson(ctx, habit.PersonID)
	if err != nil {
		return domain.Habit{}, err
	}

	habits, err := s.store.HabitsForPerson(ctx, person.ID)
	if err != nil {
		return domain.Habit{}, err
	}

	if running := runningNamed(habits, habit.Name, 0); running != nil {
		return domain.Habit{}, domain.Invalid("habit", person.Name+" already has “"+running.Name+"”.")
	}

	today := s.store.Today()

	if !habit.EndDate.Before(today.AddDate(0, 0, -1)) {
		if err := s.store.ResumeHabit(ctx, habit.ID); err != nil {
			return domain.Habit{}, err
		}

		return s.store.Habit(ctx, habit.ID)
	}

	maxOrder, err := s.store.MaxHabitDisplayOrder(ctx, person.ID)
	if err != nil {
		return domain.Habit{}, err
	}

	return s.store.CreateHabit(ctx, person.ID, habit.Name, maxOrder+1, today)
}

// runningHabit loads a habit that has not been stopped, belonging to someone
// still in the family.
func (s *Service) runningHabit(ctx context.Context, habitID int64) (domain.Habit, error) {
	habit, err := s.habit(ctx, habitID)
	if err != nil {
		return domain.Habit{}, err
	}

	if !habit.IsRunning() {
		return domain.Habit{}, domain.Invalid("habit", "That habit has been stopped.")
	}

	if _, err := s.livePerson(ctx, habit.PersonID); err != nil {
		return domain.Habit{}, err
	}

	return habit, nil
}

// runningNamed finds a running habit with the given name, ignoring case and the
// habit being renamed.
func runningNamed(habits []domain.Habit, name string, except int64) *domain.Habit {
	for i := range habits {
		if habits[i].IsRunning() && habits[i].ID != except && sameName(habits[i].Name, name) {
			return &habits[i]
		}
	}

	return nil
}

// sameName compares habit names the way a person would: case and surrounding
// space do not make two habits different.
func sameName(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// reorder returns items with the one whose key is id moved delta places,
// clamped to the ends of the list.
func reorder[T any](items []T, id int64, delta int, key func(T) int64) []T {
	out := append([]T(nil), items...)

	for index := range out {
		if key(out[index]) != id {
			continue
		}

		target := min(max(index+delta, 0), len(out)-1)
		item := out[index]
		out = append(out[:index], out[index+1:]...)
		out = append(out[:target], append([]T{item}, out[target:]...)...)

		break
	}

	return out
}
