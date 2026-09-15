package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/el/family-habit-tracker/internal/db"
	"github.com/el/family-habit-tracker/internal/domain"
)

const habitColumns = `id, sprint_participant_id, name, display_order, created_at, updated_at, deleted_at`

func scanHabit(sc interface{ Scan(...any) error }) (domain.Habit, error) {
	var (
		h                               domain.Habit
		participantID                   sql.NullInt64
		createdAt, updatedAt, deletedAt sql.NullString
	)

	err := sc.Scan(&h.ID, &participantID, &h.Name, &h.DisplayOrder, &createdAt, &updatedAt, &deletedAt)
	if err != nil {
		return domain.Habit{}, err
	}

	h.SprintParticipantID = toInt64Ptr(participantID)

	if h.CreatedAt, err = db.ScanTime(createdAt); err != nil {
		return domain.Habit{}, fmt.Errorf("habit %d created_at: %w", h.ID, err)
	}
	if h.UpdatedAt, err = db.ScanTime(updatedAt); err != nil {
		return domain.Habit{}, fmt.Errorf("habit %d updated_at: %w", h.ID, err)
	}
	if h.DeletedAt, err = db.ScanTime(deletedAt); err != nil {
		return domain.Habit{}, fmt.Errorf("habit %d deleted_at: %w", h.ID, err)
	}

	return h, nil
}

// Habit fetches one live habit. Soft-deleted habits are treated as absent so a
// removed habit cannot be ticked through a stale page.
func (s *Store) Habit(ctx context.Context, id int64) (domain.Habit, error) {
	h, err := scanHabit(s.q.QueryRowContext(ctx,
		`SELECT `+habitColumns+` FROM habits WHERE id = ? AND deleted_at IS NULL`, id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Habit{}, ErrNotFound
	}
	if err != nil {
		return domain.Habit{}, fmt.Errorf("get habit %d: %w", id, err)
	}

	return h, nil
}

// HabitsByParticipant lists a participant's live habits in display order.
func (s *Store) HabitsByParticipant(ctx context.Context, participantID int64) ([]domain.Habit, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+habitColumns+` FROM habits
		   WHERE sprint_participant_id = ? AND deleted_at IS NULL
		   ORDER BY display_order ASC, id ASC`,
		participantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list habits for participant %d: %w", participantID, err)
	}
	defer rows.Close()

	habits := make([]domain.Habit, 0, 6)
	for rows.Next() {
		h, err := scanHabit(rows)
		if err != nil {
			return nil, fmt.Errorf("scan habit: %w", err)
		}

		habits = append(habits, h)
	}

	return habits, rows.Err()
}

// CreateHabit adds a habit to a participant.
func (s *Store) CreateHabit(ctx context.Context, participantID int64, name string, displayOrder int) (domain.Habit, error) {
	now := db.FormatTime(s.Now())

	res, err := s.q.ExecContext(ctx,
		`INSERT INTO habits (sprint_participant_id, name, display_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		participantID, name, displayOrder, now, now,
	)
	if err != nil {
		return domain.Habit{}, fmt.Errorf("insert habit: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return domain.Habit{}, fmt.Errorf("insert habit id: %w", err)
	}

	return s.Habit(ctx, id)
}

// UpdateHabit renames a habit and sets its position.
func (s *Store) UpdateHabit(ctx context.Context, id int64, name string, displayOrder int) error {
	_, err := s.q.ExecContext(ctx,
		`UPDATE habits SET name = ?, display_order = ?, updated_at = ? WHERE id = ?`,
		name, displayOrder, db.FormatTime(s.Now()), id,
	)
	if err != nil {
		return fmt.Errorf("update habit %d: %w", id, err)
	}

	return nil
}

// SoftDeleteHabit removes a habit from the grid while keeping its ticks, which
// continue to count toward lifetime totals.
func (s *Store) SoftDeleteHabit(ctx context.Context, id int64) error {
	now := db.FormatTime(s.Now())

	_, err := s.q.ExecContext(ctx,
		`UPDATE habits SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
		now, now, id,
	)
	if err != nil {
		return fmt.Errorf("soft delete habit %d: %w", id, err)
	}

	return nil
}

