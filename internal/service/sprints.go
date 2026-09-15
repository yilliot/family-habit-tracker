package service

import (
	"context"
	"time"

	"github.com/el/family-habit-tracker/internal/domain"
	"github.com/el/family-habit-tracker/internal/store"
)

// Reward selection modes offered when enrolling someone in a sprint.
const (
	// RewardKeepCurrent carries over the person's existing un-achieved goal.
	RewardKeepCurrent = "keep_current"
	// RewardNew creates a fresh goal from the supplied name and cost.
	RewardNew = "new"
	// RewardNone enrols the person with no goal; their points accrue as surplus.
	RewardNone = "none"
)

// HabitInput is one habit to create for a participant.
type HabitInput struct {
	Name         string
	DisplayOrder int
}

// RewardChoice is the goal decision made for a participant at enrolment.
type RewardChoice struct {
	Mode string
	Name string
	Cost int
}

// ParticipantInput is one person's configuration within a new sprint.
type ParticipantInput struct {
	PersonID int64
	Habits   []HabitInput
	Reward   RewardChoice
}

// StartSprint opens a sprint and enrols the given roster.
//
// Each participant's opening balance is their lifetime balance, floored at
// zero, which is what lets a 250-point goal be saved toward across several
// sprints without any per-sprint bookkeeping.
func (s *Service) StartSprint(ctx context.Context, start, end time.Time, participants []ParticipantInput) (domain.Sprint, error) {
	start, end = startOfDay(start), startOfDay(end)

	if end.Before(start) {
		return domain.Sprint{}, domain.Invalid("end_date", "End date must be on or after start date.")
	}

	if len(participants) == 0 {
		return domain.Sprint{}, domain.Invalid("participants", "Pick at least one person for the sprint.")
	}

	active, err := s.store.HasActiveSprint(ctx)
	if err != nil {
		return domain.Sprint{}, err
	}
	if active {
		return domain.Sprint{}, domain.Invalid("sprint", "There is already an active sprint. End it before starting a new one.")
	}

	var created domain.Sprint

	err = s.store.Tx(ctx, func(tx *store.Store) error {
		txService := s.bind(tx)

		sprint, err := tx.CreateSprint(ctx, start, end)
		if err != nil {
			return err
		}

		for index, input := range participants {
			person, err := tx.Person(ctx, input.PersonID)
			if err != nil {
				if err == store.ErrNotFound {
					return domain.Invalid("participants", "One of the selected people no longer exists.")
				}

				return err
			}

			if _, err := txService.enrol(ctx, sprint, person, input, index+1); err != nil {
				return err
			}
		}

		created = sprint

		return nil
	})
	if err != nil {
		return domain.Sprint{}, err
	}

	return created, nil
}

// AddParticipant enrols someone in a sprint that is already under way. They
// start from their own lifetime balance, so joining late costs them nothing
// they had already earned.
func (s *Service) AddParticipant(ctx context.Context, sprintID int64, input ParticipantInput) (domain.SprintParticipant, error) {
	sprint, err := s.store.Sprint(ctx, sprintID)
	if err != nil {
		return domain.SprintParticipant{}, err
	}

	if err := requireActiveSprint(sprint, "add participants"); err != nil {
		return domain.SprintParticipant{}, err
	}

	person, err := s.store.Person(ctx, input.PersonID)
	if err != nil {
		if err == store.ErrNotFound {
			return domain.SprintParticipant{}, domain.Invalid("person_id", "That person no longer exists.")
		}

		return domain.SprintParticipant{}, err
	}

	existing, err := s.store.ParticipantForPerson(ctx, sprint.ID, person.ID)
	if err != nil {
		return domain.SprintParticipant{}, err
	}
	if existing != nil {
		return domain.SprintParticipant{}, domain.Invalid("person_id", "This person is already a participant in the sprint.")
	}

	var participant domain.SprintParticipant

	err = s.store.Tx(ctx, func(tx *store.Store) error {
		txService := s.bind(tx)

		maxOrder, err := tx.MaxParticipantDisplayOrder(ctx, sprint.ID)
		if err != nil {
			return err
		}

		participant, err = txService.enrol(ctx, sprint, person, input, maxOrder+1)

		return err
	})
	if err != nil {
		return domain.SprintParticipant{}, err
	}

	return participant, nil
}

