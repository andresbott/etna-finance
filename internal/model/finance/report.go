package finance

import (
	"context"
	"fmt"
	closuretree "github.com/go-bumbu/closure-tree"
	"time"
)

type ReportEntry struct {
	Id       uint
	ParentId uint
}

func (store *Store) GetReport(ctx context.Context, startDate, endDate time.Time, tenant string) ([]ReportEntry, error) {
	//entries, err := store.ListEntries(ctx, ListOpts{
	//	StartDate:   startDate,
	//	EndDate:     endDate,
	//	AccountIds:  nil,
	//	CategoryIds: nil,
	//	Limit:       0,
	//	Page:        0,
	//	Tenant:      tenant,
	//})
	//if err != nil {
	//	return nil, fmt.Errorf("unable to generate report %s", err.Error())
	//}
	//_ = entries
	////spew.Dump(entries)

	reportExpenseList, err := store.getIncomeDescendants(ctx, tenant)
	if err != nil {
		return nil, err
	}

	for _, item := range reportExpenseList {
		fmt.Println(item.Name)
		fmt.Printf("%d => %v \n", item.Node.NodeId, item.childrenIds)
	}

	// for each category find the entries that belongs to that category + all children

	// Select entry from entries where catgory in [list] AND rest of conditions

	return nil, nil
}

// incomeDescendants is a wrapper around Income category that holds it's own id and all descendant ids
type incomeDescendants struct {
	IncomeCategory
	childrenIds []uint
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
		ids := []uint{}
		for _, d := range descendants {
			ids = append(ids, d.Node.NodeId)
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

// expenseDescendants is a wrapper around Expense category that holds it's own id and all descendant ids
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
