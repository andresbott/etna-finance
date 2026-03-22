package backup

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/andresbott/etna/internal/accounting"
	"github.com/andresbott/etna/internal/csvimport"
	"github.com/andresbott/etna/internal/marketdata"
	"github.com/andresbott/etna/internal/toolsdata"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/text/currency"
)

func Import(ctx context.Context, store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, tdStore *toolsdata.Store, file string) error {
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
	err = mdStore.WipeData(ctx)
	if err != nil {
		return err
	}
	err = csvStore.WipeData(ctx)
	if err != nil {
		return err
	}
	if tdStore != nil {
		err = tdStore.WipeData(ctx)
		if err != nil {
			return err
		}
	}

	profilesMap, err := importProfiles(ctx, csvStore, r)
	if err != nil {
		return err
	}

	accountsMap, err := importAccounts(ctx, store, r, profilesMap)
	if err != nil {
		return err
	}
	inMap, exMap, err := importCategories(ctx, store, r)
	if err != nil {
		return err
	}

	instrumentsMap, err := importInstruments(ctx, mdStore, r)
	if err != nil {
		return err
	}

	err = importTransactions(ctx, store, r, accountsMap, inMap, exMap, instrumentsMap)
	if err != nil {
		return err
	}

	err = importPriceHistory(ctx, mdStore, r)
	if err != nil {
		return err
	}

	err = importFXRates(ctx, mdStore, r)
	if err != nil {
		return err
	}

	err = importCategoryRules(ctx, csvStore, r, inMap, exMap)
	if err != nil {
		return err
	}

	err = importCaseStudies(ctx, tdStore, r)
	if err != nil {
		return err
	}

	return nil
}

