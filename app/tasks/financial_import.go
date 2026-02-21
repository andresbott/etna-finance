package tasks

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/andresbott/etna/internal/marketdata"
)

const FinancialImportTaskName = "financial-import"

// FinancialImportTaskDef is the task definition for the financial import task, used in the API task list.
var FinancialImportTaskDef = TaskDef{
	ID:          FinancialImportTaskName,
	Name:        "Financial import",
	Description: "Run market data maintenance (retention and aggregation).",
}

// NewFinancialImportTaskFn returns a task function that runs market data maintenance
// (retention cleanup and bucket aggregation). Suitable for periodic financial data import pipelines.
func NewFinancialImportTaskFn(store *marketdata.Store, l *slog.Logger) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if store == nil {
			return fmt.Errorf("market data store is required")
		}
		if l != nil {
			l.Info("starting financial import (maintenance)",
				slog.String("component", "tasks"),
				slog.String("task", FinancialImportTaskName),
			)
		}
		err := store.Maintenance(ctx)
		if err != nil {
			if l != nil {
				l.Error("financial import failed",
					slog.String("component", "tasks"),
					slog.String("task", FinancialImportTaskName),
					slog.String("error", err.Error()),
				)
			}
			return fmt.Errorf("financial import failed: %w", err)
		}
		if l != nil {
			l.Info("financial import completed",
				slog.String("component", "tasks"),
				slog.String("task", FinancialImportTaskName),
			)
		}
		return nil
	}
}
