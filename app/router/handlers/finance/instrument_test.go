package finance

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andresbott/etna/internal/marketdata"
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
		{
			name:       "empty user rejected",
			userId:     "",
			expectErr:  "unable to list instruments: user not provided",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleHandler(t)
			defer end()

			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/fin/instrument", nil)
			h.ListInstruments(tc.userId).ServeHTTP(rec, req)

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
			payload:    bytes.NewBuffer([]byte(`{"symbol":"AAPL","name":"Apple Inc.","currency":"USD"}`)),
			expectCode: http.StatusOK,
		},
		{
			name:       "empty user",
			userId:     "",
			payload:    bytes.NewBuffer([]byte(`{"symbol":"AAPL","name":"Apple","currency":"USD"}`)),
			expectErr:  "unable to create instrument: user not provided",
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
			payload:    bytes.NewBuffer([]byte(`{"symbol":"X","name":"X","currency":"INVALID"}`)),
			expectErr:  "invalid currency: INVALID",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "empty symbol",
			userId:     tenant1,
			payload:    bytes.NewBuffer([]byte(`{"symbol":"","name":"X","currency":"USD"}`)),
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
			h.CreateInstrument(tc.userId).ServeHTTP(rec, req)

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
	req1, _ := http.NewRequest(http.MethodPost, "/fin/instrument", bytes.NewBuffer([]byte(`{"symbol":"DUP","name":"First","currency":"USD"}`)))
	h.CreateInstrument(tenant1).ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("first create: %d %s", rec1.Code, rec1.Body.String())
	}

	rec2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/fin/instrument", bytes.NewBuffer([]byte(`{"symbol":"DUP","name":"Second","currency":"EUR"}`)))
	h.CreateInstrument(tenant1).ServeHTTP(rec2, req2)
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
	createReq, _ := http.NewRequest(http.MethodPost, "/fin/instrument", bytes.NewBuffer([]byte(`{"symbol":"GET","name":"Get Test","currency":"EUR"}`)))
	h.CreateInstrument(tenant1).ServeHTTP(createRec, createReq)
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
		{
			name:       "empty user",
			id:         created.ID,
			userId:     "",
			expectErr:  "unable to get instrument: user not provided",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/fin/instrument/1", nil)
			h.GetInstrument(tc.id, tc.userId).ServeHTTP(rec, req)

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
	createReq, _ := http.NewRequest(http.MethodPost, "/fin/instrument", bytes.NewBuffer([]byte(`{"symbol":"UPD","name":"Update Me","currency":"USD"}`)))
	h.CreateInstrument(tenant1).ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create failed: %d %s", createRec.Code, createRec.Body.String())
	}
	var created instrumentPayload
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	rec2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/fin/instrument", bytes.NewBuffer([]byte(`{"symbol":"TAKEN","name":"Other","currency":"EUR"}`)))
	h.CreateInstrument(tenant1).ServeHTTP(rec2, req2)
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
			name:       "empty user",
			id:         created.ID,
			userId:     "",
			payload:    bytes.NewBuffer([]byte(`{"name":"X"}`)),
			expectErr:  "unable to update instrument: user not provided",
			expectCode: http.StatusBadRequest,
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
			h.UpdateInstrument(tc.id, tc.userId).ServeHTTP(rec, req)

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

func TestHandler_DeleteInstrument(t *testing.T) {
	h, end := SampleHandler(t)
	defer end()

	createRec := httptest.NewRecorder()
	createReq, _ := http.NewRequest(http.MethodPost, "/fin/instrument", bytes.NewBuffer([]byte(`{"symbol":"DEL","name":"Delete Me","currency":"CHF"}`)))
	h.CreateInstrument(tenant1).ServeHTTP(createRec, createReq)
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
		{
			name:       "empty user",
			id:         created.ID,
			userId:     "",
			expectErr:  "unable to delete instrument: user not provided",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodDelete, "/fin/instrument/1", nil)
			h.DeleteInstrument(tc.id, tc.userId).ServeHTTP(rec, req)

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
