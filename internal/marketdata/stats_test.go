package marketdata

import (
	"testing"
	"time"

	"github.com/go-bumbu/testdbs"
)

func TestStats(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("TestStats")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			// Empty store: all counts zero.
			got, err := store.Stats(ctx)
			if err != nil {
				t.Fatalf("Stats on empty store: %v", err)
			}
			if (got != DataStats{}) {
				t.Fatalf("expected zero stats on empty store, got %+v", got)
			}

			// Two price series: AAPL with 3 candles, MSFT with 2 candles.
			day := func(d int) time.Time { return time.Date(2025, 1, d, 0, 0, 0, 0, time.UTC) }
			if err := store.IngestPricesBulk(ctx, "AAPL", []PricePoint{
				{Time: day(1), Close: 1}, {Time: day(2), Close: 2}, {Time: day(3), Close: 3},
			}); err != nil {
				t.Fatalf("ingest AAPL: %v", err)
			}
			if err := store.IngestPricesBulk(ctx, "MSFT", []PricePoint{
				{Time: day(1), Close: 10}, {Time: day(2), Close: 20},
			}); err != nil {
				t.Fatalf("ingest MSFT: %v", err)
			}

			// One FX pair with 2 rates.
			if err := store.IngestRatesBulk(ctx, "EUR", "USD", []RatePoint{
				{Time: day(1), Rate: 1.08}, {Time: day(2), Rate: 1.09},
			}); err != nil {
				t.Fatalf("ingest EUR/USD: %v", err)
			}

			got, err = store.Stats(ctx)
			if err != nil {
				t.Fatalf("Stats: %v", err)
			}
			want := DataStats{PriceSeries: 2, PricePoints: 5, FXSeries: 1, FXPoints: 2}
			if got != want {
				t.Fatalf("Stats mismatch: got %+v, want %+v", got, want)
			}
		})
	}
}
