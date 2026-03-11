package csvimport

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/andresbott/etna/internal/accounting"
	"github.com/andresbott/etna/internal/csvimport"
)

type ImportHandler struct {
	CsvStore *csvimport.Store
	FinStore *accounting.Store
}

func (h *ImportHandler) ParseCSV() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, fmt.Sprintf("unable to parse multipart form: %s", err.Error()), http.StatusBadRequest)
			return
		}

		accountIDStr := r.FormValue("accountId")
		if accountIDStr == "" {
			http.Error(w, "accountId is required", http.StatusBadRequest)
			return
		}
		accountID64, err := strconv.ParseUint(accountIDStr, 10, 64)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid accountId: %s", err.Error()), http.StatusBadRequest)
			return
		}
		accountID := uint(accountID64)

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to get uploaded file: %s", err.Error()), http.StatusBadRequest)
			return
		}
		defer func() { _ = file.Close() }()

		// Look up account to get ImportProfileID
		account, err := h.FinStore.GetAccount(r.Context(), accountID)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to get account: %s", err.Error()), http.StatusNotFound)
			return
		}
		if account.ImportProfileID == 0 {
			http.Error(w, "account has no import profile", http.StatusBadRequest)
			return
		}

		// Look up profile
		profile, err := h.CsvStore.GetProfile(r.Context(), account.ImportProfileID)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to get import profile: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		// Load category rule groups
		groups, err := h.CsvStore.ListCategoryRuleGroups(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to list category rule groups: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		// Load existing transactions for duplicate detection
		existing, err := h.loadExistingTransactions(r, accountID)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to load existing transactions: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		// Parse CSV
		rows, err := csvimport.Parse(file, profile, groups, existing)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to parse CSV: %s", err.Error()), http.StatusBadRequest)
			return
		}

		resp := map[string]any{"rows": rows}
		respJSON, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}

func (h *ImportHandler) loadExistingTransactions(r *http.Request, accountID uint) ([]csvimport.ExistingTx, error) {
	if accountID > uint(math.MaxInt) {
		return nil, fmt.Errorf("accountID %d overflows int", accountID)
	}

	var existing []csvimport.ExistingTx

	// Paginate through all transactions to ensure complete duplicate detection
	for page := 1; ; page++ {
		opts := accounting.ListOpts{
			StartDate: time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC),
			AccountId: []int{int(accountID)},
			Limit:     accounting.MaxSearchResults,
			Page:      page,
		}

		txs, _, err := h.FinStore.ListTransactions(r.Context(), opts)
		if err != nil {
			return nil, err
		}
		if len(txs) == 0 {
			break
		}

		for _, tx := range txs {
			switch item := tx.(type) {
			case accounting.Income:
				existing = append(existing, csvimport.ExistingTx{
					Date:   item.Date.Format("2006-01-02"),
					Amount: item.Amount,
				})
			case accounting.Expense:
				existing = append(existing, csvimport.ExistingTx{
					Date:   item.Date.Format("2006-01-02"),
					Amount: -item.Amount, // CSV parser uses negative for expenses
				})
			case accounting.Transfer:
				// For transfers, include both legs if they match the account
				if item.OriginAccountID == accountID {
					existing = append(existing, csvimport.ExistingTx{
						Date:   item.Date.Format("2006-01-02"),
						Amount: -item.OriginAmount,
					})
				}
				if item.TargetAccountID == accountID {
					existing = append(existing, csvimport.ExistingTx{
						Date:   item.Date.Format("2006-01-02"),
						Amount: item.TargetAmount,
					})
				}
			}
		}
	}
	return existing, nil
}

type submitRequest struct {
	AccountID uint        `json:"accountId"`
	Rows      []submitRow `json:"rows"`
}

type submitRow struct {
	Date        string  `json:"date"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Type        string  `json:"type"`
	CategoryID  uint    `json:"categoryId"`
}

func (h *ImportHandler) SubmitImport() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		var req submitRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		if req.AccountID == 0 {
			http.Error(w, "accountId is required", http.StatusBadRequest)
			return
		}

		created := 0
		for _, row := range req.Rows {
			date, err := time.Parse("2006-01-02", row.Date)
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid date %q: %s", row.Date, err.Error()), http.StatusBadRequest)
				return
			}

			var tx accounting.Transaction
			switch row.Type {
			case "income":
				tx = accounting.Income{
					Description: row.Description,
					Amount:      row.Amount,
					AccountID:   req.AccountID,
					CategoryID:  row.CategoryID,
					Date:        date,
				}
			case "expense":
				tx = accounting.Expense{
					Description: row.Description,
					Amount:      math.Abs(row.Amount),
					AccountID:   req.AccountID,
					CategoryID:  row.CategoryID,
					Date:        date,
				}
			default:
				http.Error(w, fmt.Sprintf("unsupported transaction type: %s", row.Type), http.StatusBadRequest)
				return
			}

			if _, err := h.FinStore.CreateTransaction(r.Context(), tx); err != nil {
				var valErr accounting.ErrValidation
				if errors.As(err, &valErr) {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				http.Error(w, fmt.Sprintf("error creating transaction: %s", err.Error()), http.StatusInternalServerError)
				return
			}
			created++
		}

		resp := map[string]int{"created": created}
		respJSON, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}
