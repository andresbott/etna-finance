package marketdata

import (
	"context"
	"errors"
	"time"

	"golang.org/x/text/currency"
	"gorm.io/gorm"
)

// Instrument represents a tradeable instrument (e.g. a stock, ETF).
type Instrument struct {
	ID                   uint
	InstrumentProviderID uint
	Symbol               string
	Name                 string
	Currency             currency.Unit
}

type dbInstrument struct {
	ID         uint `gorm:"primaryKey"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
	OwnerId    string         `gorm:"uniqueIndex:idx_owner_symbol;index"`
	ProviderID uint           `gorm:"index"`

	Symbol   string `gorm:"uniqueIndex:idx_owner_symbol"`
	Name     string
	Currency string
}

func (dbInstrument) TableName() string { return "db_instruments" }

func dbToInstrument(in dbInstrument) Instrument {
	return Instrument{
		ID:                   in.ID,
		InstrumentProviderID: in.ProviderID,
		Symbol:               in.Symbol,
		Name:                 in.Name,
		Currency:             currency.MustParseISO(in.Currency),
	}
}

var ErrInstrumentNotFound = errors.New("instrument not found")
var ErrInstrumentSymbolDuplicate = errors.New("instrument symbol already exists for this tenant")
var ErrNoChanges = errors.New("no changes applied")

// ErrValidation is a validation error that can be matched with errors.As.
type ErrValidation string

func (e ErrValidation) Error() string { return string(e) }

// InstrumentUpdatePayload holds optional fields for updating an instrument.
type InstrumentUpdatePayload struct {
	Symbol   *string
	Name     *string
	Currency *string
}

func (s *Store) CreateInstrument(ctx context.Context, item Instrument, tenant string) (uint, error) {
	if item.Symbol == "" {
		return 0, ErrValidation("symbol cannot be empty")
	}
	if item.Currency == (currency.Unit{}) {
		return 0, ErrValidation("currency cannot be empty")
	}
	// Check including soft-deleted rows: duplicate (owner_id, symbol) would violate UNIQUE
	var existing dbInstrument
	err := s.db.WithContext(ctx).Unscoped().
		Where("owner_id = ? AND symbol = ?", tenant, item.Symbol).
		First(&existing).Error
	if err == nil {
		if existing.DeletedAt.Valid {
			// Restore the soft-deleted row and update fields
			existing.DeletedAt = gorm.DeletedAt{}
			existing.ProviderID = item.InstrumentProviderID
			existing.Name = item.Name
			existing.Currency = item.Currency.String()
			if u := s.db.WithContext(ctx).Unscoped().Save(&existing); u.Error != nil {
				return 0, u.Error
			}
			return existing.ID, nil
		}
		return 0, ErrInstrumentSymbolDuplicate
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	payload := dbInstrument{
		OwnerId:    tenant,
		ProviderID: item.InstrumentProviderID,
		Symbol:     item.Symbol,
		Name:       item.Name,
		Currency:   item.Currency.String(),
	}
	d := s.db.WithContext(ctx).Create(&payload)
	if d.Error != nil {
		return 0, d.Error
	}
	return payload.ID, nil
}

func (s *Store) GetInstrument(ctx context.Context, id uint, tenant string) (Instrument, error) {
	var payload dbInstrument
	d := s.db.WithContext(ctx).Where("id = ? AND owner_id = ?", id, tenant).First(&payload)
	if d.Error != nil {
		if errors.Is(d.Error, gorm.ErrRecordNotFound) {
			return Instrument{}, ErrInstrumentNotFound
		}
		return Instrument{}, d.Error
	}
	return dbToInstrument(payload), nil
}

func (s *Store) ListInstruments(ctx context.Context, tenant string) ([]Instrument, error) {
	var results []dbInstrument
	if err := s.db.WithContext(ctx).
		Where("owner_id = ?", tenant).
		Order("id ASC").
		Find(&results).Error; err != nil {
		return nil, err
	}
	out := make([]Instrument, 0, len(results))
	for _, r := range results {
		out = append(out, dbToInstrument(r))
	}
	return out, nil
}

func (s *Store) UpdateInstrument(ctx context.Context, id uint, tenant string, item InstrumentUpdatePayload) error {
	updateStruct := dbInstrument{}
	var selectedFields []string

	if item.Symbol != nil {
		if *item.Symbol == "" {
			return ErrValidation("symbol cannot be empty")
		}
		updateStruct.Symbol = *item.Symbol
		selectedFields = append(selectedFields, "Symbol")
	}
	if item.Name != nil {
		updateStruct.Name = *item.Name
		selectedFields = append(selectedFields, "Name")
	}
	if item.Currency != nil {
		if *item.Currency == "" {
			return ErrValidation("currency cannot be empty")
		}
		updateStruct.Currency = *item.Currency
		selectedFields = append(selectedFields, "Currency")
	}
	if len(selectedFields) == 0 {
		return ErrNoChanges
	}

	if item.Symbol != nil {
		var count int64
		if err := s.db.WithContext(ctx).Model(&dbInstrument{}).
			Where("owner_id = ? AND symbol = ? AND id != ?", tenant, *item.Symbol, id).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrInstrumentSymbolDuplicate
		}
	}

	res := s.db.WithContext(ctx).Model(&dbInstrument{}).
		Where("id = ? AND owner_id = ?", id, tenant).
		Select(selectedFields).
		Updates(updateStruct)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrInstrumentNotFound
	}
	return nil
}

func (s *Store) DeleteInstrument(ctx context.Context, id uint, tenant string) error {
	res := s.db.WithContext(ctx).
		Where("id = ? AND owner_id = ?", id, tenant).
		Delete(&dbInstrument{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrInstrumentNotFound
	}
	return nil
}
