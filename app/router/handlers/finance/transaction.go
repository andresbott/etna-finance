package finance

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/andresbott/etna/internal/accounting"
	"net/http"
	"strings"
	"time"
)

// generic payload struct used to handle all transaction types
type transactionPayload struct {
	Id          uint      `json:"id"`
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
}

func (h *Handler) CreateTx(userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to create entry: user not provided", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		payload := transactionPayload{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}
		var entry accounting.Transaction

		switch parseTxType(payload.Type) {
		case accounting.IncomeTransaction:
			entry = accounting.Income{
				Description: payload.Description,
				Date:        payload.Date,
				Amount:      payload.Amount,
				AccountID:   payload.AccountId,
				CategoryID:  payload.CategoryId,
			}
		case accounting.ExpenseTransaction:
			entry = accounting.Expense{
				Description: payload.Description,
				Date:        payload.Date,
				Amount:      payload.Amount,
				AccountID:   payload.AccountId,
				CategoryID:  payload.CategoryId,
			}
		case accounting.TransferTransaction:
			entry = accounting.Transfer{
				Description:     payload.Description,
				Date:            payload.Date,
				OriginAmount:    payload.OriginAmount,
				OriginAccountID: payload.OriginAccountID,
				TargetAmount:    payload.TargetAmount,
				TargetAccountID: payload.TargetAccountID,
			}
		default:
			http.Error(w, fmt.Sprintf("unknown entry type: %s", payload.Type), http.StatusBadRequest)
		}

		entryID, err := h.Store.CreateTransaction(r.Context(), entry, userId)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			} else {
				http.Error(w, fmt.Sprintf("unable to Store entry in DB: %s", err.Error()), http.StatusInternalServerError)
				return
			}
		}

		payload.Id = entryID
		respJson, err := json.Marshal(payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJson)
	})
}

type entryUpdatePayload struct {
	Type string `json:"type"`

	Description *string    `json:"description"`
	Date        *time.Time `json:"date"`

	StockAmount *float64 `json:"stockAmount"`

	// used for income / expense
	Amount     *float64 `json:"Amount"`
	AccountId  *uint    `json:"accountId"`
	CategoryId *uint    `json:"categoryId"`
	// used for transfers
	TargetAmount    *float64 `json:"targetAmount"`
	TargetAccountID *uint    `json:"targetAccountId"`
	OriginAmount    *float64 `json:"originAmount"`
	OriginAccountID *uint    `json:"originAccountId"`
}

