package marketdata

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-bumbu/timeseries"
)

const (
	fxSeriesPrefix = "fx:"
	fxField        = "rate"
)

// fxSeriesName returns the timeseries name for a currency pair (main/secondary, e.g. CHF/USD).
func fxSeriesName(main, secondary string) string {
	return fxSeriesPrefix + main + "/" + secondary
}

func fxSeries(main, secondary string) timeseries.Series {
	return timeseries.Series{
		Name:      fxSeriesName(main, secondary),
		Precision: defaultPrecision,
		Retention: defaultRetention,
		Fields:    []timeseries.Field{{Name: fxField, Aggregate: timeseries.AggLast}},
	}
}

// fxID encodes a record time as a synthetic id (UNIX seconds). Daily EOD records are at
// midnight UTC, so this round-trips exactly.
func fxID(t time.Time) uint { return uint(t.UTC().Unix()) } //nolint:gosec // G115: positive epoch seconds for daily records fit comfortably in uint

func fxTime(id uint) time.Time { return time.Unix(int64(id), 0).UTC() } //nolint:gosec // G115: synthetic ids are bounded epoch seconds, no overflow

// RateRecord is a stored exchange rate data point.
type RateRecord struct {
	ID        uint
	Main      string
	Secondary string
	Time      time.Time
	Rate      float64
}

// RatePoint is a single rate observation at a point in time.
type RatePoint struct {
	Time time.Time
	Rate float64
}

// RateUpdate holds optional fields for a partial rate update.
type RateUpdate struct {
	Time *time.Time
	Rate *float64
}

// RegisterPair creates or updates a time series for the given currency pair (main/secondary).
func (s *Store) RegisterPair(ctx context.Context, main, secondary string) error {
	if main == "" || secondary == "" {
		return fmt.Errorf("main and secondary currency cannot be empty")
	}
	if err := s.store.DefineSeries(ctx, fxSeries(main, secondary)); err != nil {
		return fmt.Errorf("failed to define series for %s/%s: %w", main, secondary, err)
	}
	return nil
}

