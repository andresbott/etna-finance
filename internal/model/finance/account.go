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

// =======================================================================================
// Account Provider
// =======================================================================================

// AccountProvider represents the institution that provides an account, e.g. a bank or a broker
// one user can have multiple accounts with the same provider.
type AccountProvider struct {
	ID          uint
	Name        string
	Description string
	Accounts    []Account
}

type dbAccountProvider struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	OwnerId   string         `gorm:"index"`

	Name        string
	Description string
	Accounts    []dbAccount `gorm:"foreignKey:ProviderID;"` // has many
}

func (store *Store) CreateAccountProvider(ctx context.Context, item AccountProvider, tenant string) (uint, error) {
	if item.Name == "" {
		return 0, ValidationErr("name cannot be empty")
	}

	payload := dbAccountProvider{
		OwnerId:     tenant, // ensure tenant is set by the signature
		Name:        item.Name,
		Description: item.Description,
	}

	d := store.db.WithContext(ctx).Create(&payload)
	if d.Error != nil {
		return 0, d.Error
	}
	return payload.ID, nil
}

func (store *Store) GetAccountProvider(ctx context.Context, Id uint, tenant string) (AccountProvider, error) {

	var payload dbAccountProvider
	d := store.db.WithContext(ctx).Where("id = ? AND owner_id = ?", Id, tenant).First(&payload)
	if d.Error != nil {
		if errors.Is(d.Error, gorm.ErrRecordNotFound) {
			return AccountProvider{}, ErrAccountProviderNotFound
		} else {
			return AccountProvider{}, d.Error
		}
	}
	return dbToAccountProvider(payload), nil
}

type AccountProviderUpdatePayload struct {
	Name        *string
	Description *string
	//Accounts    []Accountw
}

func (store *Store) UpdateAccountProvider(item AccountProviderUpdatePayload, Id uint, tenant string) error {

	payload := map[string]any{}
	hasChanges := false

	if item.Name != nil {
		hasChanges = true
		payload[store.AccountProviderColNames["Name"]] = *item.Name
	}
	if item.Description != nil {
		hasChanges = true
		payload[store.AccountProviderColNames["Description"]] = *item.Description
	}
	// TODO add account IDs

	if !hasChanges {
		return ErrNoChanges
	}

	q := store.db.Where("id = ? AND owner_id = ?", Id, tenant).Model(&dbAccountProvider{}).Updates(payload)
	if q.Error != nil {
		return q.Error
	}
	if q.RowsAffected == 0 {
		return ErrAccountProviderNotFound
	}
	return nil

}

func (store *Store) ListAccountsProvider(ctx context.Context, tenant string, fetchAccounts bool) ([]AccountProvider, error) {

	// NOTE I don't forsee the need of pagination for private usage
	db := store.db.WithContext(ctx)
	db = db.Order("db_account_providers.id ASC").Where("db_account_providers.owner_id = ?", tenant)

	if fetchAccounts {
		db = db.Preload("Accounts", "owner_id = ?", tenant)
	}

	var results []dbAccountProvider
	if err := db.Find(&results).Error; err != nil {
		return nil, err
	}

	items := make([]AccountProvider, 0, len(results))
	for _, got := range results {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			items = append(items, dbToAccountProvider(got))
		}
	}

	return items, nil

}

func (store *Store) DeleteAccountProvider(ctx context.Context, id uint, tenant string) error {

	// using a manual constraint check instead of sql since we can't ensure that constraints are available on sqlite
	var count int64
	err := store.db.WithContext(ctx).Model(&dbAccount{}).
		Where("provider_id = ? AND owner_id = ?", id, tenant).
		Count(&count).Error
	if err != nil {
		return fmt.Errorf("failed to check associated accounts: %w", err)
	}

	if count > 0 {
		return ErrAccountConstraintViolation
	}

	d := store.db.WithContext(ctx).Where("id = ? AND owner_id = ?", id, tenant).Delete(&dbAccountProvider{})
	if d.Error != nil {
		return d.Error
	}

	if d.RowsAffected == 0 {
		return ErrAccountProviderNotFound
	}
	return nil
}

// dbToAccount is used internally to transform the db struct to public facing struct
func dbToAccountProvider(in dbAccountProvider) AccountProvider {
	accounts := make([]Account, len(in.Accounts))
	for i, item := range in.Accounts {
		accounts[i] = dbToAccount(item)
	}
	return AccountProvider{
		ID:          in.ID,
		Name:        in.Name,
		Description: in.Description,
		Accounts:    accounts,
	}
}

