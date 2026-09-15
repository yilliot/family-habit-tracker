package backup

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/el/family-habit-tracker/internal/db"
)

func openDB(t *testing.T) *sql.DB {
	t.Helper()

	sqlDB, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "live.sqlite"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	t.Cleanup(func() { sqlDB.Close() })

	return sqlDB
}

func countRows(t *testing.T, path, table string) int {
	t.Helper()

	check, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer check.Close()

	var count int
	if err := check.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
		t.Fatalf("count %s in %s: %v", table, path, err)
	}

	return count
}

func touch(t *testing.T, path string) {
	t.Helper()

	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
}

func TestSnapshotIsACompleteStandaloneCopy(t *testing.T) {
	sqlDB := openDB(t)

	for _, name := range []string{"爸爸", "妈妈", "可遇", "奇乐"} {
		if _, err := sqlDB.Exec(`INSERT INTO people (name, display_order) VALUES (?, 1)`, name); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	path := filepath.Join(t.TempDir(), "nested", Name(time.Now()))

	if err := Snapshot(t.Context(), sqlDB, path); err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	if got := countRows(t, path, "people"); got != 4 {
		t.Errorf("snapshot holds %d people, want 4", got)
	}

	if _, err := os.Stat(path + tmpSuffix); !os.IsNotExist(err) {
		t.Error("no temporary file should remain after a successful snapshot")
	}

	// A snapshot must be usable as the app's database, which is what a restore
	// actually does with it.
	restored, err := db.Open(t.Context(), path)
	if err != nil {
		t.Fatalf("open snapshot as the app would: %v", err)
	}
	restored.Close()
}

// Snapshots taken while the family is ticking must each be sound, and none may
// go back in time.
func TestSnapshotsStaySoundWhileWritesContinue(t *testing.T) {
	sqlDB := openDB(t)
	dir := t.TempDir()

	ctx, stop := context.WithCancel(t.Context())

	var writer sync.WaitGroup
	writer.Add(1)

	go func() {
		defer writer.Done()

		for ctx.Err() == nil {
			sqlDB.ExecContext(ctx, `INSERT INTO people (name, display_order) VALUES ('tick', 1)`)
		}
	}()

	previous := -1

	for i := range 5 {
		path := filepath.Join(dir, Name(time.Now().Add(time.Duration(i)*time.Second)))

		if err := Snapshot(t.Context(), sqlDB, path); err != nil {
			stop()
			writer.Wait()
			t.Fatalf("snapshot %d under load: %v", i, err)
		}

		count := countRows(t, path, "people")
		if count < previous {
			t.Errorf("snapshot %d has %d rows, fewer than the %d before it", i, count, previous)
		}

		previous = count
	}

	stop()
	writer.Wait()
}

func TestPruneKeepsOnlyTheNewestSnapshots(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 1, 1, 3, 0, 0, 0, time.UTC)

	for day := range 20 {
		touch(t, filepath.Join(dir, Name(base.AddDate(0, 0, day))))
	}

	// Files that must survive: the live database, unrelated notes, and a name
	// that looks like a snapshot but carries no valid timestamp.
	decoys := []string{"habits.sqlite", "notes.txt", "snapshot-not-a-date.sqlite"}
	for _, decoy := range decoys {
		touch(t, filepath.Join(dir, decoy))
	}

	stale := filepath.Join(dir, Name(base.AddDate(0, 0, -2))+tmpSuffix)
	touch(t, stale)

	removed, err := Prune(dir, 14)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}

	if len(removed) != 7 {
		t.Errorf("removed %d files, want 7 (6 old snapshots and 1 stale temporary)", len(removed))
	}

	for day := range 20 {
		_, err := os.Stat(filepath.Join(dir, Name(base.AddDate(0, 0, day))))
		kept := err == nil

		if day < 6 && kept {
			t.Errorf("day %d should have been pruned", day)
		}
		if day >= 6 && !kept {
			t.Errorf("day %d is among the newest 14 and should have been kept", day)
		}
	}

	for _, decoy := range decoys {
		if _, err := os.Stat(filepath.Join(dir, decoy)); err != nil {
			t.Errorf("prune must never touch %s", decoy)
		}
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("a half-written snapshot from an interrupted run should be cleared")
	}
}

func TestPruneNeverDeletesEverySnapshot(t *testing.T) {
	dir := t.TempDir()
	newest := Name(time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC))

	touch(t, filepath.Join(dir, Name(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))))
	touch(t, filepath.Join(dir, newest))

	if _, err := Prune(dir, 0); err != nil {
		t.Fatalf("prune: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, newest)); err != nil {
		t.Error("a misconfigured retention of zero must still keep the newest snapshot")
	}
}

func TestUntilDue(t *testing.T) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	interval := 24 * time.Hour

	cases := []struct {
		name     string
		snapshot *time.Time
		want     time.Duration
	}{
		{"no snapshots yet", nil, 0},
		{"newest is three hours old", ptr(now.Add(-3 * time.Hour)), 21 * time.Hour},
		{"newest is overdue", ptr(now.Add(-25 * time.Hour)), 0},
		{"clock jumped backwards past the newest", ptr(now.Add(72 * time.Hour)), interval},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			if testCase.snapshot != nil {
				touch(t, filepath.Join(dir, Name(*testCase.snapshot)))
			}

			got, err := untilDue(dir, interval, now)
			if err != nil {
				t.Fatalf("untilDue: %v", err)
			}

			if got != testCase.want {
				t.Errorf("untilDue = %s, want %s", got, testCase.want)
			}
		})
	}
}

func TestRunTakesADueSnapshotAndStopsWhenCancelled(t *testing.T) {
	sqlDB := openDB(t)
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})

	go func() {
		defer close(done)
		Run(ctx, sqlDB, Schedule{Dir: dir, Keep: 3, Interval: time.Hour}, logger)
	}()

	deadline := time.After(5 * time.Second)
	for snapshots(t, dir) == 0 {
		select {
		case <-deadline:
			cancel()
			t.Fatal("no snapshot was taken although one was due")
		case <-time.After(20 * time.Millisecond):
		}
	}

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not stop after its context was cancelled")
	}

	if got := snapshots(t, dir); got != 1 {
		t.Errorf("snapshots = %d, want exactly 1", got)
	}
}

// A restart shortly after a snapshot must not take another. Otherwise a day of
// repeated deploys would each take one and push the older days out of
// retention.
func TestRunWaitsWhileTheNewestSnapshotIsRecent(t *testing.T) {
	sqlDB := openDB(t)
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	touch(t, filepath.Join(dir, Name(time.Now().Add(-time.Minute))))

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})

	go func() {
		defer close(done)
		Run(ctx, sqlDB, Schedule{Dir: dir, Keep: 3, Interval: 24 * time.Hour}, logger)
	}()

	<-time.After(300 * time.Millisecond)
	cancel()
	<-done

	if got := snapshots(t, dir); got != 1 {
		t.Errorf("snapshots = %d, want 1: a fresh snapshot already existed", got)
	}
}

func snapshots(t *testing.T, dir string) int {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}

	count := 0
	for _, entry := range entries {
		if _, ours := parseName(entry.Name()); ours {
			count++
		}
	}

	return count
}

func ptr(t time.Time) *time.Time { return &t }
