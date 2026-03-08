package csvimport

import (
	"context"
	"errors"
	"regexp"
	"time"

	"gorm.io/gorm"
)

var ErrCategoryRuleNotFound = errors.New("category rule not found")

// dbCategoryRule is the DB internal representation of a CategoryRule.
type dbCategoryRule struct {
	ID         uint   `gorm:"primarykey"`
	Pattern    string `gorm:"not null"`
	IsRegex    bool   `gorm:"default:false"`
	CategoryID uint   `gorm:"not null;index"`
	Position   int    `gorm:"not null;index"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// CategoryRule is the public-facing representation of a category rule.
type CategoryRule struct {
	ID         uint
	Pattern    string
	IsRegex    bool
	CategoryID uint
	Position   int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func dbToCategoryRule(in dbCategoryRule) CategoryRule {
	return CategoryRule(in)
}

func (s *Store) CreateCategoryRule(ctx context.Context, r CategoryRule) (uint, error) {
	if r.Pattern == "" {
		return 0, ErrValidation("pattern cannot be empty")
	}
	if r.CategoryID == 0 {
		return 0, ErrValidation("category_id cannot be zero")
	}
	if r.IsRegex {
		if _, err := regexp.Compile(r.Pattern); err != nil {
			return 0, ErrValidation("invalid regex pattern: " + err.Error())
		}
	}

	row := dbCategoryRule{
		Pattern:    r.Pattern,
		IsRegex:    r.IsRegex,
		CategoryID: r.CategoryID,
		Position:   r.Position,
	}

	d := s.db.WithContext(ctx).Create(&row)
	if d.Error != nil {
		return 0, d.Error
	}
	return row.ID, nil
}

func (s *Store) GetCategoryRule(ctx context.Context, id uint) (CategoryRule, error) {
	var row dbCategoryRule
	d := s.db.WithContext(ctx).Where("id = ?", id).First(&row)
	if d.Error != nil {
		if errors.Is(d.Error, gorm.ErrRecordNotFound) {
			return CategoryRule{}, ErrCategoryRuleNotFound
		}
		return CategoryRule{}, d.Error
	}
	return dbToCategoryRule(row), nil
}

func (s *Store) ListCategoryRules(ctx context.Context) ([]CategoryRule, error) {
	var rows []dbCategoryRule
	d := s.db.WithContext(ctx).Order("position ASC, id ASC").Find(&rows)
	if d.Error != nil {
		return nil, d.Error
	}

	rules := make([]CategoryRule, 0, len(rows))
	for _, row := range rows {
		rules = append(rules, dbToCategoryRule(row))
	}
	return rules, nil
}

func (s *Store) UpdateCategoryRule(ctx context.Context, id uint, r CategoryRule) error {
	if r.Pattern == "" {
		return ErrValidation("pattern cannot be empty")
	}
	if r.CategoryID == 0 {
		return ErrValidation("category_id cannot be zero")
	}
	if r.IsRegex {
		if _, err := regexp.Compile(r.Pattern); err != nil {
			return ErrValidation("invalid regex pattern: " + err.Error())
		}
	}

	d := s.db.WithContext(ctx).Model(&dbCategoryRule{}).Where("id = ?", id).
		Select("Pattern", "IsRegex", "CategoryID", "Position").
		Updates(dbCategoryRule{
			Pattern:    r.Pattern,
			IsRegex:    r.IsRegex,
			CategoryID: r.CategoryID,
			Position:   r.Position,
		})
	if d.Error != nil {
		return d.Error
	}
	if d.RowsAffected == 0 {
		return ErrCategoryRuleNotFound
	}
	return nil
}

func (s *Store) DeleteCategoryRule(ctx context.Context, id uint) error {
	d := s.db.WithContext(ctx).Where("id = ?", id).Delete(&dbCategoryRule{})
	if d.Error != nil {
		return d.Error
	}
	if d.RowsAffected == 0 {
		return ErrCategoryRuleNotFound
	}
	return nil
}