// ListFXPairs returns pairs that have a registered FX series (format "MAIN/SECONDARY").
func (s *Store) ListFXPairs(ctx context.Context) ([]string, error) {
	all, err := s.store.ListSeries(ctx)
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

// IngestRate records a single exchange rate for the pair (main/secondary). Series is auto-registered if needed.
func (s *Store) IngestRate(ctx context.Context, main, secondary string, t time.Time, rate float64) error {
	if main == "" || secondary == "" {
		return fmt.Errorf("main and secondary currency cannot be empty")
	}
	if err := s.RegisterPair(ctx, main, secondary); err != nil {
		return err
	}
	return s.store.Write(ctx, fxSeriesName(main, secondary), timeseries.Point{Time: t, Values: map[string]float64{fxField: rate}})
}

// IngestRatesBulk records multiple rate points for the given pair in one operation.
func (s *Store) IngestRatesBulk(ctx context.Context, main, secondary string, points []RatePoint) error {
	if main == "" || secondary == "" {
		return fmt.Errorf("main and secondary currency cannot be empty")
	}
	if len(points) == 0 {
		return nil
	}
	if err := s.RegisterPair(ctx, main, secondary); err != nil {
		return err
	}
	pts := make([]timeseries.Point, len(points))
	for i, p := range points {
		pts[i] = timeseries.Point{Time: p.Time, Values: map[string]float64{fxField: p.Rate}}
	}
	return s.store.WriteMany(ctx, fxSeriesName(main, secondary), pts)
}

// RateHistory returns rate records for the pair within a time range.
// Zero time values mean unbounded. Returns nil slice when series does not exist or has no data.
func (s *Store) RateHistory(ctx context.Context, main, secondary string, start, end time.Time) ([]RateRecord, error) {
	if main == "" || secondary == "" {
		return nil, fmt.Errorf("main and secondary currency cannot be empty")
	}
	samples, err := s.store.FieldRange(ctx, fxSeriesName(main, secondary), fxField, start, end)
	if err != nil {
		if errors.Is(err, timeseries.ErrSeriesNotFound) || errors.Is(err, timeseries.ErrFieldNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list rates for %s/%s: %w", main, secondary, err)
	}
	out := make([]RateRecord, len(samples))
	for i, sm := range samples {
		out[i] = RateRecord{ID: fxID(sm.Time), Main: main, Secondary: secondary, Time: sm.Time, Rate: sm.Value}
	}
	return out, nil
}

// RateAt returns the most recent rate record for the pair at or before time t. Returns nil if no data.
func (s *Store) RateAt(ctx context.Context, main, secondary string, t time.Time) (*RateRecord, error) {
	if main == "" || secondary == "" {
		return nil, fmt.Errorf("main and secondary currency cannot be empty")
	}
	v, ok, err := s.store.FieldAt(ctx, fxSeriesName(main, secondary), fxField, t)
	if err != nil {
		if errors.Is(err, timeseries.ErrSeriesNotFound) || errors.Is(err, timeseries.ErrFieldNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get rate for %s/%s at %s: %w", main, secondary, t.Format(time.DateOnly), err)
	}
	if !ok {
		return nil, nil
	}
	return &RateRecord{ID: fxID(t), Main: main, Secondary: secondary, Time: t, Rate: v}, nil
}

// LatestRate returns the most recent rate record for the pair. Returns nil if no data.
func (s *Store) LatestRate(ctx context.Context, main, secondary string) (*RateRecord, error) {
	if main == "" || secondary == "" {
		return nil, fmt.Errorf("main and secondary currency cannot be empty")
	}
	samples, err := s.store.FieldRange(ctx, fxSeriesName(main, secondary), fxField, time.Time{}, time.Time{})
	if err != nil {
		if errors.Is(err, timeseries.ErrSeriesNotFound) || errors.Is(err, timeseries.ErrFieldNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest rate for %s/%s: %w", main, secondary, err)
	}
	if len(samples) == 0 {
		return nil, nil
	}
	last := samples[len(samples)-1]
	return &RateRecord{ID: fxID(last.Time), Main: main, Secondary: secondary, Time: last.Time, Rate: last.Value}, nil
}

// UpdateRate applies a partial update to the rate record identified by (main/secondary, id),
// where id is the synthetic time-derived id from RateRecord.ID.
func (s *Store) UpdateRate(ctx context.Context, main, secondary string, id uint, in RateUpdate) error {
	if main == "" || secondary == "" {
		return fmt.Errorf("main and secondary currency cannot be empty")
	}
	if id == 0 {
		return fmt.Errorf("record id is required for update")
	}
	name := fxSeriesName(main, secondary)
	oldTime := fxTime(id)
	cur, ok, err := s.store.FieldAt(ctx, name, fxField, oldTime)
	if err != nil {
		if errors.Is(err, timeseries.ErrSeriesNotFound) || errors.Is(err, timeseries.ErrFieldNotFound) {
			return fmt.Errorf("rate record %d not found for %s/%s", id, main, secondary)
		}
		return err
	}
	if !ok {
		return fmt.Errorf("rate record %d not found for %s/%s", id, main, secondary)
	}
	rate := cur
	if in.Rate != nil {
		rate = *in.Rate
	}
	newTime := oldTime
	if in.Time != nil {
		newTime = *in.Time
	}
	if !newTime.Equal(oldTime) {
		if err := s.store.Delete(ctx, name, oldTime); err != nil {
			return err
		}
	}
	return s.IngestRate(ctx, main, secondary, newTime, rate)
}

// DeleteRate removes the rate record identified by (main/secondary, id).
func (s *Store) DeleteRate(ctx context.Context, main, secondary string, id uint) error {
	if main == "" || secondary == "" {
		return fmt.Errorf("main and secondary currency cannot be empty")
	}
	if id == 0 {
		return fmt.Errorf("record id is required for delete")
	}
	return s.store.Delete(ctx, fxSeriesName(main, secondary), fxTime(id))
}
