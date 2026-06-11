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
	Notes                string
	Type                 string
	Exchange             string
}

type dbInstrument struct {
	ID         uint `gorm:"primaryKey"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
	ProviderID uint           `gorm:"index"`

	Symbol   string `gorm:"uniqueIndex:idx_instrument_symbol"`
	Name     string
	Currency string
	Notes    string
	Type     string
	Exchange string
}

func (dbInstrument) TableName() string { return "db_instruments" }

func dbToInstrument(in dbInstrument) Instrument {
	return Instrument{
		ID:                   in.ID,
		InstrumentProviderID: in.ProviderID,
		Symbol:               in.Symbol,
		Name:                 in.Name,
		Currency:             currency.MustParseISO(in.Currency),
		Notes:                in.Notes,
		Type:                 in.Type,
		Exchange:             in.Exchange,
	}
}

var ErrInstrumentNotFound = errors.New("instrument not found")
var ErrInstrumentSymbolDuplicate = errors.New("instrument symbol already exists")
var ErrNoChanges = errors.New("no changes applied")

// ErrValidation is a validation error that can be matched with errors.As.
type ErrValidation string

func (e ErrValidation) Error() string { return string(e) }

// InstrumentUpdatePayload holds optional fields for updating an instrument.
type InstrumentUpdatePayload struct {
	Symbol   *string
	Name     *string
	Currency *string
	Notes    *string
	Type     *string
	Exchange *string
}

func (s *Store) CreateInstrument(ctx context.Context, item Instrument) (uint, error) {
	if item.Symbol == "" {
		return 0, ErrValidation("symbol cannot be empty")
	}
	if item.Currency == (currency.Unit{}) {
		return 0, ErrValidation("currency cannot be empty")
	}
	// Check including soft-deleted rows: duplicate symbol would violate UNIQUE
	var existing dbInstrument
	err := s.db.WithContext(ctx).Unscoped().
		Where("symbol = ?", item.Symbol).
		First(&existing).Error
	if err == nil {
		if existing.DeletedAt.Valid {
			return s.restoreSoftDeletedInstrument(ctx, existing, item)
		}
		return 0, ErrInstrumentSymbolDuplicate
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	payload := dbInstrument{
		ProviderID: item.InstrumentProviderID,
		Symbol:     item.Symbol,
		Name:       item.Name,
		Currency:   item.Currency.String(),
		Notes:      item.Notes,
		Type:       item.Type,
		Exchange:   item.Exchange,
	}
	d := s.db.WithContext(ctx).Create(&payload)
	if d.Error != nil {
		return 0, d.Error
	}
	// Define the OHLCV series now so ingest paths never need to auto-register on the write path.
	if err := s.RegisterInstrument(ctx, payload.Symbol); err != nil {
		return 0, err
	}
	return payload.ID, nil
}

// restoreSoftDeletedInstrument revives a soft-deleted instrument row with the new field values and
// (re)defines its OHLCV series, mirroring the create path.
func (s *Store) restoreSoftDeletedInstrument(ctx context.Context, existing dbInstrument, item Instrument) (uint, error) {
	existing.DeletedAt = gorm.DeletedAt{}
	existing.ProviderID = item.InstrumentProviderID
	existing.Name = item.Name
	existing.Currency = item.Currency.String()
	existing.Notes = item.Notes
	existing.Type = item.Type
	existing.Exchange = item.Exchange
	if u := s.db.WithContext(ctx).Unscoped().Save(&existing); u.Error != nil {
		return 0, u.Error
	}
	if err := s.RegisterInstrument(ctx, existing.Symbol); err != nil {
		return 0, err
	}
	return existing.ID, nil
}

func (s *Store) GetInstrument(ctx context.Context, id uint) (Instrument, error) {
	var payload dbInstrument
	d := s.db.WithContext(ctx).Where("id = ?", id).First(&payload)
	if d.Error != nil {
		if errors.Is(d.Error, gorm.ErrRecordNotFound) {
			return Instrument{}, ErrInstrumentNotFound
		}
		return Instrument{}, d.Error
	}
	return dbToInstrument(payload), nil
}

func (s *Store) ListInstruments(ctx context.Context) ([]Instrument, error) {
	var results []dbInstrument
	if err := s.db.WithContext(ctx).
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

func (s *Store) UpdateInstrument(ctx context.Context, id uint, item InstrumentUpdatePayload) error {
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
	if item.Notes != nil {
		updateStruct.Notes = *item.Notes
		selectedFields = append(selectedFields, "Notes")
	}
	if item.Type != nil {
		updateStruct.Type = *item.Type
		selectedFields = append(selectedFields, "Type")
	}
	if item.Exchange != nil {
		updateStruct.Exchange = *item.Exchange
		selectedFields = append(selectedFields, "Exchange")
	}
	if len(selectedFields) == 0 {
		return ErrNoChanges
	}

	if item.Symbol != nil {
		var count int64
		if err := s.db.WithContext(ctx).Model(&dbInstrument{}).
			Where("symbol = ? AND id != ?", *item.Symbol, id).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrInstrumentSymbolDuplicate
		}
	}

	res := s.db.WithContext(ctx).Model(&dbInstrument{}).
		Where("id = ?", id).
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

func (s *Store) DeleteInstrument(ctx context.Context, id uint) error {
	res := s.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&dbInstrument{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrInstrumentNotFound
	}
	return nil
}
