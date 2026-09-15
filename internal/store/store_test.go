package store_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/el/family-habit-tracker/internal/db"
	"github.com/el/family-habit-tracker/internal/store"
)

// The database is a single connection. Code that reaches for the outer store
// while a transaction holds that connection waits for it forever — a hazard
// easy to introduce in a future edit. This pins the property the request
// deadline relies on: the wait honours its context, and once it gives up the
// transaction rolls back and the connection comes back into service.
func TestWaitingForTheConnectionHonoursItsDeadline(t *testing.T) {
	sqlDB, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "deadline.sqlite"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()

	outer := store.New(sqlDB)
	started := time.Now()

	err = outer.Tx(t.Context(), func(*store.Store) error {
		ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
		defer cancel()

		// Deliberately the outer store, not the transaction's.
		_, err := outer.People(ctx)

		return err
	})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}

	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("gave up after %s; the deadline should have ended the wait", elapsed)
	}

	if _, err := outer.People(t.Context()); err != nil {
		t.Fatalf("the connection should be usable again after the stuck transaction: %v", err)
	}
}

func TestPingReportsAClosedDatabase(t *testing.T) {
	sqlDB, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "ping.sqlite"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	dataStore := store.New(sqlDB)

	if err := dataStore.Ping(t.Context()); err != nil {
		t.Fatalf("ping an open database: %v", err)
	}

	sqlDB.Close()

	if err := dataStore.Ping(t.Context()); err == nil {
		t.Error("ping should fail once the database is unavailable")
	}
}
