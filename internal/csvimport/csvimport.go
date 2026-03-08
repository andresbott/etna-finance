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

	err := db.AutoMigrate(&dbImportProfile{}, &dbCategoryRule{})
	if err != nil {
		return nil, fmt.Errorf("error running auto migrate: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) WipeData(ctx context.Context) error {
	tables := []string{"db_category_rules", "db_import_profiles"}
	for _, table := range tables {
		if err := s.db.WithContext(ctx).Table(table).Where("1 = 1").Delete(nil).Error; err != nil {
			return fmt.Errorf("failed to delete data in table '%s': %w", table, err)
		}
	}
	return nil
}
