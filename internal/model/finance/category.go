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
	ErrWrongCategoryType           = errors.New("wrong category type")
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

type CategoryType int8

const (
	IncomeCategory CategoryType = iota
	ExpenseCategory
)

type CategoryData struct {
	Name        string
	Description string
	Type        CategoryType
}

func (store *Store) CreateCategory(ctx context.Context, cat CategoryData, parent uint, tenant string) (uint, error) {
	var err error

	switch cat.Type {
	case IncomeCategory:
		payload := incomeCategory{
			Node:        closuretree.Node{},
			Name:        cat.Name,
			Description: cat.Description,
		}
		err = store.incomeCategoryTree.Add(ctx, &payload, parent, tenant)
		if err != nil {
			return 0, handleErr(err)
		}
		return payload.Id(), nil
	case ExpenseCategory:
		payload := expenseCategory{
			Node:        closuretree.Node{},
			Name:        cat.Name,
			Description: cat.Description,
		}
		err = store.expenseCategoryTree.Add(ctx, &payload, parent, tenant)
		if err != nil {
			return 0, handleErr(err)
		}
		return payload.Id(), nil
	default:
		return 0, ErrWrongCategoryType
	}
}

func (store *Store) UpdateCategory(ctx context.Context, Id uint, cat CategoryData, tenant string) error {
	var err error
	switch cat.Type {
	case IncomeCategory:
		payload := incomeCategory{
			Node:        closuretree.Node{},
			Name:        cat.Name,
			Description: cat.Description,
		}
		err = store.incomeCategoryTree.Update(ctx, Id, &payload, tenant)
	case ExpenseCategory:
		payload := expenseCategory{
			Node:        closuretree.Node{},
			Name:        cat.Name,
			Description: cat.Description,
		}
		err = store.expenseCategoryTree.Update(ctx, Id, &payload, tenant)
	default:
		return ErrWrongCategoryType
	}
	return handleErr(err)
}

func (store *Store) MoveCategory(ctx context.Context, Id, newParentID uint, catType CategoryType, tenant string) error {
	switch catType {
	case IncomeCategory:
		err := store.incomeCategoryTree.Move(ctx, Id, newParentID, tenant)
		return handleErr(err)
	case ExpenseCategory:
		err := store.expenseCategoryTree.Move(ctx, Id, newParentID, tenant)
		return handleErr(err)
	default:
		return ErrWrongCategoryType
	}
}

func (store *Store) DeleteCategoryRecursive(ctx context.Context, Id uint, catType CategoryType, tenant string) error {
	switch catType {
	case IncomeCategory:
		err := store.incomeCategoryTree.DeleteRecurse(ctx, Id, tenant)
		return handleErr(err)
	case ExpenseCategory:
		err := store.expenseCategoryTree.DeleteRecurse(ctx, Id, tenant)
		return handleErr(err)
	default:
		return ErrWrongCategoryType
	}
}

type Category struct {
	CategoryData
	Id       uint
	ParentId uint
}

func (store *Store) ListDescendantCategories(ctx context.Context, parent uint, depth int, catType CategoryType, tenant string) ([]Category, error) {

	switch catType {
	case IncomeCategory:
		data := &[]incomeCategory{}
		err := store.incomeCategoryTree.Descendants(ctx, parent, depth, tenant, data)
		if err != nil {
			return nil, handleErr(err)
		}
		var items []Category
		for _, item := range *data {
			add := Category{
				Id:       item.Id(),
				ParentId: item.Parent(),
				CategoryData: CategoryData{
					Name:        item.Name,
					Description: item.Description,
					Type:        IncomeCategory,
				},
			}
			items = append(items, add)
		}
		return items, nil
	case ExpenseCategory:
		data := &[]expenseCategory{}
		err := store.expenseCategoryTree.Descendants(ctx, parent, depth, tenant, data)
		if err != nil {
			return nil, handleErr(err)
		}
		var items []Category
		for _, item := range *data {
			add := Category{
				Id:       item.Id(),
				ParentId: item.Parent(),
				CategoryData: CategoryData{
					Name:        item.Name,
					Description: item.Description,
					Type:        ExpenseCategory,
				},
			}
			items = append(items, add)
		}
		return items, nil
	default:
		return nil, ErrWrongCategoryType
	}

}

// incomeCategory holds the needed information of a tag with a tree structure
type incomeCategory struct {
	closuretree.Node
	Name        string
	Description string
	Children    []*incomeCategory `gorm:"-"`
}

// expenseCategory holds the needed information of a tag with a tree structure
type expenseCategory struct {
	closuretree.Node
	Name        string
	Description string
	Children    []*expenseCategory `gorm:"-"`
}
