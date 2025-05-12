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
			name:       "empty tenant",
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
			h, end := SampleHandler(t)
			defer end()

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

func TestUpdateAccountProvider(t *testing.T) {
	tcs := []struct {
		name       string
		user       string
		payload    io.Reader
		expectErr  string
		expectCode int
	}{
		{
			name:       "successful request",
			user:       tenant1,
			payload:    bytes.NewBuffer([]byte(`{"name":"Savings","currency":"USD","type":"cash"}`)),
			expectCode: http.StatusOK,
		},
		{
			name:       "payload with wrong fields",
			user:       tenant1,
			payload:    bytes.NewBuffer([]byte(`{"currency":"USD","type":"cash"}`)),
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "missing user",
			user:       "",
			payload:    bytes.NewBuffer([]byte(`{"name":"Savings"}`)),
			expectErr:  "unable to update account: user not provided",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "empty payload",
			user:       tenant1,
			expectErr:  "request had empty body",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "malformed payload",
			user:       tenant1,
			payload:    bytes.NewBuffer([]byte(`{"name":"Savings","cur`)),
			expectErr:  "unable to decode json: unexpected EOF",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "not found on wrong user",
			user:       tenant2,
			payload:    bytes.NewBuffer([]byte(`{"name":"Savings","currency":"USD","type":"cash"}`)),
			expectErr:  "unable to update account provider in DB: account provider not found",
			expectCode: http.StatusNotFound,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleHandler(t)
			defer end()

			req, _ := http.NewRequest("PATCH", "/api/providers/1", tc.payload)
			recorder := httptest.NewRecorder()
			handler := h.UpdateAccountProvider(1, tc.user)
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
			}
		})
	}
}

func TestDeleteAccountProvider(t *testing.T) {
	tcs := []struct {
		name       string
		user       string
		deleteID   uint
		expectErr  string
		expectCode int
	}{
		{
			name:       "successful deletion",
			deleteID:   3, // id 3 does not have accounts associated
			user:       tenant1,
			expectCode: http.StatusOK,
		},
		{
			name:       "error when providers still has accounts",
			deleteID:   1, // id 3 does not have accounts associated
			user:       tenant1,
			expectErr:  "unable to delete account provider: account constraint violation",
			expectCode: http.StatusConflict,
		},
		{
			name:       "missing user",
			user:       "",
			deleteID:   1, // id 3 does not have accounts associated
			expectErr:  "unable to get account: user not provided",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "wrong user",
			deleteID:   1, // id 3 does not have accounts associated
			user:       emptyTenant,
			expectErr:  finance.ErrAccountProviderNotFound.Error(),
			expectCode: http.StatusNotFound,
		},
		{
			name:       "non-existent account",
			user:       tenant1,
			deleteID:   9999,
			expectErr:  finance.ErrAccountProviderNotFound.Error(),
			expectCode: http.StatusNotFound,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleHandler(t)
			defer end()
			req, _ := http.NewRequest("DELETE", "/api/accounts/"+strconv.FormatUint(uint64(tc.deleteID), 10), nil)
			recorder := httptest.NewRecorder()

			handler := h.DeleteAccountProvider(tc.deleteID, tc.user)
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

				_, err := h.Store.GetAccountProvider(context.Background(), tc.deleteID, tc.user)
				if err == nil {
					t.Fatalf("expected NotFoundErr, but account provider still exists")
				}
			}
		})
	}
}

