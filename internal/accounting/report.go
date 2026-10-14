package accounting

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"golang.org/x/text/currency"
)

type CategoryReport struct {
	Income   []CategoryReportItem
	Expenses []CategoryReportItem
}
type CategoryReportItem struct {
	Id          uint
	ParentId    uint
	Name        string
	Description string
	Icon        string
	Values      map[currency.Unit]CategoryReportValues
}

type CategoryReportValues struct {
	Value float64
	Count uint
}

// TODO there should be another report that converts transactions to the target currency, so that instead of
// having one report per currency we only have one with the maine currency, this currently depends on having a way
// to capture currency conversion values over time + calculating the conversion at the time of the transaciton no at the
// time of generating the report

// ReportInOutByCategory generates a tree report of all incomes and expenses by grouped categories during the selected time frame
func (store *Store) ReportInOutByCategory(ctx context.Context, startDate, endDate time.Time) (CategoryReport, error) {
	report := CategoryReport{}
	incomeReport, err := store.getCategoryReport(ctx, startDate, endDate, IncomeCategory)
	if err != nil {
		return report, err
	}
	report.Income = incomeReport

	expenseReport, err := store.getCategoryReport(ctx, startDate, endDate, ExpenseCategory)
	if err != nil {
		return report, err
	}
	report.Expenses = expenseReport
	return report, nil
}

// getCategoryReport generates a flat list of report entries, where every entry is one category + the associated report value
func (store *Store) getCategoryReport(ctx context.Context, startDate, endDate time.Time, catType CategoryType) ([]CategoryReportItem, error) {
	categories, err := store.getCategoryChildren(ctx, catType)
	if err != nil {
		return nil, err
	}
	entryTypes := mustCategory2EntryTypes(catType)
	reportItems := make([]CategoryReportItem, len(categories)+1)

	accounts, err := store.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	currencyAccounts := getAccountIdsCurrencyMap(accounts)

	// iterate over all categories
	i := 0
	for _, item := range categories {
		var values = map[currency.Unit]CategoryReportValues{}
		for curr, accountIds := range currencyAccounts {
			opts := sumEntriesOpts{
				startDate:   startDate,
				endDate:     endDate,
				categoryIds: item.childrenIds,
				accountIds:  accountIds,
				entryTypes:  entryTypes,
			}
			sum, err := store.sumEntries(ctx, opts)
			if err != nil {
				if !errors.Is(err, ErrEntryNotFound) {
					return nil, err
				}
			}
			values[curr] = CategoryReportValues{
				Value: math.Abs(sum.Sum),
				Count: sum.Count,
			}

		}

		reportItems[i] = CategoryReportItem{
			Id:          item.Id,
			ParentId:    item.ParentId,
			Name:        item.Name,
			Description: item.Description,
			Icon:        item.Icon,
			Values:      values,
		}
		i++
	}

	// find entries without category (assigned to category 0)
	var values = map[currency.Unit]CategoryReportValues{}
	for curr, accountIds := range currencyAccounts {
		opts := sumEntriesOpts{
			startDate:   startDate,
			endDate:     endDate,
			categoryIds: []uint{0},
			accountIds:  accountIds,
			entryTypes:  entryTypes,
		}
		sum, err := store.sumEntries(ctx, opts)
		if err != nil {
			if !errors.Is(err, ErrEntryNotFound) {
				return nil, err
			}
		}
		values[curr] = CategoryReportValues{
			Value: math.Abs(sum.Sum),
			Count: sum.Count,
		}
	}

	reportItems[i] = CategoryReportItem{
		Id:          0,
		ParentId:    0,
		Name:        "unclassified",
		Description: "entries without any category",
		Values:      values,
	}

	return reportItems, nil
}

// mustCategory2EntryTypes is an internal function to convert category types into entry types
// note that it will panic on wrong category type
func mustCategory2EntryTypes(catType CategoryType) []entryType {
	switch catType {
	case IncomeCategory:
		return []entryType{incomeEntry, stockVestIncomeEntry}
	case ExpenseCategory:
		return []entryType{expenseEntry}
	default:
		panic("wrong category type")
	}
}

