// Package store is the data-access layer. Every SQL statement in the
// application lives here; callers work in terms of domain types.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/el/family-habit-tracker/internal/db"
)

// ErrNotFound is returned when a lookup by primary key matches no row.
var ErrNotFound = errors.New("record not found")

// querier is the subset of *sql.DB and *sql.Tx the queries need, letting every
// method run either standalone or inside a transaction.
type querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Store provides access to the habit-tracker tables. A Store is safe for
// concurrent use; one bound to a transaction by Tx is not.
type Store struct {
	q     querier
	sqlDB *sql.DB
	now   func() time.Time
}

// New builds a Store over an open database handle.
func New(sqlDB *sql.DB) *Store {
	return &Store{q: sqlDB, sqlDB: sqlDB, now: time.Now}
}

// WithClock replaces the time source. Tests use it to pin "today" so that
// date-boundary behaviour is deterministic.
func (s *Store) WithClock(now func() time.Time) *Store {
	clone := *s
	clone.now = now

	return &clone
}

// Now returns the current instant from the store's clock, truncated to the
// second because that is the resolution the datetime columns preserve.
func (s *Store) Now() time.Time { return s.now().UTC().Truncate(time.Second) }

// Today returns the current day at midnight UTC.
func (s *Store) Today() time.Time {
	n := s.now()

	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}

// Tx runs fn inside a database transaction against a Store bound to it. The
// transaction commits when fn returns nil and rolls back on any error or panic,
// so a failed multi-step mutation leaves nothing half-written.
func (s *Store) Tx(ctx context.Context, fn func(*Store) error) error {
	if s.sqlDB == nil {
		return errors.New("store: nested transactions are not supported")
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	txStore := &Store{q: tx, sqlDB: nil, now: s.now}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(txStore); err != nil {
		_ = tx.Rollback()

		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// Ping confirms the database answers through the same single connection every
// request uses, so a health check fails if that connection is ever wedged.
func (s *Store) Ping(ctx context.Context) error {
	var one int

	return s.q.QueryRowContext(ctx, `SELECT 1`).Scan(&one)
}

// nullTime wraps a nullable instant for binding into a statement.
func nullTime(t *time.Time) any { return db.FormatTimePtr(t) }

// nullString wraps a nullable string for binding into a statement.
func nullString(s *string) any {
	if s == nil {
		return nil
	}

	return *s
}

// toStringPtr converts a scanned nullable column to a *string.
func toStringPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}

	value := ns.String

	return &value
}

// toInt64Ptr converts a scanned nullable column to an *int64.
func toInt64Ptr(ni sql.NullInt64) *int64 {
	if !ni.Valid {
		return nil
	}

	value := ni.Int64

	return &value
}
