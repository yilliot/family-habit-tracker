// Package db opens the SQLite database and provides the value helpers needed to
// read and write the date/datetime formats used by the database this app
// inherited from its Laravel predecessor.
package db

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// Open opens (creating if absent) the SQLite database at path and brings its
// schema up to date (see migrate.go). The returned handle has foreign keys
// enforced and WAL journaling on.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", path)

	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}

	// SQLite permits a single writer. Serialising through one connection keeps
	// writes free of SQLITE_BUSY without sacrificing anything at family scale.
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(0)

	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("ping sqlite %q: %w", path, err)
	}

	if err := migrate(ctx, sqlDB); err != nil {
		sqlDB.Close()
		return nil, err
	}

	return sqlDB, nil
}

// DateLayout is the canonical wire format for the date-only columns
// (ticks.tick_date, habits.start_date, habits.end_date, and the legacy sprint
// dates).
const DateLayout = "2006-01-02"

// DateTimeLayout is the canonical wire format for datetime columns. It matches
// what Eloquent wrote, so rows this app inserts are indistinguishable from rows
// the previous implementation inserted.
const DateTimeLayout = "2006-01-02 15:04:05"

// acceptedLayouts lists every format observed in databases written by the
// Laravel implementation. Eloquent's `date` cast wrote plain dates in some code
// paths and full timestamps in others (a Carbon instance passed straight to a
// date column), and its `datetime` cast emitted ISO-8601 with an offset when a
// model was hydrated from an API payload. Reads tolerate all of them; writes
// always use the canonical layouts above.
var acceptedLayouts = []string{
	DateTimeLayout,
	DateLayout,
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02 15:04:05-07:00",
	time.RFC3339Nano,
}

// ParseTime decodes a database timestamp or date in any of the formats the
// previous implementation produced. Values are interpreted as UTC when they
// carry no offset, matching Laravel's default of storing UTC.
func ParseTime(raw string) (time.Time, error) {
	for _, layout := range acceptedLayouts {
		if parsed, err := time.ParseInLocation(layout, raw, time.UTC); err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("unrecognised time value %q", raw)
}

// ScanDate reads a date-only column into a day-precision time.Time, discarding
// any time component a legacy row may carry.
func ScanDate(src sql.NullString) (time.Time, error) {
	if !src.Valid || src.String == "" {
		return time.Time{}, nil
	}

	parsed, err := ParseTime(src.String)
	if err != nil {
		return time.Time{}, err
	}

	return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC), nil
}

// ScanTime reads a nullable datetime column. A NULL or empty value yields nil.
func ScanTime(src sql.NullString) (*time.Time, error) {
	if !src.Valid || src.String == "" {
		return nil, nil
	}

	parsed, err := ParseTime(src.String)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

// FormatDate renders a day for a date column.
func FormatDate(t time.Time) string { return t.Format(DateLayout) }

// FormatTime renders an instant for a datetime column, normalised to UTC.
func FormatTime(t time.Time) string { return t.UTC().Format(DateTimeLayout) }

// FormatTimePtr renders a nullable instant, yielding NULL for nil.
func FormatTimePtr(t *time.Time) any {
	if t == nil {
		return nil
	}

	return FormatTime(*t)
}
