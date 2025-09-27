package finance

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Report struct {
	Income   []ReportItem
	Expenses []ReportItem
}

// GetReport generates a tree report of all incomes and expenses by grouped categories during the selected time frame
func (store *Store) GetReport(ctx context.Context, startDate, endDate time.Time, tenant string) (Report, error) {
	report := Report{}
	incomeReport, err := store.getReport(ctx, startDate, endDate, IncomeCategory, tenant)
	if err != nil {
		return report, err
	}
	report.Income = incomeReport

	expenseReport, err := store.getReport(ctx, startDate, endDate, ExpenseCategory, tenant)
	if err != nil {
		return report, err
	}
	report.Expenses = expenseReport
	return report, nil
}

type ReportItem struct {
	Id          uint
	ParentId    uint
	Name        string
	Description string
	Value       float64
}

// getReport generates a flat list of report entries, where every entry is one category + the associated report value
func (store *Store) getReport(ctx context.Context, startDate, endDate time.Time, catType CategoryType, tenant string) ([]ReportItem, error) {
	categories, err := store.getCategoryIds(ctx, catType, tenant)
	if err != nil {
		return nil, err
	}

	var entrytype EntryType
	if catType == IncomeCategory {
		entrytype = IncomeEntry
	} else if catType == ExpenseCategory {
		entrytype = ExpenseEntry
	} else {
		return nil, ErrWrongCategoryType
	}

	reportItems := make([]ReportItem, len(categories))
	i := 0
	for _, item := range categories {
		var sum float64
		var err error

		sum, err = store.SumEntries(ctx,
			SumOpts{StartDate: startDate, EndDate: endDate, CategoryIds: item.childrenIds, EntryType: entrytype, Tenant: tenant})
		if err != nil {
			if !errors.Is(err, ErrEntryNotFound) {
				return nil, err
			}
		}
		reportItems[i] = ReportItem{
			Id:          item.Id,
			ParentId:    item.ParentId,
			Name:        item.Name,
			Description: item.Description,
			Value:       sum,
		}
		i++
	}
	return reportItems, nil
}

type categoryIds struct {
	Category
	childrenIds []int
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
		ids := []int{}
		for _, d := range descendants {
			ids = append(ids, int(d.Id))
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
