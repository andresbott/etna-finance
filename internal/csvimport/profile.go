package csvimport

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrProfileNotFound = errors.New("import profile not found")

// dbImportProfile is the DB internal representation of an ImportProfile.
type dbImportProfile struct {
	ID                uint   `gorm:"primarykey"`
	Name              string `gorm:"not null"`
	CsvSeparator      string `gorm:"default:','"`
	SkipRows          int    `gorm:"default:0"`
	DateColumn        string `gorm:"not null"`
	DateFormat        string `gorm:"not null"`
	DescriptionColumn string `gorm:"not null"`
	AmountColumn      string `gorm:"not null"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ImportProfile is the public-facing representation of a CSV import profile.
type ImportProfile struct {
	ID                uint
	Name              string
	CsvSeparator      string
	SkipRows          int
	DateColumn        string
	DateFormat        string
	DescriptionColumn string
	AmountColumn      string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func dbToProfile(in dbImportProfile) ImportProfile {
	return ImportProfile{
		ID:                in.ID,
		Name:              in.Name,
		CsvSeparator:      in.CsvSeparator,
		SkipRows:          in.SkipRows,
		DateColumn:        in.DateColumn,
		DateFormat:        in.DateFormat,
		DescriptionColumn: in.DescriptionColumn,
		AmountColumn:      in.AmountColumn,
		CreatedAt:         in.CreatedAt,
		UpdatedAt:         in.UpdatedAt,
	}
}

func (s *Store) CreateProfile(ctx context.Context, p ImportProfile) (uint, error) {
	if p.Name == "" {
		return 0, ErrValidation("name cannot be empty")
	}
	if p.DateColumn == "" {
		return 0, ErrValidation("date_column cannot be empty")
	}
	if p.DateFormat == "" {
		return 0, ErrValidation("date_format cannot be empty")
	}
	if p.DescriptionColumn == "" {
		return 0, ErrValidation("description_column cannot be empty")
	}
	if p.AmountColumn == "" {
		return 0, ErrValidation("amount_column cannot be empty")
	}

	csvSep := p.CsvSeparator
	if csvSep == "" {
		csvSep = ","
	}

	row := dbImportProfile{
		Name:              p.Name,
		CsvSeparator:      csvSep,
		SkipRows:          p.SkipRows,
		DateColumn:        p.DateColumn,
		DateFormat:        p.DateFormat,
		DescriptionColumn: p.DescriptionColumn,
		AmountColumn:      p.AmountColumn,
	}

	d := s.db.WithContext(ctx).Create(&row)
	if d.Error != nil {
		return 0, d.Error
	}
	return row.ID, nil
}

func (s *Store) GetProfile(ctx context.Context, id uint) (ImportProfile, error) {
	var row dbImportProfile
	d := s.db.WithContext(ctx).Where("id = ?", id).First(&row)
	if d.Error != nil {
		if errors.Is(d.Error, gorm.ErrRecordNotFound) {
			return ImportProfile{}, ErrProfileNotFound
		}
		return ImportProfile{}, d.Error
	}
	return dbToProfile(row), nil
}

func (s *Store) ListProfiles(ctx context.Context) ([]ImportProfile, error) {
	var rows []dbImportProfile
	d := s.db.WithContext(ctx).Order("id ASC").Find(&rows)
	if d.Error != nil {
		return nil, d.Error
	}

	profiles := make([]ImportProfile, 0, len(rows))
	for _, row := range rows {
		profiles = append(profiles, dbToProfile(row))
	}
	return profiles, nil
}

func (s *Store) UpdateProfile(ctx context.Context, id uint, p ImportProfile) error {
	if p.Name == "" {
		return ErrValidation("name cannot be empty")
	}
	if p.DateColumn == "" {
		return ErrValidation("date_column cannot be empty")
	}
	if p.DateFormat == "" {
		return ErrValidation("date_format cannot be empty")
	}
	if p.DescriptionColumn == "" {
		return ErrValidation("description_column cannot be empty")
	}
	if p.AmountColumn == "" {
		return ErrValidation("amount_column cannot be empty")
	}

	csvSep := p.CsvSeparator
	if csvSep == "" {
		csvSep = ","
	}

	d := s.db.WithContext(ctx).Model(&dbImportProfile{}).Where("id = ?", id).
		Select("Name", "CsvSeparator", "SkipRows", "DateColumn", "DateFormat", "DescriptionColumn", "AmountColumn").
		Updates(dbImportProfile{
			Name:              p.Name,
			CsvSeparator:      csvSep,
			SkipRows:          p.SkipRows,
			DateColumn:        p.DateColumn,
			DateFormat:        p.DateFormat,
			DescriptionColumn: p.DescriptionColumn,
			AmountColumn:      p.AmountColumn,
		})
	if d.Error != nil {
		return d.Error
	}
	if d.RowsAffected == 0 {
		return ErrProfileNotFound
	}
	return nil
}

func (s *Store) DeleteProfile(ctx context.Context, id uint) error {
	d := s.db.WithContext(ctx).Where("id = ?", id).Delete(&dbImportProfile{})
	if d.Error != nil {
		return d.Error
	}
	if d.RowsAffected == 0 {
		return ErrProfileNotFound
	}
	return nil
}
