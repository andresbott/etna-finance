package marketdata

import (
	"testing"
	"time"

	"github.com/go-bumbu/testdbs"
	"golang.org/x/text/currency"
)

func TestWipeData(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("TestWipeData")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			// Create an instrument
			_, err = store.CreateInstrument(ctx, Instrument{Symbol: "AAPL", Name: "Apple Inc.", Currency: currency.USD})
			if err != nil {
				t.Fatalf("create instrument: %v", err)
			}

			// Ingest a price
			err = store.IngestPrice(ctx, "AAPL", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 150.0)
			if err != nil {
				t.Fatalf("ingest price: %v", err)
			}

			// Ingest an FX rate
			err = store.IngestRate(ctx, "EUR", "USD", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 1.08)
			if err != nil {
				t.Fatalf("ingest rate: %v", err)
			}

			// Verify data exists before wipe
			instruments, err := store.ListInstruments(ctx)
			if err != nil {
				t.Fatalf("list instruments before wipe: %v", err)
			}
			if len(instruments) == 0 {
				t.Fatal("expected at least one instrument before wipe")
			}

			symbols, err := store.ListPriceSymbols()
			if err != nil {
				t.Fatalf("list price symbols before wipe: %v", err)
			}
			if len(symbols) == 0 {
				t.Fatal("expected at least one price symbol before wipe")
			}

			pairs, err := store.ListFXPairs()
			if err != nil {
				t.Fatalf("list FX pairs before wipe: %v", err)
			}
			if len(pairs) == 0 {
				t.Fatal("expected at least one FX pair before wipe")
			}

			// Wipe all data
			err = store.WipeData(ctx)
			if err != nil {
				t.Fatalf("WipeData: %v", err)
			}

			// Assert instruments list is empty
			instruments, err = store.ListInstruments(ctx)
			if err != nil {
				t.Fatalf("list instruments after wipe: %v", err)
			}
			if len(instruments) != 0 {
				t.Errorf("expected 0 instruments after wipe, got %d", len(instruments))
			}

			// Assert price symbols list is empty
			symbols, err = store.ListPriceSymbols()
			if err != nil {
				t.Fatalf("list price symbols after wipe: %v", err)
			}
			if len(symbols) != 0 {
				t.Errorf("expected 0 price symbols after wipe, got %d", len(symbols))
			}

			// Assert FX pairs list is empty
			pairs, err = store.ListFXPairs()
			if err != nil {
				t.Fatalf("list FX pairs after wipe: %v", err)
			}
			if len(pairs) != 0 {
				t.Errorf("expected 0 FX pairs after wipe, got %d", len(pairs))
			}
		})
	}
}
