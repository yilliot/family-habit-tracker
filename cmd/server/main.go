// Command server runs the family habit tracker: a single binary containing the
// application, its templates, and its static assets, storing everything in one
// SQLite file.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/el/family-habit-tracker/internal/backup"
	"github.com/el/family-habit-tracker/internal/db"
	"github.com/el/family-habit-tracker/internal/service"
	"github.com/el/family-habit-tracker/internal/store"
	"github.com/el/family-habit-tracker/internal/web"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		addr       = flag.String("addr", envOr("HT_ADDR", "127.0.0.1:8080"), "address to listen on")
		dbPath     = flag.String("db", envOr("HT_DB", "habits.sqlite"), "path to the SQLite database file")
		tzName     = flag.String("tz", envOr("HT_TZ", "Local"), "IANA timezone deciding which day counts as today (e.g. Asia/Singapore)")
		seed       = flag.Bool("seed", false, "seed the four family members if the people table is empty, then exit")
		backupDir  = flag.String("backup-dir", envOr("HT_BACKUP_DIR", ""), "directory for automatic daily snapshots; empty disables them")
		backupKeep = flag.Int("backup-keep", envIntOr("HT_BACKUP_KEEP", 14), "number of daily snapshots to retain")
		backupTo   = flag.String("backup", "", "write one verified snapshot to this path, then exit")
		showHelp   = flag.Bool("help", false, "show usage")
	)

	flag.Parse()

	if *showHelp {
		flag.Usage()

		return nil
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sqlDB, err := db.Open(ctx, *dbPath)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	if *backupTo != "" {
		return backupOnce(ctx, sqlDB, *backupTo, logger)
	}

	// Which day "today" is decides which cells are tappable, so it has to be
	// the family's own day, not the server's. Timestamps are still stored in
	// UTC; only the date boundary is local.
	location, err := time.LoadLocation(*tzName)
	if err != nil {
		return fmt.Errorf("unknown timezone %q: %w", *tzName, err)
	}

	dataStore := store.New(sqlDB).WithClock(func() time.Time { return time.Now().In(location) })
	svc := service.New(dataStore)

	logger.Info("clock configured", "timezone", location.String(), "today", dataStore.Today().Format("2006-01-02"))

	if *seed {
		return seedFamily(ctx, svc, logger)
	}

	// Snapshots run in the background on their own schedule. The channel lets
	// shutdown wait for one that is mid-write rather than closing the database
	// underneath it.
	backupsDone := make(chan struct{})

	if *backupDir != "" {
		schedule := backup.Schedule{Dir: *backupDir, Keep: *backupKeep, Interval: 24 * time.Hour}
		logger.Info("automatic backups on", "dir", schedule.Dir, "keep", schedule.Keep, "every", schedule.Interval)

		go func() {
			defer close(backupsDone)
			backup.Run(ctx, sqlDB, schedule, logger)
		}()
	} else {
		close(backupsDone)
		logger.Warn("automatic backups are OFF; set -backup-dir to keep daily snapshots of the family's history")
	}

	server := &http.Server{
		Addr:              *addr,
		Handler:           web.NewServer(svc, logger).Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errs := make(chan error, 1)

	go func() {
		logger.Info("family habit tracker listening", "addr", *addr, "db", *dbPath)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errs <- err
		}
	}()

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
	}

	logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = server.Shutdown(shutdownCtx)

	select {
	case <-backupsDone:
	case <-shutdownCtx.Done():
		logger.Warn("stopped waiting for an in-flight backup")
	}

	return err
}

// backupOnce writes a single snapshot to an explicit path, for use before a
// deploy or migration. It refuses to overwrite an existing file, so a
// previous backup cannot be clobbered by a repeated command.
func backupOnce(ctx context.Context, sqlDB *sql.DB, path string, logger *slog.Logger) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("refusing to overwrite existing file %s", path)
	}

	started := time.Now()

	if err := backup.Snapshot(ctx, sqlDB, path); err != nil {
		return err
	}

	logger.Info("backup written and verified", "path", path, "took", time.Since(started).Round(time.Millisecond))

	return nil
}

// seedFamily populates a brand-new database with the four family members, in
// the order the ledger prints them. It is idempotent: a database that already
// has people is left untouched.
func seedFamily(ctx context.Context, svc *service.Service, logger *slog.Logger) error {
	existing, err := svc.Store().PeopleWithTrashed(ctx)
	if err != nil {
		return err
	}

	if len(existing) > 0 {
		logger.Info("database already has people, nothing to seed", "count", len(existing))

		return nil
	}

	for index, name := range []string{"爸爸", "妈妈", "可遇", "奇乐"} {
		order := index + 1

		if _, err := svc.CreatePerson(ctx, name, &order); err != nil {
			return fmt.Errorf("seed %s: %w", name, err)
		}
	}

	logger.Info("seeded the family")

	return nil
}

// envIntOr reads an integer environment variable, falling back to a default
// when it is unset or not a number.
func envIntOr(key string, fallback int) int {
	if value, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return value
	}

	return fallback
}

// envOr reads an environment variable, falling back to a default.
func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
