package finance

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/andresbott/etna/internal/accounting"
)

// dateOnlyTime unmarshals JSON date strings in "2006-01-02" or full RFC3339 format.
type dateOnlyTime struct{ time.Time }

func (t *dateOnlyTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		t.Time = time.Time{}
		return nil
	}
	parsed, err := time.Parse("2006-01-02", s)
	if err == nil {
		t.Time = parsed
		return nil
	}
	parsed, err = time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	t.Time = parsed
	return nil
}

// generic payload struct used to handle all transaction types
type transactionPayload struct {
	Id          uint         `json:"id"`
	Description string       `json:"description"`
	Date        dateOnlyTime `json:"date"`
	Type        string       `json:"type"`

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
	InstrumentID        uint    `json:"instrumentId"`
	Quantity            float64 `json:"quantity"`
	TotalAmount         float64 `json:"totalAmount"`
	InvestmentAccountID uint    `json:"investmentAccountId"`
	CashAccountID       uint    `json:"cashAccountId"`

	// used for stock grant (instruments added for free; no cash account) - reuses accountId
}

func (h *Handler) CreateTx() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				Date:        payload.Date.Time,
				Amount:      payload.Amount,
				AccountID:   payload.AccountId,
				CategoryID:  payload.CategoryId,
			}
		case accounting.ExpenseTransaction:
			entry = accounting.Expense{
				Description: payload.Description,
				Date:        payload.Date.Time,
				Amount:      payload.Amount,
				AccountID:   payload.AccountId,
				CategoryID:  payload.CategoryId,
			}
		case accounting.TransferTransaction:
			entry = accounting.Transfer{
				Description:     payload.Description,
				Date:            payload.Date.Time,
				OriginAmount:    payload.OriginAmount,
				OriginAccountID: payload.OriginAccountID,
				TargetAmount:    payload.TargetAmount,
				TargetAccountID: payload.TargetAccountID,
			}
		case accounting.StockBuyTransaction:
			entry = accounting.StockBuy{
				Description:         payload.Description,
				Date:                payload.Date.Time,
				InvestmentAccountID: payload.InvestmentAccountID,
				CashAccountID:       payload.CashAccountID,
				InstrumentID:        payload.InstrumentID,
				Quantity:            payload.Quantity,
				TotalAmount:         payload.TotalAmount,
				StockAmount:         payload.StockAmount,
			}
		case accounting.StockSellTransaction:
			entry = accounting.StockSell{
				Description:         payload.Description,
				Date:                payload.Date.Time,
				InvestmentAccountID: payload.InvestmentAccountID,
				CashAccountID:       payload.CashAccountID,
				InstrumentID:        payload.InstrumentID,
				Quantity:            payload.Quantity,
				TotalAmount:         payload.TotalAmount,
				StockAmount:         payload.StockAmount,
			}
		case accounting.StockGrantTransaction:
			entry = accounting.StockGrant{
				Description:  payload.Description,
				Date:         payload.Date.Time,
				AccountID:    payload.AccountId,
				InstrumentID: payload.InstrumentID,
				Quantity:     payload.Quantity,
			}
		case accounting.StockTransferTransaction:
			entry = accounting.StockTransfer{
				Description:     payload.Description,
				Date:            payload.Date.Time,
				SourceAccountID: payload.OriginAccountID,
				TargetAccountID: payload.TargetAccountID,
				InstrumentID:    payload.InstrumentID,
				Quantity:        payload.Quantity,
			}
		default:
			http.Error(w, fmt.Sprintf("unknown entry type: %s", payload.Type), http.StatusBadRequest)
			return
		}

		entryID, err := h.Store.CreateTransaction(r.Context(), entry)
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

// dateOnlyTimePtr is like dateOnlyTime but for pointer; unmarshals "2006-01-02" or RFC3339.
type dateOnlyTimePtr struct{ time.Time }

func (t *dateOnlyTimePtr) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		t.Time = time.Time{}
		return nil
	}
	parsed, err := time.Parse("2006-01-02", s)
	if err == nil {
		t.Time = parsed
		return nil
	}
	parsed, err = time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	t.Time = parsed
	return nil
}

func dateOnlyPtrToTime(d *dateOnlyTimePtr) *time.Time {
	if d == nil {
		return nil
	}
	t := d.Time
	return &t
}

type entryUpdatePayload struct {
	Type string `json:"type"`

	Description *string          `json:"description"`
	Date        *dateOnlyTimePtr `json:"date"`

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

	// used for stock buy / sell / grant / transfer
	InstrumentID        *uint    `json:"instrumentId"`
	Quantity            *float64 `json:"quantity"`
	TotalAmount         *float64 `json:"totalAmount"`
	InvestmentAccountID *uint    `json:"investmentAccountId"`
	CashAccountID       *uint    `json:"cashAccountId"`
}

