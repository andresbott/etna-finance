package finance

import (
	"context"
	"golang.org/x/text/currency"
	"gorm.io/gorm"
	"time"
)

// Entry is a
type Entry struct {
	Name     string
	Currency currency.Amount
	Locked   bool // does not accept changes anymore
	Type     int  //income, transfer, spend
}

// getAccount is used internally to transform the db struct to public facing struct
func getEntry(in dbEntry) Entry {
	return Entry{
		Name:     in.Name,
		Currency: in.Currency,
	}
}

// dbAccount is the DB internal representation of a Bookmark
type dbEntry struct {
	ID       uint `gorm:"primarykey"`
	Name     string
	Currency currency.Amount
	OwnerId  string `gorm:"index"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (bkm *Store) LockEntries(ctx context.Context, date time.Time) error {
	// locks all entries older than certain date
	return nil
}

func (bkm *Store) CreateEntry(ctx context.Context, item Account, tenant string) (uint, error) {

	if item.Name == "" {
		return 0, ValidationErr("name cannot be empty")
	}
	if item.Currency == (currency.Unit{}) {
		return 0, ValidationErr("currency cannot be empty")
	}
	payload := dbAccount{
		OwnerId: tenant, // ensure tenant is set by the signature
		Name:    item.Name,
	}

	d := bkm.db.WithContext(ctx).Create(&payload)
	if d.Error != nil {
		return 0, d.Error
	}
	return payload.ID, nil
}
