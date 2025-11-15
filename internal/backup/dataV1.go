package backup

import "time"

const (
	SchemaV1 = "1.0.0"
)

const metaInfoFile = "metainfo.json"

type metaInfoV1 struct {
	Version string   `json:"version"`
	Date    string   `json:"date"`
	Tenants []string `json:"tenants"`
}

const accountProviderFile = "account_provider.json"

type accountProviderV1 struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Tenant      string `json:"tenant"`
}

const accountsFile = "accounts.json"

type accountV1 struct {
	ID                uint   `json:"id"`
	AccountProviderID uint   `json:"providerId"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	Currency          string `json:"currency"`
	Type              string `json:"accountType"`
	Tenant            string `json:"tenant"`
}

const incomeCategoriesFile = "income_categories.json"
const expenseCategoriesFile = "expense_categories.json"

type categoryV1 struct {
	ID          uint   `json:"id"`
	ParentId    uint   `json:"ParentId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Tenant      string `json:"tenant"`
}

const transactionsFile = "transactions.json"

const txTypeIncome = "income"
const txTypeExpense = "expense"
const txTypeTransfer = "transfer"

type TransactionV1 struct {
	Id          uint   `json:"id"`
	Description string `json:"description"`
	// for income/expense
	Amount     float64 `json:"amount"`
	AccountID  uint    `json:"accountId"`
	CategoryID uint    `json:"categoryId"`

	// for transfer
	OriginAmount    float64 `json:"originAmount"`
	OriginAccountID uint    `json:"originAccountId"`
	TargetAmount    float64 `json:"targetAmount"`
	TargetAccountID uint    `json:"targetAccountId"`

	Date   time.Time `json:"date"`
	Type   string    `json:"type"`
	Tenant string    `json:"tenant"`
}
