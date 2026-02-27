package accounting

import (
	"context"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"
)

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

type ListLotsOpts struct {
	AccountID    uint
	InstrumentID uint
	Status       *LotStatus
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
