package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// A database file records which of these steps it has taken in SQLite's
// user_version header. Open runs whatever is missing, in order, each inside one
// transaction together with its version bump, so a file is only ever at a
// version it fully reached. Files written by the Laravel app and by the first
// Go version are at 0.
//
// Steps are append-only: never edit or reorder one that has shipped.
var migrations = []struct {
	name string
	run  func(context.Context, *sql.Tx) error
}{
	{"baseline schema", baseline},
	{"habits belong to people, not sprints", personHabits},
}

// SchemaVersion is the version a file reaches once every step has run.
var SchemaVersion = len(migrations)

// migrate brings the file up to SchemaVersion.
func migrate(ctx context.Context, sqlDB *sql.DB) error {
	var version int
	if err := sqlDB.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	if version > len(migrations) {
		return fmt.Errorf("database is at schema version %d but this binary knows only %d; run a newer build", version, len(migrations))
	}

	for ; version < len(migrations); version++ {
		step := migrations[version]

		tx, err := sqlDB.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", version+1, err)
		}

		if err := step.run(ctx, tx); err != nil {
			_ = tx.Rollback()

			return fmt.Errorf("migration %d (%s): %w", version+1, step.name, err)
		}

		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`PRAGMA user_version = %d`, version+1)); err != nil {
			_ = tx.Rollback()

			return fmt.Errorf("record schema version %d: %w", version+1, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", version+1, err)
		}
	}

	return nil
}

// baseline creates the tables the Laravel app defined. Every statement is
// IF NOT EXISTS, so on a file that app wrote it only adds two indexes.
func baseline(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, schemaSQL)

	return err
}

// personHabits ends sprints. A habit used to reach its person only through a
// sprint enrolment and lived exactly as long as that sprint. Now it belongs to
// the person directly and runs from start_date until end_date (NULL while it is
// still running), so anyone can add or stop one on any day.
//
// Nothing here moves a tick or changes a balance: ticks stay on the habit rows
// they were made on, and each old row is given the dates it actually ran.
// Deductions get the same direct link to their person.
func personHabits(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range []string{
		`ALTER TABLE habits ADD COLUMN person_id integer REFERENCES people(id)`,
		`ALTER TABLE habits ADD COLUMN start_date date`,
		`ALTER TABLE habits ADD COLUMN end_date date`,
		`UPDATE habits SET person_id = (
			SELECT sp.person_id FROM sprint_participants sp WHERE sp.id = habits.sprint_participant_id)`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}

	spans, err := habitSpans(ctx, tx)
	if err != nil {
		return err
	}

	for _, span := range spans {
		var end any
		if span.end != nil {
			end = FormatDate(*span.end)
		}

		if _, err := tx.ExecContext(ctx,
			`UPDATE habits SET start_date = ?, end_date = ? WHERE id = ?`,
			FormatDate(span.start), end, span.id,
		); err != nil {
			return fmt.Errorf("date habit %d: %w", span.id, err)
		}
	}

	for _, statement := range []string{
		`CREATE INDEX IF NOT EXISTS "habits_person_id_index" on "habits" ("person_id")`,

		// SQLite cannot relax NOT NULL in place, so the table is rebuilt with the
		// person as the owner and the old enrolment link kept, nullable, for the
		// record. Nothing references this table, so dropping it is safe with
		// foreign keys on.
		`CREATE TABLE "point_adjustments_v2" (
			"id"                    integer primary key autoincrement not null,
			"person_id"             integer not null,
			"sprint_participant_id" integer,
			"amount"                integer not null,
			"reason"                varchar,
			"created_at"            datetime not null default CURRENT_TIMESTAMP,
			foreign key("person_id")             references "people"("id")              on delete cascade,
			foreign key("sprint_participant_id") references "sprint_participants"("id") on delete set null
		)`,
		`INSERT INTO point_adjustments_v2 (id, person_id, sprint_participant_id, amount, reason, created_at)
			SELECT pa.id, sp.person_id, pa.sprint_participant_id, pa.amount, pa.reason, pa.created_at
			  FROM point_adjustments pa
			  JOIN sprint_participants sp ON sp.id = pa.sprint_participant_id`,
		`DROP TABLE point_adjustments`,
		`ALTER TABLE point_adjustments_v2 RENAME TO point_adjustments`,
		`CREATE INDEX IF NOT EXISTS "point_adjustments_person_id_index" on "point_adjustments" ("person_id")`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}

	return nil
}

// habitSpan is the run of days one existing habit row was live.
type habitSpan struct {
	id    int64
	start time.Time
	end   *time.Time
}

