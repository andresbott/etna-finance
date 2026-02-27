package accounting

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type TradeType int

const (
	BuyTrade         TradeType = 1
	SellTrade        TradeType = 2
	GrantTrade       TradeType = 3
	TransferOutTrade TradeType = 4
	TransferInTrade  TradeType = 5
)

// dbTrade records a single stock operation (buy, sell, or grant).
// Fees are tracked as expense entries in db_entries (same transaction_id).
// FX rate is not stored — derivable from the data.
type dbTrade struct {
	Id            uint      `gorm:"primaryKey"`
	TransactionID uint      `gorm:"not null;index"`
	AccountID     uint      `gorm:"not null;index"`
	InstrumentID  uint      `gorm:"not null;index"`
	TradeType     TradeType `gorm:"not null"`
	Quantity      float64   `gorm:"not null"`
	PricePerShare float64
	TotalAmount   float64
	Currency      string
	Date          time.Time `gorm:"not null;index"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Trade struct {
	Id            uint
	TransactionID uint
	AccountID     uint
	InstrumentID  uint
	TradeType     TradeType
	Quantity      float64
	PricePerShare float64
	TotalAmount   float64
	Currency      string
	Date          time.Time
}

type ListTradesOpts struct {
	AccountID    uint
	InstrumentID uint
	StartDate    time.Time
	EndDate      time.Time
}

func tradeFromDb(t dbTrade) Trade {
	return Trade{
		Id:            t.Id,
		TransactionID: t.TransactionID,
		AccountID:     t.AccountID,
		InstrumentID:  t.InstrumentID,
		TradeType:     t.TradeType,
		Quantity:      t.Quantity,
		PricePerShare: t.PricePerShare,
		TotalAmount:   t.TotalAmount,
		Currency:      t.Currency,
		Date:          t.Date,
	}
}

func (store *Store) createTrade(ctx context.Context, tx *gorm.DB, trade dbTrade) (uint, error) {
	if err := tx.WithContext(ctx).Create(&trade).Error; err != nil {
		return 0, fmt.Errorf("failed to create trade: %w", err)
	}

	// For buy/grant, create a lot
	if trade.TradeType == BuyTrade || trade.TradeType == GrantTrade {
		costPerShare := 0.0
		if trade.Quantity > 0 {
			costPerShare = trade.TotalAmount / trade.Quantity
		}
		lot := dbLot{
			TradeID:      trade.Id,
			AccountID:    trade.AccountID,
			InstrumentID: trade.InstrumentID,
			OpenDate:     trade.Date,
			Quantity:     trade.Quantity,
			OriginalQty:  trade.Quantity,
			CostPerShare: costPerShare,
			CostBasis:    trade.TotalAmount,
			Status:       LotOpen,
		}
		if err := tx.WithContext(ctx).Create(&lot).Error; err != nil {
			return 0, fmt.Errorf("failed to create lot for trade: %w", err)
		}
	}

	// For sell, allocate lots via FIFO
	if trade.TradeType == SellTrade {
		_, _, err := store.allocateLotsForSell(ctx, tx, trade.AccountID, trade.InstrumentID, trade.Quantity, trade.TotalAmount, trade.Date, trade.Id, FIFO)
		if err != nil {
			return 0, fmt.Errorf("failed to allocate lots for sell: %w", err)
		}
	}

	// Update position
	if err := store.updatePosition(ctx, tx, trade.AccountID, trade.InstrumentID); err != nil {
		return 0, fmt.Errorf("failed to update position: %w", err)
	}

	return trade.Id, nil
}

func (store *Store) GetTrade(ctx context.Context, id uint) (Trade, error) {
	var trade dbTrade
	if err := store.db.WithContext(ctx).Where("id = ?", id).First(&trade).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Trade{}, fmt.Errorf("trade not found")
		}
		return Trade{}, err
	}
	return tradeFromDb(trade), nil
}

func (store *Store) ListTrades(ctx context.Context, opts ListTradesOpts) ([]Trade, error) {
	db := store.db.WithContext(ctx).Table("db_trades")

	if opts.AccountID != 0 {
		db = db.Where("account_id = ?", opts.AccountID)
	}
	if opts.InstrumentID != 0 {
		db = db.Where("instrument_id = ?", opts.InstrumentID)
	}
	if !opts.StartDate.IsZero() {
		db = db.Where("date >= ?", opts.StartDate)
	}
	if !opts.EndDate.IsZero() {
		db = db.Where("date <= ?", endOfDay(opts.EndDate))
	}

	db = db.Order("date ASC")

	var trades []dbTrade
	if err := db.Find(&trades).Error; err != nil {
		return nil, err
	}

	result := make([]Trade, len(trades))
	for i, t := range trades {
		result[i] = tradeFromDb(t)
	}
	return result, nil
}

// deleteTrade removes a trade and its associated lots/disposals, then updates the position.
func (store *Store) deleteTrade(ctx context.Context, tx *gorm.DB, tradeID uint) error {
	var trade dbTrade
	if err := tx.WithContext(ctx).Where("id = ?", tradeID).First(&trade).Error; err != nil {
		return err
	}

	// Delete lot disposals referencing lots of this trade, or referencing this trade as sell
	if err := tx.WithContext(ctx).Where("sell_trade_id = ?", tradeID).Delete(&dbLotDisposal{}).Error; err != nil {
		return err
	}
	// Delete lot disposals from lots owned by this trade
	if err := tx.WithContext(ctx).
		Where("lot_id IN (?)", tx.Model(&dbLot{}).Select("id").Where("trade_id = ?", tradeID)).
		Delete(&dbLotDisposal{}).Error; err != nil {
		return err
	}

	// Delete lots created by this trade
	if err := tx.WithContext(ctx).Where("trade_id = ?", tradeID).Delete(&dbLot{}).Error; err != nil {
		return err
	}

	// Delete the trade
	if err := tx.WithContext(ctx).Where("id = ?", tradeID).Delete(&dbTrade{}).Error; err != nil {
		return err
	}

	// Update position
	return store.updatePosition(ctx, tx, trade.AccountID, trade.InstrumentID)
}

// deleteTradesByTransactionID removes all trades (and cascading lots/disposals) for a transaction.
func (store *Store) deleteTradesByTransactionID(ctx context.Context, tx *gorm.DB, transactionID uint) error {
	var trades []dbTrade
	if err := tx.WithContext(ctx).Where("transaction_id = ?", transactionID).Find(&trades).Error; err != nil {
		return err
	}
	for _, trade := range trades {
		if err := store.deleteTrade(ctx, tx, trade.Id); err != nil {
			return err
		}
	}
	return nil
}
