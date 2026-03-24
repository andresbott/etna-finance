package accounting

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/text/currency"
	"gorm.io/gorm"
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
	Icon        string
	Accounts    []Account
}

type dbAccountProvider struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Name        string
	Description string
	Icon        string
	Accounts    []dbAccount `gorm:"foreignKey:ProviderID;"` // has many
}

func (store *Store) CreateAccountProvider(ctx context.Context, item AccountProvider) (uint, error) {
	if item.Name == "" {
		return 0, ErrValidation("name cannot be empty")
	}

	payload := dbAccountProvider{
		Name:        item.Name,
		Description: item.Description,
		Icon:        item.Icon,
	}

	d := store.db.WithContext(ctx).Create(&payload)
	if d.Error != nil {
		return 0, d.Error
	}
	return payload.ID, nil
}

func (store *Store) GetAccountProvider(ctx context.Context, Id uint) (AccountProvider, error) {

	var payload dbAccountProvider
	d := store.db.WithContext(ctx).Where("id = ?", Id).First(&payload)
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
	Icon        *string
}

func (store *Store) UpdateAccountProvider(item AccountProviderUpdatePayload, Id uint) error {
	updateStruct := dbAccountProvider{}
	var selectedFields []string

	if item.Name != nil {
		updateStruct.Name = *item.Name
		selectedFields = append(selectedFields, "Name")
	}
	if item.Description != nil {
		updateStruct.Description = *item.Description
		selectedFields = append(selectedFields, "Description")
	}
	if item.Icon != nil {
		updateStruct.Icon = *item.Icon
		selectedFields = append(selectedFields, "Icon")
	}
	if len(selectedFields) == 0 {
		return ErrNoChanges
	}

	// Perform the update
	q := store.db.Model(&dbAccountProvider{}).
		Where("id = ?", Id).
		Select(selectedFields).
		Updates(updateStruct)

	if q.Error != nil {
		return q.Error
	}

	if q.RowsAffected == 0 {
		return ErrAccountProviderNotFound
	}

	return nil

}

