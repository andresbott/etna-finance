package marketdata

import (
	"context"
	"errors"
	"fmt"
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
		Labels:    map[string]string{labelType: typeFX, labelMain: main, labelSecondary: secondary},
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

// FXPair identifies a currency pair (main/secondary).
type FXPair struct {
	Main      string
	Secondary string
}

// ListFXPairsDetailed returns registered FX pairs as structured values, sourced
// from series labels (no name parsing).
func (s *Store) ListFXPairsDetailed(ctx context.Context) ([]FXPair, error) {
	all, err := s.store.ListSeries(ctx, timeseries.MatchLabel(labelType, typeFX))
	if err != nil {
		return nil, fmt.Errorf("failed to list series: %w", err)
	}
	var pairs []FXPair
	for _, ts := range all {
		main, secondary := ts.Labels[labelMain], ts.Labels[labelSecondary]
		if main == "" || secondary == "" {
			continue
		}
		pairs = append(pairs, FXPair{Main: main, Secondary: secondary})
	}
	return pairs, nil
}

// ListFXPairs returns pairs that have a registered FX series (format "MAIN/SECONDARY").
func (s *Store) ListFXPairs(ctx context.Context) ([]string, error) {
	pairs, err := s.ListFXPairsDetailed(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(pairs))
	for i, p := range pairs {
		out[i] = p.Main + "/" + p.Secondary
	}
	return out, nil
}

// IngestRate records a single exchange rate for the pair (main/secondary). The pair's series must
// already exist (created via RegisterPair); this does not auto-register it.
func (s *Store) IngestRate(ctx context.Context, main, secondary string, t time.Time, rate float64) error {
	if main == "" || secondary == "" {
		return fmt.Errorf("main and secondary currency cannot be empty")
	}
	return s.store.Write(ctx, fxSeriesName(main, secondary), timeseries.Point{Time: t.UTC(), Values: map[string]float64{fxField: rate}})
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
		pts[i] = timeseries.Point{Time: p.Time.UTC(), Values: map[string]float64{fxField: p.Rate}}
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

// EditRate overwrites the rate for the pair at p.Time. The date is the record's identity and
// cannot be changed: if a non-zero oldTime differs from p.Time, EditRate returns ErrDateImmutable
// and changes nothing. A zero oldTime, or oldTime equal to p.Time, is a plain upsert. Mirrors
// EditPrice.
func (s *Store) EditRate(ctx context.Context, main, secondary string, oldTime time.Time, p RatePoint) error {
	if main == "" || secondary == "" {
		return fmt.Errorf("main and secondary currency cannot be empty")
	}
	if !oldTime.IsZero() && !oldTime.Equal(p.Time) {
		return fmt.Errorf("cannot edit rate for %s/%s: %w", main, secondary, ErrDateImmutable)
	}
	return s.IngestRate(ctx, main, secondary, p.Time, p.Rate)
}

// DeleteRateAt removes the rate record for the pair at exactly t. Mirrors DeletePriceAt.
func (s *Store) DeleteRateAt(ctx context.Context, main, secondary string, t time.Time) error {
	if main == "" || secondary == "" {
		return fmt.Errorf("main and secondary currency cannot be empty")
	}
	deleted, err := s.store.Delete(ctx, fxSeriesName(main, secondary), t)
	if err != nil {
		return fmt.Errorf("failed to delete rate for %s/%s: %w", main, secondary, err)
	}
	if !deleted {
		return fmt.Errorf("no rate for %s/%s at %s: %w", main, secondary, t.Format(time.DateOnly), ErrRecordNotFound)
	}
	return nil
}
