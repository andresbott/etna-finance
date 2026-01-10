package accounting

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"gorm.io/gorm"
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

// CreateTransaction creates a new transaction in the DB for a specific tenant.
// It delegates to the appropriate CreateX function depending on the input type.
func (store *Store) CreateTransaction(ctx context.Context, input Transaction, tenant string) (uint, error) {
	switch item := input.(type) {
	case Income:
		return store.CreateIncome(ctx, item, tenant)
	case Expense:
		return store.CreateExpense(ctx, item, tenant)
	case Transfer:
		return store.CreateTransfer(ctx, item, tenant)
	default:
		return 0, errors.New("invalid transaction type")
	}
}

func (store *Store) CreateIncome(ctx context.Context, item Income, tenant string) (uint, error) {
	if item.AccountID == 0 {
		return 0, ErrValidation("account ID is required")
	}

	acc, err := store.GetAccount(ctx, item.AccountID, tenant)
	if err != nil {
		return 0, fmt.Errorf("error creating income: %w", err)
	}
	allowedAccountTypes := []AccountType{
		CashAccountType, CheckinAccountType,
	}
	if !slices.Contains(allowedAccountTypes, acc.Type) {
		return 0, NewValidationErr(fmt.Sprintf("incompatible account type %s for income transaction", acc.Type.String()))
	}

	if item.CategoryID != 0 {
		cat, err := store.GetCategory(ctx, item.CategoryID, tenant)
		if err != nil {
			return 0, fmt.Errorf("error creating income: %w", err)
		}
		if cat.Type != IncomeCategory {
			return 0, NewValidationErr("incompatible category type for Income transaction")
		}
	}

	tx := dbTransaction{
		Description: item.Description,
		Date:        item.Date,
		OwnerId:     tenant,
		Type:        IncomeTransaction,
		Entries: []dbEntry{
			{
				AccountID:  item.AccountID,
				CategoryID: item.CategoryID,
				Amount:     item.Amount,
				EntryType:  incomeEntry,
				OwnerId:    tenant,
			},
		},
	}

	if err := validateTransaction(tx); err != nil {
		return 0, err
	}

	if err := store.db.WithContext(ctx).Create(&tx).Error; err != nil {
		return 0, err
	}
	return tx.Id, nil
}

func (store *Store) CreateExpense(ctx context.Context, item Expense, tenant string) (uint, error) {
	if item.AccountID == 0 {
		return 0, ErrValidation("account ID is required")
	}
	acc, err := store.GetAccount(ctx, item.AccountID, tenant)
	if err != nil {
		return 0, fmt.Errorf("error creating expense: %w", err)
	}
	allowedAccountTypes := []AccountType{
		CashAccountType, CheckinAccountType,
	}
	if !slices.Contains(allowedAccountTypes, acc.Type) {
		return 0, NewValidationErr(fmt.Sprintf("incompatible account type %s for expense transaction", acc.Type.String()))
	}

	if item.CategoryID != 0 {
		cat, err := store.GetCategory(ctx, item.CategoryID, tenant)
		if err != nil {
			return 0, fmt.Errorf("error creating expense: %w", err)
		}
		if cat.Type != ExpenseCategory {
			return 0, NewValidationErr("incompatible category type for Expense transaction")
		}
	}

	tx := dbTransaction{
		Description: item.Description,
		Date:        item.Date,
		OwnerId:     tenant,
		Type:        ExpenseTransaction,
		Entries: []dbEntry{
			{
				AccountID:  item.AccountID,
				CategoryID: item.CategoryID,
				Amount:     -item.Amount,
				EntryType:  expenseEntry,
				OwnerId:    tenant,
			},
		},
	}

	if err := validateTransaction(tx); err != nil {
		return 0, err
	}

	if err := store.db.WithContext(ctx).Create(&tx).Error; err != nil {
		return 0, err
	}
	return tx.Id, nil
}

