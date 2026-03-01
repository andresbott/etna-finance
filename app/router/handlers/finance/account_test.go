package finance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/andresbott/etna/internal/accounting"
	"github.com/andresbott/etna/internal/marketdata"
	"github.com/glebarez/sqlite"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/text/currency"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
			name:       "successful request with icon",
			userId:     "user123",
			payload:    bytes.NewBuffer([]byte(`{ "name":"Savings", "icon":"bank-icon", "currency":"USD", "type":"cash"}`)),
			expectCode: http.StatusOK,
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
			handler := h.CreateAccountProvider()
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

				acc := accounting.Account{}
				err := json.NewDecoder(recorder.Body).Decode(&acc)
				if err != nil {
					t.Fatal(err)
				}
				if acc.ID == 0 {
					t.Error("returned account id is empty")
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
			name:       "successful request with icon update",
			user:       tenant1,
			payload:    bytes.NewBuffer([]byte(`{"icon":"new-bank-icon"}`)),
			expectCode: http.StatusOK,
		},
		{
			name:       "payload with wrong fields",
			user:       tenant1,
			payload:    bytes.NewBuffer([]byte(`{"currency":"USD","type":"cash"}`)),
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
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleHandler(t)
			defer end()

			req, _ := http.NewRequest("PATCH", "/api/providers/1", tc.payload)
			recorder := httptest.NewRecorder()
			handler := h.UpdateAccountProvider(1)
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
			deleteID:   1,
			user:       tenant1,
			expectErr:  "unable to delete account provider: account constraint violation",
			expectCode: http.StatusConflict,
		},
		{
			name:       "non-existent account",
			user:       tenant1,
			deleteID:   9999,
			expectErr:  accounting.ErrAccountProviderNotFound.Error(),
			expectCode: http.StatusNotFound,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleHandler(t)
			defer end()
			req, _ := http.NewRequest("DELETE", "/api/accounts/"+strconv.FormatUint(uint64(tc.deleteID), 10), nil)
			recorder := httptest.NewRecorder()

			handler := h.DeleteAccountProvider(tc.deleteID)
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

				_, err := h.Store.GetAccountProvider(t.Context(), tc.deleteID)
				if err == nil {
					t.Fatalf("expected NotFoundErr, but account provider still exists")
				}
			}
		})
	}
}

