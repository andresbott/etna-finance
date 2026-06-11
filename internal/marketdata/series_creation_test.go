package marketdata

import (
	"testing"
	"time"

	"github.com/go-bumbu/testdbs"
	"golang.org/x/text/currency"
)

// TestCreateInstrument_DefinesPriceSeries asserts that creating an instrument defines its OHLCV
// series immediately, so the price series exists before any price is ingested.
func TestCreateInstrument_DefinesPriceSeries(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, err := NewStore(db.ConnDbName("TestCreateInstrument_DefinesPriceSeries"))
			if err != nil {
				t.Fatal(err)
			}

			if _, err := store.CreateInstrument(ctx, Instrument{Symbol: "NEW", Name: "New Co", Currency: currency.USD}); err != nil {
				t.Fatalf("CreateInstrument: %v", err)
			}

			symbols, err := store.ListPriceSymbols(ctx)
			if err != nil {
				t.Fatalf("ListPriceSymbols: %v", err)
			}
			found := false
			for _, s := range symbols {
				if s == "NEW" {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected price series for NEW to exist right after CreateInstrument, got %v", symbols)
			}
		})
	}
}

// TestIngestPrice_RequiresExistingSeries asserts that ingesting prices no longer auto-registers the
// series: writing for an unknown symbol (no instrument created) fails instead of silently creating
// an orphan series.
func TestIngestPrice_RequiresExistingSeries(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, err := NewStore(db.ConnDbName("TestIngestPrice_RequiresExistingSeries"))
			if err != nil {
				t.Fatal(err)
			}

			err = store.IngestPrice(ctx, "GHOST", PricePoint{Time: time.Now(), Close: 1.0})
			if err == nil {
				t.Fatal("expected error ingesting price for a symbol with no instrument/series, got nil")
			}
		})
	}
}

// TestIngestRate_RequiresRegisteredPair asserts that ingesting FX rates no longer auto-registers the
// pair: writing for an unregistered pair fails. Pairs must be created explicitly via RegisterPair.
func TestIngestRate_RequiresRegisteredPair(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, err := NewStore(db.ConnDbName("TestIngestRate_RequiresRegisteredPair"))
			if err != nil {
				t.Fatal(err)
			}

			err = store.IngestRate(ctx, "EUR", "USD", time.Now(), 1.1)
			if err == nil {
				t.Fatal("expected error ingesting rate for an unregistered pair, got nil")
			}

			// After explicit registration the same ingest succeeds.
			if err := store.RegisterPair(ctx, "EUR", "USD"); err != nil {
				t.Fatalf("RegisterPair: %v", err)
			}
			if err := store.IngestRate(ctx, "EUR", "USD", time.Now(), 1.1); err != nil {
				t.Fatalf("IngestRate after RegisterPair: %v", err)
			}
		})
	}
}

// TestNewStore_RegistersSeriesForExistingInstruments asserts the startup migration: an instrument
// row that predates the "define at creation" change (no price series) gets its series defined when
// the store is opened, so later ingests do not fail with ErrSeriesNotFound.
func TestNewStore_RegistersSeriesForExistingInstruments(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("TestNewStore_RegistersSeriesForExistingInstruments")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			// Simulate a legacy instrument: insert the row directly, bypassing CreateInstrument,
			// so no price series exists for it.
			if err := dbCon.WithContext(ctx).Create(&dbInstrument{Symbol: "LEGACY", Currency: "USD"}).Error; err != nil {
				t.Fatalf("insert legacy instrument: %v", err)
			}
			symbols, err := store.ListPriceSymbols(ctx)
			if err != nil {
				t.Fatalf("ListPriceSymbols: %v", err)
			}
			for _, s := range symbols {
				if s == "LEGACY" {
					t.Fatalf("precondition failed: LEGACY already has a series before migration")
				}
			}

			// Reopening the store runs the migration that defines series for existing instruments.
			store2, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}
			symbols, err = store2.ListPriceSymbols(ctx)
			if err != nil {
				t.Fatalf("ListPriceSymbols after reopen: %v", err)
			}
			found := false
			for _, s := range symbols {
				if s == "LEGACY" {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected migration to define a price series for LEGACY, got %v", symbols)
			}
		})
	}
}
