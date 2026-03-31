package accounting

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ---------------------------------------------------------------------------
// Trades
// ---------------------------------------------------------------------------

type TradeType int

const (
	BuyTrade         TradeType = 1
	SellTrade        TradeType = 2
	GrantTrade       TradeType = 3
	TransferOutTrade TradeType = 4
	TransferInTrade  TradeType = 5
	ForfeitTrade     TradeType = 6
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

// restoreSellTradeLots restores the quantity of each lot consumed by a sell trade,
// reversing the FIFO allocation so the lots are available again.
func restoreSellTradeLots(ctx context.Context, tx *gorm.DB, tradeID uint) error {
	var disposals []dbLotDisposal
	if err := tx.WithContext(ctx).Where("sell_trade_id = ?", tradeID).Find(&disposals).Error; err != nil {
		return err
	}
	for _, d := range disposals {
		var lot dbLot
		if err := tx.WithContext(ctx).Where("id = ?", d.LotID).First(&lot).Error; err != nil {
			return fmt.Errorf("failed to find lot %d for restoration: %w", d.LotID, err)
		}
		lot.Quantity += d.Quantity
		lot.CostBasis = roundMoney(lot.Quantity * lot.CostPerShare)
		lot.ClosedDate = nil
		if lot.Quantity >= lot.OriginalQty {
			lot.Status = LotOpen
		} else {
			lot.Status = LotPartial
		}
		if err := tx.WithContext(ctx).Save(&lot).Error; err != nil {
			return fmt.Errorf("failed to restore lot %d: %w", d.LotID, err)
		}
	}
	return nil
}

// deleteTrade removes a trade and its associated lots/disposals, then updates the position.
func (store *Store) deleteTrade(ctx context.Context, tx *gorm.DB, tradeID uint) error {
	var trade dbTrade
	if err := tx.WithContext(ctx).Where("id = ?", tradeID).First(&trade).Error; err != nil {
		return err
	}

	// For sell and forfeit trades: restore the quantity back to each lot consumed
	// before deleting the disposal records.
	if trade.TradeType == SellTrade || trade.TradeType == ForfeitTrade {
		if err := restoreSellTradeLots(ctx, tx, tradeID); err != nil {
			return err
		}
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

// ---------------------------------------------------------------------------
// Lots
// ---------------------------------------------------------------------------

type LotStatus int

const (
	LotOpen    LotStatus = 1
	LotPartial LotStatus = 2
	LotClosed  LotStatus = 3
)

type dbLot struct {
	Id           uint      `gorm:"primaryKey"`
	TradeID      uint      `gorm:"not null;index"`
	AccountID    uint      `gorm:"not null;index"`
	InstrumentID uint      `gorm:"not null;index"`
	OpenDate     time.Time `gorm:"not null"`
	Quantity     float64   `gorm:"not null"`
	OriginalQty  float64   `gorm:"not null"`
	CostPerShare float64   `gorm:"not null"`
	CostBasis    float64   `gorm:"not null"`
	Status       LotStatus `gorm:"not null;default:1"`
	ClosedDate   *time.Time
	CreatedAt    time.Time
}

type dbLotDisposal struct {
	Id          uint    `gorm:"primaryKey"`
	LotID       uint    `gorm:"not null;index"`
	SellTradeID uint    `gorm:"not null;index"`
	Quantity    float64 `gorm:"not null"`
	Proceeds    float64
	RealizedGL  float64
	Date        time.Time `gorm:"not null"`
}

type Lot struct {
	Id           uint
	TradeID      uint
	AccountID    uint
	InstrumentID uint
	OpenDate     time.Time
	Quantity     float64
	OriginalQty  float64
	CostPerShare float64
	CostBasis    float64
	Status       LotStatus
	ClosedDate   *time.Time
}

type LotDisposal struct {
	Id          uint
	LotID       uint
	SellTradeID uint
	Quantity    float64
	Proceeds    float64
	RealizedGL  float64
	Date        time.Time
}

type LotAllocation struct {
	LotID      uint
	Quantity   float64
	CostBasis  float64
	RealizedGL float64
}

type CostBasisMethod int

const (
	FIFO CostBasisMethod = iota
)

func lotFromDb(l dbLot) Lot {
	return Lot{
		Id:           l.Id,
		TradeID:      l.TradeID,
		AccountID:    l.AccountID,
		InstrumentID: l.InstrumentID,
		OpenDate:     l.OpenDate,
		Quantity:     l.Quantity,
		OriginalQty:  l.OriginalQty,
		CostPerShare: l.CostPerShare,
		CostBasis:    l.CostBasis,
		Status:       l.Status,
		ClosedDate:   l.ClosedDate,
	}
}

// allocateLotsForSell allocates sell quantity against open/partial lots using the specified method.
// Returns allocations, total cost basis, and error.
func (store *Store) allocateLotsForSell(ctx context.Context, tx *gorm.DB, accountID, instrumentID uint, sellQty, proceeds float64, sellDate time.Time, sellTradeID uint, method CostBasisMethod) ([]LotAllocation, float64, error) {
	var lots []dbLot

	orderClause := "open_date ASC, id ASC" // FIFO
	if err := tx.WithContext(ctx).
		Where("account_id = ? AND instrument_id = ? AND status IN ?", accountID, instrumentID, []LotStatus{LotOpen, LotPartial}).
		Order(orderClause).
		Find(&lots).Error; err != nil {
		return nil, 0, err
	}

	remaining := sellQty
	var totalCostBasis float64
	var allocations []LotAllocation

	// Compute total available for proceeds allocation
	totalAvailableQty := 0.0
	for _, lot := range lots {
		totalAvailableQty += lot.Quantity
	}
	if totalAvailableQty < sellQty {
		return nil, 0, ErrValidation("insufficient quantity for sell")
	}

	for i := range lots {
		if remaining <= 0 {
			break
		}

		lot := &lots[i]
		allocQty := math.Min(lot.Quantity, remaining)
		allocCost := roundMoney(allocQty * lot.CostPerShare)
		allocProceeds := roundMoney(proceeds * (allocQty / sellQty))
		realizedGL := roundMoney(allocProceeds - allocCost)

		// Create disposal record
		disposal := dbLotDisposal{
			LotID:       lot.Id,
			SellTradeID: sellTradeID,
			Quantity:    allocQty,
			Proceeds:    allocProceeds,
			RealizedGL:  realizedGL,
			Date:        sellDate,
		}
		if err := tx.WithContext(ctx).Create(&disposal).Error; err != nil {
			return nil, 0, fmt.Errorf("failed to create lot disposal: %w", err)
		}

		// Update lot
		lot.Quantity -= allocQty
		if lot.Quantity <= 0 {
			lot.Quantity = 0
			lot.Status = LotClosed
			lot.ClosedDate = &sellDate
		} else {
			lot.Status = LotPartial
		}
		lot.CostBasis = roundMoney(lot.Quantity * lot.CostPerShare)

		if err := tx.WithContext(ctx).Save(lot).Error; err != nil {
			return nil, 0, fmt.Errorf("failed to update lot: %w", err)
		}

		allocations = append(allocations, LotAllocation{
			LotID:      lot.Id,
			Quantity:   allocQty,
			CostBasis:  allocCost,
			RealizedGL: realizedGL,
		})

		totalCostBasis += allocCost
		remaining -= allocQty
	}

	return allocations, totalCostBasis, nil
}

// LotSelection specifies an explicit lot and quantity for manual lot allocation.
type LotSelection struct {
	LotID    uint
	Quantity float64
}

// allocateLotsManual allocates sell quantity against explicitly specified lots.
// Returns allocations, total cost basis, and error.
func (store *Store) allocateLotsManual(ctx context.Context, tx *gorm.DB,
	selections []LotSelection, proceeds float64, sellQty float64,
	sellDate time.Time, sellTradeID uint) ([]LotAllocation, float64, error) {

	// Validate total allocated quantity equals sellQty
	totalAllocated := 0.0
	for _, sel := range selections {
		totalAllocated += sel.Quantity
	}
	if math.Abs(totalAllocated-sellQty) > 0.0001 {
		return nil, 0, ErrValidation(fmt.Sprintf(
			"manual lot allocation total (%.4f) does not equal sell quantity (%.4f)",
			totalAllocated, sellQty))
	}

	var totalCostBasis float64
	var allocations []LotAllocation

	for _, sel := range selections {
		if sel.Quantity <= 0 {
			continue
		}

		var lot dbLot
		if err := tx.WithContext(ctx).Where("id = ?", sel.LotID).First(&lot).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, 0, ErrValidation(fmt.Sprintf("lot %d not found", sel.LotID))
			}
			return nil, 0, fmt.Errorf("failed to load lot %d: %w", sel.LotID, err)
		}

		if lot.Status == LotClosed {
			return nil, 0, ErrValidation(fmt.Sprintf("lot %d is already closed", sel.LotID))
		}
		if sel.Quantity > lot.Quantity+0.0001 {
			return nil, 0, ErrValidation(fmt.Sprintf(
				"lot %d has only %.4f shares available, requested %.4f",
				sel.LotID, lot.Quantity, sel.Quantity))
		}

		allocQty := sel.Quantity
		allocCost := roundMoney(allocQty * lot.CostPerShare)
		allocProceeds := roundMoney(proceeds * (allocQty / sellQty))
		realizedGL := roundMoney(allocProceeds - allocCost)

		disposal := dbLotDisposal{
			LotID:       lot.Id,
			SellTradeID: sellTradeID,
			Quantity:    allocQty,
			Proceeds:    allocProceeds,
			RealizedGL:  realizedGL,
			Date:        sellDate,
		}
		if err := tx.WithContext(ctx).Create(&disposal).Error; err != nil {
			return nil, 0, fmt.Errorf("failed to create lot disposal: %w", err)
		}

		lot.Quantity -= allocQty
		if lot.Quantity <= 0 {
			lot.Quantity = 0
			lot.Status = LotClosed
			lot.ClosedDate = &sellDate
		} else {
			lot.Status = LotPartial
		}
		lot.CostBasis = roundMoney(lot.Quantity * lot.CostPerShare)
		if err := tx.WithContext(ctx).Save(&lot).Error; err != nil {
			return nil, 0, fmt.Errorf("failed to update lot: %w", err)
		}

		allocations = append(allocations, LotAllocation{
			LotID:      lot.Id,
			Quantity:   allocQty,
			CostBasis:  allocCost,
			RealizedGL: realizedGL,
		})
		totalCostBasis += allocCost
	}

	return allocations, totalCostBasis, nil
}

// transferLots closes source lots and creates new lots in the target account with the same cost basis.
func (store *Store) transferLots(ctx context.Context, tx *gorm.DB, sourceAccountID, targetAccountID, instrumentID uint, qty float64, date time.Time, tradeID uint) error {
	var lots []dbLot

	if err := tx.WithContext(ctx).
		Where("account_id = ? AND instrument_id = ? AND status IN ?", sourceAccountID, instrumentID, []LotStatus{LotOpen, LotPartial}).
		Order("open_date ASC, id ASC").
		Find(&lots).Error; err != nil {
		return err
	}

	remaining := qty
	totalAvailableQty := 0.0
	for _, lot := range lots {
		totalAvailableQty += lot.Quantity
	}
	if totalAvailableQty < qty {
		return ErrValidation("insufficient quantity for transfer")
	}

	for i := range lots {
		if remaining <= 0 {
			break
		}

		lot := &lots[i]
		moveQty := math.Min(lot.Quantity, remaining)
		moveCost := roundMoney(moveQty * lot.CostPerShare)

		// Reduce source lot
		lot.Quantity -= moveQty
		if lot.Quantity <= 0 {
			lot.Quantity = 0
			lot.Status = LotClosed
			lot.ClosedDate = &date
		} else {
			lot.Status = LotPartial
		}
		lot.CostBasis = roundMoney(lot.Quantity * lot.CostPerShare)

		if err := tx.WithContext(ctx).Save(lot).Error; err != nil {
			return fmt.Errorf("failed to update source lot: %w", err)
		}

		// Create new lot in target account with same cost basis
		newLot := dbLot{
			TradeID:      tradeID,
			AccountID:    targetAccountID,
			InstrumentID: instrumentID,
			OpenDate:     lot.OpenDate, // preserve original open date for FIFO ordering
			Quantity:     moveQty,
			OriginalQty:  moveQty,
			CostPerShare: lot.CostPerShare,
			CostBasis:    moveCost,
			Status:       LotOpen,
		}
		if err := tx.WithContext(ctx).Create(&newLot).Error; err != nil {
			return fmt.Errorf("failed to create target lot: %w", err)
		}

		remaining -= moveQty
	}

	return nil
}

// vestLots moves shares from source lots to a target account, overriding cost basis
// with the vesting price. Unlike transferLots which preserves cost basis, this function
// sets the new lot's CostPerShare to vestingPrice and OpenDate to vestDate.
func (store *Store) vestLots(ctx context.Context, tx *gorm.DB,
	selections []LotSelection, vestingPrice float64, vestDate time.Time,
	tradeID, sourceAccountID, targetAccountID, instrumentID uint) error {

	for _, sel := range selections {
		if sel.Quantity <= 0 {
			continue
		}

		var lot dbLot
		if err := tx.WithContext(ctx).Where("id = ?", sel.LotID).First(&lot).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrValidation(fmt.Sprintf("lot %d not found", sel.LotID))
			}
			return fmt.Errorf("failed to load lot %d: %w", sel.LotID, err)
		}

		if lot.AccountID != sourceAccountID {
			return ErrValidation(fmt.Sprintf("lot %d does not belong to the source account", sel.LotID))
		}

		if lot.InstrumentID != instrumentID {
			return ErrValidation(fmt.Sprintf("lot %d instrument does not match the vest instrument", sel.LotID))
		}

		if lot.Status == LotClosed {
			return ErrValidation(fmt.Sprintf("lot %d is already closed", lot.Id))
		}
		if sel.Quantity > lot.Quantity+0.0001 {
			return ErrValidation(fmt.Sprintf(
				"lot %d has only %.4f shares available, requested %.4f",
				lot.Id, lot.Quantity, sel.Quantity))
		}

		// Reduce source lot
		lot.Quantity -= sel.Quantity
		if lot.Quantity <= 0 {
			lot.Quantity = 0
			lot.Status = LotClosed
			lot.ClosedDate = &vestDate
		} else {
			lot.Status = LotPartial
		}
		lot.CostBasis = roundMoney(lot.Quantity * lot.CostPerShare)
		if err := tx.WithContext(ctx).Save(&lot).Error; err != nil {
			return fmt.Errorf("failed to update source lot: %w", err)
		}

		// Record which source lot was consumed (reuse dbLotDisposal for traceability).
		// This allows GetTransaction to reconstruct LotSelections with correct source lot IDs.
		disposal := dbLotDisposal{
			LotID:       lot.Id,
			SellTradeID: tradeID, // reuse field: points to the vest's InTrade
			Quantity:    sel.Quantity,
			Date:        vestDate,
		}
		if err := tx.WithContext(ctx).Create(&disposal).Error; err != nil {
			return fmt.Errorf("failed to create vest disposal: %w", err)
		}

		// Create new lot in target with vesting price as cost basis
		newLot := dbLot{
			TradeID:      tradeID,
			AccountID:    targetAccountID,
			InstrumentID: instrumentID,
			OpenDate:     vestDate,
			Quantity:     sel.Quantity,
			OriginalQty:  sel.Quantity,
			CostPerShare: vestingPrice,
			CostBasis:    roundMoney(sel.Quantity * vestingPrice),
			Status:       LotOpen,
		}
		if err := tx.WithContext(ctx).Create(&newLot).Error; err != nil {
			return fmt.Errorf("failed to create vested lot: %w", err)
		}
	}

	return nil
}

