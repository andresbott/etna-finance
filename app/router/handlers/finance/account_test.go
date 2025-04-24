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

func TestCreateAccountProvider(t *testing.T) {
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
		{
			name:       "error on missing information",
			userId:     "user123",
			payload:    bytes.NewBuffer([]byte(`{"currency":"USD", "type":"cash"}`)),
			expecErr:   "name cannot be empty",
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
			req, _ := http.NewRequest("POST", "/api/accountprovider", tc.payload)
			handler := h.CreateAccountProvider(tc.userId)
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

func TestAccountHandler_UpdateAccountProvider(t *testing.T) {
	tcs := []struct {
		name       string
		user       string
		payload    io.Reader
		expecErr   string
		expectCode int
	}{
		{
			name:       "successful request",
			user:       tenant1,
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
			user:       tenant1,
			expecErr:   "request had empty body",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "malformed payload",
			user:       tenant1,
			payload:    bytes.NewBuffer([]byte(`{"name":"Savings","cur`)),
			expecErr:   "unable to decode json: unexpected EOF",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "not found on wrong user",
			user:       tenant2,
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
			accountId, err := h.store.CreateAccount(context.Background(),
				finance.Account{Name: "Initial", Currency: currency.USD}, tenant1)
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

func TestAccountHandler_DeleteAccountProvider(t *testing.T) {
	tcs := []struct {
		name       string
		user       string
		deleteID   uint
		expectErr  string
		expectCode int
	}{
		{
			name:       "successful deletion",
			user:       tenant1,
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
			user:       tenant1,
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
				acctID, err := h.store.CreateAccount(context.Background(), finance.Account{Name: "Test", Currency: currency.USD}, tc.user)
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

				_, err := h.store.GetAccount(context.Background(), tc.deleteID, tc.user)
				if err == nil {
					t.Fatalf("expected NotFoundErr, but got account")
				}
			}
		})
	}
}

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

func TestAccountHandler_UpdateAccount(t *testing.T) {
	tcs := []struct {
		name       string
		user       string
		payload    io.Reader
		expecErr   string
		expectCode int
	}{
		{
			name:       "successful request",
			user:       tenant1,
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
			user:       tenant1,
			expecErr:   "request had empty body",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "malformed payload",
			user:       tenant1,
			payload:    bytes.NewBuffer([]byte(`{"name":"Savings","cur`)),
			expecErr:   "unable to decode json: unexpected EOF",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "not found on wrong user",
			user:       tenant2,
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
			accountId, err := h.store.CreateAccount(context.Background(),
				finance.Account{Name: "Initial", Currency: currency.USD}, tenant1)
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

func TestAccountHandler_DeleteAccount(t *testing.T) {
	tcs := []struct {
		name       string
		user       string
		deleteID   uint
		expectErr  string
		expectCode int
	}{
		{
			name:       "successful deletion",
			user:       tenant1,
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
			user:       tenant1,
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
				acctID, err := h.store.CreateAccount(context.Background(), finance.Account{Name: "Test", Currency: currency.USD}, tc.user)
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

				_, err := h.store.GetAccount(context.Background(), tc.deleteID, tc.user)
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
			user:       tenant1,
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

func getTime(timeStr string) time.Time {
	// Parse the string based on the provided layout
	parsedTime, err := time.Parse("2006-01-02 15:04:05", timeStr)
	if err != nil {
		panic(fmt.Errorf("unable to parse time: %v", err))

	}
	return parsedTime
}

func SampleHandler() (*Handler, *gorm.DB, error) {

	db, err := gorm.Open(sqlite.Open(inMemorySqlite), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		return nil, nil, err
	}

	store, err := finance.New(db)
	if err != nil {
		return nil, nil, err
	}

	bkmh := Handler{
		store: store,
	}
	return &bkmh, db, nil
}

var sampleAccountProviders = []finance.AccountProvider{
	{Name: "provider1", Description: "provider1", Accounts: []finance.Account{}}, // 1
	{Name: "provider2", Description: "provider2", Accounts: []finance.Account{}}, // 2
	{Name: "provider3", Description: "provider3", Accounts: []finance.Account{}}, // 3 does not have accounts
}

var sampleAccounts = []finance.Account{
	{ID: 1, Name: "acc1", Currency: currency.EUR, Type: 0, AccountProviderID: 1},
	{ID: 2, Name: "acc2", Currency: currency.USD, Type: 0, AccountProviderID: 2},
	{ID: 3, Name: "acc3", Currency: currency.EUR, Type: 0, AccountProviderID: 1},
	{ID: 3, Name: "acc4", Currency: currency.EUR, Type: 0, AccountProviderID: 1},
	{ID: 4, Name: "acc5", Currency: currency.EUR, Type: 0, AccountProviderID: 1},
}

var sampleEntries = []finance.Entry{
	{Description: "e1", Amount: 1, Type: finance.ExpenseEntry, Date: getTime("2025-01-01 00:00:00")}, // 1
	{Description: "e2", Amount: 2, Type: finance.ExpenseEntry, Date: getTime("2025-01-02 00:00:00"),
		TargetAccountID: 1, TargetAccountName: "acc1"},
	{Description: "e3", Amount: 3, Type: finance.ExpenseEntry, Date: getTime("2025-01-03 00:00:00"),
		TargetAccountID: 2, TargetAccountName: "acc2"},
	{Description: "e4", Amount: 4, Type: finance.ExpenseEntry, Date: getTime("2025-01-04 00:00:00")},
	{Description: "e5", Amount: 5, Type: finance.ExpenseEntry, Date: getTime("2025-01-05 00:00:00"),
		TargetAccountID: 2, TargetAccountName: "acc2"},
	{Description: "e6", Amount: 6, Type: finance.ExpenseEntry, Date: getTime("2025-01-06 00:00:00"),
		TargetAccountID: 1, TargetAccountName: "acc1"},
	{Description: "e7", Amount: 7, Type: finance.ExpenseEntry, Date: getTime("2025-01-07 00:00:00")},
	{Description: "e8", Amount: 8, Type: finance.ExpenseEntry, Date: getTime("2025-01-08 00:00:00"),
		TargetAccountID: 2, TargetAccountName: "acc2"},
	{Description: "e9", Amount: 9, Type: finance.ExpenseEntry, Date: getTime("2025-01-09 00:00:00")},
	{Description: "e10", Amount: 10, Type: finance.TransferEntry, Date: getTime("2025-01-10 00:00:00"),
		TargetAccountID: 2, TargetAccountName: "acc2", OriginAccountID: 1, OriginAccountName: "acc1"},
	{Description: "e11", Amount: 10, Type: finance.ExpenseEntry, Date: getTime("2025-01-11 00:00:00")},
	{Description: "e12", Amount: 10, Type: finance.ExpenseEntry, Date: getTime("2025-01-12 00:00:00")},
	{Description: "e13", Amount: 10, Type: finance.ExpenseEntry, Date: getTime("2025-01-13 00:00:00")},
	{Description: "e14", Amount: 10, Type: finance.ExpenseEntry, Date: getTime("2025-01-14 00:00:00")},
	{Description: "e14", Amount: 10, Type: finance.ExpenseEntry, Date: getTime("2025-01-15 00:00:00")},
	{Description: "e15", Amount: 10, Type: finance.ExpenseEntry, Date: getTime("2025-01-16 00:00:00")},
}

const (
	tenant1     = "tenant1"
	tenant2     = "tenant2"
	emptyTenant = "tenantEmpty"
)

func sampleData(t *testing.T, store *finance.Store) {
	ctx := context.Background()
	// =========================================
	// create accounts providers
	// =========================================
	provider1, err := store.CreateAccountProvider(ctx, sampleAccountProviders[0], tenant1)
	if err != nil {
		t.Fatalf("error creating provider 1: %v", err)
	}
	provider2, err := store.CreateAccountProvider(ctx, sampleAccountProviders[1], tenant1)
	if err != nil {
		t.Fatalf("error creating provider 2: %v", err)
	}
	provider3, err := store.CreateAccountProvider(ctx, sampleAccountProviders[2], tenant1)
	if err != nil {
		t.Fatalf("error creating provider 2: %v", err)
	}
	_ = provider3

	// =========================================
	// create accounts
	// =========================================

	acc := sampleAccounts[0]
	acc.AccountProviderID = provider1
	account1, err := store.CreateAccount(ctx, acc, tenant1)
	if err != nil {
		t.Fatalf("error creating account 1: %v", err)
	}
	_ = account1

	acc = sampleAccounts[1]
	acc.AccountProviderID = provider2
	account2, err := store.CreateAccount(ctx, acc, tenant1)
	if err != nil {
		t.Fatalf("error creating account 2: %v", err)
	}
	_ = account2

	for i := 2; i < len(sampleAccounts); i++ {
		acc = sampleAccounts[i]
		acc.AccountProviderID = provider1
		_, err = store.CreateAccount(ctx, acc, tenant1)
		if err != nil {
			t.Fatalf("error creating account 1: %v", err)
		}
	}

	// this account will be owned by tenant 2 but linked to a provider of tenant 1
	acc = sampleAccounts[1]
	acc.AccountProviderID = provider1
	account3, err := store.CreateAccount(ctx, acc, tenant2)
	if err != nil {
		t.Fatalf("error creating account 2: %v", err)
	}
	_ = account3

	// =========================================
	// create entries
	// =========================================

	for _, entry := range sampleEntries {
		_, err = store.CreateEntry(context.Background(), entry, tenant1)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, entry := range sampleEntries[9:16] {
		_, err = store.CreateEntry(context.Background(), entry, tenant2)
		if err != nil {
			t.Fatal(err)
		}
	}

}
