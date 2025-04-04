package finance

import (
	"fmt"
	closuretree "github.com/go-bumbu/closure-tree"
	"gorm.io/gorm"
)

type Store struct {
	db              *gorm.DB
	tree            *closuretree.Tree
	tblName         string            // hold the table name
	AccountColNames map[string]string // hold a map of struct field names to db column names
}

type ValidationErr string

func (v ValidationErr) Error() string {
	return string(v)
}

func New(db *gorm.DB) (*Store, error) {
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
	b.tblName = stmt.Schema.Table

	columnFieldMap := make(map[string]string)
	for _, field := range stmt.Schema.Fields {
		columnFieldMap[field.Name] = field.DBName
	}
	b.AccountColNames = columnFieldMap

	err = db.AutoMigrate(&dbAccount{}, &dbEntry{})
	if err != nil {
		return nil, err
	}

	tree, err := closuretree.New(db, Category{}) // init the closure tree, this includes gorm automigrate
	if err != nil {
		return nil, err
	}
	b.tree = tree

	return &b, nil
}
