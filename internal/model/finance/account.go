package finance

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/text/currency"
	"gorm.io/gorm"
	"strings"
	"time"
)

type AccountType int

func (t AccountType) String() string {
	switch t {
	case Cash:
		return CashAccount
	case Bank:
		return BankAccount
	case Stocks:
		return StocksAccount
	default:
		return "unknown"
	}
}

const (
	Unknown AccountType = iota
	Cash    AccountType = iota
	Bank
	Stocks
)

const CashAccount = "cash"
const BankAccount = "bank"
const StocksAccount = "stocks"

func ParseAccountType(in string) (AccountType, error) {
	switch strings.ToLower(in) {
	case CashAccount:
		return Cash, nil

	case BankAccount:
		return Bank, nil
	case StocksAccount:
		return Stocks, nil
	default:
		return Unknown, fmt.Errorf("invalid account type: %s", in)
	}
}

// Account is the public representation of an account
type Account struct {
	ID       uint
	Name     string
	Currency currency.Unit
	Type     AccountType
}

var AccountNotFoundErr = errors.New("account not found")

// getAccount is used internally to transform the db struct to public facing struct
func getAccount(in dbAccount) Account {
	return Account{
		ID:       in.ID,
		Name:     in.Name,
		Currency: currency.MustParseISO(in.Currency),
		Type:     in.Type,
	}
}

// dbAccount is the DB internal representation of an Account
type dbAccount struct {
	ID       uint `gorm:"primarykey"`
	Name     string
	Type     AccountType
	Currency string
	OwnerId  string `gorm:"index"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (store *Store) CreateAccount(ctx context.Context, item Account, tenant string) (uint, error) {

	if item.Name == "" {
		return 0, ValidationErr("name cannot be empty")
	}
	if item.Currency == (currency.Unit{}) {
		return 0, ValidationErr("currency cannot be empty")
	}
	payload := dbAccount{
		OwnerId:  tenant, // ensure tenant is set by the signature
		Name:     item.Name,
		Type:     item.Type,
		Currency: item.Currency.String(),
	}

	d := store.db.WithContext(ctx).Create(&payload)
	if d.Error != nil {
		return 0, d.Error
	}
	return payload.ID, nil
}

func (store *Store) GetAccount(ctx context.Context, Id uint, tenant string) (Account, error) {
	var payload dbAccount
	d := store.db.WithContext(ctx).Where("id = ? AND owner_id = ?", Id, tenant).First(&payload)
	if d.Error != nil {
		if errors.Is(d.Error, gorm.ErrRecordNotFound) {
			return Account{}, AccountNotFoundErr
		} else {
			return Account{}, d.Error
		}
	}
	return getAccount(payload), nil
}

type AccountUpdatePayload struct {
	Name     *string
	Currency *currency.Unit
	Type     AccountType
}

func (store *Store) UpdateAccount(item AccountUpdatePayload, Id uint, tenant string) error {
	payload := map[string]any{}
	hasChanges := false

	if item.Name != nil {
		hasChanges = true
		payload[store.AccountColNames["Name"]] = *item.Name
	}

	if item.Type != Unknown {
		hasChanges = true
		payload[store.AccountColNames["Type"]] = item.Type
	}

	if item.Currency != nil {
		hasChanges = true
		payload[store.AccountColNames["Currency"]] = item.Currency.String()
	}

	if hasChanges {
		q := store.db.Where("id = ? AND owner_id = ?", Id, tenant).Model(&dbAccount{}).Updates(payload)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected == 0 {
			return AccountNotFoundErr
		}
	}
	return nil
}

func (store *Store) DeleteAccount(ctx context.Context, Id uint, tenant string) error {
	// TODO add a Delete constraint, don allow if it still has entries
	d := store.db.WithContext(ctx).Where("id = ? AND owner_id = ?", Id, tenant).Delete(&dbAccount{})
	if d.Error != nil {
		return d.Error
	}
	if d.RowsAffected == 0 {
		return AccountNotFoundErr
	}
	return nil
}

func (store *Store) ListAccounts(ctx context.Context, tenant string) ([]Account, error) {

	db := store.db.WithContext(ctx)
	// NOTE I don't forsee the need of pagination for private usage
	db = db.Order("id ASC").Where("owner_id = ?", tenant)

	var results []dbAccount
	if err := db.Find(&results).Error; err != nil {
		return []Account{}, err
	}

	accounts := []Account{}
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
