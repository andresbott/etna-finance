package accounting

import (
	"context"
	"errors"
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

// ReportOnCategories generates a tree report of all incomes and expenses by grouped categories during the selected time frame
func (store *Store) ReportOnCategories(ctx context.Context, startDate, endDate time.Time, tenant string) (CategoryReport, error) {
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
				entryType:   entrytype,
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
			entryType:   entrytype,
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
