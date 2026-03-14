package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/andresbott/etna/internal/accounting"
	"github.com/andresbott/etna/internal/backup"
	"github.com/andresbott/etna/internal/csvimport"
	"github.com/andresbott/etna/internal/marketdata"
	"github.com/andresbott/etna/internal/toolsdata"
	"github.com/go-bumbu/tempo"
)

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

// BackupTaskCfg holds the configuration for the scheduled backup task.
type BackupTaskCfg struct {
	// Store is the accounting store to export data from.
	Store *accounting.Store
	// MdStore is the market data store to export data from.
	MdStore *marketdata.Store
	// CsvStore is the CSV import store to export data from.
	CsvStore *csvimport.Store
	// ToolsDataStore is the tools data store to export data from.
	ToolsDataStore *toolsdata.Store
	// Destination is the directory where backup ZIP files are written.
	Destination string
	// Interval is how often the backup runs. Defaults to 24 hours.
	Interval time.Duration
	// Logger for backup task messages.
	Logger *slog.Logger
}

// NewBackupTaskFn returns the task function that performs the actual backup export.
// It can be used to enqueue a one-off backup from the API.
func NewBackupTaskFn(store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, tdStore *toolsdata.Store, destination string, l *slog.Logger) func(ctx context.Context) error {
	return newBackupFunc(store, mdStore, csvStore, tdStore, destination, l)
}

func newBackupFunc(store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, tdStore *toolsdata.Store, destination string, l *slog.Logger) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		now := time.Now().Format("2006-01-02_15-04")
		zipFile := filepath.Join(destination, fmt.Sprintf("backup-%s.zip", now))

		l.Info("starting backup",
			slog.String("component", "tasks"),
			slog.String("file", zipFile),
		)
		tempo.Info(ctx, fmt.Sprintf("starting backup: %s", zipFile))

		err := backup.ExportToFile(ctx, store, mdStore, csvStore, tdStore, zipFile)
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
