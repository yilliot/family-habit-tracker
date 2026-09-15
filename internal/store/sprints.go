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

const sprintColumns = `id, start_date, end_date, status, started_at, ended_at, created_at, updated_at`

func scanSprint(sc interface{ Scan(...any) error }) (domain.Sprint, error) {
	var (
		sp                                       domain.Sprint
		startDate, endDate                       sql.NullString
		startedAt, endedAt, createdAt, updatedAt sql.NullString
	)

	if err := sc.Scan(&sp.ID, &startDate, &endDate, &sp.Status, &startedAt, &endedAt, &createdAt, &updatedAt); err != nil {
		return domain.Sprint{}, err
	}

	var err error
	if sp.StartDate, err = db.ScanDate(startDate); err != nil {
		return domain.Sprint{}, fmt.Errorf("sprint %d start_date: %w", sp.ID, err)
	}
	if sp.EndDate, err = db.ScanDate(endDate); err != nil {
		return domain.Sprint{}, fmt.Errorf("sprint %d end_date: %w", sp.ID, err)
	}
	if sp.StartedAt, err = db.ScanTime(startedAt); err != nil {
		return domain.Sprint{}, fmt.Errorf("sprint %d started_at: %w", sp.ID, err)
	}
	if sp.EndedAt, err = db.ScanTime(endedAt); err != nil {
		return domain.Sprint{}, fmt.Errorf("sprint %d ended_at: %w", sp.ID, err)
	}
	if sp.CreatedAt, err = db.ScanTime(createdAt); err != nil {
		return domain.Sprint{}, fmt.Errorf("sprint %d created_at: %w", sp.ID, err)
	}
	if sp.UpdatedAt, err = db.ScanTime(updatedAt); err != nil {
		return domain.Sprint{}, fmt.Errorf("sprint %d updated_at: %w", sp.ID, err)
	}

	return sp, nil
}

// ActiveSprint returns the one sprint currently accepting ticks, or nil when
// the family is between sprints.
func (s *Store) ActiveSprint(ctx context.Context) (*domain.Sprint, error) {
	sp, err := scanSprint(s.q.QueryRowContext(ctx,
		`SELECT `+sprintColumns+` FROM sprints WHERE status = ? ORDER BY id DESC LIMIT 1`,
		domain.StatusActive,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("active sprint: %w", err)
	}

	return &sp, nil
}

// Sprint fetches one sprint by id.
func (s *Store) Sprint(ctx context.Context, id int64) (domain.Sprint, error) {
	sp, err := scanSprint(s.q.QueryRowContext(ctx, `SELECT `+sprintColumns+` FROM sprints WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Sprint{}, ErrNotFound
	}
	if err != nil {
		return domain.Sprint{}, fmt.Errorf("get sprint %d: %w", id, err)
	}

	return sp, nil
}

// ArchivedSprints lists finished sprints, most recently ended first.
func (s *Store) ArchivedSprints(ctx context.Context) ([]domain.Sprint, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+sprintColumns+` FROM sprints WHERE status = ? ORDER BY end_date DESC, id DESC`,
		domain.StatusArchived,
	)
	if err != nil {
		return nil, fmt.Errorf("list archived sprints: %w", err)
	}
	defer rows.Close()

	sprints := make([]domain.Sprint, 0, 8)
	for rows.Next() {
		sp, err := scanSprint(rows)
		if err != nil {
			return nil, fmt.Errorf("scan sprint: %w", err)
		}

		sprints = append(sprints, sp)
	}

	return sprints, rows.Err()
}

// LatestSprint returns the most recent sprint of any status. It seeds the
// "start a new sprint" form with the previous roster and habits.
func (s *Store) LatestSprint(ctx context.Context) (*domain.Sprint, error) {
	sp, err := scanSprint(s.q.QueryRowContext(ctx,
		`SELECT `+sprintColumns+` FROM sprints ORDER BY end_date DESC, id DESC LIMIT 1`,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("latest sprint: %w", err)
	}

	return &sp, nil
}

// HasActiveSprint reports whether a sprint is currently running.
func (s *Store) HasActiveSprint(ctx context.Context) (bool, error) {
	var count int

	err := s.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM sprints WHERE status = ?`, domain.StatusActive).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("count active sprints: %w", err)
	}

	return count > 0, nil
}

// CreateSprint opens a new sprint over the given inclusive date range.
func (s *Store) CreateSprint(ctx context.Context, start, end time.Time) (domain.Sprint, error) {
	now := db.FormatTime(s.Now())

	res, err := s.q.ExecContext(ctx,
		`INSERT INTO sprints (start_date, end_date, status, started_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		db.FormatDate(start), db.FormatDate(end), domain.StatusActive, now, now, now,
	)
	if err != nil {
		return domain.Sprint{}, fmt.Errorf("insert sprint: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return domain.Sprint{}, fmt.Errorf("insert sprint id: %w", err)
	}

	return s.Sprint(ctx, id)
}

// ArchiveSprint marks a sprint finished and read-only.
func (s *Store) ArchiveSprint(ctx context.Context, id int64) error {
	now := db.FormatTime(s.Now())

	_, err := s.q.ExecContext(ctx,
		`UPDATE sprints SET status = ?, ended_at = ?, updated_at = ? WHERE id = ?`,
		domain.StatusArchived, now, now, id,
	)
	if err != nil {
		return fmt.Errorf("archive sprint %d: %w", id, err)
	}

	return nil
}
