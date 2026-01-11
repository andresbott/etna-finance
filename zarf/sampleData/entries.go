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
	CategoryId uint    `json:"CategoryId"`

	// used for transfers
	TargetAmount    float64 `json:"targetAmount"`
	TargetAccountID uint    `json:"targetAccountId"`
	OriginAmount    float64 `json:"originAmount"`
	OriginAccountID uint    `json:"originAccountId"`
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
func findCategoryID(categoryName string, categoryMap map[string]int) (uint, error) {
	if id, exists := categoryMap[categoryName]; exists {
		//nolint: gosec // only sample data
		return uint(id), nil
	}
	return 0, fmt.Errorf("category '%s' not found", categoryName)
}

// convertEntryDefinitionToEntry converts an EntryDefinition to an Entry, resolving string references to IDs
func convertEntryDefinitionToEntry(def EntryDefinition, expenseCategoryMap, incomeCategoryMap map[string]int) (Entry, error) {
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
		entry.AccountId = uint(accountID) //nolint: gosec // only sample data
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
		entry.AccountId = uint(accountID) //nolint: gosec // only sample data
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
		entry.OriginAccountID = uint(originAccountID) //nolint: gosec // only sample data
		entry.TargetAmount = def.TargetAmount
		entry.TargetAccountID = uint(targetAccountID) //nolint: gosec // only sample data

	default:
		return Entry{}, fmt.Errorf("unknown entry type: %s", def.Type)
	}

	return entry, nil
}
