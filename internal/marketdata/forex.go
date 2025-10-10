package marketdata

import (
	"golang.org/x/text/currency"
	"time"
)

func (t Tracker) Forex() {

}

// CurrencyPair represents a unique base/quote combination.
type dbCurrencyPair struct {
	ID    uint          `gorm:"primaryKey;autoIncrement"`
	Base  currency.Unit `gorm:"not null;index,unique"`
	Quote currency.Unit `gorm:"not null;index,unique"`
	Rates []dbRate      `gorm:"foreignKey:PairID;"`
}

// Rate stores the exchange rate for a pair at a specific timestamp.
type dbRate struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	PairID    uint      `gorm:"not null;index"`
	Timestamp time.Time `gorm:"index;not null"`
	Value     float64   `gorm:"not null"`
}
