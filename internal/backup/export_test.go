package backup

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"github.com/andresbott/etna/internal/accounting"
	"github.com/glebarez/sqlite"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/text/currency"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExport(t *testing.T) {

	db, err := gorm.Open(sqlite.Open("file:exportDb?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("unable to connect to sqlite: %v", err)
	}
	store, err := accounting.NewStore(db)
	if err != nil {
		t.Fatalf("unable to connect to finance: %v", err)
	}
	sampleData(t, store)

	// set tx list size to 2 to force pagination
	entriesLimit = 2

	tmpdir := t.TempDir()
	target := filepath.Join(tmpdir, "backup.zip")
	err = export(t.Context(), store, target)
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
			Tenants: []string{tenant1, tenant2},
		},
		Providers: []accountProviderV1{
			{ID: 1, Name: "p1", Description: "d1", Tenant: tenant1},
			{ID: 2, Name: "p2", Description: "d2", Tenant: tenant2},
		},
		Accounts: []accountV1{
			{ID: 1, AccountProviderID: 1, Name: "acc1", Description: "dacc1", Currency: "EUR", Type: "Cash", Tenant: tenant1},
			{ID: 2, AccountProviderID: 1, Name: "acc2", Description: "dacc2", Currency: "USD", Type: "Checkin", Tenant: tenant1},
			{ID: 3, AccountProviderID: 1, Name: "acc3", Description: "dacc3", Currency: "CHF", Type: "Savings", Tenant: tenant1},
			{ID: 4, AccountProviderID: 2, Name: "acc4", Description: "dacc4", Currency: "EUR", Type: "Checkin", Tenant: tenant2},
		},
		IncomeCategories: []categoryV1{
			{ID: 1, ParentId: 0, Name: "in1", Description: "din1", Tenant: tenant1},
			{ID: 2, ParentId: 1, Name: "in2", Description: "din2", Tenant: tenant1},
			{ID: 4, ParentId: 0, Name: "in3", Description: "din3", Tenant: tenant2},
		},
		ExpenseCategories: []categoryV1{
			{ID: 3, ParentId: 0, Name: "ex1", Description: "dex1", Tenant: tenant1},
		},
		Transactions: []TransactionV1{
			{Id: 1, Description: "i1", Amount: 12.5, AccountID: 1, CategoryID: 1, Date: getDate("2022-01-20"), Type: txTypeIncome, Tenant: tenant1},
			{Id: 2, Description: "e1", Amount: 22.6, AccountID: 1, CategoryID: 3, Date: getDate("2022-01-19"), Type: txTypeExpense, Tenant: tenant1},
			{Id: 3, Description: "tr1", OriginAmount: 36.6, OriginAccountID: 1, TargetAmount: 1.5, TargetAccountID: 2, Date: getDate("2022-01-18"), Type: txTypeTransfer, Tenant: tenant1},
			{Id: 4, Description: "i1", Amount: 10.5, AccountID: 4, CategoryID: 0, Date: getDate("2022-01-17"), Type: txTypeIncome, Tenant: tenant2},
		},
	}

	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(metaInfoV1{}, "Date")); diff != "" {
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

	// Iterate through files in the zip
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return payload, fmt.Errorf("failed to open file %s: %w", f.Name, err)
		}

		data, err := io.ReadAll(rc)
		if err != nil {
			_ = rc.Close()
			return payload, fmt.Errorf("failed to read file %s: %w", f.Name, err)
		}

		switch f.Name {
		case metaInfoFile:
			meta := metaInfoV1{}
			if err := json.Unmarshal(data, &meta); err != nil {
				return payload, fmt.Errorf("failed to unmarshal json: %w", err)
			}
			payload.Meta = meta
		case accountProviderFile:
			var prov []accountProviderV1
			if err := json.Unmarshal(data, &prov); err != nil {
				return payload, fmt.Errorf("failed to unmarshal json: %w", err)
			}
			payload.Providers = prov
		case accountsFile:
			var ac []accountV1
			if err := json.Unmarshal(data, &ac); err != nil {
				return payload, fmt.Errorf("failed to unmarshal json: %w", err)
			}
			payload.Accounts = ac
		case expenseCategoriesFile:
			var exp []categoryV1
			if err := json.Unmarshal(data, &exp); err != nil {
				return payload, fmt.Errorf("failed to unmarshal json: %w", err)
			}
			payload.ExpenseCategories = exp
		case incomeCategoriesFile:
			var exp []categoryV1
			if err := json.Unmarshal(data, &exp); err != nil {
				return payload, fmt.Errorf("failed to unmarshal json: %w", err)
			}
			payload.IncomeCategories = exp
		case transactionsFile:
			var items []TransactionV1
			if err := json.Unmarshal(data, &items); err != nil {
				return payload, fmt.Errorf("failed to unmarshal json: %w", err)
			}
			payload.Transactions = items

		}
		_ = rc.Close()
	}
	return payload, nil
}

