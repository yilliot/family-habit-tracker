package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/el/family-habit-tracker/internal/db"
	"github.com/el/family-habit-tracker/internal/domain"
)

const participantColumns = `id, sprint_id, person_id, carry_forward_balance, active_reward_id, display_order, created_at, updated_at`

func scanParticipant(sc interface{ Scan(...any) error }) (domain.SprintParticipant, error) {
	var (
		p                    domain.SprintParticipant
		activeRewardID       sql.NullInt64
		createdAt, updatedAt sql.NullString
	)

	err := sc.Scan(&p.ID, &p.SprintID, &p.PersonID, &p.CarryForwardBalance, &activeRewardID, &p.DisplayOrder, &createdAt, &updatedAt)
	if err != nil {
		return domain.SprintParticipant{}, err
	}

	p.ActiveRewardID = toInt64Ptr(activeRewardID)

	if p.CreatedAt, err = db.ScanTime(createdAt); err != nil {
		return domain.SprintParticipant{}, fmt.Errorf("participant %d created_at: %w", p.ID, err)
	}
	if p.UpdatedAt, err = db.ScanTime(updatedAt); err != nil {
		return domain.SprintParticipant{}, fmt.Errorf("participant %d updated_at: %w", p.ID, err)
	}

	return p, nil
}

