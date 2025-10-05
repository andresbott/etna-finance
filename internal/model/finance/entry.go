package finance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"golang.org/x/text/currency"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Entry is a
type Entry struct {
	Id          uint
	Description string
	Date        time.Time
	Locked      bool      // does not accept changes anymore
	Type        EntryType //income, transfer, expense, stock buy, stock sell ( like transfer with stock amounts added)

	StockAmount float64 // used to track the amount of stocks in the account

	// target is the account that gets the operation type, e.g. income or expense
	TargetAmount          float64
	TargetAccountID       uint
	TargetAccountName     string        // used only for printing out, TODO the client should not depend on this
	TargetAccountCurrency currency.Unit // used only for printing out, TODO the client should not depend on this

	// origin is only mandatory for transfer operations where we move from one account to another
	OriginAmount          float64
	OriginAccountID       uint
	OriginAccountName     string        // used only for printing out, TODO the client should not depend on this
	OriginAccountCurrency currency.Unit // used only for printing out, TODO the client should not depend on this

	// category is used to classify the operation
	CategoryId uint
}

// dbAccount is the DB internal representation of a Bookmark
type dbEntry struct {
	Id          uint `gorm:"primarykey"`
	Description string
	Date        time.Time `gorm:"index"`
	Locked      bool
	Type        int8

	OwnerId   string `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	TargetAmount    float64
	OriginAmount    float64
	StockAmount     float64
	TargetAccountId uint `gorm:"index"`
	OriginAccountId uint `gorm:"index"`

	CategoryId uint `gorm:"index"`
}

// dbToAccount is used internally to transform the db struct to public facing struct
func getEntry(in dbEntry) Entry {
	return Entry{
		Id:          in.Id,
		Description: in.Description,
		Date:        in.Date,
		Locked:      in.Locked,
		Type:        EntryType(in.Type),

		StockAmount: in.StockAmount,

		TargetAmount:    in.TargetAmount,
		TargetAccountID: in.TargetAccountId,

		OriginAmount:    in.OriginAmount,
		OriginAccountID: in.OriginAccountId,

		CategoryId: in.CategoryId,
	}
}

type EntryType int8

func (t EntryType) String() string {
	switch t {
	case IncomeEntry:
		return IncomeEntryStr
	case ExpenseEntry:
		return ExpenseEntryStr
	case TransferEntry:
		return TransferEntryStr
	case BuyStockEntry:
		return BuyStockEntryStr
	case SellStockEntry:
		return SellStockEntryStr
	default:
		return "unknown"
	}
}

const (
	UnsetEntry EntryType = iota
	IncomeEntry
	ExpenseEntry
	TransferEntry
	BuyStockEntry
	SellStockEntry
)

const (
	IncomeEntryStr    = "income"
	ExpenseEntryStr   = "expense"
	TransferEntryStr  = "transfer"
	BuyStockEntryStr  = "buystock"
	SellStockEntryStr = "sellstock"
)

func ParseEntryType(in string) (EntryType, error) {
	switch strings.ToLower(in) {
	case IncomeEntryStr:
		return IncomeEntry, nil
	case ExpenseEntryStr:
		return ExpenseEntry, nil
	case TransferEntryStr:
		return TransferEntry, nil
	case BuyStockEntryStr:
		return BuyStockEntry, nil
	default:
		return UnsetEntry, fmt.Errorf("invalid entry type: %s", in)
	}
}

// validates an entry before it is created
//
//nolint:nestif //many validation ifs but simple to read
func validateEntry(ctx context.Context, store *Store, item Entry, tenant string) error {
	if item.Description == "" {
		return ValidationErr("description cannot be empty")
	}
	if item.Type == UnsetEntry {
		return ValidationErr("entry type cannot be empty")
	}
	if item.TargetAmount == 0 {
		return ValidationErr("target amount cannot be empty")
	}

	if item.TargetAccountID == 0 {
		return ValidationErr("target account cannot be empty")
	}

	if item.Date.IsZero() {
		return ValidationErr("date cannot be zero")
	}

	if item.Type == TransferEntry {
		if item.OriginAmount == 0 {
			return ValidationErr("origin amount cannot be empty")
		}

		if item.OriginAccountID == 0 {
			return ValidationErr("origin account cannot be empty")
		}

		targetAccount, err := store.GetAccount(ctx, item.TargetAccountID, tenant)
		if err != nil {
			return err
		}
		if targetAccount.Type != Cash {
			return fmt.Errorf("target account must be of type cash")
		}

		originAccount, err := store.GetAccount(ctx, item.OriginAccountID, tenant)
		if err != nil {
			return err
		}
		if originAccount.Type != Cash {
			return fmt.Errorf("origin account must be of type cash")
		}
	}
	return nil
}

type sumResult struct {
	Sum   float64
	Count int64
}

type sumByCategoryOpts struct {
	StartDate   time.Time
	EndDate     time.Time
	CategoryIds []uint
	EntryType   EntryType
	Tenant      string
}

// sumEntryByCategories is an internal function to sum the values of entries filtering by Categories
// only income and expenses permitted in the sum by categories; transfers and other operations are not added to a category
func (store *Store) sumEntryByCategories(ctx context.Context, opts sumByCategoryOpts) (sumResult, error) {
	db := store.db.WithContext(ctx).Where("owner_id = ?", opts.Tenant)
	// Filter by date range
	db = db.Where("date BETWEEN ? AND ?", opts.StartDate, opts.EndDate)

	// Fiter by type
	if opts.EntryType != IncomeEntry && opts.EntryType != ExpenseEntry {
		return sumResult{Sum: 0, Count: 0}, fmt.Errorf("entry type not supported, must be income or expense: %s", opts.EntryType)
	}
	db = db.Where("type IN ? ", opts.EntryType)

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

type accountBalanceOpts struct {
	date time.Time

	Tenant string
}

// getAccountStatus sums all values from an account to calculate the current balance
func (store *Store) getAccountBalance(ctx context.Context, opts accountBalanceOpts) (sumResult, error) {
	db := store.db.WithContext(ctx).Where("owner_id = ?", opts.Tenant)
	db = db.Where("date BEFORE ?", opts.date)

	//
	// Fiter by type
	if opts.EntryType != IncomeEntry && opts.EntryType != ExpenseEntry {
		return sumResult{Sum: 0, Count: 0}, fmt.Errorf("entry type not supported, must be income or expense: %s", opts.EntryType)
	}

	db = db.Where("type IN ?", []EntryType{IncomeEntry, ExpenseEntry})

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
