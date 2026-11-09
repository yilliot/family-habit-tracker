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

const habitColumns = `id, person_id, sprint_participant_id, name, display_order, start_date, end_date, created_at, updated_at, deleted_at`

func scanHabit(sc interface{ Scan(...any) error }) (domain.Habit, error) {
	var (
		h                               domain.Habit
		personID, participantID         sql.NullInt64
		startDate, endDate              sql.NullString
		createdAt, updatedAt, deletedAt sql.NullString
	)

	err := sc.Scan(&h.ID, &personID, &participantID, &h.Name, &h.DisplayOrder, &startDate, &endDate, &createdAt, &updatedAt, &deletedAt)
	if err != nil {
		return domain.Habit{}, err
	}

	h.PersonID = personID.Int64
	h.SprintParticipantID = toInt64Ptr(participantID)

	if h.StartDate, err = db.ScanDate(startDate); err != nil {
		return domain.Habit{}, fmt.Errorf("habit %d start_date: %w", h.ID, err)
	}
	if endDate.Valid && endDate.String != "" {
		end, err := db.ScanDate(endDate)
		if err != nil {
			return domain.Habit{}, fmt.Errorf("habit %d end_date: %w", h.ID, err)
		}

		h.EndDate = &end
	}
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

// Habit fetches one habit, running or stopped.
func (s *Store) Habit(ctx context.Context, id int64) (domain.Habit, error) {
	h, err := scanHabit(s.q.QueryRowContext(ctx,
		`SELECT `+habitColumns+` FROM habits WHERE id = ? AND person_id IS NOT NULL`, id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Habit{}, ErrNotFound
	}
	if err != nil {
		return domain.Habit{}, fmt.Errorf("get habit %d: %w", id, err)
	}

	return h, nil
}

// HabitsForPerson lists every habit a person has ever had, running and
// stopped, in display order. A family adds a few a year, so callers filter by
// date in Go rather than in SQL.
func (s *Store) HabitsForPerson(ctx context.Context, personID int64) ([]domain.Habit, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+habitColumns+` FROM habits
		  WHERE person_id = ?
		  ORDER BY display_order ASC, id ASC`,
		personID,
	)
	if err != nil {
		return nil, fmt.Errorf("list habits for person %d: %w", personID, err)
	}
	defer rows.Close()

	habits := make([]domain.Habit, 0, 16)
	for rows.Next() {
		h, err := scanHabit(rows)
		if err != nil {
			return nil, fmt.Errorf("scan habit: %w", err)
		}

		habits = append(habits, h)
	}

	return habits, rows.Err()
}

// CreateHabit starts a habit for a person on a day.
func (s *Store) CreateHabit(ctx context.Context, personID int64, name string, displayOrder int, start time.Time) (domain.Habit, error) {
	now := db.FormatTime(s.Now())

	res, err := s.q.ExecContext(ctx,
		`INSERT INTO habits (person_id, name, display_order, start_date, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		personID, name, displayOrder, db.FormatDate(start), now, now,
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

// RenameHabit changes a habit's name.
func (s *Store) RenameHabit(ctx context.Context, id int64, name string) error {
	_, err := s.q.ExecContext(ctx,
		`UPDATE habits SET name = ?, updated_at = ? WHERE id = ?`,
		name, db.FormatTime(s.Now()), id,
	)
	if err != nil {
		return fmt.Errorf("rename habit %d: %w", id, err)
	}

	return nil
}

// SetHabitOrder sets a habit's position in its person's list.
func (s *Store) SetHabitOrder(ctx context.Context, id int64, displayOrder int) error {
	_, err := s.q.ExecContext(ctx,
		`UPDATE habits SET display_order = ?, updated_at = ? WHERE id = ?`,
		displayOrder, db.FormatTime(s.Now()), id,
	)
	if err != nil {
		return fmt.Errorf("order habit %d: %w", id, err)
	}

	return nil
}

// StopHabit ends a habit; end is the last day it counts as running.
func (s *Store) StopHabit(ctx context.Context, id int64, end time.Time) error {
	_, err := s.q.ExecContext(ctx,
		`UPDATE habits SET end_date = ?, updated_at = ? WHERE id = ? AND end_date IS NULL`,
		db.FormatDate(end), db.FormatTime(s.Now()), id,
	)
	if err != nil {
		return fmt.Errorf("stop habit %d: %w", id, err)
	}

	return nil
}

// DeleteHabit removes a habit outright. Only used for one added by mistake that
// has no ticks, so there is nothing to keep.
func (s *Store) DeleteHabit(ctx context.Context, id int64) error {
	if _, err := s.q.ExecContext(ctx, `DELETE FROM habits WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete habit %d: %w", id, err)
	}

	return nil
}

// MaxHabitDisplayOrder returns the highest order value among a person's habits.
func (s *Store) MaxHabitDisplayOrder(ctx context.Context, personID int64) (int, error) {
	var max sql.NullInt64

	err := s.q.QueryRowContext(ctx,
		`SELECT MAX(display_order) FROM habits WHERE person_id = ?`, personID,
	).Scan(&max)
	if err != nil {
		return 0, fmt.Errorf("max habit display_order: %w", err)
	}

	return int(max.Int64), nil
}

// DayTick is one tick, reduced to the habit and the day.
type DayTick struct {
	HabitID int64
	Day     time.Time
}

// TicksForPerson returns a person's ticks on the days from..to, inclusive.
//
// tick_date holds either "2026-08-24" or, on rows the Laravel app wrote,
// "2026-08-24 00:00:00". Comparing as text against a half-open range catches
// both: the longer form sorts after the plain day and before the next one.
func (s *Store) TicksForPerson(ctx context.Context, personID int64, from, to time.Time) ([]DayTick, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT t.habit_id, t.tick_date
		   FROM habits h
		   JOIN ticks t ON t.habit_id = h.id
		  WHERE h.person_id = ? AND t.tick_date >= ? AND t.tick_date < ?`,
		personID, db.FormatDate(from), db.FormatDate(to.AddDate(0, 0, 1)),
	)
	if err != nil {
		return nil, fmt.Errorf("list ticks for person %d: %w", personID, err)
	}
	defer rows.Close()

	ticks := make([]DayTick, 0, 64)
	for rows.Next() {
		var (
			tick DayTick
			raw  sql.NullString
		)

		if err := rows.Scan(&tick.HabitID, &raw); err != nil {
			return nil, fmt.Errorf("scan tick: %w", err)
		}

		if tick.Day, err = db.ScanDate(raw); err != nil {
			return nil, fmt.Errorf("tick date on habit %d: %w", tick.HabitID, err)
		}

		ticks = append(ticks, tick)
	}

	return ticks, rows.Err()
}

// TickCountsOn returns how many ticks each person has on one day.
func (s *Store) TickCountsOn(ctx context.Context, day time.Time) (map[int64]int, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT h.person_id, COUNT(*)
		   FROM ticks t
		   JOIN habits h ON h.id = t.habit_id
		  WHERE t.tick_date >= ? AND t.tick_date < ? AND h.person_id IS NOT NULL
		  GROUP BY h.person_id`,
		db.FormatDate(day), db.FormatDate(day.AddDate(0, 0, 1)),
	)
	if err != nil {
		return nil, fmt.Errorf("count ticks on %s: %w", db.FormatDate(day), err)
	}
	defer rows.Close()

	counts := make(map[int64]int, 4)
	for rows.Next() {
		var personID int64
		var count int

		if err := rows.Scan(&personID, &count); err != nil {
			return nil, fmt.Errorf("scan tick count: %w", err)
		}

		counts[personID] = count
	}

	return counts, rows.Err()
}