// getAccountIdsCurrencyMap takes a list of accounts and organizes them per currency
func getAccountIdsCurrencyMap(in []Account) map[currency.Unit][]uint {
	accountsByCurrency := make(map[currency.Unit][]uint)
	for _, got := range in {
		account := got
		accountsByCurrency[account.Currency] = append(accountsByCurrency[account.Currency], account.ID)
	}
	return accountsByCurrency
}

// AccountBalance is the balance of an account at a point in time, in the account's own currency and in the
// main currency.
type AccountBalance struct {
	Date time.Time
	// NativeBalance is the balance in the account's own currency.
	NativeBalance float64
	// MainCurrencyBalance is the balance converted to the main currency at the FX rate of Date. It equals
	// NativeBalance when the account uses the main currency or no main currency is configured.
	MainCurrencyBalance float64
	// HasUnconvertedAmounts is true when an FX rate (or, for investment accounts, a price) was missing, so
	// some amounts are included unconverted (or, without a price, left out).
	HasUnconvertedAmounts bool
	// EntryCount is the number of entries summed for this step; the first step counts every entry up to Date.
	// Investment accounts are valued from positions and leave it at 0.
	EntryCount uint
}

// entry types used when calling sum entries for balance purposes
// balanceEntryTypes lists entry types that affect cash balance.
// Position entries (stockBuyEntry, stockSellEntry) are no longer in db_entries — they are tracked via db_trades.
var balanceEntryTypes = []entryType{incomeEntry, expenseEntry, transferInEntry, transferOutEntry, stockCashOutEntry, stockCashInEntry, revaluationEntry}

// AccountBalanceSingle get the balance of a single account on a given point in time
func (store *Store) AccountBalanceSingle(ctx context.Context, accountID uint, endDate time.Time) (AccountBalance, error) {

	// NOTE: it is intentional to call AccountBalance with modified props, this is done to align the behavior
	// of a caller using the same props on AccountBalance instead of AccountBalanceSingle.
	// in this sense, AccountBalanceSingle is just a convenience wrapper
	got, err := store.AccountBalance(ctx, accountID, 0, time.Time{}, endDate)
	if err != nil {
		return AccountBalance{}, err
	}
	return got[0], nil
}

// AccountBalance returns a slice of AccountBalance, with N steps of balance spread between the start and the end date
// this allows to generate balance graphs.
// If zero or 1 steps are selected, a slice with size 1 is returned that contains the balance at end date.
// For cash-like accounts, the implementation sums delta entries for every step.
// For investment/restricted-stock accounts, it reconstructs position market value at each step.
// Every step carries the balance in the account's own currency and in the main currency; the main-currency
// value uses the FX rate at the step's end date. If no FX rate is available for a step, the unconverted value
// is used and AccountBalance.HasUnconvertedAmounts is set to true.
func (store *Store) AccountBalance(ctx context.Context, accountID uint, steps int, startDate, endDate time.Time) ([]AccountBalance, error) {

	if endDate.Before(startDate) {
		return nil, fmt.Errorf("end date must be after start date")
	}
	account, err := store.GetAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	accountCurrency := account.Currency.String()

	// Investment and restricted-stock accounts: use market-value-based calculation
	if isInvestmentType(account.Type) {
		return store.investmentBalanceSteps(ctx, accountID, accountCurrency, steps, startDate, endDate)
	}

	// Cash-like accounts: accumulate balance in original currency, convert total at each step's FX rate
	return store.cashBalanceSteps(ctx, accountID, accountCurrency, steps, startDate, endDate)
}

// investmentBalanceSteps calculates market-value-based balance for investment accounts
// at each step date, using the same step-splitting logic as the cash-flow methods.
func (store *Store) investmentBalanceSteps(ctx context.Context, accountID uint, accountCurrency string, steps int, startDate, endDate time.Time) ([]AccountBalance, error) {
	dates := store.computeStepDates(steps, startDate, endDate)

	results := make([]AccountBalance, len(dates))
	for i, date := range dates {
		value, err := store.investmentValueAtDate(ctx, accountID, accountCurrency, date)
		if err != nil {
			return nil, err
		}
		value.Date = toDate(date)
		results[i] = value
	}
	return results, nil
}

