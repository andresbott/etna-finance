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

// RateRecord is a stored exchange rate data point. Like price records, it is
// addressed by Time (no synthetic id): the daily series holds at most one record
// per timestamp, so Time is a stable identifier for edits and deletes.
type RateRecord struct {
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

// IngestRate records a single exchange rate for the pair (main/secondary). The pair's series must
// already exist (created via RegisterPair); this does not auto-register it.
func (s *Store) IngestRate(ctx context.Context, main, secondary string, t time.Time, rate float64) error {
	if main == "" || secondary == "" {
		return fmt.Errorf("main and secondary currency cannot be empty")
	}
	return s.store.Write(ctx, fxSeriesName(main, secondary), timeseries.Point{Time: t, Values: map[string]float64{fxField: rate}})
}

// IngestRatesBulk records multiple rate points for the given pair in one operation. The pair's
// series must already exist (created via RegisterPair); this does not auto-register it.
func (s *Store) IngestRatesBulk(ctx context.Context, main, secondary string, points []RatePoint) error {
	if main == "" || secondary == "" {
		return fmt.Errorf("main and secondary currency cannot be empty")
	}
	if len(points) == 0 {
		return nil
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
		out[i] = RateRecord{Main: main, Secondary: secondary, Time: sm.Time, Rate: sm.Value}
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
	return &RateRecord{Main: main, Secondary: secondary, Time: t, Rate: v}, nil
}

// LatestRate returns the most recent rate record for the pair. Returns nil if no data.
func (s *Store) LatestRate(ctx context.Context, main, secondary string) (*RateRecord, error) {
	if main == "" || secondary == "" {
		return nil, fmt.Errorf("main and secondary currency cannot be empty")
	}
	last, found, err := s.store.LatestField(ctx, fxSeriesName(main, secondary), fxField)
	if err != nil {
		if errors.Is(err, timeseries.ErrSeriesNotFound) || errors.Is(err, timeseries.ErrFieldNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest rate for %s/%s: %w", main, secondary, err)
	}
	if !found {
		return nil, nil
	}
	return &RateRecord{Main: main, Secondary: secondary, Time: last.Time, Rate: last.Value}, nil
}

// EditRate overwrites the rate for the pair at p.Time. If p.Time differs from oldTime, the old
// timestamp is removed as part of the same write (an atomic move via the store). A zero oldTime
// means "no prior timestamp to remove" and behaves as a plain write. Mirrors EditPrice.
func (s *Store) EditRate(ctx context.Context, main, secondary string, oldTime time.Time, p RatePoint) error {
	if main == "" || secondary == "" {
		return fmt.Errorf("main and secondary currency cannot be empty")
	}
	// Same time (or no prior time): a plain upsert, no move needed.
	if oldTime.IsZero() || oldTime.Equal(p.Time) {
		return s.IngestRate(ctx, main, secondary, p.Time, p.Rate)
	}
	// A move implies a record (and therefore the series) already exists, so no register needed.
	if err := s.store.Move(ctx, fxSeriesName(main, secondary), oldTime, timeseries.Point{Time: p.Time, Values: map[string]float64{fxField: p.Rate}}); err != nil {
		return fmt.Errorf("failed to move rate record for %s/%s: %w", main, secondary, err)
	}
	return nil
}

// DeleteRateAt removes the rate record for the pair at exactly t. Mirrors DeletePriceAt.
func (s *Store) DeleteRateAt(ctx context.Context, main, secondary string, t time.Time) error {
	if main == "" || secondary == "" {
		return fmt.Errorf("main and secondary currency cannot be empty")
	}
	if err := s.store.Delete(ctx, fxSeriesName(main, secondary), t); err != nil {
		return fmt.Errorf("failed to delete rate for %s/%s: %w", main, secondary, err)
	}
	return nil
}
