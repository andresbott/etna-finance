package csvimport

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/andresbott/etna/internal/accounting"
	"github.com/andresbott/etna/internal/csvimport"
)

// ReapplyRow represents a single transaction row in the reapply preview response.
type ReapplyRow struct {
	TransactionID    uint    `json:"transactionId"`
	TransactionType  string  `json:"transactionType"`
	Description      string  `json:"description"`
	Date             string  `json:"date"`
	Amount           float64 `json:"amount"`
	AccountID        uint    `json:"accountId"`
	AccountName      string  `json:"accountName"`
	CurrentCategoryID   uint   `json:"currentCategoryId"`
	CurrentCategoryName string `json:"currentCategoryName"`
	NewCategoryID       uint   `json:"newCategoryId"`
	NewCategoryName     string `json:"newCategoryName"`
	Changed             bool   `json:"changed"`
}

// ReapplyPreview returns an http.Handler that previews the effect of re-applying
// category matching rules to all existing income and expense transactions.
func (h *ImportHandler) ReapplyPreview() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Load category rules
		rules, err := h.CsvStore.ListCategoryRules(ctx)
		if err != nil {
			http.Error(w, "unable to list category rules: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if len(rules) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("[]"))
			return
		}

		// Load account map
		accountMap, err := h.FinStore.ListAccountsMap(ctx)
		if err != nil {
			http.Error(w, "unable to list accounts: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Build category name map
		catNames := make(map[uint]string)
		incomeCats, err := h.FinStore.ListDescendantCategories(ctx, 0, -1, accounting.IncomeCategory)
		if err != nil {
			http.Error(w, "unable to list income categories: "+err.Error(), http.StatusInternalServerError)
			return
		}
		for _, c := range incomeCats {
			catNames[c.Id] = c.Name
		}
		expenseCats, err := h.FinStore.ListDescendantCategories(ctx, 0, -1, accounting.ExpenseCategory)
		if err != nil {
			http.Error(w, "unable to list expense categories: "+err.Error(), http.StatusInternalServerError)
			return
		}
		for _, c := range expenseCats {
			catNames[c.Id] = c.Name
		}

		// Paginate through all income + expense transactions
		var rows []ReapplyRow
		for page := 1; ; page++ {
			opts := accounting.ListOpts{
				StartDate: time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC),
				Types:     []accounting.TxType{accounting.IncomeTransaction, accounting.ExpenseTransaction},
				Limit:     accounting.MaxSearchResults,
				Page:      page,
			}

			txs, _, err := h.FinStore.ListTransactions(ctx, opts)
			if err != nil {
				http.Error(w, "unable to list transactions: "+err.Error(), http.StatusInternalServerError)
				return
			}
			if len(txs) == 0 {
				break
			}

			for _, tx := range txs {
				switch item := tx.(type) {
				case accounting.Income:
					newCatID := csvimport.MatchCategory(item.Description, rules)
					if newCatID == 0 {
						continue
					}
					rows = append(rows, ReapplyRow{
						TransactionID:       item.Id,
						TransactionType:     "income",
						Description:         item.Description,
						Date:                item.Date.Format("2006-01-02"),
						Amount:              item.Amount,
						AccountID:           item.AccountID,
						AccountName:         accountMap[item.AccountID].Name,
						CurrentCategoryID:   item.CategoryID,
						CurrentCategoryName: catNames[item.CategoryID],
						NewCategoryID:       newCatID,
						NewCategoryName:     catNames[newCatID],
						Changed:             item.CategoryID != newCatID,
					})
				case accounting.Expense:
					newCatID := csvimport.MatchCategory(item.Description, rules)
					if newCatID == 0 {
						continue
					}
					rows = append(rows, ReapplyRow{
						TransactionID:       item.Id,
						TransactionType:     "expense",
						Description:         item.Description,
						Date:                item.Date.Format("2006-01-02"),
						Amount:              item.Amount,
						AccountID:           item.AccountID,
						AccountName:         accountMap[item.AccountID].Name,
						CurrentCategoryID:   item.CategoryID,
						CurrentCategoryName: catNames[item.CategoryID],
						NewCategoryID:       newCatID,
						NewCategoryName:     catNames[newCatID],
						Changed:             item.CategoryID != newCatID,
					})
				}
			}
		}

		if rows == nil {
			rows = []ReapplyRow{}
		}

		respJSON, err := json.Marshal(rows)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}

// reapplySubmitItem represents a single transaction category update request.
type reapplySubmitItem struct {
	TransactionID   uint   `json:"transactionId"`
	TransactionType string `json:"transactionType"`
	NewCategoryID   uint   `json:"newCategoryId"`
}

// ReapplySubmit returns an http.Handler that applies category changes to transactions.
func (h *ImportHandler) ReapplySubmit() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var items []reapplySubmitItem
		if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
			http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		for _, item := range items {
			catID := item.NewCategoryID
			var update accounting.TransactionUpdate
			switch item.TransactionType {
			case "expense":
				update = accounting.ExpenseUpdate{CategoryID: &catID}
			case "income":
				update = accounting.IncomeUpdate{CategoryID: &catID}
			default:
				http.Error(w, fmt.Sprintf("unsupported transaction type: %s", item.TransactionType), http.StatusBadRequest)
				return
			}

			if err := h.FinStore.UpdateTransaction(ctx, update, item.TransactionID); err != nil {
				var valErr accounting.ErrValidation
				if errors.As(err, &valErr) {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp, _ := json.Marshal(map[string]int{"updated": len(items)})
		_, _ = w.Write(resp)
	})
}
