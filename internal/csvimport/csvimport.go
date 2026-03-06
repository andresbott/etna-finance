package csvimport

import (
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
