package backup

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/andresbott/etna/internal/accounting"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/text/currency"
	"io"
	"os"
	"path/filepath"
)

func Import(ctx context.Context, store *accounting.Store, file string) error {
	zipPath, err := checkZip(file)
	if err != nil {
		return err
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer func() {
		_ = r.Close()
	}()

	metaInfo, err := loadV1Json[metaInfoV1](r, metaInfoFile)
	if err != nil {
		return err
	}
	// in the future implement further importers
	if metaInfo.Version != "1.0.0" {
		return fmt.Errorf("unsuported backup shema: %s", metaInfo.Version)
	}

	err = store.WipeData(ctx)
	if err != nil {
		return err
	}

	accountsMap, err := importAccounts(ctx, store, r)
	if err != nil {
		return err
	}
	inMap, exMap, err := importCategories(ctx, store, r)
	if err != nil {
		return err
	}
	err = importTransactions(ctx, store, r, accountsMap, inMap, exMap)
	if err != nil {
		return err
	}

	return nil
}

func importAccounts(ctx context.Context, store *accounting.Store, r *zip.ReadCloser) (map[uint]uint, error) {
	providers, err := loadV1Json[[]accountProviderV1](r, accountProviderFile)
	if err != nil {
		return nil, err
	}
	// map the provider id from the zip to the one created in the DB
	providersMap := map[uint]uint{}
	for _, provider := range providers {
		item := accounting.AccountProvider{Name: provider.Name, Description: provider.Description}
		prId, err := store.CreateAccountProvider(ctx, item, provider.Tenant)
		if err != nil {
			return nil, fmt.Errorf("failed to create account provider: %w", err)
		}
		providersMap[provider.ID] = prId
	}

	accounts, err := loadV1Json[[]accountV1](r, accountsFile)
	if err != nil {
		return nil, err
	}
	// map the accountIds from the zip to the one created in the DB
	accountsMap := map[uint]uint{}
	for _, account := range accounts {
		item := accounting.Account{Name: account.Name, Description: account.Description}
		item.AccountProviderID = providersMap[account.AccountProviderID]

		cur, err := currency.ParseISO(account.Currency)
		if err != nil {
			return nil, fmt.Errorf("failed to parse currency %s: %w", account.Currency, err)
		}
		item.Currency = cur

		t := parseAccountType(account.Type)
		if t == accounting.UnknownAccountType {
			return nil, fmt.Errorf("unable to parse account type, got unexpected %s", account.Type)
		}
		item.Type = t

		accId, err := store.CreateAccount(ctx, item, account.Tenant)
		if err != nil {
			return nil, fmt.Errorf("failed to create account %s: %w", account.Tenant, err)
		}
		accountsMap[account.ID] = accId
	}
	return accountsMap, nil
}

func parseAccountType(in string) accounting.AccountType {
	switch in {
	case "Cash":
		return accounting.CashAccountType
	case "Checkin":
		return accounting.CheckinAccountType
	case "Savings":
		return accounting.SavingsAccountType
	case "Investment":
		return accounting.InvestmentAccountType
	default:
		return accounting.UnknownAccountType
	}
}

func importCategories(ctx context.Context, store *accounting.Store, r *zip.ReadCloser) (map[uint]uint, map[uint]uint, error) {
	incomes, err := loadV1Json[[]categoryV1](r, incomeCategoriesFile)
	if err != nil {
		return nil, nil, err
	}
	incomeCategoriesMap := map[uint]uint{}
	err = createCategoriesRecursive(ctx, store, incomes, 0, accounting.IncomeCategory, &incomeCategoriesMap)
	if err != nil {
		return nil, nil, err
	}

	expenses, err := loadV1Json[[]categoryV1](r, expenseCategoriesFile)
	if err != nil {
		return nil, nil, err
	}
	expenseCategoriesMap := map[uint]uint{}
	err = createCategoriesRecursive(ctx, store, expenses, 0, accounting.ExpenseCategory, &expenseCategoriesMap)
	if err != nil {
		return nil, nil, err
	}

	return incomeCategoriesMap, expenseCategoriesMap, nil
}

// Recursive function to create categories
func createCategoriesRecursive(ctx context.Context, store *accounting.Store, categories []categoryV1, parentID uint, t accounting.CategoryType, categoriesMap *map[uint]uint) error {
	for _, cat := range categories {
		if cat.ParentId == parentID {
			data := accounting.CategoryData{
				Name:        cat.Name,
				Description: cat.Description,
				Type:        t,
			}

			newParent := uint(0)
			if parentID != 0 {
				newParent = (*categoriesMap)[parentID]
			}

			newID, err := store.CreateCategory(ctx, data, newParent, cat.Tenant)
			if err != nil {
				return err
			}
			(*categoriesMap)[cat.ID] = newID

			if err := createCategoriesRecursive(ctx, store, categories, cat.ID, t, categoriesMap); err != nil {
				return err
			}
		}
	}
	return nil
}

func importTransactions(ctx context.Context, store *accounting.Store, r *zip.ReadCloser, accountsMap, incomeMap, expenseMap map[uint]uint) error {
	txs, err := loadV1Json[[]TransactionV1](r, transactionsFile)
	if err != nil {
		return err
	}

	for _, tx := range txs {
		var item accounting.Transaction
		switch tx.Type {
		case txTypeIncome:
			in := accounting.Income{
				Description: tx.Description, Amount: tx.Amount, CategoryID: tx.CategoryID, Date: tx.Date,
			}
			in.AccountID = accountsMap[tx.AccountID]
			in.CategoryID = incomeMap[tx.CategoryID]
			item = in

		case txTypeExpense:
			ex := accounting.Expense{
				Description: tx.Description, Amount: tx.Amount, CategoryID: tx.CategoryID, Date: tx.Date,
			}
			ex.AccountID = accountsMap[tx.AccountID]
			ex.CategoryID = expenseMap[tx.CategoryID]
			item = ex
		case txTypeTransfer:
			tr := accounting.Transfer{
				Description: tx.Description, OriginAmount: tx.OriginAmount, TargetAmount: tx.TargetAmount, Date: tx.Date,
			}
			tr.OriginAccountID = accountsMap[tx.OriginAccountID]
			tr.TargetAccountID = accountsMap[tx.TargetAccountID]

			item = tr
		}

		_, err = store.CreateTransaction(ctx, item, tx.Tenant)
		if err != nil {
			return fmt.Errorf("failed to create transaction: %w", err)
		}

	}
	return nil
}

// Load V1 data from json files
func loadV1Json[T metaInfoV1 | []accountProviderV1 | []accountV1 | []categoryV1 | []TransactionV1](r *zip.ReadCloser, fileName string) (T, error) {
	var result T

	for _, f := range r.File {
		if f.Name != fileName {
			continue
		}

		fn := func() (T, error) {
			rc, err := f.Open()
			if err != nil {
				return result, fmt.Errorf("failed to open file %s: %w", f.Name, err)
			}
			defer func() {
				cerr := rc.Close()
				if cerr != nil {
					err = multierror.Append(err, cerr)
				}
			}()

			data, err := io.ReadAll(rc)
			if err != nil {
				return result, fmt.Errorf("failed to read file %s: %w", f.Name, err)
			}

			if err := json.Unmarshal(data, &result); err != nil {
				return result, fmt.Errorf("failed to unmarshal json: %w", err)
			}

			return result, nil
		}

		return fn()
	}

	return result, fmt.Errorf("file %s not found in zip", fileName)
}

// checkZip validates that the given path is a zip file
// and returns its absolute path.
func checkZip(path string) (string, error) {
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

	if info.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file: %s", absPath)
	}

	if filepath.Ext(absPath) != ".zip" {
		return "", fmt.Errorf("path is not a zip file: %s", absPath)
	}

	return absPath, nil
}