// Participant fetches one sprint enrolment by id.
func (s *Store) Participant(ctx context.Context, id int64) (domain.SprintParticipant, error) {
	p, err := scanParticipant(s.q.QueryRowContext(ctx, `SELECT `+participantColumns+` FROM sprint_participants WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.SprintParticipant{}, ErrNotFound
	}
	if err != nil {
		return domain.SprintParticipant{}, fmt.Errorf("get participant %d: %w", id, err)
	}

	return p, nil
}

// ParticipantsBySprint lists a sprint's enrolments in display order.
func (s *Store) ParticipantsBySprint(ctx context.Context, sprintID int64) ([]domain.SprintParticipant, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+participantColumns+` FROM sprint_participants WHERE sprint_id = ? ORDER BY display_order ASC, id ASC`,
		sprintID,
	)
	if err != nil {
		return nil, fmt.Errorf("list participants for sprint %d: %w", sprintID, err)
	}
	defer rows.Close()

	participants := make([]domain.SprintParticipant, 0, 8)
	for rows.Next() {
		p, err := scanParticipant(rows)
		if err != nil {
			return nil, fmt.Errorf("scan participant: %w", err)
		}

		participants = append(participants, p)
	}

	return participants, rows.Err()
}

// ParticipantForPerson finds a person's enrolment in a sprint, or nil when they
// are not taking part.
func (s *Store) ParticipantForPerson(ctx context.Context, sprintID, personID int64) (*domain.SprintParticipant, error) {
	p, err := scanParticipant(s.q.QueryRowContext(ctx,
		`SELECT `+participantColumns+` FROM sprint_participants WHERE sprint_id = ? AND person_id = ?`,
		sprintID, personID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("participant for person %d in sprint %d: %w", personID, sprintID, err)
	}

	return &p, nil
}

// ParticipantWithActiveReward finds the participant in a sprint who is
// currently earning toward the given reward, or nil when nobody is.
func (s *Store) ParticipantWithActiveReward(ctx context.Context, sprintID, rewardID int64) (*domain.SprintParticipant, error) {
	p, err := scanParticipant(s.q.QueryRowContext(ctx,
		`SELECT `+participantColumns+` FROM sprint_participants WHERE sprint_id = ? AND active_reward_id = ?`,
		sprintID, rewardID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("participant with reward %d in sprint %d: %w", rewardID, sprintID, err)
	}

	return &p, nil
}

// CreateParticipant enrols a person in a sprint with an opening balance.
func (s *Store) CreateParticipant(ctx context.Context, sprintID, personID int64, carryForward int, activeRewardID *int64, displayOrder int) (domain.SprintParticipant, error) {
	now := db.FormatTime(s.Now())

	res, err := s.q.ExecContext(ctx,
		`INSERT INTO sprint_participants
		   (sprint_id, person_id, carry_forward_balance, active_reward_id, display_order, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sprintID, personID, carryForward, activeRewardID, displayOrder, now, now,
	)
	if err != nil {
		return domain.SprintParticipant{}, fmt.Errorf("insert participant: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return domain.SprintParticipant{}, fmt.Errorf("insert participant id: %w", err)
	}

	return s.Participant(ctx, id)
}

// SetParticipantActiveReward points a participant at a different reward, or at
// none when rewardID is nil.
func (s *Store) SetParticipantActiveReward(ctx context.Context, participantID int64, rewardID *int64) error {
	_, err := s.q.ExecContext(ctx,
		`UPDATE sprint_participants SET active_reward_id = ?, updated_at = ? WHERE id = ?`,
		rewardID, db.FormatTime(s.Now()), participantID,
	)
	if err != nil {
		return fmt.Errorf("set active reward on participant %d: %w", participantID, err)
	}

	return nil
}

// MaxParticipantDisplayOrder returns the highest order value used in a sprint.
func (s *Store) MaxParticipantDisplayOrder(ctx context.Context, sprintID int64) (int, error) {
	var max sql.NullInt64

	err := s.q.QueryRowContext(ctx, `SELECT MAX(display_order) FROM sprint_participants WHERE sprint_id = ?`, sprintID).Scan(&max)
	if err != nil {
		return 0, fmt.Errorf("max participant display_order: %w", err)
	}

	return int(max.Int64), nil
}

// CountSprintsForPerson counts how many sprints a person has taken part in.
func (s *Store) CountSprintsForPerson(ctx context.Context, personID int64) (int, error) {
	var count int

	err := s.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM sprint_participants WHERE person_id = ?`, personID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count sprints for person %d: %w", personID, err)
	}

	return count, nil
}

// CreateAdjustment records a signed manual correction to a participant's
// balance.
func (s *Store) CreateAdjustment(ctx context.Context, participantID int64, amount int, reason *string) (domain.PointAdjustment, error) {
	now := s.Now()

	res, err := s.q.ExecContext(ctx,
		`INSERT INTO point_adjustments (sprint_participant_id, amount, reason, created_at) VALUES (?, ?, ?, ?)`,
		participantID, amount, nullString(reason), db.FormatTime(now),
	)
	if err != nil {
		return domain.PointAdjustment{}, fmt.Errorf("insert adjustment: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return domain.PointAdjustment{}, fmt.Errorf("insert adjustment id: %w", err)
	}

	return domain.PointAdjustment{
		ID:                  id,
		SprintParticipantID: participantID,
		Amount:              amount,
		Reason:              reason,
		CreatedAt:           now,
	}, nil
}

// AdjustmentsForParticipant lists a participant's manual corrections, newest
// first, for the audit trail shown in the UI.
func (s *Store) AdjustmentsForParticipant(ctx context.Context, participantID int64) ([]domain.PointAdjustment, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT id, sprint_participant_id, amount, reason, created_at
		   FROM point_adjustments WHERE sprint_participant_id = ? ORDER BY created_at DESC, id DESC`,
		participantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list adjustments for participant %d: %w", participantID, err)
	}
	defer rows.Close()

	adjustments := make([]domain.PointAdjustment, 0, 4)
	for rows.Next() {
		var (
			a         domain.PointAdjustment
			reason    sql.NullString
			createdAt sql.NullString
		)

		if err := rows.Scan(&a.ID, &a.SprintParticipantID, &a.Amount, &reason, &createdAt); err != nil {
			return nil, fmt.Errorf("scan adjustment: %w", err)
		}

		a.Reason = toStringPtr(reason)

		parsed, err := db.ScanTime(createdAt)
		if err != nil {
			return nil, fmt.Errorf("adjustment %d created_at: %w", a.ID, err)
		}
		if parsed != nil {
			a.CreatedAt = *parsed
		}

		adjustments = append(adjustments, a)
	}

	return adjustments, rows.Err()
}