// forfeitLots closes/reduces source lots for a stock forfeit. Unlike vestLots,
// no new lots are created (the shares simply disappear). dbLotDisposal records
// are created so that the operation can be reversed on delete.
func (store *Store) forfeitLots(ctx context.Context, tx *gorm.DB,
	selections []LotSelection, forfeitDate time.Time,
	tradeID, sourceAccountID, instrumentID uint) error {

	for _, sel := range selections {
		if sel.Quantity <= 0 {
			continue
		}

		var lot dbLot
		if err := tx.WithContext(ctx).Where("id = ?", sel.LotID).First(&lot).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrValidation(fmt.Sprintf("lot %d not found", sel.LotID))
			}
			return fmt.Errorf("failed to load lot %d: %w", sel.LotID, err)
		}

		if lot.AccountID != sourceAccountID {
			return ErrValidation(fmt.Sprintf("lot %d does not belong to the source account", sel.LotID))
		}

		if lot.InstrumentID != instrumentID {
			return ErrValidation(fmt.Sprintf("lot %d instrument does not match the forfeit instrument", sel.LotID))
		}

		if lot.Status == LotClosed {
			return ErrValidation(fmt.Sprintf("lot %d is already closed", lot.Id))
		}
		if sel.Quantity > lot.Quantity+0.0001 {
			return ErrValidation(fmt.Sprintf(
				"lot %d has only %.4f shares available, requested %.4f",
				lot.Id, lot.Quantity, sel.Quantity))
		}

		// Reduce source lot
		lot.Quantity -= sel.Quantity
		if lot.Quantity <= 0 {
			lot.Quantity = 0
			lot.Status = LotClosed
			lot.ClosedDate = &forfeitDate
		} else {
			lot.Status = LotPartial
		}
		lot.CostBasis = roundMoney(lot.Quantity * lot.CostPerShare)
		if err := tx.WithContext(ctx).Save(&lot).Error; err != nil {
			return fmt.Errorf("failed to update source lot: %w", err)
		}

		// Record disposal so we can restore on delete
		disposal := dbLotDisposal{
			LotID:       lot.Id,
			SellTradeID: tradeID, // reuse field: points to the forfeit trade
			Quantity:    sel.Quantity,
			Date:        forfeitDate,
		}
		if err := tx.WithContext(ctx).Create(&disposal).Error; err != nil {
			return fmt.Errorf("failed to create forfeit disposal: %w", err)
		}
	}

	return nil
}

