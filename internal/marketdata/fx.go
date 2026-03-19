package marketdata

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-bumbu/timeseries"
)

const (
	fxSeriesPrefix = "fx:"
)

// fxSeriesName returns the timeseries name for a currency pair (main/secondary, e.g. CHF/USD).
func fxSeriesName(main, secondary string) string {
	return fxSeriesPrefix + main + "/" + secondary
}

// RegisterPair creates or updates a time series for the given currency pair (main/secondary).
// Call before ingesting rate data for the pair.
func (s *Store) RegisterPair(main, secondary string) error {
	if main == "" || secondary == "" {
		return fmt.Errorf("main and secondary currency cannot be empty")
	}
	series := timeseries.TimeSeries{
		Name: fxSeriesName(main, secondary),
		Retention: timeseries.SamplingPolicy{
			Precision:   defaultPrecision,
			Retention:   defaultRetention,
			AggregateFn: timeseries.AggregateAVG,
		},
	}
	if err := s.registry.RegisterSeries(series); err != nil {
		return fmt.Errorf("failed to register series for %s/%s: %w", main, secondary, err)
	}
	return nil
}

// ListFXPairs returns pairs that have a registered FX series (format "MAIN/SECONDARY").
func (s *Store) ListFXPairs() ([]string, error) {
	all, err := s.registry.ListSeries()
	if err != nil {
		return nil, fmt.Errorf("failed to list series: %w", err)
	}
	var pairs []string
	for _, ts := range all {
		if strings.HasPrefix(ts.Name, fxSeriesPrefix) {
			pairs = append(pairs, strings.TrimPrefix(ts.Name, fxSeriesPrefix))
		}
	}
	return pairs, nil
}

// RateRecord is a stored exchange rate data point.
type RateRecord struct {
	ID        uint
	Main      string
	Secondary string
	Time      time.Time
	Rate      float64
}

// IngestRate records a single exchange rate for the pair (main/secondary). Series is auto-registered if needed.
func (s *Store) IngestRate(_ context.Context, main, secondary string, t time.Time, rate float64) error {
	if main == "" || secondary == "" {
		return fmt.Errorf("main and secondary currency cannot be empty")
	}
	if err := s.RegisterPair(main, secondary); err != nil {
		return fmt.Errorf("failed to ensure series for %s/%s: %w", main, secondary, err)
	}
	_, err := s.registry.Ingest(fxSeriesName(main, secondary), t, rate)
	if err != nil {
		return fmt.Errorf("failed to ingest rate for %s/%s: %w", main, secondary, err)
	}
	return nil
}

// RatePoint is a single rate observation at a point in time.
type RatePoint struct {
	Time time.Time
	Rate float64
}

// IngestRatesBulk records multiple rate points for the given pair in one operation.
func (s *Store) IngestRatesBulk(_ context.Context, main, secondary string, points []RatePoint) error {
	if main == "" || secondary == "" {
		return fmt.Errorf("main and secondary currency cannot be empty")
	}
	if len(points) == 0 {
		return nil
	}
	if err := s.RegisterPair(main, secondary); err != nil {
		return fmt.Errorf("failed to ensure series for %s/%s: %w", main, secondary, err)
	}
	dataPoints := make([]timeseries.DataPoint, len(points))
	for i, p := range points {
		dataPoints[i] = timeseries.DataPoint{Time: p.Time, Value: p.Rate}
	}
	_, err := s.registry.IngestBulk(fxSeriesName(main, secondary), dataPoints)
	if err != nil {
		return fmt.Errorf("failed to bulk ingest rates for %s/%s: %w", main, secondary, err)
	}
	return nil
}

// RateHistory returns rate records for the pair within a time range.
// Zero time values mean unbounded. Returns nil slice when series does not exist or has no data.
func (s *Store) RateHistory(_ context.Context, main, secondary string, start, end time.Time) ([]RateRecord, error) {
	if main == "" || secondary == "" {
		return nil, fmt.Errorf("main and secondary currency cannot be empty")
	}
	records, err := s.registry.ListRecords(fxSeriesName(main, secondary), start, end)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "series not found") || strings.Contains(errStr, "record not found") {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list rates for %s/%s: %w", main, secondary, err)
	}
	out := make([]RateRecord, len(records))
	for i, r := range records {
		out[i] = RateRecord{
			ID:        r.Id,
			Main:      main,
			Secondary: secondary,
			Time:      r.Time,
			Rate:      r.Value,
		}
	}
	return out, nil
}

// RateAt returns the most recent rate record for the pair at or before time t. Returns nil if no data.
func (s *Store) RateAt(_ context.Context, main, secondary string, t time.Time) (*RateRecord, error) {
	if main == "" || secondary == "" {
		return nil, fmt.Errorf("main and secondary currency cannot be empty")
	}
	rec, err := s.registry.RecordAt(fxSeriesName(main, secondary), t)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "series not found") || strings.Contains(errStr, "record not found") {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get rate for %s/%s at %s: %w", main, secondary, t.Format(time.DateOnly), err)
	}
	if rec == nil {
		return nil, nil
	}
	return &RateRecord{
		ID:        rec.Id,
		Main:      main,
		Secondary: secondary,
		Time:      rec.Time,
		Rate:      rec.Value,
	}, nil
}

// LatestRate returns the most recent rate record for the pair. Returns nil if no data.
func (s *Store) LatestRate(_ context.Context, main, secondary string) (*RateRecord, error) {
	if main == "" || secondary == "" {
		return nil, fmt.Errorf("main and secondary currency cannot be empty")
	}
	rec, err := s.registry.RecordAt(fxSeriesName(main, secondary), time.Now())
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "series not found") || strings.Contains(errStr, "record not found") {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest rate for %s/%s: %w", main, secondary, err)
	}
	if rec == nil {
		return nil, nil
	}
	return &RateRecord{
		ID:        rec.Id,
		Main:      main,
		Secondary: secondary,
		Time:      rec.Time,
		Rate:      rec.Value,
	}, nil
}

// RateUpdate holds optional fields for a partial rate update.
type RateUpdate struct {
	Time *time.Time
	Rate *float64
}

// UpdateRate applies a partial update to an existing rate record.
func (s *Store) UpdateRate(_ context.Context, id uint, in RateUpdate) error {
	if id == 0 {
		return fmt.Errorf("record id is required for update")
	}
	update := timeseries.RecordUpdate{
		Time:  in.Time,
		Value: in.Rate,
	}
	return s.registry.UpdateRecord(id, update)
}

// DeleteRate removes a rate record by ID.
func (s *Store) DeleteRate(_ context.Context, id uint) error {
	return s.registry.DeleteRecord(id)
}
