package tasks

import (
	"context"
	"log/slog"
	"math/rand"
	"time"
)

const LogOnlyTaskName = "log-only"

// LogOnlyTaskDef is the task definition for the log-only task (no-op job that just logs).
var LogOnlyTaskDef = TaskDef{
	ID:          LogOnlyTaskName,
	Name:        "Log only",
	Description: "No-op job that logs after a random delay up to 1s (for testing schedules).",
}

// NewLogOnlyTaskFn returns a task function that sleeps randomly up to 1 second then logs.
func NewLogOnlyTaskFn(l *slog.Logger) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		d := time.Duration(rand.Int63n(int64(time.Second)))
		if d > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(d):
			}
		}
		if l != nil {
			l.Info("log-only job executed",
				slog.String("component", "tasks"),
				slog.String("task", LogOnlyTaskName),
			)
		}
		return nil
	}
}

const LogOnlyLongTaskName = "log-only-long"

// LogOnlyLongTaskDef is the task definition for the long-running log-only task.
var LogOnlyLongTaskDef = TaskDef{
	ID:          LogOnlyLongTaskName,
	Name:        "Log only (long)",
	Description: "Simulates processing 10–50 items with a log per item and short delays (total up to ~5 min).",
}

// NewLogOnlyLongTaskFn returns a task function that simulates processing multiple items:
// random number of items (10–50), logs progress per item with a short random delay between, total up to ~5 min.
func NewLogOnlyLongTaskFn(l *slog.Logger) func(ctx context.Context) error {
	const minItems, maxItems = 10, 50
	const maxSleepPerItem = 6 * time.Second
	return func(ctx context.Context) error {
		numItems := minItems + rand.Intn(maxItems-minItems+1)
		if l != nil {
			l.Info("log-only-long job started",
				slog.String("component", "tasks"),
				slog.String("task", LogOnlyLongTaskName),
				slog.Int("items", numItems),
			)
		}
		for i := 1; i <= numItems; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if l != nil {
				l.Info("processing item",
					slog.String("component", "tasks"),
					slog.String("task", LogOnlyLongTaskName),
					slog.Int("item", i),
					slog.Int("total", numItems),
				)
			}
			d := time.Duration(rand.Int63n(int64(maxSleepPerItem)))
			if d > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(d):
				}
			}
		}
		if l != nil {
			l.Info("log-only-long job completed",
				slog.String("component", "tasks"),
				slog.String("task", LogOnlyLongTaskName),
				slog.Int("items_processed", numItems),
			)
		}
		return nil
	}
}