func TestListAccountProvider(t *testing.T) {

	tenant1Accounts := []accountProviderPayload{
		{Id: 1, Name: "provider1", Description: "provider1", Accounts: []accountPayload{
			{Id: 1, Name: "acc1", Currency: "EUR", Type: "cash"},
			{Id: 3, Name: "acc3", Currency: "EUR", Type: "cash"},
			{Id: 4, Name: "acc4", Currency: "EUR", Type: "cash"},
			{Id: 5, Name: "acc5", Currency: "EUR", Type: "cash"},
		}},
		{Id: 2, Name: "provider2", Description: "provider2", Accounts: []accountPayload{
			{Id: 2, Name: "acc2", Currency: "USD", Type: "cash"},
		}},
		{Id: 3, Name: "provider3", Description: "provider3", Accounts: []accountPayload{}},
	}

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
			want:       listResponse{Items: tenant1Accounts},
		},
		{
			name:       "missing user",
			user:       "",
			expectCode: http.StatusBadRequest,
			expectErr:  "unable to list accounts: user not provided",
		},
		{
			name:       "empty user",
			user:       emptyTenant,
			expectCode: http.StatusOK,
			want:       listResponse{[]accountProviderPayload{}},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleHandler(t)
			defer end()

			recorder := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/accounts", nil)
			handler := h.ListAccountProviders(tc.user)
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
				err := json.NewDecoder(recorder.Body).Decode(&got)
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

func TestCreateAccount(t *testing.T) {
	tcs := []struct {
		name       string
		tenant     string
		payload    io.Reader
		expectErr  string
		expectCode int
	}{
		{
			name:       "successful request",
			tenant:     tenant1,
			payload:    bytes.NewBuffer([]byte(`{ "name":"Savings", "currency":"USD", "type":"cash", "providerId":1 }`)),
			expectCode: http.StatusOK,
		},
		{
			name:       "empty tenant",
			tenant:     "",
			payload:    bytes.NewBuffer([]byte(`{"name":"Savings", "currency":"USD", "type":"cash"}`)),
			expectErr:  "unable to create account: user not provided",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "wrong tenant",
			tenant:     emptyTenant,
			payload:    bytes.NewBuffer([]byte(`{"name":"Savings", "currency":"USD", "type":"cash",  "providerId":1 }`)),
			expectErr:  "account provider ID not found",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "empty payload",
			tenant:     tenant1,
			payload:    nil,
			expectErr:  "request had empty body",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "malformed payload",
			tenant:     "user123",
			payload:    bytes.NewBuffer([]byte(`{"name":"Savings"`)),
			expectErr:  "unable to decode json: unexpected EOF",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "malformed payload types",
			tenant:     "user123",
			payload:    bytes.NewBuffer([]byte(`{ "name":"Savings", "currency":"USD", "type":"cash", "providerId":"1" }`)),
			expectErr:  "unable to decode json: json: cannot unmarshal string into Go struct field accountPayload.providerId of type uint",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleHandler(t)
			defer end()

			recorder := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/accounts", tc.payload)
			handler := h.CreateAccount(tc.tenant)
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

func TestUpdateAccount(t *testing.T) {
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
			h, end := SampleHandler(t)
			defer end()

			// Create a new account to get an ID
			accountId, err := h.Store.CreateAccount(context.Background(),
				finance.Account{Name: "Initial", Currency: currency.USD, AccountProviderID: 1}, tenant1)
			if err != nil {
				t.Fatal(err)
			}

			req, _ := http.NewRequest("PATCH", "/api/accounts/"+strconv.FormatUint(uint64(accountId), 10), tc.payload)
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

func TestDeleteAccount(t *testing.T) {
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
			deleteID:   1,
			expectCode: http.StatusOK,
		},
		{
			name:       "missing user",
			user:       "",
			deleteID:   2,
			expectErr:  "unable to get account: user not provided",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "non-existent account",
			user:       tenant1,
			deleteID:   9999,
			expectErr:  finance.ErrAccountNotFound.Error(),
			expectCode: http.StatusNotFound,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleHandler(t)
			defer end()

			req, _ := http.NewRequest("DELETE", "/api/accounts/"+strconv.FormatUint(uint64(tc.deleteID), 10), nil)
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

const inMemorySqlite = "file::memory:?cache=shared"

func getTime(timeStr string) time.Time {
	// Parse the string based on the provided layout
	parsedTime, err := time.Parse("2006-01-02 15:04:05", timeStr)
	if err != nil {
		panic(fmt.Errorf("unable to parse time: %v", err))

	}
	return parsedTime
}

func SampleHandler(t *testing.T) (*Handler, func()) {

	db, err := gorm.Open(sqlite.Open(inMemorySqlite), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("unable to connect to sqlite: %v", err)
	}

	store, err := finance.New(db)
	if err != nil {
		t.Fatalf("unable to connect to finance: %v", err)
	}
	sampleData(t, store)

	bkmh := Handler{
		Store: store,
	}

	closeFn := func() {
		uDb, err := db.DB()
		if err != nil {
			t.Fatalf("unable to get underlying DB: %v", err)
		}

		err = uDb.Close()
		if err != nil {
			t.Fatalf("unable to close underlying DB: %v", err)
		}
	}
	return &bkmh, closeFn
}

var sampleAccountProviders = []finance.AccountProvider{
	{Name: "provider1", Description: "provider1", Accounts: []finance.Account{}}, // 1
	{Name: "provider2", Description: "provider2", Accounts: []finance.Account{}}, // 2
	{Name: "provider3", Description: "provider3", Accounts: []finance.Account{}}, // 3 does not have accounts
}

var sampleAccountProviders2 = []finance.AccountProvider{
	{Name: "provider4_tenant2", Description: "provider4t2", Accounts: []finance.Account{}}, // 4
}

var sampleAccounts = []finance.Account{
	{ID: 1, Name: "acc1", Currency: currency.EUR, Type: finance.Cash, AccountProviderID: 1},
	{ID: 2, Name: "acc2", Currency: currency.USD, Type: finance.Cash, AccountProviderID: 2},
	{ID: 3, Name: "acc3", Currency: currency.EUR, Type: finance.Cash, AccountProviderID: 1},
	{ID: 3, Name: "acc4", Currency: currency.EUR, Type: finance.Cash, AccountProviderID: 1},
	{ID: 4, Name: "acc5", Currency: currency.EUR, Type: finance.Cash, AccountProviderID: 1},
}

var sampleEntries = []finance.Entry{
	{Description: "e1", TargetAmount: 1, Type: finance.ExpenseEntry, TargetAccountID: 1, Date: getTime("2025-01-01 00:00:00")}, // 1
	{Description: "e2", TargetAmount: 2, Type: finance.ExpenseEntry, Date: getTime("2025-01-02 00:00:00"),
		TargetAccountID: 1, TargetAccountName: "acc1"},
	{Description: "e3", TargetAmount: 3, Type: finance.ExpenseEntry, Date: getTime("2025-01-03 00:00:00"),
		TargetAccountID: 2, TargetAccountName: "acc2"},
	{Description: "e4", TargetAmount: 4, Type: finance.ExpenseEntry, TargetAccountID: 1, Date: getTime("2025-01-04 00:00:00")},
	{Description: "e5", TargetAmount: 5, Type: finance.ExpenseEntry, Date: getTime("2025-01-05 00:00:00"),
		TargetAccountID: 2, TargetAccountName: "acc2"},
	{Description: "e6", TargetAmount: 6, Type: finance.ExpenseEntry, Date: getTime("2025-01-06 00:00:00"),
		TargetAccountID: 1, TargetAccountName: "acc1"},
	{Description: "e7", TargetAmount: 7, Type: finance.ExpenseEntry, TargetAccountID: 1, Date: getTime("2025-01-07 00:00:00")},
	{Description: "e8", TargetAmount: 8, Type: finance.ExpenseEntry, Date: getTime("2025-01-08 00:00:00"),
		TargetAccountID: 2, TargetAccountName: "acc2"},
	{Description: "e9", TargetAmount: 9, Type: finance.ExpenseEntry, TargetAccountID: 1, Date: getTime("2025-01-09 00:00:00")},
	{Description: "e10", TargetAmount: 10, OriginAmount: 4.5, Type: finance.TransferEntry, Date: getTime("2025-01-10 00:00:00"),
		TargetAccountID: 2, TargetAccountName: "acc2", OriginAccountID: 1, OriginAccountName: "acc1"},
	{Description: "e11", TargetAmount: 10, Type: finance.ExpenseEntry, TargetAccountID: 1, TargetAccountName: "acc1", Date: getTime("2025-01-11 00:00:00")},
	{Description: "e12", TargetAmount: 10, Type: finance.ExpenseEntry, TargetAccountID: 1, Date: getTime("2025-01-12 00:00:00")},
	{Description: "e13", TargetAmount: 10, Type: finance.ExpenseEntry, TargetAccountID: 1, Date: getTime("2025-01-13 00:00:00")},
	{Description: "e14", TargetAmount: 10, Type: finance.ExpenseEntry, TargetAccountID: 1, Date: getTime("2025-01-14 00:00:00")},
	{Description: "e14", TargetAmount: 10, Type: finance.ExpenseEntry, TargetAccountID: 1, Date: getTime("2025-01-15 00:00:00")},
	{Description: "e15", TargetAmount: 10, Type: finance.ExpenseEntry, TargetAccountID: 1, Date: getTime("2025-01-16 00:00:00")},
}

var sampleAccounts2 = []finance.Account{
	{ID: 6, Name: "acc1tenant2", Currency: currency.EUR, Type: 0, AccountProviderID: 4},
}
var sampleEntries2 = []finance.Entry{
	{Description: "t2e13", TargetAmount: 10, Type: finance.ExpenseEntry, TargetAccountID: 6, Date: getTime("2025-01-13 00:00:00")},
	{Description: "t2e14", TargetAmount: 10, Type: finance.ExpenseEntry, TargetAccountID: 6, Date: getTime("2025-01-14 00:00:00")},
	{Description: "t2e15", TargetAmount: 10, Type: finance.ExpenseEntry, TargetAccountID: 6, TargetAccountName: "acc1tenant2", Date: getTime("2025-01-15 00:00:00")},
	{Description: "t2e16", TargetAmount: 10, Type: finance.ExpenseEntry, TargetAccountID: 6, TargetAccountName: "acc1tenant2", Date: getTime("2025-01-16 00:00:00")},
	{Description: "t2e17", TargetAmount: 10, Type: finance.ExpenseEntry, TargetAccountID: 6, Date: getTime("2025-02-17 00:00:00")},
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

	provider4, err := store.CreateAccountProvider(ctx, sampleAccountProviders2[0], tenant2)
	if err != nil {
		t.Fatalf("error creating provider 2: %v", err)
	}

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
	for i := 0; i < len(sampleAccounts2); i++ {
		acc2 := sampleAccounts2[i]
		acc2.AccountProviderID = provider4
		_, err := store.CreateAccount(ctx, acc2, tenant2)
		if err != nil {
			t.Fatalf("error creating account 1: %v", err)
		}
	}

	// =========================================
	// create entries
	// =========================================

	for _, entry := range sampleEntries {
		_, err = store.CreateEntry(context.Background(), entry, tenant1)
		if err != nil {
			t.Fatal(err)
		}
	}

	for _, entry := range sampleEntries2 {
		_, err = store.CreateEntry(context.Background(), entry, tenant2)
		if err != nil {
			t.Fatal(err)
		}
	}

}
