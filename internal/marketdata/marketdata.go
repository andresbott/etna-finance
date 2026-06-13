package marketdata

import (
	"context"
	"errors"
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

	// Series label keys. etna stores structured identity as timeseries labels
	// instead of parsing it back out of the series name.
	labelType      = "type"
	labelSymbol    = "symbol"
	labelMain      = "main"
	labelSecondary = "secondary"

	// labelType values.
	typePrice = "price"
	typeEPS   = "eps"
	typeFX    = "fx"

	// StockInstrumentType is the instrument type that has EPS data. EPS is extracted from SEC
	// filings, which only exist for individual stocks — ETFs, funds, currencies, etc. have none.
	// The EPS series is defined (at instrument creation) only for instruments of this type.
	StockInstrumentType = "stock"
)

// isStockType reports whether an instrument type has EPS data (see StockInstrumentType).
func isStockType(instrumentType string) bool {
	return strings.EqualFold(instrumentType, StockInstrumentType)
}

// ErrDateImmutable is returned (wrapped) by the Edit* methods when the edited
// point's time differs from the original timestamp. A record's date is its
// identity within the series and cannot be changed by an edit; to move a record
// to a different date, delete it and create a new one. Test for it with errors.Is.
var ErrDateImmutable = errors.New("record date cannot be changed")

// ErrRecordNotFound is returned by the Delete*At methods when no record exists at
// the requested timestamp, so a delete that removed nothing is reported as a
// not-found rather than a silent success. Test for it with errors.Is.
var ErrRecordNotFound = errors.New("record not found")

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
	s := &Store{db: db, store: ts}

	// Migration: price series are now defined at instrument creation (see CreateInstrument), not
	// lazily on the first ingest. Define the series for any pre-existing instrument that does not
	// have one yet, so ingest paths never hit ErrSeriesNotFound for a known instrument.
	if err := s.ensureInstrumentSeries(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure instrument series: %w", err)
	}

	// One-time migration: label any series created before labels existed.
	if err := s.backfillSeriesLabels(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to backfill series labels: %w", err)
	}

	return s, nil
}

// ensureInstrumentSeries defines the price series for every existing instrument that lacks one (and
// the EPS series for stock-type instruments). DefineSeries is idempotent and cheap on the no-op
// path, so this is safe to run on every startup.
func (s *Store) ensureInstrumentSeries(ctx context.Context) error {
	var rows []dbInstrument
	if err := s.db.WithContext(ctx).Model(&dbInstrument{}).Select("symbol", "type").Find(&rows).Error; err != nil {
		return fmt.Errorf("list instruments: %w", err)
	}
	for _, row := range rows {
		if row.Symbol == "" {
			continue
		}
		if err := s.defineInstrumentSeries(ctx, row.Symbol, row.Type); err != nil {
			return err
		}
	}
	return nil
}

// backfillSeriesLabels is a one-time migration: series created before labels
// existed carry none, so derive their labels from the legacy name prefix — the
// last sanctioned name-parse — and re-define with labels. Idempotent: a series
// that already has a "type" label is skipped, and re-defining an already-labeled
// series hits the timeseries no-op fast path. Remove once all deployments have
// migrated past the labels release.
func (s *Store) backfillSeriesLabels(ctx context.Context) error {
	all, err := s.store.ListSeries(ctx)
	if err != nil {
		return fmt.Errorf("backfill: list series: %w", err)
	}
	for _, ts := range all {
		if ts.Labels[labelType] != "" {
			continue
		}
		var cfg timeseries.Series
		switch {
		case strings.HasPrefix(ts.Name, seriesPrefix):
			cfg = ohlcvSeries(strings.TrimPrefix(ts.Name, seriesPrefix))
		case strings.HasPrefix(ts.Name, epsSeriesPrefix):
			cfg = epsSeries(strings.TrimPrefix(ts.Name, epsSeriesPrefix))
		case strings.HasPrefix(ts.Name, fxSeriesPrefix):
			parts := strings.SplitN(strings.TrimPrefix(ts.Name, fxSeriesPrefix), "/", 2)
			if len(parts) != 2 {
				continue
			}
			cfg = fxSeries(parts[0], parts[1])
		default:
			continue
		}
		if err := s.store.DefineSeries(ctx, cfg); err != nil {
			return fmt.Errorf("backfill labels for %q: %w", ts.Name, err)
		}
	}
	return nil
}

// defineInstrumentSeries defines an instrument's price series, plus its EPS series when the type is
// a stock. Used by the create/restore paths and the startup migration so the series exist before any
// ingest (which no longer auto-registers).
func (s *Store) defineInstrumentSeries(ctx context.Context, symbol, instrumentType string) error {
	if err := s.RegisterInstrument(ctx, symbol); err != nil {
		return err
	}
	if isStockType(instrumentType) {
		if err := s.RegisterEPSSeries(ctx, symbol); err != nil {
			return err
		}
	}
	return nil
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
		Labels:    map[string]string{labelType: typePrice, labelSymbol: symbol},
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
		Labels:    map[string]string{labelType: typeEPS, labelSymbol: symbol},
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
	all, err := s.store.ListSeries(ctx, timeseries.MatchLabel(labelType, typePrice))
	if err != nil {
		return nil, fmt.Errorf("failed to list series: %w", err)
	}
	var symbols []string
	for _, ts := range all {
		if sym := ts.Labels[labelSymbol]; sym != "" {
			symbols = append(symbols, sym)
		}
	}
	return symbols, nil
}
