package importer

import (
	"context"
	"time"
)

// PricePoint is a single price observation at a point in time.
// Consumers (e.g. the market data import job) can convert to marketdata.PricePoint
// and write to the store via Store.IngestPricesBulk.
type PricePoint struct {
	Time  time.Time
	Price float64
}

// Client is the interface for fetching market data from an external source.
// Implementations may wrap a single API (e.g. Massive) or a pool of clients for key rotation.
// The client only yields data; the caller is responsible for persisting it.
type Client interface {
	// FetchDailyPrices returns daily price points (typically close) for the given symbol
	// in the [start, end] date range (inclusive). Times should be interpreted in UTC.
	FetchDailyPrices(ctx context.Context, symbol string, start, end time.Time) ([]PricePoint, error)
}
