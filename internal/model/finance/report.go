package finance

import (
	"context"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"time"
)

type AccountReport struct {
	Account
	Value float64
}

func (store *Store) GetAccountReport(ctx context.Context, accountIds []uint, date time.Time, tenant string) ([]AccountReport, error) {

	if len(accountIds) == 0 {
		accounts, err := store.ListAccounts(ctx, tenant)
		if err != nil {
			return nil, fmt.Errorf("error getting accountIds: %w", err)
		}
		for _, accoutn := range accounts {
			accountIds = append(accountIds, accoutn.ID)
		}
	}

	sum, err := store.sumEntryByCategories(ctx,
		sumByCategoryOpts{EndDate: date, AccountIds: accountIds, Tenant: tenant})
	if err != nil {
		if !errors.Is(err, ErrEntryNotFound) {
			return nil, err
		}
	}

	spew.Dump(sum)

	//reportItems := make([]CategoryReportItem, len(categories)+1)

	return nil, nil

}

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
	categories, err := store.getCategoryIds(ctx, catType, tenant)
	if err != nil {
		return nil, err
	}

	var entrytype EntryType

	switch catType {
	case IncomeCategory:
		entrytype = IncomeEntry
	case ExpenseCategory:
		entrytype = ExpenseEntry
	default:
		return nil, ErrWrongCategoryType
	}

	reportItems := make([]CategoryReportItem, len(categories)+1)
	i := 0
	for _, item := range categories {
		var sum sumResult
		var err error

		sum, err = store.sumEntryByCategories(ctx,
			sumByCategoryOpts{StartDate: startDate, EndDate: endDate, CategoryIds: item.childrenIds, EntryType: []EntryType{entrytype}, Tenant: tenant})
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
		sumByCategoryOpts{StartDate: startDate, EndDate: endDate, CategoryIds: []uint{0}, EntryType: []EntryType{entrytype}, Tenant: tenant})
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

type categoryIds struct {
	Category
	childrenIds []uint
}

// getCategoryIds queries all categories of a type and tenant and returns a flat list of categories + the associated
// list of child ids for every single category
func (store *Store) getCategoryIds(ctx context.Context, catType CategoryType, tenant string) ([]categoryIds, error) {
	got, err := store.ListDescendantCategories(ctx, 0, -1, catType, tenant)
	if err != nil {
		return nil, fmt.Errorf("unable to get income descendants %s", err.Error())
	}

	// transform got categories into categories trees
	var catTree []categoryTree
	for _, item := range got {
		catTree = append(catTree, categoryTree{
			Category: item,
		})
	}
	lookup := buildTreeWithDescendants(catTree)

	// build a new flat list where all the children are added
	reportIncomeList := make([]categoryIds, len(got))

	var i int
	for _, node := range lookup {
		descendants := collectDescendants(node)
		ids := []uint{}
		for _, d := range descendants {
			ids = append(ids, d.Id)
		}
		reportIncomeList[i] = categoryIds{
			Category: Category{
				CategoryData: CategoryData{
					Name:        node.Name,
					Description: node.Description,
					Type:        node.Type,
				},
				Id:       node.Id,
				ParentId: node.ParentId,
			},
			childrenIds: ids,
		}
		i++
	}
	return reportIncomeList, nil
}

type categoryTree struct {
	Category
	children []*categoryTree
}

// builds a map of all items with all it's children referenced as pointers
func buildTreeWithDescendants(nodes []categoryTree) map[uint]*categoryTree {
	// create lookup map
	lookup := make(map[uint]*categoryTree)
	for i := range nodes {
		lookup[nodes[i].Id] = &nodes[i]
	}

	// link children
	for i := range nodes {
		node := &nodes[i]
		if node.ParentId != 0 {
			parent := lookup[node.ParentId]
			parent.children = append(parent.children, node)
		}
	}

	return lookup
}

// Collect all descendants including self
func collectDescendants(node *categoryTree) []*categoryTree {
	result := []*categoryTree{node}
	for _, child := range node.children {
		result = append(result, collectDescendants(child)...)
	}
	return result
}
