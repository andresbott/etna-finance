package accounting

import (
	"context"
	"errors"
	"fmt"

	closuretree "github.com/go-bumbu/closure-tree"
	"golang.org/x/text/currency"
	"gorm.io/gorm"
)

// InstrumentInfo is the minimal instrument data needed for transaction validation (e.g. currency match).
type InstrumentInfo struct {
	ID       uint
	Currency currency.Unit
}

var ErrInstrumentNotFound = errors.New("instrument not found")

// InstrumentGetter provides instrument lookup for transaction validation.
// Implementations typically adapt an external store (e.g. marketdata).
type InstrumentGetter interface {
	GetInstrument(ctx context.Context, id uint, tenant string) (InstrumentInfo, error)
}

type Store struct {
	db               *gorm.DB
	categoryTree     *closuretree.Tree
	instrumentGetter InstrumentGetter
}

func NewStore(db *gorm.DB, instrumentGetter InstrumentGetter) (*Store, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}

	b := Store{
		db:               db,
		instrumentGetter: instrumentGetter,
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

// GetInstrument returns instrument info by id and tenant, delegating to the injected InstrumentGetter.
// If no getter is set, returns ErrInstrumentNotFound.
func (s *Store) GetInstrument(ctx context.Context, id uint, tenant string) (InstrumentInfo, error) {
	if s.instrumentGetter == nil {
		return InstrumentInfo{}, ErrInstrumentNotFound
	}
	return s.instrumentGetter.GetInstrument(ctx, id, tenant)
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

	// getTask distinct owner IDs
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