// habitSpans works out, for every habit row, the days it was live:
//
//   - Habits typed in when a sprint was set up ran from the sprint's first day.
//     One added mid-sprint ran from the day it was added.
//   - A habit from any sprint but the latest ended on that sprint's last day.
//     The latest sprint's habits keep running, whether or not that sprint was
//     closed, because they are what the family was doing most recently.
//   - A habit removed mid-sprint ended the day before it was removed.
//
// Ticks always win: a span is widened to cover every tick on the row, so no
// recorded day ever falls outside its habit's dates.
func habitSpans(ctx context.Context, tx *sql.Tx) ([]habitSpan, error) {
	var latestSprint sql.NullInt64
	if err := tx.QueryRowContext(ctx,
		`SELECT id FROM sprints ORDER BY date(start_date) DESC, id DESC LIMIT 1`,
	).Scan(&latestSprint); err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("latest sprint: %w", err)
	}

	rows, err := tx.QueryContext(ctx,
		`SELECT h.id, h.created_at, h.deleted_at, sp.created_at, s.id, s.start_date, s.end_date,
		        (SELECT MIN(date(t.tick_date)) FROM ticks t WHERE t.habit_id = h.id),
		        (SELECT MAX(date(t.tick_date)) FROM ticks t WHERE t.habit_id = h.id)
		   FROM habits h
		   LEFT JOIN sprint_participants sp ON sp.id = h.sprint_participant_id
		   LEFT JOIN sprints s              ON s.id  = sp.sprint_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("read habits: %w", err)
	}
	defer rows.Close()

	var spans []habitSpan

	for rows.Next() {
		var (
			id                                          int64
			created, deleted, enrolled                  sql.NullString
			sprintID                                    sql.NullInt64
			sprintStart, sprintEnd, firstTick, lastTick sql.NullString
		)

		if err := rows.Scan(&id, &created, &deleted, &enrolled, &sprintID, &sprintStart, &sprintEnd, &firstTick, &lastTick); err != nil {
			return nil, fmt.Errorf("scan habit: %w", err)
		}

		span, err := spanFor(id, created, deleted, enrolled, sprintID, sprintStart, sprintEnd, firstTick, lastTick, latestSprint)
		if err != nil {
			return nil, err
		}

		spans = append(spans, span)
	}

	return spans, rows.Err()
}

// spanFor applies the rules in habitSpans to one row.
func spanFor(
	id int64,
	created, deleted, enrolled sql.NullString,
	sprintID sql.NullInt64,
	sprintStart, sprintEnd, firstTick, lastTick sql.NullString,
	latestSprint sql.NullInt64,
) (habitSpan, error) {
	span := habitSpan{id: id}

	createdAt, err := ScanTime(created)
	if err != nil {
		return span, fmt.Errorf("habit %d created_at: %w", id, err)
	}

	if createdAt != nil {
		span.start = dayOf(*createdAt)
	}

	if sprintID.Valid {
		start, err := ScanDate(sprintStart)
		if err != nil {
			return span, fmt.Errorf("habit %d sprint start: %w", id, err)
		}

		enrolledAt, err := ScanTime(enrolled)
		if err != nil {
			return span, fmt.Errorf("habit %d enrolment: %w", id, err)
		}

		// Typed in with the enrolment (within the hour) means it was part of the
		// sprint from its first day, even if the sprint was set up a day late.
		setUpWithSprint := createdAt == nil || enrolledAt == nil || !createdAt.After(enrolledAt.Add(time.Hour))
		if setUpWithSprint || span.start.Before(start) {
			span.start = start
		}

		if !latestSprint.Valid || sprintID.Int64 != latestSprint.Int64 {
			end, err := ScanDate(sprintEnd)
			if err != nil {
				return span, fmt.Errorf("habit %d sprint end: %w", id, err)
			}

			span.end = &end
		}
	}

	deletedAt, err := ScanTime(deleted)
	if err != nil {
		return span, fmt.Errorf("habit %d deleted_at: %w", id, err)
	}

	if deletedAt != nil {
		dayBefore := dayOf(*deletedAt).AddDate(0, 0, -1)
		if span.end == nil || dayBefore.Before(*span.end) {
			span.end = &dayBefore
		}
	}

	if firstTick.Valid {
		first, err := ScanDate(firstTick)
		if err != nil {
			return span, fmt.Errorf("habit %d first tick: %w", id, err)
		}

		if span.start.IsZero() || first.Before(span.start) {
			span.start = first
		}
	}

	if lastTick.Valid && span.end != nil {
		last, err := ScanDate(lastTick)
		if err != nil {
			return span, fmt.Errorf("habit %d last tick: %w", id, err)
		}

		if last.After(*span.end) {
			span.end = &last
		}
	}

	if span.start.IsZero() {
		// No dates at all: a row the old app never stamped. Treat it as having
		// ended before it began, so it never shows but its row is kept.
		span.start = time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)
	}

	if span.end != nil && span.end.Before(span.start) {
		never := span.start.AddDate(0, 0, -1)
		span.end = &never
	}

	return span, nil
}

// dayOf reduces an instant to its UTC calendar day.
func dayOf(t time.Time) time.Time {
	t = t.UTC()

	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
