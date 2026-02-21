package backup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/andresbott/etna/internal/accounting"
)

const timeFormat = "2006-01-02_15-04-05"

func verifyZipPath(zipFile string) error {
	if !strings.HasSuffix(strings.ToLower(zipFile), ".zip") {
		return errors.New("invalid file extension: must be a .zip file")
	}

	dir := filepath.Dir(zipFile)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return errors.New("directory does not exist: " + dir)
	}

	if _, err := os.Stat(zipFile); err == nil {
		return errors.New("file already exists: " + zipFile)
	} else if !os.IsNotExist(err) {

		return err
	}
	return nil
}

func ExportToFile(ctx context.Context, store *accounting.Store, zipFile string) error {
	err := verifyZipPath(zipFile)
	if err != nil {
		return err
	}
	return export(ctx, store, zipFile)
}

func export(ctx context.Context, store *accounting.Store, fullPath string) error {
	if store == nil {
		return errors.New("finance store was not initialized")
	}

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

	err = writeMeta(zw, time.Now())
	if err != nil {
		return err
	}

	err = writeAccountProviders(ctx, zw, store)
	if err != nil {
		return err
	}

	err = writeAccounts(ctx, zw, store)
	if err != nil {
		return err
	}

	err = writeCategories(ctx, zw, store)
	if err != nil {
		return err
	}

	err = writeTransactions(ctx, zw, store)
	if err != nil {
		return err
	}

	return nil
}

func writeMeta(zw *zipWriter, timestamp time.Time) error {
	meta := metaInfoV1{
		Version: SchemaV1,
		Date:    timestamp.Format(timeFormat),
	}
	return zw.writeJsonFile(metaInfoFile, meta)
}

func writeAccountProviders(ctx context.Context, zw *zipWriter, store *accounting.Store) error {
	providers, err := store.ListAccountsProvider(ctx, false)
	if err != nil {
		return err
	}
	jsonData := make([]accountProviderV1, len(providers))
	for i, provider := range providers {
		jsonData[i] = accountProviderV1{
			ID:          provider.ID,
			Name:        provider.Name,
			Description: provider.Description,
		}
	}
	return zw.writeJsonFile(accountProviderFile, jsonData)
}

func writeAccounts(ctx context.Context, zw *zipWriter, store *accounting.Store) error {
	accounts, err := store.ListAccounts(ctx)
	if err != nil {
		return err
	}
	jsonData := make([]accountV1, len(accounts))
	for i, acc := range accounts {
		jsonData[i] = accountV1{
			ID:                acc.ID,
			AccountProviderID: acc.AccountProviderID,
			Name:              acc.Name,
			Description:       acc.Description,
			Currency:          acc.Currency.String(),
			Type:              acc.Type.String(),
		}
	}
	return zw.writeJsonFile(accountsFile, jsonData)
}

func writeCategories(ctx context.Context, zw *zipWriter, store *accounting.Store) error {
	incomes, err := store.ListDescendantCategories(ctx, 0, -1, accounting.IncomeCategory)
	if err != nil {
		return err
	}
	jsonData := make([]categoryV1, len(incomes))
	for i, income := range incomes {
		jsonData[i] = categoryV1{
			ID:          income.Id,
			ParentId:    income.ParentId,
			Name:        income.Name,
			Description: income.Description,
		}
	}
	if err := zw.writeJsonFile(incomeCategoriesFile, jsonData); err != nil {
		return err
	}

	expenses, err := store.ListDescendantCategories(ctx, 0, -1, accounting.ExpenseCategory)
	if err != nil {
		return err
	}
	jsonData = make([]categoryV1, len(expenses))
	for i, exp := range expenses {
		jsonData[i] = categoryV1{
			ID:          exp.Id,
			ParentId:    exp.ParentId,
			Name:        exp.Name,
			Description: exp.Description,
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

func writeTransactions(ctx context.Context, zw *zipWriter, store *accounting.Store) error {
	jsonData := []TransactionV1{}
	opts := accounting.ListOpts{
		EndDate: dataFuture(),
		Types: []accounting.TxType{
			accounting.ExpenseTransaction,
			accounting.IncomeTransaction,
			accounting.TransferTransaction,
			accounting.StockBuyTransaction,
			accounting.StockSellTransaction,
			accounting.StockGrantTransaction,
			accounting.StockTransferTransaction,
			accounting.LoanTransaction,
		},
		Limit: entriesLimit,
		Page:  1,
	}

	for {
		transactions, err := store.ListTransactions(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to list transactions (page %d): %w", opts.Page, err)
		}

		if len(transactions) == 0 {
			break
		}

		for _, tx := range transactions {
			switch item := tx.(type) {
			case accounting.Transfer:
				jsonData = append(jsonData, TransactionV1{
					Id:              item.Id,
					Description:     item.Description,
					OriginAmount:    item.OriginAmount,
					OriginAccountID: item.OriginAccountID,
					TargetAmount:    item.TargetAmount,
					TargetAccountID: item.TargetAccountID,
					Date:            item.Date,
					Type:            txTypeTransfer,
				})
			case accounting.Income:
				jsonData = append(jsonData, TransactionV1{
					Id:          item.Id,
					Description: item.Description,
					Amount:      item.Amount,
					AccountID:   item.AccountID,
					CategoryID:  item.CategoryID,
					Date:        item.Date,
					Type:        txTypeIncome,
				})
			case accounting.Expense:
				jsonData = append(jsonData, TransactionV1{
					Id:          item.Id,
					Description: item.Description,
					Amount:      item.Amount,
					AccountID:   item.AccountID,
					CategoryID:  item.CategoryID,
					Date:        item.Date,
					Type:        txTypeExpense,
				})
			}
		}
		opts.Page++

		if len(transactions) < opts.Limit {
			break
		}
	}
	return zw.writeJsonFile(transactionsFile, jsonData)
}
