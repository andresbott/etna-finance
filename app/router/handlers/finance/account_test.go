package finance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/andresbott/etna/internal/model/finance"
	"github.com/glebarez/sqlite"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/text/currency"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestFinanceHandler_CreateAccount(t *testing.T) {
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
			payload:    bytes.NewBuffer([]byte(`{ "name":"Savings", "currency":"USD", "type":"cash"}`)),
			expectCode: http.StatusOK,
		},
		{
			name:       "empty userId",
			userId:     "",
			payload:    bytes.NewBuffer([]byte(`{"name":"Savings", "currency":"USD", "type":"cash"}`)),
			expecErr:   "unable to create account: user not provided",
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
			payload:    bytes.NewBuffer([]byte(`{"name":"Savings"`)),
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
			req, _ := http.NewRequest("POST", "/api/accounts", tc.payload)
			handler := h.CreateAccount(tc.userId)
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

				acc := finance.Account{}
				err := json.NewDecoder(recorder.Body).Decode(&acc)
				if err != nil {
					t.Fatal(err)
				}
				if acc.ID == 0 {
					t.Error("returned account ID is empty")
				}
			}
		})
	}
}

func TestAccountHandler_Update(t *testing.T) {
	tcs := []struct {
		name       string
		user       string
		payload    io.Reader
		expecErr   string
		expectCode int
	}{
		{
			name:       "successful request",
			user:       user1,
			payload:    bytes.NewBuffer([]byte(`{"name":"Savings","currency":"USD","type":"cash"}`)),
			expectCode: http.StatusOK,
		},
		{
			name:       "missing user",
			user:       "",
			payload:    bytes.NewBuffer([]byte(`{"name":"Savings"}`)),
			expecErr:   "unable to update account: user not provided",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "empty payload",
			user:       user1,
			expecErr:   "request had empty body",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "malformed payload",
			user:       user1,
			payload:    bytes.NewBuffer([]byte(`{"name":"Savings","cur`)),
			expecErr:   "unable to decode json: unexpected EOF",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "not found on wrong user",
			user:       user2,
			payload:    bytes.NewBuffer([]byte(`{"name":"Savings","currency":"USD","type":"cash"}`)),
			expecErr:   "unable to update account in DB: account not found",
			expectCode: http.StatusNotFound,
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

			// Create a new account to get an ID
			accountId, err := h.Store.CreateAccount(context.Background(),
				finance.Account{Name: "Initial", Currency: currency.USD}, user1)
			if err != nil {
				t.Fatal(err)
			}

			req, _ := http.NewRequest("PATCH", "/api/accounts/"+strconv.Itoa(int(accountId)), tc.payload)
			recorder := httptest.NewRecorder()
			handler := h.UpdateAccount(accountId, tc.user)
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

func TestAccountHandler_Delete(t *testing.T) {
	tcs := []struct {
		name       string
		user       string
		deleteID   uint
		expectErr  string
		expectCode int
	}{
		{
			name:       "successful deletion",
			user:       user1,
			expectCode: http.StatusOK,
		},
		{
			name:       "missing user",
			user:       "",
			expectErr:  "unable to get account: user not provided",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "non-existent account",
			user:       user1,
			deleteID:   9999,
			expectErr:  finance.AccountNotFoundErr.Error(),
			expectCode: http.StatusNotFound,
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

			// Create an account if needed
			if tc.deleteID == 0 {
				acctID, err := h.Store.CreateAccount(context.Background(), finance.Account{Name: "Test", Currency: currency.USD}, tc.user)
				if err != nil {
					t.Fatal(err)
				}
				tc.deleteID = acctID
			}

			req, _ := http.NewRequest("DELETE", "/api/accounts/"+strconv.Itoa(int(tc.deleteID)), nil)
			recorder := httptest.NewRecorder()

			handler := h.DeleteAccount(tc.deleteID, tc.user)
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
				}

				_, err := h.Store.GetAccount(context.Background(), tc.deleteID, tc.user)
				if err == nil {
					t.Fatalf("expected NotFoundErr, but got account")
				}
			}
		})
	}
}

func TestAccountHandler_List(t *testing.T) {
	tcs := []struct {
		name       string
		user       string
		expectCode int
		expectErr  string
		want       listResponse
	}{
		{
			name:       "successful request",
			user:       user1,
			expectCode: http.StatusOK,
			want: listResponse{
				Items: []accountPayload{
					{Name: "Main", Currency: "USD", Type: "bank"},
					{Name: "Under the bed", Currency: "EUR", Type: "cash"},
				},
			},
		},
		{
			name:       "missing user",
			user:       "",
			expectCode: http.StatusBadRequest,
			expectErr:  "unable to list accounts: user not provided",
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
			req, _ := http.NewRequest("GET", "/api/accounts", nil)
			handler := h.ListAccounts(tc.user)
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
				}

				got := listResponse{}
				err = json.NewDecoder(recorder.Body).Decode(&got)
				if err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(got, tc.want, cmpopts.IgnoreFields(accountPayload{}, "Id")); diff != "" {
					t.Errorf("unexpected result (-want +got):\n%s", diff)
				}
			}
		})
	}
}

const inMemorySqlite = "file::memory:?cache=shared"

var sampleAccounts = []finance.Account{
	{Name: "Main", Currency: currency.USD, Type: finance.Bank},
	{Name: "Under the bed", Currency: currency.EUR, Type: finance.Cash},
}

var sampleEntries = []finance.Entry{
	{Description: "e1", Amount: 1, Type: finance.ExpenseEntry, Date: getTime("2025-01-01 00:00:00")}, // 0
}

func getTime(timeStr string) time.Time {
	// Parse the string based on the provided layout
	parsedTime, err := time.Parse("2006-01-02 15:04:05", timeStr)
	if err != nil {
		panic(fmt.Errorf("unable to parse time: %v", err))

	}
	return parsedTime
}

const (
	user1 = "user1"
	user2 = "user2"
)

func SampleHandler() (*Handler, *gorm.DB, error) {

	db, err := gorm.Open(sqlite.Open(inMemorySqlite), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		return nil, nil, err
	}

	fiStore, err := finance.New(db)
	if err != nil {
		return nil, nil, err
	}

	for _, item := range sampleAccounts {
		_, err = fiStore.CreateAccount(context.Background(), item, user1)
		if err != nil {
			return nil, nil, err
		}
	}

	for _, entry := range sampleEntries {
		_, err = fiStore.CreateEntry(context.Background(), entry, user1)
		if err != nil {
			return nil, nil, err
		}
	}

	bkmh := Handler{
		Store: fiStore,
	}
	return &bkmh, db, nil

}
