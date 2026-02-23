package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/andresbott/etna/internal/accounting"
	"github.com/andresbott/etna/internal/backup"
	"github.com/go-bumbu/tempo"
)

// TaskDef describes an available task for the API (list and trigger).
type TaskDef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

const (
	BackupTaskName      = "backup"
	scheduledBackupName = "scheduled-backup"
)

// BackupTaskDef is the task definition for the backup task, used in the API task list.
var BackupTaskDef = TaskDef{
	ID:          BackupTaskName,
	Name:        "Backup",
	Description: "Export accounting and financial data to a ZIP file.",
}

// AvailableTasks is the full list of task definitions (including dev-only). Use AvailableTaskDefs(production) to filter.
var AvailableTasks = []TaskDef{BackupTaskDef, FinancialImportTaskDef, FinancialBackfillTaskDef, FXImportTaskDef, FXBackfillTaskDef, LogOnlyTaskDef, LogOnlyLongTaskDef, DebugFailTaskDef}

// DevOnlyTaskIDs are task IDs hidden in production (non-prod only).
var DevOnlyTaskIDs = map[string]bool{
	LogOnlyTaskName:     true,
	LogOnlyLongTaskName: true,
	DebugFailTaskName:   true,
}

// AvailableTaskDefs returns task definitions visible for the given environment. When production is true, dev-only tasks are excluded.
func AvailableTaskDefs(production bool) []TaskDef {
	if !production {
		return AvailableTasks
	}
	out := make([]TaskDef, 0, len(AvailableTasks))
	for _, t := range AvailableTasks {
		if !DevOnlyTaskIDs[t.ID] {
			out = append(out, t)
		}
	}
	return out
}

// TaskNameExists returns true if taskName is a known task ID visible in the given environment (for schedule API validation).
func TaskNameExists(taskName string, production bool) bool {
	for _, t := range AvailableTaskDefs(production) {
		if t.ID == taskName {
			return true
		}
	}
	return false
}

// BackupTaskCfg holds the configuration for the scheduled backup task.
type BackupTaskCfg struct {
	// Store is the accounting store to export data from.
	Store *accounting.Store
	// Destination is the directory where backup ZIP files are written.
	Destination string
	// Interval is how often the backup runs. Defaults to 24 hours.
	Interval time.Duration
	// Logger for backup task messages.
	Logger *slog.Logger
}

// NewBackupTaskFn returns the task function that performs the actual backup export.
// It can be used to enqueue a one-off backup from the API.
func NewBackupTaskFn(store *accounting.Store, destination string, l *slog.Logger) func(ctx context.Context) error {
	return newBackupFunc(store, destination, l)
}

func newBackupFunc(store *accounting.Store, destination string, l *slog.Logger) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		now := time.Now().Format("2006-01-02_15-04")
		zipFile := filepath.Join(destination, fmt.Sprintf("backup-%s.zip", now))

		l.Info("starting backup",
			slog.String("component", "tasks"),
			slog.String("file", zipFile),
		)
		tempo.Info(ctx, fmt.Sprintf("starting backup: %s", zipFile))

		err := backup.ExportToFile(ctx, store, zipFile)
		if err != nil {
			l.Error("backup failed",
				slog.String("component", "tasks"),
				slog.String("error", err.Error()),
			)
			tempo.Error(ctx, fmt.Sprintf("backup failed: %v", err))
			return fmt.Errorf("backup export failed: %w", err)
		}

		l.Info("backup completed",
			slog.String("component", "tasks"),
			slog.String("file", zipFile),
		)
		tempo.Info(ctx, fmt.Sprintf("backup completed: %s", zipFile))
		return nil
	}
}
