package finance

import (
	"fmt"
	closuretree "github.com/go-bumbu/closure-tree"
	"gorm.io/gorm"
)

type Store struct {
	db          *gorm.DB
	incomeTree  *closuretree.Tree
	expenseTree *closuretree.Tree

	AccountColNames         map[string]string // hold a map of struct field names to db column names
	AccountProviderColNames map[string]string // hold a map of struct field names to db column names
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
	//b.accountsTblName = stmt.Schema.Table

	columnFieldMap := make(map[string]string)
	for _, field := range stmt.Schema.Fields {
		columnFieldMap[field.Name] = field.DBName
	}
	b.AccountColNames = columnFieldMap

	err = stmt.Parse(&dbAccountProvider{})
	if err != nil {
		return nil, fmt.Errorf("error parsing schema: %w", err)
	}

	accountProviderMap := make(map[string]string)
	for _, field := range stmt.Schema.Fields {
		accountProviderMap[field.Name] = field.DBName
	}
	b.AccountProviderColNames = accountProviderMap

	err = db.AutoMigrate(&dbAccount{}, &dbAccountProvider{}, &dbEntry{})
	if err != nil {
		return nil, err
	}

	incomeTree, err := closuretree.New(db, IncomeCategory{}) // init the closure tree, this includes gorm automigrate
	if err != nil {
		return nil, err
	}
	b.incomeTree = incomeTree

	expenseTree, err := closuretree.New(db, ExpenseCategory{}) // init the closure tree, this includes gorm automigrate
	if err != nil {
		return nil, err
	}
	b.expenseTree = expenseTree

	return &b, nil
}
