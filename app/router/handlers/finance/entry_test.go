package finance

import (
	"bytes"

	"encoding/json"
	"github.com/andresbott/etna/internal/model/finance"

	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFinanceHandler_CreateEntry(t *testing.T) {
	tcs := []struct {
		name       string
		userId     string
		payload    io.Reader
		expecErr   string
		expectCode int
	}{
		{
			name:       "successful request",
			userId:     "user123",
			payload:    bytes.NewBuffer([]byte(`{"name":"Salary", "amount":1000.0, "date":"2024-01-01T00:00:00Z", "type":"income", "target_account_id":1, "category_id":1}`)),
			expectCode: http.StatusOK,
		},
		{
			name:       "empty userId",
			userId:     "",
			payload:    bytes.NewBuffer([]byte(`{"name":"Salary", "amount":1000.0, "date":"2024-01-01T00:00:00Z", "type":"income", "target_account_id":1, "category_id":1}`)),
			expecErr:   "unable to create entry: user not provided",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "empty payload",
			userId:     "user123",
			payload:    nil,
			expecErr:   "request had empty body",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "malformed payload",
			userId:     "user123",
			payload:    bytes.NewBuffer([]byte(`{"name":"Salary"`)),
			expecErr:   "unable to decode json: unexpected EOF",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, db, err := SampleHandler()
			if err != nil {
				t.Fatal(err)
			}
			uDb, _ := db.DB()
			defer uDb.Close()

			recorder := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/entries", tc.payload)
			handler := h.CreateEntry(tc.userId)
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
				}

				entry := finance.Entry{}
				err := json.NewDecoder(recorder.Body).Decode(&entry)
				if err != nil {
					t.Fatal(err)
				}
				if entry.Id == 0 {
					t.Error("returned entry ID is empty")
				}
			}
		})
	}
}

func TestFinanceHandler_UpdateEntry(t *testing.T) {
	tcs := []struct {
		name       string
		userId     string
		entryId    uint
		payload    io.Reader
		expecErr   string
		expectCode int
	}{
		{
			name:       "successful request",
			userId:     "user123",
			entryId:    1,
			payload:    bytes.NewBuffer([]byte(`{"name":"Updated Salary", "amount":2000.0}`)),
			expectCode: http.StatusOK,
		},
		{
			name:       "empty userId",
			userId:     "",
			entryId:    1,
			payload:    bytes.NewBuffer([]byte(`{"name":"Updated Salary", "amount":2000.0}`)),
			expecErr:   "unable to update entry: user not provided",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "empty payload",
			userId:     "user123",
			entryId:    1,
			payload:    nil,
			expecErr:   "request had empty body",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "malformed payload",
			userId:     "user123",
			entryId:    1,
			payload:    bytes.NewBuffer([]byte(`{"name":"Updated Salary"`)),
			expecErr:   "unable to decode json: unexpected EOF",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, db, err := SampleHandler()
			if err != nil {
				t.Fatal(err)
			}
			uDb, _ := db.DB()
			defer uDb.Close()

			recorder := httptest.NewRecorder()
			req, _ := http.NewRequest("PUT", "/api/entries/"+string(tc.entryId), tc.payload)
			handler := h.UpdateEntry(tc.entryId, tc.userId)
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
				}
			}
		})
	}
}

func TestFinanceHandler_DeleteEntry(t *testing.T) {
	tcs := []struct {
		name       string
		userId     string
		entryId    uint
		expecErr   string
		expectCode int
	}{
		{
			name:       "successful request",
			userId:     "user123",
			entryId:    1,
			expectCode: http.StatusOK,
		},
		{
			name:       "empty userId",
			userId:     "",
			entryId:    1,
			expecErr:   "unable to delete entry: user not provided",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, db, err := SampleHandler()
			if err != nil {
				t.Fatal(err)
			}
			uDb, _ := db.DB()
			defer uDb.Close()

			recorder := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", "/api/entries/"+string(tc.entryId), nil)
			handler := h.DeleteEntry(tc.entryId, tc.userId)
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
				}
			}
		})
	}
}

func TestFinanceHandler_ListEntries(t *testing.T) {
	tcs := []struct {
		name       string
		userId     string
		query      string
		expecErr   string
		expectCode int
	}{
		{
			name:       "successful request",
			userId:     "user123",
			query:      "?start_date=2024-01-01T00:00:00Z&end_date=2024-12-31T23:59:59Z",
			expectCode: http.StatusOK,
		},
		{
			name:       "empty userId",
			userId:     "",
			query:      "?start_date=2024-01-01T00:00:00Z&end_date=2024-12-31T23:59:59Z",
			expecErr:   "unable to list entries: user not provided",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "invalid date format",
			userId:     "user123",
			query:      "?start_date=invalid&end_date=2024-12-31T23:59:59Z",
			expecErr:   "invalid start_date format",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, db, err := SampleHandler()
			if err != nil {
				t.Fatal(err)
			}
			uDb, _ := db.DB()
			defer uDb.Close()

			recorder := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/entries"+tc.query, nil)
			handler := h.ListEntries(tc.userId)
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

func TestFinanceHandler_LockEntries(t *testing.T) {
	tcs := []struct {
		name       string
		userId     string
		query      string
		expecErr   string
		expectCode int
	}{
		{
			name:       "successful request",
			userId:     "user123",
			query:      "?date=2024-01-01T00:00:00Z",
			expectCode: http.StatusOK,
		},
		{
			name:       "empty userId",
			userId:     "",
			query:      "?date=2024-01-01T00:00:00Z",
			expecErr:   "unable to lock entries: user not provided",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "missing date",
			userId:     "user123",
			query:      "",
			expecErr:   "date parameter is required",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "invalid date format",
			userId:     "user123",
			query:      "?date=invalid",
			expecErr:   "invalid date format",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, db, err := SampleHandler()
			if err != nil {
				t.Fatal(err)
			}
			uDb, _ := db.DB()
			defer uDb.Close()

			recorder := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/entries/lock"+tc.query, nil)
			handler := h.LockEntries(tc.userId)
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
				}
			}
		})
	}
}
