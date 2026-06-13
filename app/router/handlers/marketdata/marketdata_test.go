package marketdata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/andresbott/etna/internal/marketdata"
	"github.com/go-bumbu/testdbs"
	"golang.org/x/text/currency"
)

func TestMain(m *testing.M) {
	testdbs.InitDBS()
	code := m.Run()
	_ = testdbs.Clean()
	os.Exit(code)
}

// An edit whose body time differs from the {date} in the path is an attempt to change a record's
// date. The date is the record's identity, so this must surface a 400, not silently relocate it.
func TestEditEndpoints_DateChangeReturns400(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := context.Background()
			store, err := marketdata.NewStore(db.ConnDbName("HandlerEditDateChange"))
			if err != nil {
				t.Fatal(err)
			}
			// A stock instrument gets both a price and an EPS series at creation.
			if _, err := store.CreateInstrument(ctx, marketdata.Instrument{Symbol: "SYM", Currency: currency.USD, Type: marketdata.StockInstrumentType}); err != nil {
				t.Fatalf("create instrument: %v", err)
			}
			if err := store.RegisterPair(ctx, "EUR", "USD"); err != nil {
				t.Fatalf("register pair: %v", err)
			}
			h := &Handler{Store: store}

			// Each request's body time differs from the path {date}: a date change, which is rejected.
			cases := []struct {
				name    string
				body    string
				handler http.Handler
			}{
				{"price", `{"time":"2025-01-15","close":10}`, h.EditPrice("SYM", "2025-01-01")},
				{"fx", `{"time":"2025-01-15","rate":1.1}`, h.EditFXRate("EUR", "USD", "2025-01-01")},
				{"eps", `{"time":"2025-01-15","eps_basic":1.2}`, h.EditEPS("SYM", "2025-01-01")},
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(tc.body))
					recorder := httptest.NewRecorder()
					tc.handler.ServeHTTP(recorder, req)
					if recorder.Code != http.StatusBadRequest {
						t.Fatalf("status = %d, want 400; body = %s", recorder.Code, recorder.Body.String())
					}
				})
			}
		})
	}
}

// Deleting a record at a timestamp that holds no data must surface a 404, not a 200: the store
// now reports whether anything was removed, so a no-op delete is a not-found rather than a
// silent success.
func TestDeleteEndpoints_MissingRecordReturns404(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := context.Background()
			store, err := marketdata.NewStore(db.ConnDbName("HandlerDeleteMissing"))
			if err != nil {
				t.Fatal(err)
			}
			// Series exist (so the delete reaches the record layer) but hold no record at the date.
			if _, err := store.CreateInstrument(ctx, marketdata.Instrument{Symbol: "SYM", Currency: currency.USD, Type: marketdata.StockInstrumentType}); err != nil {
				t.Fatalf("create instrument: %v", err)
			}
			if err := store.RegisterPair(ctx, "EUR", "USD"); err != nil {
				t.Fatalf("register pair: %v", err)
			}
			h := &Handler{Store: store}

			cases := []struct {
				name    string
				handler http.Handler
			}{
				{"price", h.DeletePrice("SYM", "2025-01-01")},
				{"fx", h.DeleteFXRate("EUR", "USD", "2025-01-01")},
				{"eps", h.DeleteEPS("SYM", "2025-01-01")},
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					req := httptest.NewRequest(http.MethodDelete, "/", nil)
					recorder := httptest.NewRecorder()
					tc.handler.ServeHTTP(recorder, req)
					if recorder.Code != http.StatusNotFound {
						t.Fatalf("status = %d, want 404; body = %s", recorder.Code, recorder.Body.String())
					}
				})
			}
		})
	}
}

// The first EPS point for a symbol whose eps: series was not auto-defined (e.g. a non-stock
// instrument the user annotates manually) must register the series and return 201, not 500.
// Mirrors the FX create handlers, which register the pair before ingest.
func TestCreateEPS_FirstPointRegistersSeries(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := context.Background()
			store, err := marketdata.NewStore(db.ConnDbName("HandlerCreateEPSRegister"))
			if err != nil {
				t.Fatal(err)
			}
			// A non-stock instrument: creation does NOT define an eps: series for it.
			if _, err := store.CreateInstrument(ctx, marketdata.Instrument{Symbol: "BOND", Currency: currency.USD, Type: "bond"}); err != nil {
				t.Fatalf("create instrument: %v", err)
			}
			h := &Handler{Store: store}

			cases := []struct {
				name    string
				body    string
				handler http.Handler
			}{
				{"single", `{"time":"2025-01-15","eps_basic":1.2}`, h.CreateEPS("BOND")},
				{"bulk", `{"points":[{"time":"2025-02-15","eps_basic":1.3}]}`, h.CreateEPSBulk("BOND")},
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
					recorder := httptest.NewRecorder()
					tc.handler.ServeHTTP(recorder, req)
					if recorder.Code != http.StatusCreated {
						t.Fatalf("status = %d, want 201; body = %s", recorder.Code, recorder.Body.String())
					}
				})
			}
		})
	}
}
