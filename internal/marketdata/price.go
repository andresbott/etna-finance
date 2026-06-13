package marketdata

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/andresbott/etna/internal/marketdata/importer"
	"github.com/go-bumbu/timeseries"
)

// PricePoint is a single OHLCV candle at a point in time.
type PricePoint struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

// PriceRecord is a stored OHLCV candle for a symbol.
type PriceRecord struct {
	Symbol string
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

func (p PricePoint) values() map[string]float64 {
	return map[string]float64{
		"open": p.Open, "high": p.High, "low": p.Low, "close": p.Close, "volume": p.Volume,
	}
}

func pointToPriceRecord(symbol string, p timeseries.Point) PriceRecord {
	return PriceRecord{
		Symbol: symbol,
		Time:   p.Time,
		Open:   p.Values["open"],
		High:   p.Values["high"],
		Low:    p.Values["low"],
		Close:  p.Values["close"],
		Volume: p.Values["volume"],
	}
}

// IngestPrice records a single OHLCV candle. The instrument's series must already exist
// (defined by CreateInstrument); this does not auto-register it.
func (s *Store) IngestPrice(ctx context.Context, symbol string, p PricePoint) error {
	if symbol == "" {
		return fmt.Errorf("instrument symbol cannot be empty")
	}
	if err := s.store.Write(ctx, seriesName(symbol), timeseries.Point{Time: p.Time.UTC(), Values: p.values()}); err != nil {
		return fmt.Errorf("failed to write price for %q: %w", symbol, err)
	}
	return nil
}

// IngestPricesBulk records many OHLCV candles in one transaction. The instrument's series must
// already exist (defined by CreateInstrument); this does not auto-register it.
func (s *Store) IngestPricesBulk(ctx context.Context, symbol string, points []PricePoint) error {
	if symbol == "" {
		return fmt.Errorf("instrument symbol cannot be empty")
	}
	if len(points) == 0 {
		return nil
	}
	pts := make([]timeseries.Point, len(points))
	for i, p := range points {
		pts[i] = timeseries.Point{Time: p.Time.UTC(), Values: p.values()}
	}
	if err := s.store.WriteMany(ctx, seriesName(symbol), pts); err != nil {
		return fmt.Errorf("failed to bulk write prices for %q: %w", symbol, err)
	}
	return nil
}

// PriceHistory returns OHLCV candles in [start, end]. Zero times mean unbounded.
// Returns nil when the series does not exist.
func (s *Store) PriceHistory(ctx context.Context, symbol string, start, end time.Time) ([]PriceRecord, error) {
	if symbol == "" {
		return nil, fmt.Errorf("instrument symbol cannot be empty")
	}
	points, err := s.store.Range(ctx, seriesName(symbol), start, end)
	if err != nil {
		if errors.Is(err, timeseries.ErrSeriesNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list prices for %q: %w", symbol, err)
	}
	out := make([]PriceRecord, len(points))
	for i, p := range points {
		out[i] = pointToPriceRecord(symbol, p)
	}
	return out, nil
}

// LatestPrice returns the most recent candle, or nil if none. As with PriceAt, a
// partial candle (the newest timestamp missing an OHLCV leg) is rejected with an
// error rather than zero-filled, so it can never silently feed a corrupt price.
func (s *Store) LatestPrice(ctx context.Context, symbol string) (*PriceRecord, error) {
	if symbol == "" {
		return nil, fmt.Errorf("instrument symbol cannot be empty")
	}
	p, cov, err := s.store.Latest(ctx, seriesName(symbol))
	if err != nil {
		if errors.Is(err, timeseries.ErrSeriesNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest price for %q: %w", symbol, err)
	}
	switch cov {
	case timeseries.CoverageNone:
		return nil, nil
	case timeseries.CoveragePartial:
		return nil, fmt.Errorf("partial price candle for %q at latest timestamp", symbol)
	}
	rec := pointToPriceRecord(symbol, p)
	return &rec, nil
}

// PriceAt returns the as-of candle (each field's latest value ≤ t). Record.Time is t.
// Returns nil if no data at or before t. Used for portfolio valuation (Close).
//
// The store's At is best-effort per field, so it can return a partial snapshot
// (some OHLCV legs missing) when a field has a gap in its history. Since candles
// are always written whole, a partial snapshot is an anomaly: rather than
// silently zero-filling the missing legs into a real-looking candle (which would
// corrupt valuation), PriceAt rejects anything short of full coverage.
func (s *Store) PriceAt(ctx context.Context, symbol string, t time.Time) (*PriceRecord, error) {
	if symbol == "" {
		return nil, fmt.Errorf("instrument symbol cannot be empty")
	}
	p, cov, err := s.store.At(ctx, seriesName(symbol), t)
	if err != nil {
		if errors.Is(err, timeseries.ErrSeriesNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get price for %q at %v: %w", symbol, t, err)
	}
	switch cov {
	case timeseries.CoverageNone:
		return nil, nil
	case timeseries.CoveragePartial:
		return nil, fmt.Errorf("partial price candle for %q at %v", symbol, t)
	}
	rec := pointToPriceRecord(symbol, p)
	return &rec, nil
}

// EditPrice overwrites the candle at its timestamp. The date is the record's identity and cannot
// be changed: if a non-zero oldTime differs from p.Time, EditPrice returns ErrDateImmutable and
// changes nothing. A zero oldTime, or oldTime equal to p.Time, is a plain upsert at p.Time.
func (s *Store) EditPrice(ctx context.Context, symbol string, oldTime time.Time, p PricePoint) error {
	if symbol == "" {
		return fmt.Errorf("instrument symbol cannot be empty")
	}
	if !oldTime.IsZero() && !oldTime.Equal(p.Time) {
		return fmt.Errorf("cannot edit price for %q: %w", symbol, ErrDateImmutable)
	}
	return s.IngestPrice(ctx, symbol, p)
}

// DeletePriceAt removes the candle at exactly t.
func (s *Store) DeletePriceAt(ctx context.Context, symbol string, t time.Time) error {
	if symbol == "" {
		return fmt.Errorf("instrument symbol cannot be empty")
	}
	deleted, err := s.store.Delete(ctx, seriesName(symbol), t)
	if err != nil {
		return fmt.Errorf("failed to delete price for %q: %w", symbol, err)
	}
	if !deleted {
		return fmt.Errorf("no price for %q at %s: %w", symbol, t.Format(time.DateOnly), ErrRecordNotFound)
	}
	return nil
}

// PricePointsFromImporter converts points yielded by marketdata/importer into the form
// expected by IngestPricesBulk. Use this when writing importer results to the store.
func PricePointsFromImporter(pts []importer.PricePoint) []PricePoint {
	if len(pts) == 0 {
		return nil
	}
	out := make([]PricePoint, len(pts))
	for i, p := range pts {
		out[i] = PricePoint{Time: p.Time, Open: p.Open, High: p.High, Low: p.Low, Close: p.Close, Volume: p.Volume}
	}
	return out
}

// Maintenance runs retention cleanup and per-field bucket aggregation.
func (s *Store) Maintenance(ctx context.Context) error {
	return s.store.Maintain(ctx)
}
