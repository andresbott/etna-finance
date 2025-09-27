package finance

import (
	"context"
	"errors"
	"fmt"
	closuretree "github.com/go-bumbu/closure-tree"
	"time"
)

type Report struct {
	Income   []ReportItem
	Expenses []ReportItem
}

type ReportItem struct {
	NodeId      uint
	ParentId    uint
	Name        string
	Description string
	Value       float64
}

// GetReport generates a tree report of all incomes and expenses by grouped categories during the selected time frame
func (store *Store) GetReport(ctx context.Context, startDate, endDate time.Time, tenant string) (*Report, error) {
	report := Report{}
	incomeReport, err := store.getReport(ctx, startDate, endDate, tenant, IncomeEntry)
	if err != nil {
		return nil, err
	}
	report.Income = incomeReport

	expenseReport, err := store.getReport(ctx, startDate, endDate, tenant, ExpenseEntry)
	if err != nil {
		return nil, err
	}
	report.Expenses = expenseReport

	return &report, nil
}

func (store *Store) getReport(ctx context.Context, startDate, endDate time.Time, tenant string, entryType EntryType) ([]ReportItem, error) {
	reportExpenseList, err := store.getIncomeDescendants(ctx, tenant)
	if err != nil {
		return nil, err
	}

	incomeReport := make([]ReportItem, len(reportExpenseList))
	i := 0
	for _, item := range reportExpenseList {
		var sum float64
		var err error

		sum, err = store.SumEntries(ctx,
			SumOpts{StartDate: startDate, EndDate: endDate, CategoryIds: item.childrenIds, EntryType: entryType, Tenant: tenant})
		if err != nil {
			if !errors.Is(err, ErrEntryNotFound) {
				return nil, err
			}
		}

		incomeReport[i] = ReportItem{
			NodeId:      item.NodeId,
			ParentId:    item.ParentId,
			Name:        item.Name,
			Description: item.Description,
			Value:       sum,
		}
	}
	return incomeReport, nil
}

// incomeDescendants is a wrapper around Income category that holds it's own id and all descendant ids
type incomeDescendants struct {
	IncomeCategory
	childrenIds []int
}

func (store *Store) getIncomeDescendants(ctx context.Context, tenant string) ([]incomeDescendants, error) {
	var got []IncomeCategory
	err := store.DescendantsIncomeCategory(ctx, 0, -1, tenant, &got)
	if err != nil {
		return nil, fmt.Errorf("unable to get income descendants %s", err.Error())
	}

	// build a new flat list where all the children are added
	reportIncomeList := make([]incomeDescendants, len(got))
	lookup := buildIncomeTreeWithDescendants(got)

	var i int
	for _, node := range lookup {
		descendants := collectIncomeDescendants(node)
		ids := []int{}
		for _, d := range descendants {
			ids = append(ids, int(d.Node.NodeId))
		}
		reportIncomeList[i] = incomeDescendants{
			IncomeCategory: IncomeCategory{
				Node: closuretree.Node{
					NodeId:   node.NodeId,
					ParentId: node.ParentId,
					Tenant:   node.Tenant,
				},
				Name:        node.Name,
				Description: node.Description,
			},
			childrenIds: ids,
		}
		i++
	}
	return reportIncomeList, nil
}

// builds a map of all items with all it's children referenced as pointers
func buildIncomeTreeWithDescendants(nodes []IncomeCategory) map[uint]*IncomeCategory {
	// create lookup map
	lookup := make(map[uint]*IncomeCategory)
	for i := range nodes {
		lookup[nodes[i].Node.NodeId] = &nodes[i]
	}

	// link children
	for i := range nodes {
		node := &nodes[i]
		if node.Node.ParentId != 0 {
			parent := lookup[node.Node.ParentId]
			parent.Children = append(parent.Children, node)
		}
	}

	return lookup
}

// Collect all descendants including self
func collectIncomeDescendants(node *IncomeCategory) []*IncomeCategory {
	result := []*IncomeCategory{node}
	for _, child := range node.Children {
		result = append(result, collectIncomeDescendants(child)...)
	}
	return result
}

// expenseDescendants is a wrapper around Expenses category that holds it's own id and all descendant ids
type expenseDescendants struct {
	ExpenseCategory
	childrenIds []uint
}

func (store *Store) getExpenseDescendants(ctx context.Context, tenant string) ([]expenseDescendants, error) {
	var got []ExpenseCategory
	err := store.DescendantsExpenseCategory(ctx, 0, -1, tenant, &got)
	if err != nil {
		return nil, fmt.Errorf("unable to get expense descendants %s", err.Error())
	}

	// build a new flat list where all the children are added
	reportExpenseList := make([]expenseDescendants, len(got))
	lookup := buildExpenseTreeWithDescendants(got)

	var i int
	for _, node := range lookup {
		descendants := collectExpenseDescendants(node)
		ids := []uint{}
		for _, d := range descendants {
			ids = append(ids, d.Node.NodeId)
		}
		reportExpenseList[i] = expenseDescendants{
			ExpenseCategory: ExpenseCategory{
				Node: closuretree.Node{
					NodeId:   node.NodeId,
					ParentId: node.ParentId,
					Tenant:   node.Tenant,
				},
				Name:        node.Name,
				Description: node.Description,
			},
			childrenIds: ids,
		}
		i++
	}
	return reportExpenseList, nil
}

// builds a map of all items with all it's children referenced as pointers
func buildExpenseTreeWithDescendants(nodes []ExpenseCategory) map[uint]*ExpenseCategory {
	// create lookup map
	lookup := make(map[uint]*ExpenseCategory)
	for i := range nodes {
		lookup[nodes[i].Node.NodeId] = &nodes[i]
	}

	// link children
	for i := range nodes {
		node := &nodes[i]
		if node.Node.ParentId != 0 {
			parent := lookup[node.Node.ParentId]
			parent.Children = append(parent.Children, node)
		}
	}

	return lookup
}

// Collect all descendants including self
func collectExpenseDescendants(node *ExpenseCategory) []*ExpenseCategory {
	result := []*ExpenseCategory{node}
	for _, child := range node.Children {
		result = append(result, collectExpenseDescendants(child)...)
	}
	return result
}
