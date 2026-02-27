package accounting

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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
