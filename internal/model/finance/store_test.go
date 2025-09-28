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
var sampleAccountProviders2 = []AccountProvider{
	{Name: "provider4_tenant2", Description: "provider4t2", Accounts: []Account{}}, // 4
}

var sampleAccounts = []Account{
	{ID: 1, Name: "acc1", Currency: currency.EUR, Type: Cash, AccountProviderID: 1},
	{ID: 2, Name: "acc2", Currency: currency.USD, Type: Cash, AccountProviderID: 2},
	{ID: 3, Name: "acc3", Currency: currency.EUR, Type: 0, AccountProviderID: 1},
	{ID: 4, Name: "acc4", Currency: currency.EUR, Type: 0, AccountProviderID: 1},
	{ID: 5, Name: "acc5", Currency: currency.EUR, Type: 0, AccountProviderID: 1},
}

var sampleEntries = []Entry{
	{Description: "e0", TargetAmount: 1, Type: ExpenseEntry, TargetAccountID: 1, Date: getTime("2025-01-01 00:00:00")}, // 1
	{Description: "e1", TargetAmount: 2, Type: ExpenseEntry, Date: getTime("2025-01-02 00:00:00"), TargetAccountID: 1, TargetAccountName: "acc1"},
	{Description: "e2", TargetAmount: 3, Type: IncomeEntry, Date: getTime("2025-01-03 00:00:00"), TargetAccountID: 2, TargetAccountName: "acc2", CategoryId: 3},
	{Description: "e3", TargetAmount: -4.1, Type: ExpenseEntry, TargetAccountID: 1, Date: getTime("2025-01-04 00:00:00")},
	{Description: "e4", TargetAmount: 5, Type: ExpenseEntry, Date: getTime("2025-01-05 00:00:00"), TargetAccountID: 2, TargetAccountName: "acc2"},
	{Description: "e5", TargetAmount: 6, Type: ExpenseEntry, Date: getTime("2025-01-06 00:00:00"), TargetAccountID: 1, TargetAccountName: "acc1"},
	{Description: "e6", TargetAmount: 7, Type: ExpenseEntry, TargetAccountID: 1, Date: getTime("2025-01-07 00:00:00")},
	{Description: "e7", TargetAmount: -8.3, Type: ExpenseEntry, Date: getTime("2025-01-08 00:00:00"), TargetAccountID: 2, TargetAccountName: "acc2"},
	{Description: "e8", TargetAmount: 9, Type: ExpenseEntry, TargetAccountID: 1, Date: getTime("2025-01-09 00:00:00")},
	{Description: "e9", TargetAmount: 10, OriginAmount: 4.5, Type: TransferEntry, Date: getTime("2025-01-10 00:00:00"), TargetAccountID: 2, TargetAccountName: "acc2", OriginAccountID: 1, OriginAccountName: "acc1", CategoryId: 3},
	{Description: "e10", TargetAmount: -100.4, Type: ExpenseEntry, TargetAccountID: 4, TargetAccountName: "acc4", Date: getTime("2025-01-11 00:00:00"), CategoryId: 2},
	{Description: "e11", TargetAmount: 200, Type: ExpenseEntry, TargetAccountID: 4, TargetAccountName: "acc4", OriginAccountID: 5, OriginAccountName: "acc5", Date: getTime("2025-01-12 00:00:00"), CategoryId: 1},
	{Description: "e12", TargetAmount: 300, Type: ExpenseEntry, TargetAccountID: 1, Date: getTime("2025-01-13 00:00:00")},
	{Description: "e13", TargetAmount: 1000, Type: ExpenseEntry, TargetAccountID: 1, Date: getTime("2025-01-14 00:00:00"), TargetAccountName: "acc1", CategoryId: 4},
	{Description: "e14", TargetAmount: 2000, Type: ExpenseEntry, TargetAccountID: 1, Date: getTime("2025-01-15 00:00:00")},
	{Description: "e15", TargetAmount: 3000, Type: ExpenseEntry, TargetAccountID: 1, Date: getTime("2025-01-16 00:00:00")},
	{Description: "e16", TargetAmount: 550.5, Type: IncomeEntry, Date: getTime("2025-01-17 00:00:00"), TargetAccountID: 1, TargetAccountName: "acc1", CategoryId: 2},
}

var sampleAccounts2 = []Account{
	{ID: 6, Name: "acc1tenant2", Currency: currency.EUR, Type: 0, AccountProviderID: 4},
}

var sampleEntries2 = []Entry{
	{Description: "t2e13", TargetAmount: 10, Type: ExpenseEntry, TargetAccountID: 6, Date: getTime("2025-01-13 00:00:00")},
	{Description: "t2e14", TargetAmount: 10, Type: ExpenseEntry, TargetAccountID: 6, Date: getTime("2025-01-14 00:00:00")},
	{Description: "t2e15", TargetAmount: 10, Type: ExpenseEntry, TargetAccountID: 6, TargetAccountName: "acc1tenant2", Date: getTime("2025-01-15 00:00:00")},
	{Description: "t2e16", TargetAmount: 10, Type: ExpenseEntry, TargetAccountID: 6, TargetAccountName: "acc1tenant2", Date: getTime("2025-01-16 00:00:00")},
	{Description: "t2e17", TargetAmount: 10, Type: ExpenseEntry, TargetAccountID: 6, Date: getTime("2025-02-17 00:00:00")},
}

var sampleCategories = []struct {
	CategoryData
	parent uint
}{
	// income
	{CategoryData: CategoryData{Name: "in_top1", Type: IncomeCategory}, parent: 0}, // id 1
	{CategoryData: CategoryData{Name: "in_sub1", Type: IncomeCategory}, parent: 1}, // id 2
	{CategoryData: CategoryData{Name: "in_sub2", Type: IncomeCategory}, parent: 2}, // id 3
	{CategoryData: CategoryData{Name: "in_top2", Type: IncomeCategory}, parent: 0}, // id 4
	// expenses
	{CategoryData: CategoryData{Name: "ex_top1", Type: ExpenseCategory}, parent: 0}, // id 1
	{CategoryData: CategoryData{Name: "ex_sub1", Type: ExpenseCategory}, parent: 1}, // id 2
	{CategoryData: CategoryData{Name: "ex_sub2", Type: ExpenseCategory}, parent: 2}, // id 3
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
	// create categories
	// =========================================
	for _, category := range sampleCategories {
		_, err := store.CreateCategory(ctx, category.CategoryData, category.parent, tenant1)
		if err != nil {
			t.Fatalf("error creating income category: %v", err)
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
