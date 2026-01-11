package finance

import (
	"encoding/json"
	"github.com/google/go-cmp/cmp"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFinanceHandler_IncomeExpenseReport(t *testing.T) {
	tcs := []struct {
		name       string
		userId     string
		query      string
		expecErr   string
		expectCode int
	}{
		{
			name:       "successful request with date range",
			userId:     tenant1,
			query:      "?startDate=2025-01-01&endDate=2025-12-31",
			expectCode: http.StatusOK,
		},
		//{ // not testable
		//	name:       "successful request with default date range",
		//	userId:     tenant1,
		//	query:      "",
		//	expectCode: http.StatusOK,
		//},
		{
			name:       "successful request with only end date",
			userId:     tenant1,
			query:      "?endDate=2025-01-31",
			expectCode: http.StatusOK,
		},
		{
			name:       "successful request with only start date",
			userId:     tenant1,
			query:      "?startDate=2025-01-01",
			expectCode: http.StatusOK,
		},
		{
			name:       "empty tenant",
			userId:     "",
			query:      "?startDate=2025-01-01&endDate=2025-12-31",
			expecErr:   "unable to list entries: user not provided",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "invalid start date format",
			userId:     tenant1,
			query:      "?startDate=invalid&end_date=2025-12-31",
			expecErr:   "unable to parse start date: parsing time \"invalid\" as \"2006-01-02\": cannot parse \"invalid\" as \"2006\"",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "invalid end date format",
			userId:     tenant1,
			query:      "?startDate=2025-01-01&endDate=invalid",
			expecErr:   "unable to parse end date: parsing time \"invalid\" as \"2006-01-02\": cannot parse \"invalid\" as \"2006\"",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleHandler(t)
			defer end()

			recorder := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/report"+tc.query, nil)
			handler := h.IncomeExpenseReport(tc.userId)
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

				var response incomeExpenseResponse
				err := json.NewDecoder(recorder.Body).Decode(&response)
				if err != nil {
					t.Fatal(err)
				}

				if !hasData(response) {
					t.Errorf("report did not contain any data")
				}
			}
		})
	}
}

func hasData(report incomeExpenseResponse) bool {
	// Check incomes
	for _, income := range report.Incomes {
		for _, val := range income.Values {
			if val.Value != 0 {
				return true
			}
		}
	}

	// Check expenses
	for _, expense := range report.Expenses {
		for _, val := range expense.Values {
			if val.Value != 0 {
				return true
			}
		}
	}

	// No non-zero values found
	return false
}

func TestFinanceHandler_AccountBalance(t *testing.T) {
	tcs := []struct {
		name       string
		userId     string
		query      string
		expecErr   string
		expectCode int
		wantValue  map[uint][]float64
	}{
		{
			name:       "successful request with one account",
			userId:     tenant1,
			query:      "?accountIds=1",
			expectCode: http.StatusOK,
			wantValue: map[uint][]float64{
				1: {-80},
			},
		},
		{
			name:       "successful request with two accounts",
			userId:     tenant1,
			query:      "?accountIds=1,2",
			expectCode: http.StatusOK,
			wantValue: map[uint][]float64{
				1: {-80},
				2: {-26},
			},
		},
		{
			name:       "successful request with accounts and steps",
			userId:     tenant1,
			query:      "?accountIds=1,2&steps=3",
			expectCode: http.StatusOK,
			wantValue: map[uint][]float64{
				1: {-79, -79, -80},
				2: {-26, -26, -26},
			},
		},
		{
			name:       "successful request with accounts and steps and end date",
			userId:     tenant1,
			query:      "?accountIds=1,2&steps=5&endDate=2025-01-15",
			expectCode: http.StatusOK,
			wantValue: map[uint][]float64{
				1: {-39, -49, -59, -69, -79},
				2: {-26, -26, -26, -26, -26},
			},
		},

		{
			name:       "successful request with accounts and dates",
			userId:     tenant1,
			query:      "?accountIds=1,2&steps=3&endDate=2025-01-15&startDate=2025-01-03",
			expectCode: http.StatusOK,
			wantValue: map[uint][]float64{
				1: {-3, -29, -79},
				2: {-3, -16, -26},
			},
		},
		{
			name:       "empty result on different tenant",
			userId:     "other",
			query:      "?accountIds=1,2",
			expectCode: http.StatusBadRequest,
			expecErr:   "account id not found: 1",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleHandler(t)
			defer end()

			recorder := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/report"+tc.query, nil)
			handler := h.AccountBalance(tc.userId)
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

				var response accountBalancesResponse
				err := json.NewDecoder(recorder.Body).Decode(&response)
				if err != nil {
					t.Fatal(err)
				}

				// verify all values given in the want
				for key, values := range tc.wantValue {
					gotVals := make([]float64, len(response.Accounts[key]))
					for i, val := range response.Accounts[key] {
						gotVals[i] = val.Sum
					}
					if diff := cmp.Diff(values, gotVals); diff != "" {
						t.Errorf("unexpected value for account id %d (+want -got):\n%s", key, diff)
					}
				}
			}
		})
	}
}
