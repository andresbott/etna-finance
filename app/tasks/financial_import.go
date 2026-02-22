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
func NewFinancialImportTaskFn(store *marketdata.Store, l *slog.Logger, client importer.Client) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if store == nil {
			return fmt.Errorf("market data store is required")
		}
		if l != nil {
			l.Info("starting financial import",
				slog.String("component", "tasks"),
				slog.String("task", FinancialImportTaskName),
			)
		}
		tempo.Info(ctx, "starting financial import")

		if client != nil {
			instruments, err := store.ListInstruments(ctx)
			if err != nil {
				if l != nil {
					l.Error("financial import: list instruments failed",
						slog.String("component", "tasks"),
						slog.String("task", FinancialImportTaskName),
						slog.String("error", err.Error()),
					)
				}
				tempo.Error(ctx, fmt.Sprintf("financial import: list instruments failed: %v", err))
				return fmt.Errorf("list instruments: %w", err)
			}
			now := time.Now().UTC()
			end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Add(24 * time.Hour)
			start := end.Add(-BackfillDays * 24 * time.Hour)
			dateFmt := "2006-01-02"
			if l != nil {
				l.Info("financial import: backfill range",
					slog.String("component", "tasks"),
					slog.String("task", FinancialImportTaskName),
					slog.String("start", start.Format(dateFmt)),
					slog.String("end", end.Format(dateFmt)),
					slog.Int("instruments", len(instruments)),
				)
			}
			tempo.Info(ctx, fmt.Sprintf("financial import: backfill range %s to %s, %d instruments", start.Format(dateFmt), end.Format(dateFmt), len(instruments)))
			deadline := time.Now().Add(MaxTaskDuration)
			for _, inst := range instruments {
				existing, err := store.PriceHistory(ctx, inst.Symbol, start, end)
				if err != nil {
					if l != nil {
						l.Warn("financial import: skip symbol (could not read existing prices)",
							slog.String("component", "tasks"),
							slog.String("task", FinancialImportTaskName),
							slog.String("symbol", inst.Symbol),
							slog.String("reason", "price history failed"),
							slog.String("error", err.Error()),
						)
					}
					tempo.Info(ctx, fmt.Sprintf("financial import: skip %s — price history failed: %v", inst.Symbol, err))
					continue
				}
				existingTimes := make(map[time.Time]struct{}, len(existing))
				for _, r := range existing {
					day := time.Date(r.Time.Year(), r.Time.Month(), r.Time.Day(), 0, 0, 0, 0, time.UTC)
					existingTimes[day] = struct{}{}
				}
				tradingDays := tradingDaysInRange(start, end)
				if tradingDays == 0 {
					// Range is all weekends/holidays; no point requesting (market closed).
					if l != nil {
						l.Info("financial import: skip symbol (no trading days in range)",
							slog.String("component", "tasks"),
							slog.String("task", FinancialImportTaskName),
							slog.String("symbol", inst.Symbol),
							slog.String("range", start.Format(dateFmt)+" to "+end.Format(dateFmt)),
						)
					}
					tempo.Info(ctx, fmt.Sprintf("financial import: skip %s — no trading days in range", inst.Symbol))
					continue
				}
				if len(existingTimes) >= tradingDays {
					if l != nil {
						l.Info("financial import: skip symbol (already have data for all trading days)",
							slog.String("component", "tasks"),
							slog.String("task", FinancialImportTaskName),
							slog.String("symbol", inst.Symbol),
							slog.String("range", start.Format(dateFmt)+" to "+end.Format(dateFmt)),
							slog.Int("existingDays", len(existingTimes)),
							slog.Int("tradingDaysInRange", tradingDays),
						)
					}
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
					if l != nil {
						l.Info("financial import: skip symbol (already have data through last trading day)",
							slog.String("component", "tasks"),
							slog.String("task", FinancialImportTaskName),
							slog.String("symbol", inst.Symbol),
							slog.String("range", start.Format(dateFmt)+" to "+end.Format(dateFmt)),
							slog.String("latestExisting", maxExisting.Format(dateFmt)),
							slog.String("lastTradingDay", lastTrading.Format(dateFmt)),
						)
					}
					tempo.Info(ctx, fmt.Sprintf("financial import: skip %s — already have data through %s (last trading day)", inst.Symbol, lastTrading.Format(dateFmt)))
					continue
				}
				var points []importer.PricePoint
				var fetchErr error
				for {
					points, fetchErr = client.FetchDailyPrices(ctx, inst.Symbol, start, end)
					if fetchErr == nil {
						break
					}
					if !importer.IsRateLimit429(fetchErr) {
						break
					}
					retryAfter := importer.RetryAfterFrom429Err(fetchErr, importer.Default429RetryAfter)
					if time.Now().Add(retryAfter).After(deadline) {
						if l != nil {
							l.Error("financial import: 429 rate limit, retry would exceed task deadline",
								slog.String("component", "tasks"),
								slog.String("task", FinancialImportTaskName),
								slog.String("symbol", inst.Symbol),
								slog.Duration("retryAfter", retryAfter),
								slog.Duration("maxTaskDuration", MaxTaskDuration),
							)
						}
						tempo.Error(ctx, fmt.Sprintf("financial import: 429 for %s — retry would exceed 1h deadline (retry after %v)", inst.Symbol, retryAfter))
						return fmt.Errorf("429 rate limit for %s: retry would exceed task deadline (%v): %w", inst.Symbol, retryAfter, fetchErr)
					}
					if l != nil {
						l.Info("financial import: 429 rate limit, waiting before retry",
							slog.String("component", "tasks"),
							slog.String("task", FinancialImportTaskName),
							slog.String("symbol", inst.Symbol),
							slog.Duration("wait", retryAfter),
						)
					}
					tempo.Info(ctx, fmt.Sprintf("financial import: 429 for %s — waiting %v before retry", inst.Symbol, retryAfter))
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(retryAfter):
						// continue loop and retry
					}
				}
				if fetchErr != nil {
					if l != nil {
						l.Warn("financial import: skip symbol (fetch failed)",
							slog.String("component", "tasks"),
							slog.String("task", FinancialImportTaskName),
							slog.String("symbol", inst.Symbol),
							slog.String("range", start.Format(dateFmt)+" to "+end.Format(dateFmt)),
							slog.String("error", fetchErr.Error()),
						)
					}
					tempo.Info(ctx, fmt.Sprintf("financial import: skip %s — fetch failed: %v", inst.Symbol, fetchErr))
					continue
				}
				var newPoints []marketdata.PricePoint
				for _, p := range points {
					day := time.Date(p.Time.Year(), p.Time.Month(), p.Time.Day(), 0, 0, 0, 0, time.UTC)
					if _, exists := existingTimes[day]; !exists {
						newPoints = append(newPoints, marketdata.PricePoint{Time: p.Time, Price: p.Price})
						existingTimes[day] = struct{}{}
					}
				}
				if len(newPoints) == 0 {
					if l != nil {
						l.Info("financial import: skip symbol (no new points after fetch)",
							slog.String("component", "tasks"),
							slog.String("task", FinancialImportTaskName),
							slog.String("symbol", inst.Symbol),
							slog.Int("fetched", len(points)),
							slog.Int("existingDays", len(existingTimes)),
						)
					}
					tempo.Info(ctx, fmt.Sprintf("financial import: skip %s — no new points (fetched %d, all already stored)", inst.Symbol, len(points)))
					continue
				}
				if err := store.IngestPricesBulk(ctx, inst.Symbol, newPoints); err != nil {
					if l != nil {
						l.Error("financial import: ingest failed for symbol",
							slog.String("component", "tasks"),
							slog.String("task", FinancialImportTaskName),
							slog.String("symbol", inst.Symbol),
							slog.String("error", err.Error()),
						)
					}
					tempo.Error(ctx, fmt.Sprintf("financial import: ingest %s: %v", inst.Symbol, err))
					return fmt.Errorf("ingest %s: %w", inst.Symbol, err)
				}
				first, last := newPoints[0].Time.Format(dateFmt), newPoints[len(newPoints)-1].Time.Format(dateFmt)
				if l != nil {
					l.Info("financial import: imported symbol",
						slog.String("component", "tasks"),
						slog.String("task", FinancialImportTaskName),
						slog.String("symbol", inst.Symbol),
						slog.Int("points", len(newPoints)),
						slog.String("dateRange", first+" to "+last),
					)
				}
				tempo.Info(ctx, fmt.Sprintf("financial import: imported %s — %d points (%s to %s)", inst.Symbol, len(newPoints), first, last))
			}
		}
		return runMaintenance(ctx, store, l, FinancialImportTaskName)
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
