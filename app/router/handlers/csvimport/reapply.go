package csvimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/andresbott/etna/internal/accounting"
	"github.com/andresbott/etna/internal/csvimport"
)

// ReapplyRow represents a single transaction row in the reapply preview response.
type ReapplyRow struct {
	TransactionID       uint    `json:"transactionId"`
	TransactionType     string  `json:"transactionType"`
	Description         string  `json:"description"`
	Date                string  `json:"date"`
	Amount              float64 `json:"amount"`
	AccountID           uint    `json:"accountId"`
	AccountName         string  `json:"accountName"`
	CurrentCategoryID   uint    `json:"currentCategoryId"`
	CurrentCategoryName string  `json:"currentCategoryName"`
	NewCategoryID       uint    `json:"newCategoryId"`
	NewCategoryName     string  `json:"newCategoryName"`
	Changed             bool    `json:"changed"`
}

type categoryRulesPreviewRequest struct {
	AdhocRule *adhocRuleInput `json:"adhocRule"`
}

type adhocRuleInput struct {
	CategoryID uint   `json:"categoryId"`
	Pattern    string `json:"pattern"`
	IsRegex    bool   `json:"isRegex"`
}

// resolveRuleGroupsForPreview returns rule groups to use for preview.
// For adhoc rules it validates and returns a synthetic group.
// For stored rules it loads them from the database.
// Returns a non-zero HTTP status code alongside the error on failure.
func (h *ImportHandler) resolveRuleGroupsForPreview(ctx context.Context, req categoryRulesPreviewRequest) ([]csvimport.CategoryRuleGroup, int, error) {
	if req.AdhocRule == nil {
		groups, err := h.CsvStore.ListCategoryRuleGroups(ctx)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("unable to list category rule groups: %w", err)
		}
		return groups, 0, nil
	}
	if req.AdhocRule.CategoryID == 0 {
		return nil, http.StatusBadRequest, errors.New("adhocRule.categoryId must be non-zero")
	}
	if req.AdhocRule.Pattern == "" {
		return nil, http.StatusBadRequest, errors.New("adhocRule.pattern must not be empty")
	}
	if req.AdhocRule.IsRegex {
		if _, err := regexp.Compile(req.AdhocRule.Pattern); err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("adhocRule.pattern is not a valid regex: %w", err)
		}
	}
	return []csvimport.CategoryRuleGroup{{
		CategoryID: req.AdhocRule.CategoryID,
		Priority:   0,
		Patterns: []csvimport.CategoryRulePattern{{
			Pattern: req.AdhocRule.Pattern,
			IsRegex: req.AdhocRule.IsRegex,
		}},
	}}, 0, nil
}

// buildCategoryNamesMap returns a map of category ID to name for all income and expense categories.
func (h *ImportHandler) buildCategoryNamesMap(ctx context.Context) (map[uint]string, error) {
	catNames := make(map[uint]string)
	incomeCats, err := h.FinStore.ListDescendantCategories(ctx, 0, -1, accounting.IncomeCategory)
	if err != nil {
		return nil, fmt.Errorf("unable to list income categories: %w", err)
	}
	for _, c := range incomeCats {
		catNames[c.Id] = c.Name
	}
	expenseCats, err := h.FinStore.ListDescendantCategories(ctx, 0, -1, accounting.ExpenseCategory)
	if err != nil {
		return nil, fmt.Errorf("unable to list expense categories: %w", err)
	}
	for _, c := range expenseCats {
		catNames[c.Id] = c.Name
	}
	return catNames, nil
}

// collectPreviewRows paginates through all income and expense transactions and returns
// rows where the given rule groups would assign a different category.
func (h *ImportHandler) collectPreviewRows(ctx context.Context, groups []csvimport.CategoryRuleGroup, accountMap map[uint]accounting.Account, catNames map[uint]string) ([]ReapplyRow, error) {
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
			return nil, fmt.Errorf("unable to list transactions: %w", err)
		}
		if len(txs) == 0 {
			break
		}
		for _, tx := range txs {
			switch item := tx.(type) {
			case accounting.Income:
				newCatID := csvimport.MatchCategory(item.Description, groups)
				if newCatID != 0 && newCatID != item.CategoryID {
					rows = append(rows, ReapplyRow{
						TransactionID: item.Id, TransactionType: "income",
						Description: item.Description, Date: item.Date.Format("2006-01-02"),
						Amount: item.Amount, AccountID: item.AccountID, AccountName: accountMap[item.AccountID].Name,
						CurrentCategoryID: item.CategoryID, CurrentCategoryName: catNames[item.CategoryID],
						NewCategoryID: newCatID, NewCategoryName: catNames[newCatID], Changed: true,
					})
				}
			case accounting.Expense:
				newCatID := csvimport.MatchCategory(item.Description, groups)
				if newCatID != 0 && newCatID != item.CategoryID {
					rows = append(rows, ReapplyRow{
						TransactionID: item.Id, TransactionType: "expense",
						Description: item.Description, Date: item.Date.Format("2006-01-02"),
						Amount: item.Amount, AccountID: item.AccountID, AccountName: accountMap[item.AccountID].Name,
						CurrentCategoryID: item.CategoryID, CurrentCategoryName: catNames[item.CategoryID],
						NewCategoryID: newCatID, NewCategoryName: catNames[newCatID], Changed: true,
					})
				}
			}
		}
	}
	return rows, nil
}

// CategoryRulesPreview returns an http.Handler that previews the effect of re-applying
// category matching rules to all existing income and expense transactions.
func (h *ImportHandler) CategoryRulesPreview() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req categoryRulesPreviewRequest
		if r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
				http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
				return
			}
		}

		groups, httpStatus, err := h.resolveRuleGroupsForPreview(ctx, req)
		if err != nil {
			http.Error(w, err.Error(), httpStatus)
			return
		}
		if len(groups) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("[]"))
			return
		}

		accountMap, err := h.FinStore.ListAccountsMap(ctx)
		if err != nil {
			http.Error(w, "unable to list accounts: "+err.Error(), http.StatusInternalServerError)
			return
		}

		catNames, err := h.buildCategoryNamesMap(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		rows, err := h.collectPreviewRows(ctx, groups, accountMap, catNames)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
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

// CategoryRulesSubmit returns an http.Handler that applies category changes to transactions.
func (h *ImportHandler) CategoryRulesSubmit() http.Handler {
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
