package marketdata

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/andresbott/etna/internal/marketdata/importer"
	"github.com/go-bumbu/timeseries"
)

// EPSPoint is a single quarterly EPS observation to write to the store.
type EPSPoint struct {
	Time    time.Time
	Basic   float64
	Diluted float64
}

// EPSRecord is a stored EPS observation for a symbol.
type EPSRecord struct {
	Symbol  string
	Time    time.Time
	Basic   float64
	Diluted float64
}

func (p EPSPoint) values() map[string]float64 {
	return map[string]float64{"basic": p.Basic, "diluted": p.Diluted}
}

func pointToEPSRecord(symbol string, p timeseries.Point) EPSRecord {
	return EPSRecord{
		Symbol:  symbol,
		Time:    p.Time,
		Basic:   p.Values["basic"],
		Diluted: p.Values["diluted"],
	}
}

// RegisterEPSSeries defines (creates or updates) the eps: series for the symbol. Unlike
// RegisterInstrument it does not touch the price series or instrument metadata. Adding the first
// EPS point via the API is how the series is introduced for a symbol that was not auto-defined,
// so the create handlers call this before ingest (mirroring RegisterPair for FX).
func (s *Store) RegisterEPSSeries(ctx context.Context, symbol string) error {
	if symbol == "" {
		return fmt.Errorf("instrument symbol cannot be empty")
	}
	if err := s.store.DefineSeries(ctx, epsSeries(symbol)); err != nil {
		return fmt.Errorf("failed to define EPS series for %q: %w", symbol, err)
	}
	return nil
}

// IngestEPS records a single EPS observation. The EPS series must already exist (defined by
// CreateInstrument for stock-type instruments); this does not auto-register it.
func (s *Store) IngestEPS(ctx context.Context, symbol string, p EPSPoint) error {
	if symbol == "" {
		return fmt.Errorf("instrument symbol cannot be empty")
	}
	if err := s.store.Write(ctx, epsSeriesName(symbol), timeseries.Point{Time: p.Time.UTC(), Values: p.values()}); err != nil {
		return fmt.Errorf("failed to write EPS for %q: %w", symbol, err)
	}
	return nil
}

// IngestEPSBulk records many EPS observations in one transaction. The EPS series must already exist
// (defined by CreateInstrument for stock-type instruments); this does not auto-register it.
func (s *Store) IngestEPSBulk(ctx context.Context, symbol string, points []EPSPoint) error {
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
	if err := s.store.WriteMany(ctx, epsSeriesName(symbol), pts); err != nil {
		return fmt.Errorf("failed to bulk write EPS for %q: %w", symbol, err)
	}
	return nil
}

// EPSHistory returns EPS records in [start, end]. Zero times mean unbounded. Returns nil when the
// series does not exist.
func (s *Store) EPSHistory(ctx context.Context, symbol string, start, end time.Time) ([]EPSRecord, error) {
	if symbol == "" {
		return nil, fmt.Errorf("instrument symbol cannot be empty")
	}
	points, err := s.store.Range(ctx, epsSeriesName(symbol), start, end)
	if err != nil {
		if errors.Is(err, timeseries.ErrSeriesNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list EPS for %q: %w", symbol, err)
	}
	out := make([]EPSRecord, len(points))
	for i, p := range points {
		out[i] = pointToEPSRecord(symbol, p)
	}
	return out, nil
}

// LatestEPS returns the most recent EPS record, or nil if none. A partial record
// (the newest timestamp missing the basic or diluted leg) is rejected with an
// error rather than zero-filled.
func (s *Store) LatestEPS(ctx context.Context, symbol string) (*EPSRecord, error) {
	if symbol == "" {
		return nil, fmt.Errorf("instrument symbol cannot be empty")
	}
	p, cov, err := s.store.Latest(ctx, epsSeriesName(symbol))
	if err != nil {
		if errors.Is(err, timeseries.ErrSeriesNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest EPS for %q: %w", symbol, err)
	}
	switch cov {
	case timeseries.CoverageNone:
		return nil, nil
	case timeseries.CoveragePartial:
		return nil, fmt.Errorf("partial EPS record for %q at latest timestamp", symbol)
	}
	rec := pointToEPSRecord(symbol, p)
	return &rec, nil
}

// EditEPS overwrites the EPS observation at its timestamp. The date is the record's identity and
// cannot be changed: if a non-zero oldTime differs from p.Time, EditEPS returns ErrDateImmutable
// and changes nothing. A zero oldTime, or oldTime equal to p.Time, is a plain upsert at p.Time.
func (s *Store) EditEPS(ctx context.Context, symbol string, oldTime time.Time, p EPSPoint) error {
	if symbol == "" {
		return fmt.Errorf("instrument symbol cannot be empty")
	}
	if !oldTime.IsZero() && !oldTime.Equal(p.Time) {
		return fmt.Errorf("cannot edit EPS for %q: %w", symbol, ErrDateImmutable)
	}
	return s.IngestEPS(ctx, symbol, p)
}

// DeleteEPSAt removes the EPS observation at exactly t.
func (s *Store) DeleteEPSAt(ctx context.Context, symbol string, t time.Time) error {
	if symbol == "" {
		return fmt.Errorf("instrument symbol cannot be empty")
	}
	deleted, err := s.store.Delete(ctx, epsSeriesName(symbol), t)
	if err != nil {
		return fmt.Errorf("failed to delete EPS for %q: %w", symbol, err)
	}
	if !deleted {
		return fmt.Errorf("no EPS for %q at %s: %w", symbol, t.Format(time.DateOnly), ErrRecordNotFound)
	}
	return nil
}

// EPSPointsFromImporter converts importer EPS points into store EPSPoints for IngestEPSBulk.
func EPSPointsFromImporter(pts []importer.EPSPoint) []EPSPoint {
	if len(pts) == 0 {
		return nil
	}
	out := make([]EPSPoint, len(pts))
	for i, p := range pts {
		out[i] = EPSPoint{Time: p.Time, Basic: p.Basic, Diluted: p.Diluted}
	}
	return out
}
