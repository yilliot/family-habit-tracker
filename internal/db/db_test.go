package db_test

import (
	"testing"
	"time"

	"database/sql"

	"github.com/el/family-habit-tracker/internal/db"
)

// The database this app inherits was written by Eloquent, whose date and
// datetime casts did not agree on a single format. Reads must accept every
// shape that is actually out there, or a production row becomes a parse error.
func TestParseTimeAcceptsEveryLegacyFormat(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want time.Time
	}{
		{"date only", "2026-04-19", time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC)},
		{"datetime", "2026-04-19 14:35:55", time.Date(2026, 4, 19, 14, 35, 55, 0, time.UTC)},
		{"date column holding a timestamp", "2026-04-19 00:00:00", time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC)},
		{"iso with zulu", "2026-04-19T14:35:55Z", time.Date(2026, 4, 19, 14, 35, 55, 0, time.UTC)},
		{"iso without zone", "2026-04-19T14:35:55", time.Date(2026, 4, 19, 14, 35, 55, 0, time.UTC)},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := db.ParseTime(testCase.raw)
			if err != nil {
				t.Fatalf("ParseTime(%q): %v", testCase.raw, err)
			}

			if !got.Equal(testCase.want) {
				t.Errorf("ParseTime(%q) = %s, want %s", testCase.raw, got, testCase.want)
			}
		})
	}
}

func TestParseTimeRejectsNonsense(t *testing.T) {
	if _, err := db.ParseTime("not a date"); err == nil {
		t.Error("expected an error for an unparseable value")
	}
}

// A date column that happens to hold a full timestamp must still read back as a
// plain day, or a habit's first day would drift.
func TestScanDateDiscardsAnyTimeComponent(t *testing.T) {
	got, err := db.ScanDate(sql.NullString{String: "2026-04-19 23:59:59", Valid: true})
	if err != nil {
		t.Fatalf("ScanDate: %v", err)
	}

	want := time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("ScanDate = %s, want %s", got, want)
	}
}

func TestScanTimeHandlesNull(t *testing.T) {
	got, err := db.ScanTime(sql.NullString{Valid: false})
	if err != nil {
		t.Fatalf("ScanTime: %v", err)
	}

	if got != nil {
		t.Errorf("ScanTime(NULL) = %v, want nil", got)
	}
}

// Writes must produce exactly the format the previous implementation used, so
// rows written by this app are indistinguishable from the ones already there.
func TestFormatMatchesTheInheritedWireFormat(t *testing.T) {
	instant := time.Date(2026, 4, 19, 14, 35, 55, 123456789, time.UTC)

	if got := db.FormatTime(instant); got != "2026-04-19 14:35:55" {
		t.Errorf("FormatTime = %q, want %q", got, "2026-04-19 14:35:55")
	}

	if got := db.FormatDate(instant); got != "2026-04-19" {
		t.Errorf("FormatDate = %q, want %q", got, "2026-04-19")
	}
}

// A timestamp in another zone is normalised to UTC on the way in, matching how
// the inherited database stores every datetime.
func TestFormatTimeNormalisesToUTC(t *testing.T) {
	singapore := time.FixedZone("+08", 8*60*60)
	instant := time.Date(2026, 4, 19, 8, 0, 0, 0, singapore)

	if got := db.FormatTime(instant); got != "2026-04-19 00:00:00" {
		t.Errorf("FormatTime = %q, want the UTC equivalent %q", got, "2026-04-19 00:00:00")
	}
}

// Opening a database again must be safe: each schema step runs once, so an
// existing production file is left exactly as it was.
func TestOpenIsIdempotent(t *testing.T) {
	path := t.TempDir() + "/repeat.sqlite"

	for range 3 {
		sqlDB, err := db.Open(t.Context(), path)
		if err != nil {
			t.Fatalf("open: %v", err)
		}

		if _, err := sqlDB.ExecContext(t.Context(), `INSERT INTO people (name, display_order) VALUES ('爸爸', 1)`); err != nil {
			t.Fatalf("insert: %v", err)
		}

		sqlDB.Close()
	}

	sqlDB, err := db.Open(t.Context(), path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer sqlDB.Close()

	var count int
	if err := sqlDB.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM people`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}

	if count != 3 {
		t.Errorf("rows survived re-opening = %d, want 3", count)
	}
}

// Foreign keys must be enforced, since several rules depend on the schema
// refusing to orphan a row.
func TestForeignKeysAreEnforced(t *testing.T) {
	sqlDB, err := db.Open(t.Context(), t.TempDir()+"/fk.sqlite")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()

	_, err = sqlDB.ExecContext(t.Context(), `INSERT INTO ticks (habit_id, tick_date) VALUES (9999, '2026-04-19')`)
	if err == nil {
		t.Error("expected a foreign key violation for a tick on a missing habit")
	}
}

// The unique index is what makes ticking idempotent, so it is worth asserting
// directly rather than only through the service.
func TestOneTickPerHabitPerDay(t *testing.T) {
	ctx := t.Context()

	sqlDB, err := db.Open(ctx, t.TempDir()+"/unique.sqlite")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()

	mustExec(t, sqlDB, `INSERT INTO people (id, name, display_order) VALUES (1, '可遇', 1)`)
	mustExec(t, sqlDB, `INSERT INTO habits (id, person_id, name, start_date) VALUES (1, 1, '阅读', '2026-04-19')`)
	mustExec(t, sqlDB, `INSERT INTO ticks (habit_id, tick_date) VALUES (1, '2026-04-20')`)

	_, err = sqlDB.ExecContext(ctx, `INSERT INTO ticks (habit_id, tick_date) VALUES (1, '2026-04-20')`)
	if err == nil {
		t.Error("expected the unique index to reject a second tick on the same day")
	}
}

func mustExec(t *testing.T, sqlDB *sql.DB, query string) {
	t.Helper()

	if _, err := sqlDB.ExecContext(t.Context(), query); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}
