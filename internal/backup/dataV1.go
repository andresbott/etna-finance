package backup

import (
	"encoding/json"
	"time"
)

const (
	SchemaV1 = "1.0.0"
)

const metaInfoFile = "metainfo.json"

type metaInfoV1 struct {
	Version string `json:"version"`
	Date    string `json:"date"`
}

const accountProviderFile = "account_provider.json"

type accountProviderV1 struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

const accountsFile = "accounts.json"

type accountV1 struct {
	ID                uint   `json:"id"`
	AccountProviderID uint   `json:"providerId"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	Icon              string `json:"icon"`
	Notes             string `json:"notes,omitempty"`
	Currency          string `json:"currency"`
	Type              string `json:"accountType"`
	ImportProfileID   uint   `json:"importProfileId"`
}

const incomeCategoriesFile = "income_categories.json"
const expenseCategoriesFile = "expense_categories.json"

type categoryV1 struct {
	ID          uint   `json:"id"`
	ParentId    uint   `json:"ParentId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

const transactionsFile = "transactions.json"

const txTypeIncome = "income"
const txTypeExpense = "expense"
const txTypeTransfer = "transfer"
const txTypeStockBuy = "stockbuy"
const txTypeStockSell = "stocksell"
const txTypeStockGrant = "stockgrant"
const txTypeStockTransfer = "stocktransfer"

type TransactionV1 struct {
	Id          uint   `json:"id"`
	Description string `json:"description"`
	Notes       string `json:"notes,omitempty"`
	// for income/expense
	Amount     float64 `json:"amount"`
	AccountID  uint    `json:"accountId"`
	CategoryID uint    `json:"categoryId"`

	// for transfer
	OriginAmount    float64 `json:"originAmount"`
	OriginAccountID uint    `json:"originAccountId"`
	TargetAmount    float64 `json:"targetAmount"`
	TargetAccountID uint    `json:"targetAccountId"`

	// for stock buy/sell
	InstrumentID        uint    `json:"instrumentId,omitempty"`
	Quantity            float64 `json:"quantity,omitempty"`
	TotalAmount         float64 `json:"totalAmount,omitempty"`
	StockAmount         float64 `json:"stockAmount,omitempty"`
	InvestmentAccountID uint    `json:"investmentAccountId,omitempty"`
	CashAccountID       uint    `json:"cashAccountId,omitempty"`
	Fees                float64 `json:"fees,omitempty"`

	// for stock grant
	FairMarketValue float64 `json:"fairMarketValue,omitempty"`

	// for stock transfer
	SourceAccountID uint `json:"sourceAccountId,omitempty"`

	Date time.Time `json:"date"`
	Type string    `json:"type"`
}

const instrumentsFile = "instruments.json"
const priceHistoryFile = "price_history.json"
const fxRatesFile = "fx_rates.json"
const importProfilesFile = "import_profiles.json"
const categoryRulesFile = "category_rules.json"

type instrumentV1 struct {
	ID                   uint   `json:"id"`
	InstrumentProviderID uint   `json:"instrumentProviderId"`
	Symbol               string `json:"symbol"`
	Name                 string `json:"name"`
	Currency             string `json:"currency"`
}

type priceRecordV1 struct {
	Symbol string    `json:"symbol"`
	Time   time.Time `json:"time"`
	Price  float64   `json:"price"`
}

type fxRateRecordV1 struct {
	Main      string    `json:"main"`
	Secondary string    `json:"secondary"`
	Time      time.Time `json:"time"`
	Rate      float64   `json:"rate"`
}

type importProfileV1 struct {
	ID                uint   `json:"id"`
	Name              string `json:"name"`
	CsvSeparator      string `json:"csvSeparator"`
	SkipRows          int    `json:"skipRows"`
	DateColumn        string `json:"dateColumn"`
	DateFormat        string `json:"dateFormat"`
	DescriptionColumn string `json:"descriptionColumn"`
	AmountColumn      string `json:"amountColumn"`
	AmountMode        string `json:"amountMode"`
	CreditColumn      string `json:"creditColumn"`
	DebitColumn       string `json:"debitColumn"`
}

type categoryRuleGroupV1 struct {
	ID         uint                    `json:"id"`
	Name       string                  `json:"name"`
	CategoryID uint                    `json:"categoryId"`
	Priority   int                     `json:"position"`
	Patterns   []categoryRulePatternV1 `json:"patterns"`
}

type categoryRulePatternV1 struct {
	ID      uint   `json:"id"`
	Pattern string `json:"pattern"`
	IsRegex bool   `json:"isRegex"`
}

const caseStudiesFile = "case_studies.json"

type caseStudyV1 struct {
	ID                   uint            `json:"id"`
	ToolType             string          `json:"toolType"`
	Name                 string          `json:"name"`
	Description          string          `json:"description"`
	ExpectedAnnualReturn float64         `json:"expectedAnnualReturn"`
	Params               json.RawMessage `json:"params"`
}
