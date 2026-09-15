package service

import (
	"context"

	"github.com/el/family-habit-tracker/internal/domain"
	"github.com/el/family-habit-tracker/internal/store"
)

// CreatePerson adds a family member. Display order defaults to the end of the
// list when not supplied.
func (s *Service) CreatePerson(ctx context.Context, name string, displayOrder *int) (domain.Person, error) {
	validator := &domain.Validator{}
	cleanName := validator.Name("name", name, domain.MaxPersonNameLen)

	if err := validator.Err(); err != nil {
		return domain.Person{}, err
	}

	order := 0
	if displayOrder != nil {
		order = *displayOrder
	} else {
		next, err := s.store.NextPersonDisplayOrder(ctx)
		if err != nil {
			return domain.Person{}, err
		}

		order = next
	}

	return s.store.CreatePerson(ctx, cleanName, order)
}

// UpdatePerson renames a family member and sets their position in the ledger.
func (s *Service) UpdatePerson(ctx context.Context, id int64, name string, displayOrder int) (domain.Person, error) {
	person, err := s.store.PersonWithTrashed(ctx, id)
	if err != nil {
		return domain.Person{}, err
	}

	validator := &domain.Validator{}
	cleanName := validator.Name("name", name, domain.MaxPersonNameLen)

	if displayOrder < 0 {
		validator.Add("display_order", "Order cannot be negative.")
	}

	if err := validator.Err(); err != nil {
		return domain.Person{}, err
	}

	if err := s.store.UpdatePerson(ctx, person.ID, cleanName, displayOrder); err != nil {
		return domain.Person{}, err
	}

	return s.store.PersonWithTrashed(ctx, person.ID)
}

// ArchivePerson removes a family member from the ledger without destroying any
// of their history.
//
// If they are mid-sprint their habits are archived, which takes them off the
// grid, but the enrolment row itself is kept. That row is the only link from a
// tick back to a person: ticks belong to habits, and habits reach a person only
// through it. Deleting it would set habits.sprint_participant_id to NULL and
// silently orphan every point they had earned, which is exactly what the
// implementation this replaces did — it archived the habits to "preserve the
// ticks", then dropped the row that made those ticks findable.
//
// Keeping the row costs nothing: an archived person is filtered out of the
// active grid, so they disappear from the tablet either way.
func (s *Service) ArchivePerson(ctx context.Context, id int64) error {
	person, err := s.store.Person(ctx, id)
	if err != nil {
		if err == store.ErrNotFound {
			return domain.Invalid("person", "That person is already archived.")
		}

		return err
	}

	return s.store.Tx(ctx, func(tx *store.Store) error {
		sprint, err := tx.ActiveSprint(ctx)
		if err != nil {
			return err
		}

		if sprint != nil {
			participant, err := tx.ParticipantForPerson(ctx, sprint.ID, person.ID)
			if err != nil {
				return err
			}

			if participant != nil {
				if err := tx.SoftDeleteHabitsForParticipant(ctx, participant.ID); err != nil {
					return err
				}
			}
		}

		return tx.SoftDeletePerson(ctx, person.ID)
	})
}