type ListLotsOpts struct {
	AccountID    uint
	InstrumentID uint
	Status       *LotStatus
	BeforeDate   *time.Time
}

func (store *Store) ListLots(ctx context.Context, opts ListLotsOpts) ([]Lot, error) {
	db := store.db.WithContext(ctx).Table("db_lots")

	if opts.AccountID != 0 {
		db = db.Where("account_id = ?", opts.AccountID)
	}
	if opts.InstrumentID != 0 {
		db = db.Where("instrument_id = ?", opts.InstrumentID)
	}
	if opts.Status != nil {
		db = db.Where("status = ?", *opts.Status)
	}
	if opts.BeforeDate != nil {
		db = db.Where("open_date <= ?", *opts.BeforeDate)
	}

	db = db.Order("open_date ASC, id ASC")

	var lots []dbLot
	if err := db.Find(&lots).Error; err != nil {
		return nil, err
	}

	result := make([]Lot, len(lots))
	for i, l := range lots {
		result[i] = lotFromDb(l)
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Positions
// ---------------------------------------------------------------------------

type dbPosition struct {
	Id           uint    `gorm:"primaryKey"`
	AccountID    uint    `gorm:"not null;uniqueIndex:idx_acct_inst"`
	InstrumentID uint    `gorm:"not null;uniqueIndex:idx_acct_inst"`
	Quantity     float64 `gorm:"not null;default:0"`
	CostBasis    float64 `gorm:"not null;default:0"`
	AvgCost      float64 `gorm:"not null;default:0"`
	UpdatedAt    time.Time
}

type Position struct {
	Id           uint
	AccountID    uint
	InstrumentID uint
	Quantity     float64
	CostBasis    float64
	AvgCost      float64
}

func positionFromDb(p dbPosition) Position {
	return Position{
		Id:           p.Id,
		AccountID:    p.AccountID,
		InstrumentID: p.InstrumentID,
		Quantity:     p.Quantity,
		CostBasis:    p.CostBasis,
		AvgCost:      p.AvgCost,
	}
}

// updatePosition recalculates position from open lots and upserts db_positions.
func (store *Store) updatePosition(ctx context.Context, tx *gorm.DB, accountID, instrumentID uint) error {
	var result struct {
		TotalQty  float64
		TotalCost float64
	}
	if err := tx.WithContext(ctx).
		Model(&dbLot{}).
		Select("COALESCE(SUM(quantity), 0) as total_qty, COALESCE(SUM(cost_basis), 0) as total_cost").
		Where("account_id = ? AND instrument_id = ? AND status IN ?", accountID, instrumentID, []LotStatus{LotOpen, LotPartial}).
		Scan(&result).Error; err != nil {
		return err
	}

	avgCost := 0.0
	if result.TotalQty > 0 {
		avgCost = roundMoney(result.TotalCost / result.TotalQty)
	}

	pos := dbPosition{
		AccountID:    accountID,
		InstrumentID: instrumentID,
		Quantity:     result.TotalQty,
		CostBasis:    roundMoney(result.TotalCost),
		AvgCost:      avgCost,
	}

	return tx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "account_id"}, {Name: "instrument_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"quantity", "cost_basis", "avg_cost", "updated_at"}),
		}).
		Create(&pos).Error
}

