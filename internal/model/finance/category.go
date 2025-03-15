package finance

import closuretree "github.com/go-bumbu/closure-tree"

// Category holds the needed information of a tag with a tree structure
type Category struct {
	closuretree.Node
	Name     string
	Children []*Category `gorm:"-"`
}

func (store *Store) CreateCategory(cat *Category, parent uint, tenant string) error {
	err := store.tree.Add(cat, parent, tenant)
	if err != nil {
		return err
	}
	return nil
}
func (store *Store) Move(Id, newParentID uint, tenant string) error {
	err := store.tree.Move(Id, newParentID, tenant)
	if err != nil {
		return err
	}
	return nil
}

func (store *Store) Update(Id uint, payload Category, tenant string) error {
	err := store.tree.Update(Id, payload, tenant)
	if err != nil {
		return err
	}
	return nil
}

func (store *Store) DeleteRecurse(Id uint, tenant string) error {
	err := store.tree.DeleteRecurse(Id, tenant)
	if err != nil {
		return err
	}
	return nil
}
