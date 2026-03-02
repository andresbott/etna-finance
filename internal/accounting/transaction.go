package accounting

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/andresbott/etna/internal/marketdata"
	"golang.org/x/text/currency"
	"gorm.io/gorm"
)

type TxType int

const (
	UnknownTransaction TxType = iota
	IncomeTransaction
	ExpenseTransaction
	TransferTransaction
	StockBuyTransaction
	StockSellTransaction
	StockGrantTransaction // position increase without cash (vest, gift, grant, etc.)
	StockTransferTransaction
	LoanTransaction
)

type dbTransaction struct {
	Id          uint      `gorm:"primaryKey"`
	Date        time.Time `gorm:"not null"`
	Description string    `gorm:"size:255"`
	Type        TxType
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Entries     []dbEntry      `gorm:"foreignKey:TransactionID"` // One-to-many relationship
	Trades      []dbTrade      `gorm:"foreignKey:TransactionID"` // One-to-many for stock operations
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

// StockBuy represents a stock purchase.
// It creates two entries: one on the investment account (securities), one on the cash account (money out).
// Currencies may differ between accounts.
type StockBuy struct {
	Id                  uint
	Description         string
	Date                time.Time
	InvestmentAccountID uint // account of type Investment (position entry)
	CashAccountID       uint // account of type Cash/Checkin/Savings (money in that account's currency)
	InstrumentID        uint
	Quantity            float64
	TotalAmount         float64 // total cash spent (positive), in cash account currency
	StockAmount         float64 // monetary value of shares (positive), in investment account / instrument currency
	baseTx
}

// StockSell represents a stock sale.
// Creates 2–4 entries: position decrease at cost, capital repatriation, optional P&L, optional fees.
// CostBasis and RealizedGainLoss are computed from replay; Fees is user input.
type StockSell struct {
	Id                  uint
	Description         string
	Date                time.Time
	InvestmentAccountID uint
	CashAccountID       uint
	InstrumentID        uint
	Quantity            float64
	TotalAmount         float64  // gross proceeds (positive)
	Fees                float64  // sell-side fees (optional, default 0)
	CostBasis           float64  // allocated cost from replay (computed)
	RealizedGainLoss    float64  // P&L = totalAmount - costBasis - fees (computed)
	LotSelections       []LotSelection // nil/empty → FIFO; non-nil → manual allocation
	baseTx
}

// StockGrant represents a position increase without a cash leg (RSU vest, gift, award, etc.).
// Single entry on a position account (Investment or Grant).
// FairMarketValue is per-share FMV at grant/vest; used for cost basis when shares are sold or transferred.
type StockGrant struct {
	Id              uint
	Description     string
	Date            time.Time
	AccountID       uint // Investment or Unvested account that receives the shares
	InstrumentID    uint
	Quantity        float64
	FairMarketValue float64 // per-share FMV at grant/vest; 0 if omitted (cost basis = 0 for those shares)
	baseTx
}

// StockTransfer represents a transfer of shares between two position accounts (e.g. Unvested → Investment).
type StockTransfer struct {
	Id              uint
	Description     string
	Date            time.Time
	SourceAccountID uint // Investment or Unvested
	TargetAccountID uint // Investment or Unvested
	InstrumentID    uint
	Quantity        float64
	baseTx
}

// CreateTransaction creates a new transaction in the DB.
// It delegates to the appropriate CreateX function depending on the input type.
func (store *Store) CreateTransaction(ctx context.Context, input Transaction) (uint, error) {
	switch item := input.(type) {
	case Income:
		return store.CreateIncome(ctx, item)
	case Expense:
		return store.CreateExpense(ctx, item)
	case Transfer:
		return store.CreateTransfer(ctx, item)
	case StockBuy:
		return store.CreateStockBuy(ctx, item)
	case StockSell:
		return store.CreateStockSell(ctx, item)
	case StockGrant:
		return store.CreateStockGrant(ctx, item)
	case StockTransfer:
		return store.CreateStockTransfer(ctx, item)
	default:
		return 0, errors.New("invalid transaction type")
	}
}

func (store *Store) CreateIncome(ctx context.Context, item Income) (uint, error) {
	if item.AccountID == 0 {
		return 0, ErrValidation("account id is required")
	}

	acc, err := store.GetAccount(ctx, item.AccountID)
	if err != nil {
		return 0, fmt.Errorf("error creating income: %w", err)
	}
	allowedAccountTypes := []AccountType{
		CashAccountType, CheckinAccountType, SavingsAccountType,
	}
	if !slices.Contains(allowedAccountTypes, acc.Type) {
		return 0, NewValidationErr(fmt.Sprintf("incompatible account type %s for income transaction", acc.Type.String()))
	}

	if item.CategoryID != 0 {
		cat, err := store.GetCategory(ctx, item.CategoryID)
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
		Type:        IncomeTransaction,
		Entries: []dbEntry{
			{
				AccountID:  item.AccountID,
				CategoryID: item.CategoryID,
				Amount:     item.Amount,
				EntryType:  incomeEntry,
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

func (store *Store) CreateExpense(ctx context.Context, item Expense) (uint, error) {
	if item.AccountID == 0 {
		return 0, ErrValidation("account id is required")
	}
	acc, err := store.GetAccount(ctx, item.AccountID)
	if err != nil {
		return 0, fmt.Errorf("error creating expense: %w", err)
	}
	allowedAccountTypes := []AccountType{
		CashAccountType, CheckinAccountType, SavingsAccountType,
	}
	if !slices.Contains(allowedAccountTypes, acc.Type) {
		return 0, NewValidationErr(fmt.Sprintf("incompatible account type %s for expense transaction", acc.Type.String()))
	}

	if item.CategoryID != 0 {
		cat, err := store.GetCategory(ctx, item.CategoryID)
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
		Type:        ExpenseTransaction,
		Entries: []dbEntry{
			{
				AccountID:  item.AccountID,
				CategoryID: item.CategoryID,
				Amount:     -item.Amount,
				EntryType:  expenseEntry,
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

func (store *Store) CreateTransfer(ctx context.Context, item Transfer) (uint, error) {

	if item.OriginAccountID == 0 || item.TargetAccountID == 0 {
		return 0, ErrValidation("origin and target account IDs are required")
	}

	originAcc, err := store.GetAccount(ctx, item.OriginAccountID)
	if err != nil {
		return 0, fmt.Errorf("error creating transfer: %w", err)
	}

	allowedAccountTypes := []AccountType{
		CashAccountType, CheckinAccountType, SavingsAccountType,
	}
	if !slices.Contains(allowedAccountTypes, originAcc.Type) {
		return 0, NewValidationErr(fmt.Sprintf("incompatible account type %s for transfer transaction", originAcc.Type.String()))
	}
	targetAcc, err := store.GetAccount(ctx, item.TargetAccountID)
	if err != nil {
		return 0, fmt.Errorf("error creating transfer: %w", err)
	}
	if !slices.Contains(allowedAccountTypes, targetAcc.Type) {
		return 0, NewValidationErr(fmt.Sprintf("incompatible account type %s for transfer transaction", targetAcc.Type.String()))
	}

	tx := dbTransaction{
		Description: item.Description,
		Date:        item.Date,
		Type:        TransferTransaction,
		Entries: []dbEntry{
			{
				AccountID: item.OriginAccountID,
				Amount:    -item.OriginAmount,
				EntryType: transferOutEntry,
			},
			{
				AccountID: item.TargetAccountID,
				Amount:    item.TargetAmount,
				EntryType: transferInEntry,
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

var allowedCashAccountTypes = []AccountType{CashAccountType, CheckinAccountType, SavingsAccountType}
var allowedPositionAccountTypes = []AccountType{InvestmentAccountType, UnvestedAccountType}

// resolveInstrumentCurrencyFromTx fetches the instrument currency from the existing transaction.
func (store *Store) resolveInstrumentCurrencyFromTx(ctx context.Context, txID uint) (currency.Unit, error) {
	existing, err := store.GetTransaction(ctx, txID)
	if err != nil {
		return currency.Unit{}, fmt.Errorf("error validating currency match: %w", err)
	}
	var existingInstrumentID uint
	switch tx := existing.(type) {
	case StockBuy:
		existingInstrumentID = tx.InstrumentID
	case StockSell:
		existingInstrumentID = tx.InstrumentID
	}
	if existingInstrumentID == 0 {
		return currency.Unit{}, nil
	}
	inst, err := store.GetInstrument(ctx, existingInstrumentID)
	if err != nil {
		return currency.Unit{}, fmt.Errorf("error validating currency match: %w", err)
	}
	return inst.Currency, nil
}

// resolveAccountCurrencyFromTx fetches the investment account currency from the existing transaction.
func (store *Store) resolveAccountCurrencyFromTx(ctx context.Context, txID uint) (currency.Unit, error) {
	existing, err := store.GetTransaction(ctx, txID)
	if err != nil {
		return currency.Unit{}, fmt.Errorf("error validating currency match: %w", err)
	}
	var existingAccountID uint
	switch tx := existing.(type) {
	case StockBuy:
		existingAccountID = tx.InvestmentAccountID
	case StockSell:
		existingAccountID = tx.InvestmentAccountID
	}
	if existingAccountID == 0 {
		return currency.Unit{}, nil
	}
	acc, err := store.GetAccount(ctx, existingAccountID)
	if err != nil {
		return currency.Unit{}, fmt.Errorf("error validating currency match: %w", err)
	}
	return acc.Currency, nil
}

// validateStockCurrencyMatch checks that the instrument currency matches the investment account currency
// during partial updates. If either the account or instrument is being changed, it resolves the other
// from the existing transaction to perform the comparison.
func (store *Store) validateStockCurrencyMatch(
	ctx context.Context, txID uint,
	newAccountID *uint, newInstrumentID *uint,
	newInstrument *marketdata.Instrument, txType TxType,
) error {
	if newAccountID == nil && newInstrumentID == nil {
		return nil
	}

	var accCurrency, instCurrency currency.Unit

	if newAccountID != nil {
		acc, err := store.GetAccount(ctx, *newAccountID)
		if err != nil {
			return fmt.Errorf("error validating currency match: %w", err)
		}
		accCurrency = acc.Currency
	}

	if newInstrument != nil {
		instCurrency = newInstrument.Currency
	}

	// If only one changed, resolve the other from the existing transaction
	if newAccountID != nil && newInstrumentID == nil {
		var err error
		instCurrency, err = store.resolveInstrumentCurrencyFromTx(ctx, txID)
		if err != nil {
			return err
		}
	} else if newAccountID == nil && newInstrumentID != nil {
		var err error
		accCurrency, err = store.resolveAccountCurrencyFromTx(ctx, txID)
		if err != nil {
			return err
		}
	}

	if accCurrency != (currency.Unit{}) && instCurrency != (currency.Unit{}) && accCurrency != instCurrency {
		return NewValidationErr(fmt.Sprintf(
			"instrument currency %s does not match investment account currency %s",
			instCurrency, accCurrency))
	}
	return nil
}

func (store *Store) CreateStockBuy(ctx context.Context, item StockBuy) (uint, error) {
	if item.InvestmentAccountID == 0 {
		return 0, ErrValidation("investment account id is required")
	}
	if item.CashAccountID == 0 {
		return 0, ErrValidation("cash account id is required")
	}
	if item.InstrumentID == 0 {
		return 0, ErrValidation("instrument id is required")
	}
	if item.Quantity <= 0 {
		return 0, ErrValidation("quantity must be positive")
	}
	if item.TotalAmount <= 0 {
		return 0, ErrValidation("total amount must be positive")
	}
	if item.StockAmount <= 0 {
		return 0, ErrValidation("stock amount must be positive")
	}

	invAcc, err := store.GetAccount(ctx, item.InvestmentAccountID)
	if err != nil {
		return 0, fmt.Errorf("error creating stock buy: %w", err)
	}
	if !slices.Contains(allowedPositionAccountTypes, invAcc.Type) {
		return 0, NewValidationErr("investment account must be Investment or Unvested for stock buy")
	}

	cashAcc, err := store.GetAccount(ctx, item.CashAccountID)
	if err != nil {
		return 0, fmt.Errorf("error creating stock buy: %w", err)
	}
	if !slices.Contains(allowedCashAccountTypes, cashAcc.Type) {
		return 0, NewValidationErr("cash account must be Cash, Checkin or Savings for stock buy")
	}

	instrument, err := store.GetInstrument(ctx, item.InstrumentID)
	if err != nil {
		if errors.Is(err, marketdata.ErrInstrumentNotFound) {
			return 0, ErrValidation("instrument not found")
		}
		return 0, fmt.Errorf("error creating stock buy: %w", err)
	}

	if instrument.Currency != invAcc.Currency {
		return 0, NewValidationErr(fmt.Sprintf(
			"instrument currency %s does not match investment account currency %s",
			instrument.Currency, invAcc.Currency))
	}

	// Cash entry only — position is tracked via trades/lots
	tx := dbTransaction{
		Description: item.Description,
		Date:        item.Date,
		Type:        StockBuyTransaction,
		Entries: []dbEntry{
			{
				AccountID: item.CashAccountID,
				Amount:    -item.TotalAmount,
				EntryType: stockCashOutEntry,
			},
		},
	}

	if err := validateTransaction(tx); err != nil {
		return 0, err
	}

	var txID uint
	err = store.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		if err := dbTx.Create(&tx).Error; err != nil {
			return err
		}
		txID = tx.Id

		pricePerShare := 0.0
		if item.Quantity > 0 {
			pricePerShare = item.StockAmount / item.Quantity
		}
		trade := dbTrade{
			TransactionID: tx.Id,
			AccountID:     item.InvestmentAccountID,
			InstrumentID:  item.InstrumentID,
			TradeType:     BuyTrade,
			Quantity:      item.Quantity,
			PricePerShare: pricePerShare,
			TotalAmount:   item.StockAmount,
			Currency:      instrument.Currency.String(),
			Date:          item.Date,
		}
		_, err := store.createTrade(ctx, dbTx, trade)
		return err
	})
	if err != nil {
		return 0, err
	}
	return txID, nil
}

// Note: computeCostBasisForSell and ListStockPositionTransactions have been removed.
// Cost basis is now computed via FIFO lot allocation in lot.go.

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}

// validateStockSell validates all inputs for a stock sell and returns the resolved instrument and computed fees.
func (store *Store) validateStockSell(ctx context.Context, item StockSell) (marketdata.Instrument, float64, error) {
	if item.InvestmentAccountID == 0 {
		return marketdata.Instrument{}, 0, ErrValidation("investment account id is required")
	}
	if item.CashAccountID == 0 {
		return marketdata.Instrument{}, 0, ErrValidation("cash account id is required")
	}
	if item.InstrumentID == 0 {
		return marketdata.Instrument{}, 0, ErrValidation("instrument id is required")
	}
	if item.Quantity <= 0 {
		return marketdata.Instrument{}, 0, ErrValidation("quantity must be positive")
	}
	if item.TotalAmount <= 0 {
		return marketdata.Instrument{}, 0, ErrValidation("total amount must be positive")
	}
	fees := roundMoney(item.Fees)
	if fees < 0 {
		return marketdata.Instrument{}, 0, ErrValidation("fees cannot be negative")
	}

	invAcc, err := store.GetAccount(ctx, item.InvestmentAccountID)
	if err != nil {
		return marketdata.Instrument{}, 0, fmt.Errorf("error creating stock sell: %w", err)
	}
	if !slices.Contains(allowedPositionAccountTypes, invAcc.Type) {
		return marketdata.Instrument{}, 0, NewValidationErr("investment account must be Investment or Unvested for stock sell")
	}

	cashAcc, err := store.GetAccount(ctx, item.CashAccountID)
	if err != nil {
		return marketdata.Instrument{}, 0, fmt.Errorf("error creating stock sell: %w", err)
	}
	if !slices.Contains(allowedCashAccountTypes, cashAcc.Type) {
		return marketdata.Instrument{}, 0, NewValidationErr("cash account must be Cash, Checkin or Savings for stock sell")
	}

	instrument, err := store.GetInstrument(ctx, item.InstrumentID)
	if err != nil {
		if errors.Is(err, marketdata.ErrInstrumentNotFound) {
			return marketdata.Instrument{}, 0, ErrValidation("instrument not found")
		}
		return marketdata.Instrument{}, 0, fmt.Errorf("error creating stock sell: %w", err)
	}

	if instrument.Currency != invAcc.Currency {
		return marketdata.Instrument{}, 0, NewValidationErr(fmt.Sprintf(
			"instrument currency %s does not match investment account currency %s",
			instrument.Currency, invAcc.Currency))
	}
	if instrument.Currency != cashAcc.Currency {
		return marketdata.Instrument{}, 0, NewValidationErr(fmt.Sprintf(
			"instrument currency %s does not match cash account currency %s",
			instrument.Currency, cashAcc.Currency))
	}
	return instrument, fees, nil
}

func (store *Store) CreateStockSell(ctx context.Context, item StockSell) (uint, error) {
	instrument, fees, err := store.validateStockSell(ctx, item)
	if err != nil {
		return 0, err
	}

	var txID uint
	err = store.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		// Create the sell trade first (to get its ID for lot disposals)
		pricePerShare := 0.0
		if item.Quantity > 0 {
			pricePerShare = item.TotalAmount / item.Quantity
		}
		trade := dbTrade{
			AccountID:     item.InvestmentAccountID,
			InstrumentID:  item.InstrumentID,
			TradeType:     SellTrade,
			Quantity:      item.Quantity,
			PricePerShare: pricePerShare,
			TotalAmount:   item.TotalAmount,
			Currency:      instrument.Currency.String(),
			Date:          item.Date,
		}
		if err := dbTx.Create(&trade).Error; err != nil {
			return err
		}

		// Lot allocation: manual if selections provided, otherwise FIFO
		var allocations []LotAllocation
		var costBasis float64
		var lotErr error
		if len(item.LotSelections) > 0 {
			allocations, costBasis, lotErr = store.allocateLotsManual(ctx, dbTx, item.LotSelections, item.TotalAmount, item.Quantity, item.Date, trade.Id)
		} else {
			allocations, costBasis, lotErr = store.allocateLotsForSell(ctx, dbTx, item.InvestmentAccountID, item.InstrumentID, item.Quantity, item.TotalAmount, item.Date, trade.Id, FIFO)
		}
		if lotErr != nil {
			return lotErr
		}
		_ = allocations
		costBasis = roundMoney(costBasis)
		realizedGainLoss := roundMoney(item.TotalAmount - fees - costBasis)

		// Cash entries
		entries := []dbEntry{
			{
				AccountID: item.CashAccountID,
				Amount:    item.TotalAmount - fees,
				EntryType: stockCashInEntry,
			},
		}
		if realizedGainLoss > 0 {
			entries = append(entries, dbEntry{
				AccountID: item.CashAccountID,
				Amount:    realizedGainLoss,
				EntryType: incomeEntry,
			})
		} else if realizedGainLoss < 0 {
			entries = append(entries, dbEntry{
				AccountID: item.CashAccountID,
				Amount:    -realizedGainLoss,
				EntryType: expenseEntry,
			})
		}
		if fees > 0 {
			entries = append(entries, dbEntry{
				AccountID: item.CashAccountID,
				Amount:    fees,
				EntryType: expenseEntry,
			})
		}
		// Investment account entry: cost basis leaving the position
		entries = append(entries, dbEntry{
			AccountID: item.InvestmentAccountID,
			Amount:    -costBasis,
			EntryType: stockSellEntry,
		})

		tx := dbTransaction{
			Description: item.Description,
			Date:        item.Date,
			Type:        StockSellTransaction,
			Entries:     entries,
		}
		if err := validateTransaction(tx); err != nil {
			return err
		}
		if err := dbTx.Create(&tx).Error; err != nil {
			return err
		}
		txID = tx.Id

		// Update trade with transaction ID
		if err := dbTx.Model(&trade).Update("transaction_id", tx.Id).Error; err != nil {
			return err
		}

		// Update position
		return store.updatePosition(ctx, dbTx, item.InvestmentAccountID, item.InstrumentID)
	})
	if err != nil {
		return 0, err
	}
	return txID, nil
}

func (store *Store) CreateStockGrant(ctx context.Context, item StockGrant) (uint, error) {
	if item.AccountID == 0 {
		return 0, ErrValidation("account id is required")
	}
	if item.InstrumentID == 0 {
		return 0, ErrValidation("instrument id is required")
	}
	if item.Quantity <= 0 {
		return 0, ErrValidation("quantity must be positive")
	}

	acc, err := store.GetAccount(ctx, item.AccountID)
	if err != nil {
		return 0, fmt.Errorf("error creating stock grant: %w", err)
	}
	if !slices.Contains(allowedPositionAccountTypes, acc.Type) {
		return 0, NewValidationErr("account must be Investment or Unvested for stock grant")
	}

	instrument, err := store.GetInstrument(ctx, item.InstrumentID)
	if err != nil {
		if errors.Is(err, marketdata.ErrInstrumentNotFound) {
			return 0, ErrValidation("instrument not found")
		}
		return 0, fmt.Errorf("error creating stock grant: %w", err)
	}

	if instrument.Currency != acc.Currency {
		return 0, NewValidationErr(fmt.Sprintf(
			"instrument currency %s does not match account currency %s",
			instrument.Currency, acc.Currency))
	}

	grantCostBasis := item.FairMarketValue * item.Quantity
	if item.FairMarketValue < 0 {
		return 0, ErrValidation("fair market value cannot be negative")
	}

	// Grant: no cash movement, so no entries in db_entries. Only trade + lot + position.
	tx := dbTransaction{
		Description: item.Description,
		Date:        item.Date,
		Type:        StockGrantTransaction,
	}

	if tx.Description == "" {
		return 0, NewValidationErr("description cannot be empty")
	}
	if tx.Date.IsZero() {
		return 0, NewValidationErr("date cannot be zero")
	}

	var txID uint
	err = store.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		if err := dbTx.Create(&tx).Error; err != nil {
			return err
		}
		txID = tx.Id

		trade := dbTrade{
			TransactionID: tx.Id,
			AccountID:     item.AccountID,
			InstrumentID:  item.InstrumentID,
			TradeType:     GrantTrade,
			Quantity:      item.Quantity,
			PricePerShare: item.FairMarketValue,
			TotalAmount:   grantCostBasis,
			Currency:      instrument.Currency.String(),
			Date:          item.Date,
		}
		_, err := store.createTrade(ctx, dbTx, trade)
		return err
	})
	if err != nil {
		return 0, err
	}
	return txID, nil
}

// validateStockTransfer validates all inputs for a stock transfer.
func (store *Store) validateStockTransfer(ctx context.Context, item StockTransfer) error {
	if item.SourceAccountID == 0 || item.TargetAccountID == 0 {
		return ErrValidation("source and target account ids are required")
	}
	if item.SourceAccountID == item.TargetAccountID {
		return ErrValidation("source and target accounts must be different")
	}
	if item.InstrumentID == 0 {
		return ErrValidation("instrument id is required")
	}
	if item.Quantity <= 0 {
		return ErrValidation("quantity must be positive")
	}
	if item.Description == "" {
		return NewValidationErr("description cannot be empty")
	}
	if item.Date.IsZero() {
		return NewValidationErr("date cannot be zero")
	}

	srcAcc, err := store.GetAccount(ctx, item.SourceAccountID)
	if err != nil {
		return fmt.Errorf("error creating stock transfer: %w", err)
	}
	if !slices.Contains(allowedPositionAccountTypes, srcAcc.Type) {
		return NewValidationErr("source account must be Investment or Unvested for stock transfer")
	}

	tgtAcc, err := store.GetAccount(ctx, item.TargetAccountID)
	if err != nil {
		return fmt.Errorf("error creating stock transfer: %w", err)
	}
	if !slices.Contains(allowedPositionAccountTypes, tgtAcc.Type) {
		return NewValidationErr("target account must be Investment or Unvested for stock transfer")
	}

	instrument, err := store.GetInstrument(ctx, item.InstrumentID)
	if err != nil {
		if errors.Is(err, marketdata.ErrInstrumentNotFound) {
			return ErrValidation("instrument not found")
		}
		return fmt.Errorf("error creating stock transfer: %w", err)
	}

	if instrument.Currency != srcAcc.Currency {
		return NewValidationErr(fmt.Sprintf(
			"instrument currency %s does not match source account currency %s",
			instrument.Currency, srcAcc.Currency))
	}
	if instrument.Currency != tgtAcc.Currency {
		return NewValidationErr(fmt.Sprintf(
			"instrument currency %s does not match target account currency %s",
			instrument.Currency, tgtAcc.Currency))
	}
	return nil
}

func (store *Store) CreateStockTransfer(ctx context.Context, item StockTransfer) (uint, error) {
	if err := store.validateStockTransfer(ctx, item); err != nil {
		return 0, err
	}

	// Transfer: no cash movement, lot transfer logic + trade records for metadata.
	tx := dbTransaction{
		Description: item.Description,
		Date:        item.Date,
		Type:        StockTransferTransaction,
	}

	var txID uint
	err := store.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		if err := dbTx.Create(&tx).Error; err != nil {
			return err
		}
		txID = tx.Id

		// Create trade records for metadata (source out + target in)
		outTrade := dbTrade{
			TransactionID: tx.Id,
			AccountID:     item.SourceAccountID,
			InstrumentID:  item.InstrumentID,
			TradeType:     TransferOutTrade,
			Quantity:      item.Quantity,
			Date:          item.Date,
		}
		if err := dbTx.Create(&outTrade).Error; err != nil {
			return err
		}
		inTrade := dbTrade{
			TransactionID: tx.Id,
			AccountID:     item.TargetAccountID,
			InstrumentID:  item.InstrumentID,
			TradeType:     TransferInTrade,
			Quantity:      item.Quantity,
			Date:          item.Date,
		}
		if err := dbTx.Create(&inTrade).Error; err != nil {
			return err
		}

		// Transfer lots from source to target
		if err := store.transferLots(ctx, dbTx, item.SourceAccountID, item.TargetAccountID, item.InstrumentID, item.Quantity, item.Date, inTrade.Id); err != nil {
			return err
		}

		// Update positions for both accounts
		if err := store.updatePosition(ctx, dbTx, item.SourceAccountID, item.InstrumentID); err != nil {
			return err
		}
		return store.updatePosition(ctx, dbTx, item.TargetAccountID, item.InstrumentID)
	})
	if err != nil {
		return 0, err
	}
	return txID, nil
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
			return NewValidationErr("account id cannot be zero")
		}
	}
	return nil
}

var ErrTransactionNotFound = errors.New("transaction not found")
var ErrTransactionTypeNotFound = errors.New("transaction type not found")
var ErrEntryNotFound = errors.New("transaction entry not found")

// GetTransaction Returns a transaction after reading it from the DB
// Note that type assertion needs to be used to transform the Transaction into a specific type
func (store *Store) GetTransaction(ctx context.Context, Id uint) (Transaction, error) {
	var payload dbTransaction
	q := store.db.WithContext(ctx).Preload("Entries").Preload("Trades").Where("id = ?", Id).First(&payload)
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
		return incomeFromDb(in)
	case ExpenseTransaction:
		return expenseFromDb(in)
	case TransferTransaction:
		return transferFromDb(in)
	case StockBuyTransaction:
		return stockBuyFromDb(in)
	case StockSellTransaction:
		return stockSellFromDb(in)
	case StockGrantTransaction:
		return stockGrantFromDb(in)
	case StockTransferTransaction:
		return stockTransferFromDb(in)
	default:
		return EmptyTransaction{}, ErrTransactionTypeNotFound
	}
}

func incomeFromDb(in dbTransaction) (Transaction, error) {
	return Income{
		Description: in.Description,
		Date:        in.Date,
		Amount:      in.Entries[0].Amount,
		AccountID:   in.Entries[0].AccountID,
		CategoryID:  in.Entries[0].CategoryID,
	}, nil
}

func expenseFromDb(in dbTransaction) (Transaction, error) {
	return Expense{
		Description: in.Description,
		Date:        in.Date,
		Amount:      -in.Entries[0].Amount,
		AccountID:   in.Entries[0].AccountID,
		CategoryID:  in.Entries[0].CategoryID,
	}, nil
}

func transferFromDb(in dbTransaction) (Transaction, error) {
	var inEntity, outEntry dbEntry
	for _, entry := range in.Entries {
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
}

func stockBuyFromDb(in dbTransaction) (Transaction, error) {
	// Read trade data
	var trade *dbTrade
	for i := range in.Trades {
		if in.Trades[i].TradeType == BuyTrade {
			trade = &in.Trades[i]
			break
		}
	}

	// Find cash entry
	var cashEntry *dbEntry
	for i := range in.Entries {
		if in.Entries[i].EntryType == stockCashOutEntry {
			cashEntry = &in.Entries[i]
			break
		}
	}

	if trade == nil || cashEntry == nil {
		return nil, fmt.Errorf("stock buy transaction must have trade and cash entry")
	}
	return StockBuy{
		Id:                  in.Id,
		Description:         in.Description,
		Date:                in.Date,
		InvestmentAccountID: trade.AccountID,
		CashAccountID:       cashEntry.AccountID,
		InstrumentID:        trade.InstrumentID,
		Quantity:            trade.Quantity,
		TotalAmount:         -cashEntry.Amount,
		StockAmount:         trade.TotalAmount,
	}, nil
}

// deriveCostBasis computes lossAmount and fees from the expense entries of a stock sell transaction.
// Two expense entries: the larger is the realized loss, the smaller is fees.
// One expense entry: it is fees if there is also income (gain scenario), otherwise it is a loss.
func deriveCostBasis(expenseAmounts []float64, incomeAmount float64) (lossAmount, fees float64) {
	if len(expenseAmounts) == 2 {
		if expenseAmounts[0] >= expenseAmounts[1] {
			return expenseAmounts[0], expenseAmounts[1]
		}
		return expenseAmounts[1], expenseAmounts[0]
	}
	if len(expenseAmounts) == 1 {
		if incomeAmount > 0 {
			return 0, expenseAmounts[0]
		}
		return expenseAmounts[0], 0
	}
	return 0, 0
}

func stockSellFromDb(in dbTransaction) (Transaction, error) {
	// Read trade data
	var trade *dbTrade
	for i := range in.Trades {
		if in.Trades[i].TradeType == SellTrade {
			trade = &in.Trades[i]
			break
		}
	}

	// Find cash entries
	var cashEntry *dbEntry
	var incomeAmount float64
	var expenseAmounts []float64
	for i := range in.Entries {
		e := &in.Entries[i]
		switch e.EntryType {
		case stockCashInEntry:
			cashEntry = e
		case incomeEntry:
			incomeAmount += e.Amount
		case expenseEntry:
			expenseAmounts = append(expenseAmounts, e.Amount)
		}
	}

	if trade == nil {
		return nil, fmt.Errorf("stock sell transaction must have a sell trade")
	}

	// Derive cost basis from lot disposals or compute from entries
	totalAmount := trade.TotalAmount
	lossAmount, fees := deriveCostBasis(expenseAmounts, incomeAmount)
	realizedGainLoss := incomeAmount - lossAmount
	costBasis := roundMoney(totalAmount - fees - realizedGainLoss)

	cashAccountID := uint(0)
	if cashEntry != nil {
		cashAccountID = cashEntry.AccountID
	}

	return StockSell{
		Id:                  in.Id,
		Description:         in.Description,
		Date:                in.Date,
		InvestmentAccountID: trade.AccountID,
		CashAccountID:       cashAccountID,
		InstrumentID:        trade.InstrumentID,
		Quantity:            trade.Quantity,
		TotalAmount:         totalAmount,
		CostBasis:           costBasis,
		RealizedGainLoss:    realizedGainLoss,
		Fees:                fees,
	}, nil
}

func stockGrantFromDb(in dbTransaction) (Transaction, error) {
	var trade *dbTrade
	for i := range in.Trades {
		if in.Trades[i].TradeType == GrantTrade {
			trade = &in.Trades[i]
			break
		}
	}
	if trade == nil {
		return nil, fmt.Errorf("stock grant transaction must have a grant trade")
	}
	return StockGrant{
		Id:              in.Id,
		Description:     in.Description,
		Date:            in.Date,
		AccountID:       trade.AccountID,
		InstrumentID:    trade.InstrumentID,
		Quantity:        trade.Quantity,
		FairMarketValue: trade.PricePerShare,
	}, nil
}

func stockTransferFromDb(in dbTransaction) (Transaction, error) {
	var outTrade, inTrade *dbTrade
	for i := range in.Trades {
		t := &in.Trades[i]
		switch t.TradeType {
		case TransferOutTrade:
			outTrade = t
		case TransferInTrade:
			inTrade = t
		}
	}
	if outTrade == nil || inTrade == nil {
		return nil, fmt.Errorf("stock transfer transaction must have out and in trade records")
	}
	return StockTransfer{
		Id:              in.Id,
		Description:     in.Description,
		Date:            in.Date,
		SourceAccountID: outTrade.AccountID,
		TargetAccountID: inTrade.AccountID,
		InstrumentID:    outTrade.InstrumentID,
		Quantity:        outTrade.Quantity,
	}, nil
}

func (store *Store) DeleteTransaction(ctx context.Context, Id uint) error {
	return store.db.Transaction(func(tx *gorm.DB) error {
		// Delete trades (and cascading lots/disposals/positions) for stock transactions
		if err := store.deleteTradesByTransactionID(ctx, tx, Id); err != nil {
			return err
		}

		// Delete entries
		if err := tx.WithContext(ctx).
			Where("transaction_id = ?", Id).
			Delete(&dbEntry{}).Error; err != nil {
			return err
		}

		// Delete transaction
		d := tx.WithContext(ctx).
			Where("id = ?", Id).
			Delete(&dbTransaction{})
		if d.Error != nil {
			return d.Error
		}
		if d.RowsAffected == 0 {
			return ErrTransactionNotFound
		}
		return nil
	})
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

type StockBuyUpdate struct {
	Description         *string
	Date                *time.Time
	InstrumentID        *uint
	Quantity            *float64
	TotalAmount         *float64
	StockAmount         *float64
	InvestmentAccountID *uint
	CashAccountID       *uint

	txUpdate
}

type StockSellUpdate struct {
	Description         *string
	Date                *time.Time
	InstrumentID        *uint
	Quantity            *float64
	TotalAmount         *float64
	Fees                *float64
	InvestmentAccountID *uint
	CashAccountID       *uint
	LotSelections       []LotSelection // nil/empty → FIFO; non-nil → manual allocation

	txUpdate
}

type StockGrantUpdate struct {
	Description     *string
	Date            *time.Time
	InstrumentID    *uint
	Quantity        *float64
	AccountID       *uint
	FairMarketValue *float64

	txUpdate
}

type StockTransferUpdate struct {
	Description     *string
	Date            *time.Time
	InstrumentID    *uint
	Quantity        *float64
	SourceAccountID *uint
	TargetAccountID *uint

	txUpdate
}

// TODO: there is nothing preventing an income category to be tagged with an expense entry

func (store *Store) UpdateTransaction(ctx context.Context, input TransactionUpdate, Id uint) error {
	switch item := input.(type) {
	case IncomeUpdate:
		return store.UpdateIncome(ctx, item, Id)
	case ExpenseUpdate:
		return store.UpdateExpense(ctx, item, Id)
	case TransferUpdate:
		return store.UpdateTransfer(ctx, item, Id)
	case StockBuyUpdate:
		return store.UpdateStockBuy(ctx, item, Id)
	case StockSellUpdate:
		return store.UpdateStockSell(ctx, item, Id)
	case StockGrantUpdate:
		return store.UpdateStockGrant(ctx, item, Id)
	case StockTransferUpdate:
		return store.UpdateStockTransfer(ctx, item, Id)
	default:
		return errors.New("invalid baseTx type")
	}
}

func (store *Store) UpdateIncome(ctx context.Context, input IncomeUpdate, id uint) error {
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
	return store.updateIncomeExpense(ctx, params, id)
}

func (store *Store) UpdateExpense(ctx context.Context, input ExpenseUpdate, id uint) error {
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
	return store.updateIncomeExpense(ctx, params, id)
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
func (store *Store) updateIncomeExpense(ctx context.Context, params updateIncomeExpenseParams, id uint) error {
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
		acc, err := store.GetAccount(ctx, *params.accountID)
		if err != nil {
			return fmt.Errorf("error updating transaction: %w", err)
		}

		allowedAccountTypes := []AccountType{
			CashAccountType, CheckinAccountType, SavingsAccountType,
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
			cat, err := store.GetCategory(ctx, *params.categoryID)
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

	if err := store.writeTxUpdate(wParams, id); err != nil {
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

func (store *Store) writeTxUpdate(params writeTxUpdateParams, id uint) error {
	return store.db.Transaction(func(tx *gorm.DB) error {
		// Update the main transaction
		if len(params.selectedFields) > 0 {
			q := tx.Model(&dbTransaction{}).
				Where("id = ? AND type = ?", id, params.txType).
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
				Where("transaction_id = ? AND entry_type = ?", id, params.entryType1).
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
				Where("transaction_id = ? AND entry_type = ?", id, params.entryType2).
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
func (store *Store) UpdateTransfer(ctx context.Context, input TransferUpdate, Id uint) error {
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
		CashAccountType, CheckinAccountType, SavingsAccountType,
	}

	if input.TargetAccountID != nil {
		if *input.TargetAccountID == 0 {
			return NewValidationErr("amount cannot be zero")
		}
		acc, err := store.GetAccount(ctx, *input.TargetAccountID)
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
		acc, err := store.GetAccount(ctx, *input.OriginAccountID)
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
	if err := store.writeTxUpdate(wParams, Id); err != nil {
		return fmt.Errorf("error updating transaction: %w", err)
	}
	return nil
}

//nolint:gocyclo // field merge applies individual nil-checks and validation for each optional field
func (store *Store) mergeStockBuyFields(ctx context.Context, buy *StockBuy, input StockBuyUpdate) error {
	if input.Description != nil {
		if *input.Description == "" {
			return NewValidationErr("description cannot be empty")
		}
		buy.Description = *input.Description
	}
	if input.Date != nil {
		if input.Date.IsZero() {
			return NewValidationErr("date cannot be zero")
		}
		buy.Date = *input.Date
	}
	if input.InvestmentAccountID != nil {
		if *input.InvestmentAccountID == 0 {
			return NewValidationErr("investment account is required")
		}
		acc, err := store.GetAccount(ctx, *input.InvestmentAccountID)
		if err != nil {
			return fmt.Errorf("error updating stock buy: %w", err)
		}
		if !slices.Contains(allowedPositionAccountTypes, acc.Type) {
			return NewValidationErr("investment account must be Investment or Unvested")
		}
		buy.InvestmentAccountID = *input.InvestmentAccountID
	}
	if input.InstrumentID != nil {
		if *input.InstrumentID == 0 {
			return NewValidationErr("instrument is required")
		}
		if _, err := store.GetInstrument(ctx, *input.InstrumentID); err != nil {
			if errors.Is(err, marketdata.ErrInstrumentNotFound) {
				return ErrValidation("instrument not found")
			}
			return fmt.Errorf("error updating stock buy: %w", err)
		}
		buy.InstrumentID = *input.InstrumentID
	}
	if input.CashAccountID != nil {
		if *input.CashAccountID == 0 {
			return NewValidationErr("cash account is required")
		}
		acc, err := store.GetAccount(ctx, *input.CashAccountID)
		if err != nil {
			return fmt.Errorf("error updating stock buy: %w", err)
		}
		if !slices.Contains(allowedCashAccountTypes, acc.Type) {
			return NewValidationErr("cash account must be Cash, Checkin or Savings")
		}
		buy.CashAccountID = *input.CashAccountID
	}
	if input.Quantity != nil {
		if *input.Quantity <= 0 {
			return NewValidationErr("quantity must be positive")
		}
		buy.Quantity = *input.Quantity
	}
	if input.TotalAmount != nil {
		if *input.TotalAmount <= 0 {
			return NewValidationErr("total amount must be positive")
		}
		buy.TotalAmount = *input.TotalAmount
	}
	if input.StockAmount != nil {
		if *input.StockAmount <= 0 {
			return NewValidationErr("stock amount must be positive")
		}
		buy.StockAmount = *input.StockAmount
	}
	return nil
}

func (store *Store) UpdateStockBuy(ctx context.Context, input StockBuyUpdate, id uint) error {
	existing, err := store.GetTransaction(ctx, id)
	if err != nil {
		return err
	}
	buy, ok := existing.(StockBuy)
	if !ok {
		return ErrTransactionNotFound
	}

	if err := store.mergeStockBuyFields(ctx, &buy, input); err != nil {
		return err
	}

	if err := store.validateStockCurrencyMatch(ctx, id, &buy.InvestmentAccountID, &buy.InstrumentID, nil, StockBuyTransaction); err != nil {
		return err
	}

	// Delete and recreate: delete old trades/lots/entries, recreate everything
	return store.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		if err := store.deleteTradesByTransactionID(ctx, dbTx, id); err != nil {
			return err
		}
		if err := dbTx.Where("transaction_id = ?", id).Delete(&dbEntry{}).Error; err != nil {
			return err
		}

		// Update transaction fields
		if err := dbTx.Model(&dbTransaction{}).Where("id = ?", id).Updates(map[string]interface{}{
			"description": buy.Description,
			"date":        buy.Date,
		}).Error; err != nil {
			return err
		}

		// Recreate cash entry
		cashEntry := dbEntry{
			TransactionID: id,
			AccountID:     buy.CashAccountID,
			Amount:        -buy.TotalAmount,
			EntryType:     stockCashOutEntry,
		}
		if err := dbTx.Create(&cashEntry).Error; err != nil {
			return err
		}

		// Recreate trade + lot + position
		pricePerShare := 0.0
		if buy.Quantity > 0 {
			pricePerShare = buy.StockAmount / buy.Quantity
		}
		trade := dbTrade{
			TransactionID: id,
			AccountID:     buy.InvestmentAccountID,
			InstrumentID:  buy.InstrumentID,
			TradeType:     BuyTrade,
			Quantity:      buy.Quantity,
			PricePerShare: pricePerShare,
			TotalAmount:   buy.StockAmount,
			Date:          buy.Date,
		}
		_, err := store.createTrade(ctx, dbTx, trade)
		return err
	})
}

//nolint:gocyclo // field merge applies individual nil-checks and validation for each optional field
func (store *Store) mergeStockSellFields(ctx context.Context, sell *StockSell, input StockSellUpdate) error {
	if input.Description != nil {
		if *input.Description == "" {
			return NewValidationErr("description cannot be empty")
		}
		sell.Description = *input.Description
	}
	if input.Date != nil {
		if input.Date.IsZero() {
			return NewValidationErr("date cannot be zero")
		}
		sell.Date = *input.Date
	}
	if input.InvestmentAccountID != nil {
		if *input.InvestmentAccountID == 0 {
			return NewValidationErr("investment account is required")
		}
		acc, err := store.GetAccount(ctx, *input.InvestmentAccountID)
		if err != nil {
			return fmt.Errorf("error updating stock sell: %w", err)
		}
		if !slices.Contains(allowedPositionAccountTypes, acc.Type) {
			return NewValidationErr("investment account must be Investment or Unvested")
		}
		sell.InvestmentAccountID = *input.InvestmentAccountID
	}
	if input.CashAccountID != nil {
		if *input.CashAccountID == 0 {
			return NewValidationErr("cash account is required")
		}
		acc, err := store.GetAccount(ctx, *input.CashAccountID)
		if err != nil {
			return fmt.Errorf("error updating stock sell: %w", err)
		}
		if !slices.Contains(allowedCashAccountTypes, acc.Type) {
			return NewValidationErr("cash account must be Cash, Checkin or Savings")
		}
		sell.CashAccountID = *input.CashAccountID
	}
	if input.InstrumentID != nil {
		if *input.InstrumentID == 0 {
			return NewValidationErr("instrument is required")
		}
		if _, err := store.GetInstrument(ctx, *input.InstrumentID); err != nil {
			if errors.Is(err, marketdata.ErrInstrumentNotFound) {
				return ErrValidation("instrument not found")
			}
			return fmt.Errorf("error updating stock sell: %w", err)
		}
		sell.InstrumentID = *input.InstrumentID
	}
	if input.Quantity != nil {
		if *input.Quantity <= 0 {
			return NewValidationErr("quantity must be positive")
		}
		sell.Quantity = *input.Quantity
	}
	if input.TotalAmount != nil {
		if *input.TotalAmount <= 0 {
			return NewValidationErr("total amount must be positive")
		}
		sell.TotalAmount = *input.TotalAmount
	}
	if input.Fees != nil {
		if *input.Fees < 0 {
			return NewValidationErr("fees cannot be negative")
		}
		sell.Fees = *input.Fees
	}
	// Propagate manual lot selections (nil → keep FIFO)
	sell.LotSelections = input.LotSelections
	return nil
}

func (store *Store) UpdateStockSell(ctx context.Context, input StockSellUpdate, id uint) error {
	existing, err := store.GetTransaction(ctx, id)
	if err != nil {
		return err
	}
	sell, ok := existing.(StockSell)
	if !ok {
		return ErrTransactionNotFound
	}

	if err := store.mergeStockSellFields(ctx, &sell, input); err != nil {
		return err
	}

	if err := store.validateStockCurrencyMatch(ctx, id, &sell.InvestmentAccountID, &sell.InstrumentID, nil, StockSellTransaction); err != nil {
		return err
	}

	// Delete and recreate: delete old trades/lots/entries, then recreate
	return store.recreateStockSell(ctx, id, sell)
}

func (store *Store) recreateStockSell(ctx context.Context, id uint, sell StockSell) error {
	return store.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		// Delete old trades/lots/disposals
		if err := store.deleteTradesByTransactionID(ctx, dbTx, id); err != nil {
			return err
		}
		// Delete old entries
		if err := dbTx.Where("transaction_id = ?", id).Delete(&dbEntry{}).Error; err != nil {
			return err
		}

		// Update transaction fields
		if err := dbTx.Model(&dbTransaction{}).Where("id = ?", id).Updates(map[string]interface{}{
			"description": sell.Description,
			"date":        sell.Date,
		}).Error; err != nil {
			return err
		}

		// Create sell trade
		pricePerShare := 0.0
		if sell.Quantity > 0 {
			pricePerShare = sell.TotalAmount / sell.Quantity
		}
		trade := dbTrade{
			TransactionID: id,
			AccountID:     sell.InvestmentAccountID,
			InstrumentID:  sell.InstrumentID,
			TradeType:     SellTrade,
			Quantity:      sell.Quantity,
			PricePerShare: pricePerShare,
			TotalAmount:   sell.TotalAmount,
			Date:          sell.Date,
		}
		if err := dbTx.Create(&trade).Error; err != nil {
			return err
		}

		// Lot allocation: manual if selections provided, otherwise FIFO
		var costBasis float64
		var lotErr error
		if len(sell.LotSelections) > 0 {
			_, costBasis, lotErr = store.allocateLotsManual(ctx, dbTx, sell.LotSelections, sell.TotalAmount, sell.Quantity, sell.Date, trade.Id)
		} else {
			_, costBasis, lotErr = store.allocateLotsForSell(ctx, dbTx, sell.InvestmentAccountID, sell.InstrumentID, sell.Quantity, sell.TotalAmount, sell.Date, trade.Id, FIFO)
		}
		if lotErr != nil {
			return lotErr
		}
		costBasis = roundMoney(costBasis)
		fees := roundMoney(sell.Fees)
		realizedGainLoss := roundMoney(sell.TotalAmount - fees - costBasis)

		// Cash entries
		entries := []dbEntry{
			{TransactionID: id, AccountID: sell.CashAccountID, Amount: sell.TotalAmount - fees, EntryType: stockCashInEntry},
		}
		if realizedGainLoss > 0 {
			entries = append(entries, dbEntry{TransactionID: id, AccountID: sell.CashAccountID, Amount: realizedGainLoss, EntryType: incomeEntry})
		} else if realizedGainLoss < 0 {
			entries = append(entries, dbEntry{TransactionID: id, AccountID: sell.CashAccountID, Amount: -realizedGainLoss, EntryType: expenseEntry})
		}
		if fees > 0 {
			entries = append(entries, dbEntry{TransactionID: id, AccountID: sell.CashAccountID, Amount: fees, EntryType: expenseEntry})
		}
		// Investment account entry: cost basis leaving the position
		entries = append(entries, dbEntry{TransactionID: id, AccountID: sell.InvestmentAccountID, Amount: -costBasis, EntryType: stockSellEntry})
		if err := dbTx.Create(&entries).Error; err != nil {
			return err
		}

		// Update position
		return store.updatePosition(ctx, dbTx, sell.InvestmentAccountID, sell.InstrumentID)
	})
}

func (store *Store) mergeStockGrantFields(ctx context.Context, grant *StockGrant, input StockGrantUpdate) error {
	if input.Description != nil {
		if *input.Description == "" {
			return NewValidationErr("description cannot be empty")
		}
		grant.Description = *input.Description
	}
	if input.Date != nil {
		if input.Date.IsZero() {
			return NewValidationErr("date cannot be zero")
		}
		grant.Date = *input.Date
	}
	if input.AccountID != nil {
		if *input.AccountID == 0 {
			return NewValidationErr("account is required")
		}
		acc, err := store.GetAccount(ctx, *input.AccountID)
		if err != nil {
			return fmt.Errorf("error updating stock grant: %w", err)
		}
		if !slices.Contains(allowedPositionAccountTypes, acc.Type) {
			return NewValidationErr("account must be Investment or Unvested")
		}
		grant.AccountID = *input.AccountID
	}
	if input.InstrumentID != nil {
		if *input.InstrumentID == 0 {
			return NewValidationErr("instrument is required")
		}
		if _, err := store.GetInstrument(ctx, *input.InstrumentID); err != nil {
			if errors.Is(err, marketdata.ErrInstrumentNotFound) {
				return ErrValidation("instrument not found")
			}
			return fmt.Errorf("error updating stock grant: %w", err)
		}
		grant.InstrumentID = *input.InstrumentID
	}
	if input.Quantity != nil {
		if *input.Quantity <= 0 {
			return NewValidationErr("quantity must be positive")
		}
		grant.Quantity = *input.Quantity
	}
	if input.FairMarketValue != nil {
		if *input.FairMarketValue < 0 {
			return NewValidationErr("fair market value cannot be negative")
		}
		grant.FairMarketValue = *input.FairMarketValue
	}
	return nil
}

func (store *Store) UpdateStockGrant(ctx context.Context, input StockGrantUpdate, id uint) error {
	existing, err := store.GetTransaction(ctx, id)
	if err != nil {
		return err
	}
	grant, ok := existing.(StockGrant)
	if !ok {
		return ErrTransactionNotFound
	}

	if err := store.mergeStockGrantFields(ctx, &grant, input); err != nil {
		return err
	}

	// Delete and recreate trades/lots
	return store.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		if err := store.deleteTradesByTransactionID(ctx, dbTx, id); err != nil {
			return err
		}

		// Update transaction fields
		if err := dbTx.Model(&dbTransaction{}).Where("id = ?", id).Updates(map[string]interface{}{
			"description": grant.Description,
			"date":        grant.Date,
		}).Error; err != nil {
			return err
		}

		// Recreate trade + lot + position
		grantCostBasis := grant.FairMarketValue * grant.Quantity
		trade := dbTrade{
			TransactionID: id,
			AccountID:     grant.AccountID,
			InstrumentID:  grant.InstrumentID,
			TradeType:     GrantTrade,
			Quantity:      grant.Quantity,
			PricePerShare: grant.FairMarketValue,
			TotalAmount:   grantCostBasis,
			Date:          grant.Date,
		}
		_, err := store.createTrade(ctx, dbTx, trade)
		return err
	})
}

func (store *Store) mergeStockTransferFields(ctx context.Context, transfer *StockTransfer, input StockTransferUpdate) error {
	if input.Description != nil {
		if *input.Description == "" {
			return NewValidationErr("description cannot be empty")
		}
		transfer.Description = *input.Description
	}
	if input.Date != nil {
		if input.Date.IsZero() {
			return NewValidationErr("date cannot be zero")
		}
		transfer.Date = *input.Date
	}
	if input.SourceAccountID != nil {
		if *input.SourceAccountID == 0 {
			return NewValidationErr("source account is required")
		}
		acc, err := store.GetAccount(ctx, *input.SourceAccountID)
		if err != nil {
			return fmt.Errorf("error updating stock transfer: %w", err)
		}
		if !slices.Contains(allowedPositionAccountTypes, acc.Type) {
			return NewValidationErr("source account must be Investment or Unvested")
		}
		transfer.SourceAccountID = *input.SourceAccountID
	}
	if input.TargetAccountID != nil {
		if *input.TargetAccountID == 0 {
			return NewValidationErr("target account is required")
		}
		acc, err := store.GetAccount(ctx, *input.TargetAccountID)
		if err != nil {
			return fmt.Errorf("error updating stock transfer: %w", err)
		}
		if !slices.Contains(allowedPositionAccountTypes, acc.Type) {
			return NewValidationErr("target account must be Investment or Unvested")
		}
		transfer.TargetAccountID = *input.TargetAccountID
	}
	if input.InstrumentID != nil {
		if *input.InstrumentID == 0 {
			return NewValidationErr("instrument is required")
		}
		if _, err := store.GetInstrument(ctx, *input.InstrumentID); err != nil {
			if errors.Is(err, marketdata.ErrInstrumentNotFound) {
				return ErrValidation("instrument not found")
			}
			return fmt.Errorf("error updating stock transfer: %w", err)
		}
		transfer.InstrumentID = *input.InstrumentID
	}
	if input.Quantity != nil {
		if *input.Quantity <= 0 {
			return NewValidationErr("quantity must be positive")
		}
		transfer.Quantity = *input.Quantity
	}
	return nil
}

func (store *Store) UpdateStockTransfer(ctx context.Context, input StockTransferUpdate, id uint) error {
	existing, err := store.GetTransaction(ctx, id)
	if err != nil {
		return err
	}
	transfer, ok := existing.(StockTransfer)
	if !ok {
		return ErrTransactionNotFound
	}

	if err := store.mergeStockTransferFields(ctx, &transfer, input); err != nil {
		return err
	}

	// Delete and recreate
	return store.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		if err := store.deleteTradesByTransactionID(ctx, dbTx, id); err != nil {
			return err
		}

		// Update transaction fields
		if err := dbTx.Model(&dbTransaction{}).Where("id = ?", id).Updates(map[string]interface{}{
			"description": transfer.Description,
			"date":        transfer.Date,
		}).Error; err != nil {
			return err
		}

		// Recreate trade records
		outTrade := dbTrade{
			TransactionID: id,
			AccountID:     transfer.SourceAccountID,
			InstrumentID:  transfer.InstrumentID,
			TradeType:     TransferOutTrade,
			Quantity:      transfer.Quantity,
			Date:          transfer.Date,
		}
		if err := dbTx.Create(&outTrade).Error; err != nil {
			return err
		}
		inTrade := dbTrade{
			TransactionID: id,
			AccountID:     transfer.TargetAccountID,
			InstrumentID:  transfer.InstrumentID,
			TradeType:     TransferInTrade,
			Quantity:      transfer.Quantity,
			Date:          transfer.Date,
		}
		if err := dbTx.Create(&inTrade).Error; err != nil {
			return err
		}

		// Transfer lots
		if err := store.transferLots(ctx, dbTx, transfer.SourceAccountID, transfer.TargetAccountID, transfer.InstrumentID, transfer.Quantity, transfer.Date, inTrade.Id); err != nil {
			return err
		}

		// Update positions
		if err := store.updatePosition(ctx, dbTx, transfer.SourceAccountID, transfer.InstrumentID); err != nil {
			return err
		}
		return store.updatePosition(ctx, dbTx, transfer.TargetAccountID, transfer.InstrumentID)
	})
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
func (store *Store) ListTransactions(ctx context.Context, opts ListOpts) ([]Transaction, error) {

	db := store.db.WithContext(ctx).Table("db_transactions")

	startDate := toDate(opts.StartDate)
	endDate := endOfDay(opts.EndDate)

	db = db.Select(`
        db_transactions.id AS transaction_id,
        db_transactions.date,
        db_transactions.description,
        db_transactions.type,
		COALESCE(MAX(db_entries.category_id), 0) AS category_id,
		COALESCE(MAX(db_entries.account_id), 0) AS account_id,

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
        CAST(SUM(CASE WHEN db_entries.entry_type = 3 THEN db_entries.amount ELSE 0 END) AS REAL) AS target_amount,

        -- stock cash leg (out=7, in=8)
        CAST(MAX(CASE WHEN db_entries.entry_type IN (7, 8) THEN db_entries.account_id END) AS INTEGER) AS stock_cash_account_id,
        CAST(MAX(CASE WHEN db_entries.entry_type IN (7, 8) THEN db_entries.amount END) AS REAL) AS stock_cash_amount,

        -- stock buy/sell/grant from trades
        CAST(MAX(CASE WHEN db_trades.trade_type = 1 THEN db_trades.account_id END) AS INTEGER) AS trade_buy_account_id,
        CAST(MAX(CASE WHEN db_trades.trade_type = 1 THEN db_trades.instrument_id END) AS INTEGER) AS trade_buy_instrument_id,
        CAST(MAX(CASE WHEN db_trades.trade_type = 1 THEN db_trades.quantity END) AS REAL) AS trade_buy_quantity,
        CAST(MAX(CASE WHEN db_trades.trade_type = 1 THEN db_trades.total_amount END) AS REAL) AS trade_buy_amount,
        CAST(MAX(CASE WHEN db_trades.trade_type = 2 THEN db_trades.account_id END) AS INTEGER) AS trade_sell_account_id,
        CAST(MAX(CASE WHEN db_trades.trade_type = 2 THEN db_trades.instrument_id END) AS INTEGER) AS trade_sell_instrument_id,
        CAST(MAX(CASE WHEN db_trades.trade_type = 2 THEN db_trades.quantity END) AS REAL) AS trade_sell_quantity,
        CAST(MAX(CASE WHEN db_trades.trade_type = 2 THEN db_trades.total_amount END) AS REAL) AS trade_sell_amount,
        CAST(MAX(CASE WHEN db_trades.trade_type = 3 THEN db_trades.account_id END) AS INTEGER) AS trade_grant_account_id,
        CAST(MAX(CASE WHEN db_trades.trade_type = 3 THEN db_trades.instrument_id END) AS INTEGER) AS trade_grant_instrument_id,
        CAST(MAX(CASE WHEN db_trades.trade_type = 3 THEN db_trades.quantity END) AS REAL) AS trade_grant_quantity,
        CAST(MAX(CASE WHEN db_trades.trade_type = 3 THEN db_trades.price_per_share END) AS REAL) AS trade_grant_fmv,
        CAST(MAX(CASE WHEN db_trades.trade_type = 4 THEN db_trades.account_id END) AS INTEGER) AS trade_transfer_source_id,
        CAST(MAX(CASE WHEN db_trades.trade_type = 5 THEN db_trades.account_id END) AS INTEGER) AS trade_transfer_target_id,
        CAST(MAX(CASE WHEN db_trades.trade_type IN (4, 5) THEN db_trades.instrument_id END) AS INTEGER) AS trade_transfer_instrument_id,
        CAST(MAX(CASE WHEN db_trades.trade_type IN (4, 5) THEN db_trades.quantity END) AS REAL) AS trade_transfer_quantity
    `).
		Joins("LEFT JOIN db_entries ON db_entries.transaction_id = db_transactions.id").
		Joins("LEFT JOIN db_trades ON db_trades.transaction_id = db_transactions.id")

	// ensure proper owner
	// Filter by date range
	db = db.Where("db_transactions.date BETWEEN ? AND ?", startDate, endDate)
	// filter by type
	if len(opts.Types) > 0 {
		db = db.Where("db_transactions.type IN (?)", opts.Types)
	}
	// filter by accounts (check both entries and trades)
	if len(opts.AccountId) > 0 {
		db = db.Where(
			"(EXISTS (SELECT 1 FROM db_entries AS e WHERE e.transaction_id = db_transactions.id AND e.account_id IN (?))"+
				" OR EXISTS (SELECT 1 FROM db_trades AS t WHERE t.transaction_id = db_transactions.id AND t.account_id IN (?)))",
			opts.AccountId, opts.AccountId)
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

	db = db.Order("db_transactions.date DESC").Limit(opts.Limit).Offset(offset)

	//debugtarget := []map[string]any{} // left for debugging
	type intermediate struct {
		Date          time.Time
		Description   string
		Type          TxType
		TransactionId uint

		CategoryId       uint
		AccountId        uint
		IncomeAccountId  uint
		IncomeAmount     float64
		ExpenseAccountId uint
		ExpenseAmount    float64
		OriginAccountId  uint
		OriginAmount     float64
		TargetAccountId  uint
		TargetAmount     float64

		StockCashAccountId uint
		StockCashAmount    float64

		// From trades
		TradeBuyAccountId      uint
		TradeBuyInstrumentId   uint
		TradeBuyQuantity       float64
		TradeBuyAmount         float64
		TradeSellAccountId     uint
		TradeSellInstrumentId  uint
		TradeSellQuantity      float64
		TradeSellAmount        float64
		TradeGrantAccountId    uint
		TradeGrantInstrumentId uint
		TradeGrantQuantity     float64
		TradeGrantFmv          float64

		TradeTransferSourceId     uint
		TradeTransferTargetId     uint
		TradeTransferInstrumentId uint
		TradeTransferQuantity     float64
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
		case StockBuyTransaction:
			totalAmount := item.StockCashAmount
			if totalAmount < 0 {
				totalAmount = -totalAmount
			}
			txs = append(txs, StockBuy{
				Id:                  item.TransactionId,
				Description:         item.Description,
				Date:                item.Date,
				InvestmentAccountID: item.TradeBuyAccountId,
				CashAccountID:       item.StockCashAccountId,
				InstrumentID:        item.TradeBuyInstrumentId,
				Quantity:            item.TradeBuyQuantity,
				TotalAmount:         totalAmount,
				StockAmount:         item.TradeBuyAmount,
			})
		case StockSellTransaction:
			realizedGainLoss := item.IncomeAmount - item.ExpenseAmount
			costBasis := roundMoney(item.TradeSellAmount - roundMoney(item.IncomeAmount-item.ExpenseAmount))
			// Infer fees: expense entries can include both loss and fees
			txs = append(txs, StockSell{
				Id:                  item.TransactionId,
				Description:         item.Description,
				Date:                item.Date,
				InvestmentAccountID: item.TradeSellAccountId,
				CashAccountID:       item.StockCashAccountId,
				InstrumentID:        item.TradeSellInstrumentId,
				Quantity:            item.TradeSellQuantity,
				TotalAmount:         item.TradeSellAmount,
				CostBasis:           costBasis,
				RealizedGainLoss:    realizedGainLoss,
			})
		case StockGrantTransaction:
			txs = append(txs, StockGrant{
				Id:              item.TransactionId,
				Description:     item.Description,
				Date:            item.Date,
				AccountID:       item.TradeGrantAccountId,
				InstrumentID:    item.TradeGrantInstrumentId,
				Quantity:        item.TradeGrantQuantity,
				FairMarketValue: item.TradeGrantFmv,
			})
		case StockTransferTransaction:
			txs = append(txs, StockTransfer{
				Id:              item.TransactionId,
				Description:     item.Description,
				Date:            item.Date,
				SourceAccountID: item.TradeTransferSourceId,
				TargetAccountID: item.TradeTransferTargetId,
				InstrumentID:    item.TradeTransferInstrumentId,
				Quantity:        item.TradeTransferQuantity,
			})
		default:
			tx := EmptyTransaction{}
			txs = append(txs, tx)
		}
	}
	return txs, nil
}
