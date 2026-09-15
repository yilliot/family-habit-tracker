package db_test

import (
	"testing"

	"github.com/el/family-habit-tracker/internal/db"
)

// The settings that make a write survive a crash or power cut are applied per
// connection through the DSN. A typo there fails silently, so assert that each
// one is actually in effect rather than trusting the connection string.
func TestDurabilitySettingsAreInEffect(t *testing.T) {
	sqlDB, err := db.Open(t.Context(), t.TempDir()+"/durable.sqlite")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()

	var journalMode string
	if err := sqlDB.QueryRowContext(t.Context(), `PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		t.Fatalf("journal_mode: %v", err)
	}

	var synchronous, foreignKeys, busyTimeout int
	for pragma, target := range map[string]*int{"synchronous": &synchronous, "foreign_keys": &foreignKeys, "busy_timeout": &busyTimeout} {
		if err := sqlDB.QueryRowContext(t.Context(), `PRAGMA `+pragma).Scan(target); err != nil {
			t.Fatalf("%s: %v", pragma, err)
		}
	}

	t.Logf("journal_mode=%s synchronous=%d foreign_keys=%d busy_timeout=%d", journalMode, synchronous, foreignKeys, busyTimeout)

	if journalMode != "wal" {
		t.Errorf("journal_mode = %q, want wal", journalMode)
	}

	// 2 is FULL: every commit is fsynced before it is acknowledged, so a power
	// cut can never lose a tick the tablet already showed as saved.
	if synchronous != 2 {
		t.Errorf("synchronous = %d, want 2 (FULL)", synchronous)
	}

	if foreignKeys != 1 {
		t.Errorf("foreign_keys = %d, want 1", foreignKeys)
	}

	if busyTimeout != 5000 {
		t.Errorf("busy_timeout = %d, want 5000", busyTimeout)
	}
}
