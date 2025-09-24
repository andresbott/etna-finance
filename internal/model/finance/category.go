package finance

import (
	"context"
	"errors"
	closuretree "github.com/go-bumbu/closure-tree"
)

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

// IncomeCategory holds the needed information of a tag with a tree structure
type IncomeCategory struct {
	closuretree.Node
	Name        string
	Description string
	Children    []*IncomeCategory `gorm:"-"`
}

func (store *Store) CreateIncomeCategory(ctx context.Context, cat *IncomeCategory, parent uint, tenant string) error {
	err := store.incomeCategoryTree.Add(ctx, cat, parent, tenant)
	return handleErr(err)
}
func (store *Store) MoveIncomeCategory(ctx context.Context, Id, newParentID uint, tenant string) error {
	err := store.incomeCategoryTree.Move(ctx, Id, newParentID, tenant)
	return handleErr(err)
}

func (store *Store) UpdateIncomeCategory(ctx context.Context, Id uint, payload IncomeCategory, tenant string) error {
	err := store.incomeCategoryTree.Update(ctx, Id, payload, tenant)
	return handleErr(err)
}

func (store *Store) DeleteRecurseIncomeCategory(ctx context.Context, Id uint, tenant string) error {
	err := store.incomeCategoryTree.DeleteRecurse(ctx, Id, tenant)
	return handleErr(err)
}

func (store *Store) DescendantsIncomeCategory(ctx context.Context, parent uint, depth int, tenant string, items *[]IncomeCategory) error {
	err := store.incomeCategoryTree.Descendants(ctx, parent, depth, tenant, items)
	return handleErr(err)
}

// ExpenseCategory holds the needed information of a tag with a tree structure
type ExpenseCategory struct {
	closuretree.Node
	Name        string
	Description string
	Children    []*ExpenseCategory `gorm:"-"`
}

func (store *Store) CreateExpenseCategory(ctx context.Context, cat *ExpenseCategory, parent uint, tenant string) error {
	err := store.expenseCategoryTree.Add(ctx, cat, parent, tenant)
	return handleErr(err)
}
func (store *Store) MoveExpenseCategory(ctx context.Context, Id, newParentID uint, tenant string) error {
	err := store.expenseCategoryTree.Move(ctx, Id, newParentID, tenant)
	return handleErr(err)
}

func (store *Store) UpdateExpenseCategory(ctx context.Context, Id uint, payload ExpenseCategory, tenant string) error {
	err := store.expenseCategoryTree.Update(ctx, Id, payload, tenant)
	return handleErr(err)
}

func (store *Store) DeleteRecurseExpenseCategory(ctx context.Context, Id uint, tenant string) error {
	err := store.expenseCategoryTree.DeleteRecurse(ctx, Id, tenant)
	return handleErr(err)
}

func (store *Store) DescendantsExpenseCategory(ctx context.Context, parent uint, depth int, tenant string, items *[]ExpenseCategory) error {
	err := store.expenseCategoryTree.Descendants(ctx, parent, depth, tenant, items)
	return handleErr(err)
}
