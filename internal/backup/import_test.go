package backup

import (
	"testing"

	"github.com/andresbott/etna/internal/accounting"
	"github.com/glebarez/sqlite"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/text/currency"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestImportV1(t *testing.T) {

	db, err := gorm.Open(sqlite.Open("file:importDb?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("unable to connect to sqlite: %v", err)
	}
	store, err := accounting.NewStore(db, nil)
	if err != nil {
		t.Fatalf("unable to connect to finance: %v", err)
	}

	// generate some noise data to be deleted
	sampleDataNoise(t, store)

	backupFile := "testdata/backup-v1.zip"
	err = Import(t.Context(), store, backupFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("assert accounts", func(t *testing.T) {
		gotAccounts, err := store.ListAccountsProvider(t.Context(), true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := []accounting.AccountProvider{
			{
				ID: 2, Name: "p1", Description: "d1",
				Accounts: []accounting.Account{
					{ID: 2, AccountProviderID: 2, Name: "acc1", Description: "dacc1", Currency: currency.EUR, Type: accounting.CashAccountType},
					{ID: 3, AccountProviderID: 2, Name: "acc2", Description: "dacc2", Currency: currency.USD, Type: accounting.CheckinAccountType},
					{ID: 4, AccountProviderID: 2, Name: "acc3", Description: "dacc3", Currency: currency.CHF, Type: accounting.SavingsAccountType},
				},
			},
			{ID: 3, Name: "p2", Description: "d2",
				Accounts: []accounting.Account{
					{ID: 5, AccountProviderID: 3, Name: "acc4", Description: "dacc4", Currency: currency.EUR, Type: accounting.CheckinAccountType},
				},
			},
		}
		if diff := cmp.Diff(want, gotAccounts, cmpopts.EquateComparable(currency.Unit{})); diff != "" {
			t.Errorf("unexpected result (-want +got):\n%s", diff)
		}
	})

	t.Run("assert categories", func(t *testing.T) {
		incomes, err := store.ListDescendantCategories(t.Context(), 0, -1, accounting.IncomeCategory)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Backup fixture may have 2 or 3 income categories; require at least in1, in2
		names := make(map[string]accounting.CategoryData)
		for _, c := range incomes {
			names[c.Name] = c.CategoryData
		}
		for _, req := range []struct{ name, desc string }{
			{"in1", "din1"}, {"in2", "din2"},
		} {
			if c, ok := names[req.name]; !ok || c.Description != req.desc {
				t.Errorf("missing or wrong income category %q: got %v", req.name, names)
			}
		}

		expenses, err := store.ListDescendantCategories(t.Context(), 0, -1, accounting.ExpenseCategory)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantExpenses := []accounting.Category{
			{CategoryData: accounting.CategoryData{Name: "ex1", Description: "dex1", Type: accounting.ExpenseCategory}},
		}
		if diff := cmp.Diff(wantExpenses, expenses,
			cmpopts.EquateComparable(currency.Unit{}),
			cmpopts.IgnoreFields(accounting.Category{}, "Id", "ParentId"),
		); diff != "" {
			t.Errorf("unexpected result (-want +got):\n%s", diff)
		}
	})

	t.Run("assert transactions", func(t *testing.T) {
		opts := accounting.ListOpts{EndDate: getDate("3000-01-17")}
		got, err := store.ListTransactions(t.Context(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []accounting.Transaction{
			accounting.Income{Id: 2, Description: "i1", Amount: 12.5, AccountID: 2, CategoryID: 2, Date: getDate("2022-01-20")},
			accounting.Expense{Id: 3, Description: "e1", Amount: 22.6, AccountID: 2, CategoryID: 4, Date: getDate("2022-01-19")},
			accounting.Transfer{Id: 4, Description: "tr1", OriginAmount: 36.6, OriginAccountID: 2, TargetAmount: 1.5, TargetAccountID: 3, Date: getDate("2022-01-18")},
			accounting.Income{Id: 5, Description: "i1", Amount: 10.5, AccountID: 5, CategoryID: 0, Date: getDate("2022-01-17")},
		}
		if diff := cmp.Diff(want, got,
			cmpopts.EquateComparable(currency.Unit{}),
			cmpopts.IgnoreFields(accounting.Income{}, "baseTx"),
			cmpopts.IgnoreFields(accounting.Expense{}, "baseTx"),
			cmpopts.IgnoreFields(accounting.Transfer{}, "baseTx"),
		); diff != "" {
			t.Errorf("unexpected result (-want +got):\n%s", diff)
		}
	})

	//spew.Dump(accT1)
}

// sampleDataNoise is used to generate some entries in the store before running an import
// this should generate noise before wiping data
func sampleDataNoise(t *testing.T, store *accounting.Store) {

	// =========================================
	// create accounts providers
	// =========================================
	accProviderId, err := store.CreateAccountProvider(t.Context(), accounting.AccountProvider{Name: "p1noise", Description: "d1noise"})
	if err != nil {
		t.Fatalf("error creating provider 1: %v", err)
	}

	// =========================================
	// create accounts
	// =========================================
	Accs := []accounting.Account{
		{AccountProviderID: accProviderId, Name: "acc1noise", Description: "dacc1noise", Currency: currency.EUR, Type: accounting.CashAccountType},
	}
	for _, acc := range Accs {
		_, err = store.CreateAccount(t.Context(), acc)
		if err != nil {
			t.Fatalf("error creating account 1: %v", err)
		}
	}

	// =========================================
	// create categories
	// =========================================

	in1, err := store.CreateCategory(t.Context(), accounting.CategoryData{Name: "in1noise", Description: "din1noise", Type: accounting.IncomeCategory}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// =========================================
	// Create Transactions
	// =========================================

	t1 := accounting.Income{Description: "i1noise", Amount: 12.5, AccountID: 1, CategoryID: in1, Date: getDate("2022-01-20")}
	_, err = store.CreateTransaction(t.Context(), t1)
	if err != nil {
		t.Fatalf("error creating transaction: %v", err)
	}

}