// SoftDeleteHabitsForParticipant archives every habit belonging to a
// participant, used when a person leaves a sprint mid-flight.
func (s *Store) SoftDeleteHabitsForParticipant(ctx context.Context, participantID int64) error {
	now := db.FormatTime(s.Now())

	_, err := s.q.ExecContext(ctx,
		`UPDATE habits SET deleted_at = ?, updated_at = ?
		   WHERE sprint_participant_id = ? AND deleted_at IS NULL`,
		now, now, participantID,
	)
	if err != nil {
		return fmt.Errorf("soft delete habits for participant %d: %w", participantID, err)
	}

	return nil
}

// MaxHabitDisplayOrder returns the highest order value among a participant's
// habits.
func (s *Store) MaxHabitDisplayOrder(ctx context.Context, participantID int64) (int, error) {
	var max sql.NullInt64

	err := s.q.QueryRowContext(ctx,
		`SELECT MAX(display_order) FROM habits WHERE sprint_participant_id = ? AND deleted_at IS NULL`,
		participantID,
	).Scan(&max)
	if err != nil {
		return 0, fmt.Errorf("max habit display_order: %w", err)
	}

	return int(max.Int64), nil
}

// TickDatesByParticipant returns, for each of a participant's live habits, the
// set of dates that habit is ticked on. One query serves a whole grid row,
// keeping the page render free of per-habit round trips.
func (s *Store) TickDatesByParticipant(ctx context.Context, participantID int64) (map[int64]map[string]bool, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT t.habit_id, t.tick_date
		   FROM ticks t
		   JOIN habits h ON h.id = t.habit_id
		  WHERE h.sprint_participant_id = ? AND h.deleted_at IS NULL`,
		participantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list ticks for participant %d: %w", participantID, err)
	}
	defer rows.Close()

	byHabit := make(map[int64]map[string]bool)
	for rows.Next() {
		var (
			habitID  int64
			tickDate sql.NullString
		)

		if err := rows.Scan(&habitID, &tickDate); err != nil {
			return nil, fmt.Errorf("scan tick: %w", err)
		}

		parsed, err := db.ScanDate(tickDate)
		if err != nil {
			return nil, fmt.Errorf("tick date on habit %d: %w", habitID, err)
		}

		if byHabit[habitID] == nil {
			byHabit[habitID] = make(map[string]bool)
		}

		byHabit[habitID][db.FormatDate(parsed)] = true
	}

	return byHabit, rows.Err()
}

// HasTick reports whether a habit is already ticked on a day.
func (s *Store) HasTick(ctx context.Context, habitID int64, day time.Time) (bool, error) {
	var count int

	err := s.q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ticks WHERE habit_id = ? AND date(tick_date) = ?`,
		habitID, db.FormatDate(day),
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check tick on habit %d: %w", habitID, err)
	}

	return count > 0, nil
}

// InsertTick records a completed day.
func (s *Store) InsertTick(ctx context.Context, habitID int64, day time.Time) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO ticks (habit_id, tick_date, created_at) VALUES (?, ?, ?)`,
		habitID, db.FormatDate(day), db.FormatTime(s.Now()),
	)
	if err != nil {
		return fmt.Errorf("insert tick on habit %d: %w", habitID, err)
	}

	return nil
}

// DeleteTick clears a completed day. Matching on date() rather than the raw
// string means a row written in either legacy date format is still found.
func (s *Store) DeleteTick(ctx context.Context, habitID int64, day time.Time) error {
	_, err := s.q.ExecContext(ctx,
		`DELETE FROM ticks WHERE habit_id = ? AND date(tick_date) = ?`,
		habitID, db.FormatDate(day),
	)
	if err != nil {
		return fmt.Errorf("delete tick on habit %d: %w", habitID, err)
	}

	return nil
}
