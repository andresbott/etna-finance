package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/andresbott/etna/internal/marketdata"
	"github.com/andresbott/etna/internal/marketdata/importer"
	"github.com/go-bumbu/tempo"
)

const FinancialImportTaskName = "financial-import"

// BackfillDays is the number of calendar days of history to backfill per instrument when an importer client is configured.
const BackfillDays = 7

// MaxTaskDuration is the maximum duration for the financial import task (backfill + maintenance).
// On 429 rate limit, the task waits and retries only if the retry time is before this deadline; otherwise it fails fast.
const MaxTaskDuration = 1 * time.Hour

// tradingDaysInRange returns the number of weekdays (Mon–Fri) in [start, end).
// Used to decide if we already have enough data to skip the API (markets are closed on weekends).
func tradingDaysInRange(start, end time.Time) int {
	n := 0
	for d := start; d.Before(end); d = d.Add(24 * time.Hour) {
		w := d.Weekday()
		if w != time.Sunday && w != time.Saturday {
			n++
		}
	}
	return n
}

// lastTradingDayInRange returns the last weekday in [start, end), or zero time if none.
// Used to skip the API when we already have data through that day.
func lastTradingDayInRange(start, end time.Time) time.Time {
	var last time.Time
	for d := start; d.Before(end); d = d.Add(24 * time.Hour) {
		w := d.Weekday()
		if w != time.Sunday && w != time.Saturday {
			last = d
		}
	}
	return last
}

// FinancialImportTaskDef is the task definition for the financial import task, used in the API task list.
var FinancialImportTaskDef = TaskDef{
	ID:          FinancialImportTaskName,
	Name:        "Financial import",
	Description: "Run market data maintenance (retention and aggregation).",
}

// NewFinancialImportTaskFn returns a task function that runs market data maintenance
// (retention cleanup and bucket aggregation). When client is non-nil, it first backfills
// BackfillDays of data for every configured instrument using the importer pool, then runs maintenance.
// It does not log to the default slog logger; use tempo (e.g. task logs) for observability. Exception: debugger/backfill jobs may pass a logger.
func NewFinancialImportTaskFn(store *marketdata.Store, client importer.Client) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if store == nil {
			return fmt.Errorf("market data store is required")
		}
		tempo.Info(ctx, "starting financial import")

		if client != nil {
			instruments, err := store.ListInstruments(ctx)
			if err != nil {
				tempo.Error(ctx, fmt.Sprintf("financial import: list instruments failed: %v", err))
				return fmt.Errorf("list instruments: %w", err)
			}
			now := time.Now().UTC()
			end := dateRangeEnd(now)
			start := dateRangeStart(end, BackfillDays)
			tempo.Info(ctx, fmt.Sprintf("financial import: backfill range %s to %s, %d instruments", start.Format(dateFmt), end.Format(dateFmt), len(instruments)))
			deadline := time.Now().Add(MaxTaskDuration)
			for _, inst := range instruments {
				existing, err := store.PriceHistory(ctx, inst.Symbol, start, end)
				if err != nil {
					tempo.Info(ctx, fmt.Sprintf("financial import: skip %s — price history failed: %v", inst.Symbol, err))
					continue
				}
				existingTimes := daySetFromRecords(existing, func(r marketdata.PriceRecord) time.Time { return r.Time })
				tradingDays := tradingDaysInRange(start, end)
				if tradingDays == 0 {
					tempo.Info(ctx, fmt.Sprintf("financial import: skip %s — no trading days in range", inst.Symbol))
					continue
				}
				if len(existingTimes) >= tradingDays {
					tempo.Info(ctx, fmt.Sprintf("financial import: skip %s — already have full range (%d trading days)", inst.Symbol, tradingDays))
					continue
				}
				// Skip if we already have data through the last trading day (e.g. ran on weekend; have Fri data).
				lastTrading := lastTradingDayInRange(start, end)
				var maxExisting time.Time
				for day := range existingTimes {
					if day.After(maxExisting) {
						maxExisting = day
					}
				}
				if !lastTrading.IsZero() && !maxExisting.IsZero() && (maxExisting.Equal(lastTrading) || maxExisting.After(lastTrading)) {
					tempo.Info(ctx, fmt.Sprintf("financial import: skip %s — already have data through %s (last trading day)", inst.Symbol, lastTrading.Format(dateFmt)))
					continue
				}
				points, fetchErr := FetchWith429Retry(ctx, deadline, "financial import: "+inst.Symbol, func() ([]importer.PricePoint, error) {
					return client.FetchDailyPrices(ctx, inst.Symbol, start, end)
				})
				if fetchErr != nil {
					if strings.Contains(fetchErr.Error(), "429") {
						return fmt.Errorf("429 rate limit for %s: %w", inst.Symbol, fetchErr)
					}
					tempo.Info(ctx, fmt.Sprintf("financial import: skip %s — fetch failed: %v", inst.Symbol, fetchErr))
					continue
				}
				newPoints := pricePointsNewDays(points, existingTimes)
				if len(newPoints) == 0 {
					tempo.Info(ctx, fmt.Sprintf("financial import: skip %s — no new points (fetched %d, all already stored)", inst.Symbol, len(points)))
					continue
				}
				if err := store.IngestPricesBulk(ctx, inst.Symbol, newPoints); err != nil {
					tempo.Error(ctx, fmt.Sprintf("financial import: ingest %s: %v", inst.Symbol, err))
					return fmt.Errorf("ingest %s: %w", inst.Symbol, err)
				}
				first, last := newPoints[0].Time.Format(dateFmt), newPoints[len(newPoints)-1].Time.Format(dateFmt)
				tempo.Info(ctx, fmt.Sprintf("financial import: imported %s — %d points (%s to %s)", inst.Symbol, len(newPoints), first, last))
			}
		}
		return runMaintenance(ctx, store, nil, FinancialImportTaskName)
	}
}

// runMaintenance runs store.Maintenance and logs; used by financial import and financial backfill tasks.
func runMaintenance(ctx context.Context, store *marketdata.Store, l *slog.Logger, taskName string) error {
	if l != nil {
		l.Info("running market data maintenance",
			slog.String("component", "tasks"),
			slog.String("task", taskName),
		)
	}
	tempo.Info(ctx, "running market data maintenance")
	if err := store.Maintenance(ctx); err != nil {
		if l != nil {
			l.Error("maintenance failed",
				slog.String("component", "tasks"),
				slog.String("task", taskName),
				slog.String("error", err.Error()),
			)
		}
		tempo.Error(ctx, fmt.Sprintf("maintenance failed: %v", err))
		return fmt.Errorf("market data maintenance: %w", err)
	}
	if l != nil {
		l.Info("task completed",
			slog.String("component", "tasks"),
			slog.String("task", taskName),
		)
	}
	tempo.Info(ctx, "task completed")
	return nil
}