func (store *Store) CreateTransfer(ctx context.Context, item Transfer, tenant string) (uint, error) {

	if item.OriginAccountID == 0 || item.TargetAccountID == 0 {
		return 0, ErrValidation("origin and target account IDs are required")
	}

	originAcc, err := store.GetAccount(ctx, item.OriginAccountID, tenant)
	if err != nil {
		return 0, fmt.Errorf("error creating transfer: %w", err)
	}

	allowedAccountTypes := []AccountType{
		CashAccountType, CheckinAccountType,
	}
	if !slices.Contains(allowedAccountTypes, originAcc.Type) {
		return 0, NewValidationErr(fmt.Sprintf("incompatible account type %s for transfer transaction", originAcc.Type.String()))
	}
	targetAcc, err := store.GetAccount(ctx, item.TargetAccountID, tenant)
	if err != nil {
		return 0, fmt.Errorf("error creating transfer: %w", err)
	}
	if !slices.Contains(allowedAccountTypes, targetAcc.Type) {
		return 0, NewValidationErr(fmt.Sprintf("incompatible account type %s for transfer transaction", targetAcc.Type.String()))
	}

	tx := dbTransaction{
		Description: item.Description,
		Date:        item.Date,
		OwnerId:     tenant,
		Type:        TransferTransaction,
		Entries: []dbEntry{
			{
				AccountID: item.OriginAccountID,
				Amount:    -item.OriginAmount,
				EntryType: transferOutEntry,
				OwnerId:   tenant,
			},
			{
				AccountID: item.TargetAccountID,
				Amount:    item.TargetAmount,
				EntryType: transferInEntry,
				OwnerId:   tenant,
			},
		},
	}

	if err := validateTransaction(tx); err != nil {
		return 0, err
	}

	if err := store.db.WithContext(ctx).Create(&tx).Error; err != nil {
		return 0, err
	}
	return tx.Id, nil
}

