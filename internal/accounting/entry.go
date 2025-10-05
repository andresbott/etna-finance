package accounting

import (
	"context"
	"database/sql"
	"fmt"
	"gorm.io/gorm"
	"time"
)

type entryType int

const (
	unknownentryType entryType = iota
	incomeEntry
	expenseEntry
	transferInEntry
	transferOutEntry
)

type dbEntry struct {
	Id            uint `gorm:"primarykey"`
	TransactionID uint `gorm:"not null;index"` // Foreign key
	AccountID     uint `gorm:"not null;index"` // Foreign key

	Amount   float64 `gorm:"not null"` // Amount in account currency
	Quantity float64 // -- for stock shares (nullable for cash-only entries)

	EntryType entryType //income, expense, transferIn transferOut, stockbuy, stock sell

	OwnerId   string `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type sumResult struct {
	Sum   float64
	Count int64
}

type sumByCategoryOpts struct {
	StartDate   time.Time
	EndDate     time.Time
	CategoryIds []uint
	entryType   entryType
	Tenant      string
}

// sumEntryByCategories is an internal function to sum the values of entries filtering by Categories
// only income and expenses permitted in the sum by categories; transfers and other operations are not added to a category
func (store *Store) sumEntryByCategories(ctx context.Context, opts sumByCategoryOpts) (sumResult, error) {
	db := store.db.WithContext(ctx).Where("owner_id = ?", opts.Tenant)
	// Filter by date range
	db = db.Where("date BETWEEN ? AND ?", opts.StartDate, opts.EndDate)

	// Fiter by type
	if opts.entryType != incomeEntry && opts.entryType != expenseEntry {
		return sumResult{Sum: 0, Count: 0}, fmt.Errorf("entry type not supported, must be income or expense: %d", opts.entryType)
	}
	db = db.Where("type IN ? ", opts.entryType)

	// Filter by category ID if provided
	if len(opts.CategoryIds) > 0 {
		db = db.Where("category_id IN ?", opts.CategoryIds)
	}

	type scan struct {
		Sum   sql.NullFloat64
		Count int64
	}

	var result scan
	err := db.Model(&dbEntry{}).
		Select("SUM(target_amount) as sum, COUNT(*) as count").
		Scan(&result).Error
	if err != nil {
		return sumResult{}, fmt.Errorf("unable to sum entries %w", err)
	}

	if !result.Sum.Valid {
		return sumResult{}, ErrEntryNotFound
	}
	if result.Count == 0 || !result.Sum.Valid {
		return sumResult{}, ErrEntryNotFound
	}

	return sumResult{
		Sum:   result.Sum.Float64,
		Count: result.Count,
	}, nil
}
