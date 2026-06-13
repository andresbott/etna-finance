package marketdata

import (
	"context"
	"fmt"

	"github.com/go-bumbu/timeseries"
)

// DataStats summarizes the volume of stored market data and FX rates.
type DataStats struct {
	PriceSeries int // number of instruments with a price series
	PricePoints int // total OHLCV candles across all price series
	FXSeries    int // number of currency pairs with an FX series
	FXPoints    int // total rate observations across all FX series
}

// Stats counts the price and FX series and their data points across all series.
// Point counts come from the store's server-side Count (distinct timestamps per
// series), so no row data is transferred.
func (s *Store) Stats(ctx context.Context) (DataStats, error) {
	priceSeries, err := s.store.ListSeries(ctx, timeseries.MatchLabel(labelType, typePrice))
	if err != nil {
		return DataStats{}, fmt.Errorf("failed to list price series: %w", err)
	}
	fxSeries, err := s.store.ListSeries(ctx, timeseries.MatchLabel(labelType, typeFX))
	if err != nil {
		return DataStats{}, fmt.Errorf("failed to list fx series: %w", err)
	}
	var stats DataStats
	for _, ts := range priceSeries {
		n, err := s.store.Count(ctx, ts.Name)
		if err != nil {
			return DataStats{}, fmt.Errorf("failed to count price points for %q: %w", ts.Name, err)
		}
		stats.PriceSeries++
		stats.PricePoints += n
	}
	for _, ts := range fxSeries {
		n, err := s.store.Count(ctx, ts.Name)
		if err != nil {
			return DataStats{}, fmt.Errorf("failed to count fx points for %q: %w", ts.Name, err)
		}
		stats.FXSeries++
		stats.FXPoints += n
	}
	return stats, nil
}
