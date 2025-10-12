package accounting

import (
	"context"
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
	CategoryID    uint `gorm:"index"`          // Foreign key, only populated for income and expense

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
	Count uint
}

type sumEntriesOpts struct {
	startDate   time.Time
	endDate     time.Time
	categoryIds []uint
	accountIds  []uint
	entryTypes  []entryType
	tenant      string
}

// sumEntries is an internal function to sum the values of entries filtering by Categories entry types etc
// Important: this function needs to stay internal as it mixes accounts with different currencies
// the library needs to take extra care to handle the situations clearly
func (store *Store) sumEntries(ctx context.Context, opts sumEntriesOpts) (sumResult, error) {
	db := store.db.WithContext(ctx).Table("db_entries")

	//db = db.Select("db_entries.*, db_transactions.date").
	db = db.Select("ABS(SUM(amount)) as sum, COUNT(*) as count").
		Joins("JOIN db_transactions ON db_transactions.id = db_entries.transaction_id")

	// ensure proper owner
	db = db.Where("db_entries.owner_id = ? AND db_transactions.owner_id = ? ", opts.tenant, opts.tenant)
	// Filter by date range
	db = db.Where("db_transactions.date BETWEEN ? AND ?", opts.startDate, opts.endDate)

	// filter by accountId
	if opts.accountIds != nil {
		db = db.Where("db_entries.account_id IN (?)", opts.accountIds)
	}

	// select the entry type
	if len(opts.entryTypes) == 0 {
		return sumResult{Sum: 0, Count: 0}, fmt.Errorf("entry type must be set")
	}
	db = db.Where("db_entries.entry_type IN (?) ", opts.entryTypes)

	// filter by cat type
	if len(opts.categoryIds) > 0 {
		db = db.Where("db_entries.category_id IN (?)", opts.categoryIds)
	}

	//target := []map[string]any{} // left for debugging
	var target sumResult

	q := db.Scan(&target)
	if q.Error != nil {
		return sumResult{}, q.Error
	}
	//spew.Dump(target)
	return target, nil
}
