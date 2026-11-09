package db_test

import (
	"database/sql"
	"os"
	"testing"

	"github.com/el/family-habit-tracker/internal/db"
)

// legacyFile writes a database the way the sprint-era app left it: the Laravel
// schema, user_version 0, two sprints (the last one still marked active after
// its end date, as production was), habits copied from one sprint to the next,
// ticks in both date formats, and deductions tied to enrolments.
func legacyFile(t *testing.T) string {
	t.Helper()

	path := t.TempDir() + "/legacy.sqlite"

	raw, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	defer raw.Close()

	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}

	for _, statement := range []string{
		string(schema),
		`INSERT INTO people (id, name, display_order) VALUES (1, '爸爸', 1), (2, '可遇', 2)`,
		`INSERT INTO sprints (id, start_date, end_date, status, started_at, created_at) VALUES
			(1, '2026-04-19 00:00:00', '2026-05-16 00:00:00', 'archived', '2026-04-26 15:11:25', '2026-04-26 15:11:25'),
			(2, '2026-09-29', '2026-10-31', 'active', '2026-09-30 14:11:13', '2026-09-30 14:11:13')`,
		`INSERT INTO sprint_participants (id, sprint_id, person_id, created_at) VALUES
			(1, 1, 1, '2026-04-26 15:11:25'), (2, 1, 2, '2026-04-26 15:11:25'),
			(3, 2, 1, '2026-09-30 14:11:13'), (4, 2, 2, '2026-09-30 14:11:13')`,
		// 10–12 typed in with sprint 1; 13 added to it two weeks in, then
		// ticked a day before it was added; 14 removed mid-sprint.
		`INSERT INTO habits (id, sprint_participant_id, name, display_order, created_at, deleted_at) VALUES
			(10, 1, '带敬拜', 1, '2026-04-26 15:11:25', NULL),
			(11, 2, '阅读', 1, '2026-04-26 15:11:25', NULL),
			(12, 2, '整理书桌', 2, '2026-04-26 15:11:25', NULL),
			(13, 2, 'piano', 3, '2026-05-03 10:00:00', NULL),
			(14, 1, '跑步', 2, '2026-04-26 15:11:25', '2026-05-01 08:00:00'),
			(20, 3, '带敬拜', 1, '2026-09-30 14:11:13', NULL),
			(21, 4, '阅读', 1, '2026-09-30 14:11:13', NULL)`,
		`INSERT INTO ticks (habit_id, tick_date) VALUES
			(10, '2026-04-19 00:00:00'), (10, '2026-05-16'),
			(11, '2026-04-20 00:00:00'), (13, '2026-05-02'), (14, '2026-04-25'),
			(20, '2026-09-29'), (21, '2026-10-01')`,
		`INSERT INTO point_adjustments (id, sprint_participant_id, amount, reason) VALUES (1, 2, -5, 'zoo'), (2, 4, -3, NULL)`,
	} {
		if _, err := raw.Exec(statement); err != nil {
			t.Fatalf("seed legacy file: %v\n%s", err, statement)
		}
	}

	return path
}

func TestUpgradeGivesEveryHabitItsOwnerAndDates(t *testing.T) {
	sqlDB, err := db.Open(t.Context(), legacyFile(t))
	if err != nil {
		t.Fatalf("open legacy file: %v", err)
	}
	defer sqlDB.Close()

	want := map[int64]struct {
		person int64
		start  string
		end    sql.NullString
	}{
		10: {1, "2026-04-19", sql.NullString{String: "2026-05-16", Valid: true}},
		11: {2, "2026-04-19", sql.NullString{String: "2026-05-16", Valid: true}},
		12: {2, "2026-04-19", sql.NullString{String: "2026-05-16", Valid: true}},
		13: {2, "2026-05-02", sql.NullString{String: "2026-05-16", Valid: true}},
		14: {1, "2026-04-19", sql.NullString{String: "2026-04-30", Valid: true}},
		// The latest sprint's habits keep running, though it ended on 31 Oct.
		20: {1, "2026-09-29", sql.NullString{}},
		21: {2, "2026-09-29", sql.NullString{}},
	}

	rows, err := sqlDB.Query(`SELECT id, person_id, CAST(start_date AS TEXT), CAST(end_date AS TEXT) FROM habits`)
	if err != nil {
		t.Fatalf("read habits: %v", err)
	}
	defer rows.Close()

	seen := 0
	for rows.Next() {
		var (
			id, person int64
			start      string
			end        sql.NullString
		)

		if err := rows.Scan(&id, &person, &start, &end); err != nil {
			t.Fatalf("scan: %v", err)
		}

		seen++
		w := want[id]

		if person != w.person || start != w.start || end != w.end {
			t.Errorf("habit %d = person %d, %s → %v; want person %d, %s → %v", id, person, start, end, w.person, w.start, w.end)
		}
	}

	if seen != len(want) {
		t.Errorf("saw %d habits, want %d", seen, len(want))
	}

	var ticks int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM ticks`).Scan(&ticks); err != nil || ticks != 7 {
		t.Errorf("ticks after upgrade = %d (%v), want all 7 untouched", ticks, err)
	}
}

func TestUpgradeMovesDeductionsOntoPeople(t *testing.T) {
	sqlDB, err := db.Open(t.Context(), legacyFile(t))
	if err != nil {
		t.Fatalf("open legacy file: %v", err)
	}
	defer sqlDB.Close()

	var person1, person2 int64
	if err := sqlDB.QueryRow(`SELECT
		(SELECT person_id FROM point_adjustments WHERE id = 1),
		(SELECT person_id FROM point_adjustments WHERE id = 2)`).Scan(&person1, &person2); err != nil {
		t.Fatalf("read adjustments: %v", err)
	}

	if person1 != 2 || person2 != 2 {
		t.Errorf("adjustments belong to %d and %d, want both on person 2", person1, person2)
	}

	// A new deduction needs a person and nothing else.
	if _, err := sqlDB.Exec(`INSERT INTO point_adjustments (person_id, amount) VALUES (1, -1)`); err != nil {
		t.Errorf("a deduction with no enrolment should be accepted: %v", err)
	}

	if _, err := sqlDB.Exec(`INSERT INTO point_adjustments (person_id, amount) VALUES (999, -1)`); err == nil {
		t.Error("a deduction for a missing person should be refused")
	}
}

func TestUpgradeRunsOnceAndRecordsItsVersion(t *testing.T) {
	path := legacyFile(t)

	for range 3 {
		sqlDB, err := db.Open(t.Context(), path)
		if err != nil {
			t.Fatalf("open: %v", err)
		}

		var version int
		if err := sqlDB.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
			t.Fatalf("read version: %v", err)
		}

		if version != db.SchemaVersion {
			t.Errorf("user_version = %d, want %d", version, db.SchemaVersion)
		}

		var problems int
		if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&problems); err != nil || problems != 0 {
			t.Errorf("foreign key check found %d problems (%v)", problems, err)
		}

		sqlDB.Close()
	}
}

func TestANewerFileIsRefused(t *testing.T) {
	path := t.TempDir() + "/future.sqlite"

	raw, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}

	if _, err := raw.Exec(`PRAGMA user_version = 99`); err != nil {
		t.Fatalf("set version: %v", err)
	}
	raw.Close()

	if sqlDB, err := db.Open(t.Context(), path); err == nil {
		sqlDB.Close()
		t.Fatal("a file written by a newer build should be refused, not silently used")
	}
}
