package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/el/family-habit-tracker/internal/db"
)

// Balances are always derived from ticks, rewards and adjustments at read
// time; no running total is ever persisted. That keeps a corrected tick or an
// edited reward cost from leaving a stale number behind.
//
// Ticks belonging to soft-deleted habits are deliberately still counted. A
// habit removed from the grid mid-sprint keeps the points it already earned,
// which is the behaviour the family expects: you do not lose points because a
// habit was retired.

// TicksEarnedByPerson counts every point a person has ever earned, across all
// sprints and including habits that have since been removed from the grid.
func (s *Store) TicksEarnedByPerson(ctx context.Context, personID int64) (int, error) {
	var count int

	err := s.q.QueryRowContext(ctx,
		`SELECT COUNT(*)
		   FROM ticks t
		   JOIN habits h              ON h.id  = t.habit_id
		   JOIN sprint_participants sp ON sp.id = h.sprint_participant_id
		  WHERE sp.person_id = ?`,
		personID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count lifetime ticks for person %d: %w", personID, err)
	}

	return count, nil
}

// TicksEarnedByParticipant counts the points earned inside one sprint.
func (s *Store) TicksEarnedByParticipant(ctx context.Context, participantID int64) (int, error) {
	var count int

	err := s.q.QueryRowContext(ctx,
		`SELECT COUNT(*)
		   FROM ticks t
		   JOIN habits h ON h.id = t.habit_id
		  WHERE h.sprint_participant_id = ?`,
		participantID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count sprint ticks for participant %d: %w", participantID, err)
	}

	return count, nil
}

// SpentByPerson totals the cost of every reward a person has ever achieved.
func (s *Store) SpentByPerson(ctx context.Context, personID int64) (int, error) {
	var total sql.NullInt64

	err := s.q.QueryRowContext(ctx,
		`SELECT SUM(cost) FROM rewards WHERE person_id = ? AND achieved_at IS NOT NULL`,
		personID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum spent for person %d: %w", personID, err)
	}

	return int(total.Int64), nil
}

// SpentByPersonSince totals rewards a person achieved at or after an instant.
// Comparing through datetime() normalises the stored text, so rows written in
// any of the legacy timestamp formats still compare correctly.
func (s *Store) SpentByPersonSince(ctx context.Context, personID int64, since time.Time) (int, error) {
	var total sql.NullInt64

	err := s.q.QueryRowContext(ctx,
		`SELECT SUM(cost) FROM rewards
		  WHERE person_id = ?
		    AND achieved_at IS NOT NULL
		    AND datetime(achieved_at) >= datetime(?)`,
		personID, db.FormatTime(since),
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum spent since for person %d: %w", personID, err)
	}

	return int(total.Int64), nil
}

// AdjustmentTotalByPerson sums a person's manual corrections across all sprints.
func (s *Store) AdjustmentTotalByPerson(ctx context.Context, personID int64) (int, error) {
	var total sql.NullInt64

	err := s.q.QueryRowContext(ctx,
		`SELECT SUM(pa.amount)
		   FROM point_adjustments pa
		   JOIN sprint_participants sp ON sp.id = pa.sprint_participant_id
		  WHERE sp.person_id = ?`,
		personID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum adjustments for person %d: %w", personID, err)
	}

	return int(total.Int64), nil
}

// AdjustmentTotalByParticipant sums one enrolment's manual corrections.
func (s *Store) AdjustmentTotalByParticipant(ctx context.Context, participantID int64) (int, error) {
	var total sql.NullInt64

	err := s.q.QueryRowContext(ctx,
		`SELECT SUM(amount) FROM point_adjustments WHERE sprint_participant_id = ?`,
		participantID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum adjustments for participant %d: %w", participantID, err)
	}

	return int(total.Int64), nil
}
