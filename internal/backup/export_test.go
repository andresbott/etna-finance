package backup

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

func TestExport(t *testing.T) {

	db, err := gorm.Open(sqlite.Open("file:exportDb?mode=memory&cache=shared"), &gorm.Config{
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
	sampleData(t, store, mdStore, csvStore, tdStore)

	// set tx list size to 2 to force pagination
	entriesLimit = 2

	tmpdir := t.TempDir()
	target := filepath.Join(tmpdir, "backup.zip")
	err = export(t.Context(), store, mdStore, csvStore, tdStore, target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// copyFile(target) // left here as we use it to update sample data

	// read back data written into the zip
	got, err := readFromZip(target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := backupPayload{
		Meta: metaInfoV1{
			Version: SchemaV1,
		},
		Providers: []accountProviderV1{
			{ID: 1, Name: "p1", Description: "d1", Icon: "bank"},
			{ID: 2, Name: "p2", Description: "d2"},
		},
		Accounts: []accountV1{
			{ID: 1, AccountProviderID: 1, Name: "acc1", Description: "dacc1", Icon: "wallet", Currency: "EUR", Type: "Cash"},
			{ID: 2, AccountProviderID: 1, Name: "acc2", Description: "dacc2", Currency: "USD", Type: "Checkin"},
			{ID: 3, AccountProviderID: 1, Name: "acc3", Description: "dacc3", Currency: "CHF", Type: "Savings"},
			{ID: 4, AccountProviderID: 2, Name: "acc4", Description: "dacc4", Currency: "EUR", Type: "Checkin"},
		},
		IncomeCategories: []categoryV1{
			{ID: 1, ParentId: 0, Name: "in1", Description: "din1", Icon: "income-icon"},
			{ID: 2, ParentId: 1, Name: "in2", Description: "din2"},
			{ID: 4, ParentId: 0, Name: "in3", Description: "din3"},
		},
		ExpenseCategories: []categoryV1{
			{ID: 3, ParentId: 0, Name: "ex1", Description: "dex1", Icon: "expense-icon"},
		},
		Transactions: []TransactionV1{
			{Id: 1, Description: "i1", Amount: 12.5, AccountID: 1, CategoryID: 1, Date: getDate("2022-01-20"), Type: txTypeIncome},
			{Id: 2, Description: "e1", Amount: 22.6, AccountID: 1, CategoryID: 3, Date: getDate("2022-01-19"), Type: txTypeExpense},
			{Id: 3, Description: "tr1", OriginAmount: 36.6, OriginAccountID: 1, TargetAmount: 1.5, TargetAccountID: 2, Date: getDate("2022-01-18"), Type: txTypeTransfer},
			{Id: 4, Description: "i1", Amount: 10.5, AccountID: 4, CategoryID: 0, Date: getDate("2022-01-17"), Type: txTypeIncome},
		},
		Instruments: []instrumentV1{
			{ID: 1, Symbol: "AAPL", Name: "Apple Inc", Currency: "USD"},
		},
		PriceHistory: []priceRecordV1{
			{Symbol: "AAPL", Time: getDate("2024-01-15"), Price: 185.50},
			{Symbol: "AAPL", Time: getDate("2024-01-16"), Price: 186.00},
		},
		FXRates: []fxRateRecordV1{
			{Main: "USD", Secondary: "EUR", Time: getDate("2024-01-15"), Rate: 0.92},
		},
		ImportProfiles: []importProfileV1{
			{ID: 1, Name: "bank-csv", CsvSeparator: ",", DateColumn: "Date", DateFormat: "2006-01-02", DescriptionColumn: "Description", AmountColumn: "Amount", AmountMode: "single"},
		},
		CategoryRules: []categoryRuleGroupV1{
			{ID: 1, Name: "grocery", CategoryID: 3, Priority: 0, Patterns: []categoryRulePatternV1{
				{ID: 1, Pattern: "grocery"},
			}},
		},
		CaseStudies: []caseStudyV1{
			{ID: 1, ToolType: "buy_vs_rent", Name: "test-case", Description: "test desc", ExpectedAnnualReturn: 7.5, Params: json.RawMessage(`{"key":"value"}`)},
		},
	}

	sortCategories := func(a, b categoryV1) bool { return a.ID < b.ID }
	if diff := cmp.Diff(want, got,
		cmpopts.IgnoreFields(metaInfoV1{}, "Date"),
		cmpopts.SortSlices(sortCategories),
		cmpopts.SortSlices(func(a, b instrumentV1) bool { return a.ID < b.ID }),
		cmpopts.SortSlices(func(a, b priceRecordV1) bool { return a.Time.Before(b.Time) }),
		cmpopts.SortSlices(func(a, b fxRateRecordV1) bool { return a.Main < b.Main }),
		cmpopts.SortSlices(func(a, b importProfileV1) bool { return a.ID < b.ID }),
		cmpopts.SortSlices(func(a, b categoryRuleGroupV1) bool { return a.ID < b.ID }),
		cmpopts.SortSlices(func(a, b caseStudyV1) bool { return a.ID < b.ID }),
		cmpopts.IgnoreFields(caseStudyV1{}, "Params"),
	); diff != "" {
		t.Errorf("unexpected result (-want +got):\n%s", diff)
	}

}

// copyFile copies the generated backup file into the test folder;
// this is used when we make changes to the backup file structure and want to extend the test-data
//
//nolint:unused // to be used when updating test cases
func copyFile(source string) {
	dst := "backup.zip"

	srcFile, _ := os.Open(source) //nolint:gosec // only test case
	defer func() {
		_ = srcFile.Close()
	}()

	dstFile, _ := os.Create(dst)
	defer func() {
		_ = dstFile.Close()
	}()
	_, _ = io.Copy(dstFile, srcFile)
}

type backupPayload struct {
	Meta              metaInfoV1
	Providers         []accountProviderV1
	Accounts          []accountV1
	IncomeCategories  []categoryV1
	ExpenseCategories []categoryV1
	Transactions      []TransactionV1
	Instruments       []instrumentV1
	PriceHistory      []priceRecordV1
	FXRates           []fxRateRecordV1
	ImportProfiles    []importProfileV1
	CategoryRules     []categoryRuleGroupV1
	CaseStudies       []caseStudyV1
}

func unmarshalJSON[T any](data []byte) (T, error) {
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return result, fmt.Errorf("failed to unmarshal json: %w", err)
	}
	return result, nil
}

func readZipEntry(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", f.Name, err)
	}
	defer func() {
		_ = rc.Close()
	}()
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", f.Name, err)
	}
	return data, nil
}

func readFromZip(zipPath string) (backupPayload, error) {
	payload := backupPayload{}
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return payload, fmt.Errorf("failed to open zip: %w", err)
	}
	defer func() {
		_ = r.Close()
	}()

	for _, f := range r.File {
		data, err := readZipEntry(f)
		if err != nil {
			return payload, err
		}

		switch f.Name {
		case metaInfoFile:
			payload.Meta, err = unmarshalJSON[metaInfoV1](data)
		case accountProviderFile:
			payload.Providers, err = unmarshalJSON[[]accountProviderV1](data)
		case accountsFile:
			payload.Accounts, err = unmarshalJSON[[]accountV1](data)
		case expenseCategoriesFile:
			payload.ExpenseCategories, err = unmarshalJSON[[]categoryV1](data)
		case incomeCategoriesFile:
			payload.IncomeCategories, err = unmarshalJSON[[]categoryV1](data)
		case transactionsFile:
			payload.Transactions, err = unmarshalJSON[[]TransactionV1](data)
		case instrumentsFile:
			payload.Instruments, err = unmarshalJSON[[]instrumentV1](data)
		case priceHistoryFile:
			payload.PriceHistory, err = unmarshalJSON[[]priceRecordV1](data)
		case fxRatesFile:
			payload.FXRates, err = unmarshalJSON[[]fxRateRecordV1](data)
		case importProfilesFile:
			payload.ImportProfiles, err = unmarshalJSON[[]importProfileV1](data)
		case categoryRulesFile:
			payload.CategoryRules, err = unmarshalJSON[[]categoryRuleGroupV1](data)
		case caseStudiesFile:
			payload.CaseStudies, err = unmarshalJSON[[]caseStudyV1](data)
		}
		if err != nil {
			return payload, err
		}
	}
	return payload, nil
}

