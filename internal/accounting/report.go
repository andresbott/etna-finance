package accounting

import (
	"context"
	"errors"
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
	Value       float64
	Count       int64
}

// GetCategoryReport generates a tree report of all incomes and expenses by grouped categories during the selected time frame
func (store *Store) GetCategoryReport(ctx context.Context, startDate, endDate time.Time, tenant string) (CategoryReport, error) {
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

	var entrytype entryType

	switch catType {
	case IncomeCategory:
		entrytype = incomeEntry
	case ExpenseCategory:
		entrytype = expenseEntry
	default:
		return nil, ErrWrongCategoryType
	}

	reportItems := make([]CategoryReportItem, len(categories)+1)
	i := 0
	for _, item := range categories {
		var sum sumResult
		var err error

		sum, err = store.sumEntryByCategories(ctx,
			sumByCategoryOpts{startDate: startDate, endDate: endDate, categoryIds: item.childrenIds, entryType: entrytype, tenant: tenant})
		if err != nil {
			if !errors.Is(err, ErrEntryNotFound) {
				return nil, err
			}
		}
		reportItems[i] = CategoryReportItem{
			Id:          item.Id,
			ParentId:    item.ParentId,
			Name:        item.Name,
			Description: item.Description,
			Value:       sum.Sum,
			Count:       sum.Count,
		}
		i++
	}

	// find entries with not category (assigned to category 0)
	sum, err := store.sumEntryByCategories(ctx,
		sumByCategoryOpts{startDate: startDate, endDate: endDate, categoryIds: []uint{0}, entryType: entrytype, tenant: tenant})
	if err != nil {
		if !errors.Is(err, ErrEntryNotFound) {
			return nil, err
		}
	}
	reportItems[i] = CategoryReportItem{
		Id:          0,
		ParentId:    0,
		Name:        "unclassified",
		Description: "entries without any category",
		Value:       sum.Sum,
		Count:       sum.Count,
	}

	return reportItems, nil
}

type categoryTree struct {
	Category
	children []*categoryTree
}
