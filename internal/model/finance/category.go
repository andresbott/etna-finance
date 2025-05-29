package finance

import (
	"context"
	"errors"
	closuretree "github.com/go-bumbu/closure-tree"
)

// IncomeCategory holds the needed information of a tag with a tree structure
type IncomeCategory struct {
	closuretree.Node
	Name     string
	Children []*IncomeCategory `gorm:"-"`
}

// Define error variables to match with model errors
var (
	ErrCategoryNotFound            = errors.New("category not found")
	ErrCategoryConstraintViolation = errors.New("category constraint violation")
)

func handleErr(err error) error {
	if err != nil {
		if errors.Is(err, closuretree.ErrNodeNotFound) {
			return ErrCategoryNotFound
		}
		return err
	}
	return nil
}

func (store *Store) CreateIncomeCategory(ctx context.Context, cat *IncomeCategory, parent uint, tenant string) error {
	err := store.incomeTree.Add(ctx, cat, parent, tenant)
	return handleErr(err)
}
func (store *Store) MoveIncomeCategory(ctx context.Context, Id, newParentID uint, tenant string) error {
	err := store.incomeTree.Move(ctx, Id, newParentID, tenant)
	return handleErr(err)
}

func (store *Store) UpdateIncomeCategory(ctx context.Context, Id uint, payload IncomeCategory, tenant string) error {
	err := store.incomeTree.Update(ctx, Id, payload, tenant)
	return handleErr(err)
}

func (store *Store) DeleteRecurseIncomeCategory(ctx context.Context, Id uint, tenant string) error {
	err := store.incomeTree.DeleteRecurse(ctx, Id, tenant)
	return handleErr(err)
}

func (store *Store) DescendantsIncomeCategory(ctx context.Context, parent uint, depth int, tenant string, items *[]IncomeCategory) error {
	err := store.incomeTree.Descendants(ctx, parent, depth, tenant, items)
	return handleErr(err)
}

// ExpenseCategory holds the needed information of a tag with a tree structure
type ExpenseCategory struct {
	closuretree.Node
	Name     string
	Children []*ExpenseCategory `gorm:"-"`
}

func (store *Store) CreateExpenseCategory(ctx context.Context, cat *ExpenseCategory, parent uint, tenant string) error {
	err := store.expenseTree.Add(ctx, cat, parent, tenant)
	return handleErr(err)
}
func (store *Store) MoveExpenseCategory(ctx context.Context, Id, newParentID uint, tenant string) error {
	err := store.expenseTree.Move(ctx, Id, newParentID, tenant)
	return handleErr(err)
}

func (store *Store) UpdateExpenseCategory(ctx context.Context, Id uint, payload ExpenseCategory, tenant string) error {
	err := store.expenseTree.Update(ctx, Id, payload, tenant)
	return handleErr(err)
}

func (store *Store) DeleteRecurseExpenseCategory(ctx context.Context, Id uint, tenant string) error {
	err := store.expenseTree.DeleteRecurse(ctx, Id, tenant)
	return handleErr(err)
}

func (store *Store) DescendantsExpenseCategory(ctx context.Context, parent uint, depth int, tenant string, items *[]ExpenseCategory) error {
	err := store.expenseTree.Descendants(ctx, parent, depth, tenant, items)
	return handleErr(err)
}