func (h *Handler) UpdateTx(Id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		tr, err := h.Store.GetTransaction(r.Context(), Id)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to retrive transaction: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		datePtr := dateOnlyPtrToTime(payload.Date)
		var entry accounting.TransactionUpdate
		switch tr.(type) {
		case accounting.Income:
			entry = accounting.IncomeUpdate{
				Description: payload.Description,
				Date:        datePtr,
				Amount:      payload.Amount,
				AccountID:   payload.AccountId,
				CategoryID:  payload.CategoryId,
			}
		case accounting.Expense:
			entry = accounting.ExpenseUpdate{
				Description: payload.Description,
				Date:        datePtr,
				Amount:      payload.Amount,
				AccountID:   payload.AccountId,
				CategoryID:  payload.CategoryId,
			}
		case accounting.Transfer:
			entry = accounting.TransferUpdate{
				Description:     payload.Description,
				Date:            datePtr,
				OriginAmount:    payload.OriginAmount,
				OriginAccountID: payload.OriginAccountID,
				TargetAmount:    payload.TargetAmount,
				TargetAccountID: payload.TargetAccountID,
			}
		case accounting.StockBuy:
			entry = accounting.StockBuyUpdate{
				Description:         payload.Description,
				Date:                datePtr,
				InstrumentID:        payload.InstrumentID,
				Quantity:            payload.Quantity,
				TotalAmount:         payload.TotalAmount,
				StockAmount:         payload.StockAmount,
				InvestmentAccountID: payload.InvestmentAccountID,
				CashAccountID:       payload.CashAccountID,
			}
		case accounting.StockSell:
			entry = accounting.StockSellUpdate{
				Description:         payload.Description,
				Date:                datePtr,
				InstrumentID:        payload.InstrumentID,
				Quantity:            payload.Quantity,
				TotalAmount:         payload.TotalAmount,
				StockAmount:         payload.StockAmount,
				InvestmentAccountID: payload.InvestmentAccountID,
				CashAccountID:       payload.CashAccountID,
			}
		case accounting.StockGrant:
			entry = accounting.StockGrantUpdate{
				Description:  payload.Description,
				Date:         datePtr,
				InstrumentID: payload.InstrumentID,
				Quantity:     payload.Quantity,
				AccountID:    payload.AccountId,
			}
		case accounting.StockTransfer:
			entry = accounting.StockTransferUpdate{
				Description:     payload.Description,
				Date:            datePtr,
				InstrumentID:    payload.InstrumentID,
				Quantity:        payload.Quantity,
				SourceAccountID: payload.OriginAccountID,
				TargetAccountID: payload.TargetAccountID,
			}
		default:
			http.Error(w, fmt.Sprintf("unknown entry type: %T", tr), http.StatusBadRequest)
			return
		}

		err = h.Store.UpdateTransaction(r.Context(), entry, Id)
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

func (h *Handler) DeleteTx(Id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := h.Store.DeleteTransaction(r.Context(), Id)
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

func (h *Handler) ListTx() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		entries, err := h.Store.ListTransactions(r.Context(), opts)
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
					Date:        dateOnlyTime{Time: item.Date},
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
					Date:        dateOnlyTime{Time: item.Date},
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
					Date:            dateOnlyTime{Time: item.Date},
					Type:            transferTxStr,
					TargetAmount:    item.TargetAmount,
					TargetAccountID: item.TargetAccountID,
					OriginAmount:    item.OriginAmount,
					OriginAccountID: item.OriginAccountID,
				}
			case accounting.StockBuy:
				item := entry
				response.Items[i] = transactionPayload{
					Id:                  item.Id,
					Description:         item.Description,
					Date:                dateOnlyTime{Time: item.Date},
					Type:                stockBuyTxStr,
					StockAmount:         item.StockAmount,
					InstrumentID:        item.InstrumentID,
					Quantity:            item.Quantity,
					TotalAmount:         item.TotalAmount,
					InvestmentAccountID: item.InvestmentAccountID,
					CashAccountID:       item.CashAccountID,
				}
			case accounting.StockSell:
				item := entry
				response.Items[i] = transactionPayload{
					Id:                  item.Id,
					Description:         item.Description,
					Date:                dateOnlyTime{Time: item.Date},
					Type:                stockSellTxStr,
					StockAmount:         item.StockAmount,
					InstrumentID:        item.InstrumentID,
					Quantity:            item.Quantity,
					TotalAmount:         item.TotalAmount,
					InvestmentAccountID: item.InvestmentAccountID,
					CashAccountID:       item.CashAccountID,
				}
			case accounting.StockGrant:
				item := entry
				response.Items[i] = transactionPayload{
					Id:           item.Id,
					Description:  item.Description,
					Date:         dateOnlyTime{Time: item.Date},
					Type:         stockGrantTxStr,
					AccountId:    item.AccountID,
					InstrumentID: item.InstrumentID,
					Quantity:     item.Quantity,
				}
			case accounting.StockTransfer:
				item := entry
				response.Items[i] = transactionPayload{
					Id:              item.Id,
					Description:     item.Description,
					Date:            dateOnlyTime{Time: item.Date},
					Type:            stockTransferTxStr,
					OriginAccountID: item.SourceAccountID,
					TargetAccountID: item.TargetAccountID,
					InstrumentID:    item.InstrumentID,
					Quantity:        item.Quantity,
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
	unknownTxStr       = "unknown"
	incomeTxStr        = "income"
	expenseTxStr       = "expense"
	transferTxStr      = "transfer"
	stockBuyTxStr      = "stockbuy"
	stockSellTxStr     = "stocksell"
	stockGrantTxStr    = "stockgrant"
	stockTransferTxStr = "stocktransfer"
)

func parseTxType(in string) accounting.TxType {
	switch strings.ToLower(in) {
	case incomeTxStr:
		return accounting.IncomeTransaction
	case expenseTxStr:
		return accounting.ExpenseTransaction
	case transferTxStr:
		return accounting.TransferTransaction
	case stockBuyTxStr:
		return accounting.StockBuyTransaction
	case stockSellTxStr:
		return accounting.StockSellTransaction
	case stockGrantTxStr:
		return accounting.StockGrantTransaction
	case stockTransferTxStr:
		return accounting.StockTransferTransaction
	default:
		return accounting.UnknownTransaction
	}
}
