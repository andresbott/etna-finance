package tasks

import (
	"context"
	"fmt"
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

// sleepContext blocks for up to d or until ctx is cancelled. Returns ctx.Err() if cancelled.
func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// NewLogOnlyLongTaskFn returns a task function that simulates processing multiple items:
// random number of items (10–50), logs progress per item with a short random delay between, total up to ~5 min.
// It respects context cancellation so the runner can stop the execution.
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
			if err := ctx.Err(); err != nil {
				return err
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
			if err := sleepContext(ctx, d); err != nil {
				return err
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

const DebugFailTaskName = "debug-fail"

// DebugFailTaskDef is the task definition for the debug task that errors after a short delay.
var DebugFailTaskDef = TaskDef{
	ID:          DebugFailTaskName,
	Name:        "Debug fail",
	Description: "Runs for 1–2 seconds then returns an error (for testing failed execution).",
}

// NewDebugFailTaskFn returns a task function that sleeps 1–2 seconds then returns an error.
// Respects context cancellation.
func NewDebugFailTaskFn(l *slog.Logger) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		d := time.Second + time.Duration(rand.Int63n(int64(time.Second)))
		if l != nil {
			l.Info("debug-fail job started, will error after delay",
				slog.String("component", "tasks"),
				slog.String("task", DebugFailTaskName),
				slog.Duration("delay", d),
			)
		}
		if err := sleepContext(ctx, d); err != nil {
			return err
		}
		err := fmt.Errorf("debug-fail: intentional error after %v", d)
		if l != nil {
			l.Info("debug-fail job erroring",
				slog.String("component", "tasks"),
				slog.String("task", DebugFailTaskName),
				slog.String("error", err.Error()),
			)
		}
		return err
	}
}
