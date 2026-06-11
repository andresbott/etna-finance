package tasks

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/andresbott/etna/internal/marketdata"
	"github.com/andresbott/etna/internal/marketdata/importer"
	"github.com/go-bumbu/testdbs"
	"golang.org/x/text/currency"
)

func TestMain(m *testing.M) {
	testdbs.InitDBS()
	code := m.Run()
	_ = testdbs.Clean()
	os.Exit(code)
}

type fakeEPSClient struct {
	bySymbol map[string][]importer.EPSPoint
	errBy    map[string]error
}

func (f *fakeEPSClient) FetchEPS(_ context.Context, symbol string) ([]importer.EPSPoint, error) {
	if e := f.errBy[symbol]; e != nil {
		return nil, e
	}
	return f.bySymbol[symbol], nil
}

func TestEPSImportTask(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, err := marketdata.NewStore(db.ConnDbName("TestEPSImportTask"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.CreateInstrument(ctx, marketdata.Instrument{Symbol: "AAA", Name: "Alpha", Currency: currency.USD, Type: "Stock"}); err != nil {
				t.Fatal(err)
			}
			if _, err := store.CreateInstrument(ctx, marketdata.Instrument{Symbol: "BBB", Name: "Beta", Currency: currency.USD, Type: "Stock"}); err != nil {
				t.Fatal(err)
			}
			// VOO is an ETF: it must be skipped before any fetch, even though the client would
			// return data for it. EPS is stock-only.
			if _, err := store.CreateInstrument(ctx, marketdata.Instrument{Symbol: "VOO", Name: "Vanguard S&P 500", Currency: currency.USD, Type: "ETF"}); err != nil {
				t.Fatal(err)
			}

			now := time.Now().UTC()
			client := &fakeEPSClient{
				bySymbol: map[string][]importer.EPSPoint{
					"AAA": {
						{Time: now.AddDate(0, -9, 0), Basic: 1.0, Diluted: 0.9},
						{Time: now.AddDate(0, -6, 0), Basic: 1.1, Diluted: 1.0},
						{Time: now.AddDate(0, -3, 0), Basic: 1.2, Diluted: 1.1},
					},
					"VOO": {
						{Time: now.AddDate(0, -3, 0), Basic: 5.0, Diluted: 5.0},
					},
				},
				errBy: map[string]error{"BBB": errors.New("no financials found for BBB")},
			}

			taskFn := NewEPSImportTaskFn(store, client)
			if err := taskFn(ctx); err != nil {
				t.Fatalf("task returned error (should skip failures, not abort): %v", err)
			}

			recs, err := store.EPSHistory(ctx, "AAA", time.Time{}, time.Time{})
			if err != nil {
				t.Fatal(err)
			}
			if len(recs) != 3 {
				t.Fatalf("expected 3 EPS records for AAA, got %d", len(recs))
			}
			// BBB errored: no series, history is nil.
			bbb, _ := store.EPSHistory(ctx, "BBB", time.Time{}, time.Time{})
			if len(bbb) != 0 {
				t.Errorf("expected no EPS for BBB, got %d", len(bbb))
			}
			// VOO is an ETF: skipped before fetch despite the client having data for it.
			voo, _ := store.EPSHistory(ctx, "VOO", time.Time{}, time.Time{})
			if len(voo) != 0 {
				t.Errorf("expected no EPS for VOO (ETF should be skipped), got %d", len(voo))
			}

			// Idempotent: a second run dedups and writes nothing new.
			if err := taskFn(ctx); err != nil {
				t.Fatalf("second run errored: %v", err)
			}
			recs2, _ := store.EPSHistory(ctx, "AAA", time.Time{}, time.Time{})
			if len(recs2) != 3 {
				t.Errorf("expected 3 records after re-run (dedup), got %d", len(recs2))
			}
		})
	}
}

func TestNewEPSImportTaskFn_requiresStoreAndClient(t *testing.T) {
	if err := NewEPSImportTaskFn(nil, &fakeEPSClient{})(context.Background()); err == nil {
		t.Error("expected error when store is nil")
	}

	for _, db := range testdbs.DBs() {
		t.Run("nil-client/"+db.DbType(), func(t *testing.T) {
			store, err := marketdata.NewStore(db.ConnDbName("TestNewEPSImportTaskFn_nilClient"))
			if err != nil {
				t.Fatal(err)
			}
			if err := NewEPSImportTaskFn(store, nil)(context.Background()); err == nil {
				t.Error("expected error when client is nil")
			}
		})
	}
}