func (h *Handler) UpdateTx(Id uint, userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to update entry: user not provided", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		payload := entryUpdatePayload{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		tr, err := h.Store.GetTransaction(r.Context(), Id, userId)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to retrive transaction: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		var entry accounting.TransactionUpdate
		switch tr.(type) {
		case accounting.Income:
			entry = accounting.IncomeUpdate{
				Description: payload.Description,
				Date:        payload.Date,
				Amount:      payload.Amount,
				AccountID:   payload.AccountId,
				CategoryID:  payload.CategoryId,
			}
		case accounting.Expense:
			entry = accounting.ExpenseUpdate{
				Description: payload.Description,
				Date:        payload.Date,
				Amount:      payload.Amount,
				AccountID:   payload.AccountId,
				CategoryID:  payload.CategoryId,
			}
		case accounting.Transfer:
			entry = accounting.TransferUpdate{
				Description:     payload.Description,
				Date:            payload.Date,
				OriginAmount:    payload.OriginAmount,
				OriginAccountID: payload.OriginAccountID,
				TargetAmount:    payload.TargetAmount,
				TargetAccountID: payload.TargetAccountID,
			}
		default:
			http.Error(w, fmt.Sprintf("unknown entry type: %s", payload.Type), http.StatusBadRequest)
			return
		}

		err = h.Store.UpdateTransaction(r.Context(), entry, Id, userId)
		if err != nil {
			if errors.Is(err, accounting.ErrEntryNotFound) {
				http.Error(w, "entry not found", http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("unable to update entry: %s", err.Error()), http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func (h *Handler) DeleteTx(Id uint, userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to delete entry: user not provided", http.StatusBadRequest)
			return
		}

		err := h.Store.DeleteTransaction(r.Context(), Id, userId)
		if err != nil {
			if errors.Is(err, accounting.ErrEntryNotFound) {
				http.Error(w, "entry not found", http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("unable to delete entry: %s", err.Error()), http.StatusInternalServerError)
			}
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}

type listEntriesResponse struct {
	Items []transactionPayload `json:"items"`
}

func (h *Handler) ListTx(userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to list entries: user not provided", http.StatusBadRequest)
			return
		}

		now := time.Now()
		startDate, endDate, err := getDateRange(
			r.URL.Query().Get("startDate"), r.URL.Query().Get("endDate"),
			now.AddDate(0, 0, -30), now,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Parse pagination parameters
		limitStr := r.URL.Query().Get("limit")
		limit := 30 // default
		if limitStr != "" {
			if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil {
				http.Error(w, "invalid limit format", http.StatusBadRequest)
				return
			}
		}

		page := 1 // default
		pageStr := r.URL.Query().Get("page")
		if pageStr != "" {
			if _, err := fmt.Sscanf(pageStr, "%d", &page); err != nil {
				http.Error(w, "invalid page format", http.StatusBadRequest)
				return
			}
		}

		// parse account ids
		var accountIds []int
		accountIdsQuery := r.URL.Query().Get("accountIds")
		if accountIdsQuery != "" {
			ids := strings.Split(accountIdsQuery, ",")
			if len(ids) > 0 {
				for _, id := range ids {
					var idInt int
					if _, err := fmt.Sscanf(id, "%d", &idInt); err != nil {
						http.Error(w, "invalid account ID format", http.StatusBadRequest)
						return
					}
					accountIds = append(accountIds, idInt)
				}
			}
		}

		opts := accounting.ListOpts{
			StartDate: startDate,
			EndDate:   endDate,
			Limit:     limit,
			AccountId: accountIds,
			Page:      page,
		}

		entries, err := h.Store.ListTransactions(r.Context(), opts, userId)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to list entries: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		response := listEntriesResponse{
			Items: make([]transactionPayload, len(entries)),
		}

		for i, entry := range entries {
			switch entry := entry.(type) {
			case accounting.Income:
				item := entry
				response.Items[i] = transactionPayload{
					Id:          item.Id,
					Description: item.Description,
					Date:        item.Date,
					Type:        incomeTxStr,
					Amount:      item.Amount,
					AccountId:   item.AccountID,
					CategoryId:  item.CategoryID,
				}
			case accounting.Expense:
				item := entry
				response.Items[i] = transactionPayload{
					Id:          item.Id,
					Description: item.Description,
					Date:        item.Date,
					Type:        expenseTxStr,
					Amount:      item.Amount,
					AccountId:   item.AccountID,
					CategoryId:  item.CategoryID,
				}
			case accounting.Transfer:
				item := entry
				response.Items[i] = transactionPayload{
					Id:              item.Id,
					Description:     item.Description,
					Date:            item.Date,
					Type:            transferTxStr,
					TargetAmount:    item.TargetAmount,
					TargetAccountID: item.TargetAccountID,
					OriginAmount:    item.OriginAmount,
					OriginAccountID: item.OriginAccountID,
				}
			}
		}

		respJson, err := json.Marshal(response)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJson)
	})
}

const (
	unknownTxStr  = "unknown"
	incomeTxStr   = "income"
	expenseTxStr  = "expense"
	transferTxStr = "transfer"
)

func parseTxType(in string) accounting.TxType {
	switch strings.ToLower(in) {
	case incomeTxStr:
		return accounting.IncomeTransaction
	case expenseTxStr:
		return accounting.ExpenseTransaction
	case transferTxStr:
		return accounting.TransferTransaction
	default:
		return accounting.UnknownTransaction
	}
}
