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
// Point counts come from the store's server-side CountAll (distinct timestamps
// per series, one GROUP BY per type), so no row data is transferred and there is
// no per-series query fan-out.
func (s *Store) Stats(ctx context.Context) (DataStats, error) {
	// Price series are listed (not just counted) because the symbol-label guard
	// below needs their labels, which the name-keyed CountAll map does not carry.
	priceSeries, err := s.store.ListSeries(ctx, timeseries.MatchLabel(labelType, typePrice))
	if err != nil {
		return DataStats{}, fmt.Errorf("failed to list price series: %w", err)
	}
	priceCounts, err := s.store.CountAll(ctx, timeseries.MatchLabel(labelType, typePrice))
	if err != nil {
		return DataStats{}, fmt.Errorf("failed to count price points: %w", err)
	}
	fxCounts, err := s.store.CountAll(ctx, timeseries.MatchLabel(labelType, typeFX))
	if err != nil {
		return DataStats{}, fmt.Errorf("failed to count fx points: %w", err)
	}
	var stats DataStats
	for _, ts := range priceSeries {
		// Skip price series with an empty symbol label, matching ListPriceSymbols,
		// so both screens agree on the instrument count if a label ever regresses.
		if ts.Labels[labelSymbol] == "" {
			continue
		}
		stats.PriceSeries++
		stats.PricePoints += priceCounts[ts.Name]
	}
	for _, n := range fxCounts {
		stats.FXSeries++
		stats.FXPoints += n
	}
	return stats, nil
}
