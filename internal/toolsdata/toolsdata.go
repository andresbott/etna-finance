package toolsdata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ErrValidation represents a validation error for case study data.
type ErrValidation string

func (v ErrValidation) Error() string {
	return string(v)
}

var ErrCaseStudyNotFound = errors.New("case study not found")

// dbToolsData is the DB-internal representation of a case study.
type dbToolsData struct {
	ID                   uint   `gorm:"primarykey"`
	ToolType             string `gorm:"uniqueIndex:idx_tool_name;not null"`
	Name                 string `gorm:"uniqueIndex:idx_tool_name;not null"`
	Description          string
	ExpectedAnnualReturn float64
	Params               string // JSON string stored as TEXT in SQLite
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// CaseStudy is the public-facing representation of a tool case study.
type CaseStudy struct {
	ID                   uint
	ToolType             string
	Name                 string
	Description          string
	ExpectedAnnualReturn float64
	Params               json.RawMessage
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func dbToCaseStudy(in dbToolsData) CaseStudy {
	return CaseStudy{
		ID:                   in.ID,
		ToolType:             in.ToolType,
		Name:                 in.Name,
		Description:          in.Description,
		ExpectedAnnualReturn: in.ExpectedAnnualReturn,
		Params:               json.RawMessage(in.Params),
		CreatedAt:            in.CreatedAt,
		UpdatedAt:            in.UpdatedAt,
	}
}

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) (*Store, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}
	err := db.AutoMigrate(&dbToolsData{})
	if err != nil {
		return nil, fmt.Errorf("error running auto migrate: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Create(ctx context.Context, cs CaseStudy) (CaseStudy, error) {
	if cs.ToolType == "" {
		return CaseStudy{}, ErrValidation("tool_type cannot be empty")
	}
	if cs.Name == "" {
		return CaseStudy{}, ErrValidation("name cannot be empty")
	}

	row := dbToolsData{
		ToolType:             cs.ToolType,
		Name:                 cs.Name,
		Description:          cs.Description,
		ExpectedAnnualReturn: cs.ExpectedAnnualReturn,
		Params:               string(cs.Params),
	}
	d := s.db.WithContext(ctx).Create(&row)
	if d.Error != nil {
		return CaseStudy{}, d.Error
	}
	return dbToCaseStudy(row), nil
}

func (s *Store) Get(ctx context.Context, toolType string, id uint) (CaseStudy, error) {
	var row dbToolsData
	d := s.db.WithContext(ctx).Where("id = ? AND tool_type = ?", id, toolType).First(&row)
	if d.Error != nil {
		if errors.Is(d.Error, gorm.ErrRecordNotFound) {
			return CaseStudy{}, ErrCaseStudyNotFound
		}
		return CaseStudy{}, d.Error
	}
	return dbToCaseStudy(row), nil
}

func (s *Store) List(ctx context.Context, toolType string) ([]CaseStudy, error) {
	var rows []dbToolsData
	d := s.db.WithContext(ctx).Where("tool_type = ?", toolType).Order("id ASC").Find(&rows)
	if d.Error != nil {
		return nil, d.Error
	}
	result := make([]CaseStudy, 0, len(rows))
	for _, row := range rows {
		result = append(result, dbToCaseStudy(row))
	}
	return result, nil
}

func (s *Store) Update(ctx context.Context, toolType string, id uint, cs CaseStudy) (CaseStudy, error) {
	if cs.Name == "" {
		return CaseStudy{}, ErrValidation("name cannot be empty")
	}

	d := s.db.WithContext(ctx).Model(&dbToolsData{}).Where("id = ? AND tool_type = ?", id, toolType).
		Select("Name", "Description", "ExpectedAnnualReturn", "Params").
		Updates(dbToolsData{
			Name:                 cs.Name,
			Description:          cs.Description,
			ExpectedAnnualReturn: cs.ExpectedAnnualReturn,
			Params:               string(cs.Params),
		})
	if d.Error != nil {
		return CaseStudy{}, d.Error
	}
	if d.RowsAffected == 0 {
		return CaseStudy{}, ErrCaseStudyNotFound
	}
	return s.Get(ctx, toolType, id)
}

func (s *Store) Delete(ctx context.Context, toolType string, id uint) error {
	d := s.db.WithContext(ctx).Where("id = ? AND tool_type = ?", id, toolType).Delete(&dbToolsData{})
	if d.Error != nil {
		return d.Error
	}
	if d.RowsAffected == 0 {
		return ErrCaseStudyNotFound
	}
	return nil
}
