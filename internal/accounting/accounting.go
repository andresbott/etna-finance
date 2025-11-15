package accounting

import (
	"context"
	"fmt"
	closuretree "github.com/go-bumbu/closure-tree"
	"gorm.io/gorm"
)

type Store struct {
	db           *gorm.DB
	categoryTree *closuretree.Tree
}

func NewStore(db *gorm.DB) (*Store, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}

	b := Store{
		db: db,
	}

	stmt := &gorm.Statement{DB: db}
	err := stmt.Parse(&dbAccount{})
	if err != nil {
		return nil, fmt.Errorf("error parsing schema: %w", err)
	}

	err = db.AutoMigrate(&dbAccountProvider{}, &dbAccount{}, &dbTransaction{}, &dbEntry{})
	if err != nil {
		return nil, err
	}

	categoryTree, err := closuretree.New(db, dbCategory{}) // init the closure tree, this includes gorm automigrate
	if err != nil {
		return nil, err
	}
	b.categoryTree = categoryTree

	return &b, nil
}

func NewValidationErr(in string) ErrValidation {
	return ErrValidation(in)
}

type ErrValidation string

func (v ErrValidation) Error() string {
	return string(v)
}

func (store *Store) ListTenants(ctx context.Context) ([]string, error) {
	db := store.db.WithContext(ctx).Table("db_account_providers")

	// Get distinct owner IDs
	var tenants []string
	if err := db.
		Select("DISTINCT(owner_id)").
		Order("owner_id ASC").
		Pluck("owner_id", &tenants).Error; err != nil {
		return nil, err
	}

	return tenants, nil
}

func (store *Store) WipeData(ctx context.Context) error {
	tables := []string{
		"db_account_providers",
		"db_accounts",
		"db_transactions",
		"db_entries",
		"db_categories",
	}

	for _, table := range tables {
		db := store.db.WithContext(ctx).Table(table)
		if err := db.Where("1 = 1").Delete(nil).Error; err != nil {
			return fmt.Errorf("failed to delete data in table '%s' : %w", table, err)
		}
	}

	return nil
}
