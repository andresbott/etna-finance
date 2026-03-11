package csvimport

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) (*Store, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}

	err := db.AutoMigrate(&dbImportProfile{}, &dbCategoryRuleGroup{}, &dbCategoryRulePattern{})
	if err != nil {
		return nil, fmt.Errorf("error running auto migrate: %w", err)
	}

	err = migrateOldCategoryRules(db)
	if err != nil {
		return nil, fmt.Errorf("error migrating old category rules: %w", err)
	}

	return &Store{db: db}, nil
}

func migrateOldCategoryRules(db *gorm.DB) error {
	if !db.Migrator().HasTable("db_category_rules") {
		return nil
	}

	type oldRule struct {
		ID         uint
		Pattern    string
		IsRegex    bool
		CategoryID uint
		Priority   int
	}

	var oldRules []oldRule
	if err := db.Table("db_category_rules").Find(&oldRules).Error; err != nil {
		return fmt.Errorf("failed to read old category rules: %w", err)
	}

	for _, old := range oldRules {
		group := dbCategoryRuleGroup{
			Name:       old.Pattern,
			CategoryID: old.CategoryID,
			Priority:   old.Priority,
			Patterns: []dbCategoryRulePattern{
				{Pattern: old.Pattern, IsRegex: old.IsRegex},
			},
		}
		if err := db.Create(&group).Error; err != nil {
			return fmt.Errorf("failed to migrate rule %d: %w", old.ID, err)
		}
	}

	if err := db.Migrator().DropTable("db_category_rules"); err != nil {
		return fmt.Errorf("failed to drop old table: %w", err)
	}
	return nil
}

func (s *Store) WipeData(ctx context.Context) error {
	tables := []string{"db_category_rule_patterns", "db_category_rule_groups", "db_import_profiles"}
	for _, table := range tables {
		if err := s.db.WithContext(ctx).Table(table).Where("1 = 1").Delete(nil).Error; err != nil {
			return fmt.Errorf("failed to delete data in table '%s': %w", table, err)
		}
	}
	return nil
}
