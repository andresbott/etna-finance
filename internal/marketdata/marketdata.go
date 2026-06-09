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

	seriesPrefix = "price:"
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
	// Order matters: records reference fields, fields reference series, instruments standalone
	tables := []string{"records", "fields", "series", "db_instruments"}
	for _, table := range tables {
		if err := s.db.WithContext(ctx).Unscoped().Table(table).Where("1 = 1").Delete(nil).Error; err != nil {
			return fmt.Errorf("failed to delete data in table '%s': %w", table, err)
		}
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
