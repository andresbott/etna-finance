package accounting

import (
	"context"
	"fmt"

	"github.com/andresbott/etna/internal/marketdata"
	closuretree "github.com/go-bumbu/closure-tree"
	"gorm.io/gorm"
)

type Store struct {
	db           *gorm.DB
	categoryTree *closuretree.Tree
	marketStore  *marketdata.Store
}

func NewStore(db *gorm.DB, marketStore *marketdata.Store) (*Store, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}

	b := Store{
		db:          db,
		marketStore: marketStore,
	}

	stmt := &gorm.Statement{DB: db}
	err := stmt.Parse(&dbAccount{})
	if err != nil {
		return nil, fmt.Errorf("error parsing schema: %w", err)
	}

	err = db.AutoMigrate(&dbAccountProvider{}, &dbAccount{}, &dbTransaction{}, &dbEntry{}, &dbTrade{}, &dbLot{}, &dbLotDisposal{}, &dbPosition{})
	if err != nil {
		return nil, err
	}

	// Migration: remove soft-deleted transactions and drop the deleted_at column.
	// Transactions previously used GORM soft-delete, but this caused ghost records
	// that broke backup export and category-rule reapply (entries were hard-deleted
	// while the transaction row lingered with deleted_at set).
	if db.Migrator().HasColumn(&dbTransaction{}, "deleted_at") {
		if err := db.Exec("DELETE FROM db_transactions WHERE deleted_at IS NOT NULL").Error; err != nil {
			return nil, fmt.Errorf("purge soft-deleted transactions: %w", err)
		}
		if err := db.Migrator().DropColumn(&dbTransaction{}, "deleted_at"); err != nil {
			return nil, fmt.Errorf("drop deleted_at column: %w", err)
		}
	}

	categoryTree, err := closuretree.New(db, dbCategory{}) // init the closure tree, this includes gorm automigrate
	if err != nil {
		return nil, err
	}
	b.categoryTree = categoryTree

	return &b, nil
}

// GetInstrument returns instrument info by id from the marketdata store.
// Returns marketdata.ErrInstrumentNotFound if no marketdata store is set or the instrument is missing.
func (s *Store) GetInstrument(ctx context.Context, id uint) (marketdata.Instrument, error) {
	if s.marketStore == nil {
		return marketdata.Instrument{}, marketdata.ErrInstrumentNotFound
	}
	return s.marketStore.GetInstrument(ctx, id)
}

func NewValidationErr(in string) ErrValidation {
	return ErrValidation(in)
}

type ErrValidation string

func (v ErrValidation) Error() string {
	return string(v)
}

func (store *Store) WipeData(ctx context.Context) error {
	tables := []string{
		"db_lot_disposals",
		"db_lots",
		"db_trades",
		"db_positions",
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
