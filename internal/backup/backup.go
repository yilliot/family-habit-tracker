// Package backup keeps verified point-in-time copies of the database, so that
// years of family history survive a bad deploy, an accidental deletion, or a
// damaged file.
package backup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	namePrefix = "snapshot-"
	nameSuffix = ".sqlite"
	nameStamp  = "20060102T150405Z"
	tmpSuffix  = ".tmp"
)

// retryDelay is how long to wait after a failed snapshot before trying again:
// soon enough that a passing problem, such as a briefly full disk, costs an
// hour rather than a day, and slow enough not to spin.
var retryDelay = time.Hour

// Schedule is the policy for automatic snapshots.
type Schedule struct {
	// Dir holds the snapshots. It is created if missing.
	Dir string
	// Keep is how many of the newest snapshots to retain. At least one is
	// always kept.
	Keep int
	// Interval is the time between snapshots.
	Interval time.Duration
}

// Name returns the file name for a snapshot taken at t. The UTC timestamp in
// the name, not the file's modification time, orders snapshots, so they still
// sort correctly after being copied somewhere that does not preserve mtimes.
func Name(t time.Time) string {
	return namePrefix + t.UTC().Format(nameStamp) + nameSuffix
}

// parseName extracts the timestamp from a snapshot file name, reporting whether
// the name is one of ours at all.
func parseName(name string) (time.Time, bool) {
	if !strings.HasPrefix(name, namePrefix) || !strings.HasSuffix(name, nameSuffix) {
		return time.Time{}, false
	}

	stamp := strings.TrimSuffix(strings.TrimPrefix(name, namePrefix), nameSuffix)

	at, err := time.ParseInLocation(nameStamp, stamp, time.UTC)
	if err != nil {
		return time.Time{}, false
	}

	return at, true
}

// Snapshot writes a standalone copy of the live database to path.
//
// VACUUM INTO reads inside a single transaction, so the copy is one coherent
// moment even while the family is ticking. It is written under a temporary
// name, integrity-checked, flushed to disk, and only then renamed into place.
// A file at path is therefore always a complete, verified backup, never a
// half-written one that would only be discovered on the day it was needed.
func Snapshot(ctx context.Context, sqlDB *sql.DB, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}

	tmp := path + tmpSuffix
	if err := os.Remove(tmp); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("clear stale %s: %w", tmp, err)
	}

	if _, err := sqlDB.ExecContext(ctx, `VACUUM INTO ?`, tmp); err != nil {
		os.Remove(tmp)

		return fmt.Errorf("write snapshot: %w", err)
	}

	if err := verify(ctx, tmp); err != nil {
		os.Remove(tmp)

		return err
	}

	if err := syncFile(tmp); err != nil {
		os.Remove(tmp)

		return fmt.Errorf("flush snapshot: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)

		return fmt.Errorf("finalise snapshot: %w", err)
	}

	// Persist the rename itself. Not every platform supports syncing a
	// directory, and the snapshot is already complete either way.
	_ = syncDir(filepath.Dir(path))

	return nil
}

// verify opens a snapshot on its own and confirms SQLite considers it sound.
func verify(ctx context.Context, path string) error {
	check, err := sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("open snapshot for verification: %w", err)
	}
	defer check.Close()

	var result string
	if err := check.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&result); err != nil {
		return fmt.Errorf("verify snapshot: %w", err)
	}

	if result != "ok" {
		return fmt.Errorf("snapshot failed its integrity check: %s", result)
	}

	return nil
}

func syncFile(path string) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer file.Close()

	return file.Sync()
}

func syncDir(dir string) error {
	handle, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer handle.Close()

	return handle.Sync()
}

// Prune deletes all but the newest keep snapshots in dir, and clears any
// half-written file an interrupted snapshot left behind. It touches nothing
// whose name is not a snapshot's, so a directory that also holds other files is
// safe to point it at. It always keeps at least one snapshot.
//
// Prune must not run concurrently with Snapshot in the same directory; Run
// guarantees that by doing one after the other.
func Prune(dir string, keep int) ([]string, error) {
	keep = max(keep, 1)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read backup directory: %w", err)
	}

	type snapshot struct {
		name string
		at   time.Time
	}

	var (
		snapshots []snapshot
		removed   []string
	)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		if strings.HasSuffix(name, nameSuffix+tmpSuffix) {
			if _, ours := parseName(strings.TrimSuffix(name, tmpSuffix)); ours {
				if err := os.Remove(filepath.Join(dir, name)); err == nil {
					removed = append(removed, name)
				}
			}

			continue
		}

		if at, ours := parseName(name); ours {
			snapshots = append(snapshots, snapshot{name: name, at: at})
		}
	}

	sort.Slice(snapshots, func(i, j int) bool { return snapshots[i].at.Before(snapshots[j].at) })

	for len(snapshots) > keep {
		oldest := snapshots[0]
		snapshots = snapshots[1:]

		if err := os.Remove(filepath.Join(dir, oldest.name)); err != nil {
			return removed, fmt.Errorf("remove old snapshot %s: %w", oldest.name, err)
		}

		removed = append(removed, oldest.name)
	}

	return removed, nil
}

// untilDue reports how long until the next snapshot is due, judged from the
// newest snapshot already in dir. Zero means one is due now.
//
// The wait never exceeds one interval, so a snapshot stamped in the future by a
// clock that later jumped backwards cannot postpone backups indefinitely.
func untilDue(dir string, interval time.Duration, now time.Time) (time.Duration, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read backup directory: %w", err)
	}

	var newest time.Time
	for _, entry := range entries {
		if at, ours := parseName(entry.Name()); ours && at.After(newest) {
			newest = at
		}
	}

	if newest.IsZero() {
		return 0, nil
	}

	return min(max(newest.Add(interval).Sub(now), 0), interval), nil
}

// Run takes a snapshot whenever one is due and prunes to the retention limit,
// returning when ctx is cancelled.
//
// Whether a snapshot is due is judged from the files already on disk, not from
// when the process started. Otherwise every restart would take one, and a busy
// afternoon of deploys would push out the older days that retention exists to
// keep.
func Run(ctx context.Context, sqlDB *sql.DB, schedule Schedule, logger *slog.Logger) {
	for {
		wait, err := untilDue(schedule.Dir, schedule.Interval, time.Now())
		if err != nil {
			logger.Error("backup: cannot read the backup directory", "dir", schedule.Dir, "error", err)
			wait = retryDelay
		}

		if !pause(ctx, wait) {
			return
		}

		path := filepath.Join(schedule.Dir, Name(time.Now()))
		started := time.Now()

		if err := Snapshot(ctx, sqlDB, path); err != nil {
			if ctx.Err() != nil {
				return
			}

			logger.Error("backup failed; will retry", "retry_in", retryDelay, "error", err)

			if !pause(ctx, retryDelay) {
				return
			}

			continue
		}

		removed, err := Prune(schedule.Dir, schedule.Keep)
		if err != nil {
			logger.Error("backup: removing old snapshots failed", "error", err)
		}

		logger.Info("backup written", "path", path, "took", time.Since(started).Round(time.Millisecond), "pruned", len(removed))
	}
}

// pause waits for d, reporting false if ctx was cancelled first.
func pause(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
