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

const FXBackfillTaskName = "fx-backfill"

// FXBackfillYearDays is the number of calendar days of history to backfill (1 year).
const FXBackfillYearDays = 365

// FXBackfillBatchSize is the max number of days per batch for the 1-year range.
// Polygon aggregates support large ranges; using the full year per batch minimizes API calls.
const FXBackfillBatchSize = 365

// MaxFXBackfillTaskDuration is the maximum duration for the FX backfill task (2 hours).
const MaxFXBackfillTaskDuration = 2 * time.Hour

// FXBackfillTaskDef is the task definition for the FX 1-year backfill task.
var FXBackfillTaskDef = TaskDef{
	ID:          FXBackfillTaskName,
	Name:        "Currency exchange backfill (1 year)",
	Description: "Backfill up to 1 year of FX rates per configured pair in batches. Timeout 2h.",
}

// NewFXBackfillTaskFn returns a task function that backfills up to 1 year of FX data per configured pair
// in batches, then runs market data maintenance.
// When client is nil, only maintenance is run.
func NewFXBackfillTaskFn(store *marketdata.Store, l *slog.Logger, mainCurrency string, currencies []string, client importer.FXClient) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if store == nil {
			return fmt.Errorf("market data store is required")
		}
		if l != nil {
			l.Info("starting currency exchange backfill (1 year)",
				slog.String("component", "tasks"),
				slog.String("task", FXBackfillTaskName),
			)
		}
		tempo.Info(ctx, "starting currency exchange backfill (1 year)")

		if client == nil || mainCurrency == "" || len(currencies) == 0 {
			if l != nil {
				l.Info("fx backfill: no FX client or no pairs configured, running maintenance only",
					slog.String("component", "tasks"),
					slog.String("task", FXBackfillTaskName),
				)
			}
			tempo.Info(ctx, "fx backfill: no client or pairs, maintenance only")
			return runMaintenance(ctx, store, l, FXBackfillTaskName)
		}

		now := time.Now().UTC()
		end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Add(24 * time.Hour)
		start := end.Add(-FXBackfillYearDays * 24 * time.Hour)
		dateFmt := "2006-01-02"
		deadline := time.Now().Add(MaxFXBackfillTaskDuration)

		pairs := 0
		for _, c := range currencies {
			if c != mainCurrency {
				pairs++
			}
		}
		if l != nil {
			l.Info("fx backfill: range and pairs",
				slog.String("component", "tasks"),
				slog.String("task", FXBackfillTaskName),
				slog.String("start", start.Format(dateFmt)),
				slog.String("end", end.Format(dateFmt)),
				slog.Int("pairs", pairs),
				slog.Int("batchSizeDays", FXBackfillBatchSize),
			)
		}
		tempo.Info(ctx, fmt.Sprintf("fx backfill: range %s to %s, %d pairs, batches of %d days", start.Format(dateFmt), end.Format(dateFmt), pairs, FXBackfillBatchSize))

		for _, secondary := range currencies {
			if secondary == mainCurrency {
				continue
			}
			for batchStart := start; batchStart.Before(end); batchStart = batchStart.Add(FXBackfillBatchSize * 24 * time.Hour) {
				batchEnd := batchStart.Add(FXBackfillBatchSize * 24 * time.Hour)
				if batchEnd.After(end) {
					batchEnd = end
				}

				if time.Now().After(deadline) {
					if l != nil {
						l.Info("fx backfill: stopping (deadline)",
							slog.String("component", "tasks"),
							slog.String("task", FXBackfillTaskName),
						)
					}
					tempo.Info(ctx, "fx backfill: deadline reached, running maintenance")
					return runMaintenance(ctx, store, l, FXBackfillTaskName)
				}

				existing, err := store.RateHistory(ctx, mainCurrency, secondary, batchStart, batchEnd)
				if err != nil {
					if l != nil {
						l.Warn("fx backfill: skip batch (rate history failed)",
							slog.String("component", "tasks"),
							slog.String("task", FXBackfillTaskName),
							slog.String("pair", mainCurrency+"/"+secondary),
							slog.String("error", err.Error()),
						)
					}
					tempo.Info(ctx, fmt.Sprintf("fx backfill: skip %s/%s batch — %v", mainCurrency, secondary, err))
					continue
				}
				existingTimes := make(map[time.Time]struct{}, len(existing))
				for _, r := range existing {
					day := time.Date(r.Time.Year(), r.Time.Month(), r.Time.Day(), 0, 0, 0, 0, time.UTC)
					existingTimes[day] = struct{}{}
				}

				var points []importer.RatePoint
				var fetchErr error
				for {
					points, fetchErr = client.FetchDailyRates(ctx, mainCurrency, secondary, batchStart, batchEnd)
					if fetchErr == nil {
						break
					}
					if !importer.IsRateLimit429(fetchErr) {
						break
					}
					retryAfter := importer.RetryAfterFrom429Err(fetchErr, importer.Default429RetryAfter)
					if time.Now().Add(retryAfter).After(deadline) {
						tempo.Error(ctx, fmt.Sprintf("fx backfill: 429 for %s/%s — retry would exceed deadline", mainCurrency, secondary))
						return fmt.Errorf("429 rate limit for %s/%s: %w", mainCurrency, secondary, fetchErr)
					}
					tempo.Info(ctx, fmt.Sprintf("fx backfill: 429 for %s/%s — waiting %v", mainCurrency, secondary, retryAfter))
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(retryAfter):
					}
				}
				if fetchErr != nil {
					if l != nil {
						l.Warn("fx backfill: skip batch (fetch failed)",
							slog.String("component", "tasks"),
							slog.String("task", FXBackfillTaskName),
							slog.String("pair", mainCurrency+"/"+secondary),
							slog.String("error", fetchErr.Error()),
						)
					}
					tempo.Info(ctx, fmt.Sprintf("fx backfill: skip %s/%s batch — fetch failed: %v", mainCurrency, secondary, fetchErr))
					continue
				}

				var newPoints []marketdata.RatePoint
				for _, p := range points {
					day := time.Date(p.Time.Year(), p.Time.Month(), p.Time.Day(), 0, 0, 0, 0, time.UTC)
					if _, exists := existingTimes[day]; !exists {
						newPoints = append(newPoints, marketdata.RatePoint{Time: p.Time, Rate: p.Rate})
						existingTimes[day] = struct{}{}
					}
				}
				if len(newPoints) == 0 {
					tempo.Info(ctx, fmt.Sprintf("fx backfill: skip %s/%s batch — no new points (fetched %d)", mainCurrency, secondary, len(points)))
					continue
				}

				if err := store.IngestRatesBulk(ctx, mainCurrency, secondary, newPoints); err != nil {
					if l != nil {
						l.Error("fx backfill: ingest failed",
							slog.String("component", "tasks"),
							slog.String("task", FXBackfillTaskName),
							slog.String("pair", mainCurrency+"/"+secondary),
							slog.String("error", err.Error()),
						)
					}
					tempo.Error(ctx, fmt.Sprintf("fx backfill: ingest %s/%s: %v", mainCurrency, secondary, err))
					return fmt.Errorf("ingest %s/%s: %w", mainCurrency, secondary, err)
				}
				first, last := newPoints[0].Time.Format(dateFmt), newPoints[len(newPoints)-1].Time.Format(dateFmt)
				if l != nil {
					l.Info("fx backfill: imported batch",
						slog.String("component", "tasks"),
						slog.String("task", FXBackfillTaskName),
						slog.String("pair", mainCurrency+"/"+secondary),
						slog.Int("points", len(newPoints)),
						slog.String("dateRange", first+" to "+last),
					)
				}
				tempo.Info(ctx, fmt.Sprintf("fx backfill: %s/%s — %d points (%s to %s)", mainCurrency, secondary, len(newPoints), first, last))
			}
		}

		return runMaintenance(ctx, store, l, FXBackfillTaskName)
	}
}
