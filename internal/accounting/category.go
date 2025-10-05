package accounting

import (
	"context"
	"errors"
	"fmt"
	closuretree "github.com/go-bumbu/closure-tree"
)

// Define error variables to match with model errors
var (
	ErrCategoryNotFound            = errors.New("category not found")
	ErrCategoryConstraintViolation = errors.New("category type constraint violation")
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
	UndefinedCategory CategoryType = iota
	IncomeCategory
	ExpenseCategory
)

// dbCategory holds the needed information of a tag with a tree structure
type dbCategory struct {
	closuretree.Node
	Name        string
	Description string
	Type        CategoryType
	Children    []*dbCategory `gorm:"-"`
}

type CategoryData struct {
	Name        string
	Description string
	Type        CategoryType
}

func (store *Store) CreateCategory(ctx context.Context, cat CategoryData, parent uint, tenant string) (uint, error) {
	var err error

	if cat.Type != IncomeCategory && cat.Type != ExpenseCategory {
		return 0, ErrWrongCategoryType
	}

	if parent != 0 {
		parentNode := dbCategory{}
		err = store.categoryTree.GetNode(ctx, parent, tenant, &parentNode)
		if err != nil {
			return 0, fmt.Errorf("unable to get parent category: %w", err)
		}
		if parentNode.Type != cat.Type {
			return 0, ErrCategoryConstraintViolation
		}
	}

	payload := dbCategory{
		Node:        closuretree.Node{},
		Name:        cat.Name,
		Description: cat.Description,
		Type:        cat.Type,
	}
	err = store.categoryTree.Add(ctx, &payload, parent, tenant)
	if err != nil {
		return 0, handleErr(err)
	}
	return payload.Id(), nil

}

func (store *Store) UpdateCategory(ctx context.Context, Id uint, cat CategoryData, tenant string) error {

	var err error

	if cat.Type != IncomeCategory && cat.Type != ExpenseCategory {
		return ErrWrongCategoryType
	}

	node := dbCategory{}
	err = store.categoryTree.GetNode(ctx, Id, tenant, &node)
	if err != nil {
		return fmt.Errorf("unable to get parent category: %w", err)
	}
	if node.Type != cat.Type {
		return ErrCategoryConstraintViolation
	}

	payload := dbCategory{
		Node:        closuretree.Node{},
		Name:        cat.Name,
		Description: cat.Description,
	}
	err = store.categoryTree.Update(ctx, Id, &payload, tenant)
	return handleErr(err)
}

func (store *Store) MoveCategory(ctx context.Context, Id, newParentID uint, tenant string) error {

	var err error

	node := dbCategory{}
	err = store.categoryTree.GetNode(ctx, Id, tenant, &node)
	if err != nil {
		return fmt.Errorf("unable to get parent category: %w", err)
	}

	newParent := dbCategory{}
	err = store.categoryTree.GetNode(ctx, newParentID, tenant, &newParent)
	if err != nil {
		return fmt.Errorf("unable to get parent category: %w", err)
	}

	if node.Type != newParent.Type {
		return ErrCategoryConstraintViolation
	}

	err = store.categoryTree.Move(ctx, Id, newParentID, tenant)
	return handleErr(err)

}

func (store *Store) DeleteCategoryRecursive(ctx context.Context, Id uint, tenant string) error {
	err := store.categoryTree.DeleteRecurse(ctx, Id, tenant)
	return handleErr(err)
}

type Category struct {
	CategoryData
	Id       uint
	ParentId uint
}

func (store *Store) ListDescendantCategories(ctx context.Context, parent uint, depth int, categoryType CategoryType, tenant string) ([]Category, error) {
	data := &[]dbCategory{}
	err := store.categoryTree.Descendants(ctx, parent, depth, tenant, data)
	if err != nil {
		return nil, handleErr(err)
	}
	var items []Category
	for _, item := range *data {
		if item.Type != categoryType {
			continue
		}
		add := Category{
			Id:       item.Id(),
			ParentId: item.Parent(),
			CategoryData: CategoryData{
				Name:        item.Name,
				Description: item.Description,
				Type:        item.Type,
			},
		}
		items = append(items, add)
	}
	return items, nil

}
