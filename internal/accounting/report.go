package accounting

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/text/currency"
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
				Value: sum.Sum,
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
			Value: sum.Sum,
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

// AccountBalance get the balance of a single account on a given point in time
func (store *Store) AccountBalance(ctx context.Context, accountID uint, endDate time.Time, tenant string) (AccountBalance, error) {
	end := endOfDay(endDate)
	// The first step is to calculate all historical values until the start date
	opts := sumEntriesOpts{
		startDate:  time.Time{},
		endDate:    end,
		accountIds: []uint{accountID},
		entryTypes: []entryType{incomeEntry, expenseEntry, transferInEntry, transferOutEntry},
		tenant:     tenant,
	}
	sum, err := store.sumEntries(ctx, opts)
	if err != nil && !errors.Is(err, ErrEntryNotFound) {
		return AccountBalance{}, err
	}
	return AccountBalance{
		Date:  time.Time{},
		Sum:   sum.Sum,
		Count: sum.Count,
	}, err
}

type AccountBalance struct {
	Date  time.Time
	Sum   float64
	Count uint
}

func (store *Store) AccountBalanceProgression(
	ctx context.Context,
	accountID uint,
	startDate, endDate time.Time,
	steps int,
	tenant string,
) ([]AccountBalance, error) {

	if endDate.Before(startDate) {
		return nil, fmt.Errorf("end date must be after start date")
	}
	if steps < 2 {
		return nil, fmt.Errorf("steps must be greater than or equal to 2")
	}

	// round the start date to the exact day start date minus one nanosecond
	startDate = endOfDay(startDate) // we want to start just at the end of the previous day
	// ensure all queries run with end of the day to include operations on the end date
	endDate = endOfDay(endDate)

	// get the balance on the beginning of the time
	historicalSum, err := store.AccountBalance(ctx, accountID, startDate, tenant)
	if err != nil {
		return nil, err
	}

	var prevSum float64
	prevSum = historicalSum.Sum
	prevCount := historicalSum.Count

	results := make([]AccountBalance, 0, steps)
	results = append(results, AccountBalance{
		Date:  toDate(startDate),
		Sum:   prevSum,
		Count: prevCount,
	})

	var stepCount int
	var stepDuration time.Duration

	// Total daysInRange in the range
	daysInRange := int(endDate.Sub(startDate).Hours() / 24)
	if daysInRange <= 0 {
		daysInRange = 1
	}

	stepCount = steps - 1
	if stepCount > daysInRange {
		stepCount = daysInRange
	}
	stepDuration = endDate.Sub(startDate) / time.Duration(stepCount)

	//--- Steps 1..N: evenly spaced intervals ---
	for i := 0; i < stepCount; i++ {
		from := startDate.Add(time.Duration(i) * stepDuration)
		to := from.Add(stepDuration)
		from = from.Add(time.Nanosecond)

		// last step ends exactly at endDate
		if i == stepCount-1 {
			to = endDate
		}

		opts := sumEntriesOpts{
			startDate:  from,
			endDate:    to,
			accountIds: []uint{accountID},
			entryTypes: []entryType{incomeEntry, expenseEntry, transferInEntry, transferOutEntry},
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
			Date:  toDate(to), // remove the microsecond from the report for simplicity
			Sum:   prevSum,
			Count: sum.Count,
		})
	}

	return results, nil
}

func endOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location()).Add(-time.Nanosecond)
}

func toDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
