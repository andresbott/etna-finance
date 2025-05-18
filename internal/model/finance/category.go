package finance

import (
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

func (store *Store) CreateIncomeCategory(cat *IncomeCategory, parent uint, tenant string) error {
	err := store.incomeTree.Add(cat, parent, tenant)
	return handleErr(err)
}
func (store *Store) MoveIncomeCategory(Id, newParentID uint, tenant string) error {
	err := store.incomeTree.Move(Id, newParentID, tenant)
	return handleErr(err)
}

func (store *Store) UpdateIncomeCategory(Id uint, payload IncomeCategory, tenant string) error {
	err := store.incomeTree.Update(Id, payload, tenant)
	return handleErr(err)
}

func (store *Store) DeleteRecurseIncomeCategory(Id uint, tenant string) error {
	err := store.incomeTree.DeleteRecurse(Id, tenant)
	return handleErr(err)
}

// ExpenseCategory holds the needed information of a tag with a tree structure
type ExpenseCategory struct {
	closuretree.Node
	Name     string
	Children []*ExpenseCategory `gorm:"-"`
}

func (store *Store) CreateExpenseCategory(cat *ExpenseCategory, parent uint, tenant string) error {
	err := store.expenseTree.Add(cat, parent, tenant)
	return handleErr(err)
}
func (store *Store) MoveExpenseCategory(Id, newParentID uint, tenant string) error {
	err := store.expenseTree.Move(Id, newParentID, tenant)
	return handleErr(err)
}

func (store *Store) UpdateExpenseCategory(Id uint, payload ExpenseCategory, tenant string) error {
	err := store.expenseTree.Update(Id, payload, tenant)
	return handleErr(err)
}

func (store *Store) DeleteRecurseExpenseCategory(Id uint, tenant string) error {
	err := store.expenseTree.DeleteRecurse(Id, tenant)
	return handleErr(err)
}
