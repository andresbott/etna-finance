package finance

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestFinanceHandler_CreateTx(t *testing.T) {
	tcs := []struct {
		name       string
		userId     string
		payload    io.Reader
		expecErr   string
		expectCode int
	}{
		{
			name:       "successful request",
			userId:     tenant1,
			payload:    bytes.NewBuffer([]byte(`{"description":"Salary", "Amount":1000.0, "date":"2024-01-01T00:00:00Z", "type":"income", "AccountId":1, "categoryId":0}`)),
			expectCode: http.StatusOK,
		},
		{
			name:       "empty tenant",
			userId:     "",
			payload:    bytes.NewBuffer([]byte(`{"description":"Salary", "Amount":1000.0, "date":"2024-01-01T00:00:00Z", "type":"income", "AccountId":1, "categoryId":0}`)),
			expecErr:   "unable to create entry: user not provided",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "empty payload",
			userId:     tenant1,
			payload:    nil,
			expecErr:   "request had empty body",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "malformed payload",
			userId:     tenant1,
			payload:    bytes.NewBuffer([]byte(`{"description":"Salary"`)),
			expecErr:   "unable to decode json: unexpected EOF",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleHandler(t)
			defer end()

			recorder := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/entries", tc.payload)
			handler := h.CreateTx(tc.userId)
			handler.ServeHTTP(recorder, req)

			if tc.expecErr != "" {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectCode)
				}
				respText, err := io.ReadAll(recorder.Body)
				if err != nil {
					t.Fatal(err)
				}
				got := strings.TrimSuffix(string(respText), "\n")
				if got != tc.expecErr {
					t.Errorf("unexpected error message: got \"%s\" want \"%v\"", got, tc.expecErr)
				}
			} else {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectCode)
					t.Errorf("response body: %s", recorder.Body)
					return
				}

				entry := transactionPayload{}
				err := json.NewDecoder(recorder.Body).Decode(&entry)
				if err != nil {
					t.Fatal(err)
				}
				if entry.Id == 0 {
					t.Error("returned entry ID is empty")
				}
				_, err = h.Store.GetTransaction(t.Context(), entry.Id, tc.userId)
				if err != nil {
					t.Errorf("unexpected error in transaction store: %v", err)
				}
			}
		})
	}
}

func TestFinanceHandler_UpdateTx(t *testing.T) {
	tcs := []struct {
		name       string
		userId     string
		entryId    uint
		payload    io.Reader
		expectErr  string
		expectCode int
	}{
		{
			name:       "successful request",
			userId:     tenant1,
			entryId:    1,
			payload:    bytes.NewBuffer([]byte(`{"description":"Updated Salary", "amount":2000.5}`)),
			expectCode: http.StatusOK,
		},
		{
			name:       "empty tenant",
			userId:     "",
			entryId:    1,
			payload:    bytes.NewBuffer([]byte(`{"description":"Updated Salary", "amount":2000.0}`)),
			expectErr:  "unable to update entry: user not provided",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "empty payload",
			userId:     tenant1,
			entryId:    1,
			payload:    nil,
			expectErr:  "request had empty body",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "malformed payload",
			userId:     tenant1,
			entryId:    1,
			payload:    bytes.NewBuffer([]byte(`{"description":"Updated Salary"`)),
			expectErr:  "unable to decode json: unexpected EOF",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleHandler(t)
			defer end()

			recorder := httptest.NewRecorder()
			req, _ := http.NewRequest("PUT", "/api/entries/"+strconv.FormatUint(uint64(tc.entryId), 10), tc.payload)
			handler := h.UpdateTx(tc.entryId, tc.userId)
			handler.ServeHTTP(recorder, req)

			if tc.expectErr != "" {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectCode)
				}
				respText, err := io.ReadAll(recorder.Body)
				if err != nil {
					t.Fatal(err)
				}
				got := strings.TrimSuffix(string(respText), "\n")
				if got != tc.expectErr {
					t.Errorf("unexpected error message: got \"%s\" want \"%v\"", got, tc.expectErr)
				}
			} else {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectCode)
					t.Errorf("response body: %s", recorder.Body)
					return
				}
			}
		})
	}
}

