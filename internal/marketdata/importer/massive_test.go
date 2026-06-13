package importer

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// massiveClientForServer returns a MassiveClient whose shared HTTP client is pointed at srv.
// Retries are disabled so error-path tests fail fast instead of backing off.
func massiveClientForServer(srv *httptest.Server) *MassiveClient {
	c := NewMassiveClient("test-key")
	c.rest.HTTP.SetBaseURL(srv.URL)
	c.rest.HTTP.SetRetryCount(0)
	return c
}

func TestMassiveClient_FetchDailyPrices(t *testing.T) {
	ts := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	t.Run("returns candles and skips entries without a timestamp", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/v2/aggs/ticker/") {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			// Second result has no "t": it must be skipped (zero timestamp).
			writeJSON(w, `{"results":[`+
				`{"t":`+millis(ts)+`,"o":98,"h":101,"l":97,"c":100,"v":1000},`+
				`{"o":1,"c":2}`+
				`]}`)
		}))
		defer srv.Close()

		got, err := massiveClientForServer(srv).FetchDailyPrices(context.Background(), "AAPL", ts, ts.AddDate(0, 0, 1))
		if err != nil {
			t.Fatalf("FetchDailyPrices: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d points, want 1 (zero-timestamp entry skipped)", len(got))
		}
		p := got[0]
		if !p.Time.Equal(ts) || p.Open != 98 || p.High != 101 || p.Low != 97 || p.Close != 100 || p.Volume != 1000 {
			t.Errorf("unexpected point: %+v", p)
		}
	})

	t.Run("propagates API errors", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"status":"ERROR"}`, http.StatusInternalServerError)
		}))
		defer srv.Close()

		if _, err := massiveClientForServer(srv).FetchDailyPrices(context.Background(), "AAPL", ts, ts); err == nil {
			t.Fatal("expected error from 500 response")
		}
	})
}

func TestMassiveClient_FetchDailyRates(t *testing.T) {
	ts := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	t.Run("requires both currencies", func(t *testing.T) {
		c := NewMassiveClient("test-key")
		if _, err := c.FetchDailyRates(context.Background(), "", "USD", ts, ts); err == nil {
			t.Error("expected error when main currency is empty")
		}
		if _, err := c.FetchDailyRates(context.Background(), "CHF", "", ts, ts); err == nil {
			t.Error("expected error when secondary currency is empty")
		}
	})

	t.Run("returns close as the rate for the forex ticker", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.URL.Path, "C:CHFUSD") {
				t.Errorf("expected forex ticker C:CHFUSD in path, got %s", r.URL.Path)
			}
			writeJSON(w, `{"results":[{"t":`+millis(ts)+`,"c":1.12},{"c":9.9}]}`)
		}))
		defer srv.Close()

		got, err := massiveClientForServer(srv).FetchDailyRates(context.Background(), "chf", "usd", ts, ts.AddDate(0, 0, 1))
		if err != nil {
			t.Fatalf("FetchDailyRates: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d rates, want 1 (zero-timestamp entry skipped)", len(got))
		}
		if !got[0].Time.Equal(ts) || got[0].Rate != 1.12 {
			t.Errorf("unexpected rate: %+v", got[0])
		}
	})
}

func TestMassiveClient_GetTickerDetails(t *testing.T) {
	t.Run("requires a symbol", func(t *testing.T) {
		c := NewMassiveClient("test-key")
		if _, err := c.GetTickerDetails(context.Background(), ""); err == nil {
			t.Error("expected error when symbol is empty")
		}
	})

	t.Run("maps results and prefixes notes with the exchange", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.URL.Path, "/v3/reference/tickers/AAPL") {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			writeJSON(w, `{"results":{"name":"Apple Inc.","currency_name":"usd","type":"CS","primary_exchange":"XNAS","description":"Maker of phones."}}`)
		}))
		defer srv.Close()

		got, err := massiveClientForServer(srv).GetTickerDetails(context.Background(), "aapl")
		if err != nil {
			t.Fatalf("GetTickerDetails: %v", err)
		}
		want := TickerDetails{
			Name:     "Apple Inc.",
			Currency: "USD",
			Type:     "CS",
			Exchange: "XNAS",
			Notes:    "XNAS — Maker of phones.",
			Found:    true,
		}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("treats 404 as a clean not-found", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"status":"NOT_FOUND"}`, http.StatusNotFound)
		}))
		defer srv.Close()

		got, err := massiveClientForServer(srv).GetTickerDetails(context.Background(), "NOPE")
		if err != nil {
			t.Fatalf("expected nil error for unknown ticker, got %v", err)
		}
		if got.Found {
			t.Errorf("expected Found=false, got %+v", got)
		}
	})

	t.Run("surfaces other errors", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"status":"ERROR"}`, http.StatusInternalServerError)
		}))
		defer srv.Close()

		if _, err := massiveClientForServer(srv).GetTickerDetails(context.Background(), "AAPL"); err == nil {
			t.Fatal("expected error from 500 response")
		}
	})
}

func TestMassiveClient_FetchEPS(t *testing.T) {
	t.Run("requires a symbol", func(t *testing.T) {
		c := NewMassiveClient("test-key")
		if _, err := c.FetchEPS(context.Background(), ""); err == nil {
			t.Error("expected error when symbol is empty")
		}
	})

	t.Run("extracts EPS from quarterly filings", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/vX/reference/financials") {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			writeJSON(w, `{"results":[{"filing_date":"2025-02-01","fiscal_period":"Q1","fiscal_year":"2025","financials":{"income_statement":{"basic_earnings_per_share":{"value":6.42},"diluted_earnings_per_share":{"value":6.38}}}}]}`)
		}))
		defer srv.Close()

		got, err := massiveClientForServer(srv).FetchEPS(context.Background(), "AAPL")
		if err != nil {
			t.Fatalf("FetchEPS: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d points, want 1", len(got))
		}
		want := EPSPoint{
			Symbol: "AAPL", FiscalPeriod: "Q1", FiscalYear: "2025",
			Time:  time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
			Basic: 6.42, Diluted: 6.38,
		}
		if got[0] != want {
			t.Errorf("got %+v, want %+v", got[0], want)
		}
	})

	t.Run("returns ErrNoFinancials when the provider has none", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, `{"results":[]}`)
		}))
		defer srv.Close()

		_, err := massiveClientForServer(srv).FetchEPS(context.Background(), "AAPL")
		if !errors.Is(err, ErrNoFinancials) {
			t.Fatalf("expected ErrNoFinancials, got %v", err)
		}
	})
}

// writeJSON sends body as an application/json response. The Massive client (resty) only
// auto-unmarshals when the Content-Type is JSON, so the header is required.
func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}

// millis renders t as a Unix-millisecond integer literal for embedding in JSON fixtures.
func millis(t time.Time) string {
	return strconv.FormatInt(t.UnixMilli(), 10)
}
