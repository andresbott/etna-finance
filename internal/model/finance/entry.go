package finance

import (
	"context"
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
	Type        EntryType //income, transfer, spend, stock buy, stock sell ( like transfer with stock amounts added)

	StockAmount float64 // used to track the amount of stocks in the account

	// target is the account that gets the operation type, e.g. income or expense
	TargetAmount          float64
	TargetAccountID       uint
	TargetAccountName     string
	TargetAccountCurrency currency.Unit

	// origin is only mandatory for transfer operations where we move from one account to another
	OriginAmount          float64
	OriginAccountID       uint
	OriginAccountName     string
	OriginAccountCurrency currency.Unit

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
	case StockValueEntry:
		return StockValueEntryStr
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
	StockValueEntry
)

const (
	IncomeEntryStr     = "income"
	ExpenseEntryStr    = "expense"
	TransferEntryStr   = "transfer"
	BuyStockEntryStr   = "buystock"
	SellStockEntryStr  = "sellstock"
	StockValueEntryStr = "stockvalue"
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
	case StockValueEntryStr:
		return StockValueEntry, nil
	default:
		return UnsetEntry, fmt.Errorf("invalid entry type: %s", in)
	}
}

var ErrEntryNotFound = errors.New("entry not found")

func (store *Store) CreateEntry(ctx context.Context, item Entry, tenant string) (uint, error) {

	if item.Description == "" {
		return 0, ValidationErr("description cannot be empty")
	}
	if item.Type == UnsetEntry {
		return 0, ValidationErr("entry type cannot be empty")
	}
	if item.TargetAmount == 0 {
		return 0, ValidationErr("target amount cannot be empty")
	}
	if item.Date.IsZero() {
		return 0, ValidationErr("date cannot be zero")
	}

	payload := dbEntry{
		Description: item.Description,
		Date:        item.Date,
		Type:        int8(item.Type),
		OwnerId:     tenant, // ensure tenant is set by the signature
		Locked:      false,  // entries are always created unlocked

		TargetAmount:    item.TargetAmount,
		TargetAccountId: item.TargetAccountID,
		OriginAmount:    item.OriginAmount,
		OriginAccountId: item.OriginAccountID,

		CategoryId: item.CategoryId,
	}

	d := store.db.WithContext(ctx).Create(&payload)
	if d.Error != nil {
		return 0, d.Error
	}
	return payload.Id, nil
}

func (store *Store) GetEntry(ctx context.Context, Id uint, tenant string) (Entry, error) {
	var payload dbEntry
	d := store.db.WithContext(ctx).Where("id = ? AND owner_id = ?", Id, tenant).First(&payload)
	if d.Error != nil {
		if errors.Is(d.Error, gorm.ErrRecordNotFound) {
			return Entry{}, ErrEntryNotFound
		} else {
			return Entry{}, d.Error
		}
	}
	return getEntry(payload), nil
}

func (store *Store) DeleteEntry(ctx context.Context, Id uint, tenant string) error {
	d := store.db.WithContext(ctx).Where("id = ? AND owner_id = ?", Id, tenant).Delete(&dbEntry{})
	if d.Error != nil {
		return d.Error
	}
	if d.RowsAffected == 0 {
		return ErrEntryNotFound
	}
	return nil
}

type EntryUpdatePayload struct {
	Description *string
	Date        *time.Time

	StockAmount     *float64
	TargetAmount    *float64
	TargetAccountID *uint
	OriginAmount    *float64
	OriginAccountID *uint

	CategoryId *uint
}

func (store *Store) UpdateEntry(item EntryUpdatePayload, Id uint, tenant string) error {
	payload := map[string]any{}
	hasChanges := false

	if item.Description != nil {
		hasChanges = true
		payload["description"] = *item.Description
	}

	if item.TargetAmount != nil {
		hasChanges = true
		payload["target_amount"] = *item.TargetAmount
	}

	if item.OriginAmount != nil {
		hasChanges = true
		payload["origin_amount"] = *item.OriginAmount
	}

	if item.Date != nil {
		hasChanges = true
		payload["date"] = *item.Date
	}

	if hasChanges {
		q := store.db.Where("id = ? AND owner_id = ?", Id, tenant).Model(&dbEntry{}).Updates(payload)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected == 0 {
			return ErrEntryNotFound
		}
	}
	return nil
}

const MaxSearchResults = 90
const DefaultSearchResults = 30

func (store *Store) ListEntries(ctx context.Context, startDate, endDate time.Time, accountID *uint, limit, page int, tenant string) ([]Entry, error) {

	accountsMap, err := store.ListAccountsMap(ctx, tenant)
	if err != nil {
		return nil, err
	}

	db := store.db.WithContext(ctx).Where("owner_id = ?", tenant)

	// Filter by date range
	db = db.Where("date BETWEEN ? AND ?", startDate, endDate)

	// Filter by account ID if provided
	if accountID != nil {
		db = db.Where("target_account_id = ? OR origin_account_id = ?", *accountID, *accountID)
	}

	if limit == 0 {
		limit = DefaultSearchResults
	}
	if limit > MaxSearchResults {
		limit = MaxSearchResults
	}

	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	db = db.Order("date DESC").Limit(limit).Offset(offset)

	var results []dbEntry
	err = db.Find(&results).Error
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, got := range results {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			newEntry := getEntry(got)

			// inject account info into the results, I know this could also be done with sql
			if newEntry.OriginAccountID != 0 {
				if account, ok := accountsMap[newEntry.OriginAccountID]; ok {
					newEntry.OriginAccountName = account.Name
					newEntry.OriginAccountCurrency = account.Currency
				} else {
					return nil, fmt.Errorf("unable to find OriginAccountID  %d referenced by entry %d", newEntry.OriginAccountID, newEntry.Id)
				}
			}

			if newEntry.TargetAccountID != 0 {
				if account, ok := accountsMap[newEntry.TargetAccountID]; ok {
					newEntry.TargetAccountName = account.Name
					newEntry.TargetAccountCurrency = account.Currency
				} else {
					return nil, fmt.Errorf("unable to find TargetAccountID  %d referenced by entry %d", newEntry.TargetAccountID, newEntry.Id)
				}
			}
			entries = append(entries, newEntry)
		}
	}
	return entries, nil
}

func (store *Store) LockEntries(ctx context.Context, date time.Time) error {
	// locks all entries older than certain date
	return nil
}
