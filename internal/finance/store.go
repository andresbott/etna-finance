package finance

import (
	"fmt"
	"gorm.io/gorm"
	"time"
)

type Store struct {
	db *gorm.DB

	AccountColNames         map[string]string // hold a map of struct field names to db column names
	AccountProviderColNames map[string]string // hold a map of struct field names to db column names
}

func New(db *gorm.DB) (*Store, error) {
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

	err = db.AutoMigrate(&dbAccount{}, &dbAccountProvider{}, &dbEntry{}, &dbTransaction{})
	if err != nil {
		return nil, err
	}

	return &b, nil
}

type dbTransaction struct {
	Id          uint      `gorm:"primaryKey"`
	Date        time.Time `gorm:"not null"`
	Description string    `gorm:"size:255"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Entries     []dbEntry      `gorm:"foreignKey:TransactionID"` // One-to-many relationship
}

type dbEntry struct {
	Id            uint `gorm:"primarykey"`
	TransactionID uint `gorm:"not null;index"` // Foreign key
	AccountID     uint `gorm:"not null;index"` // Foreign key

	Amount   float64 `gorm:"not null"` // Amount in account currency
	Quantity float64 // -- for stock shares (nullable for cash-only entries)

	entryTyp EntryType //income, expense, transferIn transferOut, stockbuy, stock sell

	OwnerId   string `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// Entry represents a debit/credit to a specific account
type Entry struct {
}

type ErrValidation string

func (v ErrValidation) Error() string {
	return string(v)
}
