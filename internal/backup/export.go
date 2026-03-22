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
	"github.com/andresbott/etna/internal/csvimport"
	"github.com/andresbott/etna/internal/marketdata"
	"github.com/andresbott/etna/internal/toolsdata"
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

func ExportToFile(ctx context.Context, store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, tdStore *toolsdata.Store, zipFile string) error {
	err := verifyZipPath(zipFile)
	if err != nil {
		return err
	}
	return export(ctx, store, mdStore, csvStore, tdStore, zipFile)
}

func export(ctx context.Context, store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, tdStore *toolsdata.Store, fullPath string) error {
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

	err = writeInstruments(ctx, zw, mdStore)
	if err != nil {
		return err
	}

	err = writePriceHistory(ctx, zw, mdStore)
	if err != nil {
		return err
	}

	err = writeFXRates(ctx, zw, mdStore)
	if err != nil {
		return err
	}

	err = writeImportProfiles(ctx, zw, csvStore)
	if err != nil {
		return err
	}

	err = writeCategoryRules(ctx, zw, csvStore)
	if err != nil {
		return err
	}

	err = writeCaseStudies(ctx, zw, tdStore)
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
			Icon:        provider.Icon,
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
			Icon:              acc.Icon,
			Notes:             acc.Notes,
			Currency:          acc.Currency.String(),
			Type:              acc.Type.String(),
			ImportProfileID:   acc.ImportProfileID,
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
			Icon:        income.Icon,
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
			Icon:        exp.Icon,
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
		transactions, _, err := store.ListTransactions(ctx, opts)
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
					Notes:           item.Notes,
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
					Notes:       item.Notes,
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
					Notes:       item.Notes,
					Amount:      item.Amount,
					AccountID:   item.AccountID,
					CategoryID:  item.CategoryID,
					Date:        item.Date,
					Type:        txTypeExpense,
				})
			case accounting.StockBuy:
				jsonData = append(jsonData, TransactionV1{
					Id:                  item.Id,
					Description:         item.Description,
					Notes:               item.Notes,
					InstrumentID:        item.InstrumentID,
					Quantity:            item.Quantity,
					TotalAmount:         item.TotalAmount,
					StockAmount:         item.StockAmount,
					InvestmentAccountID: item.InvestmentAccountID,
					CashAccountID:       item.CashAccountID,
					Date:                item.Date,
					Type:                txTypeStockBuy,
				})
			case accounting.StockSell:
				jsonData = append(jsonData, TransactionV1{
					Id:                  item.Id,
					Description:         item.Description,
					Notes:               item.Notes,
					InstrumentID:        item.InstrumentID,
					Quantity:            item.Quantity,
					TotalAmount:         item.TotalAmount,
					Fees:                item.Fees,
					InvestmentAccountID: item.InvestmentAccountID,
					CashAccountID:       item.CashAccountID,
					Date:                item.Date,
					Type:                txTypeStockSell,
				})
			case accounting.StockGrant:
				jsonData = append(jsonData, TransactionV1{
					Id:              item.Id,
					Description:     item.Description,
					Notes:           item.Notes,
					AccountID:       item.AccountID,
					InstrumentID:    item.InstrumentID,
					Quantity:        item.Quantity,
					FairMarketValue: item.FairMarketValue,
					Date:            item.Date,
					Type:            txTypeStockGrant,
				})
			case accounting.StockTransfer:
				jsonData = append(jsonData, TransactionV1{
					Id:              item.Id,
					Description:     item.Description,
					Notes:           item.Notes,
					SourceAccountID: item.SourceAccountID,
					TargetAccountID: item.TargetAccountID,
					InstrumentID:    item.InstrumentID,
					Quantity:        item.Quantity,
					Date:            item.Date,
					Type:            txTypeStockTransfer,
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

func writeInstruments(ctx context.Context, zw *zipWriter, mdStore *marketdata.Store) error {
	instruments, err := mdStore.ListInstruments(ctx)
	if err != nil {
		return err
	}
	jsonData := make([]instrumentV1, len(instruments))
	for i, inst := range instruments {
		jsonData[i] = instrumentV1{
			ID:                   inst.ID,
			InstrumentProviderID: inst.InstrumentProviderID,
			Symbol:               inst.Symbol,
			Name:                 inst.Name,
			Currency:             inst.Currency.String(),
		}
	}
	return zw.writeJsonFile(instrumentsFile, jsonData)
}

func writePriceHistory(ctx context.Context, zw *zipWriter, mdStore *marketdata.Store) error {
	symbols, err := mdStore.ListPriceSymbols()
	if err != nil {
		return err
	}
	var jsonData []priceRecordV1
	for _, symbol := range symbols {
		records, err := mdStore.PriceHistory(ctx, symbol, time.Time{}, time.Time{})
		if err != nil {
			return fmt.Errorf("failed to get price history for %s: %w", symbol, err)
		}
		for _, rec := range records {
			jsonData = append(jsonData, priceRecordV1{
				Symbol: rec.Symbol,
				Time:   rec.Time,
				Price:  rec.Price,
			})
		}
	}
	if jsonData == nil {
		jsonData = []priceRecordV1{}
	}
	return zw.writeJsonFile(priceHistoryFile, jsonData)
}

func writeFXRates(ctx context.Context, zw *zipWriter, mdStore *marketdata.Store) error {
	pairs, err := mdStore.ListFXPairs()
	if err != nil {
		return err
	}
	var jsonData []fxRateRecordV1
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "/", 2)
		if len(parts) != 2 {
			continue
		}
		records, err := mdStore.RateHistory(ctx, parts[0], parts[1], time.Time{}, time.Time{})
		if err != nil {
			return fmt.Errorf("failed to get FX history for %s: %w", pair, err)
		}
		for _, rec := range records {
			jsonData = append(jsonData, fxRateRecordV1{
				Main:      rec.Main,
				Secondary: rec.Secondary,
				Time:      rec.Time,
				Rate:      rec.Rate,
			})
		}
	}
	if jsonData == nil {
		jsonData = []fxRateRecordV1{}
	}
	return zw.writeJsonFile(fxRatesFile, jsonData)
}

