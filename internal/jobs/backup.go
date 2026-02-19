package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/andresbott/etna/internal/accounting"
	"github.com/andresbott/etna/internal/backup"
)

const backupJobName = "scheduled-backup"

// BackupJobCfg holds the configuration for the scheduled backup job.
type BackupJobCfg struct {
	// Store is the accounting store to export data from.
	Store *accounting.Store
	// Destination is the directory where backup ZIP files are written.
	Destination string
	// Interval is how often the backup runs. Defaults to 24 hours.
	Interval time.Duration
	// Logger for backup job messages.
	Logger *slog.Logger
}

// ScheduleBackup starts a goroutine that periodically enqueues a backup job.
// It runs the first backup immediately, then repeats at the configured interval.
// The goroutine stops when ctx is cancelled.
func (r *Runner) ScheduleBackup(ctx context.Context, cfg BackupJobCfg) error {
	if cfg.Store == nil {
		return fmt.Errorf("accounting store is required")
	}
	if cfg.Destination == "" {
		return fmt.Errorf("backup destination is required")
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 24 * time.Hour
	}

	l := cfg.Logger
	if l == nil {
		l = r.logger
	}

	go func() {
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()

		enqueue := func() {
			err := r.Enqueue(newBackupFunc(cfg.Store, cfg.Destination, l), backupJobName)
			if err != nil {
				l.Error("failed to enqueue backup job",
					slog.String("component", "jobs"),
					slog.String("error", err.Error()),
				)
			}
		}

		// Run the first backup immediately.
		enqueue()

		for {
			select {
			case <-ctx.Done():
				l.Info("scheduled backup stopped", slog.String("component", "jobs"))
				return
			case <-ticker.C:
				enqueue()
			}
		}
	}()

	l.Info("scheduled backup configured",
		slog.String("component", "jobs"),
		slog.Duration("interval", cfg.Interval),
		slog.String("destination", cfg.Destination),
	)

	return nil
}

// newBackupFunc returns the job function that performs the actual backup export.
func newBackupFunc(store *accounting.Store, destination string, l *slog.Logger) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		now := time.Now().Format("2006-01-02_15-04")
		zipFile := filepath.Join(destination, fmt.Sprintf("backup-%s.zip", now))

		l.Info("starting backup",
			slog.String("component", "jobs"),
			slog.String("file", zipFile),
		)

		err := backup.ExportToFile(ctx, store, zipFile)
		if err != nil {
			l.Error("backup failed",
				slog.String("component", "jobs"),
				slog.String("error", err.Error()),
			)
			return fmt.Errorf("backup export failed: %w", err)
		}

		l.Info("backup completed",
			slog.String("component", "jobs"),
			slog.String("file", zipFile),
		)
		return nil
	}
}