func (store *Store) GetPosition(ctx context.Context, accountID, instrumentID uint) (Position, error) {
	var pos dbPosition
	if err := store.db.WithContext(ctx).
		Where("account_id = ? AND instrument_id = ?", accountID, instrumentID).
		First(&pos).Error; err != nil {
		return Position{}, err
	}
	return positionFromDb(pos), nil
}

type ListPositionsOpts struct {
	AccountID uint
}

func (store *Store) ListPositions(ctx context.Context, opts ListPositionsOpts) ([]Position, error) {
	db := store.db.WithContext(ctx).Model(&dbPosition{})

	if opts.AccountID != 0 {
		db = db.Where("account_id = ?", opts.AccountID)
	}

	// Only return positions with non-zero quantity
	db = db.Where("quantity > 0")

	var positions []dbPosition
	if err := db.Find(&positions).Error; err != nil {
		return nil, err
	}

	result := make([]Position, len(positions))
	for i, p := range positions {
		result[i] = positionFromDb(p)
	}
	return result, nil
}

func (store *Store) ListAllPositions(ctx context.Context) ([]Position, error) {
	return store.ListPositions(ctx, ListPositionsOpts{})
}

// ---------------------------------------------------------------------------
// Instrument Returns (aggregated per instrument across all accounts)
// ---------------------------------------------------------------------------

