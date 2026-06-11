package tasks

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/andresbott/etna/internal/marketdata"
	"github.com/andresbott/etna/internal/marketdata/importer"
	"github.com/go-bumbu/tempo"
)

// epsInstrumentType is the instrument type EPS applies to. EPS is extracted from SEC filings,
// which only exist for individual stocks — ETFs, funds, currencies, etc. have no EPS.
const epsInstrumentType = "stock"

const EPSImportTaskName = "eps-import"

// MaxEPSSyncDuration is the overall budget for the EPS import task.
const MaxEPSSyncDuration = 2 * time.Hour

// epsLookbackYears is how far back to look for existing filings when deduplicating (quarterly data).
const epsLookbackYears = 2

// EPSImportTaskDef is the task definition for the EPS import task, used in the API task list.
var EPSImportTaskDef = TaskDef{
	ID:          EPSImportTaskName,
	Name:        "EPS import",
	Description: "Fetch quarterly EPS (basic and diluted) from SEC filings and store it locally.",
}

// NewEPSImportTaskFn returns a task function that fetches recent quarterly EPS for every
// stock-type instrument and stores basic/diluted EPS in the eps: timeseries series. Non-stock
// instruments (ETFs, funds, currencies, untyped) are skipped without an API call. Instruments that
// error or return no financials are skipped — never aborting the whole task. Re-runs deduplicate
// against existing filings over a 2-year lookback, so the task is idempotent and safe to run daily.
func NewEPSImportTaskFn(store *marketdata.Store, client importer.FundamentalsClient) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if store == nil {
			return fmt.Errorf("market data store is required")
		}
		if client == nil {
			return fmt.Errorf("no fundamentals client configured — set API key via ETNA_MARKETDATAIMPORTERS_MASSIVE_APIKEYS_0 or config file")
		}
		tempo.Info(ctx, "starting EPS import")

		instruments, err := store.ListInstruments(ctx)
		if err != nil {
			tempo.Error(ctx, fmt.Sprintf("EPS import: list instruments failed: %v", err))
			return fmt.Errorf("list instruments: %w", err)
		}
		tempo.Info(ctx, fmt.Sprintf("EPS import: %d instruments", len(instruments)))

		deadline := time.Now().Add(MaxEPSSyncDuration)
		lookbackStart := time.Now().UTC().AddDate(-epsLookbackYears, 0, 0)
		lookbackEnd := time.Now().UTC()

		for _, inst := range instruments {
			// EPS only exists for individual stocks; skip ETFs/funds/currencies/untyped instruments
			// before spending an API call (and rate-limit budget) on them.
			if !strings.EqualFold(inst.Type, epsInstrumentType) {
				tempo.Info(ctx, fmt.Sprintf("EPS import: skip %s — type %q is not %s", inst.Symbol, inst.Type, epsInstrumentType))
				continue
			}

			existing, err := store.EPSHistory(ctx, inst.Symbol, lookbackStart, lookbackEnd)
			if err != nil {
				tempo.Info(ctx, fmt.Sprintf("EPS import: skip %s — history failed: %v", inst.Symbol, err))
				continue
			}
			existingDays := daySetFromRecords(existing, func(r marketdata.EPSRecord) time.Time { return r.Time })

			// Per-symbol deadline (max 5 min) so one stuck symbol does not block the rest.
			symbolDeadline := time.Now().Add(5 * time.Minute)
			if symbolDeadline.After(deadline) {
				symbolDeadline = deadline
			}
			filings, fetchErr := FetchWith429Retry(ctx, symbolDeadline, "EPS import: "+inst.Symbol, func() ([]importer.EPSPoint, error) {
				return client.FetchEPS(ctx, inst.Symbol)
			})
			if fetchErr != nil {
				// Includes "no financials found" for non-stocks and 429-deadline exhaustion. Skip, never abort.
				tempo.Info(ctx, fmt.Sprintf("EPS import: skip %s — fetch failed: %v", inst.Symbol, fetchErr))
				continue
			}

			// Sort oldest first so the series fills forward.
			sort.Slice(filings, func(i, j int) bool { return filings[i].Time.Before(filings[j].Time) })

			newPoints := epsPointsNewDays(filings, existingDays)
			if len(newPoints) == 0 {
				tempo.Info(ctx, fmt.Sprintf("EPS import: skip %s — all %d filings already stored", inst.Symbol, len(filings)))
				continue
			}
			if time.Now().After(deadline) {
				tempo.Info(ctx, "EPS import: stopping before full run (deadline reached)")
				break
			}
			if err := store.IngestEPSBulk(ctx, inst.Symbol, newPoints); err != nil {
				tempo.Error(ctx, fmt.Sprintf("EPS import: ingest %s: %v", inst.Symbol, err))
				return fmt.Errorf("ingest EPS %s: %w", inst.Symbol, err)
			}
			first, last := newPoints[0].Time.Format(dateFmt), newPoints[len(newPoints)-1].Time.Format(dateFmt)
			tempo.Info(ctx, fmt.Sprintf("EPS import: imported %s — %d filings (%s to %s)", inst.Symbol, len(newPoints), first, last))
		}

		tempo.Info(ctx, "EPS import completed")
		return nil
	}
}
