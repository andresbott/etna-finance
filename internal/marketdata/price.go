package marketdata

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-bumbu/timeseries"
)

// PricePoint represents a single price observation at a point in time.
type PricePoint struct {
	Time  time.Time
	Price float64
}

// PriceRecord is a stored price data point with its record ID.
type PriceRecord struct {
	ID     uint
	Symbol string
	Time   time.Time
	Price  float64
}

// IngestPrice records a single price data point for the given instrument.
// The series is auto-registered if it does not exist yet.
func (s *Store) IngestPrice(_ context.Context, symbol string, t time.Time, price float64) error {
	if symbol == "" {
		return fmt.Errorf("instrument symbol cannot be empty")
	}
	if err := s.RegisterInstrument(symbol); err != nil {
		return fmt.Errorf("failed to ensure series for %q: %w", symbol, err)
	}
	_, err := s.registry.Ingest(seriesName(symbol), t, price)
	if err != nil {
		return fmt.Errorf("failed to ingest price for %q: %w", symbol, err)
	}
	return nil
}

// IngestPricesBulk records multiple price data points for the given instrument in one operation.
// The series is auto-registered if it does not exist yet.
func (s *Store) IngestPricesBulk(_ context.Context, symbol string, points []PricePoint) error {
	if symbol == "" {
		return fmt.Errorf("instrument symbol cannot be empty")
	}
	if len(points) == 0 {
		return nil
	}
	if err := s.RegisterInstrument(symbol); err != nil {
		return fmt.Errorf("failed to ensure series for %q: %w", symbol, err)
	}

	dataPoints := make([]timeseries.DataPoint, len(points))
	for i, p := range points {
		dataPoints[i] = timeseries.DataPoint{
			Time:  p.Time,
			Value: p.Price,
		}
	}

	_, err := s.registry.IngestBulk(seriesName(symbol), dataPoints)
	if err != nil {
		return fmt.Errorf("failed to bulk ingest prices for %q: %w", symbol, err)
	}
	return nil
}

// PriceHistory returns price records for the given instrument within a time range.
// Use zero time values for unbounded queries (e.g. time.Time{} for no lower/upper bound).
// Returns an empty list when the series does not exist or has no data.
func (s *Store) PriceHistory(_ context.Context, symbol string, start, end time.Time) ([]PriceRecord, error) {
	if symbol == "" {
		return nil, fmt.Errorf("instrument symbol cannot be empty")
	}

	records, err := s.registry.ListRecords(seriesName(symbol), start, end)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "series not found") || strings.Contains(errStr, "record not found") {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list prices for %q: %w", symbol, err)
	}

	out := make([]PriceRecord, len(records))
	for i, r := range records {
		out[i] = PriceRecord{
			ID:     r.Id,
			Symbol: symbol,
			Time:   r.Time,
			Price:  r.Value,
		}
	}
	return out, nil
}

// LatestPrice returns the most recent price record for the given instrument.
// Returns nil if no price data exists.
func (s *Store) LatestPrice(_ context.Context, symbol string) (*PriceRecord, error) {
	if symbol == "" {
		return nil, fmt.Errorf("instrument symbol cannot be empty")
	}

	rec, err := s.registry.RecordAt(seriesName(symbol), time.Now())
	if err != nil {
		// No data: series not registered yet, or series has no records.
		errStr := err.Error()
		if strings.Contains(errStr, "series not found") || strings.Contains(errStr, "record not found") {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest price for %q: %w", symbol, err)
	}
	if rec == nil {
		return nil, nil
	}

	return &PriceRecord{
		ID:     rec.Id,
		Symbol: symbol,
		Time:   rec.Time,
		Price:  rec.Value,
	}, nil
}

// PriceUpdate holds optional fields for a partial price update.
type PriceUpdate struct {
	Time  *time.Time
	Price *float64
}

// UpdatePrice applies a partial update to an existing price record.
func (s *Store) UpdatePrice(_ context.Context, id uint, in PriceUpdate) error {
	if id == 0 {
		return fmt.Errorf("record id is required for update")
	}
	update := timeseries.RecordUpdate{
		Time:  in.Time,
		Value: in.Price,
	}
	return s.registry.UpdateRecord(id, update)
}

// DeletePrice removes a price record by its ID.
func (s *Store) DeletePrice(_ context.Context, id uint) error {
	return s.registry.DeleteRecord(id)
}

// Maintenance runs retention cleanup and bucket aggregation for all registered series.
// This should be called periodically (e.g. daily via a cron job or ticker).
func (s *Store) Maintenance(ctx context.Context) error {
	return s.registry.Maintenance(ctx)
}
