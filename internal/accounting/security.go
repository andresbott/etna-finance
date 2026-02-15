package accounting

import (
	"context"
	"errors"
	"time"

	"golang.org/x/text/currency"
	"gorm.io/gorm"
)

// Security represents a tradeable instrument (e.g. a stock).
type Security struct {
	ID       uint
	Symbol   string
	Name     string
	Currency currency.Unit
}

type dbSecurity struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	OwnerId   string         `gorm:"index"`

	Symbol   string
	Name     string
	Currency string
}

func dbToSecurity(in dbSecurity) Security {
	return Security{
		ID:       in.ID,
		Symbol:   in.Symbol,
		Name:     in.Name,
		Currency: currency.MustParseISO(in.Currency),
	}
}

var ErrSecurityNotFound = errors.New("security not found")

func (store *Store) CreateSecurity(ctx context.Context, item Security, tenant string) (uint, error) {
	if item.Symbol == "" {
		return 0, ErrValidation("symbol cannot be empty")
	}
	if item.Currency == (currency.Unit{}) {
		return 0, ErrValidation("currency cannot be empty")
	}

	payload := dbSecurity{
		OwnerId:  tenant,
		Symbol:   item.Symbol,
		Name:     item.Name,
		Currency: item.Currency.String(),
	}

	d := store.db.WithContext(ctx).Create(&payload)
	if d.Error != nil {
		return 0, d.Error
	}
	return payload.ID, nil
}

func (store *Store) GetSecurity(ctx context.Context, id uint, tenant string) (Security, error) {
	var payload dbSecurity
	d := store.db.WithContext(ctx).Where("id = ? AND owner_id = ?", id, tenant).First(&payload)
	if d.Error != nil {
		if errors.Is(d.Error, gorm.ErrRecordNotFound) {
			return Security{}, ErrSecurityNotFound
		}
		return Security{}, d.Error
	}
	return dbToSecurity(payload), nil
}

func (store *Store) ListSecurities(ctx context.Context, tenant string) ([]Security, error) {
	var results []dbSecurity
	if err := store.db.WithContext(ctx).
		Where("owner_id = ?", tenant).
		Order("id ASC").
		Find(&results).Error; err != nil {
		return nil, err
	}
	out := make([]Security, 0, len(results))
	for _, r := range results {
		out = append(out, dbToSecurity(r))
	}
	return out, nil
}