func TestListAccountProvider(t *testing.T) {

	allProviders := []accountProviderPayload{
		{Id: 1, Name: "provider1", Description: "provider1", Icon: "bank", Accounts: []accountPayload{
			{Id: 1, Name: "acc1", Icon: "euro", Currency: "EUR", Type: "cash"},
			{Id: 3, Name: "acc3", Icon: "", Currency: "EUR", Type: "cash"},
			{Id: 4, Name: "acc4", Icon: "savings", Currency: "EUR", Type: "cash"},
			{Id: 5, Name: "acc5", Icon: "chart", Currency: "EUR", Type: "cash"},
		}},
		{Id: 2, Name: "provider2", Description: "provider2", Icon: "wallet", Accounts: []accountPayload{
			{Id: 2, Name: "acc2", Icon: "dollar", Currency: "USD", Type: "cash"},
		}},
		{Id: 3, Name: "provider3", Description: "provider3", Icon: "", Accounts: []accountPayload{}},
		{Id: 4, Name: "provider4_tenant2", Description: "provider4t2", Icon: "credit", Accounts: []accountPayload{
			{Id: 6, Name: "acc1tenant2", Icon: "foreign", Currency: "EUR", Type: "cash"},
		}},
	}

	tcs := []struct {
		name       string
		user       string
		expectCode int
		expectErr  string
		want       listResponse
	}{
		{
			name:       "list all account providers",
			user:       tenant1,
			expectCode: http.StatusOK,
			want:       listResponse{Items: allProviders},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleHandler(t)
			defer end()

			recorder := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/accounts", nil)
			handler := h.ListAccountProviders()
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
			name:       "successful request with icon",
			tenant:     tenant1,
			payload:    bytes.NewBuffer([]byte(`{ "name":"Savings", "icon":"wallet-icon", "currency":"USD", "type":"cash", "providerId":1 }`)),
			expectCode: http.StatusOK,
		},
		{
			name:       "successful request with type unvested",
			tenant:     tenant1,
			payload:    bytes.NewBuffer([]byte(`{ "name":"RSU account", "type":"unvested", "currency":"USD", "providerId":1 }`)),
			expectCode: http.StatusOK,
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
			handler := h.CreateAccount()
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

				acc := accountPayload{}
				err := json.NewDecoder(recorder.Body).Decode(&acc)
				if err != nil {
					t.Fatal(err)
				}
				if acc.Id == 0 {
					t.Error("returned account id is empty")
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
			name:       "successful request with icon update",
			user:       tenant1,
			payload:    bytes.NewBuffer([]byte(`{"icon":"new-wallet-icon"}`)),
			expectCode: http.StatusOK,
		},
		{
			name:       "successful request with type unvested",
			user:       tenant1,
			payload:    bytes.NewBuffer([]byte(`{"type":"unvested"}`)),
			expectCode: http.StatusOK,
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
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleHandler(t)
			defer end()

			// Create a new account to get an id
			accountId, err := h.Store.CreateAccount(t.Context(),
				accounting.Account{Name: "Initial", Currency: currency.USD, AccountProviderID: 1})
			if err != nil {
				t.Fatal(err)
			}

			req, _ := http.NewRequest("PATCH", "/api/accounts/"+strconv.FormatUint(uint64(accountId), 10), tc.payload)
			recorder := httptest.NewRecorder()
			handler := h.UpdateAccount(accountId)
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
			deleteID:   4, // account 4 does not have any entries linked,
			expectCode: http.StatusOK,
		},
		{
			name:       "expect error",
			user:       tenant1,
			deleteID:   1,
			expectErr:  "unable to delete account: account still contains referenced transactions",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "non-existent account",
			user:       tenant1,
			deleteID:   9999,
			expectErr:  accounting.ErrAccountNotFound.Error(),
			expectCode: http.StatusNotFound,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleHandler(t)
			defer end()

			req, _ := http.NewRequest("DELETE", "/api/accounts/"+strconv.FormatUint(uint64(tc.deleteID), 10), nil)
			recorder := httptest.NewRecorder()

			handler := h.DeleteAccount(tc.deleteID)
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

				_, err := h.Store.GetAccount(t.Context(), tc.deleteID)
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

	mktStore, err := marketdata.NewStore(db)
	if err != nil {
		t.Fatalf("unable to create marketdata store: %v", err)
	}
	store, err := accounting.NewStore(db, mktStore)
	if err != nil {
		t.Fatalf("unable to connect to finance: %v", err)
	}
	sampleData(t, store)

	bkmh := Handler{
		Store:           store,
		InstrumentStore: mktStore,
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

var sampleAccountProviders = []accounting.AccountProvider{
	{Name: "provider1", Description: "provider1", Icon: "bank", Accounts: []accounting.Account{}},   // 1
	{Name: "provider2", Description: "provider2", Icon: "wallet", Accounts: []accounting.Account{}}, // 2
	{Name: "provider3", Description: "provider3", Icon: "", Accounts: []accounting.Account{}},       // 3 does not have accounts
}

var sampleAccountProviders2 = []accounting.AccountProvider{
	{Name: "provider4_tenant2", Description: "provider4t2", Icon: "credit", Accounts: []accounting.Account{}}, // 4
}

var sampleAccounts = []accounting.Account{
	{ID: 1, Name: "acc1", Icon: "euro", Currency: currency.EUR, Type: accounting.CashAccountType, AccountProviderID: 1},
	{ID: 2, Name: "acc2", Icon: "dollar", Currency: currency.USD, Type: accounting.CashAccountType, AccountProviderID: 2},
	{ID: 3, Name: "acc3", Icon: "", Currency: currency.EUR, Type: accounting.CashAccountType, AccountProviderID: 1},
	{ID: 3, Name: "acc4", Icon: "savings", Currency: currency.EUR, Type: accounting.CashAccountType, AccountProviderID: 1},
	{ID: 4, Name: "acc5", Icon: "chart", Currency: currency.EUR, Type: accounting.CashAccountType, AccountProviderID: 1},
}

var sampleCategories = []accounting.CategoryData{
	{Name: "Groceries", Icon: "pi-shopping-cart", Type: accounting.ExpenseCategory}, // 1
	{Name: "Bills", Icon: "pi-file", Type: accounting.ExpenseCategory},              // 2
	{Name: "Salary", Icon: "pi-wallet", Type: accounting.IncomeCategory},            // 3
	{Name: "Investments", Icon: "pi-chart-line", Type: accounting.IncomeCategory},   // 4
}

var sampleEntries = []accounting.Transaction{
	accounting.Expense{Description: "e1", Amount: 1, AccountID: 1, CategoryID: 1, Date: getTime("2025-01-01 00:00:00")},
	accounting.Expense{Description: "e2", Amount: 2, AccountID: 1, CategoryID: 1, Date: getTime("2025-01-02 00:00:00")},
	accounting.Expense{Description: "e3", Amount: 3, AccountID: 2, CategoryID: 2, Date: getTime("2025-01-03 00:00:00")},
	accounting.Expense{Description: "e4", Amount: 4, AccountID: 1, CategoryID: 1, Date: getTime("2025-01-04 00:00:00")},
	accounting.Expense{Description: "e5", Amount: 5, AccountID: 2, CategoryID: 2, Date: getTime("2025-01-05 00:00:00")},
	accounting.Expense{Description: "e6", Amount: 6, AccountID: 1, CategoryID: 1, Date: getTime("2025-01-06 00:00:00")},
	accounting.Expense{Description: "e7", Amount: 7, AccountID: 1, CategoryID: 2, Date: getTime("2025-01-07 00:00:00")},
	accounting.Expense{Description: "e8", Amount: 8, AccountID: 2, CategoryID: 1, Date: getTime("2025-01-08 00:00:00")},
	accounting.Expense{Description: "e9", Amount: 9, AccountID: 1, CategoryID: 2, Date: getTime("2025-01-09 00:00:00")},
	accounting.Expense{Description: "e10", Amount: 10, AccountID: 2, CategoryID: 1, Date: getTime("2025-01-10 00:00:00")},
	accounting.Expense{Description: "e11", Amount: 10, AccountID: 1, CategoryID: 2, Date: getTime("2025-01-11 00:00:00")},
	accounting.Expense{Description: "e12", Amount: 10, AccountID: 1, CategoryID: 1, Date: getTime("2025-01-12 00:00:00")},
	accounting.Expense{Description: "e13", Amount: 10, AccountID: 1, CategoryID: 2, Date: getTime("2025-01-13 00:00:00")},
	accounting.Expense{Description: "e14", Amount: 10, AccountID: 1, CategoryID: 1, Date: getTime("2025-01-14 00:00:00")},
	accounting.Expense{Description: "e14", Amount: 10, AccountID: 1, CategoryID: 2, Date: getTime("2025-01-15 00:00:00")},
	accounting.Expense{Description: "e15", Amount: 10, AccountID: 1, CategoryID: 1, Date: getTime("2025-01-16 00:00:00")},
	accounting.Income{Description: "i1", Amount: 10, AccountID: 1, CategoryID: 3, Date: getTime("2025-01-17 00:00:00")},
}

var sampleAccounts2 = []accounting.Account{
	{ID: 6, Name: "acc1tenant2", Icon: "foreign", Currency: currency.EUR, Type: accounting.CashAccountType, AccountProviderID: 4},
}
var sampleEntries2 = []accounting.Transaction{
	accounting.Expense{Description: "t2e13", Amount: 10, AccountID: 6, Date: getTime("2025-01-13 00:00:00")},
	accounting.Expense{Description: "t2e14", Amount: 10, AccountID: 6, Date: getTime("2025-01-14 00:00:00")},
	accounting.Expense{Description: "t2e15", Amount: 10, AccountID: 6, Date: getTime("2025-01-15 00:00:00")},
	accounting.Expense{Description: "t2e16", Amount: 10, AccountID: 6, Date: getTime("2025-01-16 00:00:00")},
	accounting.Expense{Description: "t2e17", Amount: 10, AccountID: 6, Date: getTime("2025-02-17 00:00:00")},
}

const (
	tenant1     = "tenant1"
	tenant2     = "tenant2"
	emptyTenant = "tenantEmpty"
)

func sampleData(t *testing.T, store *accounting.Store) {
	ctx := t.Context()

	// =========================================
	// create accounts providers
	// =========================================
	provider1, err := store.CreateAccountProvider(ctx, sampleAccountProviders[0])
	if err != nil {
		t.Fatalf("error creating provider 1: %v", err)
	}
	provider2, err := store.CreateAccountProvider(ctx, sampleAccountProviders[1])
	if err != nil {
		t.Fatalf("error creating provider 2: %v", err)
	}
	provider3, err := store.CreateAccountProvider(ctx, sampleAccountProviders[2])
	if err != nil {
		t.Fatalf("error creating provider 2: %v", err)
	}
	_ = provider3

	provider4, err := store.CreateAccountProvider(ctx, sampleAccountProviders2[0])
	if err != nil {
		t.Fatalf("error creating provider 2: %v", err)
	}

	// =========================================
	// create accounts
	// =========================================

	acc := sampleAccounts[0]
	acc.AccountProviderID = provider1
	account1, err := store.CreateAccount(ctx, acc)
	if err != nil {
		t.Fatalf("error creating account 1: %v", err)
	}
	_ = account1

	acc = sampleAccounts[1]
	acc.AccountProviderID = provider2
	account2, err := store.CreateAccount(ctx, acc)
	if err != nil {
		t.Fatalf("error creating account 2: %v", err)
	}
	_ = account2

	for i := 2; i < len(sampleAccounts); i++ {
		acc = sampleAccounts[i]
		acc.AccountProviderID = provider1
		_, err = store.CreateAccount(ctx, acc)
		if err != nil {
			t.Fatalf("error creating account 1: %v", err)
		}
	}
	for i := 0; i < len(sampleAccounts2); i++ {
		acc2 := sampleAccounts2[i]
		acc2.AccountProviderID = provider4
		_, err := store.CreateAccount(ctx, acc2)
		if err != nil {
			t.Fatalf("error creating account 1: %v", err)
		}
	}

	// =========================================
	// create categories
	// =========================================

	for _, cat := range sampleCategories {
		_, err = store.CreateCategory(ctx, cat, 0)
		if err != nil {
			t.Fatalf("error creating category: %v", err)
		}
	}

	// =========================================
	// create entries
	// =========================================

	for _, entry := range sampleEntries {
		_, err = store.CreateTransaction(t.Context(), entry)
		if err != nil {
			t.Fatal(err)
		}
	}

	// create an entry with time now for test purposes
	entry := accounting.Expense{Description: "now1", Amount: 1, AccountID: 1, Date: time.Now()}
	_, err = store.CreateTransaction(t.Context(), entry)
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range sampleEntries2 {
		_, err = store.CreateTransaction(t.Context(), entry)
		if err != nil {
			t.Fatal(err)
		}
	}

}
