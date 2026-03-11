package csvimport

import (
	"context"
	"errors"
	"regexp"
	"time"

	"gorm.io/gorm"
)

var (
	ErrCategoryRuleGroupNotFound   = errors.New("category rule group not found")
	ErrCategoryRulePatternNotFound = errors.New("category rule pattern not found")
)

type dbCategoryRuleGroup struct {
	ID         uint                    `gorm:"primarykey"`
	Name       string                  `gorm:"not null"`
	CategoryID uint                    `gorm:"not null;index"`
	Priority   int                     `gorm:"column:position;not null;index"`
	Patterns   []dbCategoryRulePattern `gorm:"foreignKey:GroupID"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type dbCategoryRulePattern struct {
	ID        uint   `gorm:"primarykey"`
	GroupID   uint   `gorm:"not null;index"`
	Pattern   string `gorm:"not null"`
	IsRegex   bool   `gorm:"default:false"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CategoryRuleGroup struct {
	ID         uint
	Name       string
	CategoryID uint
	Priority   int
	Patterns   []CategoryRulePattern
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type CategoryRulePattern struct {
	ID        uint
	GroupID   uint
	Pattern   string
	IsRegex   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func dbToGroup(in dbCategoryRuleGroup) CategoryRuleGroup {
	g := CategoryRuleGroup{
		ID:         in.ID,
		Name:       in.Name,
		CategoryID: in.CategoryID,
		Priority:   in.Priority,
		CreatedAt:  in.CreatedAt,
		UpdatedAt:  in.UpdatedAt,
	}
	for _, p := range in.Patterns {
		g.Patterns = append(g.Patterns, CategoryRulePattern(p))
	}
	return g
}

func (s *Store) CreateCategoryRuleGroup(ctx context.Context, g CategoryRuleGroup) (uint, error) {
	if g.Name == "" {
		return 0, ErrValidation("name cannot be empty")
	}
	if g.CategoryID == 0 {
		return 0, ErrValidation("category_id cannot be zero")
	}

	row := dbCategoryRuleGroup{
		Name:       g.Name,
		CategoryID: g.CategoryID,
		Priority:   g.Priority,
	}
	for _, p := range g.Patterns {
		if p.Pattern == "" {
			return 0, ErrValidation("pattern cannot be empty")
		}
		if p.IsRegex {
			if _, err := regexp.Compile(p.Pattern); err != nil {
				return 0, ErrValidation("invalid regex pattern: " + err.Error())
			}
		}
		row.Patterns = append(row.Patterns, dbCategoryRulePattern{
			Pattern: p.Pattern,
			IsRegex: p.IsRegex,
		})
	}

	d := s.db.WithContext(ctx).Create(&row)
	if d.Error != nil {
		return 0, d.Error
	}
	return row.ID, nil
}

func (s *Store) GetCategoryRuleGroup(ctx context.Context, id uint) (CategoryRuleGroup, error) {
	var row dbCategoryRuleGroup
	d := s.db.WithContext(ctx).Preload("Patterns").Where("id = ?", id).First(&row)
	if d.Error != nil {
		if errors.Is(d.Error, gorm.ErrRecordNotFound) {
			return CategoryRuleGroup{}, ErrCategoryRuleGroupNotFound
		}
		return CategoryRuleGroup{}, d.Error
	}
	return dbToGroup(row), nil
}

func (s *Store) ListCategoryRuleGroups(ctx context.Context) ([]CategoryRuleGroup, error) {
	var rows []dbCategoryRuleGroup
	d := s.db.WithContext(ctx).Preload("Patterns").Order("position ASC, id ASC").Find(&rows)
	if d.Error != nil {
		return nil, d.Error
	}
	groups := make([]CategoryRuleGroup, 0, len(rows))
	for _, row := range rows {
		groups = append(groups, dbToGroup(row))
	}
	return groups, nil
}

func (s *Store) UpdateCategoryRuleGroup(ctx context.Context, id uint, g CategoryRuleGroup) error {
	if g.Name == "" {
		return ErrValidation("name cannot be empty")
	}
	if g.CategoryID == 0 {
		return ErrValidation("category_id cannot be zero")
	}

	d := s.db.WithContext(ctx).Model(&dbCategoryRuleGroup{}).Where("id = ?", id).
		Select("Name", "CategoryID", "Priority").
		Updates(dbCategoryRuleGroup{
			Name:       g.Name,
			CategoryID: g.CategoryID,
			Priority:   g.Priority,
		})
	if d.Error != nil {
		return d.Error
	}
	if d.RowsAffected == 0 {
		return ErrCategoryRuleGroupNotFound
	}
	return nil
}

func (s *Store) DeleteCategoryRuleGroup(ctx context.Context, id uint) error {
	// Delete patterns first
	if err := s.db.WithContext(ctx).Where("group_id = ?", id).Delete(&dbCategoryRulePattern{}).Error; err != nil {
		return err
	}
	d := s.db.WithContext(ctx).Where("id = ?", id).Delete(&dbCategoryRuleGroup{})
	if d.Error != nil {
		return d.Error
	}
	if d.RowsAffected == 0 {
		return ErrCategoryRuleGroupNotFound
	}
	return nil
}

func (s *Store) CreateCategoryRulePattern(ctx context.Context, groupID uint, p CategoryRulePattern) (uint, error) {
	if p.Pattern == "" {
		return 0, ErrValidation("pattern cannot be empty")
	}
	if p.IsRegex {
		if _, err := regexp.Compile(p.Pattern); err != nil {
			return 0, ErrValidation("invalid regex pattern: " + err.Error())
		}
	}
	// Verify group exists
	var count int64
	s.db.WithContext(ctx).Model(&dbCategoryRuleGroup{}).Where("id = ?", groupID).Count(&count)
	if count == 0 {
		return 0, ErrCategoryRuleGroupNotFound
	}

	row := dbCategoryRulePattern{
		GroupID: groupID,
		Pattern: p.Pattern,
		IsRegex: p.IsRegex,
	}
	d := s.db.WithContext(ctx).Create(&row)
	if d.Error != nil {
		return 0, d.Error
	}
	return row.ID, nil
}

func (s *Store) UpdateCategoryRulePattern(ctx context.Context, groupID, patternID uint, p CategoryRulePattern) error {
	if p.Pattern == "" {
		return ErrValidation("pattern cannot be empty")
	}
	if p.IsRegex {
		if _, err := regexp.Compile(p.Pattern); err != nil {
			return ErrValidation("invalid regex pattern: " + err.Error())
		}
	}

	d := s.db.WithContext(ctx).Model(&dbCategoryRulePattern{}).
		Where("id = ? AND group_id = ?", patternID, groupID).
		Select("Pattern", "IsRegex").
		Updates(dbCategoryRulePattern{
			Pattern: p.Pattern,
			IsRegex: p.IsRegex,
		})
	if d.Error != nil {
		return d.Error
	}
	if d.RowsAffected == 0 {
		return ErrCategoryRulePatternNotFound
	}
	return nil
}

func (s *Store) DeleteCategoryRulePattern(ctx context.Context, groupID, patternID uint) error {
	d := s.db.WithContext(ctx).Where("id = ? AND group_id = ?", patternID, groupID).Delete(&dbCategoryRulePattern{})
	if d.Error != nil {
		return d.Error
	}
	if d.RowsAffected == 0 {
		return ErrCategoryRulePatternNotFound
	}
	return nil
}