func sampleData(t *testing.T, store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, tdStore *toolsdata.Store) {

	// =========================================
	// create accounts providers
	// =========================================

	accProviderId, err := store.CreateAccountProvider(t.Context(), accounting.AccountProvider{Name: "p1", Description: "d1", Icon: "bank"})
	if err != nil {
		t.Fatalf("error creating provider 1: %v", err)
	}
	accProviderId2, err := store.CreateAccountProvider(t.Context(), accounting.AccountProvider{Name: "p2", Description: "d2"})
	if err != nil {
		t.Fatalf("error creating provider 1: %v", err)
	}
	// =========================================
	// create accounts
	// =========================================
	Accs := []accounting.Account{
		{AccountProviderID: accProviderId, Name: "acc1", Description: "dacc1", Icon: "wallet", Currency: currency.EUR, Type: accounting.CashAccountType},
		{AccountProviderID: accProviderId, Name: "acc2", Description: "dacc2", Currency: currency.USD, Type: accounting.CheckinAccountType},
		{AccountProviderID: accProviderId, Name: "acc3", Description: "dacc3", Currency: currency.CHF, Type: accounting.SavingsAccountType},
	}
	for _, acc := range Accs {
		_, err = store.CreateAccount(t.Context(), acc)
		if err != nil {
			t.Fatalf("error creating account 1: %v", err)
		}
	}

	acc := accounting.Account{AccountProviderID: accProviderId2, Name: "acc4", Description: "dacc4", Currency: currency.EUR, Type: accounting.CheckinAccountType}
	_, err = store.CreateAccount(t.Context(), acc)
	if err != nil {
		t.Fatalf("error creating account 1: %v", err)
	}

	// =========================================
	// create categories
	// =========================================

	in1, err := store.CreateCategory(t.Context(), accounting.CategoryData{Name: "in1", Description: "din1", Icon: "income-icon", Type: accounting.IncomeCategory}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// income children1
	_, err = store.CreateCategory(t.Context(), accounting.CategoryData{Name: "in2", Description: "din2", Type: accounting.IncomeCategory}, in1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// expense
	ex1, err := store.CreateCategory(t.Context(), accounting.CategoryData{Name: "ex1", Description: "dex1", Icon: "expense-icon", Type: accounting.ExpenseCategory}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// tenant 2
	_, err = store.CreateCategory(t.Context(), accounting.CategoryData{Name: "in3", Description: "din3", Type: accounting.IncomeCategory}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// =========================================
	// Create Transactions
	// =========================================

	t1 := accounting.Income{Description: "i1", Amount: 12.5, AccountID: 1, CategoryID: in1, Date: getDate("2022-01-20")}
	_, err = store.CreateTransaction(t.Context(), t1)
	if err != nil {
		t.Fatalf("error creating transaction: %v", err)
	}

	t2 := accounting.Expense{Description: "e1", Amount: 22.6, AccountID: 1, CategoryID: ex1, Date: getDate("2022-01-19")}
	_, err = store.CreateTransaction(t.Context(), t2)
	if err != nil {
		t.Fatalf("error creating transaction: %v", err)
	}
	tr1 := accounting.Transfer{Description: "tr1", OriginAmount: 36.6, OriginAccountID: 1, TargetAmount: 1.5, TargetAccountID: 2, Date: getDate("2022-01-18")}
	_, err = store.CreateTransaction(t.Context(), tr1)
	if err != nil {
		t.Fatalf("error creating transaction: %v", err)
	}

	t3 := accounting.Income{Description: "i1", Amount: 10.5, AccountID: 4, CategoryID: 0, Date: getDate("2022-01-17")}
	_, err = store.CreateTransaction(t.Context(), t3)
	if err != nil {
		t.Fatalf("error creating transaction: %v", err)
	}

	sampleExtraData(t, mdStore, csvStore, tdStore, ex1)
}

func sampleExtraData(t *testing.T, mdStore *marketdata.Store, csvStore *csvimport.Store, tdStore *toolsdata.Store, expenseCategoryID uint) {
	t.Helper()

	_, err := mdStore.CreateInstrument(t.Context(), marketdata.Instrument{
		Symbol: "AAPL", Name: "Apple Inc", Currency: currency.USD,
	})
	if err != nil {
		t.Fatalf("error creating instrument: %v", err)
	}

	err = mdStore.IngestPrice(t.Context(), "AAPL", getDate("2024-01-15"), 185.50)
	if err != nil {
		t.Fatalf("error ingesting price: %v", err)
	}
	err = mdStore.IngestPrice(t.Context(), "AAPL", getDate("2024-01-16"), 186.00)
	if err != nil {
		t.Fatalf("error ingesting price: %v", err)
	}

	err = mdStore.IngestRate(t.Context(), "USD", "EUR", getDate("2024-01-15"), 0.92)
	if err != nil {
		t.Fatalf("error ingesting FX rate: %v", err)
	}

	_, err = csvStore.CreateProfile(t.Context(), csvimport.ImportProfile{
		Name: "bank-csv", CsvSeparator: ",", DateColumn: "Date",
		DateFormat: "2006-01-02", DescriptionColumn: "Description",
		AmountColumn: "Amount", AmountMode: "single",
	})
	if err != nil {
		t.Fatalf("error creating import profile: %v", err)
	}

	_, err = csvStore.CreateCategoryRuleGroup(t.Context(), csvimport.CategoryRuleGroup{
		Name: "grocery", CategoryID: expenseCategoryID, Priority: 0,
		Patterns: []csvimport.CategoryRulePattern{{Pattern: "grocery"}},
	})
	if err != nil {
		t.Fatalf("error creating category rule group: %v", err)
	}

	_, err = tdStore.Create(t.Context(), toolsdata.CaseStudy{
		ToolType:             "buy_vs_rent",
		Name:                 "test-case",
		Description:          "test desc",
		ExpectedAnnualReturn: 7.5,
		Params:               json.RawMessage(`{"key":"value"}`),
	})
	if err != nil {
		t.Fatalf("error creating case study: %v", err)
	}
}
func TestRoundTrip(t *testing.T) {
	// Source DB
	db1, err := gorm.Open(sqlite.Open("file:rtSource?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	store1, err := accounting.NewStore(db1, nil)
	if err != nil {
		t.Fatal(err)
	}
	mdStore1, err := marketdata.NewStore(db1)
	if err != nil {
		t.Fatal(err)
	}
	csvStore1, err := csvimport.NewStore(db1)
	if err != nil {
		t.Fatal(err)
	}
	tdStore1, err := toolsdata.NewStore(db1)
	if err != nil {
		t.Fatal(err)
	}
	sampleData(t, store1, mdStore1, csvStore1, tdStore1)

	// Export 1
	tmpdir := t.TempDir()
	target1 := filepath.Join(tmpdir, "export1.zip")
	err = export(t.Context(), store1, mdStore1, csvStore1, tdStore1, target1)
	if err != nil {
		t.Fatalf("export1 failed: %v", err)
	}

	// Destination DB
	db2, err := gorm.Open(sqlite.Open("file:rtDest?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	store2, err := accounting.NewStore(db2, nil)
	if err != nil {
		t.Fatal(err)
	}
	mdStore2, err := marketdata.NewStore(db2)
	if err != nil {
		t.Fatal(err)
	}
	csvStore2, err := csvimport.NewStore(db2)
	if err != nil {
		t.Fatal(err)
	}
	tdStore2, err := toolsdata.NewStore(db2)
	if err != nil {
		t.Fatal(err)
	}

	// Import
	err = Import(t.Context(), store2, mdStore2, csvStore2, tdStore2, target1)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}

	// Export 2
	target2 := filepath.Join(tmpdir, "export2.zip")
	err = export(t.Context(), store2, mdStore2, csvStore2, tdStore2, target2)
	if err != nil {
		t.Fatalf("export2 failed: %v", err)
	}

	// Compare
	got1, err := readFromZip(target1)
	if err != nil {
		t.Fatalf("readFromZip1 failed: %v", err)
	}
	got2, err := readFromZip(target2)
	if err != nil {
		t.Fatalf("readFromZip2 failed: %v", err)
	}

	// IDs will differ, so ignore all ID fields
	if diff := cmp.Diff(got1, got2,
		cmpopts.IgnoreFields(metaInfoV1{}, "Date"),
		cmpopts.IgnoreFields(accountProviderV1{}, "ID"),
		cmpopts.IgnoreFields(accountV1{}, "ID", "AccountProviderID", "ImportProfileID"),
		cmpopts.IgnoreFields(categoryV1{}, "ID", "ParentId"),
		cmpopts.IgnoreFields(TransactionV1{}, "Id", "AccountID", "CategoryID", "OriginAccountID", "TargetAccountID", "InvestmentAccountID", "CashAccountID", "SourceAccountID", "InstrumentID"),
		cmpopts.IgnoreFields(instrumentV1{}, "ID", "InstrumentProviderID"),
		cmpopts.IgnoreFields(importProfileV1{}, "ID"),
		cmpopts.IgnoreFields(categoryRuleGroupV1{}, "ID", "CategoryID"),
		cmpopts.IgnoreFields(categoryRulePatternV1{}, "ID"),
		cmpopts.SortSlices(func(a, b accountProviderV1) bool { return a.Name < b.Name }),
		cmpopts.SortSlices(func(a, b accountV1) bool { return a.Name < b.Name }),
		cmpopts.SortSlices(func(a, b categoryV1) bool { return a.Name < b.Name }),
		cmpopts.SortSlices(func(a, b TransactionV1) bool { return a.Date.Before(b.Date) }),
		cmpopts.SortSlices(func(a, b instrumentV1) bool { return a.Symbol < b.Symbol }),
		cmpopts.SortSlices(func(a, b priceRecordV1) bool { return a.Time.Before(b.Time) }),
		cmpopts.SortSlices(func(a, b fxRateRecordV1) bool { return a.Main < b.Main }),
		cmpopts.SortSlices(func(a, b importProfileV1) bool { return a.Name < b.Name }),
		cmpopts.SortSlices(func(a, b categoryRuleGroupV1) bool { return a.Name < b.Name }),
		cmpopts.IgnoreFields(caseStudyV1{}, "ID"),
		cmpopts.SortSlices(func(a, b caseStudyV1) bool { return a.Name < b.Name }),
	); diff != "" {
		t.Errorf("round-trip mismatch (-first +second):\n%s", diff)
	}
}

func getDate(timeStr string) time.Time {
	// Parse the string based on the provided layout
	parsedTime, err := time.Parse("2006-01-02", timeStr)
	if err != nil {
		panic(fmt.Errorf("unable to parse time: %v", err))
	}
	return parsedTime
}
