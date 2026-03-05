package tasks

import (
	"context"
	"fmt"
	"strings"
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

		if client == nil {
			return fmt.Errorf("no FX importer configured — set API key via ETNA_MARKETDATAIMPORTERS_MASSIVE_APIKEYS_0 or config file")
		}

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
		}

		if mainCurrency != "" && len(currencies) > 0 {
			if err := backfillFXRates(ctx, store, client, mainCurrency, currencies); err != nil {
				return err
			}
		}

		return runMaintenance(ctx, store, nil, FXImportTaskName)
	}
}

// backfillFXRates fetches and stores recent FX rate history for all configured currency pairs.
func backfillFXRates(ctx context.Context, store *marketdata.Store, client importer.FXClient, mainCurrency string, currencies []string) error {
	now := time.Now().UTC()
	end := dateRangeEnd(now)
	start := dateRangeStart(end, FXBackfillDays)
	pairs := 0
	for _, c := range currencies {
		if c != mainCurrency {
			pairs++
		}
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
		existingTimes := daySetFromRecords(existing, func(r marketdata.RateRecord) time.Time { return r.Time })

		pairLabel := mainCurrency + "/" + secondary
		points, fetchErr := FetchWith429Retry(ctx, deadline, "fx import: "+pairLabel, func() ([]importer.RatePoint, error) {
			return client.FetchDailyRates(ctx, mainCurrency, secondary, start, end)
		})
		if fetchErr != nil {
			if strings.Contains(fetchErr.Error(), "401") {
				return fmt.Errorf("invalid API key for %s: %w", pairLabel, fetchErr)
			}
			if strings.Contains(fetchErr.Error(), "429") {
				return fmt.Errorf("429 rate limit for %s: %w", pairLabel, fetchErr)
			}
			tempo.Info(ctx, fmt.Sprintf("fx import: skip %s/%s — fetch failed: %v", mainCurrency, secondary, fetchErr))
			continue
		}

		newPoints := ratePointsNewDays(points, existingTimes)
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
	return nil
}
