package finance

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFinanceHandler_GetReport(t *testing.T) {
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
			expecErr:   "invalid startDate format",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "invalid end date format",
			userId:     "user123",
			query:      "?startDate=2024-01-01&endDate=invalid",
			expecErr:   "invalid endDate format",
			expectCode: http.StatusBadRequest,
		},
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
