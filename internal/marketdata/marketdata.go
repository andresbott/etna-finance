package marketdata

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-bumbu/timeseries"
	"gorm.io/gorm"
)

const (
	// defaultPrecision is the time bucket size for price data (daily).
	defaultPrecision = 24 * time.Hour
	// defaultRetention is how long price data is kept (10 years).
	defaultRetention = 10 * 365 * 24 * time.Hour

	seriesPrefix    = "price:"
	epsSeriesPrefix = "eps:"
)

// Store manages market data time series and instrument definitions.
type Store struct {
	db    *gorm.DB
	store *timeseries.Store
}

// NewStore creates a new market data store backed by the given database.
func NewStore(db *gorm.DB) (*Store, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}
	if err := db.AutoMigrate(&dbInstrument{}); err != nil {
		return nil, fmt.Errorf("failed to migrate instruments: %w", err)
	}
	ts, err := timeseries.New(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create timeseries store: %w", err)
	}
	return &Store{db: db, store: ts}, nil
}

// seriesName returns the timeseries name for a given instrument symbol.
func seriesName(symbol string) string {
	return seriesPrefix + symbol
}

// ohlcvSeries returns the Series definition for an instrument's price candles.
func ohlcvSeries(symbol string) timeseries.Series {
	return timeseries.Series{
		Name:      seriesName(symbol),
		Precision: defaultPrecision,
		Retention: defaultRetention,
		Fields: []timeseries.Field{
			{Name: "open", Aggregate: timeseries.AggFirst},
			{Name: "high", Aggregate: timeseries.AggMax},
			{Name: "low", Aggregate: timeseries.AggMin},
			{Name: "close", Aggregate: timeseries.AggLast},
			{Name: "volume", Aggregate: timeseries.AggSum},
		},
	}
}

// epsSeriesName returns the timeseries name for a given instrument symbol's EPS series.
func epsSeriesName(symbol string) string {
	return epsSeriesPrefix + symbol
}

// epsSeries returns the Series definition for an instrument's quarterly EPS. Both fields use
// AggLast so a restatement at the same date overwrites the prior value.
func epsSeries(symbol string) timeseries.Series {
	return timeseries.Series{
		Name:      epsSeriesName(symbol),
		Precision: defaultPrecision,
		Retention: defaultRetention,
		Fields: []timeseries.Field{
			{Name: "basic", Aggregate: timeseries.AggLast},
			{Name: "diluted", Aggregate: timeseries.AggLast},
		},
	}
}

// RegisterInstrument creates or updates the OHLCV series for the given symbol.
func (s *Store) RegisterInstrument(ctx context.Context, symbol string) error {
	if symbol == "" {
		return fmt.Errorf("instrument symbol cannot be empty")
	}
	if err := s.store.DefineSeries(ctx, ohlcvSeries(symbol)); err != nil {
		return fmt.Errorf("failed to define series for %q: %w", symbol, err)
	}
	return nil
}

// WipeData deletes all data from the market data store: instruments, time series,
// fields, and records. This is used during backup restore to clear existing data.
func (s *Store) WipeData(ctx context.Context) error {
	// The timeseries Store owns its own tables (records/fields/series) and wipes
	// them atomically under its lock, so we no longer hardcode those names here.
	if err := s.store.Wipe(ctx); err != nil {
		return fmt.Errorf("failed to wipe timeseries data: %w", err)
	}
	// db_instruments is etna's own table (migrated in NewStore), not the library's.
	if err := s.db.WithContext(ctx).Unscoped().Table("db_instruments").Where("1 = 1").Delete(nil).Error; err != nil {
		return fmt.Errorf("failed to delete instruments: %w", err)
	}
	return nil
}

// ListPriceSymbols returns instrument symbols that have a registered price series.
func (s *Store) ListPriceSymbols(ctx context.Context) ([]string, error) {
	all, err := s.store.ListSeries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list series: %w", err)
	}
	var symbols []string
	for _, ts := range all {
		if strings.HasPrefix(ts.Name, seriesPrefix) {
			symbols = append(symbols, strings.TrimPrefix(ts.Name, seriesPrefix))
		}
	}
	return symbols, nil
}
