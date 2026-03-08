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
	db       *gorm.DB
	registry *timeseries.Registry
}

// NewStore creates a new market data store backed by the given database.
// It initialises the timeseries registry and instrument table migrations.
func NewStore(db *gorm.DB) (*Store, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}

	if err := db.AutoMigrate(&dbInstrument{}); err != nil {
		return nil, fmt.Errorf("failed to migrate instruments: %w", err)
	}

	registry, err := timeseries.NewRegistry(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create timeseries registry: %w", err)
	}

	return &Store{db: db, registry: registry}, nil
}

// seriesName returns the timeseries name for a given instrument symbol.
func seriesName(symbol string) string {
	return seriesPrefix + symbol
}

// RegisterInstrument creates or updates a price time series for the given instrument symbol.
// This must be called before ingesting price data for the instrument.
func (s *Store) RegisterInstrument(symbol string) error {
	if symbol == "" {
		return fmt.Errorf("instrument symbol cannot be empty")
	}

	series := timeseries.TimeSeries{
		Name: seriesName(symbol),
		Retention: timeseries.SamplingPolicy{
			Precision:   defaultPrecision,
			Retention:   defaultRetention,
			AggregateFn: timeseries.AggregateAVG,
		},
	}
	if err := s.registry.RegisterSeries(series); err != nil {
		return fmt.Errorf("failed to register series for %q: %w", symbol, err)
	}
	return nil
}

// WipeData deletes all data from the market data store: instruments, time series,
// sampling policies, and records. This is used during backup restore to clear existing data.
func (s *Store) WipeData(ctx context.Context) error {
	// Order matters: records reference policies, policies reference series, instruments standalone
	tables := []string{"db_records", "db_sampling_policies", "db_time_series", "db_instruments"}
	for _, table := range tables {
		if err := s.db.WithContext(ctx).Unscoped().Table(table).Where("1 = 1").Delete(nil).Error; err != nil {
			return fmt.Errorf("failed to delete data in table '%s': %w", table, err)
		}
	}
	return nil
}

// ListPriceSymbols returns the list of instrument symbols that have a registered price series.
func (s *Store) ListPriceSymbols() ([]string, error) {
	all, err := s.registry.ListSeries()
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
