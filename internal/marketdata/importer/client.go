package importer

import (
	"context"
	"time"
)

// PricePoint is a single OHLCV candle at a point in time.
// Consumers (e.g. the market data import job) can convert to marketdata.PricePoint
// and write to the store via Store.IngestPricesBulk.
type PricePoint struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

// Client is the interface for fetching market data from an external source.
// Implementations may wrap a single API (e.g. Massive) or a pool of clients for key rotation.
// The client only yields data; the caller is responsible for persisting it.
type Client interface {
	// FetchDailyPrices returns daily OHLCV candles for the given symbol
	// in the [start, end] date range (inclusive). Times should be interpreted in UTC.
	FetchDailyPrices(ctx context.Context, symbol string, start, end time.Time) ([]PricePoint, error)
}

// TickerDetails is reference/metadata about a tradeable instrument, as returned by an
// external provider (e.g. Massive). Fields carry the provider's RAW values; mapping to
// app-facing values (e.g. MIC code -> exchange name) is the caller's responsibility.
type TickerDetails struct {
	Name     string // e.g. "Apple Inc."
	Currency string // ISO currency, upper-cased, e.g. "USD"
	Type     string // raw provider type code, e.g. "CS", "ETF"
	Exchange string // raw MIC code, e.g. "XNAS"
	Notes    string // short human description (exchange + description)
	Found    bool   // false when the provider returned no match for the symbol
}

// ReferenceClient fetches reference/metadata for a single symbol. Implementations may wrap a
// single API key or a pool of clients for key rotation. A not-found symbol must be returned as
// TickerDetails{Found: false} with a nil error (only transport/parse failures return an error).
type ReferenceClient interface {
	GetTickerDetails(ctx context.Context, symbol string) (TickerDetails, error)
}
