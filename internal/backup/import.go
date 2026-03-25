package backup

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/andresbott/etna/internal/accounting"
	"github.com/andresbott/etna/internal/csvimport"
	"github.com/andresbott/etna/internal/filestore"
	"github.com/andresbott/etna/internal/marketdata"
	"github.com/andresbott/etna/internal/toolsdata"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/text/currency"
)

func wipeStores(ctx context.Context, store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, fileStore *filestore.Store, tdStore *toolsdata.Store) error {
	if err := store.WipeData(ctx); err != nil {
		return err
	}
	if err := mdStore.WipeData(ctx); err != nil {
		return err
	}
	if err := csvStore.WipeData(ctx); err != nil {
		return err
	}
	if tdStore != nil {
		if err := tdStore.WipeData(ctx); err != nil {
			return err
		}
	}
	if fileStore != nil {
		if err := fileStore.WipeData(ctx); err != nil {
			return err
		}
	}
	return nil
}

func Import(ctx context.Context, store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, fileStore *filestore.Store, tdStore *toolsdata.Store, file string) error {
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
	if metaInfo.Version != "1.0.0" {
		return fmt.Errorf("unsuported backup shema: %s", metaInfo.Version)
	}

	if err := wipeStores(ctx, store, mdStore, csvStore, fileStore, tdStore); err != nil {
		return err
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

	attachmentsMap, err := importAttachments(ctx, fileStore, r)
	if err != nil {
		return err
	}

	err = importTransactions(ctx, store, r, accountsMap, inMap, exMap, instrumentsMap, attachmentsMap)
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

	return importCaseStudies(ctx, tdStore, r)
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
	case "RestrictedStock", "Unvested":
		return accounting.RestrictedStockAccountType
	case "Lent":
		return accounting.LentAccountType
	case "Pension":
		return accounting.PensionAccountType
	case "PrepaidExpense":
		return accounting.PrepaidExpenseAccountType
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

type importMaps struct {
	accounts    map[uint]uint
	income      map[uint]uint
	expense     map[uint]uint
	instruments map[uint]uint
	attachments map[uint]uint
}

func v1ToBasicTx(tx TransactionV1, m importMaps, attID *uint) (accounting.Transaction, bool) {
	switch tx.Type {
	case txTypeIncome:
		if tx.AccountID == 0 {
			return nil, false
		}
		return accounting.Income{
			Description: tx.Description, Notes: tx.Notes, Amount: tx.Amount,
			AccountID: m.accounts[tx.AccountID], CategoryID: m.income[tx.CategoryID],
			Date: tx.Date, AttachmentID: attID,
		}, true
	case txTypeExpense:
		if tx.AccountID == 0 {
			return nil, false
		}
		return accounting.Expense{
			Description: tx.Description, Notes: tx.Notes, Amount: tx.Amount,
			AccountID: m.accounts[tx.AccountID], CategoryID: m.expense[tx.CategoryID],
			Date: tx.Date, AttachmentID: attID,
		}, true
	case txTypeTransfer:
		if tx.OriginAccountID == 0 || tx.TargetAccountID == 0 {
			return nil, false
		}
		return accounting.Transfer{
			Description: tx.Description, Notes: tx.Notes,
			OriginAmount: tx.OriginAmount, TargetAmount: tx.TargetAmount,
			OriginAccountID: m.accounts[tx.OriginAccountID], TargetAccountID: m.accounts[tx.TargetAccountID],
			Date: tx.Date, AttachmentID: attID,
		}, true
	case txTypeStockBuy:
		return accounting.StockBuy{
			Description: tx.Description, Notes: tx.Notes, Date: tx.Date,
			InvestmentAccountID: m.accounts[tx.InvestmentAccountID], CashAccountID: m.accounts[tx.CashAccountID],
			InstrumentID: m.instruments[tx.InstrumentID], Quantity: tx.Quantity,
			TotalAmount: tx.TotalAmount, StockAmount: tx.StockAmount, AttachmentID: attID,
		}, true
	case txTypeStockSell:
		return accounting.StockSell{
			Description: tx.Description, Notes: tx.Notes, Date: tx.Date,
			InvestmentAccountID: m.accounts[tx.InvestmentAccountID], CashAccountID: m.accounts[tx.CashAccountID],
			InstrumentID: m.instruments[tx.InstrumentID], Quantity: tx.Quantity,
			PricePerShare: tx.PricePerShare, TotalAmount: tx.TotalAmount, Fees: tx.Fees, AttachmentID: attID,
		}, true
	case txTypeStockGrant:
		return accounting.StockGrant{
			Description: tx.Description, Notes: tx.Notes, Date: tx.Date,
			AccountID: m.accounts[tx.AccountID], InstrumentID: m.instruments[tx.InstrumentID],
			Quantity: tx.Quantity, FairMarketValue: tx.FairMarketValue, AttachmentID: attID,
		}, true
	case txTypeStockTransfer:
		return accounting.StockTransfer{
			Description: tx.Description, Notes: tx.Notes, Date: tx.Date,
			SourceAccountID: m.accounts[tx.SourceAccountID], TargetAccountID: m.accounts[tx.TargetAccountID],
			InstrumentID: m.instruments[tx.InstrumentID], Quantity: tx.Quantity, AttachmentID: attID,
		}, true
	case txTypeBalanceStatus:
		return accounting.BalanceStatus{
			Description: tx.Description, Notes: tx.Notes, Date: tx.Date,
			Amount: tx.Amount, AccountID: m.accounts[tx.AccountID], AttachmentID: attID,
		}, true
	case txTypeRevaluation:
		return accounting.Revaluation{
			Description: tx.Description, Notes: tx.Notes, Date: tx.Date,
			Amount: tx.Amount, Balance: tx.Balance, AccountID: m.accounts[tx.AccountID], AttachmentID: attID,
		}, true
	default:
		return nil, false
	}
}

func v1ToLotTx(ctx context.Context, store *accounting.Store, tx TransactionV1, m importMaps, attID *uint) (accounting.Transaction, error) {
	switch tx.Type {
	case txTypeStockVest:
		sourceAccID := m.accounts[tx.SourceAccountID]
		instID := m.instruments[tx.InstrumentID]
		lotSels, err := fifoLotSelections(ctx, store, sourceAccID, instID, tx.Quantity, tx.Date)
		if err != nil {
			return nil, fmt.Errorf("failed to build lot selections for vest %q: %w", tx.Description, err)
		}
		return accounting.StockVest{
			Description: tx.Description, Notes: tx.Notes, Date: tx.Date,
			SourceAccountID: sourceAccID, TargetAccountID: m.accounts[tx.TargetAccountID],
			InstrumentID: instID, VestingPrice: tx.VestingPrice,
			CategoryID: m.income[tx.CategoryID], LotSelections: lotSels, AttachmentID: attID,
		}, nil
	case txTypeStockForfeit:
		accID := m.accounts[tx.AccountID]
		instID := m.instruments[tx.InstrumentID]
		lotSels, err := fifoLotSelections(ctx, store, accID, instID, tx.Quantity, tx.Date)
		if err != nil {
			return nil, fmt.Errorf("failed to build lot selections for forfeit %q: %w", tx.Description, err)
		}
		return accounting.StockForfeit{
			Description: tx.Description, Notes: tx.Notes, Date: tx.Date,
			AccountID: accID, InstrumentID: instID, LotSelections: lotSels, AttachmentID: attID,
		}, nil
	default:
		return nil, nil
	}
}

func importTransactions(ctx context.Context, store *accounting.Store, r *zip.ReadCloser, accountsMap, incomeMap, expenseMap, instrumentsMap, attachmentsMap map[uint]uint) error {
	txs, err := loadV1Json[[]TransactionV1](r, transactionsFile)
	if err != nil {
		return err
	}

	// Sort transactions by date ASC (then by original ID) so that buys/grants
	// create lots before sells/transfers try to consume them.
	sort.Slice(txs, func(i, j int) bool {
		if txs[i].Date.Equal(txs[j].Date) {
			return txs[i].Id < txs[j].Id
		}
		return txs[i].Date.Before(txs[j].Date)
	})

	m := importMaps{accounts: accountsMap, income: incomeMap, expense: expenseMap, instruments: instrumentsMap, attachments: attachmentsMap}

	for _, tx := range txs {
		var remappedAttID *uint
		if tx.AttachmentID != nil {
			if newID, ok := attachmentsMap[*tx.AttachmentID]; ok {
				remappedAttID = &newID
			}
		}

		var item accounting.Transaction
		switch tx.Type {
		case txTypeStockVest, txTypeStockForfeit:
			item, err = v1ToLotTx(ctx, store, tx, m, remappedAttID)
			if err != nil {
				return err
			}
		default:
			var ok bool
			item, ok = v1ToBasicTx(tx, m, remappedAttID)
			if !ok {
				continue
			}
		}

		if item == nil {
			continue
		}
		newTxID, err := store.CreateTransaction(ctx, item)
		if err != nil {
			return fmt.Errorf("failed to create transaction: %w", err)
		}
		if remappedAttID != nil {
			if err := store.SetAttachmentID(ctx, newTxID, remappedAttID); err != nil {
				return fmt.Errorf("failed to set attachment ID on transaction %d: %w", newTxID, err)
			}
		}
	}
	return nil
}

// fifoLotSelections builds lot selections by picking open lots in FIFO order
// until the requested quantity is fulfilled. Only lots opened on or before txDate are considered.
func fifoLotSelections(ctx context.Context, store *accounting.Store, accountID, instrumentID uint, quantity float64, txDate time.Time) ([]accounting.LotSelection, error) {
	lots, err := store.ListLots(ctx, accounting.ListLotsOpts{AccountID: accountID, InstrumentID: instrumentID, BeforeDate: &txDate})
	if err != nil {
		return nil, err
	}
	var sels []accounting.LotSelection
	remaining := quantity
	for _, lot := range lots {
		if remaining <= 0 {
			break
		}
		if lot.Quantity <= 0 {
			continue
		}
		take := lot.Quantity
		if take > remaining {
			take = remaining
		}
		sels = append(sels, accounting.LotSelection{LotID: lot.Id, Quantity: take})
		remaining -= take
	}
	if remaining > 0 {
		return nil, fmt.Errorf("not enough lots: need %.2f more shares", remaining)
	}
	return sels, nil
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

func importAttachments(ctx context.Context, fileStore *filestore.Store, r *zip.ReadCloser) (map[uint]uint, error) {
	attachmentsMap := map[uint]uint{}
	if fileStore == nil {
		return attachmentsMap, nil
	}

	manifest, err := loadAttachmentManifest(r)
	if err != nil {
		return attachmentsMap, nil // gracefully skip (old backup without attachments)
	}

	for _, att := range manifest {
		content, err := readZipBinary(r, att.ZipPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read attachment %s from zip: %w", att.ZipPath, err)
		}

		date := time.Now()
		newID, err := fileStore.SaveRaw(ctx, date, content, att.OriginalName, att.MimeType)
		if err != nil {
			return nil, fmt.Errorf("failed to save attachment %d: %w", att.ID, err)
		}
		attachmentsMap[att.ID] = newID
	}

	return attachmentsMap, nil
}

func loadAttachmentManifest(r *zip.ReadCloser) ([]attachmentV1, error) {
	for _, f := range r.File {
		if f.Name != attachmentsFile {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer func() { _ = rc.Close() }()

		data, err := io.ReadAll(rc)
		if err != nil {
			return nil, err
		}

		var manifest []attachmentV1
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, err
		}
		return manifest, nil
	}
	return nil, fmt.Errorf("attachments manifest not found")
}

func readZipBinary(r *zip.ReadCloser, path string) ([]byte, error) {
	for _, f := range r.File {
		if f.Name != path {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer func() { _ = rc.Close() }()
		return io.ReadAll(rc)
	}
	return nil, fmt.Errorf("file %s not found in zip", path)
}
