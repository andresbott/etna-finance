package accounting

import (
	"context"
	"fmt"
	"time"
)

type entryType int

const (
	unknownentryType entryType = iota
	incomeEntry
	expenseEntry
	transferInEntry
	transferOutEntry
	stockBuyEntry
	stockSellEntry
	stockCashOutEntry // cash leaving account on stock buy
	stockCashInEntry  // cash entering account on stock sell
	stockGrantEntry   // position increase without cash (vest, gift, grant, etc.)
	stockTransferOutEntry
	stockTransferInEntry
	balanceStatusEntry
	stockVestIncomeEntry
	revaluationEntry
)

type dbEntry struct {
	Id            uint `gorm:"primarykey"`
	TransactionID uint `gorm:"not null;index"`           // Foreign key
	AccountID     uint `gorm:"not null;index"`           // Foreign key
	CategoryID    uint `gorm:"index"`                    // Foreign key, only populated for income and expense
	InstrumentID  uint `gorm:"column:security_id;index"` // Foreign key, only populated for stock buy/sell entries

	Amount   float64 `gorm:"not null"` // Amount in account currency; for stock position entries (buy/sell) is 0; for stock cash entries signed (out negative, in positive)
	Quantity float64 // for stock position entries: shares; unused for stock cash entries
	Balance  float64 // informative: for revaluation entries, the target balance the user entered

	EntryType entryType

	CreatedAt time.Time
	UpdatedAt time.Time
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
}

// sumEntries is an internal function to sum the values of entries filtering by Categories entry types etc
// Important: this function needs to stay internal as it mixes accounts with different currencies
// the library needs to take extra care to handle the situations clearly
func (store *Store) sumEntries(ctx context.Context, opts sumEntriesOpts) (sumResult, error) {
	db := store.db.WithContext(ctx).Table("db_entries")

	//db = db.Select("db_entries.*, db_transactions.date").
	db = db.Select("SUM(amount) as sum, COUNT(*) as count").
		Joins("JOIN db_transactions ON db_transactions.id = db_entries.transaction_id")

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

// sumBalanceEntries sums entries for cash-balance purposes. It works like
// sumEntries but excludes income/expense entries from stock-sell transactions.
// Those entries record realized gain/loss and fees for P&L reporting but must
// not affect cash-balance sums because the actual cash flow is already captured
// by the stockCashInEntry.
func (store *Store) sumBalanceEntries(ctx context.Context, opts sumEntriesOpts) (sumResult, error) {
	db := store.db.WithContext(ctx).Table("db_entries")

	db = db.Select("SUM(amount) as sum, COUNT(*) as count").
		Joins("JOIN db_transactions ON db_transactions.id = db_entries.transaction_id")

	db = db.Where("db_transactions.date BETWEEN ? AND ?", opts.startDate, opts.endDate)

	if opts.accountIds != nil {
		db = db.Where("db_entries.account_id IN (?)", opts.accountIds)
	}

	if len(opts.entryTypes) == 0 {
		return sumResult{Sum: 0, Count: 0}, fmt.Errorf("entry type must be set")
	}
	db = db.Where("db_entries.entry_type IN (?)", opts.entryTypes)

	db = db.Where(
		"NOT (db_entries.entry_type IN (?) AND db_transactions.type = ?)",
		[]entryType{incomeEntry, expenseEntry}, StockSellTransaction,
	)

	var target sumResult
	if err := db.Scan(&target).Error; err != nil {
		return sumResult{}, err
	}
	return target, nil
}
