package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/el/family-habit-tracker/internal/db"
	"github.com/el/family-habit-tracker/internal/domain"
)

const personColumns = `id, name, display_order, created_at, updated_at, deleted_at`

// scanPerson reads one person row from the given scanner.
func scanPerson(sc interface{ Scan(...any) error }) (domain.Person, error) {
	var (
		p                               domain.Person
		createdAt, updatedAt, deletedAt sql.NullString
	)

	if err := sc.Scan(&p.ID, &p.Name, &p.DisplayOrder, &createdAt, &updatedAt, &deletedAt); err != nil {
		return domain.Person{}, err
	}

	var err error
	if p.CreatedAt, err = db.ScanTime(createdAt); err != nil {
		return domain.Person{}, fmt.Errorf("person %d created_at: %w", p.ID, err)
	}
	if p.UpdatedAt, err = db.ScanTime(updatedAt); err != nil {
		return domain.Person{}, fmt.Errorf("person %d updated_at: %w", p.ID, err)
	}
	if p.DeletedAt, err = db.ScanTime(deletedAt); err != nil {
		return domain.Person{}, fmt.Errorf("person %d deleted_at: %w", p.ID, err)
	}

	return p, nil
}

// People lists every active family member in display order.
func (s *Store) People(ctx context.Context) ([]domain.Person, error) {
	return s.people(ctx, true)
}

// PeopleWithTrashed lists all family members including soft-deleted ones, so
// Settings can show archived members without letting them rejoin implicitly.
func (s *Store) PeopleWithTrashed(ctx context.Context) ([]domain.Person, error) {
	return s.people(ctx, false)
}

func (s *Store) people(ctx context.Context, excludeTrashed bool) ([]domain.Person, error) {
	query := `SELECT ` + personColumns + ` FROM people`
	if excludeTrashed {
		query += ` WHERE deleted_at IS NULL`
	}
	query += ` ORDER BY display_order ASC, id ASC`

	rows, err := s.q.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list people: %w", err)
	}
	defer rows.Close()

	people := make([]domain.Person, 0, 8)
	for rows.Next() {
		p, err := scanPerson(rows)
		if err != nil {
			return nil, fmt.Errorf("scan person: %w", err)
		}

		people = append(people, p)
	}

	return people, rows.Err()
}

// Person fetches one active family member. Soft-deleted people are treated as
// absent, mirroring the route-model binding they replace.
func (s *Store) Person(ctx context.Context, id int64) (domain.Person, error) {
	return s.person(ctx, id, true)
}

// PersonWithTrashed fetches one family member whether or not they are archived.
func (s *Store) PersonWithTrashed(ctx context.Context, id int64) (domain.Person, error) {
	return s.person(ctx, id, false)
}

func (s *Store) person(ctx context.Context, id int64, excludeTrashed bool) (domain.Person, error) {
	query := `SELECT ` + personColumns + ` FROM people WHERE id = ?`
	if excludeTrashed {
		query += ` AND deleted_at IS NULL`
	}

	p, err := scanPerson(s.q.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Person{}, ErrNotFound
	}
	if err != nil {
		return domain.Person{}, fmt.Errorf("get person %d: %w", id, err)
	}

	return p, nil
}

// CreatePerson inserts a family member and returns the stored row.
func (s *Store) CreatePerson(ctx context.Context, name string, displayOrder int) (domain.Person, error) {
	now := db.FormatTime(s.Now())

	res, err := s.q.ExecContext(ctx,
		`INSERT INTO people (name, display_order, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		name, displayOrder, now, now,
	)
	if err != nil {
		return domain.Person{}, fmt.Errorf("insert person: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return domain.Person{}, fmt.Errorf("insert person id: %w", err)
	}

	return s.Person(ctx, id)
}

// UpdatePerson changes a family member's name and display order.
func (s *Store) UpdatePerson(ctx context.Context, id int64, name string, displayOrder int) error {
	_, err := s.q.ExecContext(ctx,
		`UPDATE people SET name = ?, display_order = ?, updated_at = ? WHERE id = ?`,
		name, displayOrder, db.FormatTime(s.Now()), id,
	)
	if err != nil {
		return fmt.Errorf("update person %d: %w", id, err)
	}

	return nil
}

// SoftDeletePerson archives a family member, preserving all of their history.
func (s *Store) SoftDeletePerson(ctx context.Context, id int64) error {
	now := db.FormatTime(s.Now())

	_, err := s.q.ExecContext(ctx,
		`UPDATE people SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
		now, now, id,
	)
	if err != nil {
		return fmt.Errorf("soft delete person %d: %w", id, err)
	}

	return nil
}

// NextPersonDisplayOrder returns the order value to give a newly added member.
func (s *Store) NextPersonDisplayOrder(ctx context.Context) (int, error) {
	var max sql.NullInt64

	if err := s.q.QueryRowContext(ctx, `SELECT MAX(display_order) FROM people`).Scan(&max); err != nil {
		return 0, fmt.Errorf("max person display_order: %w", err)
	}

	return int(max.Int64) + 1, nil
}
