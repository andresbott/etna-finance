package marketdata

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// DataStats summarizes the volume of stored market data and FX rates.
type DataStats struct {
	PriceSeries int // number of instruments with a price series
	PricePoints int // total OHLCV candles across all price series
	FXSeries    int // number of currency pairs with an FX series
	FXPoints    int // total rate observations across all FX series
}

// Stats counts the price and FX series and their data points across all series.
// Counting reads each series' samples for a single field (one sample per stored
// timestamp); fine for this single-user app's data volume.
func (s *Store) Stats(ctx context.Context) (DataStats, error) {
	all, err := s.store.ListSeries(ctx)
	if err != nil {
		return DataStats{}, fmt.Errorf("failed to list series: %w", err)
	}
	var stats DataStats
	for _, ts := range all {
		switch {
		case strings.HasPrefix(ts.Name, seriesPrefix):
			samples, err := s.store.FieldRange(ctx, ts.Name, "close", time.Time{}, time.Time{})
			if err != nil {
				return DataStats{}, fmt.Errorf("failed to count price points for %q: %w", ts.Name, err)
			}
			stats.PriceSeries++
			stats.PricePoints += len(samples)
		case strings.HasPrefix(ts.Name, fxSeriesPrefix):
			samples, err := s.store.FieldRange(ctx, ts.Name, fxField, time.Time{}, time.Time{})
			if err != nil {
				return DataStats{}, fmt.Errorf("failed to count fx points for %q: %w", ts.Name, err)
			}
			stats.FXSeries++
			stats.FXPoints += len(samples)
		}
	}
	return stats, nil
}
