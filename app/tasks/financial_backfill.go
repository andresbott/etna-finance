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
		taskLogInfo(ctx, l, FinancialBackfillTaskName, "starting financial backfill (1 year)")

		if client == nil {
			return fmt.Errorf("no market data importer configured — set API key via ETNA_MARKETDATAIMPORTERS_MASSIVE_APIKEYS_0 or config file")
		}

		instruments, err := store.ListInstruments(ctx)
		if err != nil {
			taskLogError(ctx, l, FinancialBackfillTaskName, fmt.Sprintf("financial backfill: list instruments failed: %v", err), slog.String("error", err.Error()))
			return fmt.Errorf("list instruments: %w", err)
		}

		now := time.Now().UTC()
		end := dateRangeEnd(now)
		start := dateRangeStart(end, BackfillYearDays)
		deadline := time.Now().Add(MaxBackfillTaskDuration)

		taskLogInfo(ctx, l, FinancialBackfillTaskName, fmt.Sprintf("financial backfill: range %s to %s, %d instruments, batches of %d days", start.Format(dateFmt), end.Format(dateFmt), len(instruments), BackfillBatchSize),
			slog.String("start", start.Format(dateFmt)), slog.String("end", end.Format(dateFmt)), slog.Int("instruments", len(instruments)), slog.Int("batchSizeDays", BackfillBatchSize))

		for _, inst := range instruments {
			for batchStart := start; batchStart.Before(end); batchStart = batchStart.Add(BackfillBatchSize * 24 * time.Hour) {
				batchEnd := batchStart.Add(BackfillBatchSize * 24 * time.Hour)
				if batchEnd.After(end) {
					batchEnd = end
				}

				existing, err := store.PriceHistory(ctx, inst.Symbol, batchStart, batchEnd)
				if err != nil {
					taskLogWarn(ctx, l, FinancialBackfillTaskName, fmt.Sprintf("financial backfill: skip %s batch %s—%s: %v", inst.Symbol, batchStart.Format(dateFmt), batchEnd.Format(dateFmt), err),
						slog.String("symbol", inst.Symbol), slog.String("batch", batchStart.Format(dateFmt)+" to "+batchEnd.Format(dateFmt)), slog.String("error", err.Error()))
					continue
				}

				existingTimes := daySetFromRecords(existing, func(r marketdata.PriceRecord) time.Time { return r.Time })
				tradingDays := tradingDaysInRange(batchStart, batchEnd)
				if tradingDays == 0 {
					taskLogInfo(ctx, l, FinancialBackfillTaskName, fmt.Sprintf("financial backfill: skip %s batch — no trading days", inst.Symbol),
						slog.String("symbol", inst.Symbol))
					continue
				}
				if len(existingTimes) >= tradingDays {
					taskLogInfo(ctx, l, FinancialBackfillTaskName, fmt.Sprintf("financial backfill: skip %s batch — already full (%d trading days)", inst.Symbol, tradingDays),
						slog.String("symbol", inst.Symbol), slog.Int("existingDays", len(existingTimes)), slog.Int("tradingDaysInRange", tradingDays))
					continue
				}

				label := fmt.Sprintf("financial backfill: %s", inst.Symbol)
				points, fetchErr := FetchWith429Retry(ctx, deadline, label, func() ([]importer.PricePoint, error) {
					return client.FetchDailyPrices(ctx, inst.Symbol, batchStart, batchEnd)
				})
				if fetchErr != nil {
					if strings.Contains(fetchErr.Error(), "401") {
						return fmt.Errorf("invalid API key for %s: %w", inst.Symbol, fetchErr)
					}
					if strings.Contains(fetchErr.Error(), "429") {
						return fmt.Errorf("429 rate limit for %s: %w", inst.Symbol, fetchErr)
					}
					taskLogWarn(ctx, l, FinancialBackfillTaskName, fmt.Sprintf("financial backfill: skip %s batch — fetch failed: %v", inst.Symbol, fetchErr),
						slog.String("symbol", inst.Symbol), slog.String("error", fetchErr.Error()))
					continue
				}

				newPoints := pricePointsNewDays(points, existingTimes)
				if len(newPoints) == 0 {
					taskLogInfo(ctx, l, FinancialBackfillTaskName, fmt.Sprintf("financial backfill: skip %s batch — no new points (fetched %d)", inst.Symbol, len(points)),
						slog.String("symbol", inst.Symbol), slog.Int("fetched", len(points)))
					continue
				}

				if err := store.IngestPricesBulk(ctx, inst.Symbol, newPoints); err != nil {
					taskLogError(ctx, l, FinancialBackfillTaskName, fmt.Sprintf("financial backfill: ingest %s: %v", inst.Symbol, err), slog.String("symbol", inst.Symbol), slog.String("error", err.Error()))
					return fmt.Errorf("ingest %s: %w", inst.Symbol, err)
				}

				first, last := newPoints[0].Time.Format(dateFmt), newPoints[len(newPoints)-1].Time.Format(dateFmt)
				taskLogInfo(ctx, l, FinancialBackfillTaskName, fmt.Sprintf("financial backfill: %s — %d points (%s to %s)", inst.Symbol, len(newPoints), first, last),
					slog.String("symbol", inst.Symbol), slog.Int("points", len(newPoints)), slog.String("dateRange", first+" to "+last))
			}
		}

		return runMaintenance(ctx, store, l, FinancialBackfillTaskName)
	}
}