// computeStepDates returns the list of step end-dates for a balance report, matching
// the same logic used by the cash-flow methods.
func (store *Store) computeStepDates(steps int, startDate, endDate time.Time) []time.Time {
	if steps <= 1 {
		return []time.Time{endDate}
	}

	if startDate.IsZero() {
		// One step per day, ending at endDate
		start := toDate(endDate).AddDate(0, 0, -1*(steps-1))
		stepDuration := time.Hour * 24
		dates := make([]time.Time, steps)
		for i := 0; i < steps; i++ {
			dateFrom := start.Add(time.Duration(i) * stepDuration)
			dateTo := dateFrom.Add(stepDuration).Add(-time.Nanosecond)
			if i == steps-1 {
				dateTo = endOfDay(endDate)
			}
			dates[i] = dateTo
		}
		return dates
	}

	// Equal steps between startDate and endDate
	startDate = toDate(startDate)
	dates := []time.Time{endOfDay(startDate)}

	advancedStart := startDate.Add(24 * time.Hour)
	if !endDate.After(advancedStart) {
		return dates
	}

	daysInRange := int(endDate.Sub(advancedStart).Hours() / 24)
	if daysInRange <= 0 {
		daysInRange = 1
	}
	stepCount := steps - 1
	if stepCount > daysInRange {
		stepCount = daysInRange
	}
	stepDuration := endDate.Sub(advancedStart) / time.Duration(stepCount)

	for i := 0; i < stepCount; i++ {
		dateFrom := advancedStart.Add(time.Duration(i) * stepDuration)
		dateTo := dateFrom.Add(stepDuration).Add(-time.Nanosecond)
		if i == stepCount-1 {
			dateTo = endOfDay(endDate)
		}
		dates = append(dates, dateTo)
	}
	return dates
}

// convertAmount converts an amount from one currency to another using the FX rate at time t. Rates are
// stored against the main currency with the convention "main/X" (e.g. "CHF/USD" = how many USD per 1 CHF),
// so an amount in X converts to the main currency as amount / rate, and a conversion between two
// non-main currencies goes through the main currency.
// Returns the original amount and unconverted=false when there is nothing to convert: same currency,
// or no market store or main currency configured. Returns the original amount and unconverted=true when
// a needed rate is missing or zero.
func (store *Store) convertAmount(ctx context.Context, amount float64, from, to string, t time.Time) (float64, bool) {
	if from == to || store.marketStore == nil || store.mainCurrency == "" {
		return amount, false
	}
	// The market store (timeseries) rejects non-UTC times. t may be in a non-UTC zone
	// (e.g. the report handler's default time.Now()); .UTC() preserves the instant while
	// satisfying that contract.
	t = t.UTC()

	inMain := amount
	if from != store.mainCurrency {
		rate, ok := store.mainCurrencyRate(ctx, from, t)
		if !ok {
			return amount, true
		}
		inMain = amount / rate
	}
	if to == store.mainCurrency {
		return inMain, false
	}
	rate, ok := store.mainCurrencyRate(ctx, to, t)
	if !ok {
		return amount, true
	}
	return inMain * rate, false
}

// mainCurrencyRate returns the "main/secondary" FX rate at time t, or false when it is missing or zero.
func (store *Store) mainCurrencyRate(ctx context.Context, secondary string, t time.Time) (float64, bool) {
	rec, err := store.marketStore.RateAt(ctx, store.mainCurrency, secondary, t)
	if err != nil || rec == nil || rec.Rate == 0 {
		return 0, false
	}
	return rec.Rate, true
}

// cashBalanceSteps calculates cash-flow-based balance for cash-like accounts at each step date.
// At each step it accumulates the raw balance in the account's original currency, then converts
// the entire balance to main currency using the FX rate at that step's date, and returns both. This
// correctly reflects FX movements on the full balance, not just on each delta.
func (store *Store) cashBalanceSteps(ctx context.Context, accountID uint, accountCurrency string, steps int, startDate, endDate time.Time) ([]AccountBalance, error) {
	dates := store.computeStepDates(steps, startDate, endDate)

	var rawBalance float64 // cumulative balance in original currency
	var results []AccountBalance
	prevDateTo := time.Time{}

	for i, dateTo := range dates {
		dateFrom := prevDateTo
		if i == 0 {
			dateFrom = time.Time{} // first step: sum all entries from beginning of time
		}

		opts := sumEntriesOpts{
			startDate:  dateFrom,
			endDate:    dateTo,
			accountIds: []uint{accountID},
			entryTypes: balanceEntryTypes,
		}

		sum, err := store.sumBalanceEntries(ctx, opts)
		if err != nil {
			return nil, err
		}

		rawBalance += sum.Sum // accumulate in original currency
		// Convert the entire balance at this step's FX rate
		converted, unconverted := store.convertAmount(ctx, rawBalance, accountCurrency, store.mainCurrency, dateTo)

		results = append(results, AccountBalance{
			Date:                  toDate(dateTo),
			NativeBalance:         rawBalance,
			MainCurrencyBalance:   converted,
			HasUnconvertedAmounts: unconverted,
			EntryCount:            sum.Count,
		})

		prevDateTo = dateTo
	}

	return results, nil
}

