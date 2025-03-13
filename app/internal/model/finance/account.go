package finance

import (
	"context"
	"errors"
	"golang.org/x/text/currency"
	"gorm.io/gorm"
	"time"
)

// Account is the public representation of an account
type Account struct {
	Name     string
	Currency currency.Unit
	Type     int // bank, cash, stocks (other..)
}

// getAccount is used internally to transform the db struct to public facing struct
func getAccount(in dbAccount) Account {
	return Account{
		Name:     in.Name,
		Currency: in.Currency,
	}
}

// dbAccount is the DB internal representation of a Bookmark
type dbAccount struct {
	ID       uint `gorm:"primarykey"`
	Name     string
	Currency currency.Unit
	OwnerId  string `gorm:"index"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (bkm *Store) CreateAccount(ctx context.Context, item Account, tenant string) (uint, error) {

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

func (bkm *Store) GetAccount(ctx context.Context, Id uint, tenant string) (Account, error) {
	var payload dbAccount
	d := bkm.db.WithContext(ctx).Where("uuid = ? AND owner_id = ?", Id, tenant).Preload("Tags").Where("owner_id = ?", tenant).First(&payload)
	if d.Error != nil {
		if errors.Is(d.Error, gorm.ErrRecordNotFound) {
			return Account{}, NotFoundErr
		} else {
			return Account{}, d.Error
		}
	}
	return getAccount(payload), nil
}

func (bkm *Store) DeleteAccount(ctx context.Context, Id uint, tenant string) error {
	d := bkm.db.WithContext(ctx).Where("uuid = ? AND owner_id = ?", Id, tenant).Delete(&dbAccount{})
	if d.Error != nil {
		return d.Error
	}
	if d.RowsAffected == 0 {
		return NotFoundErr
	}
	return nil
}

func (bkm *Store) ListAccounts(ctx context.Context, tenant string) ([]Account, error) {

	db := bkm.db.WithContext(ctx)
	// NOTE I don't forsee the need of pagination for private usage
	db = db.Order("name DESC").Where("owner_id = ?", tenant)

	var results []dbAccount
	if err := db.Find(&results).Error; err != nil {
		return nil, err
	}

	var accounts []Account
	for _, got := range results {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			accounts = append(accounts, getAccount(got))
		}
	}
	return accounts, nil
}
