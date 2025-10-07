package accounting

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"time"
)

type TxType int

const (
	UnknownTransaction TxType = iota
	IncomeTransaction
	ExpenseTransaction
	TransferTransaction
	StockTransaction
	LoanTransaction
)

type dbTransaction struct {
	Id          uint      `gorm:"primaryKey"`
	Date        time.Time `gorm:"not null"`
	Description string    `gorm:"size:255"`
	Type        TxType
	OwnerId     string `gorm:"index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Entries     []dbEntry      `gorm:"foreignKey:TransactionID"` // One-to-many relationship
}

type Transaction interface {
	isTransaction() // ensure only this package can implement the Transaction interface
}

type baseTx struct{}

func (t baseTx) isTransaction() {}

type EmptyTransaction struct {
	baseTx
}

type Income struct {
	Id          uint
	Description string
	Amount      float64
	AccountID   uint
	CategoryID  uint
	Date        time.Time

	baseTx
}

type Expense struct {
	Id          uint
	Description string
	Amount      float64
	AccountID   uint
	CategoryID  uint
	Date        time.Time

	baseTx
}

type Transfer struct {
	Id              uint
	Description     string
	OriginAmount    float64
	OriginAccountID uint
	TargetAmount    float64
	TargetAccountID uint
	Date            time.Time

	baseTx
}

// CreateTransaction Allows to create a new baseTx in the DB for a specific tenant
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
			return 0, NewValidationErr("incompatible account type for Income transaction")
		}

		if item.CategoryID != 0 {
			cat, err := store.GetCategory(ctx, item.CategoryID, tenant)
			if err != nil {
				return 0, fmt.Errorf("error creating transaction: %w", err)
			}
			if cat.Type != IncomeCategory {
				return 0, NewValidationErr("incompatible category type for Income transaction")
			}
		}

		tx = dbTransaction{
			Description: item.Description,
			Date:        item.Date,
			OwnerId:     tenant,
			Type:        IncomeTransaction,
			Entries: []dbEntry{
				{AccountID: item.AccountID, CategoryID: item.CategoryID, Amount: item.Amount,
					EntryType: incomeEntry, OwnerId: tenant},
			},
		}
	case Expense:
		item := input.(Expense)
		acc, err := store.GetAccount(ctx, item.AccountID, tenant)
		if err != nil {
			return 0, fmt.Errorf("error creating transaction: %w", err)
		}
		if acc.Type != Cash {
			return 0, NewValidationErr("Incompatible account type for expense transaction")
		}

		if item.CategoryID != 0 {
			cat, err := store.GetCategory(ctx, item.CategoryID, tenant)
			if err != nil {
				return 0, fmt.Errorf("error creating transaction: %w", err)
			}
			if cat.Type != ExpenseCategory {
				return 0, NewValidationErr("incompatible category type for Expense transaction")
			}
		}

		tx = dbTransaction{
			Description: item.Description,
			Date:        item.Date,
			OwnerId:     tenant,
			Type:        ExpenseTransaction,
			Entries: []dbEntry{
				{AccountID: item.AccountID, CategoryID: item.CategoryID, Amount: -item.Amount,
					EntryType: expenseEntry, OwnerId: tenant},
			},
		}

	case Transfer:
		item := input.(Transfer)
		acc, err := store.GetAccount(ctx, item.OriginAccountID, tenant)
		if err != nil {
			return 0, fmt.Errorf("error creating transaction: %w", err)
		}
		if acc.Type != Cash {
			return 0, NewValidationErr("Incompatible account type for Transfer transaction")
		}

		acc, err = store.GetAccount(ctx, item.TargetAccountID, tenant)
		if err != nil {
			return 0, fmt.Errorf("error creating transaction: %w", err)
		}
		if acc.Type != Cash {
			return 0, NewValidationErr("Incompatible account type for Transfer transaction")
		}

		tx = dbTransaction{
			Description: item.Description,
			Date:        item.Date,
			OwnerId:     tenant,
			Type:        TransferTransaction,
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
	case IncomeTransaction:
		return Income{
			Description: in.Description,
			Date:        in.Date,
			Amount:      in.Entries[0].Amount,
			AccountID:   in.Entries[0].AccountID,
			CategoryID:  in.Entries[0].CategoryID,
		}, nil
	case ExpenseTransaction:
		return Expense{
			Description: in.Description,
			Date:        in.Date,
			Amount:      -in.Entries[0].Amount,
			AccountID:   in.Entries[0].AccountID,
			CategoryID:  in.Entries[0].CategoryID,
		}, nil
	case TransferTransaction:
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
	isTxUpdate() // ensure only this package can implement the Transaction interface
}

type txUpdate struct{}

func (t txUpdate) isTxUpdate() {}

type EmptyTransactionUpdate struct {
	txUpdate
}

type IncomeUpdate struct {
	Description *string
	Amount      *float64
	AccountID   *uint
	CategoryID  *uint
	Date        *time.Time

	txUpdate
}

type ExpenseUpdate struct {
	Description *string
	Amount      *float64
	AccountID   *uint
	CategoryID  *uint
	Date        *time.Time

	txUpdate
}

type TransferUpdate struct {
	Description     *string
	OriginAmount    *float64
	OriginAccountID *uint
	TargetAmount    *float64
	TargetAccountID *uint
	Date            *time.Time

	txUpdate
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
		return errors.New("invalid baseTx type")
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
			return fmt.Errorf("error updating income: %w", err)
		}
		if acc.Type != Cash {
			return NewValidationErr("incompatible account type for Income transaction")
		}

		updateEntity.AccountID = *input.AccountID
		selectedEntryFields = append(selectedEntryFields, "AccountID")
	}

	if input.CategoryID != nil {
		if *input.CategoryID != 0 {
			cat, err := store.GetCategory(ctx, *input.CategoryID, tenant)
			if err != nil {
				return fmt.Errorf("error updating income: %w", err)
			}
			if cat.Type != IncomeCategory {
				return NewValidationErr("incompatible category type for Income transaction")
			}
		}
		updateEntity.CategoryID = *input.CategoryID
		selectedEntryFields = append(selectedEntryFields, "CategoryID")
	}

	if len(selectedFields) == 0 && len(selectedEntryFields) == 0 {
		return ErrNoChanges
	}

	// Perform the update

	err := store.db.Transaction(func(tx *gorm.DB) error {
		// update the main baseTx
		if len(selectedFields) > 0 {
			q := store.db.Model(&dbTransaction{}).
				Where("id = ? AND owner_id = ? AND type = ?", Id, tenant, IncomeTransaction).
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
			return fmt.Errorf("error creating baseTx: %w", err)
		}
		if acc.Type != Cash {
			return NewValidationErr("incompatible account type for Expense transaction")
		}

		updateEntity.AccountID = *input.AccountID
		selectedEntryFields = append(selectedEntryFields, "AccountID")
	}

	if input.CategoryID != nil {
		if *input.CategoryID != 0 {
			cat, err := store.GetCategory(ctx, *input.CategoryID, tenant)
			if err != nil {
				return fmt.Errorf("error updating income: %w", err)
			}
			if cat.Type != ExpenseCategory {
				return NewValidationErr("incompatible category type for Income transaction")
			}
		}

		updateEntity.CategoryID = *input.CategoryID
		selectedEntryFields = append(selectedEntryFields, "CategoryID")
	}

	if len(selectedFields) == 0 && len(selectedEntryFields) == 0 {
		return ErrNoChanges
	}

	// Perform the update

	err := store.db.Transaction(func(tx *gorm.DB) error {
		// update the main baseTx
		if len(selectedFields) > 0 {
			q := store.db.Model(&dbTransaction{}).
				Where("id = ? AND owner_id = ? AND type =  ?", Id, tenant, ExpenseTransaction).
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
			return NewValidationErr("incompatible account type for Transfer transaction")
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
			return NewValidationErr("incompatible account type for Transfer transaction")
		}
		originEntry.AccountID = *input.OriginAccountID
		originFields = append(originFields, "AccountID")
	}

	if len(selectedFields) == 0 && len(targetFields) == 0 && len(originFields) == 0 {
		return ErrNoChanges
	}

	// Perform the update

	err := store.db.Transaction(func(tx *gorm.DB) error {
		// update the main baseTx
		if len(selectedFields) > 0 {
			q := store.db.Model(&dbTransaction{}).
				Where("id = ? AND owner_id = ? AND type = ?", Id, tenant, TransferTransaction).
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

type ListOpts struct {
	StartDate time.Time
	EndDate   time.Time
	//AccountIds []int
	//categoryIds []int
	Types []TxType
	Limit int
	Page  int
}

const MaxSearchResults = 90
const DefaultSearchResults = 30

// ListTransactions returns an unsorted list of transactions matching the filter criteria
func (store *Store) ListTransactions(ctx context.Context, opts ListOpts, tenant string) ([]Transaction, error) {

	// code sample using preload, left in case of debugging
	//var payload dbTransaction
	//q1 := store.db.WithContext(ctx).Preload("Entries").Where("id = ? AND owner_id = ?", 1, tenant).First(&payload)
	//if q1.Error != nil {
	//	if errors.Is(q1.Error, gorm.ErrRecordNotFound) {
	//		return nil, ErrTransactionNotFound
	//	} else {
	//		return nil, q1.Error
	//	}
	//}
	//spew.Dump(payload)

	// =====

	db := store.db.WithContext(ctx).Table("db_transactions")

	db = db.Select(`
        db_transactions.id AS transaction_id,
        db_transactions.date,
        db_transactions.description,
        db_transactions.type,

        -- income
        CAST(MAX(CASE WHEN db_entries.entry_type = 1 THEN db_entries.account_id END) AS INTEGER) AS income_account_id,
        CAST(SUM(CASE WHEN db_entries.entry_type = 1 THEN db_entries.amount ELSE 0 END) AS REAL) AS income_amount,

        -- expense
        CAST(MAX(CASE WHEN db_entries.entry_type = 2 THEN db_entries.account_id END) AS INTEGER) AS expense_account_id,
        CAST(SUM(CASE WHEN db_entries.entry_type = 2 THEN db_entries.amount ELSE 0 END) AS REAL) AS expense_amount,

        -- transfer (out)
        CAST(MAX(CASE WHEN db_entries.entry_type = 4 THEN db_entries.account_id END) AS INTEGER) AS origin_account_id,
        CAST(SUM(CASE WHEN db_entries.entry_type = 4 THEN db_entries.amount ELSE 0 END) AS REAL) AS origin_amount,

        -- transfer (in)
        CAST(MAX(CASE WHEN db_entries.entry_type = 3 THEN db_entries.account_id END) AS INTEGER) AS target_account_id,
        CAST(SUM(CASE WHEN db_entries.entry_type = 3 THEN db_entries.amount ELSE 0 END) AS REAL) AS target_amount
    `).Joins("JOIN db_entries ON db_entries.transaction_id = db_transactions.id")

	// ensure proper owner
	db = db.Where("db_entries.owner_id = ? AND db_transactions.owner_id = ? ", tenant, tenant)
	// Filter by date range
	db = db.Where("db_transactions.date BETWEEN ? AND ?", opts.StartDate, opts.EndDate)
	if len(opts.Types) > 0 {
		db = db.Where("db_transactions.type IN (?)", opts.Types)
	}

	db = db.Group("db_transactions.id, db_transactions.date, db_transactions.description, db_transactions.type")

	if opts.Limit == 0 {
		opts.Limit = DefaultSearchResults
	}
	if opts.Limit > MaxSearchResults {
		opts.Limit = MaxSearchResults
	}

	if opts.Page < 1 {
		opts.Page = 1
	}
	offset := (opts.Page - 1) * opts.Limit

	db = db.Order("date DESC").Limit(opts.Limit).Offset(offset)

	//debugtarget := []map[string]any{} // left for debugging
	type intermediate struct {
		Date          time.Time
		Description   string
		Type          TxType
		TransactionId uint

		IncomeAccountId  uint
		IncomeAmount     float64
		ExpenseAccountId uint
		ExpenseAmount    float64
		OriginAccountId  uint
		OriginAmount     float64
		TargetAccountId  uint
		TargetAmount     float64
	}

	var target []intermediate
	q := db.Scan(&target)
	if q.Error != nil {
		if errors.Is(q.Error, gorm.ErrRecordNotFound) {
			return nil, ErrTransactionNotFound
		} else {
			return nil, q.Error
		}
	}
	//spew.Dump(debugtarget)
	//return nil, nil

	var txs []Transaction
	for _, item := range target {
		switch item.Type {
		case IncomeTransaction:

			tx := Income{
				Id:          item.TransactionId,
				Description: item.Description,
				Amount:      item.IncomeAmount,
				AccountID:   item.IncomeAccountId,
				Date:        item.Date,
			}
			txs = append(txs, tx)
		case ExpenseTransaction:
			tx := Expense{
				Id:          item.TransactionId,
				Description: item.Description,
				Amount:      -item.ExpenseAmount,
				AccountID:   item.ExpenseAccountId,
				Date:        item.Date,
			}
			txs = append(txs, tx)
		case TransferTransaction:
			tx := Transfer{
				Id:              item.TransactionId,
				Description:     item.Description,
				Date:            item.Date,
				OriginAmount:    -item.OriginAmount,
				OriginAccountID: item.OriginAccountId,
				TargetAmount:    item.TargetAmount,
				TargetAccountID: item.TargetAccountId,
			}
			txs = append(txs, tx)
		default:
			tx := EmptyTransaction{}
			txs = append(txs, tx)
		}
	}
	return txs, nil
}
