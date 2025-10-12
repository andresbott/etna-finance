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

	// TODO refactor to use a struct with pointers for the update
	var err error
	if cat.Type != IncomeCategory && cat.Type != ExpenseCategory {
		return ErrWrongCategoryType
	}

	node := dbCategory{}
	err = store.categoryTree.GetNode(ctx, Id, tenant, &node)
	if err != nil {
		if errors.Is(err, closuretree.ErrNodeNotFound) {
			return fmt.Errorf("unable to get parent category: %w", ErrCategoryNotFound)
		}
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
		if errors.Is(err, closuretree.ErrNodeNotFound) {
			return fmt.Errorf("unable to get parent category: %w", ErrCategoryNotFound)
		}
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

func (store *Store) GetCategory(ctx context.Context, Id uint, tenant string) (Category, error) {
	var err error
	node := dbCategory{}
	err = store.categoryTree.GetNode(ctx, Id, tenant, &node)
	if err != nil {
		return Category{}, fmt.Errorf("unable to get category: %w", err)
	}
	category := Category{
		CategoryData: CategoryData{
			Name:        node.Name,
			Description: node.Description,
			Type:        node.Type,
		},
		Id:       node.NodeId,
		ParentId: node.ParentId,
	}
	return category, nil
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

// ListDescendantCategories returns a flat list of all child categories of a given parent and a specific category type
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

type categoryIds struct {
	Category
	childrenIds []uint
}

type categoryTree struct {
	Category
	children []*categoryTree
}

// getCategoryChildren queries all categories of a type and tenant and returns a flat list of ALL categories + the associated
// list of child ids for every single category.
// eg, for a tree like
// 0 - root
// 1 -  |- income (type income)
// 2 -  |    |-  salary
// 3 -  |- expense (type expense)
// 4 -  |    |-  general
// 5 -  |    |     | - groceries
// 6 -  |    |     | - restaurant
//
// the return for expense would be
// {<Expense Category detail>, children: 3,4,5,6} , {<general Category detail>, children: 4,5,6},
// {<groceries Category detail>, children: 5}, {<restaurant Category detail>, children: 6},
//
// This is a support function for generating a full income/expense report per category
func (store *Store) getCategoryChildren(ctx context.Context, catType CategoryType, tenant string) ([]categoryIds, error) {
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

// Collect all descendants including self
func collectDescendants(node *categoryTree) []*categoryTree {
	result := []*categoryTree{node}
	for _, child := range node.children {
		result = append(result, collectDescendants(child)...)
	}
	return result
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
