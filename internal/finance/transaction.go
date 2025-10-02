package finance

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"time"
)

type transactionType int

const (
	incomeTransaction transactionType = iota
	expenseTransaction
	transferTransaction
	stockTransaction
	loanTransaction
)

type dbTransaction struct {
	Id          uint      `gorm:"primaryKey"`
	Date        time.Time `gorm:"not null"`
	Description string    `gorm:"size:255"`
	Type        transactionType
	OwnerId     string `gorm:"index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Entries     []dbEntry      `gorm:"foreignKey:TransactionID"` // One-to-many relationship
}

type EntryType int

const (
	unknownEntryType EntryType = iota
	incomeEntry
	expenseEntry
	transferInEntry
	transferOutEntry
)

type dbEntry struct {
	Id            uint `gorm:"primarykey"`
	TransactionID uint `gorm:"not null;index"` // Foreign key
	AccountID     uint `gorm:"not null;index"` // Foreign key

	Amount   float64 `gorm:"not null"` // Amount in account currency
	Quantity float64 // -- for stock shares (nullable for cash-only entries)

	EntryType EntryType //income, expense, transferIn transferOut, stockbuy, stock sell

	OwnerId   string `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Transaction interface {
	isTransaction() // ensure only this package can implement the Transaction interface
}

type transaction struct{}

func (t transaction) isTransaction() {}

type EmptyTransaction struct {
	transaction
}

type Income struct {
	Description string
	Amount      float64
	AccountID   uint
	Date        time.Time

	transaction
}

type Expense struct {
	Description string
	Amount      float64
	AccountID   uint
	Date        time.Time

	transaction
}

type Transfer struct {
	Description     string
	OriginAmount    float64
	OriginAccountID uint
	TargetAmount    float64
	TargetAccountID uint
	Date            time.Time

	transaction
}

// CreateTransaction Allows to create a new transaction in the DB for a specific tenant
// note that there are only a limited type of transactions that can be created
func (store *Store) CreateTransaction(ctx context.Context, input Transaction, tenant string) (uint, error) {

	var tx dbTransaction
	switch input.(type) {
	case Income:

		item := input.(Income)
		acc, err := store.GetAccount(ctx, item.AccountID, tenant)
		if err != nil {
			return 0, fmt.Errorf("error creating transaction: %w", err)
		}
		if acc.Type != Cash {
			return 0, NewValidationErr("Incompatible account type for Income transaction")
		}

		tx = dbTransaction{
			Description: item.Description,
			Date:        item.Date,
			OwnerId:     tenant,
			Type:        incomeTransaction,
			Entries: []dbEntry{
				{AccountID: item.AccountID, Amount: item.Amount, EntryType: incomeEntry, OwnerId: tenant},
			},
		}
	case Expense:
		item := input.(Expense)
		tx = dbTransaction{
			Description: item.Description,
			Date:        item.Date,
			OwnerId:     tenant,
			Type:        expenseTransaction,
			Entries: []dbEntry{
				{AccountID: item.AccountID, Amount: -item.Amount, EntryType: expenseEntry, OwnerId: tenant},
			},
		}

	case Transfer:
		item := input.(Transfer)
		tx = dbTransaction{
			Description: item.Description,
			Date:        item.Date,
			OwnerId:     tenant,
			Type:        transferTransaction,
			Entries: []dbEntry{
				{AccountID: item.OriginAccountID, Amount: -item.OriginAmount, EntryType: transferOutEntry, OwnerId: tenant},
				{AccountID: item.TargetAccountID, Amount: item.TargetAmount, EntryType: transferInEntry, OwnerId: tenant},
			},
		}
	default:
		return 0, errors.New("invalid input type")
	}

	if tx.Description == "" {
		return 0, NewValidationErr("description cannot be empty")
	}

	if tx.Date.IsZero() {
		return 0, NewValidationErr("date cannot be zero")
	}

	for _, entry := range tx.Entries {
		if entry.Amount == 0 {
			return 0, NewValidationErr("amount cannot be zero")
		}
		if entry.AccountID == 0 {
			return 0, NewValidationErr("account ID cannot be zero")
		}
	}

	if err := store.db.WithContext(ctx).Create(&tx).Error; err != nil {
		return 0, err
	}

	return tx.Id, nil
}

var ErrTransactionNotFound = errors.New("transaction not found")
var ErrTransactionTypeNotFound = errors.New("transaction type not found")
var ErrEntryNotFound = errors.New("transaction entry not found")

// GetTransaction Returns a transaction after reading it from the DB
// Note that type assertion needs to be used to transform the Transaction into a specific type
func (store *Store) GetTransaction(ctx context.Context, Id uint, tenant string) (Transaction, error) {
	var payload dbTransaction
	q := store.db.WithContext(ctx).Preload("Entries").Where("id = ? AND owner_id = ?", Id, tenant).First(&payload)
	if q.Error != nil {
		if errors.Is(q.Error, gorm.ErrRecordNotFound) {
			return nil, ErrTransactionNotFound
		} else {
			return nil, q.Error
		}
	}
	return publicTransactions(payload)

}

// publicTransactions takes a db representation of the transaction and returns a specific type
func publicTransactions(in dbTransaction) (Transaction, error) {
	switch in.Type {
	case incomeTransaction:
		return Income{
			Description: in.Description,
			Date:        in.Date,
			Amount:      in.Entries[0].Amount,
			AccountID:   in.Entries[0].AccountID,
		}, nil
	case expenseTransaction:
		return Expense{
			Description: in.Description,
			Date:        in.Date,
			Amount:      -in.Entries[0].Amount,
			AccountID:   in.Entries[0].AccountID,
		}, nil
	case transferTransaction:
		var inEntity dbEntry
		var outEntry dbEntry
		entries := in.Entries
		for _, entry := range entries {
			if entry.EntryType == transferInEntry {
				inEntity = entry
			} else if entry.EntryType == transferOutEntry {
				outEntry = entry
			} else {
				return nil, fmt.Errorf("unexpected entry type: %v found in transfer", entry.EntryType)
			}
		}
		return Transfer{
			Description:     in.Description,
			OriginAmount:    -outEntry.Amount,
			OriginAccountID: outEntry.AccountID,
			TargetAmount:    inEntity.Amount,
			TargetAccountID: inEntity.AccountID,
			Date:            in.Date,
		}, nil
	default:
		return EmptyTransaction{}, ErrTransactionTypeNotFound
	}
}

func (store *Store) DeleteTransaction(ctx context.Context, Id uint, tenant string) error {
	err := store.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).
			Where("transaction_id = ?", Id).
			Delete(&dbEntry{}).Error; err != nil {
			return err
		}
		d := tx.WithContext(ctx).
			Where("id = ? AND owner_id = ?", Id, tenant).
			Delete(&dbTransaction{})
		if d.Error != nil {
			return d.Error
		}
		if d.RowsAffected == 0 {
			return ErrTransactionNotFound
		}
		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

type TransactionUpdate interface {
	isTransactionUpdate() // ensure only this package can implement the Transaction interface
}

type transactionUpdate struct{}

func (t transactionUpdate) isTransactionUpdate() {}

type EmptyTransactionUpdate struct {
	transactionUpdate
}

type IncomeUpdate struct {
	Description *string
	Amount      *float64
	AccountID   *uint
	Date        *time.Time

	transactionUpdate
}

type ExpenseUpdate struct {
	Description *string
	Amount      *float64
	AccountID   *uint
	Date        *time.Time

	transactionUpdate
}

type TransferUpdate struct {
	Description     *string
	OriginAmount    *float64
	OriginAccountID *uint
	TargetAmount    *float64
	TargetAccountID *uint
	Date            *time.Time

	transactionUpdate
}

// TODO: there is nothing preventing an income category to be tagged with an expense entry

func (store *Store) UpdateTransaction(ctx context.Context, input TransactionUpdate, Id uint, tenant string) error {
	switch input.(type) {
	case IncomeUpdate:
		return store.UpdateIncome(ctx, input.(IncomeUpdate), Id, tenant)

	case ExpenseUpdate:
		return nil

	case TransferUpdate:
		return nil
	default:
		return errors.New("invalid transaction type")
	}

}

func (store *Store) UpdateIncome(ctx context.Context, input IncomeUpdate, Id uint, tenant string) error {
	var selectedFields []string
	var updateStruct dbTransaction

	if input.Description != nil {
		if *input.Description == "" {
			return NewValidationErr("description cannot be empty")
		}
		updateStruct.Description = *input.Description
		selectedFields = append(selectedFields, "Description")
	}

	if input.Date != nil {
		if input.Date.IsZero() {
			return NewValidationErr("date cannot be zero")
		}
		updateStruct.Date = *input.Date
		selectedFields = append(selectedFields, "Date")
	}

	var updateEntity dbEntry
	var selectedEntryFields []string

	// these are entries
	if input.Amount != nil {
		if *input.Amount == 0 {
			return NewValidationErr("amount cannot be zero")
		}
		updateEntity.Amount = *input.Amount
		selectedEntryFields = append(selectedEntryFields, "Amount")
	}

	if input.AccountID != nil {
		if *input.AccountID == 0 {
			return NewValidationErr("account cannot be zero")
		}
		acc, err := store.GetAccount(ctx, *input.AccountID, tenant)
		if err != nil {
			return fmt.Errorf("error creating transaction: %w", err)
		}
		if acc.Type != Cash {
			return NewValidationErr("Incompatible account type for Income transaction")
		}

		updateEntity.AccountID = *input.AccountID
		selectedEntryFields = append(selectedEntryFields, "AccountID")
	}

	if len(selectedFields) == 0 && len(selectedEntryFields) == 0 {
		return ErrNoChanges
	}

	// Perform the update

	err := store.db.Transaction(func(tx *gorm.DB) error {
		// update the main transaction
		if len(selectedFields) > 0 {
			q := store.db.Model(&dbTransaction{}).
				Where("id = ? AND owner_id = ? AND type = ?", Id, tenant, incomeTransaction).
				Select(selectedFields).
				Updates(updateStruct)

			if q.Error != nil {
				return q.Error
			}
			if q.RowsAffected == 0 {
				return ErrTransactionNotFound
			}
		}

		if len(selectedEntryFields) > 0 {
			var entries []dbEntry
			q1 := store.db.Find(&entries)
			if q1.Error != nil {
				return q1.Error
			}

			q := store.db.Model(&dbEntry{}).
				Where("transaction_id = ? AND owner_id = ? AND entry_type = ?", Id, tenant, incomeEntry).
				Select(selectedEntryFields).
				Updates(updateEntity)

			if q.Error != nil {
				return q.Error
			}
			if q.RowsAffected == 0 {
				return ErrEntryNotFound
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error updating transaction: %w", err)
	}

	return nil
}

func (store *Store) UpdateExpense(ctx context.Context, input ExpenseUpdate, Id uint, tenant string) error {
	var selectedFields []string
	var updateStruct dbTransaction

	if input.Description != nil {
		if *input.Description == "" {
			return NewValidationErr("description cannot be empty")
		}
		updateStruct.Description = *input.Description
		selectedFields = append(selectedFields, "Description")
	}

	if input.Date != nil {
		if input.Date.IsZero() {
			return NewValidationErr("date cannot be zero")
		}
		updateStruct.Date = *input.Date
		selectedFields = append(selectedFields, "Date")
	}

	var updateEntity dbEntry
	var selectedEntryFields []string

	// these are entries
	if input.Amount != nil {
		if *input.Amount == 0 {
			return NewValidationErr("amount cannot be zero")
		}
		updateEntity.Amount = -*input.Amount
		selectedEntryFields = append(selectedEntryFields, "Amount")
	}

	if input.AccountID != nil {
		if *input.AccountID == 0 {
			return NewValidationErr("account cannot be zero")
		}
		acc, err := store.GetAccount(ctx, *input.AccountID, tenant)
		if err != nil {
			return fmt.Errorf("error creating transaction: %w", err)
		}
		if acc.Type != Cash {
			return NewValidationErr("Incompatible account type for Expense transaction")
		}

		updateEntity.AccountID = *input.AccountID
		selectedEntryFields = append(selectedEntryFields, "AccountID")
	}

	if len(selectedFields) == 0 && len(selectedEntryFields) == 0 {
		return ErrNoChanges
	}

	// Perform the update

	err := store.db.Transaction(func(tx *gorm.DB) error {
		// update the main transaction
		if len(selectedFields) > 0 {
			q := store.db.Model(&dbTransaction{}).
				Where("id = ? AND owner_id = ? AND type =  ?", Id, tenant, expenseTransaction).
				Select(selectedFields).
				Updates(updateStruct)

			if q.Error != nil {
				return q.Error
			}
			if q.RowsAffected == 0 {
				return ErrTransactionNotFound
			}
		}

		if len(selectedEntryFields) > 0 {
			var entries []dbEntry
			q1 := store.db.Find(&entries)
			if q1.Error != nil {
				return q1.Error
			}

			q := store.db.Model(&dbEntry{}).
				Where("transaction_id = ? AND owner_id = ? AND entry_type = ?", Id, tenant, expenseEntry).
				Select(selectedEntryFields).
				Updates(updateEntity)

			if q.Error != nil {
				return q.Error
			}
			if q.RowsAffected == 0 {
				return ErrEntryNotFound
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error updating transaction: %w", err)
	}

	return nil
}

func (store *Store) UpdateTransfer(ctx context.Context, input TransferUpdate, Id uint, tenant string) error {
	var selectedFields []string
	var updateStruct dbTransaction

	if input.Description != nil {
		if *input.Description == "" {
			return NewValidationErr("description cannot be empty")
		}
		updateStruct.Description = *input.Description
		selectedFields = append(selectedFields, "Description")
	}

	if input.Date != nil {
		if input.Date.IsZero() {
			return NewValidationErr("date cannot be zero")
		}
		updateStruct.Date = *input.Date
		selectedFields = append(selectedFields, "Date")
	}

	var targetEntry dbEntry
	var targetFields []string

	// these are entries
	if input.TargetAmount != nil {
		if *input.TargetAmount == 0 {
			return NewValidationErr("amount cannot be zero")
		}
		targetEntry.Amount = *input.TargetAmount
		targetFields = append(targetFields, "Amount")
	}

	if input.TargetAccountID != nil {
		if *input.TargetAccountID == 0 {
			return NewValidationErr("amount cannot be zero")
		}
		acc, err := store.GetAccount(ctx, *input.TargetAccountID, tenant)
		if err != nil {
			return fmt.Errorf("error creating transaction: %w", err)
		}
		if acc.Type != Cash {
			return NewValidationErr("Incompatible account type for Income transaction")
		}
		targetEntry.AccountID = *input.TargetAccountID
		targetFields = append(targetFields, "AccountID")
	}

	var originEntry dbEntry
	var originFields []string

	// these are entries
	if input.OriginAmount != nil {
		if *input.OriginAmount == 0 {
			return NewValidationErr("amount cannot be zero")
		}
		originEntry.Amount = -*input.OriginAmount
		originFields = append(originFields, "Amount")
	}

	if input.OriginAccountID != nil {
		if *input.OriginAccountID == 0 {
			return NewValidationErr("amount cannot be zero")
		}
		acc, err := store.GetAccount(ctx, *input.OriginAccountID, tenant)
		if err != nil {
			return fmt.Errorf("error creating transaction: %w", err)
		}
		if acc.Type != Cash {
			return NewValidationErr("Incompatible account type for Income transaction")
		}
		originEntry.AccountID = *input.OriginAccountID
		originFields = append(originFields, "AccountID")
	}

	if len(selectedFields) == 0 && len(targetFields) == 0 && len(originFields) == 0 {
		return ErrNoChanges
	}

	// Perform the update

	err := store.db.Transaction(func(tx *gorm.DB) error {
		// update the main transaction
		if len(selectedFields) > 0 {
			q := store.db.Model(&dbTransaction{}).
				Where("id = ? AND owner_id = ? AND type = ?", Id, tenant, transferTransaction).
				Select(selectedFields).
				Updates(updateStruct)

			if q.Error != nil {
				return q.Error
			}
			if q.RowsAffected == 0 {
				return ErrTransactionNotFound
			}
		}

		if len(targetFields) > 0 {
			q := store.db.Model(&dbEntry{}).
				Where("transaction_id = ? AND owner_id = ? AND entry_type = ?", Id, tenant, transferInEntry).
				Select(targetFields).
				Updates(targetEntry)
			if q.Error != nil {
				return q.Error
			}
			if q.RowsAffected == 0 {
				return ErrTransactionNotFound
			}
		}

		if len(originFields) > 0 {
			q := store.db.Model(&dbEntry{}).
				Where("transaction_id = ? AND owner_id = ? AND entry_type = ?", Id, tenant, transferOutEntry).
				Select(originFields).
				Updates(originEntry)
			if q.Error != nil {
				return q.Error
			}
			if q.RowsAffected == 0 {
				return ErrTransactionNotFound
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error updating transaction: %w", err)
	}

	return nil
}
