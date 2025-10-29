package backup

import (
	"context"
	"fmt"
	"github.com/andresbott/etna/internal/accounting"
	"os"
	"path/filepath"
	"time"
)

const timeFormat = "2006-01-02_15-04-05"

func Export(ctx context.Context, store *accounting.Store, tenant, dest string) error {

	absPath, err := checkDir(dest)
	if err != nil {
		return err
	}

	timestamp := time.Now()
	filename := fmt.Sprintf("backup-%s-%s.zip", tenant, timestamp.Format(timeFormat))
	fullPath := filepath.Join(absPath, filename)

	zw, err := createZipFile(fullPath)
	if err != nil {
		return err
	}
	defer func() {
		e := zw.Close()
		if e != nil {
			err = e
		}
	}()

	err = writeMeta(zw, timestamp, tenant)
	if err != nil {
		return err
	}

	err = writeAccountProviders(ctx, zw, store, tenant)
	if err != nil {
		return err
	}

	err = writeAccounts(ctx, zw, store, tenant)
	if err != nil {
		return err
	}

	err = writeCategories(ctx, zw, store, tenant)
	if err != nil {
		return err
	}

	// iterate over items and generate Json payload

	// categories
	// transactions
	return nil
}

func writeMeta(zw *zipWriter, timestamp time.Time, tenant string) error {
	meta := metaInfoV1{
		Version: SchemaV1,
		Date:    timestamp.Format(timeFormat),
		Tenant:  tenant,
	}
	return zw.writeJsonFile("metainfo.json", meta)
}

func writeAccountProviders(ctx context.Context, zw *zipWriter, store *accounting.Store, tenant string) error {
	providers, err := store.ListAccountsProvider(ctx, tenant, false)
	if err != nil {
		return err
	}

	var jsonData []AccountProviderV1
	for _, provider := range providers {
		jsonData = append(jsonData, AccountProviderV1{
			ID:          provider.ID,
			Name:        provider.Name,
			Description: provider.Description,
		})
	}
	return zw.writeJsonFile("accountproviders.json", jsonData)
}

func writeAccounts(ctx context.Context, zw *zipWriter, store *accounting.Store, tenant string) error {
	Accounts, err := store.ListAccounts(ctx, tenant)
	if err != nil {
		return err
	}

	var jsonData []AccountV1
	for _, acc := range Accounts {
		jsonData = append(jsonData, AccountV1{
			ID:                acc.ID,
			AccountProviderID: acc.AccountProviderID,
			Name:              acc.Name,
			Description:       acc.Description,
			Currency:          acc.Currency.String(),
			Type:              acc.Type.String(),
		})
	}
	return zw.writeJsonFile("accounts.json", jsonData)
}

func writeCategories(ctx context.Context, zw *zipWriter, store *accounting.Store, tenant string) error {
	Incomes, err := store.ListDescendantCategories(ctx, 0, -1, accounting.IncomeCategory, tenant)
	if err != nil {
		return err
	}

	var jsonData []CategoryV1
	for _, income := range Incomes {
		jsonData = append(jsonData, CategoryV1{
			ID:          income.Id,
			ParentId:    income.ParentId,
			Name:        income.Name,
			Description: income.Description,
		})
	}
	err = zw.writeJsonFile("incomecategories.json", jsonData)
	if err != nil {
		return err
	}

	expenses, err := store.ListDescendantCategories(ctx, 0, -1, accounting.ExpenseCategory, tenant)
	if err != nil {
		return err
	}

	jsonData = []CategoryV1{}
	for _, income := range expenses {
		jsonData = append(jsonData, CategoryV1{
			ID:          income.Id,
			ParentId:    income.ParentId,
			Name:        income.Name,
			Description: income.Description,
		})
	}
	return zw.writeJsonFile("expensecategories.json", jsonData)
}

// checkDir validates that the given path is a directory
// and returns its absolute path.
func checkDir(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("path does not exist: %s", absPath)
		}
		return "", fmt.Errorf("failed to stat path: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", absPath)
	}

	return absPath, nil
}

func Import(store accounting.Store, tenant string) {
	// wipe data
	// re-create data
}
