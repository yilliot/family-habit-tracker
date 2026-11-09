package service

import (
	"context"
	"errors"

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

// RenamePerson changes a family member's name.
func (s *Service) RenamePerson(ctx context.Context, id int64, name string) (domain.Person, error) {
	person, err := s.anyPerson(ctx, id)
	if err != nil {
		return domain.Person{}, err
	}

	validator := &domain.Validator{}
	cleanName := validator.Name("name", name, domain.MaxPersonNameLen)

	if err := validator.Err(); err != nil {
		return domain.Person{}, err
	}

	if err := s.store.UpdatePerson(ctx, person.ID, cleanName, person.DisplayOrder); err != nil {
		return domain.Person{}, err
	}

	return s.store.PersonWithTrashed(ctx, person.ID)
}

// MovePerson moves a family member one place up (−1) or down (+1) the list.
func (s *Service) MovePerson(ctx context.Context, id int64, delta int) error {
	person, err := s.livePerson(ctx, id)
	if err != nil {
		return err
	}

	people, err := s.store.People(ctx)
	if err != nil {
		return err
	}

	return s.store.Tx(ctx, func(tx *store.Store) error {
		for index, p := range reorder(people, person.ID, delta, func(p domain.Person) int64 { return p.ID }) {
			if p.DisplayOrder == index+1 {
				continue
			}

			if err := tx.SetPersonOrder(ctx, p.ID, index+1); err != nil {
				return err
			}
		}

		return nil
	})
}

// ArchivePerson takes a family member off the list without destroying any of
// their history. Their habits, ticks, goals and points are all kept, and
// restoring them brings everything back as it was.
func (s *Service) ArchivePerson(ctx context.Context, id int64) error {
	person, err := s.store.Person(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return domain.Invalid("person", "That person is already archived.")
	}
	if err != nil {
		return err
	}

	return s.store.SoftDeletePerson(ctx, person.ID)
}

// RestorePerson brings an archived family member back onto the list.
func (s *Service) RestorePerson(ctx context.Context, id int64) error {
	person, err := s.anyPerson(ctx, id)
	if err != nil {
		return err
	}

	if !person.IsArchived() {
		return domain.Invalid("person", person.Name+" is already on the list.")
	}

	return s.store.RestorePerson(ctx, person.ID)
}

// anyPerson loads a family member whether or not they are archived.
func (s *Service) anyPerson(ctx context.Context, id int64) (domain.Person, error) {
	person, err := s.store.PersonWithTrashed(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return domain.Person{}, domain.Invalid("person", "That person no longer exists.")
	}

	return person, err
}
