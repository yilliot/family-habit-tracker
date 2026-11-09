package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/el/family-habit-tracker/internal/db"
	"github.com/el/family-habit-tracker/internal/domain"
)

// Balances are always derived from ticks, rewards and adjustments at read
// time; no running total is ever persisted. That keeps a corrected tick or an
// edited reward cost from leaving a stale number behind.
//
// Ticks on stopped habits still count: you do not lose points because a habit
// was retired.

// TicksEarnedByPerson counts every point a person has ever earned.
func (s *Store) TicksEarnedByPerson(ctx context.Context, personID int64) (int, error) {
	var count int

	err := s.q.QueryRowContext(ctx,
		`SELECT COUNT(*)
		   FROM habits h
		   JOIN ticks t ON t.habit_id = h.id
		  WHERE h.person_id = ?`,
		personID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count lifetime ticks for person %d: %w", personID, err)
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

// AdjustmentTotalByPerson sums a person's manual corrections.
func (s *Store) AdjustmentTotalByPerson(ctx context.Context, personID int64) (int, error) {
	var total sql.NullInt64

	err := s.q.QueryRowContext(ctx,
		`SELECT SUM(amount) FROM point_adjustments WHERE person_id = ?`,
		personID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum adjustments for person %d: %w", personID, err)
	}

	return int(total.Int64), nil
}

// CreateAdjustment records a signed correction to a person's balance.
func (s *Store) CreateAdjustment(ctx context.Context, personID int64, amount int, reason *string) (domain.PointAdjustment, error) {
	now := s.Now()

	res, err := s.q.ExecContext(ctx,
		`INSERT INTO point_adjustments (person_id, amount, reason, created_at) VALUES (?, ?, ?, ?)`,
		personID, amount, nullString(reason), db.FormatTime(now),
	)
	if err != nil {
		return domain.PointAdjustment{}, fmt.Errorf("insert adjustment: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return domain.PointAdjustment{}, fmt.Errorf("insert adjustment id: %w", err)
	}

	return domain.PointAdjustment{ID: id, PersonID: personID, Amount: amount, Reason: reason, CreatedAt: now}, nil
}

// AdjustmentsForPerson lists a person's corrections made in [from, to), newest
// first.
func (s *Store) AdjustmentsForPerson(ctx context.Context, personID int64, from, to time.Time) ([]domain.PointAdjustment, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT id, amount, reason, created_at FROM point_adjustments
		  WHERE person_id = ? AND datetime(created_at) >= datetime(?) AND datetime(created_at) < datetime(?)
		  ORDER BY datetime(created_at) DESC, id DESC`,
		personID, db.FormatTime(from), db.FormatTime(to),
	)
	if err != nil {
		return nil, fmt.Errorf("list adjustments for person %d: %w", personID, err)
	}
	defer rows.Close()

	var adjustments []domain.PointAdjustment
	for rows.Next() {
		var (
			adjustment = domain.PointAdjustment{PersonID: personID}
			reason     sql.NullString
			createdAt  sql.NullString
		)

		if err := rows.Scan(&adjustment.ID, &adjustment.Amount, &reason, &createdAt); err != nil {
			return nil, fmt.Errorf("scan adjustment: %w", err)
		}

		adjustment.Reason = toStringPtr(reason)

		created, err := db.ScanTime(createdAt)
		if err != nil {
			return nil, fmt.Errorf("adjustment %d created_at: %w", adjustment.ID, err)
		}
		if created != nil {
			adjustment.CreatedAt = *created
		}

		adjustments = append(adjustments, adjustment)
	}

	return adjustments, rows.Err()
}
