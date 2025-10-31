package backup

import (
	"github.com/andresbott/etna/internal/accounting"
	"github.com/glebarez/sqlite"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/text/currency"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"testing"
)

func TestImportV1(t *testing.T) {

	db, err := gorm.Open(sqlite.Open("file:importDb?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("unable to connect to sqlite: %v", err)
	}
	store, err := accounting.NewStore(db)
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
		gotAccounts, err := store.ListAccountsProvider(t.Context(), tenant1, true)
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
		}
		if diff := cmp.Diff(want, gotAccounts, cmpopts.EquateComparable(currency.Unit{})); diff != "" {
			t.Errorf("unexpected result (-want +got):\n%s", diff)
		}

		gotAccounts, err = store.ListAccountsProvider(t.Context(), tenant2, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want = []accounting.AccountProvider{
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
		wantIncomes := map[string][]accounting.Category{
			tenant1: {
				{Id: 2, ParentId: 0, CategoryData: accounting.CategoryData{Name: "in1", Description: "din1", Type: accounting.IncomeCategory}},
				{Id: 3, ParentId: 2, CategoryData: accounting.CategoryData{Name: "in2", Description: "din2", Type: accounting.IncomeCategory}},
			},
		}

		for _, tenant := range []string{tenant1, tenant2} {
			incomes, err := store.ListDescendantCategories(t.Context(), 0, -1, accounting.IncomeCategory, tenant)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if diff := cmp.Diff(wantIncomes[tenant], incomes, cmpopts.EquateComparable(currency.Unit{})); diff != "" {
				t.Errorf("unexpected result (-wantIncomes +got):\n%s", diff)
			}
		}

		wantExpenses := map[string][]accounting.Category{
			tenant1: {
				{Id: 4, ParentId: 0, CategoryData: accounting.CategoryData{Name: "ex1", Description: "dex1", Type: accounting.ExpenseCategory}},
			},
		}

		for _, tenant := range []string{tenant1, tenant2} {
			incomes, err := store.ListDescendantCategories(t.Context(), 0, -1, accounting.ExpenseCategory, tenant)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if diff := cmp.Diff(wantExpenses[tenant], incomes, cmpopts.EquateComparable(currency.Unit{})); diff != "" {
				t.Errorf("unexpected result (-wantIncomes +got):\n%s", diff)
			}
		}

	})

	t.Run("assert transactions", func(t *testing.T) {
		wantTransactions := map[string][]accounting.Transaction{tenant1: {}, tenant2: {}}
		wantTransactions[tenant1] = append(wantTransactions[tenant1], accounting.Income{
			Id: 2, Description: "i1", Amount: 12.5, AccountID: 2, CategoryID: 2, Date: getDate("2022-01-20"),
		})
		wantTransactions[tenant1] = append(wantTransactions[tenant1], accounting.Expense{
			Id: 3, Description: "e1", Amount: 22.6, AccountID: 2, CategoryID: 4, Date: getDate("2022-01-19"),
		})
		wantTransactions[tenant1] = append(wantTransactions[tenant1], accounting.Transfer{
			Id: 4, Description: "tr1", OriginAmount: 36.6, OriginAccountID: 2, TargetAmount: 1.5, TargetAccountID: 3, Date: getDate("2022-01-18"),
		})
		wantTransactions[tenant2] = append(wantTransactions[tenant2], accounting.Income{
			Id: 5, Description: "i1", Amount: 10.5, AccountID: 5, CategoryID: 0, Date: getDate("2022-01-17"),
		})

		for _, tenant := range []string{tenant1, tenant2} {
			opts := accounting.ListOpts{EndDate: getDate("3000-01-17")}
			incomes, err := store.ListTransactions(t.Context(), opts, tenant)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if diff := cmp.Diff(wantTransactions[tenant], incomes,
				cmpopts.EquateComparable(currency.Unit{}),
				cmpopts.IgnoreFields(accounting.Income{}, "baseTx"),
				cmpopts.IgnoreFields(accounting.Expense{}, "baseTx"),
				cmpopts.IgnoreFields(accounting.Transfer{}, "baseTx"),
			); diff != "" {
				t.Errorf("unexpected result (-wantIncomes +got):\n%s", diff)
			}
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
	accProviderId, err := store.CreateAccountProvider(t.Context(), accounting.AccountProvider{Name: "p1noise", Description: "d1noise"}, tenant1)
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
		_, err = store.CreateAccount(t.Context(), acc, tenant1)
		if err != nil {
			t.Fatalf("error creating account 1: %v", err)
		}
	}

	// =========================================
	// create categories
	// =========================================

	in1, err := store.CreateCategory(t.Context(), accounting.CategoryData{Name: "in1noise", Description: "din1noise", Type: accounting.IncomeCategory}, 0, tenant1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// =========================================
	// Create Transactions
	// =========================================

	t1 := accounting.Income{Description: "i1noise", Amount: 12.5, AccountID: 1, CategoryID: in1, Date: getDate("2022-01-20")}
	_, err = store.CreateTransaction(t.Context(), t1, tenant1)
	if err != nil {
		t.Fatalf("error creating transaction: %v", err)
	}

}