// enrol creates one participant row plus its habits. It must run inside a
// transaction.
func (s *Service) enrol(ctx context.Context, sprint domain.Sprint, person domain.Person, input ParticipantInput, fallbackOrder int) (domain.SprintParticipant, error) {
	reward, err := s.resolveReward(ctx, person, input.Reward)
	if err != nil {
		return domain.SprintParticipant{}, err
	}

	lifetime, err := s.points.LifetimeBalance(ctx, person.ID)
	if err != nil {
		return domain.SprintParticipant{}, err
	}

	carryForward := max(0, lifetime)

	displayOrder := person.DisplayOrder
	if displayOrder == 0 {
		displayOrder = fallbackOrder
	}

	var rewardID *int64
	if reward != nil {
		rewardID = &reward.ID
	}

	participant, err := s.store.CreateParticipant(ctx, sprint.ID, person.ID, carryForward, rewardID, displayOrder)
	if err != nil {
		return domain.SprintParticipant{}, err
	}

	for index, habit := range input.Habits {
		order := habit.DisplayOrder
		if order == 0 {
			order = index + 1
		}

		if _, err := s.store.CreateHabit(ctx, participant.ID, habit.Name, order); err != nil {
			return domain.SprintParticipant{}, err
		}
	}

	return participant, nil
}

// resolveReward turns a reward choice into the goal to link.
//
// Unlike the implementation this replaces, choosing "none" genuinely enrols the
// person with no goal rather than silently re-linking an existing one. The
// label now matches the behaviour.
func (s *Service) resolveReward(ctx context.Context, person domain.Person, choice RewardChoice) (*domain.Reward, error) {
	existing, err := s.store.ActiveRewardForPerson(ctx, person.ID)
	if err != nil {
		return nil, err
	}

	switch choice.Mode {
	case RewardNone:
		return nil, nil

	case RewardNew:
		// A person may only pursue one goal at a time, so an un-achieved goal
		// always wins over a newly requested one.
		if existing != nil {
			return existing, nil
		}

		validator := &domain.Validator{}
		name := validator.Name("reward_name", choice.Name, domain.MaxRewardNameLen)
		validator.IntRange("reward_cost", choice.Cost, 1, domain.MaxRewardCost)

		if err := validator.Err(); err != nil {
			return nil, err
		}

		created, err := s.store.CreateReward(ctx, person.ID, name, choice.Cost)
		if err != nil {
			return nil, err
		}

		return &created, nil

	default:
		return existing, nil
	}
}

// EndSprint archives a sprint, freezing it as a permanent record. Unspent
// points are not touched here; they surface again as the opening balance of
// whatever sprint the family starts next.
func (s *Service) EndSprint(ctx context.Context, sprintID int64) (domain.Sprint, error) {
	sprint, err := s.store.Sprint(ctx, sprintID)
	if err != nil {
		return domain.Sprint{}, err
	}

	if sprint.IsArchived() {
		return domain.Sprint{}, domain.Invalid("sprint", "Sprint is already archived.")
	}

	if err := s.store.ArchiveSprint(ctx, sprint.ID); err != nil {
		return domain.Sprint{}, err
	}

	return s.store.Sprint(ctx, sprint.ID)
}

// SetActiveReward points a participant at a different goal, or clears it.
//
// It returns the reward when the participant's existing surplus already covers
// the newly chosen goal. The implementation this replaces left such a goal
// sitting unachieved until the next unrelated tick happened to trigger a check;
// here, picking a goal you can already afford earns it immediately.
func (s *Service) SetActiveReward(ctx context.Context, participantID int64, rewardID *int64) (*domain.Reward, error) {
	participant, err := s.store.Participant(ctx, participantID)
	if err != nil {
		return nil, err
	}

	sprint, err := s.sprintForParticipant(ctx, participant)
	if err != nil {
		return nil, err
	}

	if err := requireActiveSprint(sprint, "edit participants"); err != nil {
		return nil, err
	}

	if rewardID == nil {
		return nil, s.store.SetParticipantActiveReward(ctx, participant.ID, nil)
	}

	reward, err := s.store.Reward(ctx, *rewardID)
	if err != nil {
		if err == store.ErrNotFound {
			return nil, domain.Invalid("reward_id", "That reward no longer exists.")
		}

		return nil, err
	}

	if reward.PersonID != participant.PersonID {
		return nil, domain.Invalid("reward_id", "Reward does not belong to this participant.")
	}

	if reward.IsAchieved() {
		return nil, domain.Invalid("reward_id", "Cannot link an achieved reward.")
	}

	if err := s.store.SetParticipantActiveReward(ctx, participant.ID, &reward.ID); err != nil {
		return nil, err
	}

	refreshed, err := s.store.Participant(ctx, participant.ID)
	if err != nil {
		return nil, err
	}

	return s.detectAchievement(ctx, sprint, refreshed)
}