func importAccounts(ctx context.Context, store *accounting.Store, r *zip.ReadCloser, profilesMap map[uint]uint) (map[uint]uint, error) {
	providers, err := loadV1Json[[]accountProviderV1](r, accountProviderFile)
	if err != nil {
		return nil, err
	}
	// map the provider id from the zip to the one created in the DB
	providersMap := map[uint]uint{}
	for _, provider := range providers {
		item := accounting.AccountProvider{Name: provider.Name, Description: provider.Description, Icon: provider.Icon}
		prId, err := store.CreateAccountProvider(ctx, item)
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
		item := accounting.Account{Name: account.Name, Description: account.Description, Icon: account.Icon, Notes: account.Notes}
		item.AccountProviderID = providersMap[account.AccountProviderID]
		if account.ImportProfileID != 0 {
			item.ImportProfileID = profilesMap[account.ImportProfileID]
		}

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

		accId, err := store.CreateAccount(ctx, item)
		if err != nil {
			return nil, fmt.Errorf("failed to create account: %w", err)
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
	case "Unvested":
		return accounting.UnvestedAccountType
	case "Lent":
		return accounting.LentAccountType
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
				Icon:        cat.Icon,
				Type:        t,
			}

			newParent := uint(0)
			if parentID != 0 {
				newParent = (*categoriesMap)[parentID]
			}

			newID, err := store.CreateCategory(ctx, data, newParent)
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

func importTransactions(ctx context.Context, store *accounting.Store, r *zip.ReadCloser, accountsMap, incomeMap, expenseMap, instrumentsMap map[uint]uint) error {
	txs, err := loadV1Json[[]TransactionV1](r, transactionsFile)
	if err != nil {
		return err
	}

	for _, tx := range txs {
		var item accounting.Transaction
		switch tx.Type {
		case txTypeIncome:
			in := accounting.Income{
				Description: tx.Description, Notes: tx.Notes, Amount: tx.Amount, CategoryID: tx.CategoryID, Date: tx.Date,
			}
			in.AccountID = accountsMap[tx.AccountID]
			in.CategoryID = incomeMap[tx.CategoryID]
			item = in

		case txTypeExpense:
			ex := accounting.Expense{
				Description: tx.Description, Notes: tx.Notes, Amount: tx.Amount, CategoryID: tx.CategoryID, Date: tx.Date,
			}
			ex.AccountID = accountsMap[tx.AccountID]
			ex.CategoryID = expenseMap[tx.CategoryID]
			item = ex
		case txTypeTransfer:
			tr := accounting.Transfer{
				Description: tx.Description, Notes: tx.Notes, OriginAmount: tx.OriginAmount, TargetAmount: tx.TargetAmount, Date: tx.Date,
			}
			tr.OriginAccountID = accountsMap[tx.OriginAccountID]
			tr.TargetAccountID = accountsMap[tx.TargetAccountID]
			item = tr

		case txTypeStockBuy:
			item = accounting.StockBuy{
				Description:         tx.Description,
				Notes:               tx.Notes,
				Date:                tx.Date,
				InvestmentAccountID: accountsMap[tx.InvestmentAccountID],
				CashAccountID:       accountsMap[tx.CashAccountID],
				InstrumentID:        instrumentsMap[tx.InstrumentID],
				Quantity:            tx.Quantity,
				TotalAmount:         tx.TotalAmount,
				StockAmount:         tx.StockAmount,
			}
		case txTypeStockSell:
			item = accounting.StockSell{
				Description:         tx.Description,
				Notes:               tx.Notes,
				Date:                tx.Date,
				InvestmentAccountID: accountsMap[tx.InvestmentAccountID],
				CashAccountID:       accountsMap[tx.CashAccountID],
				InstrumentID:        instrumentsMap[tx.InstrumentID],
				Quantity:            tx.Quantity,
				TotalAmount:         tx.TotalAmount,
				Fees:                tx.Fees,
			}
		case txTypeStockGrant:
			item = accounting.StockGrant{
				Description:     tx.Description,
				Notes:           tx.Notes,
				Date:            tx.Date,
				AccountID:       accountsMap[tx.AccountID],
				InstrumentID:    instrumentsMap[tx.InstrumentID],
				Quantity:        tx.Quantity,
				FairMarketValue: tx.FairMarketValue,
			}
		case txTypeStockTransfer:
			item = accounting.StockTransfer{
				Description:     tx.Description,
				Notes:           tx.Notes,
				Date:            tx.Date,
				SourceAccountID: accountsMap[tx.SourceAccountID],
				TargetAccountID: accountsMap[tx.TargetAccountID],
				InstrumentID:    instrumentsMap[tx.InstrumentID],
				Quantity:        tx.Quantity,
			}
		}

		if item == nil {
			continue
		}
		_, err = store.CreateTransaction(ctx, item)
		if err != nil {
			return fmt.Errorf("failed to create transaction: %w", err)
		}
	}
	return nil
}

// Load V1 data from json files
func loadV1Json[T metaInfoV1 | []accountProviderV1 | []accountV1 | []categoryV1 | []TransactionV1 | []instrumentV1 | []priceRecordV1 | []fxRateRecordV1 | []importProfileV1 | []categoryRuleGroupV1 | []caseStudyV1](r *zip.ReadCloser, fileName string) (T, error) {
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

func importInstruments(ctx context.Context, mdStore *marketdata.Store, r *zip.ReadCloser) (map[uint]uint, error) {
	instruments, err := loadV1Json[[]instrumentV1](r, instrumentsFile)
	if err != nil {
		return nil, err
	}
	instrumentsMap := map[uint]uint{}
	for _, inst := range instruments {
		cur, err := currency.ParseISO(inst.Currency)
		if err != nil {
			return nil, fmt.Errorf("failed to parse currency %s: %w", inst.Currency, err)
		}
		item := marketdata.Instrument{
			InstrumentProviderID: inst.InstrumentProviderID,
			Symbol:               inst.Symbol,
			Name:                 inst.Name,
			Currency:             cur,
		}
		newID, err := mdStore.CreateInstrument(ctx, item)
		if err != nil {
			return nil, fmt.Errorf("failed to create instrument: %w", err)
		}
		instrumentsMap[inst.ID] = newID
	}
	return instrumentsMap, nil
}

func importPriceHistory(ctx context.Context, mdStore *marketdata.Store, r *zip.ReadCloser) error {
	records, err := loadV1Json[[]priceRecordV1](r, priceHistoryFile)
	if err != nil {
		return err
	}
	bySymbol := map[string][]marketdata.PricePoint{}
	for _, rec := range records {
		bySymbol[rec.Symbol] = append(bySymbol[rec.Symbol], marketdata.PricePoint{
			Time:  rec.Time,
			Price: rec.Price,
		})
	}
	for symbol, points := range bySymbol {
		if err := mdStore.IngestPricesBulk(ctx, symbol, points); err != nil {
			return fmt.Errorf("failed to ingest prices for %s: %w", symbol, err)
		}
	}
	return nil
}

func importFXRates(ctx context.Context, mdStore *marketdata.Store, r *zip.ReadCloser) error {
	records, err := loadV1Json[[]fxRateRecordV1](r, fxRatesFile)
	if err != nil {
		return err
	}
	type pair struct{ main, secondary string }
	byPair := map[pair][]marketdata.RatePoint{}
	for _, rec := range records {
		key := pair{rec.Main, rec.Secondary}
		byPair[key] = append(byPair[key], marketdata.RatePoint{
			Time: rec.Time,
			Rate: rec.Rate,
		})
	}
	for p, points := range byPair {
		if err := mdStore.IngestRatesBulk(ctx, p.main, p.secondary, points); err != nil {
			return fmt.Errorf("failed to ingest FX rates for %s/%s: %w", p.main, p.secondary, err)
		}
	}
	return nil
}

func importProfiles(ctx context.Context, csvStore *csvimport.Store, r *zip.ReadCloser) (map[uint]uint, error) {
	profiles, err := loadV1Json[[]importProfileV1](r, importProfilesFile)
	if err != nil {
		return nil, err
	}
	profilesMap := map[uint]uint{}
	for _, p := range profiles {
		item := csvimport.ImportProfile{
			Name:              p.Name,
			CsvSeparator:      p.CsvSeparator,
			SkipRows:          p.SkipRows,
			DateColumn:        p.DateColumn,
			DateFormat:        p.DateFormat,
			DescriptionColumn: p.DescriptionColumn,
			AmountColumn:      p.AmountColumn,
			AmountMode:        p.AmountMode,
			CreditColumn:      p.CreditColumn,
			DebitColumn:       p.DebitColumn,
		}
		newID, err := csvStore.CreateProfile(ctx, item)
		if err != nil {
			return nil, fmt.Errorf("failed to create import profile: %w", err)
		}
		profilesMap[p.ID] = newID
	}
	return profilesMap, nil
}

func importCaseStudies(ctx context.Context, tdStore *toolsdata.Store, r *zip.ReadCloser) error {
	if tdStore == nil {
		return nil
	}
	studies, err := loadV1Json[[]caseStudyV1](r, caseStudiesFile)
	if err != nil {
		// Old backups may not have this file; skip gracefully.
		if strings.Contains(err.Error(), "not found in zip") {
			return nil
		}
		return err
	}
	for _, s := range studies {
		_, err := tdStore.Create(ctx, toolsdata.CaseStudy{
			ToolType:             s.ToolType,
			Name:                 s.Name,
			Description:          s.Description,
			ExpectedAnnualReturn: s.ExpectedAnnualReturn,
			Params:               s.Params,
		})
		if err != nil {
			return fmt.Errorf("failed to create case study: %w", err)
		}
	}
	return nil
}

func importCategoryRules(ctx context.Context, csvStore *csvimport.Store, r *zip.ReadCloser, incomeMap, expenseMap map[uint]uint) error {
	groups, err := loadV1Json[[]categoryRuleGroupV1](r, categoryRulesFile)
	if err != nil {
		return err
	}
	for _, g := range groups {
		catID := incomeMap[g.CategoryID]
		if catID == 0 {
			catID = expenseMap[g.CategoryID]
		}
		item := csvimport.CategoryRuleGroup{
			Name:       g.Name,
			CategoryID: catID,
			Priority:   g.Priority,
		}
		for _, p := range g.Patterns {
			item.Patterns = append(item.Patterns, csvimport.CategoryRulePattern{
				Pattern: p.Pattern,
				IsRegex: p.IsRegex,
			})
		}
		_, err := csvStore.CreateCategoryRuleGroup(ctx, item)
		if err != nil {
			return fmt.Errorf("failed to create category rule group: %w", err)
		}
	}
	return nil
}
