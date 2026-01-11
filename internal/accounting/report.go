package accounting

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/text/currency"
	"math"
	"time"
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
func (store *Store) ReportInOutByCategory(ctx context.Context, startDate, endDate time.Time, tenant string) (CategoryReport, error) {
	report := CategoryReport{}
	incomeReport, err := store.getCategoryReport(ctx, startDate, endDate, IncomeCategory, tenant)
	if err != nil {
		return report, err
	}
	report.Income = incomeReport

	expenseReport, err := store.getCategoryReport(ctx, startDate, endDate, ExpenseCategory, tenant)
	if err != nil {
		return report, err
	}
	report.Expenses = expenseReport
	return report, nil
}

// getCategoryReport generates a flat list of report entries, where every entry is one category + the associated report value
func (store *Store) getCategoryReport(ctx context.Context, startDate, endDate time.Time, catType CategoryType, tenant string) ([]CategoryReportItem, error) {
	categories, err := store.getCategoryChildren(ctx, catType, tenant)
	if err != nil {
		return nil, err
	}
	entrytype := mustCategory2Entry(catType)
	reportItems := make([]CategoryReportItem, len(categories)+1)

	accounts, err := store.ListAccounts(ctx, tenant)
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
				entryTypes:  []entryType{entrytype},
				tenant:      tenant,
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
			entryTypes:  []entryType{entrytype},
			tenant:      tenant,
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

// mustCategory2Entry is an internal functino to convert category tipes into entry types
// note that it will panic on wong category type
func mustCategory2Entry(catType CategoryType) entryType {
	var entrytype entryType
	switch catType {
	case IncomeCategory:
		entrytype = incomeEntry
	case ExpenseCategory:
		entrytype = expenseEntry
	default:
		panic("wrong category type")
	}
	return entrytype
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

type AccountBalance struct {
	Date  time.Time
	Sum   float64
	Count uint
}

// entry types used when calling sum entries for balance purposes
var balanceEntryTypes = []entryType{incomeEntry, expenseEntry, transferInEntry, transferOutEntry}

// AccountBalanceSingle get the balance of a single account on a given point in time
func (store *Store) AccountBalanceSingle(ctx context.Context, accountID uint, endDate time.Time, tenant string) (AccountBalance, error) {

	// NOTE: it is intentional to call AccountBalance with modified props, this is done to align the behavior
	// of a caller using the same props on AccountBalance instead of AccountBalanceSingle.
	// in this sense, AccountBalanceSingle is just a convenience wrapper
	got, err := store.AccountBalance(ctx, accountID, 0, time.Time{}, endDate, tenant)
	if err != nil {
		return AccountBalance{}, err
	}
	return got[0], nil
}

// AccountBalance returns a slice of AccountBalance, with N steps of balance spread between the start and the end date
// this allows to generate balance graphs.
// If zero or 1 steps are selected, a slice with size 1 is returned that contains the balance at end date.
// The implementation tries to be efficient to only sum the delta entries for every step.
func (store *Store) AccountBalance(ctx context.Context, accountID uint, steps int, startDate, endDate time.Time, tenant string) ([]AccountBalance, error) {

	if endDate.Before(startDate) {
		return nil, fmt.Errorf("end date must be after start date")
	}
	// check the account exists
	_, err := store.GetAccount(ctx, accountID, tenant)
	if err != nil {
		return nil, err
	}

	if steps <= 1 {
		// handle single result requests
		return store.accountBalanceSingle(ctx, accountID, endDate, tenant)
	} else {
		if startDate.IsZero() {
			return store.accountBalanceMultipleEmptyStartDate(ctx, accountID, steps, endDate, tenant)
		} else {
			return store.accountBalanceMultipleEqualSteps(ctx, accountID, steps, startDate, endDate, tenant)
		}
	}
}

// accountBalanceSingle internal function used to get a single account balance,
func (store *Store) accountBalanceSingle(ctx context.Context, accountID uint, endDate time.Time, tenant string) ([]AccountBalance, error) {

	opts := sumEntriesOpts{
		startDate:  time.Time{},
		endDate:    endDate,
		accountIds: []uint{accountID},
		entryTypes: balanceEntryTypes,
		tenant:     tenant,
	}
	sum, err := store.sumEntries(ctx, opts)
	if err != nil && !errors.Is(err, ErrEntryNotFound) {
		return nil, err
	}
	b := AccountBalance{
		Date:  toDate(endDate),
		Sum:   sum.Sum,
		Count: sum.Count,
	}
	return []AccountBalance{b}, nil
}

// accountBalanceMultipleEmptyStartDate internal function used to get multiple account balances when no start date is
// provided, this case the historical data is one day per step
func (store *Store) accountBalanceMultipleEmptyStartDate(ctx context.Context, accountID uint, steps int, endDate time.Time, tenant string) ([]AccountBalance, error) {

	var prevSum float64
	var results []AccountBalance

	// calculate start date one based on the number of steps, one per day
	startDate := toDate(endDate).AddDate(0, 0, -1*(steps-1))
	stepDuration := time.Hour * 24

	for i := 0; i < steps; i++ {
		dateFrom := startDate.Add(time.Duration(i) * stepDuration)
		dateTo := dateFrom.Add(stepDuration).Add(-time.Nanosecond)

		// if first iteration calculate all historical data
		if i == 0 {
			dateFrom = time.Time{}
		}

		//last step ends exactly at endDate
		if i == steps-1 {
			dateTo = endOfDay(endDate)
		}

		opts := sumEntriesOpts{
			startDate:  dateFrom,
			endDate:    dateTo,
			accountIds: []uint{accountID},
			entryTypes: balanceEntryTypes,
			tenant:     tenant,
		}

		sum, err := store.sumEntries(ctx, opts)
		if err != nil {
			if errors.Is(err, ErrEntryNotFound) {
				continue
			}
			return nil, err
		}
		prevSum += sum.Sum
		results = append(results, AccountBalance{
			Date:  toDate(dateTo), // remove the microsecond from the report for simplicity
			Sum:   prevSum,
			Count: sum.Count,
		})
	}

	return results, nil
}

// accountBalanceMultipleEqualSteps internal function used to get multiple account balances where N amaunt of
// steps are returned equialy spread over the time range
func (store *Store) accountBalanceMultipleEqualSteps(ctx context.Context, accountID uint, steps int, startDate, endDate time.Time, tenant string) ([]AccountBalance, error) {
	var prevSum float64

	var results []AccountBalance
	// get the balance at the start time
	startDate = toDate(startDate)
	opts := sumEntriesOpts{
		startDate:  time.Time{},
		endDate:    endOfDay(startDate), // sum everything up to 1 nanosecond before the start date
		accountIds: []uint{accountID},
		entryTypes: balanceEntryTypes,
		tenant:     tenant,
	}

	sum, err := store.sumEntries(ctx, opts)
	if err != nil && !errors.Is(err, ErrEntryNotFound) {
		return nil, err
	}
	prevSum = sum.Sum
	results = append(results, AccountBalance{
		Date:  startDate,
		Sum:   prevSum,
		Count: sum.Count,
	})

	// modify the start date before starting the iteration
	startDate = startDate.Add(24 * time.Hour)

	// Total daysInRange in the range
	daysInRange := int(endDate.Sub(startDate).Hours() / 24)
	if daysInRange <= 0 {
		daysInRange = 1
	}

	// remove the initial step from the counter
	stepCount := steps - 1
	// limit the steps up to a max of one per day
	if stepCount > daysInRange {
		stepCount = daysInRange
	}
	stepDuration := endDate.Sub(startDate) / time.Duration(stepCount)

	//--- Steps 1...N: evenly spaced intervals ---
	for i := 0; i < stepCount; i++ {

		dateFrom := startDate.Add(time.Duration(i) * stepDuration) // for i = 0 dateFrom = start date
		dateTo := dateFrom.Add(stepDuration).Add(-time.Nanosecond)
		//dateFrom = dateFrom.addTask(time.Nanosecond) // avoid complete overlap

		//last step ends exactly at endDate
		if i == stepCount-1 {
			dateTo = endOfDay(endDate)
		}

		opts = sumEntriesOpts{
			startDate:  dateFrom,
			endDate:    dateTo,
			accountIds: []uint{accountID},
			entryTypes: balanceEntryTypes,
			tenant:     tenant,
		}

		sum, err = store.sumEntries(ctx, opts)
		if err != nil {
			if errors.Is(err, ErrEntryNotFound) {
				continue
			}
			return nil, err
		}
		prevSum += sum.Sum
		results = append(results, AccountBalance{
			Date:  toDate(dateTo), // remove the microsecond from the report for simplicity
			Sum:   prevSum,
			Count: sum.Count,
		})
	}

	return results, nil
}

// endOfDay returns a time.Time on the last nanosecond of the day for the provided input time
func endOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location()).Add(-time.Nanosecond)
}

// toDate returns the first second of the day for the provided input time
func toDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
