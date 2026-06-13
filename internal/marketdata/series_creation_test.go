package marketdata

import (
	"context"
	"testing"
	"time"

	"github.com/go-bumbu/testdbs"
	"github.com/go-bumbu/timeseries"
	"golang.org/x/text/currency"
)

// hasEPSSeries reports whether the eps: series for symbol is defined in the store.
func hasEPSSeries(t *testing.T, ctx context.Context, store *Store, symbol string) bool {
	t.Helper()
	all, err := store.store.ListSeries(ctx)
	if err != nil {
		t.Fatalf("ListSeries: %v", err)
	}
	for _, ts := range all {
		if ts.Name == epsSeriesName(symbol) {
			return true
		}
	}
	return false
}

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

// TestCreateInstrument_DefinesEPSSeriesForStock asserts the eps: series is defined at creation only
// for stock-type instruments (EPS applies to stocks), so EPS ingest no longer needs to auto-register.
func TestCreateInstrument_DefinesEPSSeriesForStock(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, err := NewStore(db.ConnDbName("TestCreateInstrument_DefinesEPSSeriesForStock"))
			if err != nil {
				t.Fatal(err)
			}

			if _, err := store.CreateInstrument(ctx, Instrument{Symbol: "STK", Currency: currency.USD, Type: "stock"}); err != nil {
				t.Fatalf("CreateInstrument stock: %v", err)
			}
			if !hasEPSSeries(t, ctx, store, "STK") {
				t.Fatal("expected EPS series for stock instrument STK right after CreateInstrument")
			}

			if _, err := store.CreateInstrument(ctx, Instrument{Symbol: "FND", Currency: currency.USD, Type: "etf"}); err != nil {
				t.Fatalf("CreateInstrument etf: %v", err)
			}
			if hasEPSSeries(t, ctx, store, "FND") {
				t.Fatal("did not expect an EPS series for non-stock instrument FND")
			}
		})
	}
}

// TestUpdateInstrument_DefinesEPSSeriesOnTypeStock asserts that switching an instrument's type to
// stock defines its eps: series, so a later EPS ingest does not fail with ErrSeriesNotFound.
func TestUpdateInstrument_DefinesEPSSeriesOnTypeStock(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, err := NewStore(db.ConnDbName("TestUpdateInstrument_DefinesEPSSeriesOnTypeStock"))
			if err != nil {
				t.Fatal(err)
			}

			id, err := store.CreateInstrument(ctx, Instrument{Symbol: "MUT", Currency: currency.USD, Type: "etf"})
			if err != nil {
				t.Fatalf("CreateInstrument: %v", err)
			}
			if hasEPSSeries(t, ctx, store, "MUT") {
				t.Fatal("precondition: a non-stock instrument should not have an EPS series")
			}

			stock := "stock"
			if err := store.UpdateInstrument(ctx, id, InstrumentUpdatePayload{Type: &stock}); err != nil {
				t.Fatalf("UpdateInstrument: %v", err)
			}
			if !hasEPSSeries(t, ctx, store, "MUT") {
				t.Fatal("expected an EPS series after switching the instrument type to stock")
			}
		})
	}
}

// TestIngestEPS_RequiresExistingSeries asserts EPS ingest no longer auto-registers: writing EPS for a
// symbol with no series (no stock instrument created) fails instead of creating an orphan series.
func TestIngestEPS_RequiresExistingSeries(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, err := NewStore(db.ConnDbName("TestIngestEPS_RequiresExistingSeries"))
			if err != nil {
				t.Fatal(err)
			}

			if err := store.IngestEPS(ctx, "GHOST", EPSPoint{Time: time.Now(), Basic: 1.0}); err == nil {
				t.Fatal("expected error ingesting EPS for a symbol with no series, got nil")
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

// TestRegisteredSeriesCarryLabels asserts price/eps/fx series are labeled.
func TestRegisteredSeriesCarryLabels(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			store, err := NewStore(db.ConnDbName("TestRegisteredSeriesCarryLabels"))
			if err != nil {
				t.Fatal(err)
			}
			price := ohlcvSeries("AAPL")
			if price.Labels[labelType] != typePrice || price.Labels[labelSymbol] != "AAPL" {
				t.Fatalf("price labels wrong: %+v", price.Labels)
			}
			eps := epsSeries("AAPL")
			if eps.Labels[labelType] != typeEPS || eps.Labels[labelSymbol] != "AAPL" {
				t.Fatalf("eps labels wrong: %+v", eps.Labels)
			}
			fx := fxSeries("EUR", "USD")
			if fx.Labels[labelType] != typeFX || fx.Labels[labelMain] != "EUR" || fx.Labels[labelSecondary] != "USD" {
				t.Fatalf("fx labels wrong: %+v", fx.Labels)
			}
			_ = store
		})
	}
}

// TestBackfillSeriesLabels asserts pre-existing unlabeled series get labels on
// startup, and that a second startup is a no-op.
func TestBackfillSeriesLabels(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("TestBackfillSeriesLabels")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}
			// Simulate a legacy FX series with no labels by defining one directly
			// through the timeseries store, bypassing RegisterPair's labels.
			legacy := timeseries.Series{
				Name:      fxSeriesName("GBP", "JPY"),
				Precision: defaultPrecision,
				Retention: defaultRetention,
				Fields:    []timeseries.Field{{Name: fxField, Aggregate: timeseries.AggLast}},
			}
			if err := store.store.DefineSeries(ctx, legacy); err != nil {
				t.Fatalf("define legacy: %v", err)
			}

			// Run the backfill (also runs implicitly via NewStore; call directly to assert).
			if err := store.backfillSeriesLabels(ctx); err != nil {
				t.Fatalf("backfill: %v", err)
			}
			got, err := store.store.GetSeries(ctx, fxSeriesName("GBP", "JPY"))
			if err != nil {
				t.Fatal(err)
			}
			if got.Labels[labelType] != typeFX || got.Labels[labelMain] != "GBP" || got.Labels[labelSecondary] != "JPY" {
				t.Fatalf("backfill labels wrong: %+v", got.Labels)
			}

			// Second run is a no-op (still labeled, no error).
			if err := store.backfillSeriesLabels(ctx); err != nil {
				t.Fatalf("second backfill: %v", err)
			}
			pairs, err := store.ListFXPairsDetailed(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if len(pairs) != 1 || pairs[0] != (FXPair{Main: "GBP", Secondary: "JPY"}) {
				t.Fatalf("got %+v, want [{GBP JPY}]", pairs)
			}
		})
	}
}
