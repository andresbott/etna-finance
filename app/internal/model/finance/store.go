package finance

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
)

type Store struct {
	db       *gorm.DB
	tblName  string            // hold the table name
	colNames map[string]string // hold a map of struct field names to db column names
}

var NotFoundErr = errors.New("bookmark not found")

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
	b.colNames = columnFieldMap

	err = db.AutoMigrate(&dbAccount{})
	if err != nil {
		return nil, err
	}
	return &b, nil
}
