package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/andresbott/etna/internal/marketdata"
	"github.com/andresbott/etna/internal/marketdata/importer"
	"github.com/go-bumbu/tempo"
)

// TaskDef describes an available task for the API (list and trigger).
type TaskDef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
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

// ——— Shared helpers for market-data tasks ———

const dateFmt = "2006-01-02"

// dayNormalize returns the date (midnight UTC) for t.
func dayNormalize(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// daySetFromTimes returns a set of day-normalized times. Used to track which days already have data.
func daySetFromTimes(times []time.Time) map[time.Time]struct{} {
	m := make(map[time.Time]struct{}, len(times))
	for _, t := range times {
		m[dayNormalize(t)] = struct{}{}
	}
	return m
}

// daySetFromRecords builds a day set from a slice of records using getTime to extract the time. Used for PriceRecord/RateRecord.
func daySetFromRecords[T any](recs []T, getTime func(T) time.Time) map[time.Time]struct{} {
	times := make([]time.Time, len(recs))
	for i, r := range recs {
		times[i] = getTime(r)
	}
	return daySetFromTimes(times)
}

// dateRangeEnd returns end of "today" (start of next day) in UTC; dateRangeStart returns start of day N days before that.
func dateRangeEnd(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Add(24 * time.Hour)
}

func dateRangeStart(end time.Time, daysBack int) time.Time {
	return end.Add(-time.Duration(daysBack) * 24 * time.Hour)
}

// pricePointsNewDays returns price points whose day is not in existingDays, and adds those days to existingDays.
func pricePointsNewDays(points []importer.PricePoint, existingDays map[time.Time]struct{}) []marketdata.PricePoint {
	var out []marketdata.PricePoint
	for _, p := range points {
		day := dayNormalize(p.Time)
		if _, exists := existingDays[day]; !exists {
			out = append(out, marketdata.PricePoint{Time: p.Time, Price: p.Price})
			existingDays[day] = struct{}{}
		}
	}
	return out
}

// ratePointsNewDays returns rate points whose day is not in existingDays, and adds those days to existingDays.
func ratePointsNewDays(points []importer.RatePoint, existingDays map[time.Time]struct{}) []marketdata.RatePoint {
	var out []marketdata.RatePoint
	for _, p := range points {
		day := dayNormalize(p.Time)
		if _, exists := existingDays[day]; !exists {
			out = append(out, marketdata.RatePoint{Time: p.Time, Rate: p.Rate})
			existingDays[day] = struct{}{}
		}
	}
	return out
}

// FetchWith429Retry calls fetch until it returns a non-429 error or the deadline is exceeded.
// On 429 it waits Retry-After (or default) then retries. Logs wait messages via tempo.
// Returns the last fetch result and error (including deadline-exceeded).
func FetchWith429Retry[T any](ctx context.Context, deadline time.Time, logLabel string, fetch func() (T, error)) (T, error) {
	var zero T
	for {
		result, err := fetch()
		if err == nil {
			return result, nil
		}
		if !importer.IsRateLimit429(err) {
			return zero, err
		}
		retryAfter := importer.RetryAfterFrom429Err(err, importer.Default429RetryAfter)
		if time.Now().Add(retryAfter).After(deadline) {
			return zero, fmt.Errorf("429 retry would exceed deadline (%v): %w", retryAfter, err)
		}
		tempo.Info(ctx, fmt.Sprintf("%s: 429 — waiting %v before retry", logLabel, retryAfter))
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(retryAfter):
			// retry
		}
	}
}

// taskLogInfo writes the message to both tempo (task log) and optionally slog when l is non-nil.
func taskLogInfo(ctx context.Context, l *slog.Logger, taskName, msg string, slogAttrs ...slog.Attr) {
	tempo.Info(ctx, msg)
	if l != nil {
		attrs := append([]slog.Attr{
			slog.String("component", "tasks"),
			slog.String("task", taskName),
		}, slogAttrs...)
		l.LogAttrs(context.Background(), slog.LevelInfo, msg, attrs...)
	}
}

func taskLogWarn(ctx context.Context, l *slog.Logger, taskName, msg string, slogAttrs ...slog.Attr) {
	tempo.Info(ctx, msg) // tempo has no warn; use Info for consistency
	if l != nil {
		attrs := append([]slog.Attr{
			slog.String("component", "tasks"),
			slog.String("task", taskName),
		}, slogAttrs...)
		l.LogAttrs(context.Background(), slog.LevelWarn, msg, attrs...)
	}
}

func taskLogError(ctx context.Context, l *slog.Logger, taskName, msg string, slogAttrs ...slog.Attr) {
	tempo.Error(ctx, msg)
	if l != nil {
		attrs := append([]slog.Attr{
			slog.String("component", "tasks"),
			slog.String("task", taskName),
		}, slogAttrs...)
		l.LogAttrs(context.Background(), slog.LevelError, msg, attrs...)
	}
}
