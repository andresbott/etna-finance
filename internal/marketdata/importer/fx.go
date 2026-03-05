package importer

import (
	"context"
	"time"
)

// RatePoint is a single exchange rate observation at a point in time.
// Consumers can convert to marketdata.RatePoint and write via Store.IngestRatesBulk.
type RatePoint struct {
	Time time.Time
	Rate float64
}

// FXClient is the interface for fetching currency exchange rates from an external source.
// Implementations may call a third-party API. The client only yields data; the caller persists it.
type FXClient interface {
	// FetchDailyRates returns daily rate points for the pair (main/secondary)
	// in the [start, end) or [start, end] date range. Times are typically UTC midnight.
	FetchDailyRates(ctx context.Context, main, secondary string, start, end time.Time) ([]RatePoint, error)
}
