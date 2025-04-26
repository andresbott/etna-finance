package finance

import (
	"context"
	"github.com/go-bumbu/testdbs"
	"golang.org/x/text/currency"
	"os"
	"testing"
	"time"
)

// TestMain modifies how test are run,
// it makes sure that the needed DBs are ready and does cleanup in the end.
func TestMain(m *testing.M) {
	testdbs.InitDBS()
	// main block that runs tests
	code := m.Run()
	_ = testdbs.Clean()
	os.Exit(code)
}

func ptr[T any](v T) *T {
	return &v
}

var date1 = time.Date(2025, time.March, 15, 0, 0, 0, 0, time.UTC)
var date2 = time.Date(2025, time.March, 16, 0, 0, 0, 0, time.UTC)

var sampleAccountProviders = []AccountProvider{
	{Name: "provider1", Description: "provider1", Accounts: []Account{}}, // 1
	{Name: "provider2", Description: "provider2", Accounts: []Account{}}, // 2
	{Name: "provider3", Description: "provider3", Accounts: []Account{}}, // 3 does not have accounts
}

var sampleAccounts = []Account{
	{ID: 1, Name: "acc1", Currency: currency.EUR, Type: 0, AccountProviderID: 1},
	{ID: 2, Name: "acc2", Currency: currency.USD, Type: 0, AccountProviderID: 2},
	{ID: 3, Name: "acc3", Currency: currency.EUR, Type: 0, AccountProviderID: 1},
	{ID: 3, Name: "acc4", Currency: currency.EUR, Type: 0, AccountProviderID: 1},
	{ID: 4, Name: "acc5", Currency: currency.EUR, Type: 0, AccountProviderID: 1},
}

var sampleEntries = []Entry{
	{Description: "e1", Amount: 1, Type: ExpenseEntry, Date: getTime("2025-01-01 00:00:00")}, // 1
	{Description: "e2", Amount: 2, Type: ExpenseEntry, Date: getTime("2025-01-02 00:00:00"),
		TargetAccountID: 1, TargetAccountName: "acc1"},
	{Description: "e3", Amount: 3, Type: ExpenseEntry, Date: getTime("2025-01-03 00:00:00"),
		TargetAccountID: 2, TargetAccountName: "acc2"},
	{Description: "e4", Amount: 4, Type: ExpenseEntry, Date: getTime("2025-01-04 00:00:00")},
	{Description: "e5", Amount: 5, Type: ExpenseEntry, Date: getTime("2025-01-05 00:00:00"),
		TargetAccountID: 2, TargetAccountName: "acc2"},
	{Description: "e6", Amount: 6, Type: ExpenseEntry, Date: getTime("2025-01-06 00:00:00"),
		TargetAccountID: 1, TargetAccountName: "acc1"},
	{Description: "e7", Amount: 7, Type: ExpenseEntry, Date: getTime("2025-01-07 00:00:00")},
	{Description: "e8", Amount: 8, Type: ExpenseEntry, Date: getTime("2025-01-08 00:00:00"),
		TargetAccountID: 2, TargetAccountName: "acc2"},
	{Description: "e9", Amount: 9, Type: ExpenseEntry, Date: getTime("2025-01-09 00:00:00")},
	{Description: "e10", Amount: 10, Type: TransferEntry, Date: getTime("2025-01-10 00:00:00"),
		TargetAccountID: 2, TargetAccountName: "acc2", OriginAccountID: 1, OriginAccountName: "acc1"},
	{Description: "e11", Amount: 10, Type: ExpenseEntry, Date: getTime("2025-01-11 00:00:00")},
	{Description: "e12", Amount: 10, Type: ExpenseEntry, Date: getTime("2025-01-12 00:00:00")},
	{Description: "e13", Amount: 10, Type: ExpenseEntry, Date: getTime("2025-01-13 00:00:00")},
	{Description: "e14", Amount: 10, Type: ExpenseEntry, Date: getTime("2025-01-14 00:00:00")},
	{Description: "e14", Amount: 10, Type: ExpenseEntry, Date: getTime("2025-01-15 00:00:00")},
	{Description: "e15", Amount: 10, Type: ExpenseEntry, Date: getTime("2025-01-16 00:00:00")},
}

const (
	tenant1     = "tenant1"
	tenant2     = "tenant2"
	emptyTenant = "tenantEmpty"
)

func sampleData(t *testing.T, store *Store) {
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