func TestFinanceHandler_DeleteTx(t *testing.T) {
	tcs := []struct {
		name       string
		userId     string
		entryId    uint
		expecErr   string
		expectCode int
	}{
		{
			name:       "successful request",
			userId:     tenant1,
			entryId:    1,
			expectCode: http.StatusOK,
		},
		{
			name:       "empty tenant",
			userId:     "",
			entryId:    1,
			expecErr:   "unable to delete entry: user not provided",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleHandler(t)
			defer end()

			recorder := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", "/api/entries/"+strconv.FormatUint(uint64(tc.entryId), 10), nil)
			handler := h.DeleteTx(tc.entryId, tc.userId)
			handler.ServeHTTP(recorder, req)

			if tc.expecErr != "" {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectCode)
				}
				respText, err := io.ReadAll(recorder.Body)
				if err != nil {
					t.Fatal(err)
				}
				got := strings.TrimSuffix(string(respText), "\n")
				if got != tc.expecErr {
					t.Errorf("unexpected error message: got \"%s\" want \"%v\"", got, tc.expecErr)
				}
			} else {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectCode)
					t.Errorf("response body: %s", recorder.Body)
					return
				}
			}
		})
	}
}

func TestFinanceHandler_ListTx(t *testing.T) {
	tcs := []struct {
		name       string
		userId     string
		query      string
		expecErr   string
		expectCode int
	}{
		{
			name:       "successful request with date range",
			userId:     "user123",
			query:      "?startDate=2024-01-01&end_date=2024-12-31",
			expectCode: http.StatusOK,
		},
		{
			name:       "successful request with default date range",
			userId:     "user123",
			query:      "",
			expectCode: http.StatusOK,
		},
		{
			name:       "successful request with only start date",
			userId:     "user123",
			query:      "?startDate=2024-01-01",
			expectCode: http.StatusOK,
		},
		{
			name:       "successful request with only end date",
			userId:     "user123",
			query:      "?end_date=2024-12-31",
			expectCode: http.StatusOK,
		},
		{
			name:       "empty tenant",
			userId:     "",
			query:      "?startDate=2024-01-01&end_date=2024-12-31",
			expecErr:   "unable to list entries: user not provided",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "invalid start date format",
			userId:     "user123",
			query:      "?startDate=invalid&end_date=2024-12-31",
			expecErr:   "unable to parse start date: parsing time \"invalid\" as \"2006-01-02\": cannot parse \"invalid\" as \"2006\"",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "invalid end date format",
			userId:     "user123",
			query:      "?startDate=2024-01-01&endDate=invalid",
			expecErr:   "unable to parse end date: parsing time \"invalid\" as \"2006-01-02\": cannot parse \"invalid\" as \"2006\"",
			expectCode: http.StatusBadRequest,
		},
		//{
		//	name:       "successful request with single accountId",
		//	userId:     "user123",
		//	query:      "?accountIds=1",
		//	expectCode: http.StatusOK,
		//},
		//{
		//	name:       "successful request with multiple accountIds",
		//	userId:     "user123",
		//	query:      "?accountIds=1&accountIds=2&accountIds=3",
		//	expectCode: http.StatusOK,
		//},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleHandler(t)
			defer end()

			recorder := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/entries"+tc.query, nil)
			handler := h.ListTx(tc.userId)
			handler.ServeHTTP(recorder, req)

			if tc.expecErr != "" {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectCode)
				}
				respText, err := io.ReadAll(recorder.Body)
				if err != nil {
					t.Fatal(err)
				}
				got := strings.TrimSuffix(string(respText), "\n")
				if got != tc.expecErr {
					t.Errorf("unexpected error message: got \"%s\" want \"%v\"", got, tc.expecErr)
				}
			} else {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectCode)
					t.Errorf("response body: %s", recorder.Body)
					return
				}

				var response listEntriesResponse
				err := json.NewDecoder(recorder.Body).Decode(&response)
				if err != nil {
					t.Fatal(err)
				}
				if response.Items == nil {
					t.Error("response items is nil")
				}
			}
		})
	}
}
