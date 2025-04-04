package finance

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"time"
)

// Entry is a
type Entry struct {
	Id         uint
	Name       string
	Amount     float64
	Date       time.Time
	Locked     bool      // does not accept changes anymore
	Type       EntryType //income, transfer, spend
	AccountId  uint
	CategoryId uint
}

// dbAccount is the DB internal representation of a Bookmark
type dbEntry struct {
	Id         uint `gorm:"primarykey"`
	Name       string
	Amount     float64
	Type       int8
	OwnerId    string    `gorm:"index"`
	Date       time.Time `gorm:"index"`
	Locked     bool
	AccountId  uint `gorm:"index"`
	CategoryId uint `gorm:"index"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// getAccount is used internally to transform the db struct to public facing struct
func getEntry(in dbEntry) Entry {
	return Entry{
		Id:         in.Id,
		Name:       in.Name,
		Amount:     in.Amount,
		Date:       in.Date,
		Locked:     in.Locked,
		AccountId:  in.AccountId,
		CategoryId: in.CategoryId,
		Type:       EntryType(in.Type),
	}
}

type EntryType int8

const (
	UnsetEntry EntryType = iota
	IncomeEntry
	ExpenseEntry
	TransferEntry
)

var EntryNotFoundErr = errors.New("entry not found")

func (store *Store) CreateEntry(ctx context.Context, item Entry, tenant string) (uint, error) {

	if item.Name == "" {
		return 0, ValidationErr("name cannot be empty")
	}
	if item.Type == UnsetEntry {
		return 0, ValidationErr("entry type cannot be empty")
	}
	if item.Amount == 0 {
		return 0, ValidationErr("amount cannot be empty")
	}
	if item.Date.IsZero() {
		return 0, ValidationErr("date cannot be zero")
	}

	payload := dbEntry{
		OwnerId:    tenant, // ensure tenant is set by the signature
		Name:       item.Name,
		Type:       int8(item.Type),
		Amount:     item.Amount,
		Date:       item.Date,
		Locked:     false, // entries are always created unlocked
		AccountId:  item.AccountId,
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
			return Entry{}, EntryNotFoundErr
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
		return EntryNotFoundErr
	}
	return nil
}

type EntryUpdatePayload struct {
	Name   *string
	Amount *int
	Date   *time.Time
}

func (store *Store) UpdateEntry(item EntryUpdatePayload, Id uint, tenant string) error {
	payload := map[string]any{}
	hasChanges := false

	if item.Name != nil {
		hasChanges = true
		payload["name"] = *item.Name
	}

	if item.Amount != nil {
		hasChanges = true
		payload["amount"] = *item.Amount
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
			return errors.New("entry not found")
		}
	}
	return nil
}

const MaxSearchResults = 90
const DefaultSearchResults = 30

func (store *Store) ListEntries(ctx context.Context, startDate, endDate time.Time, accountID *uint, limit, page int, tenant string) ([]Entry, error) {
	db := store.db.WithContext(ctx).Where("owner_id = ?", tenant)

	// Filter by date range
	db = db.Where("date BETWEEN ? AND ?", startDate, endDate)

	// Filter by account ID if provided
	if accountID != nil {
		db = db.Where("account_id = ?", *accountID)
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

	db = db.Order("created_at DESC").Limit(limit).Offset(offset)

	var results []dbEntry
	if err := db.Find(&results).Error; err != nil {
		return nil, err
	}

	var entries []Entry
	for _, got := range results {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			entries = append(entries, getEntry(got))
		}
	}
	return entries, nil
}

func (store *Store) LockEntries(ctx context.Context, date time.Time) error {
	// locks all entries older than certain date
	return nil
}
