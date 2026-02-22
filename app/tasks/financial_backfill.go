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

const FinancialBackfillTaskName = "financial-backfill"

// BackfillYearDays is the number of calendar days of history to backfill (1 year).
const BackfillYearDays = 365

// BackfillBatchSize is the max number of results per API request; the 1-year range is split into chunks of this many days.
const BackfillBatchSize = 10000

// MaxBackfillTaskDuration is the maximum duration for the financial backfill task (2 hours).
const MaxBackfillTaskDuration = 2 * time.Hour

// FinancialBackfillTaskDef is the task definition for the 1-year backfill task.
var FinancialBackfillTaskDef = TaskDef{
	ID:          FinancialBackfillTaskName,
	Name:        "Financial backfill (1 year)",
	Description: "Backfill up to 1 year of market data per instrument in batches of 10K results. Timeout 2h.",
}

// NewFinancialBackfillTaskFn returns a task function that backfills up to 1 year of data per instrument
// in batches of BackfillBatchSize, then runs market data maintenance. Uses a 2h deadline.
func NewFinancialBackfillTaskFn(store *marketdata.Store, l *slog.Logger, client importer.Client) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if store == nil {
			return fmt.Errorf("market data store is required")
		}
		if l != nil {
			l.Info("starting financial backfill (1 year)",
				slog.String("component", "tasks"),
				slog.String("task", FinancialBackfillTaskName),
			)
		}
		tempo.Info(ctx, "starting financial backfill (1 year)")

		if client == nil {
			if l != nil {
				l.Info("financial backfill: no importer client configured, running maintenance only",
					slog.String("component", "tasks"),
					slog.String("task", FinancialBackfillTaskName),
				)
			}
			tempo.Info(ctx, "financial backfill: no importer client, maintenance only")
			return runMaintenance(ctx, store, l, FinancialBackfillTaskName)
		}

		instruments, err := store.ListInstruments(ctx)
		if err != nil {
			if l != nil {
				l.Error("financial backfill: list instruments failed",
					slog.String("component", "tasks"),
					slog.String("task", FinancialBackfillTaskName),
					slog.String("error", err.Error()),
				)
			}
			tempo.Error(ctx, fmt.Sprintf("financial backfill: list instruments failed: %v", err))
			return fmt.Errorf("list instruments: %w", err)
		}

		now := time.Now().UTC()
		end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Add(24 * time.Hour)
		start := end.Add(-BackfillYearDays * 24 * time.Hour)
		dateFmt := "2006-01-02"
		deadline := time.Now().Add(MaxBackfillTaskDuration)

		if l != nil {
			l.Info("financial backfill: range and batch size",
				slog.String("component", "tasks"),
				slog.String("task", FinancialBackfillTaskName),
				slog.String("start", start.Format(dateFmt)),
				slog.String("end", end.Format(dateFmt)),
				slog.Int("instruments", len(instruments)),
				slog.Int("batchSizeDays", BackfillBatchSize),
			)
		}
		tempo.Info(ctx, fmt.Sprintf("financial backfill: range %s to %s, %d instruments, batches of %d days", start.Format(dateFmt), end.Format(dateFmt), len(instruments), BackfillBatchSize))

		for _, inst := range instruments {
			// Process instrument in batches of BackfillBatchSize days (max 10K results per API call).
			for batchStart := start; batchStart.Before(end); batchStart = batchStart.Add(BackfillBatchSize * 24 * time.Hour) {
				batchEnd := batchStart.Add(BackfillBatchSize * 24 * time.Hour)
				if batchEnd.After(end) {
					batchEnd = end
				}

				existing, err := store.PriceHistory(ctx, inst.Symbol, batchStart, batchEnd)
				if err != nil {
					if l != nil {
						l.Warn("financial backfill: skip batch (price history failed)",
							slog.String("component", "tasks"),
							slog.String("task", FinancialBackfillTaskName),
							slog.String("symbol", inst.Symbol),
							slog.String("batch", batchStart.Format(dateFmt)+" to "+batchEnd.Format(dateFmt)),
							slog.String("error", err.Error()),
						)
					}
					tempo.Info(ctx, fmt.Sprintf("financial backfill: skip %s batch %s—%s: %v", inst.Symbol, batchStart.Format(dateFmt), batchEnd.Format(dateFmt), err))
					continue
				}

				existingTimes := make(map[time.Time]struct{}, len(existing))
				for _, r := range existing {
					day := time.Date(r.Time.Year(), r.Time.Month(), r.Time.Day(), 0, 0, 0, 0, time.UTC)
					existingTimes[day] = struct{}{}
				}

				tradingDays := tradingDaysInRange(batchStart, batchEnd)
				if tradingDays == 0 {
					if l != nil {
						l.Info("financial backfill: skip batch (no trading days in range)",
							slog.String("component", "tasks"),
							slog.String("task", FinancialBackfillTaskName),
							slog.String("symbol", inst.Symbol),
							slog.String("batch", batchStart.Format(dateFmt)+" to "+batchEnd.Format(dateFmt)),
						)
					}
					tempo.Info(ctx, fmt.Sprintf("financial backfill: skip %s batch — no trading days", inst.Symbol))
					continue
				}
				if len(existingTimes) >= tradingDays {
					if l != nil {
						l.Info("financial backfill: skip batch (already have data for all trading days)",
							slog.String("component", "tasks"),
							slog.String("task", FinancialBackfillTaskName),
							slog.String("symbol", inst.Symbol),
							slog.String("batch", batchStart.Format(dateFmt)+" to "+batchEnd.Format(dateFmt)),
							slog.Int("existingDays", len(existingTimes)),
							slog.Int("tradingDaysInRange", tradingDays),
						)
					}
					tempo.Info(ctx, fmt.Sprintf("financial backfill: skip %s batch — already full (%d trading days)", inst.Symbol, tradingDays))
					continue
				}
				// Do not skip just because we have data through the last trading day: for a 1-year
				// range we may have recent data but still miss older data in the range.

				var points []importer.PricePoint
				var fetchErr error
				for {
					points, fetchErr = client.FetchDailyPrices(ctx, inst.Symbol, batchStart, batchEnd)
					if fetchErr == nil {
						break
					}
					if !importer.IsRateLimit429(fetchErr) {
						break
					}
					retryAfter := importer.RetryAfterFrom429Err(fetchErr, importer.Default429RetryAfter)
					if time.Now().Add(retryAfter).After(deadline) {
						if l != nil {
							l.Error("financial backfill: 429 retry would exceed 2h deadline",
								slog.String("component", "tasks"),
								slog.String("task", FinancialBackfillTaskName),
								slog.String("symbol", inst.Symbol),
								slog.Duration("retryAfter", retryAfter),
							)
						}
						tempo.Error(ctx, fmt.Sprintf("financial backfill: 429 for %s — retry would exceed 2h deadline", inst.Symbol))
						return fmt.Errorf("429 rate limit for %s: retry would exceed task deadline (%v): %w", inst.Symbol, retryAfter, fetchErr)
					}
					if l != nil {
						l.Info("financial backfill: 429, waiting before retry",
							slog.String("component", "tasks"),
							slog.String("task", FinancialBackfillTaskName),
							slog.String("symbol", inst.Symbol),
							slog.Duration("wait", retryAfter),
						)
					}
					tempo.Info(ctx, fmt.Sprintf("financial backfill: 429 for %s — waiting %v", inst.Symbol, retryAfter))
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(retryAfter):
					}
				}

				if fetchErr != nil {
					if l != nil {
						l.Warn("financial backfill: skip batch (fetch failed)",
							slog.String("component", "tasks"),
							slog.String("task", FinancialBackfillTaskName),
							slog.String("symbol", inst.Symbol),
							slog.String("batch", batchStart.Format(dateFmt)+" to "+batchEnd.Format(dateFmt)),
							slog.String("error", fetchErr.Error()),
						)
					}
					tempo.Info(ctx, fmt.Sprintf("financial backfill: skip %s batch — fetch failed: %v", inst.Symbol, fetchErr))
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
						l.Info("financial backfill: skip batch (no new points after fetch)",
							slog.String("component", "tasks"),
							slog.String("task", FinancialBackfillTaskName),
							slog.String("symbol", inst.Symbol),
							slog.Int("fetched", len(points)),
							slog.Int("existingDays", len(existingTimes)),
						)
					}
					tempo.Info(ctx, fmt.Sprintf("financial backfill: skip %s batch — no new points (fetched %d)", inst.Symbol, len(points)))
					continue
				}

				if err := store.IngestPricesBulk(ctx, inst.Symbol, newPoints); err != nil {
					if l != nil {
						l.Error("financial backfill: ingest failed",
							slog.String("component", "tasks"),
							slog.String("task", FinancialBackfillTaskName),
							slog.String("symbol", inst.Symbol),
							slog.String("error", err.Error()),
						)
					}
					tempo.Error(ctx, fmt.Sprintf("financial backfill: ingest %s: %v", inst.Symbol, err))
					return fmt.Errorf("ingest %s: %w", inst.Symbol, err)
				}

				first, last := newPoints[0].Time.Format(dateFmt), newPoints[len(newPoints)-1].Time.Format(dateFmt)
				if l != nil {
					l.Info("financial backfill: imported batch",
						slog.String("component", "tasks"),
						slog.String("task", FinancialBackfillTaskName),
						slog.String("symbol", inst.Symbol),
						slog.Int("points", len(newPoints)),
						slog.String("dateRange", first+" to "+last),
					)
				}
				tempo.Info(ctx, fmt.Sprintf("financial backfill: %s — %d points (%s to %s)", inst.Symbol, len(newPoints), first, last))
			}
		}

		return runMaintenance(ctx, store, l, FinancialBackfillTaskName)
	}
}
