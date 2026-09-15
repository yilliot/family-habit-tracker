package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/el/family-habit-tracker/internal/db"
	"github.com/el/family-habit-tracker/internal/domain"
)

const rewardColumns = `id, person_id, name, cost, achieved_at, created_at, updated_at`

func scanReward(sc interface{ Scan(...any) error }) (domain.Reward, error) {
	var (
		r                                domain.Reward
		achievedAt, createdAt, updatedAt sql.NullString
	)

	err := sc.Scan(&r.ID, &r.PersonID, &r.Name, &r.Cost, &achievedAt, &createdAt, &updatedAt)
	if err != nil {
		return domain.Reward{}, err
	}

	if r.AchievedAt, err = db.ScanTime(achievedAt); err != nil {
		return domain.Reward{}, fmt.Errorf("reward %d achieved_at: %w", r.ID, err)
	}
	if r.CreatedAt, err = db.ScanTime(createdAt); err != nil {
		return domain.Reward{}, fmt.Errorf("reward %d created_at: %w", r.ID, err)
	}
	if r.UpdatedAt, err = db.ScanTime(updatedAt); err != nil {
		return domain.Reward{}, fmt.Errorf("reward %d updated_at: %w", r.ID, err)
	}

	return r, nil
}

// Reward fetches one reward by id.
func (s *Store) Reward(ctx context.Context, id int64) (domain.Reward, error) {
	r, err := scanReward(s.q.QueryRowContext(ctx, `SELECT `+rewardColumns+` FROM rewards WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Reward{}, ErrNotFound
	}
	if err != nil {
		return domain.Reward{}, fmt.Errorf("get reward %d: %w", id, err)
	}

	return r, nil
}

// RewardByIDPtr resolves an optional reward reference.
func (s *Store) RewardByIDPtr(ctx context.Context, id *int64) (*domain.Reward, error) {
	if id == nil {
		return nil, nil
	}

	r, err := s.Reward(ctx, *id)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &r, nil
}

// ActiveRewardForPerson returns the person's current un-achieved goal, or nil
// when they are between rewards.
func (s *Store) ActiveRewardForPerson(ctx context.Context, personID int64) (*domain.Reward, error) {
	r, err := scanReward(s.q.QueryRowContext(ctx,
		`SELECT `+rewardColumns+` FROM rewards WHERE person_id = ? AND achieved_at IS NULL ORDER BY id ASC LIMIT 1`,
		personID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("active reward for person %d: %w", personID, err)
	}

	return &r, nil
}

// RewardsForPerson lists a person's rewards with the active one first, then
// achieved ones most recent first.
func (s *Store) RewardsForPerson(ctx context.Context, personID int64) ([]domain.Reward, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+rewardColumns+` FROM rewards
		  WHERE person_id = ?
		  ORDER BY (achieved_at IS NULL) DESC, achieved_at DESC, id DESC`,
		personID,
	)
	if err != nil {
		return nil, fmt.Errorf("list rewards for person %d: %w", personID, err)
	}
	defer rows.Close()

	rewards := make([]domain.Reward, 0, 4)
	for rows.Next() {
		r, err := scanReward(rows)
		if err != nil {
			return nil, fmt.Errorf("scan reward: %w", err)
		}

		rewards = append(rewards, r)
	}

	return rewards, rows.Err()
}

// CreateReward records a new goal for a person.
func (s *Store) CreateReward(ctx context.Context, personID int64, name string, cost int) (domain.Reward, error) {
	now := db.FormatTime(s.Now())

	res, err := s.q.ExecContext(ctx,
		`INSERT INTO rewards (person_id, name, cost, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		personID, name, cost, now, now,
	)
	if err != nil {
		return domain.Reward{}, fmt.Errorf("insert reward: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return domain.Reward{}, fmt.Errorf("insert reward id: %w", err)
	}

	return s.Reward(ctx, id)
}

// UpdateReward changes a reward's name and cost.
func (s *Store) UpdateReward(ctx context.Context, id int64, name string, cost int) error {
	_, err := s.q.ExecContext(ctx,
		`UPDATE rewards SET name = ?, cost = ?, updated_at = ? WHERE id = ?`,
		name, cost, db.FormatTime(s.Now()), id,
	)
	if err != nil {
		return fmt.Errorf("update reward %d: %w", id, err)
	}

	return nil
}

// MarkRewardAchieved stamps a reward as earned.
func (s *Store) MarkRewardAchieved(ctx context.Context, id int64) error {
	now := db.FormatTime(s.Now())

	_, err := s.q.ExecContext(ctx,
		`UPDATE rewards SET achieved_at = ?, updated_at = ? WHERE id = ? AND achieved_at IS NULL`,
		now, now, id,
	)
	if err != nil {
		return fmt.Errorf("mark reward %d achieved: %w", id, err)
	}

	return nil
}

// DeleteReward permanently removes an unachieved reward.
func (s *Store) DeleteReward(ctx context.Context, id int64) error {
	if _, err := s.q.ExecContext(ctx, `DELETE FROM rewards WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete reward %d: %w", id, err)
	}

	return nil
}

// CountAchievedRewards counts the goals a person has earned over their whole
// history.
func (s *Store) CountAchievedRewards(ctx context.Context, personID int64) (int, error) {
	var count int

	err := s.q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM rewards WHERE person_id = ? AND achieved_at IS NOT NULL`, personID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count achieved rewards for person %d: %w", personID, err)
	}

	return count, nil
}