const tenant1 = "tenant1"
const tenant2 = "tenant2"

func sampleData(t *testing.T, store *accounting.Store) {

	// =========================================
	// create accounts providers
	// =========================================

	accProviderId, err := store.CreateAccountProvider(t.Context(), accounting.AccountProvider{Name: "p1", Description: "d1"}, tenant1)
	if err != nil {
		t.Fatalf("error creating provider 1: %v", err)
	}
	accProviderId2, err := store.CreateAccountProvider(t.Context(), accounting.AccountProvider{Name: "p2", Description: "d2"}, tenant2)
	if err != nil {
		t.Fatalf("error creating provider 1: %v", err)
	}
	// =========================================
	// create accounts
	// =========================================
	Accs := []accounting.Account{
		{AccountProviderID: accProviderId, Name: "acc1", Description: "dacc1", Currency: currency.EUR, Type: accounting.CashAccountType},
		{AccountProviderID: accProviderId, Name: "acc2", Description: "dacc2", Currency: currency.USD, Type: accounting.CheckinAccountType},
		{AccountProviderID: accProviderId, Name: "acc3", Description: "dacc3", Currency: currency.CHF, Type: accounting.SavingsAccountType},
	}
	for _, acc := range Accs {
		_, err = store.CreateAccount(t.Context(), acc, tenant1)
		if err != nil {
			t.Fatalf("error creating account 1: %v", err)
		}
	}

	acc := accounting.Account{AccountProviderID: accProviderId2, Name: "acc4", Description: "dacc4", Currency: currency.EUR, Type: accounting.CheckinAccountType}
	_, err = store.CreateAccount(t.Context(), acc, tenant2)
	if err != nil {
		t.Fatalf("error creating account 1: %v", err)
	}

	// =========================================
	// create categories
	// =========================================

	in1, err := store.CreateCategory(t.Context(), accounting.CategoryData{Name: "in1", Description: "din1", Type: accounting.IncomeCategory}, 0, tenant1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// income children1
	_, err = store.CreateCategory(t.Context(), accounting.CategoryData{Name: "in2", Description: "din2", Type: accounting.IncomeCategory}, in1, tenant1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// expense
	ex1, err := store.CreateCategory(t.Context(), accounting.CategoryData{Name: "ex1", Description: "dex1", Type: accounting.ExpenseCategory}, 0, tenant1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// tenant 2
	_, err = store.CreateCategory(t.Context(), accounting.CategoryData{Name: "in3", Description: "din3", Type: accounting.IncomeCategory}, 0, tenant2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// =========================================
	// Create Transactions
	// =========================================

	t1 := accounting.Income{Description: "i1", Amount: 12.5, AccountID: 1, CategoryID: in1, Date: getDate("2022-01-20")}
	_, err = store.CreateTransaction(t.Context(), t1, tenant1)
	if err != nil {
		t.Fatalf("error creating transaction: %v", err)
	}

	t2 := accounting.Expense{Description: "e1", Amount: 22.6, AccountID: 1, CategoryID: ex1, Date: getDate("2022-01-19")}
	_, err = store.CreateTransaction(t.Context(), t2, tenant1)
	if err != nil {
		t.Fatalf("error creating transaction: %v", err)
	}
	tr1 := accounting.Transfer{Description: "tr1", OriginAmount: 36.6, OriginAccountID: 1, TargetAmount: 1.5, TargetAccountID: 2, Date: getDate("2022-01-18")}
	_, err = store.CreateTransaction(t.Context(), tr1, tenant1)
	if err != nil {
		t.Fatalf("error creating transaction: %v", err)
	}

	t3 := accounting.Income{Description: "i1", Amount: 10.5, AccountID: 4, CategoryID: 0, Date: getDate("2022-01-17")}
	_, err = store.CreateTransaction(t.Context(), t3, tenant2)
	if err != nil {
		t.Fatalf("error creating transaction: %v", err)
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
