package service

import (
	"context"
	"time"

	"github.com/el/family-habit-tracker/internal/domain"
)

// PersonRewards pairs a family member with their goal catalogue.
type PersonRewards struct {
	Person  domain.Person
	Rewards []domain.Reward
}

// HasActive reports whether the person is currently pursuing a goal, which
// determines whether the "add a reward" form is offered.
func (p PersonRewards) HasActive() bool {
	for _, reward := range p.Rewards {
		if !reward.IsAchieved() {
			return true
		}
	}

	return false
}

// NextSprintParticipant pre-fills one person's row in the "start a sprint"
// form, carrying over the habits and goal they finished the last sprint with.
type NextSprintParticipant struct {
	Person       domain.Person
	Habits       []domain.Habit
	ActiveReward *domain.Reward
	// Included marks people who were in the previous sprint, so the form opens
	// with the familiar roster already ticked.
	Included bool
	// Balance is what they would carry into the new sprint.
	Balance int
}

// NextSprintDefaults seeds the "start a new sprint" form.
type NextSprintDefaults struct {
	StartDate    time.Time
	EndDate      time.Time
	Participants []NextSprintParticipant
}

// SettingsData is everything the settings page renders.
type SettingsData struct {
	People             []domain.Person
	ActiveGrid         *Grid
	AvailableForSprint []domain.Person
	RewardsByPerson    []PersonRewards
	NextSprint         NextSprintDefaults
	Today              time.Time
}

// SprintLengthDays is the default sprint length offered by the form: four
// weeks, matching the cadence of the family meeting that opens and closes one.
const SprintLengthDays = 27

// Settings assembles the settings page.
func (s *Service) Settings(ctx context.Context) (SettingsData, error) {
	today := s.store.Today()

	data := SettingsData{
		Today: today,
		NextSprint: NextSprintDefaults{
			StartDate: today,
			EndDate:   today.AddDate(0, 0, SprintLengthDays),
		},
	}

	var err error
	if data.People, err = s.store.PeopleWithTrashed(ctx); err != nil {
		return SettingsData{}, err
	}

	activeSprint, err := s.store.ActiveSprint(ctx)
	if err != nil {
		return SettingsData{}, err
	}

	if activeSprint != nil {
		grid, err := s.BuildGrid(ctx, *activeSprint)
		if err != nil {
			return SettingsData{}, err
		}

		data.ActiveGrid = &grid
	}

	livePeople, err := s.store.People(ctx)
	if err != nil {
		return SettingsData{}, err
	}

	enrolled := map[int64]bool{}
	if data.ActiveGrid != nil {
		for _, participant := range data.ActiveGrid.Participants {
			enrolled[participant.Person.ID] = true
		}
	}

	for _, person := range livePeople {
		if !enrolled[person.ID] {
			data.AvailableForSprint = append(data.AvailableForSprint, person)
		}

		rewards, err := s.store.RewardsForPerson(ctx, person.ID)
		if err != nil {
			return SettingsData{}, err
		}

		data.RewardsByPerson = append(data.RewardsByPerson, PersonRewards{Person: person, Rewards: rewards})
	}

	if data.NextSprint.Participants, err = s.nextSprintRoster(ctx, livePeople); err != nil {
		return SettingsData{}, err
	}

	return data, nil
}

// nextSprintRoster proposes the roster for the next sprint: everyone from the
// most recent sprint pre-selected with the habits they were already running,
// plus anyone else in the family listed but unselected.
func (s *Service) nextSprintRoster(ctx context.Context, livePeople []domain.Person) ([]NextSprintParticipant, error) {
	previousHabits := map[int64][]domain.Habit{}

	latest, err := s.store.LatestSprint(ctx)
	if err != nil {
		return nil, err
	}

	if latest != nil {
		participants, err := s.store.ParticipantsBySprint(ctx, latest.ID)
		if err != nil {
			return nil, err
		}

		for _, participant := range participants {
			habits, err := s.store.HabitsByParticipant(ctx, participant.ID)
			if err != nil {
				return nil, err
			}

			previousHabits[participant.PersonID] = habits
		}
	}

	roster := make([]NextSprintParticipant, 0, len(livePeople))

	for _, person := range livePeople {
		habits, wasIncluded := previousHabits[person.ID]

		reward, err := s.store.ActiveRewardForPerson(ctx, person.ID)
		if err != nil {
			return nil, err
		}

		balance, err := s.points.LifetimeBalance(ctx, person.ID)
		if err != nil {
			return nil, err
		}

		roster = append(roster, NextSprintParticipant{
			Person:       person,
			Habits:       habits,
			ActiveReward: reward,
			Included:     wasIncluded,
			Balance:      max(0, balance),
		})
	}

	return roster, nil
}
