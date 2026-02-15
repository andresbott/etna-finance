package finance

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler_ListSecurities(t *testing.T) {
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
			expectErr:  "unable to list securities: user not provided",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleHandler(t)
			defer end()

			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/fin/security", nil)
			h.ListSecurities(tc.userId).ServeHTTP(rec, req)

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
				Items []securityPayload `json:"items"`
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

func TestHandler_CreateSecurity(t *testing.T) {
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
			expectErr:  "unable to create security: user not provided",
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
			req, _ := http.NewRequest(http.MethodPost, "/fin/security", tc.payload)
			h.CreateSecurity(tc.userId).ServeHTTP(rec, req)

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
			var out securityPayload
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

func TestHandler_GetSecurity(t *testing.T) {
	h, end := SampleHandler(t)
	defer end()

	// Create a security first
	createRec := httptest.NewRecorder()
	createReq, _ := http.NewRequest(http.MethodPost, "/fin/security", bytes.NewBuffer([]byte(`{"symbol":"GET","name":"Get Test","currency":"EUR"}`)))
	h.CreateSecurity(tenant1).ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create failed: %d %s", createRec.Code, createRec.Body.String())
	}
	var created securityPayload
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
			expectErr:  "security not found",
			expectCode: http.StatusNotFound,
		},
		{
			name:       "empty user",
			id:         created.ID,
			userId:     "",
			expectErr:  "unable to get security: user not provided",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/fin/security/1", nil)
			h.GetSecurity(tc.id, tc.userId).ServeHTTP(rec, req)

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
			var out securityPayload
			if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if out.ID != created.ID || out.Symbol != "GET" || out.Currency != "EUR" {
				t.Errorf("unexpected response: %+v", out)
			}
		})
	}
}

func TestHandler_UpdateSecurity(t *testing.T) {
	h, end := SampleHandler(t)
	defer end()

	createRec := httptest.NewRecorder()
	createReq, _ := http.NewRequest(http.MethodPost, "/fin/security", bytes.NewBuffer([]byte(`{"symbol":"UPD","name":"Update Me","currency":"USD"}`)))
	h.CreateSecurity(tenant1).ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create failed: %d %s", createRec.Code, createRec.Body.String())
	}
	var created securityPayload
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatal(err)
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
			expectErr:  "security not found",
			expectCode: http.StatusNotFound,
		},
		{
			name:       "empty user",
			id:         created.ID,
			userId:     "",
			payload:    bytes.NewBuffer([]byte(`{"name":"X"}`)),
			expectErr:  "unable to update security: user not provided",
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
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPut, "/fin/security/1", tc.payload)
			h.UpdateSecurity(tc.id, tc.userId).ServeHTTP(rec, req)

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

func TestHandler_DeleteSecurity(t *testing.T) {
	h, end := SampleHandler(t)
	defer end()

	createRec := httptest.NewRecorder()
	createReq, _ := http.NewRequest(http.MethodPost, "/fin/security", bytes.NewBuffer([]byte(`{"symbol":"DEL","name":"Delete Me","currency":"CHF"}`)))
	h.CreateSecurity(tenant1).ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create failed: %d %s", createRec.Code, createRec.Body.String())
	}
	var created securityPayload
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
			expectErr:  "security not found",
			expectCode: http.StatusNotFound,
		},
		{
			name:       "empty user",
			id:         created.ID,
			userId:     "",
			expectErr:  "unable to delete security: user not provided",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodDelete, "/fin/security/1", nil)
			h.DeleteSecurity(tc.id, tc.userId).ServeHTTP(rec, req)

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
