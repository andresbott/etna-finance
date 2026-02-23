package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/andresbott/etna/internal/marketdata"
	"github.com/andresbott/etna/internal/marketdata/importer"
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
		taskLogInfo(ctx, l, FXBackfillTaskName, "starting currency exchange backfill (1 year)")

		if client == nil || mainCurrency == "" || len(currencies) == 0 {
			taskLogInfo(ctx, l, FXBackfillTaskName, "fx backfill: no FX client or no pairs configured, running maintenance only")
			return runMaintenance(ctx, store, l, FXBackfillTaskName)
		}

		now := time.Now().UTC()
		end := dateRangeEnd(now)
		start := dateRangeStart(end, FXBackfillYearDays)
		deadline := time.Now().Add(MaxFXBackfillTaskDuration)
		pairs := 0
		for _, c := range currencies {
			if c != mainCurrency {
				pairs++
			}
		}
		taskLogInfo(ctx, l, FXBackfillTaskName, fmt.Sprintf("fx backfill: range %s to %s, %d pairs, batches of %d days", start.Format(dateFmt), end.Format(dateFmt), pairs, FXBackfillBatchSize),
			slog.String("start", start.Format(dateFmt)), slog.String("end", end.Format(dateFmt)), slog.Int("pairs", pairs), slog.Int("batchSizeDays", FXBackfillBatchSize))

		for _, secondary := range currencies {
			if secondary == mainCurrency {
				continue
			}
			pairLabel := mainCurrency + "/" + secondary
			for batchStart := start; batchStart.Before(end); batchStart = batchStart.Add(FXBackfillBatchSize * 24 * time.Hour) {
				batchEnd := batchStart.Add(FXBackfillBatchSize * 24 * time.Hour)
				if batchEnd.After(end) {
					batchEnd = end
				}

				if time.Now().After(deadline) {
					taskLogInfo(ctx, l, FXBackfillTaskName, "fx backfill: deadline reached, running maintenance")
					return runMaintenance(ctx, store, l, FXBackfillTaskName)
				}

				existing, err := store.RateHistory(ctx, mainCurrency, secondary, batchStart, batchEnd)
				if err != nil {
					taskLogWarn(ctx, l, FXBackfillTaskName, fmt.Sprintf("fx backfill: skip %s/%s batch — %v", mainCurrency, secondary, err),
						slog.String("pair", pairLabel), slog.String("error", err.Error()))
					continue
				}
				existingTimes := daySetFromRecords(existing, func(r marketdata.RateRecord) time.Time { return r.Time })

				points, fetchErr := FetchWith429Retry(ctx, deadline, "fx backfill: "+pairLabel, func() ([]importer.RatePoint, error) {
					return client.FetchDailyRates(ctx, mainCurrency, secondary, batchStart, batchEnd)
				})
				if fetchErr != nil {
					if strings.Contains(fetchErr.Error(), "429") {
						return fmt.Errorf("429 rate limit for %s: %w", pairLabel, fetchErr)
					}
					taskLogWarn(ctx, l, FXBackfillTaskName, fmt.Sprintf("fx backfill: skip %s/%s batch — fetch failed: %v", mainCurrency, secondary, fetchErr),
						slog.String("pair", pairLabel), slog.String("error", fetchErr.Error()))
					continue
				}

				newPoints := ratePointsNewDays(points, existingTimes)
				if len(newPoints) == 0 {
					taskLogInfo(ctx, l, FXBackfillTaskName, fmt.Sprintf("fx backfill: skip %s/%s batch — no new points (fetched %d)", mainCurrency, secondary, len(points)),
						slog.String("pair", pairLabel))
					continue
				}

				if err := store.IngestRatesBulk(ctx, mainCurrency, secondary, newPoints); err != nil {
					taskLogError(ctx, l, FXBackfillTaskName, fmt.Sprintf("fx backfill: ingest %s/%s: %v", mainCurrency, secondary, err), slog.String("pair", pairLabel), slog.String("error", err.Error()))
					return fmt.Errorf("ingest %s/%s: %w", mainCurrency, secondary, err)
				}
				first, last := newPoints[0].Time.Format(dateFmt), newPoints[len(newPoints)-1].Time.Format(dateFmt)
				taskLogInfo(ctx, l, FXBackfillTaskName, fmt.Sprintf("fx backfill: %s/%s — %d points (%s to %s)", mainCurrency, secondary, len(newPoints), first, last),
					slog.String("pair", pairLabel), slog.Int("points", len(newPoints)), slog.String("dateRange", first+" to "+last))
			}
		}

		return runMaintenance(ctx, store, l, FXBackfillTaskName)
	}
}