// TickCountsByHabit returns how many ticks each of a person's habits has ever
// had.
func (s *Store) TickCountsByHabit(ctx context.Context, personID int64) (map[int64]int, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT h.id, COUNT(t.id)
		   FROM habits h
		   JOIN ticks t ON t.habit_id = h.id
		  WHERE h.person_id = ?
		  GROUP BY h.id`,
		personID,
	)
	if err != nil {
		return nil, fmt.Errorf("count ticks by habit for person %d: %w", personID, err)
	}
	defer rows.Close()

	counts := make(map[int64]int, 16)
	for rows.Next() {
		var habitID int64
		var count int

		if err := rows.Scan(&habitID, &count); err != nil {
			return nil, fmt.Errorf("scan habit tick count: %w", err)
		}

		counts[habitID] = count
	}

	return counts, rows.Err()
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

// ResumeHabit clears a habit's end date, so it runs again with no gap.
func (s *Store) ResumeHabit(ctx context.Context, id int64) error {
	_, err := s.q.ExecContext(ctx,
		`UPDATE habits SET end_date = NULL, updated_at = ? WHERE id = ?`,
		db.FormatTime(s.Now()), id,
	)
	if err != nil {
		return fmt.Errorf("resume habit %d: %w", id, err)
	}

	return nil
}