type InstrumentReturn struct {
	InstrumentID     uint
	TotalInvested    float64 // sum of cost basis from buy/grant trades
	RealizedProceeds float64 // sum of proceeds from sell disposals
	RealizedGL       float64 // sum of realized gain/loss from sell disposals
	CurrentQuantity  float64 // quantity still held (open positions)
	CurrentCostBasis float64 // cost basis of remaining open position
	FirstTradeDate   time.Time
	LastTradeDate    time.Time
}

func (store *Store) ListInstrumentReturns(ctx context.Context) ([]InstrumentReturn, error) {
	// Step 1: invested amounts from buy/grant trades per instrument
	type investedRow struct {
		InstrumentID  uint    `gorm:"column:instrument_id"`
		TotalInvested float64 `gorm:"column:total_invested"`
	}
	var invested []investedRow
	if err := store.db.WithContext(ctx).
		Table("db_trades").
		Select("instrument_id, SUM(total_amount) as total_invested").
		Where("trade_type IN ?", []TradeType{BuyTrade, GrantTrade}).
		Group("instrument_id").
		Find(&invested).Error; err != nil {
		return nil, fmt.Errorf("failed to query invested amounts: %w", err)
	}

	// Step 2: realized returns from sell disposals per instrument
	type realizedRow struct {
		InstrumentID    uint    `gorm:"column:instrument_id"`
		TotalProceeds   float64 `gorm:"column:total_proceeds"`
		TotalRealizedGL float64 `gorm:"column:total_realized_gl"`
	}
	var realized []realizedRow
	if err := store.db.WithContext(ctx).
		Table("db_lot_disposals d").
		Joins("JOIN db_lots l ON d.lot_id = l.id").
		Joins("JOIN db_trades t ON d.sell_trade_id = t.id").
		Select("l.instrument_id, SUM(d.proceeds) as total_proceeds, SUM(d.realized_gl) as total_realized_gl").
		Where("t.trade_type = ?", SellTrade).
		Group("l.instrument_id").
		Find(&realized).Error; err != nil {
		return nil, fmt.Errorf("failed to query realized returns: %w", err)
	}

	// Step 3: current open positions per instrument (across all accounts)
	type positionRow struct {
		InstrumentID uint    `gorm:"column:instrument_id"`
		TotalQty     float64 `gorm:"column:total_qty"`
		TotalCost    float64 `gorm:"column:total_cost"`
	}
	var positions []positionRow
	if err := store.db.WithContext(ctx).
		Table("db_positions").
		Select("instrument_id, SUM(quantity) as total_qty, SUM(cost_basis) as total_cost").
		Where("quantity > 0").
		Group("instrument_id").
		Find(&positions).Error; err != nil {
		return nil, fmt.Errorf("failed to query current positions: %w", err)
	}

	// Step 4: trade date ranges per instrument — query via GORM model to avoid
	// raw-string date scanning issues with SQLite aggregate functions.
	var allTrades []dbTrade
	if err := store.db.WithContext(ctx).
		Order("date ASC").
		Find(&allTrades).Error; err != nil {
		return nil, fmt.Errorf("failed to query trades for date ranges: %w", err)
	}
	type dateRange struct {
		first time.Time
		last  time.Time
	}
	tradeDates := map[uint]*dateRange{}
	for _, t := range allTrades {
		dr, ok := tradeDates[t.InstrumentID]
		if !ok {
			dr = &dateRange{first: t.Date, last: t.Date}
			tradeDates[t.InstrumentID] = dr
		}
		if t.Date.Before(dr.first) {
			dr.first = t.Date
		}
		if t.Date.After(dr.last) {
			dr.last = t.Date
		}
	}

	// Combine into a map by instrument_id
	byInst := map[uint]*InstrumentReturn{}

	for _, inv := range invested {
		r := &InstrumentReturn{
			InstrumentID:  inv.InstrumentID,
			TotalInvested: inv.TotalInvested,
		}
		if dr, ok := tradeDates[inv.InstrumentID]; ok {
			r.FirstTradeDate = dr.first
			r.LastTradeDate = dr.last
		}
		byInst[inv.InstrumentID] = r
	}

	for _, real := range realized {
		r, ok := byInst[real.InstrumentID]
		if !ok {
			r = &InstrumentReturn{InstrumentID: real.InstrumentID}
			if dr, ok := tradeDates[real.InstrumentID]; ok {
				r.FirstTradeDate = dr.first
				r.LastTradeDate = dr.last
			}
			byInst[real.InstrumentID] = r
		}
		r.RealizedProceeds = real.TotalProceeds
		r.RealizedGL = real.TotalRealizedGL
	}

	for _, pos := range positions {
		r, ok := byInst[pos.InstrumentID]
		if !ok {
			r = &InstrumentReturn{InstrumentID: pos.InstrumentID}
			if dr, ok := tradeDates[pos.InstrumentID]; ok {
				r.FirstTradeDate = dr.first
				r.LastTradeDate = dr.last
			}
			byInst[pos.InstrumentID] = r
		}
		r.CurrentQuantity = pos.TotalQty
		r.CurrentCostBasis = pos.TotalCost
	}

	result := make([]InstrumentReturn, 0, len(byInst))
	for _, r := range byInst {
		result = append(result, *r)
	}
	return result, nil
}
