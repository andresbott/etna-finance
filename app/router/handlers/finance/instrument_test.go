package finance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andresbott/etna/internal/marketdata"
	"github.com/andresbott/etna/internal/marketdata/importer"
)

// ---------------------------------------------------------------------------
// Instrument tests
// ---------------------------------------------------------------------------

func TestHandler_ListInstruments(t *testing.T) {
	tcs := []struct {
		name       string
		userId     string
		expectErr  string
		expectCode int
		wantCount  int
	}{
		{
			name:       "empty list for tenant",
			userId:     tenant1,
			expectCode: http.StatusOK,
			wantCount:  0,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleHandler(t)
			defer end()

			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/fin/instrument", nil)
			h.ListInstruments().ServeHTTP(rec, req)

			if tc.expectErr != "" {
				if rec.Code != tc.expectCode {
					t.Errorf("status: got %d want %d", rec.Code, tc.expectCode)
				}
				body, _ := io.ReadAll(rec.Body)
				if got := strings.TrimSuffix(string(body), "\n"); got != tc.expectErr {
					t.Errorf("body: got %q want %q", got, tc.expectErr)
				}
				return
			}
			if rec.Code != tc.expectCode {
				t.Errorf("status: got %d want %d", rec.Code, tc.expectCode)
			}
			var resp struct {
				Items []instrumentPayload `json:"items"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(resp.Items) != tc.wantCount {
				t.Errorf("items: got %d want %d", len(resp.Items), tc.wantCount)
			}
		})
	}
}

func TestHandler_CreateInstrument(t *testing.T) {
	tcs := []struct {
		name       string
		userId     string
		payload    io.Reader
		expectErr  string
		expectCode int
	}{
		{
			name:       "success",
			userId:     tenant1,
			payload:    bytes.NewBuffer([]byte(`{"symbol":"AAPL","name":"Apple Inc.","currency":"USD","type":"Stock","exchange":"NASDAQ"}`)),
			expectCode: http.StatusOK,
		},
		{
			name:       "missing type",
			userId:     tenant1,
			payload:    bytes.NewBuffer([]byte(`{"symbol":"AAPL","name":"Apple Inc.","currency":"USD","exchange":"NASDAQ"}`)),
			expectErr:  "type cannot be empty",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "missing exchange",
			userId:     tenant1,
			payload:    bytes.NewBuffer([]byte(`{"symbol":"AAPL","name":"Apple Inc.","currency":"USD","type":"Stock"}`)),
			expectErr:  "exchange cannot be empty",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "empty body",
			userId:     tenant1,
			payload:    nil,
			expectErr:  "request had empty body",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "invalid currency",
			userId:     tenant1,
			payload:    bytes.NewBuffer([]byte(`{"symbol":"X","name":"X","currency":"INVALID","type":"Stock","exchange":"NASDAQ"}`)),
			expectErr:  "invalid currency: INVALID",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "empty symbol",
			userId:     tenant1,
			payload:    bytes.NewBuffer([]byte(`{"symbol":"","name":"X","currency":"USD","type":"Stock","exchange":"NASDAQ"}`)),
			expectErr:  "symbol cannot be empty",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleHandler(t)
			defer end()

			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/fin/instrument", tc.payload)
			h.CreateInstrument().ServeHTTP(rec, req)

			if tc.expectErr != "" {
				if rec.Code != tc.expectCode {
					t.Errorf("status: got %d want %d", rec.Code, tc.expectCode)
				}
				body, _ := io.ReadAll(rec.Body)
				if got := strings.TrimSuffix(string(body), "\n"); got != tc.expectErr {
					t.Errorf("body: got %q want %q", got, tc.expectErr)
				}
				return
			}
			if rec.Code != tc.expectCode {
				t.Errorf("status: got %d want %d", rec.Code, tc.expectCode)
			}
			var out instrumentPayload
			if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if out.ID == 0 {
				t.Error("expected non-zero id")
			}
			if out.Symbol != "AAPL" || out.Name != "Apple Inc." || out.Currency != "USD" {
				t.Errorf("unexpected response: %+v", out)
			}
		})
	}
}

func TestHandler_CreateInstrument_duplicateSymbol(t *testing.T) {
	h, end := SampleHandler(t)
	defer end()

	rec1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/fin/instrument", bytes.NewBuffer([]byte(`{"symbol":"DUP","name":"First","currency":"USD","type":"Stock","exchange":"NASDAQ"}`)))
	h.CreateInstrument().ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("first create: %d %s", rec1.Code, rec1.Body.String())
	}

	rec2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/fin/instrument", bytes.NewBuffer([]byte(`{"symbol":"DUP","name":"Second","currency":"EUR","type":"Stock","exchange":"NASDAQ"}`)))
	h.CreateInstrument().ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Errorf("duplicate create: got status %d want 409", rec2.Code)
	}
	body, _ := io.ReadAll(rec2.Body)
	if got := strings.TrimSuffix(string(body), "\n"); got != marketdata.ErrInstrumentSymbolDuplicate.Error() {
		t.Errorf("body: got %q want %q", got, marketdata.ErrInstrumentSymbolDuplicate.Error())
	}
}

func TestHandler_GetInstrument(t *testing.T) {
	h, end := SampleHandler(t)
	defer end()

	// Create an instrument first
	createRec := httptest.NewRecorder()
	createReq, _ := http.NewRequest(http.MethodPost, "/fin/instrument", bytes.NewBuffer([]byte(`{"symbol":"GET","name":"Get Test","currency":"EUR","type":"Stock","exchange":"NASDAQ"}`)))
	h.CreateInstrument().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create failed: %d %s", createRec.Code, createRec.Body.String())
	}
	var created instrumentPayload
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	tcs := []struct {
		name       string
		id         uint
		userId     string
		expectErr  string
		expectCode int
	}{
		{
			name:       "success",
			id:         created.ID,
			userId:     tenant1,
			expectCode: http.StatusOK,
		},
		{
			name:       "not found",
			id:         99999,
			userId:     tenant1,
			expectErr:  "instrument not found",
			expectCode: http.StatusNotFound,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/fin/instrument/1", nil)
			h.GetInstrument(tc.id).ServeHTTP(rec, req)

			if tc.expectErr != "" {
				if rec.Code != tc.expectCode {
					t.Errorf("status: got %d want %d", rec.Code, tc.expectCode)
				}
				body, _ := io.ReadAll(rec.Body)
				if got := strings.TrimSuffix(string(body), "\n"); got != tc.expectErr {
					t.Errorf("body: got %q want %q", got, tc.expectErr)
				}
				return
			}
			if rec.Code != tc.expectCode {
				t.Errorf("status: got %d want %d", rec.Code, tc.expectCode)
			}
			var out instrumentPayload
			if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if out.ID != created.ID || out.Symbol != "GET" || out.Currency != "EUR" {
				t.Errorf("unexpected response: %+v", out)
			}
		})
	}
}

func TestHandler_UpdateInstrument(t *testing.T) {
	h, end := SampleHandler(t)
	defer end()

	createRec := httptest.NewRecorder()
	createReq, _ := http.NewRequest(http.MethodPost, "/fin/instrument", bytes.NewBuffer([]byte(`{"symbol":"UPD","name":"Update Me","currency":"USD","type":"Stock","exchange":"NASDAQ"}`)))
	h.CreateInstrument().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create failed: %d %s", createRec.Code, createRec.Body.String())
	}
	var created instrumentPayload
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	rec2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/fin/instrument", bytes.NewBuffer([]byte(`{"symbol":"TAKEN","name":"Other","currency":"EUR","type":"ETF","exchange":"NYSE"}`)))
	h.CreateInstrument().ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("second create: %d %s", rec2.Code, rec2.Body.String())
	}

	tcs := []struct {
		name       string
		id         uint
		userId     string
		payload    io.Reader
		expectErr  string
		expectCode int
	}{
		{
			name:       "success update name",
			id:         created.ID,
			userId:     tenant1,
			payload:    bytes.NewBuffer([]byte(`{"name":"Updated Name"}`)),
			expectCode: http.StatusOK,
		},
		{
			name:       "not found",
			id:         99999,
			userId:     tenant1,
			payload:    bytes.NewBuffer([]byte(`{"name":"X"}`)),
			expectErr:  "instrument not found",
			expectCode: http.StatusNotFound,
		},
		{
			name:       "no changes",
			id:         created.ID,
			userId:     tenant1,
			payload:    bytes.NewBuffer([]byte(`{}`)),
			expectErr:  "no changes applied",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "duplicate symbol rejected",
			id:         created.ID,
			userId:     tenant1,
			payload:    bytes.NewBuffer([]byte(`{"symbol":"TAKEN"}`)),
			expectErr:  marketdata.ErrInstrumentSymbolDuplicate.Error(),
			expectCode: http.StatusConflict,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPut, "/fin/instrument/1", tc.payload)
			h.UpdateInstrument(tc.id).ServeHTTP(rec, req)

			if tc.expectErr != "" {
				if rec.Code != tc.expectCode {
					t.Errorf("status: got %d want %d", rec.Code, tc.expectCode)
				}
				body, _ := io.ReadAll(rec.Body)
				if got := strings.TrimSuffix(string(body), "\n"); got != tc.expectErr {
					t.Errorf("body: got %q want %q", got, tc.expectErr)
				}
				return
			}
			if rec.Code != tc.expectCode {
				t.Errorf("status: got %d want %d", rec.Code, tc.expectCode)
			}
		})
	}
}

func TestHandler_InstrumentNotes(t *testing.T) {
	h, end := SampleHandler(t)
	defer end()

	// Create with notes
	createRec := httptest.NewRecorder()
	createReq, _ := http.NewRequest(http.MethodPost, "/fin/instrument",
		bytes.NewBuffer([]byte(`{"symbol":"NOTE","name":"With Notes","currency":"USD","notes":"initial details","type":"Stock","exchange":"NASDAQ"}`)))
	h.CreateInstrument().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create failed: %d %s", createRec.Code, createRec.Body.String())
	}
	var created instrumentPayload
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Notes != "initial details" {
		t.Errorf("create response notes: got %q want %q", created.Notes, "initial details")
	}

	// Get returns the notes
	getRec := httptest.NewRecorder()
	getReq, _ := http.NewRequest(http.MethodGet, "/fin/instrument/1", nil)
	h.GetInstrument(created.ID).ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get failed: %d %s", getRec.Code, getRec.Body.String())
	}
	var got instrumentPayload
	if err := json.NewDecoder(getRec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Notes != "initial details" {
		t.Errorf("get notes: got %q want %q", got.Notes, "initial details")
	}

	// Update notes
	updRec := httptest.NewRecorder()
	updReq, _ := http.NewRequest(http.MethodPut, "/fin/instrument/1",
		bytes.NewBuffer([]byte(`{"notes":"updated details"}`)))
	h.UpdateInstrument(created.ID).ServeHTTP(updRec, updReq)
	if updRec.Code != http.StatusOK {
		t.Fatalf("update failed: %d %s", updRec.Code, updRec.Body.String())
	}

	// Get reflects the updated notes
	getRec2 := httptest.NewRecorder()
	getReq2, _ := http.NewRequest(http.MethodGet, "/fin/instrument/1", nil)
	h.GetInstrument(created.ID).ServeHTTP(getRec2, getReq2)
	var got2 instrumentPayload
	if err := json.NewDecoder(getRec2.Body).Decode(&got2); err != nil {
		t.Fatal(err)
	}
	if got2.Notes != "updated details" {
		t.Errorf("get after update notes: got %q want %q", got2.Notes, "updated details")
	}
}

func TestHandler_DeleteInstrument(t *testing.T) {
	h, end := SampleHandler(t)
	defer end()

	createRec := httptest.NewRecorder()
	createReq, _ := http.NewRequest(http.MethodPost, "/fin/instrument", bytes.NewBuffer([]byte(`{"symbol":"DEL","name":"Delete Me","currency":"CHF","type":"Stock","exchange":"SIX"}`)))
	h.CreateInstrument().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create failed: %d %s", createRec.Code, createRec.Body.String())
	}
	var created instrumentPayload
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	tcs := []struct {
		name       string
		id         uint
		userId     string
		expectErr  string
		expectCode int
	}{
		{
			name:       "success",
			id:         created.ID,
			userId:     tenant1,
			expectCode: http.StatusOK,
		},
		{
			name:       "not found",
			id:         99999,
			userId:     tenant1,
			expectErr:  "instrument not found",
			expectCode: http.StatusNotFound,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodDelete, "/fin/instrument/1", nil)
			h.DeleteInstrument(tc.id).ServeHTTP(rec, req)

			if tc.expectErr != "" {
				if rec.Code != tc.expectCode {
					t.Errorf("status: got %d want %d", rec.Code, tc.expectCode)
				}
				body, _ := io.ReadAll(rec.Body)
				if got := strings.TrimSuffix(string(body), "\n"); got != tc.expectErr {
					t.Errorf("body: got %q want %q", got, tc.expectErr)
				}
				return
			}
			if rec.Code != tc.expectCode {
				t.Errorf("status: got %d want %d", rec.Code, tc.expectCode)
			}
		})
	}
}

type fakeReference struct {
	details   importer.TickerDetails
	err       error
	gotSymbol string
}

func (f *fakeReference) GetTickerDetails(_ context.Context, symbol string) (importer.TickerDetails, error) {
	f.gotSymbol = symbol
	return f.details, f.err
}

func TestMapTickerDetails(t *testing.T) {
	tcs := []struct {
		name string
		in   importer.TickerDetails
		want instrumentLookupResponse
	}{
		{
			name: "common stock on nyse",
			in:   importer.TickerDetails{Name: "Apple Inc.", Currency: "USD", Type: "CS", Exchange: "XNYS", Notes: "n", Found: true},
			want: instrumentLookupResponse{Name: "Apple Inc.", Currency: "USD", Type: "Stock", Exchange: "NYSE", Notes: "n"},
		},
		{
			name: "etf on nasdaq",
			in:   importer.TickerDetails{Name: "SPY", Currency: "usd", Type: "ETF", Exchange: "XNAS", Found: true},
			want: instrumentLookupResponse{Name: "SPY", Currency: "usd", Type: "ETF", Exchange: "NASDAQ"},
		},
		{
			name: "unknown type and exchange pass through raw",
			in:   importer.TickerDetails{Name: "X", Type: "ETN", Exchange: "XFOO", Found: true},
			want: instrumentLookupResponse{Name: "X", Type: "ETN", Exchange: "XFOO"},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := mapTickerDetails(tc.in)
			if got != tc.want {
				t.Errorf("mapTickerDetails(%+v) = %+v, want %+v", tc.in, got, tc.want)
			}
		})
	}
}

func TestHandler_LookupInstrument(t *testing.T) {
	t.Run("nil client returns 204", func(t *testing.T) {
		h, end := SampleHandler(t)
		defer end()
		rec := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/fin/instrument/lookup?symbol=AAPL", nil)
		h.LookupInstrument().ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Errorf("status: got %d want %d", rec.Code, http.StatusNoContent)
		}
	})

	t.Run("missing symbol returns 400", func(t *testing.T) {
		h, end := SampleHandler(t)
		defer end()
		h.Reference = &fakeReference{details: importer.TickerDetails{Found: true, Name: "x"}}
		rec := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/fin/instrument/lookup", nil)
		h.LookupInstrument().ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status: got %d want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("not found returns 204", func(t *testing.T) {
		h, end := SampleHandler(t)
		defer end()
		h.Reference = &fakeReference{details: importer.TickerDetails{Found: false}}
		rec := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/fin/instrument/lookup?symbol=NOPE", nil)
		h.LookupInstrument().ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Errorf("status: got %d want %d", rec.Code, http.StatusNoContent)
		}
	})

	t.Run("client error returns 204", func(t *testing.T) {
		h, end := SampleHandler(t)
		defer end()
		h.Reference = &fakeReference{err: errors.New("upstream down")}
		rec := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/fin/instrument/lookup?symbol=AAPL", nil)
		h.LookupInstrument().ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Errorf("status: got %d want %d", rec.Code, http.StatusNoContent)
		}
	})

	t.Run("rate limit returns 429 with Retry-After", func(t *testing.T) {
		h, end := SampleHandler(t)
		defer end()
		h.Reference = &fakeReference{err: errors.New("429 Too Many Requests: retry after 30")}
		rec := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/fin/instrument/lookup?symbol=AAPL", nil)
		h.LookupInstrument().ServeHTTP(rec, req)
		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("status: got %d want %d", rec.Code, http.StatusTooManyRequests)
		}
		if ra := rec.Header().Get("Retry-After"); ra != "30" {
			t.Errorf("Retry-After: got %q want %q", ra, "30")
		}
	})

	t.Run("found returns mapped 200", func(t *testing.T) {
		h, end := SampleHandler(t)
		defer end()
		ref := &fakeReference{details: importer.TickerDetails{
			Name: "Apple Inc.", Currency: "USD", Type: "CS", Exchange: "XNAS", Notes: "n", Found: true,
		}}
		h.Reference = ref
		rec := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/fin/instrument/lookup?symbol=AAPL", nil)
		h.LookupInstrument().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d want %d", rec.Code, http.StatusOK)
		}
		var got instrumentLookupResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		want := instrumentLookupResponse{Name: "Apple Inc.", Currency: "USD", Type: "Stock", Exchange: "NASDAQ", Notes: "n"}
		if got != want {
			t.Errorf("body: got %+v want %+v", got, want)
		}
		if ref.gotSymbol != "AAPL" {
			t.Errorf("symbol forwarded to client: got %q want %q", ref.gotSymbol, "AAPL")
		}
	})
}