func (store *Store) ListAccountsProvider(ctx context.Context, fetchAccounts bool) ([]AccountProvider, error) {

	// NOTE I don't forsee the need of pagination for private usage
	db := store.db.WithContext(ctx)
	db = db.Order("db_account_providers.id ASC")

	if fetchAccounts {
		db = db.Preload("Accounts")
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

func (store *Store) DeleteAccountProvider(ctx context.Context, id uint) error {

	// using a manual constraint check instead of sql since we can't ensure that constraints are available on sqlite
	var count int64
	err := store.db.WithContext(ctx).Model(&dbAccount{}).
		Where("provider_id = ?", id).
		Count(&count).Error
	if err != nil {
		return fmt.Errorf("failed to check associated accounts: %w", err)
	}

	if count > 0 {
		return ErrAccountConstraintViolation
	}

	d := store.db.WithContext(ctx).Where("id = ?", id).Delete(&dbAccountProvider{})
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
		Icon:        in.Icon,
		Accounts:    accounts,
	}
}

var ErrAccountProviderNotFound = errors.New("account provider not found")
var ErrAccountConstraintViolation = errors.New("account constraint violation")
var ErrNoChanges = errors.New("no changes were performed")

// =======================================================================================
// Account
// =======================================================================================

var ErrAccountNotFound = errors.New("account not found")
var ErrAccountContainsEntries = errors.New("account still contains referenced transactions")

type AccountType int

const (
	UnknownAccountType    AccountType = iota
	CashAccountType                   // e.g. wallet
	CheckinAccountType                // where the salary goes
	SavingsAccountType                // where you save money and get dividends
	InvestmentAccountType             // stocks and others (vested)
	UnvestedAccountType               // not yet accessible (e.g. unvested RSUs); can transfer to Investment
	LentAccountType                   // money lent to others; owned but not in any account
	PensionAccountType                // pension/retirement fund; contributions via transfer, value changes via revaluation
)

func (t AccountType) String() string {
	switch t {
	case CashAccountType:
		return "Cash"
	case CheckinAccountType:
		return "Checkin"
	case SavingsAccountType:
		return "Savings"
	case InvestmentAccountType:
		return "Investment"
	case UnvestedAccountType:
		return "Unvested"
	case LentAccountType:
		return "Lent"
	case PensionAccountType:
		return "Pension"
	default:
		return "Unknown"
	}
}

// RequiresCurrency returns true for account types that require a currency (all except Unknown).
func (t AccountType) RequiresCurrency() bool {
	switch t {
	case CashAccountType, CheckinAccountType, SavingsAccountType, InvestmentAccountType, UnvestedAccountType, LentAccountType, PensionAccountType:
		return true
	default:
		return false
	}
}

type Account struct {
	ID                uint
	AccountProviderID uint
	Name              string
	Description       string
	Icon              string
	Notes             string
	Currency          currency.Unit
	Type              AccountType
	ImportProfileID   uint // 0 = no linked import profile
}

// dbAccount is the DB internal representation of an Account
type dbAccount struct {
	ID              uint `gorm:"primarykey"`
	ProviderID      uint `gorm:"index"`
	Name            string
	Description     string
	Icon            string
	Notes           string `gorm:"size:1024"`
	Type            AccountType
	Currency        string
	ImportProfileID uint `gorm:"default:null"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// dbToAccount is used internally to transform the db struct to public facing struct.
func dbToAccount(in dbAccount) Account {
	var cur currency.Unit
	if in.Currency != "" {
		cur = currency.MustParseISO(in.Currency)
	}
	return Account{
		ID:                in.ID,
		AccountProviderID: in.ProviderID,
		Name:              in.Name,
		Description:       in.Description,
		Icon:              in.Icon,
		Notes:             in.Notes,
		Currency:          cur,
		Type:              in.Type,
		ImportProfileID:   in.ImportProfileID,
	}
}

func (store *Store) CreateAccount(ctx context.Context, item Account) (uint, error) {
	if item.Name == "" {
		return 0, ErrValidation("name cannot be empty")
	}
	if item.Type.RequiresCurrency() && item.Currency == (currency.Unit{}) {
		return 0, ErrValidation("currency cannot be empty")
	}
	if item.AccountProviderID == 0 {
		return 0, ErrValidation("account provider id cannot be empty")
	}
	_, err := store.GetAccountProvider(ctx, item.AccountProviderID)
	if err != nil && errors.Is(err, ErrAccountProviderNotFound) {
		return 0, ErrValidation("account provider id not found")
	}

	currencyStr := ""
	if item.Type.RequiresCurrency() {
		currencyStr = item.Currency.String()
	}
	payload := dbAccount{
		ProviderID:      item.AccountProviderID,
		Name:            item.Name,
		Description:     item.Description,
		Icon:            item.Icon,
		Notes:           item.Notes,
		Type:            item.Type,
		Currency:        currencyStr,
		ImportProfileID: item.ImportProfileID,
	}

	d := store.db.WithContext(ctx).Create(&payload)
	if d.Error != nil {
		return 0, d.Error
	}
	return payload.ID, nil
}

func (store *Store) GetAccount(ctx context.Context, Id uint) (Account, error) {
	var payload dbAccount
	d := store.db.WithContext(ctx).Where("id = ?", Id).First(&payload)
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
	Name            *string
	Description     *string
	Icon            *string
	Notes           *string
	Currency        *currency.Unit
	ProviderID      *uint
	Type            AccountType
	ImportProfileID *uint
}

func (store *Store) UpdateAccount(ctx context.Context, item AccountUpdatePayload, Id uint) error {
	// Resolve target account type for currency rules: use payload type if set, else current account type
	targetType := item.Type
	if targetType == UnknownAccountType && item.Currency != nil {
		current, err := store.GetAccount(ctx, Id)
		if err != nil {
			return err
		}
		targetType = current.Type
	}

	// Build a dbAccount struct with only the fields to update
	updateStruct := dbAccount{}
	var selectedFields []string

	if item.Name != nil {
		updateStruct.Name = *item.Name
		selectedFields = append(selectedFields, "Name")
	}

	if item.Description != nil {
		updateStruct.Description = *item.Description
		selectedFields = append(selectedFields, "Description")
	}

	if item.Icon != nil {
		updateStruct.Icon = *item.Icon
		selectedFields = append(selectedFields, "Icon")
	}

	if item.Notes != nil {
		updateStruct.Notes = *item.Notes
		selectedFields = append(selectedFields, "Notes")
	}

	if item.Type != UnknownAccountType {
		updateStruct.Type = item.Type
		selectedFields = append(selectedFields, "Type")
	}

	// Currency: required for Cash/Checkin/Savings/Investment/Unvested; store when provided
	if item.Currency != nil {
		if targetType.RequiresCurrency() {
			updateStruct.Currency = item.Currency.String()
		}
		selectedFields = append(selectedFields, "Currency")
	}

	if item.ProviderID != nil {
		updateStruct.ProviderID = *item.ProviderID
		selectedFields = append(selectedFields, "ProviderID")
	}

	if item.ImportProfileID != nil {
		updateStruct.ImportProfileID = *item.ImportProfileID
		selectedFields = append(selectedFields, "ImportProfileID")
	}

	if len(selectedFields) == 0 {
		return ErrNoChanges
	}

	// Perform the update
	q := store.db.Model(&dbAccount{}).
		WithContext(ctx).
		Where("id = ?", Id).
		Select(selectedFields).
		Updates(updateStruct)

	if q.Error != nil {
		return q.Error
	}

	if q.RowsAffected == 0 {
		return ErrAccountNotFound
	}

	return nil
}

func (store *Store) DeleteAccount(ctx context.Context, Id uint) error {
	db := store.db.WithContext(ctx)

	// Check if any entries reference this account
	var count int64
	if err := db.Model(&dbEntry{}).
		Where("account_id = ?", Id).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check account entries: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("unable to delete account: %w", ErrAccountContainsEntries)
	}

	d := store.db.Where("id = ?", Id).Delete(&dbAccount{})
	if d.Error != nil {
		return d.Error
	}
	if d.RowsAffected == 0 {
		return ErrAccountNotFound
	}
	return nil
}

func (store *Store) ListAccounts(ctx context.Context) ([]Account, error) {

	db := store.db.WithContext(ctx)
	// NOTE I don't forsee the need of pagination for private usage
	db = db.Order("id ASC")

	var results []dbAccount
	if err := db.Find(&results).Error; err != nil {
		return []Account{}, err
	}

	var accounts []Account
	for _, got := range results {
		accounts = append(accounts, dbToAccount(got))
	}
	return accounts, nil
}

func (store *Store) ListAccountsByCurrency(ctx context.Context) (map[currency.Unit][]Account, error) {
	results, err := store.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}

	accountsByCurrency := make(map[currency.Unit][]Account)
	for _, got := range results {
		account := got
		accountsByCurrency[account.Currency] = append(accountsByCurrency[account.Currency], account)
	}

	return accountsByCurrency, nil
}

// ListAccountsMap is a wrapper function around ListAccounts that returns a map [uint]Account where the
// key is the account id
func (store *Store) ListAccountsMap(ctx context.Context) (map[uint]Account, error) {
	accounts, err := store.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[uint]Account, len(accounts))
	for _, account := range accounts {
		result[account.ID] = account
	}
	return result, nil
}