func validateTransaction(tx dbTransaction) error {
	if tx.Description == "" {
		return NewValidationErr("description cannot be empty")
	}
	if tx.Date.IsZero() {
		return NewValidationErr("date cannot be zero")
	}
	for _, entry := range tx.Entries {
		if entry.Amount == 0 {
			return NewValidationErr("amount cannot be zero")
		}
		if entry.AccountID == 0 {
			return NewValidationErr("account ID cannot be zero")
		}
	}
	return nil
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
			switch entry.EntryType {
			case transferInEntry:
				inEntity = entry
			case transferOutEntry:
				outEntry = entry
			default:
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
	switch item := input.(type) {
	case IncomeUpdate:
		return store.UpdateIncome(ctx, item, Id, tenant)
	case ExpenseUpdate:
		return store.UpdateExpense(ctx, item, Id, tenant)
	case TransferUpdate:
		return store.UpdateTransfer(ctx, item, Id, tenant)
	default:
		return errors.New("invalid baseTx type")
	}
}

func (store *Store) UpdateIncome(ctx context.Context, input IncomeUpdate, id uint, tenant string) error {
	params := updateIncomeExpenseParams{
		description:          input.Description,
		date:                 input.Date,
		amount:               input.Amount,
		accountID:            input.AccountID,
		categoryID:           input.CategoryID,
		amountMultiplier:     1,
		expectedCategoryType: IncomeCategory,
		txType:               IncomeTransaction,
		entryType:            incomeEntry,
	}
	return store.updateIncomeExpense(ctx, params, id, tenant)
}

func (store *Store) UpdateExpense(ctx context.Context, input ExpenseUpdate, id uint, tenant string) error {
	params := updateIncomeExpenseParams{
		description:          input.Description,
		date:                 input.Date,
		amount:               input.Amount,
		accountID:            input.AccountID,
		categoryID:           input.CategoryID,
		amountMultiplier:     -1,
		expectedCategoryType: ExpenseCategory,
		txType:               ExpenseTransaction,
		entryType:            expenseEntry,
	}
	return store.updateIncomeExpense(ctx, params, id, tenant)
}

type updateIncomeExpenseParams struct {
	description          *string
	date                 *time.Time
	amount               *float64
	accountID            *uint
	categoryID           *uint
	amountMultiplier     int
	expectedCategoryType CategoryType
	txType               TxType
	entryType            entryType
}

// updateIncomeExpense is a common function to update incomes and expenses
//
//nolint:nestif// the linter flags it but the code is simple enough to follow it without refactoring
func (store *Store) updateIncomeExpense(ctx context.Context, params updateIncomeExpenseParams, id uint, tenant string) error {
	var selectedFields []string
	var updateStruct dbTransaction
	var selectedEntryFields []string
	var updateEntry dbEntry

	// Description
	if params.description != nil {
		if *params.description == "" {
			return NewValidationErr("description cannot be empty")
		}
		updateStruct.Description = *params.description
		selectedFields = append(selectedFields, "Description")
	}

	// Date
	if params.date != nil {
		if params.date.IsZero() {
			return NewValidationErr("date cannot be zero")
		}
		updateStruct.Date = *params.date
		selectedFields = append(selectedFields, "Date")
	}

	// Amount
	if params.amount != nil {
		if *params.amount == 0 {
			return NewValidationErr("amount cannot be zero")
		}
		updateEntry.Amount = float64(params.amountMultiplier) * (*params.amount)
		selectedEntryFields = append(selectedEntryFields, "Amount")
	}

	// Account
	if params.accountID != nil {
		if *params.accountID == 0 {
			return NewValidationErr("account cannot be zero")
		}
		acc, err := store.GetAccount(ctx, *params.accountID, tenant)
		if err != nil {
			return fmt.Errorf("error updating transaction: %w", err)
		}

		allowedAccountTypes := []AccountType{
			CashAccountType, CheckinAccountType,
		}
		if !slices.Contains(allowedAccountTypes, acc.Type) {
			return NewValidationErr(fmt.Sprintf("incompatible account type '%s' for transaction", acc.Type.String()))
		}

		updateEntry.AccountID = *params.accountID
		selectedEntryFields = append(selectedEntryFields, "AccountID")
	}

	// Category
	if params.categoryID != nil {
		if *params.categoryID != 0 {
			cat, err := store.GetCategory(ctx, *params.categoryID, tenant)
			if err != nil {
				return fmt.Errorf("error updating transaction: %w", err)
			}
			if cat.Type != params.expectedCategoryType {
				return NewValidationErr("incompatible category type for transaction")
			}
		}
		updateEntry.CategoryID = *params.categoryID
		selectedEntryFields = append(selectedEntryFields, "CategoryID")
	}
	if len(selectedFields) == 0 && len(selectedEntryFields) == 0 {
		return ErrNoChanges
	}

	wParams := writeTxUpdateParams{
		selectedFields:   selectedFields,
		updateStruct:     updateStruct,
		entryType1Fields: selectedEntryFields,
		entryType1Values: updateEntry,
		txType:           params.txType,
		entryType1:       params.entryType,
	}

	if err := store.writeTxUpdate(wParams, id, tenant); err != nil {
		return fmt.Errorf("error updating transaction: %w", err)
	}
	return nil
}

type writeTxUpdateParams struct {
	// define the transaction
	selectedFields []string
	updateStruct   dbTransaction
	txType         TxType
	// define the first entry type
	entryType1Fields []string
	entryType1Values dbEntry
	entryType1       entryType
	// define the second entry type
	entryType2Fields []string
	entryType2Values dbEntry
	entryType2       entryType
}

func (store *Store) writeTxUpdate(params writeTxUpdateParams, id uint, tenant string) error {
	return store.db.Transaction(func(tx *gorm.DB) error {
		// Update the main transaction
		if len(params.selectedFields) > 0 {
			q := tx.Model(&dbTransaction{}).
				Where("id = ? AND owner_id = ? AND type = ?", id, tenant, params.txType).
				Select(params.selectedFields).
				Updates(params.updateStruct)

			if q.Error != nil {
				return q.Error
			}
			if q.RowsAffected == 0 {
				return ErrTransactionNotFound
			}
		}

		// Update fields of the first related entries
		if len(params.entryType1Fields) > 0 {
			q := tx.Model(&dbEntry{}).
				Where("transaction_id = ? AND owner_id = ? AND entry_type = ?", id, tenant, params.entryType1).
				Select(params.entryType1Fields).
				Updates(params.entryType1Values)
			if q.Error != nil {
				return q.Error
			}
			if q.RowsAffected == 0 {
				return ErrEntryNotFound
			}
		}
		// Update fields of the second related entries
		if len(params.entryType2Fields) > 0 {
			q := tx.Model(&dbEntry{}).
				Where("transaction_id = ? AND owner_id = ? AND entry_type = ?", id, tenant, params.entryType2).
				Select(params.entryType2Fields).
				Updates(params.entryType2Values)
			if q.Error != nil {
				return q.Error
			}
			if q.RowsAffected == 0 {
				return ErrEntryNotFound
			}
		}

		return nil
	})
}

//nolint:gocyclo// the linter flags it but the code is simply different input validations and payload generation
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

	allowedAccountTypes := []AccountType{
		CashAccountType, CheckinAccountType,
	}

	if input.TargetAccountID != nil {
		if *input.TargetAccountID == 0 {
			return NewValidationErr("amount cannot be zero")
		}
		acc, err := store.GetAccount(ctx, *input.TargetAccountID, tenant)
		if err != nil {
			return fmt.Errorf("error creating transaction: %w", err)
		}
		if !slices.Contains(allowedAccountTypes, acc.Type) {
			return NewValidationErr(fmt.Sprintf("incompatible account type '%s' for transaction", acc.Type.String()))
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
		if !slices.Contains(allowedAccountTypes, acc.Type) {
			return NewValidationErr(fmt.Sprintf("incompatible account type '%s' for transaction", acc.Type.String()))
		}
		originEntry.AccountID = *input.OriginAccountID
		originFields = append(originFields, "AccountID")
	}

	if len(selectedFields) == 0 && len(targetFields) == 0 && len(originFields) == 0 {
		return ErrNoChanges
	}

	wParams := writeTxUpdateParams{
		selectedFields: selectedFields,
		updateStruct:   updateStruct,
		txType:         TransferTransaction,

		entryType1Fields: targetFields,
		entryType1Values: targetEntry,
		entryType1:       transferInEntry,

		entryType2Fields: originFields,
		entryType2Values: originEntry,
		entryType2:       transferOutEntry,
	}
	if err := store.writeTxUpdate(wParams, Id, tenant); err != nil {
		return fmt.Errorf("error updating transaction: %w", err)
	}
	return nil
}

type ListOpts struct {
	StartDate time.Time
	EndDate   time.Time
	AccountId []int
	Types     []TxType
	Limit     int
	Page      int
}

const MaxSearchResults = 300
const DefaultSearchResults = 30

// ListTransactions returns an unsorted list of transactions matching the filter criteria
func (store *Store) ListTransactions(ctx context.Context, opts ListOpts, tenant string) ([]Transaction, error) {

	db := store.db.WithContext(ctx).Table("db_transactions")

	startDate := toDate(opts.StartDate)
	endDate := endOfDay(opts.EndDate)

	db = db.Select(`
        db_transactions.id AS transaction_id,
        db_transactions.date,
        db_transactions.description,
        db_transactions.type,
		db_entries.category_id,
		db_entries.account_id,

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
	db = db.Where("db_transactions.date BETWEEN ? AND ?", startDate, endDate)
	// filter by type
	if len(opts.Types) > 0 {
		db = db.Where("db_transactions.type IN (?)", opts.Types)
	}
	// filter by accounts
	if len(opts.AccountId) > 0 {
		db = db.Where("EXISTS (   SELECT 1  FROM db_entries AS e WHERE e.transaction_id = db_transactions.id"+
			" AND e.account_id IN (?)   )", opts.AccountId)
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

		CategoryId       uint
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
	//q := db.Scan(&debugtarget)
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
				CategoryID:  item.CategoryId,
				Date:        item.Date,
			}
			txs = append(txs, tx)
		case ExpenseTransaction:
			tx := Expense{
				Id:          item.TransactionId,
				Description: item.Description,
				Amount:      -item.ExpenseAmount,
				AccountID:   item.ExpenseAccountId,
				CategoryID:  item.CategoryId,
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
