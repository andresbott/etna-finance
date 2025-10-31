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

func zipAbsPath(dest string) (string, error) {
	absPath, err := checkDir(dest)
	if err != nil {
		return "", err
	}
	timestamp := time.Now()
	filename := fmt.Sprintf("backup-%s.zip", timestamp.Format(timeFormat))
	fullPath := filepath.Join(absPath, filename)
	return fullPath, nil
}

func Export(ctx context.Context, store *accounting.Store, destination string) error {
	absPath, err := zipAbsPath(destination)
	if err != nil {
		return err
	}
	return export(ctx, store, absPath)
}

func export(ctx context.Context, store *accounting.Store, fullPath string) error {

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

	tenants, err := store.ListTenants(ctx)
	if err != nil {
		return err
	}

	err = writeMeta(zw, time.Now(), tenants)
	if err != nil {
		return err
	}

	err = writeAccountProviders(ctx, zw, store, tenants)
	if err != nil {
		return err
	}

	err = writeAccounts(ctx, zw, store, tenants)
	if err != nil {
		return err
	}

	err = writeCategories(ctx, zw, store, tenants)
	if err != nil {
		return err
	}

	err = writeTransactions(ctx, zw, store, tenants)
	if err != nil {
		return err
	}

	return nil
}

func writeMeta(zw *zipWriter, timestamp time.Time, tenants []string) error {
	meta := metaInfoV1{
		Version: SchemaV1,
		Date:    timestamp.Format(timeFormat),
		Tenants: tenants,
	}
	return zw.writeJsonFile(metaInfoFile, meta)
}

func writeAccountProviders(ctx context.Context, zw *zipWriter, store *accounting.Store, tenants []string) error {
	var jsonData []accountProviderV1
	for _, tenant := range tenants {
		providers, err := store.ListAccountsProvider(ctx, tenant, false)
		if err != nil {
			return err
		}
		for _, provider := range providers {
			jsonData = append(jsonData, accountProviderV1{
				ID:          provider.ID,
				Name:        provider.Name,
				Description: provider.Description,
				Tenant:      tenant,
			})
		}
	}

	return zw.writeJsonFile(accountProviderFile, jsonData)
}

func writeAccounts(ctx context.Context, zw *zipWriter, store *accounting.Store, tenants []string) error {
	var jsonData []accountV1
	for _, tenant := range tenants {
		Accounts, err := store.ListAccounts(ctx, tenant)
		if err != nil {
			return err
		}
		for _, acc := range Accounts {
			jsonData = append(jsonData, accountV1{
				ID:                acc.ID,
				AccountProviderID: acc.AccountProviderID,
				Name:              acc.Name,
				Description:       acc.Description,
				Currency:          acc.Currency.String(),
				Type:              acc.Type.String(),
				Tenant:            tenant,
			})
		}
	}

	return zw.writeJsonFile(accountsFile, jsonData)
}

func writeCategories(ctx context.Context, zw *zipWriter, store *accounting.Store, tenants []string) error {

	var jsonData []categoryV1
	for _, tenant := range tenants {
		Incomes, err := store.ListDescendantCategories(ctx, 0, -1, accounting.IncomeCategory, tenant)
		if err != nil {
			return err
		}
		for _, income := range Incomes {
			jsonData = append(jsonData, categoryV1{
				ID:          income.Id,
				ParentId:    income.ParentId,
				Name:        income.Name,
				Description: income.Description,
				Tenant:      tenant,
			})
		}
		err = zw.writeJsonFile(incomeCategoriesFile, jsonData)
		if err != nil {
			return err
		}
	}
	jsonData = []categoryV1{}
	for _, tenant := range tenants {
		expenses, err := store.ListDescendantCategories(ctx, 0, -1, accounting.ExpenseCategory, tenant)
		if err != nil {
			return err
		}
		for _, income := range expenses {
			jsonData = append(jsonData, categoryV1{
				ID:          income.Id,
				ParentId:    income.ParentId,
				Name:        income.Name,
				Description: income.Description,
				Tenant:      tenant,
			})
		}
	}
	return zw.writeJsonFile(expenseCategoriesFile, jsonData)
}

func dataFuture() time.Time {
	// Parse the string based on the provided layout
	parsedTime, err := time.Parse("2006-01-02", "3000-01-01")
	if err != nil {
		panic(fmt.Errorf("unable to parse time: %v", err))
	}
	return parsedTime
}

var entriesLimit = 100

func writeTransactions(ctx context.Context, zw *zipWriter, store *accounting.Store, tenants []string) error {
	var jsonData []TransactionV1

	for _, tenant := range tenants {
		opts := accounting.ListOpts{
			EndDate: dataFuture(),
			Types: []accounting.TxType{
				accounting.ExpenseTransaction,
				accounting.IncomeTransaction,
				accounting.TransferTransaction,
				accounting.StockTransaction,
				accounting.LoanTransaction,
			},
			Limit: entriesLimit,
			Page:  1,
		}

		for {
			transactions, err := store.ListTransactions(ctx, opts, tenant)
			if err != nil {
				return fmt.Errorf("failed to list transactions for tenant %s (page %d): %w", tenant, opts.Page, err)
			}

			// Break when no more transactions
			if len(transactions) == 0 {
				break
			}

			for _, tx := range transactions {
				switch item := tx.(type) {
				case accounting.Transfer:
					payload := TransactionV1{
						Id:              item.Id,
						Description:     item.Description,
						OriginAmount:    item.OriginAmount,
						OriginAccountID: item.OriginAccountID,
						TargetAmount:    item.TargetAmount,
						TargetAccountID: item.TargetAccountID,
						Date:            item.Date,
						Tenant:          tenant,
						Type:            txTypeTransfer,
					}
					jsonData = append(jsonData, payload)
				case accounting.Income:
					payload := TransactionV1{
						Id:          item.Id,
						Description: item.Description,
						Amount:      item.Amount,
						AccountID:   item.AccountID,
						CategoryID:  item.CategoryID,
						Date:        item.Date,
						Tenant:      tenant,
						Type:        txTypeIncome,
					}
					jsonData = append(jsonData, payload)
				case accounting.Expense:
					payload := TransactionV1{
						Id:          item.Id,
						Description: item.Description,
						Amount:      item.Amount,
						AccountID:   item.AccountID,
						CategoryID:  item.CategoryID,
						Date:        item.Date,
						Tenant:      tenant,
						Type:        txTypeExpense,
					}
					jsonData = append(jsonData, payload)
				}
			}
			opts.Page++

			// Stop early if fewer than limit results (no more pages)
			if len(transactions) < opts.Limit {
				break
			}
		}
	}

	return zw.writeJsonFile(transactionsFile, jsonData)
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
