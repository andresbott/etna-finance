package accounting

import (
	"fmt"
	closuretree "github.com/go-bumbu/closure-tree"
	"gorm.io/gorm"
)

type Store struct {
	db           *gorm.DB
	categoryTree *closuretree.Tree
}

func NewStore(db *gorm.DB) (*Store, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}

	b := Store{
		db: db,
	}

	stmt := &gorm.Statement{DB: db}
	err := stmt.Parse(&dbAccount{})
	if err != nil {
		return nil, fmt.Errorf("error parsing schema: %w", err)
	}

	err = db.AutoMigrate(&dbAccountProvider{}, &dbAccount{}, &dbTransaction{}, &dbEntry{})
	if err != nil {
		return nil, err
	}

	categoryTree, err := closuretree.New(db, dbCategory{}) // init the closure tree, this includes gorm automigrate
	if err != nil {
		return nil, err
	}
	b.categoryTree = categoryTree

	return &b, nil
}

func NewValidationErr(in string) ErrValidation {
	return ErrValidation(in)
}

type ErrValidation string

func (v ErrValidation) Error() string {
	return string(v)
}
