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
	Fees                float64 `json:"fees"`
	InvestmentAccountID uint    `json:"investmentAccountId"`
	CashAccountID       uint    `json:"cashAccountId"`
	CostBasis           float64 `json:"costBasis"`
	RealizedGainLoss    float64 `json:"realizedGainLoss"`

	// used for stock grant (instruments added for free; no cash account) - reuses accountId
	FairMarketValue float64 `json:"fairMarketValue"`

	// used for stock sell manual lot selection
	LotAllocations []struct {
		LotID    uint    `json:"lotId"`
		Quantity float64 `json:"quantity"`
	} `json:"lotAllocations,omitempty"`
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
			var lotSelections []accounting.LotSelection
			for _, a := range payload.LotAllocations {
				lotSelections = append(lotSelections, accounting.LotSelection{LotID: a.LotID, Quantity: a.Quantity})
			}
			entry = accounting.StockSell{
				Description:         payload.Description,
				Date:                payload.Date.Time,
				InvestmentAccountID: payload.InvestmentAccountID,
				CashAccountID:       payload.CashAccountID,
				InstrumentID:        payload.InstrumentID,
				Quantity:            payload.Quantity,
				TotalAmount:         payload.TotalAmount,
				Fees:                payload.Fees,
				LotSelections:       lotSelections,
			}
		case accounting.StockGrantTransaction:
			entry = accounting.StockGrant{
				Description:     payload.Description,
				Date:            payload.Date.Time,
				AccountID:       payload.AccountId,
				InstrumentID:    payload.InstrumentID,
				Quantity:        payload.Quantity,
				FairMarketValue: payload.FairMarketValue,
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
		if created, err := h.Store.GetTransaction(r.Context(), entryID); err == nil {
			if sell, ok := created.(accounting.StockSell); ok {
				payload.CostBasis = sell.CostBasis
				payload.RealizedGainLoss = sell.RealizedGainLoss
				payload.Fees = sell.Fees
			}
		}
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
	Fees        *float64 `json:"fees"`

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
	FairMarketValue     *float64 `json:"fairMarketValue"`
	InvestmentAccountID *uint    `json:"investmentAccountId"`
	CashAccountID       *uint    `json:"cashAccountId"`

	// used for stock sell manual lot selection
	LotAllocations []struct {
		LotID    uint    `json:"lotId"`
		Quantity float64 `json:"quantity"`
	} `json:"lotAllocations,omitempty"`
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
			var lotSelections []accounting.LotSelection
			for _, a := range payload.LotAllocations {
				lotSelections = append(lotSelections, accounting.LotSelection{LotID: a.LotID, Quantity: a.Quantity})
			}
			entry = accounting.StockSellUpdate{
				Description:         payload.Description,
				Date:                datePtr,
				InstrumentID:        payload.InstrumentID,
				Quantity:            payload.Quantity,
				TotalAmount:         payload.TotalAmount,
				Fees:                payload.Fees,
				InvestmentAccountID: payload.InvestmentAccountID,
				CashAccountID:       payload.CashAccountID,
				LotSelections:       lotSelections,
			}
		case accounting.StockGrant:
			entry = accounting.StockGrantUpdate{
				Description:     payload.Description,
				Date:            datePtr,
				InstrumentID:    payload.InstrumentID,
				Quantity:        payload.Quantity,
				AccountID:       payload.AccountId,
				FairMarketValue: payload.FairMarketValue,
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
	Total int64                `json:"total"`
}

// transactionToPayload converts an accounting.Transaction to the API payload shape.
func transactionToPayload(entry accounting.Transaction) transactionPayload {
	switch entry := entry.(type) {
	case accounting.Income:
		return transactionPayload{
			Id:          entry.Id,
			Description: entry.Description,
			Date:        dateOnlyTime{Time: entry.Date},
			Type:        incomeTxStr,
			Amount:      entry.Amount,
			AccountId:   entry.AccountID,
			CategoryId:  entry.CategoryID,
		}
	case accounting.Expense:
		return transactionPayload{
			Id:          entry.Id,
			Description: entry.Description,
			Date:        dateOnlyTime{Time: entry.Date},
			Type:        expenseTxStr,
			Amount:      entry.Amount,
			AccountId:   entry.AccountID,
			CategoryId:  entry.CategoryID,
		}
	case accounting.Transfer:
		return transactionPayload{
			Id:              entry.Id,
			Description:     entry.Description,
			Date:            dateOnlyTime{Time: entry.Date},
			Type:            transferTxStr,
			TargetAmount:    entry.TargetAmount,
			TargetAccountID: entry.TargetAccountID,
			OriginAmount:    entry.OriginAmount,
			OriginAccountID: entry.OriginAccountID,
		}
	case accounting.StockBuy:
		return transactionPayload{
			Id:                  entry.Id,
			Description:         entry.Description,
			Date:                dateOnlyTime{Time: entry.Date},
			Type:                stockBuyTxStr,
			StockAmount:         entry.StockAmount,
			InstrumentID:        entry.InstrumentID,
			Quantity:            entry.Quantity,
			TotalAmount:         entry.TotalAmount,
			InvestmentAccountID: entry.InvestmentAccountID,
			CashAccountID:       entry.CashAccountID,
		}
	case accounting.StockSell:
		return transactionPayload{
			Id:                  entry.Id,
			Description:         entry.Description,
			Date:                dateOnlyTime{Time: entry.Date},
			Type:                stockSellTxStr,
			InstrumentID:        entry.InstrumentID,
			Quantity:            entry.Quantity,
			TotalAmount:         entry.TotalAmount,
			CostBasis:           entry.CostBasis,
			RealizedGainLoss:    entry.RealizedGainLoss,
			Fees:                entry.Fees,
			InvestmentAccountID: entry.InvestmentAccountID,
			CashAccountID:       entry.CashAccountID,
		}
	case accounting.StockGrant:
		return transactionPayload{
			Id:              entry.Id,
			Description:     entry.Description,
			Date:            dateOnlyTime{Time: entry.Date},
			Type:            stockGrantTxStr,
			AccountId:       entry.AccountID,
			InstrumentID:    entry.InstrumentID,
			Quantity:        entry.Quantity,
			FairMarketValue: entry.FairMarketValue,
		}
	case accounting.StockTransfer:
		return transactionPayload{
			Id:              entry.Id,
			Description:     entry.Description,
			Date:            dateOnlyTime{Time: entry.Date},
			Type:            stockTransferTxStr,
			OriginAccountID: entry.SourceAccountID,
			TargetAccountID: entry.TargetAccountID,
			InstrumentID:    entry.InstrumentID,
			Quantity:        entry.Quantity,
		}
	default:
		return transactionPayload{Type: unknownTxStr}
	}
}

func (h *Handler) GetTx(id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tr, err := h.Store.GetTransaction(r.Context(), id)
		if err != nil {
			if errors.Is(err, accounting.ErrTransactionNotFound) {
				http.Error(w, "entry not found", http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("unable to get entry: %s", err.Error()), http.StatusInternalServerError)
			}
			return
		}
		payload := transactionToPayload(tr)
		respJSON, err := json.Marshal(payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
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

		// parse account ids (support multiple ?accountIds=1&accountIds=2 and comma-separated)
		var accountIds []int
		for _, raw := range r.URL.Query()["accountIds"] {
			for _, id := range strings.Split(raw, ",") {
				id = strings.TrimSpace(id)
				if id == "" {
					continue
				}
				var idInt int
				if _, err := fmt.Sscanf(id, "%d", &idInt); err != nil {
					http.Error(w, "invalid account ID format", http.StatusBadRequest)
					return
				}
				accountIds = append(accountIds, idInt)
			}
		}

		opts := accounting.ListOpts{
			StartDate: startDate,
			EndDate:   endDate,
			Limit:     limit,
			AccountId: accountIds,
			Page:      page,
		}

		entries, total, err := h.Store.ListTransactions(r.Context(), opts)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to list entries: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		response := listEntriesResponse{
			Items: make([]transactionPayload, len(entries)),
			Total: total,
		}
		for i, entry := range entries {
			response.Items[i] = transactionToPayload(entry)
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
