package main

import (
	"fmt"
	"time"
)

// Entry represents a transaction entry (income, expense, or transfer)
type Entry struct {
	ID          uint      `json:"id,omitempty"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	Type        string    `json:"type"`

	StockAmount float64 `json:"StockAmount"`

	// used for income / expense
	Amount     float64 `json:"Amount"`
	AccountId  uint    `json:"accountId"`
	CategoryId uint    `json:"categoryId"`

	// used for transfers
	TargetAmount    float64 `json:"targetAmount"`
	TargetAccountID uint    `json:"targetAccountId"`
	OriginAmount    float64 `json:"originAmount"`
	OriginAccountID uint    `json:"originAccountId"`

	// used for stock buy / sell
	InstrumentID        uint    `json:"instrumentId,omitempty"`
	Quantity            float64 `json:"quantity,omitempty"`
	TotalAmount         float64 `json:"totalAmount,omitempty"`
	InvestmentAccountID uint    `json:"investmentAccountId,omitempty"`
	CashAccountID       uint    `json:"cashAccountId,omitempty"`
}

// stockEntryPayload is sent to the API for stock transactions (date as string).
type stockEntryPayload struct {
	Description         string  `json:"description"`
	Date                string  `json:"date"` // "2006-01-02"
	Type                string  `json:"type"`
	InstrumentID        uint    `json:"instrumentId"`
	Quantity            float64 `json:"quantity"`
	TotalAmount         float64 `json:"totalAmount,omitempty"`
	StockAmount         float64 `json:"StockAmount,omitempty"`
	InvestmentAccountID uint    `json:"investmentAccountId,omitempty"`
	CashAccountID       uint    `json:"cashAccountId,omitempty"`
	AccountId           uint    `json:"accountId,omitempty"`
	OriginAccountID     uint    `json:"originAccountId,omitempty"`
	TargetAccountID     uint    `json:"targetAccountId,omitempty"`
}

// EntryDefinition represents an entry definition with string references
type EntryDefinition struct {
	Description  string  `json:"description"`
	DaysDelta    int     `json:"daysDelta"`        // Days to add/subtract from current time
	Type         string  `json:"type"`             // "income", "expense", or "transfer"
	Amount       float64 `json:"amount,omitempty"` // For income/expense
	ProviderName string  `json:"providerName,omitempty"`
	AccountName  string  `json:"accountName,omitempty"`
	CategoryName string  `json:"categoryName,omitempty"`

	// For transfers
	OriginAmount   float64 `json:"originAmount,omitempty"`
	OriginProvider string  `json:"originProvider,omitempty"`
	OriginAccount  string  `json:"originAccount,omitempty"`
	TargetAmount   float64 `json:"targetAmount,omitempty"`
	TargetProvider string  `json:"targetProvider,omitempty"`
	TargetAccount  string  `json:"targetAccount,omitempty"`
}

// createEntry sends a POST request to create an entry
func createEntry(baseURL string, entry Entry) (uint, error) {
	url := fmt.Sprintf("%s/api/v0/fin/entries", baseURL)

	var entryResp Entry
	err := postJSON(url, entry, &entryResp)
	if err != nil {
		return 0, err
	}
	return entryResp.ID, nil
}

// deltaTime creates a time by adding days to the current time
func deltaTime(daysDelta int) time.Time {
	return time.Now().AddDate(0, 0, daysDelta)
}

// findCategoryID searches for a category by name in the given category map and returns the category id
func findCategoryID(categoryName string, categoryMap map[string]uint) (uint, error) {
	if id, exists := categoryMap[categoryName]; exists {
		return id, nil
	}
	return 0, fmt.Errorf("category '%s' not found", categoryName)
}

// convertEntryDefinitionToEntry converts an EntryDefinition to an Entry, resolving string references to IDs
func convertEntryDefinitionToEntry(def EntryDefinition, expenseCategoryMap, incomeCategoryMap map[string]uint) (Entry, error) {
	entry := Entry{
		Description: def.Description,
		Date:        deltaTime(def.DaysDelta),
		Type:        def.Type,
	}

	switch def.Type {
	case "income":
		accountID, err := findAccountID(def.ProviderName, def.AccountName)
		if err != nil {
			return Entry{}, fmt.Errorf("failed to find account for income entry: %v", err)
		}

		categoryID, err := findCategoryID(def.CategoryName, incomeCategoryMap)
		if err != nil {
			return Entry{}, fmt.Errorf("failed to find income category: %v", err)
		}

		entry.Amount = def.Amount
		entry.AccountId = accountID
		entry.CategoryId = categoryID

	case "expense":
		accountID, err := findAccountID(def.ProviderName, def.AccountName)
		if err != nil {
			return Entry{}, fmt.Errorf("failed to find account for expense entry: %v", err)
		}

		categoryID, err := findCategoryID(def.CategoryName, expenseCategoryMap)
		if err != nil {
			return Entry{}, fmt.Errorf("failed to find expense category: %v", err)
		}

		entry.Amount = def.Amount
		entry.AccountId = accountID
		entry.CategoryId = categoryID

	case "transfer":
		originAccountID, err := findAccountID(def.OriginProvider, def.OriginAccount)
		if err != nil {
			return Entry{}, fmt.Errorf("failed to find origin account for transfer: %v", err)
		}

		targetAccountID, err := findAccountID(def.TargetProvider, def.TargetAccount)
		if err != nil {
			return Entry{}, fmt.Errorf("failed to find target account for transfer: %v", err)
		}

		entry.OriginAmount = def.OriginAmount
		entry.OriginAccountID = originAccountID
		entry.TargetAmount = def.TargetAmount
		entry.TargetAccountID = targetAccountID

	default:
		return Entry{}, fmt.Errorf("unknown entry type: %s", def.Type)
	}

	return entry, nil
}

// convertStockEntryDefinitionToPayload converts a StockEntryDefinition to the API payload, resolving names to IDs.
func convertStockEntryDefinitionToPayload(def StockEntryDefinition) (stockEntryPayload, error) {
	date := deltaTime(def.DaysDelta)
	payload := stockEntryPayload{
		Description: def.Description,
		Date:        date.Format("2006-01-02"),
		Type:        def.Type,
		Quantity:    def.Quantity,
	}

	instrumentID, err := findInstrumentID(def.Instrument)
	if err != nil {
		return stockEntryPayload{}, fmt.Errorf("instrument: %w", err)
	}
	payload.InstrumentID = instrumentID

	switch def.Type {
	case "stockbuy", "stocksell":
		invID, err := findAccountID(def.InvestmentProvider, def.InvestmentAccount)
		if err != nil {
			return stockEntryPayload{}, fmt.Errorf("investment account: %w", err)
		}
		cashID, err := findAccountID(def.CashProvider, def.CashAccount)
		if err != nil {
			return stockEntryPayload{}, fmt.Errorf("cash account: %w", err)
		}
		payload.InvestmentAccountID = invID
		payload.CashAccountID = cashID
		payload.TotalAmount = def.TotalAmount
		payload.StockAmount = def.StockAmount
	case "stockgrant":
		accID, err := findAccountID(def.InvestmentProvider, def.InvestmentAccount)
		if err != nil {
			return stockEntryPayload{}, fmt.Errorf("account: %w", err)
		}
		payload.AccountId = accID
	case "stocktransfer":
		originID, err := findAccountID(def.OriginProvider, def.OriginAccount)
		if err != nil {
			return stockEntryPayload{}, fmt.Errorf("origin account: %w", err)
		}
		targetID, err := findAccountID(def.TargetProvider, def.TargetAccount)
		if err != nil {
			return stockEntryPayload{}, fmt.Errorf("target account: %w", err)
		}
		payload.OriginAccountID = originID
		payload.TargetAccountID = targetID
	default:
		return stockEntryPayload{}, fmt.Errorf("unknown stock entry type: %s", def.Type)
	}

	return payload, nil
}

type stockEntryCreateResponse struct {
	Id uint `json:"id"`
}

// createStockEntry POSTs a stock transaction to the entries API and returns the created id.
func createStockEntry(baseURL string, payload stockEntryPayload) (uint, error) {
	url := fmt.Sprintf("%s/api/v0/fin/entries", baseURL)
	var resp stockEntryCreateResponse
	err := postJSON(url, payload, &resp)
	if err != nil {
		return 0, err
	}
	return resp.Id, nil
}