// investmentValueAtDate calculates the total market value of an investment/restricted-stock
// account at a given date. It reconstructs positions from lots and disposals, then multiplies
// by the instrument price at that date, converting to the account currency and to the main currency.
// The returned AccountBalance has no Date set. HasUnconvertedAmounts is true if any position could
// not be fully converted (missing price or FX rate).
func (store *Store) investmentValueAtDate(ctx context.Context, accountID uint, accountCurrency string, date time.Time) (AccountBalance, error) {
	if store.marketStore == nil {
		return AccountBalance{HasUnconvertedAmounts: true}, nil
	}

	// Get all lots for this account opened on or before the date
	beforeDate := endOfDay(date)
	lots, err := store.ListLots(ctx, ListLotsOpts{
		AccountID:  accountID,
		BeforeDate: &beforeDate,
	})
	if err != nil {
		return AccountBalance{}, fmt.Errorf("failed to list lots: %w", err)
	}

	// For each lot, calculate quantity held at the given date by subtracting disposals
	// that happened on or before that date.
	type instrumentPosition struct {
		quantity float64
	}
	positions := make(map[uint]*instrumentPosition) // instrumentID -> position

	for _, lot := range lots {
		// Skip lots that were fully closed before the target date
		if lot.ClosedDate != nil && !lot.ClosedDate.After(date) {
			continue
		}

		// Get disposals for this lot up to the target date
		var disposedQty float64
		var disposals []dbLotDisposal
		if err := store.db.WithContext(ctx).
			Where("lot_id = ? AND date <= ?", lot.Id, endOfDay(date)).
			Find(&disposals).Error; err != nil {
			return AccountBalance{}, fmt.Errorf("failed to query disposals: %w", err)
		}
		for _, d := range disposals {
			disposedQty += d.Quantity
		}

		qtyAtDate := lot.OriginalQty - disposedQty
		if qtyAtDate <= 0 {
			continue
		}

		pos, ok := positions[lot.InstrumentID]
		if !ok {
			pos = &instrumentPosition{}
			positions[lot.InstrumentID] = pos
		}
		pos.quantity += qtyAtDate
	}

	// Calculate market value for each instrument position
	var total AccountBalance

	for instrumentID, pos := range positions {
		inst, err := store.marketStore.GetInstrument(ctx, instrumentID)
		if err != nil {
			total.HasUnconvertedAmounts = true
			continue
		}

		// .UTC() satisfies the market store's UTC-only contract without shifting the instant
		// (date may be in a non-UTC zone from the handler's default time.Now()).
		priceRec, err := store.marketStore.PriceAt(ctx, inst.Symbol, endOfDay(date).UTC())
		if err != nil || priceRec == nil {
			total.HasUnconvertedAmounts = true
			continue
		}

		value := pos.quantity * priceRec.Close
		instCurrency := inst.Currency.String()

		native, nativeUnconverted := store.convertAmount(ctx, value, instCurrency, accountCurrency, date)
		converted, unconverted := store.convertAmount(ctx, value, instCurrency, store.mainCurrency, date)
		if nativeUnconverted || unconverted {
			total.HasUnconvertedAmounts = true
		}
		total.NativeBalance += native
		total.MainCurrencyBalance += converted
	}

	return total, nil
}

// isInvestmentType returns true for account types that hold positions valued at market price.
func isInvestmentType(t AccountType) bool {
	return t == InvestmentAccountType || t == RestrictedStockAccountType
}

// endOfDay returns a time.Time on the last nanosecond of the day for the provided input time
func endOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location()).Add(-time.Nanosecond)
}

// toDate returns the first second of the day for the provided input time
func toDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
