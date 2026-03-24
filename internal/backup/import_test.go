package backup

import (
	"testing"
	"time"

	"github.com/andresbott/etna/internal/accounting"
	"github.com/andresbott/etna/internal/csvimport"
	"github.com/andresbott/etna/internal/marketdata"
	"github.com/andresbott/etna/internal/toolsdata"
	"github.com/glebarez/sqlite"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/text/currency"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type testStores struct {
	accounting *accounting.Store
	marketdata *marketdata.Store
	csvimport  *csvimport.Store
	toolsdata  *toolsdata.Store
}

func setupImportTest(t *testing.T) testStores {
	t.Helper()
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
	mdStore, err := marketdata.NewStore(db)
	if err != nil {
		t.Fatalf("unable to create marketdata store: %v", err)
	}
	csvStore, err := csvimport.NewStore(db)
	if err != nil {
		t.Fatalf("unable to create csvimport store: %v", err)
	}
	tdStore, err := toolsdata.NewStore(db)
	if err != nil {
		t.Fatalf("unable to create toolsdata store: %v", err)
	}

	sampleDataNoise(t, store, mdStore, csvStore)

	backupFile := "testdata/backup-v1.zip"
	err = Import(t.Context(), store, mdStore, csvStore, tdStore, backupFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return testStores{accounting: store, marketdata: mdStore, csvimport: csvStore, toolsdata: tdStore}
}

func TestImportV1(t *testing.T) {
	stores := setupImportTest(t)

	t.Run("assert accounts", func(t *testing.T) {
		assertImportedAccounts(t, stores.accounting)
	})
	t.Run("assert categories", func(t *testing.T) {
		assertImportedCategories(t, stores.accounting)
	})
	t.Run("assert transactions", func(t *testing.T) {
		assertImportedTransactions(t, stores.accounting)
	})
	t.Run("assert instruments", func(t *testing.T) {
		assertImportedInstruments(t, stores.marketdata)
	})
	t.Run("assert price history", func(t *testing.T) {
		assertImportedPriceHistory(t, stores.marketdata)
	})
	t.Run("assert fx rates", func(t *testing.T) {
		assertImportedFXRates(t, stores.marketdata)
	})
	t.Run("assert import profiles", func(t *testing.T) {
		assertImportedProfiles(t, stores.csvimport)
	})
	t.Run("assert category rules", func(t *testing.T) {
		assertImportedCategoryRules(t, stores.csvimport)
	})
}

func assertImportedAccounts(t *testing.T, store *accounting.Store) {
	t.Helper()
	gotAccounts, err := store.ListAccountsProvider(t.Context(), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []accounting.AccountProvider{
		{
			ID: 2, Name: "p1", Description: "d1", Icon: "bank",
			Accounts: []accounting.Account{
				{ID: 2, AccountProviderID: 2, Name: "acc1", Description: "dacc1", Icon: "wallet", Currency: currency.EUR, Type: accounting.CashAccountType},
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
}

func assertImportedCategories(t *testing.T, store *accounting.Store) {
	t.Helper()
	incomes, err := store.ListDescendantCategories(t.Context(), 0, -1, accounting.IncomeCategory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	names := make(map[string]accounting.CategoryData)
	for _, c := range incomes {
		names[c.Name] = c.CategoryData
	}
	for _, req := range []struct{ name, desc, icon string }{
		{"in1", "din1", "income-icon"}, {"in2", "din2", ""},
	} {
		c, ok := names[req.name]
		if !ok || c.Description != req.desc {
			t.Errorf("missing or wrong income category %q: got %v", req.name, names)
		}
		if ok && c.Icon != req.icon {
			t.Errorf("income category %q: expected icon %q, got %q", req.name, req.icon, c.Icon)
		}
	}

	expenses, err := store.ListDescendantCategories(t.Context(), 0, -1, accounting.ExpenseCategory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantExpenses := []accounting.Category{
		{CategoryData: accounting.CategoryData{Name: "ex1", Description: "dex1", Icon: "expense-icon", Type: accounting.ExpenseCategory}},
	}
	if diff := cmp.Diff(wantExpenses, expenses,
		cmpopts.EquateComparable(currency.Unit{}),
		cmpopts.IgnoreFields(accounting.Category{}, "Id", "ParentId"),
	); diff != "" {
		t.Errorf("unexpected result (-want +got):\n%s", diff)
	}
}

func assertImportedTransactions(t *testing.T, store *accounting.Store) {
	t.Helper()
	opts := accounting.ListOpts{EndDate: getDate("3000-01-17")}
	got, _, err := store.ListTransactions(t.Context(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []accounting.Transaction{
		accounting.Income{Description: "i1", Amount: 12.5, AccountID: 2, CategoryID: 2, Date: getDate("2022-01-20")},
		accounting.Expense{Description: "e1", Amount: 22.6, AccountID: 2, CategoryID: 5, Date: getDate("2022-01-19")},
		accounting.Transfer{Description: "tr1", OriginAmount: 36.6, OriginAccountID: 2, TargetAmount: 1.5, TargetAccountID: 3, Date: getDate("2022-01-18")},
		accounting.Income{Description: "i1", Amount: 10.5, AccountID: 5, CategoryID: 0, Date: getDate("2022-01-17")},
	}
	if diff := cmp.Diff(want, got,
		cmpopts.EquateComparable(currency.Unit{}),
		cmpopts.IgnoreFields(accounting.Income{}, "Id", "baseTx"),
		cmpopts.IgnoreFields(accounting.Expense{}, "Id", "baseTx"),
		cmpopts.IgnoreFields(accounting.Transfer{}, "Id", "baseTx"),
	); diff != "" {
		t.Errorf("unexpected result (-want +got):\n%s", diff)
	}
}

func assertImportedInstruments(t *testing.T, mdStore *marketdata.Store) {
	t.Helper()
	instruments, err := mdStore.ListInstruments(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(instruments) != 1 {
		t.Fatalf("expected 1 instrument, got %d", len(instruments))
	}
	if instruments[0].Symbol != "AAPL" {
		t.Errorf("expected symbol AAPL, got %s", instruments[0].Symbol)
	}
	if instruments[0].Name != "Apple Inc" {
		t.Errorf("expected name Apple Inc, got %s", instruments[0].Name)
	}
}

func assertImportedPriceHistory(t *testing.T, mdStore *marketdata.Store) {
	t.Helper()
	records, err := mdStore.PriceHistory(t.Context(), "AAPL", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) < 1 {
		t.Fatalf("expected at least 1 price record, got %d", len(records))
	}
}

func assertImportedFXRates(t *testing.T, mdStore *marketdata.Store) {
	t.Helper()
	pairs, err := mdStore.ListFXPairs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 1 {
		t.Fatalf("expected 1 FX pair, got %d", len(pairs))
	}
}

func assertImportedProfiles(t *testing.T, csvStore *csvimport.Store) {
	t.Helper()
	profiles, err := csvStore.ListProfiles(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}
	if profiles[0].Name != "bank-csv" {
		t.Errorf("expected profile name bank-csv, got %s", profiles[0].Name)
	}
}

func assertImportedCategoryRules(t *testing.T, csvStore *csvimport.Store) {
	t.Helper()
	groups, err := csvStore.ListCategoryRuleGroups(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Patterns) != 1 {
		t.Fatalf("expected 1 pattern, got %d", len(groups[0].Patterns))
	}
	if groups[0].Patterns[0].Pattern != "grocery" {
		t.Errorf("expected pattern grocery, got %s", groups[0].Patterns[0].Pattern)
	}
}

// sampleDataNoise is used to generate some entries in the store before running an import
// this should generate noise before wiping data
func sampleDataNoise(t *testing.T, store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store) {

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

	// marketdata noise
	_, _ = mdStore.CreateInstrument(t.Context(), marketdata.Instrument{
		Symbol: "NOISE", Name: "Noise Corp", Currency: currency.EUR,
	})
	_ = mdStore.IngestPrice(t.Context(), "NOISE", time.Now(), 999.0)

	// csvimport noise
	_, _ = csvStore.CreateProfile(t.Context(), csvimport.ImportProfile{
		Name: "noise-profile", DateColumn: "d", DateFormat: "2006-01-02",
		DescriptionColumn: "desc", AmountColumn: "amt", AmountMode: "single",
	})

}
