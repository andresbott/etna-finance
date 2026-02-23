package tasks

import (
	"context"
	"fmt"
	"time"

	"github.com/andresbott/etna/internal/marketdata"
	"github.com/andresbott/etna/internal/marketdata/importer"
	"github.com/go-bumbu/tempo"
)

const FXImportTaskName = "fx-import"

// FXBackfillDays is the number of calendar days of history to backfill per pair when an FX client is configured.
const FXBackfillDays = 7

// MaxFXTaskDuration is the maximum duration for the FX import task.
const MaxFXTaskDuration = 1 * time.Hour

// FXImportTaskDef is the task definition for the FX import task.
var FXImportTaskDef = TaskDef{
	ID:          FXImportTaskName,
	Name:        "Currency exchange import",
	Description: "Import recent FX rates for configured pairs, then run market data maintenance.",
}

// NewFXImportTaskFn returns a task function that imports recent FX rates for each configured pair
// (main + secondaries from config), then runs market data maintenance.
// When client is nil, it still registers all configured pairs so FX series exist (e.g. for manual entry), then runs maintenance.
func NewFXImportTaskFn(store *marketdata.Store, mainCurrency string, currencies []string, client importer.FXClient) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if store == nil {
			return fmt.Errorf("market data store is required")
		}
		tempo.Info(ctx, "starting currency exchange import")

		if mainCurrency != "" && len(currencies) > 0 {
			// Ensure all configured pairs are registered (so series exist for manual entry or future import).
			registered := 0
			for _, secondary := range currencies {
				if secondary == mainCurrency {
					continue
				}
				if err := store.RegisterPair(mainCurrency, secondary); err != nil {
					tempo.Info(ctx, fmt.Sprintf("fx import: register %s/%s: %v", mainCurrency, secondary, err))
					continue
				}
				registered++
			}
			if registered > 0 {
				tempo.Info(ctx, fmt.Sprintf("fx import: registered %d pair(s) for %s", registered, mainCurrency))
			}
			if client == nil && registered > 0 {
				tempo.Info(ctx, "fx import: no FX client configured — add rates manually or configure an FX importer for automatic fetch")
			}
		}

		if client != nil && mainCurrency != "" && len(currencies) > 0 {
			now := time.Now().UTC()
			end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Add(24 * time.Hour)
			start := end.Add(-FXBackfillDays * 24 * time.Hour)
			dateFmt := "2006-01-02"

			pairs := 0
			for _, c := range currencies {
				if c == mainCurrency {
					continue
				}
				pairs++
			}
			tempo.Info(ctx, fmt.Sprintf("fx import: backfill range %s to %s, %d pairs", start.Format(dateFmt), end.Format(dateFmt), pairs))
			deadline := time.Now().Add(MaxFXTaskDuration)

			for _, secondary := range currencies {
				if secondary == mainCurrency {
					continue
				}
				existing, err := store.RateHistory(ctx, mainCurrency, secondary, start, end)
				if err != nil {
					tempo.Info(ctx, fmt.Sprintf("fx import: skip %s/%s — rate history failed: %v", mainCurrency, secondary, err))
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
					points, fetchErr = client.FetchDailyRates(ctx, mainCurrency, secondary, start, end)
					if fetchErr == nil {
						break
					}
					if !importer.IsRateLimit429(fetchErr) {
						break
					}
					retryAfter := importer.RetryAfterFrom429Err(fetchErr, importer.Default429RetryAfter)
					if time.Now().Add(retryAfter).After(deadline) {
						tempo.Error(ctx, fmt.Sprintf("fx import: 429 for %s/%s — retry would exceed deadline", mainCurrency, secondary))
						return fmt.Errorf("429 rate limit for %s/%s: %w", mainCurrency, secondary, fetchErr)
					}
					tempo.Info(ctx, fmt.Sprintf("fx import: 429 for %s/%s — waiting %v before retry", mainCurrency, secondary, retryAfter))
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(retryAfter):
					}
				}
				if fetchErr != nil {
					tempo.Info(ctx, fmt.Sprintf("fx import: skip %s/%s — fetch failed: %v", mainCurrency, secondary, fetchErr))
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
					tempo.Info(ctx, fmt.Sprintf("fx import: skip %s/%s — no new points (fetched %d)", mainCurrency, secondary, len(points)))
					continue
				}
				if time.Now().After(deadline) {
					tempo.Info(ctx, "fx import: stopping before full run (deadline)")
					break
				}
				if err := store.IngestRatesBulk(ctx, mainCurrency, secondary, newPoints); err != nil {
					tempo.Error(ctx, fmt.Sprintf("fx import: ingest %s/%s: %v", mainCurrency, secondary, err))
					return fmt.Errorf("ingest %s/%s: %w", mainCurrency, secondary, err)
				}
				first, last := newPoints[0].Time.Format(dateFmt), newPoints[len(newPoints)-1].Time.Format(dateFmt)
				tempo.Info(ctx, fmt.Sprintf("fx import: imported %s/%s — %d points (%s to %s)", mainCurrency, secondary, len(newPoints), first, last))
			}
		}

		return runMaintenance(ctx, store, nil, FXImportTaskName)
	}
}