// =======================================================================================
// Account
// =======================================================================================

// Account is the public representation of an account
type Account struct {
	ID                uint `gorm:"primarykey"`
	AccountProviderID uint
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         gorm.DeletedAt `gorm:"index"`

	Name        string
	Description string
	Currency    currency.Unit
	Type        AccountType
}

// dbAccount is the DB internal representation of an Account
type dbAccount struct {
	ID         uint `gorm:"primarykey"`
	ProviderID uint `gorm:"index"`
	Name       string
	Type       AccountType
	Currency   string
	OwnerId    string `gorm:"index"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type AccountType int

const (
	Unknown AccountType = iota
	Cash    AccountType = iota
	Investment
)
const (
	CashAccount       = "cash"
	InvestmentAccount = "investment"
)

func (t AccountType) String() string {
	switch t {
	case Cash:
		return CashAccount
	case Investment:
		return InvestmentAccount
	default:
		return "unknown"
	}
}

func ParseAccountType(in string) (AccountType, error) {
	switch strings.ToLower(in) {
	case CashAccount:
		return Cash, nil
	case InvestmentAccount:
		return Investment, nil
	default:
		return Unknown, fmt.Errorf("invalid account type: %s", in)
	}
}

var ErrAccountNotFound = errors.New("account not found")
var ErrAccountProviderNotFound = errors.New("account provider not found")
var ErrAccountConstraintViolation = errors.New("account constraint violation")
var ErrNoChanges = errors.New("no changes were performed")

// dbToAccount is used internally to transform the db struct to public facing struct
func dbToAccount(in dbAccount) Account {
	return Account{
		ID:                in.ID,
		AccountProviderID: in.ProviderID,
		Name:              in.Name,
		Currency:          currency.MustParseISO(in.Currency),
		Type:              in.Type,
	}
}

func (store *Store) CreateAccount(ctx context.Context, item Account, tenant string) (uint, error) {

	// TODO: limit the amount of accounts a user can create (?)

	if item.Name == "" {
		return 0, ValidationErr("name cannot be empty")
	}
	if item.Currency == (currency.Unit{}) {
		return 0, ValidationErr("currency cannot be empty")
	}
	if item.AccountProviderID == 0 {
		return 0, ValidationErr("account provider ID cannot be empty")
	}
	// validate that the account provider tenant is also account tenant
	_, err := store.GetAccountProvider(ctx, item.AccountProviderID, tenant)
	if err != nil && errors.Is(err, ErrAccountProviderNotFound) {
		return 0, ValidationErr("account provider ID not found")
	}

	payload := dbAccount{
		OwnerId:    tenant, // ensure tenant is set by the signature
		ProviderID: item.AccountProviderID,
		Name:       item.Name,
		Type:       item.Type,
		Currency:   item.Currency.String(),
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
			return Account{}, ErrAccountNotFound
		} else {
			return Account{}, d.Error
		}
	}
	return dbToAccount(payload), nil
}

type AccountUpdatePayload struct {
	Name       *string
	Currency   *currency.Unit
	ProviderID *int
	Type       AccountType
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

	if item.ProviderID != nil {
		hasChanges = true
		// TODO ensure provider restriction (?)
		payload[store.AccountColNames["ProviderID"]] = item.ProviderID
	}

	if !hasChanges {
		return ErrNoChanges
	}

	q := store.db.Where("id = ? AND owner_id = ?", Id, tenant).Model(&dbAccount{}).Updates(payload)
	if q.Error != nil {
		return q.Error
	}
	if q.RowsAffected == 0 {
		return ErrAccountNotFound
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
		return ErrAccountNotFound
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
			accounts = append(accounts, dbToAccount(got))
		}
	}
	return accounts, nil
}

// ListAccountsMap is a wrapper function around ListAccounts that returns a map [uint]Account where the
// key is the account id
func (store *Store) ListAccountsMap(ctx context.Context, tenant string) (map[uint]Account, error) {
	accounts, err := store.ListAccounts(ctx, tenant)
	if err != nil {
		return nil, err
	}

	result := make(map[uint]Account, len(accounts))
	for _, account := range accounts {
		result[account.ID] = account
	}
	return result, nil
}