func writeImportProfiles(ctx context.Context, zw *zipWriter, csvStore *csvimport.Store) error {
	profiles, err := csvStore.ListProfiles(ctx)
	if err != nil {
		return err
	}
	jsonData := make([]importProfileV1, len(profiles))
	for i, p := range profiles {
		jsonData[i] = importProfileV1{
			ID:                p.ID,
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
	}
	return zw.writeJsonFile(importProfilesFile, jsonData)
}

func writeCategoryRules(ctx context.Context, zw *zipWriter, csvStore *csvimport.Store) error {
	groups, err := csvStore.ListCategoryRuleGroups(ctx)
	if err != nil {
		return err
	}
	jsonData := make([]categoryRuleGroupV1, len(groups))
	for i, g := range groups {
		patterns := make([]categoryRulePatternV1, len(g.Patterns))
		for j, p := range g.Patterns {
			patterns[j] = categoryRulePatternV1{
				ID:      p.ID,
				Pattern: p.Pattern,
				IsRegex: p.IsRegex,
			}
		}
		jsonData[i] = categoryRuleGroupV1{
			ID:         g.ID,
			Name:       g.Name,
			CategoryID: g.CategoryID,
			Priority:   g.Priority,
			Patterns:   patterns,
		}
	}
	return zw.writeJsonFile(categoryRulesFile, jsonData)
}

func writeCaseStudies(ctx context.Context, zw *zipWriter, tdStore *toolsdata.Store) error {
	if tdStore == nil {
		return zw.writeJsonFile(caseStudiesFile, []caseStudyV1{})
	}
	studies, err := tdStore.ListAll(ctx)
	if err != nil {
		return err
	}
	jsonData := make([]caseStudyV1, len(studies))
	for i, cs := range studies {
		jsonData[i] = caseStudyV1{
			ID:                   cs.ID,
			ToolType:             cs.ToolType,
			Name:                 cs.Name,
			Description:          cs.Description,
			ExpectedAnnualReturn: cs.ExpectedAnnualReturn,
			Params:               cs.Params,
		}
	}
	return zw.writeJsonFile(caseStudiesFile, jsonData)
}
