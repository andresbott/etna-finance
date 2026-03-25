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
	Notes       string       `json:"notes"`
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
	PricePerShare       float64 `json:"pricePerShare,omitempty"`
	TotalAmount         float64 `json:"totalAmount"`
	Fees                float64 `json:"fees"`
	InvestmentAccountID uint    `json:"investmentAccountId"`
	CashAccountID       uint    `json:"cashAccountId"`
	CostBasis           float64 `json:"costBasis"`
	RealizedGainLoss    float64 `json:"realizedGainLoss"`

	// used for stock grant (instruments added for free; no cash account) - reuses accountId
	FairMarketValue float64 `json:"fairMarketValue"`

	// used for stock vest
	VestingPrice float64 `json:"vestingPrice"`

	// used for stock sell manual lot selection
	LotAllocations []struct {
		LotID    uint    `json:"lotId"`
		Quantity float64 `json:"quantity"`
	} `json:"lotAllocations,omitempty"`

	// used for revaluation (informative target balance)
	Balance float64 `json:"balance"`

	AttachmentID *uint `json:"attachmentId,omitempty"`
}

// parseLotSelections converts payload lot allocations to accounting LotSelection values.
func parseLotSelections(allocations []struct {
	LotID    uint    `json:"lotId"`
	Quantity float64 `json:"quantity"`
}) []accounting.LotSelection {
	var selections []accounting.LotSelection
	for _, a := range allocations {
		selections = append(selections, accounting.LotSelection{LotID: a.LotID, Quantity: a.Quantity})
	}
	return selections
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
				Notes:       payload.Notes,
				Date:        payload.Date.Time,
				Amount:      payload.Amount,
				AccountID:   payload.AccountId,
				CategoryID:  payload.CategoryId,
			}
		case accounting.ExpenseTransaction:
			entry = accounting.Expense{
				Description: payload.Description,
				Notes:       payload.Notes,
				Date:        payload.Date.Time,
				Amount:      payload.Amount,
				AccountID:   payload.AccountId,
				CategoryID:  payload.CategoryId,
			}
		case accounting.TransferTransaction:
			entry = accounting.Transfer{
				Description:     payload.Description,
				Notes:           payload.Notes,
				Date:            payload.Date.Time,
				OriginAmount:    payload.OriginAmount,
				OriginAccountID: payload.OriginAccountID,
				TargetAmount:    payload.TargetAmount,
				TargetAccountID: payload.TargetAccountID,
			}
		case accounting.StockBuyTransaction:
			entry = accounting.StockBuy{
				Description:         payload.Description,
				Notes:               payload.Notes,
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
				Notes:               payload.Notes,
				Date:                payload.Date.Time,
				InvestmentAccountID: payload.InvestmentAccountID,
				CashAccountID:       payload.CashAccountID,
				InstrumentID:        payload.InstrumentID,
				Quantity:            payload.Quantity,
				PricePerShare:       payload.PricePerShare,
				TotalAmount:         payload.TotalAmount,
				Fees:                payload.Fees,
				LotSelections:       parseLotSelections(payload.LotAllocations),
			}
		case accounting.StockGrantTransaction:
			entry = accounting.StockGrant{
				Description:     payload.Description,
				Notes:           payload.Notes,
				Date:            payload.Date.Time,
				AccountID:       payload.AccountId,
				InstrumentID:    payload.InstrumentID,
				Quantity:        payload.Quantity,
				FairMarketValue: payload.FairMarketValue,
			}
		case accounting.StockTransferTransaction:
			entry = accounting.StockTransfer{
				Description:     payload.Description,
				Notes:           payload.Notes,
				Date:            payload.Date.Time,
				SourceAccountID: payload.OriginAccountID,
				TargetAccountID: payload.TargetAccountID,
				InstrumentID:    payload.InstrumentID,
				Quantity:        payload.Quantity,
			}
		case accounting.StockVestTransaction:
			entry = accounting.StockVest{
				Description:     payload.Description,
				Notes:           payload.Notes,
				Date:            payload.Date.Time,
				SourceAccountID: payload.OriginAccountID,
				TargetAccountID: payload.TargetAccountID,
				InstrumentID:    payload.InstrumentID,
				VestingPrice:    payload.VestingPrice,
				CategoryID:      payload.CategoryId,
				LotSelections:   parseLotSelections(payload.LotAllocations),
			}
		case accounting.StockForfeitTransaction:
			entry = accounting.StockForfeit{
				Description:   payload.Description,
				Notes:         payload.Notes,
				Date:          payload.Date.Time,
				AccountID:     payload.AccountId,
				InstrumentID:  payload.InstrumentID,
				LotSelections: parseLotSelections(payload.LotAllocations),
			}
		case accounting.BalanceStatusTransaction:
			entry = accounting.BalanceStatus{
				Description: payload.Description,
				Notes:       payload.Notes,
				Date:        payload.Date.Time,
				Amount:      payload.Amount,
				AccountID:   payload.AccountId,
			}
		case accounting.RevaluationTransaction:
			entry = accounting.Revaluation{
				Description: payload.Description,
				Notes:       payload.Notes,
				Date:        payload.Date.Time,
				Amount:      payload.Amount,
				Balance:     payload.Balance,
				AccountID:   payload.AccountId,
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
	Notes       *string          `json:"notes"`
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
	PricePerShare       *float64 `json:"pricePerShare"`
	TotalAmount         *float64 `json:"totalAmount"`
	FairMarketValue     *float64 `json:"fairMarketValue"`
	// used for stock vest
	VestingPrice *float64 `json:"vestingPrice"`

	InvestmentAccountID *uint `json:"investmentAccountId"`
	CashAccountID       *uint    `json:"cashAccountId"`

	// used for revaluation (informative target balance)
	Balance *float64 `json:"balance"`

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
				Notes:       payload.Notes,
				Date:        datePtr,
				Amount:      payload.Amount,
				AccountID:   payload.AccountId,
				CategoryID:  payload.CategoryId,
			}
		case accounting.Expense:
			entry = accounting.ExpenseUpdate{
				Description: payload.Description,
				Notes:       payload.Notes,
				Date:        datePtr,
				Amount:      payload.Amount,
				AccountID:   payload.AccountId,
				CategoryID:  payload.CategoryId,
			}
		case accounting.Transfer:
			entry = accounting.TransferUpdate{
				Description:     payload.Description,
				Notes:           payload.Notes,
				Date:            datePtr,
				OriginAmount:    payload.OriginAmount,
				OriginAccountID: payload.OriginAccountID,
				TargetAmount:    payload.TargetAmount,
				TargetAccountID: payload.TargetAccountID,
			}
		case accounting.StockBuy:
			entry = accounting.StockBuyUpdate{
				Description:         payload.Description,
				Notes:               payload.Notes,
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
				Notes:               payload.Notes,
				Date:                datePtr,
				InstrumentID:        payload.InstrumentID,
				Quantity:            payload.Quantity,
				PricePerShare:       payload.PricePerShare,
				TotalAmount:         payload.TotalAmount,
				Fees:                payload.Fees,
				InvestmentAccountID: payload.InvestmentAccountID,
				CashAccountID:       payload.CashAccountID,
				LotSelections:       parseLotSelections(payload.LotAllocations),
			}
		case accounting.StockGrant:
			entry = accounting.StockGrantUpdate{
				Description:     payload.Description,
				Notes:           payload.Notes,
				Date:            datePtr,
				InstrumentID:    payload.InstrumentID,
				Quantity:        payload.Quantity,
				AccountID:       payload.AccountId,
				FairMarketValue: payload.FairMarketValue,
			}
		case accounting.StockTransfer:
			entry = accounting.StockTransferUpdate{
				Description:     payload.Description,
				Notes:           payload.Notes,
				Date:            datePtr,
				InstrumentID:    payload.InstrumentID,
				Quantity:        payload.Quantity,
				SourceAccountID: payload.OriginAccountID,
				TargetAccountID: payload.TargetAccountID,
			}
		case accounting.StockVest:
			entry = accounting.StockVestUpdate{
				Description:     payload.Description,
				Notes:           payload.Notes,
				Date:            datePtr,
				InstrumentID:    payload.InstrumentID,
				VestingPrice:    payload.VestingPrice,
				CategoryID:      payload.CategoryId,
				SourceAccountID: payload.OriginAccountID,
				TargetAccountID: payload.TargetAccountID,
				LotSelections:   parseLotSelections(payload.LotAllocations),
			}
		case accounting.StockForfeit:
			entry = accounting.StockForfeitUpdate{
				Description:   payload.Description,
				Notes:         payload.Notes,
				Date:          datePtr,
				AccountID:     payload.AccountId,
				InstrumentID:  payload.InstrumentID,
				LotSelections: parseLotSelections(payload.LotAllocations),
			}
		case accounting.BalanceStatus:
			entry = accounting.BalanceStatusUpdate{
				Description: payload.Description,
				Notes:       payload.Notes,
				Date:        datePtr,
				Amount:      payload.Amount,
				AccountID:   payload.AccountId,
			}
		case accounting.Revaluation:
			entry = accounting.RevaluationUpdate{
				Description: payload.Description,
				Notes:       payload.Notes,
				Date:        datePtr,
				Amount:      payload.Amount,
				Balance:     payload.Balance,
				AccountID:   payload.AccountId,
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
		// Clean up attachment if exists
		tx, err := h.Store.GetTransaction(r.Context(), Id)
		if err == nil {
			if attID := transactionAttachmentID(tx); attID != nil && h.FileStore != nil {
				_ = h.FileStore.Delete(r.Context(), *attID)
			}
		}

		err = h.Store.DeleteTransaction(r.Context(), Id)
		if err != nil {
			if errors.Is(err, accounting.ErrEntryNotFound) || errors.Is(err, accounting.ErrTransactionNotFound) {
				http.Error(w, "entry not found", http.StatusNotFound)
			} else if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				http.Error(w, fmt.Sprintf("unable to delete entry: %s", err.Error()), http.StatusInternalServerError)
			}
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}

type listEntriesResponse struct {
	Items        []transactionPayload `json:"items"`
	Total        int64                `json:"total"`
	PriorBalance float64              `json:"priorBalance,omitempty"`
}

// transactionToPayload converts an accounting.Transaction to the API payload shape.
func transactionToPayload(entry accounting.Transaction) transactionPayload {
	switch entry := entry.(type) {
	case accounting.Income:
		return transactionPayload{
			Id:           entry.Id,
			Description:  entry.Description,
			Notes:        entry.Notes,
			Date:         dateOnlyTime{Time: entry.Date},
			Type:         incomeTxStr,
			Amount:       entry.Amount,
			AccountId:    entry.AccountID,
			CategoryId:   entry.CategoryID,
			AttachmentID: entry.AttachmentID,
		}
	case accounting.Expense:
		return transactionPayload{
			Id:           entry.Id,
			Description:  entry.Description,
			Notes:        entry.Notes,
			Date:         dateOnlyTime{Time: entry.Date},
			Type:         expenseTxStr,
			Amount:       entry.Amount,
			AccountId:    entry.AccountID,
			CategoryId:   entry.CategoryID,
			AttachmentID: entry.AttachmentID,
		}
	case accounting.Transfer:
		return transactionPayload{
			Id:              entry.Id,
			Description:     entry.Description,
			Notes:           entry.Notes,
			Date:            dateOnlyTime{Time: entry.Date},
			Type:            transferTxStr,
			TargetAmount:    entry.TargetAmount,
			TargetAccountID: entry.TargetAccountID,
			OriginAmount:    entry.OriginAmount,
			OriginAccountID: entry.OriginAccountID,
			AttachmentID:    entry.AttachmentID,
		}
	case accounting.StockBuy:
		return transactionPayload{
			Id:                  entry.Id,
			Description:         entry.Description,
			Notes:               entry.Notes,
			Date:                dateOnlyTime{Time: entry.Date},
			Type:                stockBuyTxStr,
			StockAmount:         entry.StockAmount,
			InstrumentID:        entry.InstrumentID,
			Quantity:            entry.Quantity,
			TotalAmount:         entry.TotalAmount,
			InvestmentAccountID: entry.InvestmentAccountID,
			CashAccountID:       entry.CashAccountID,
			AttachmentID:        entry.AttachmentID,
		}
	case accounting.StockSell:
		payload := transactionPayload{
			Id:                  entry.Id,
			Description:         entry.Description,
			Notes:               entry.Notes,
			Date:                dateOnlyTime{Time: entry.Date},
			Type:                stockSellTxStr,
			InstrumentID:        entry.InstrumentID,
			Quantity:            entry.Quantity,
			PricePerShare:       entry.PricePerShare,
			TotalAmount:         entry.TotalAmount,
			CostBasis:           entry.CostBasis,
			RealizedGainLoss:    entry.RealizedGainLoss,
			Fees:                entry.Fees,
			InvestmentAccountID: entry.InvestmentAccountID,
			CashAccountID:       entry.CashAccountID,
			AttachmentID:        entry.AttachmentID,
		}
		for _, ls := range entry.LotSelections {
			payload.LotAllocations = append(payload.LotAllocations, struct {
				LotID    uint    `json:"lotId"`
				Quantity float64 `json:"quantity"`
			}{LotID: ls.LotID, Quantity: ls.Quantity})
		}
		return payload
	case accounting.StockGrant:
		return transactionPayload{
			Id:              entry.Id,
			Description:     entry.Description,
			Notes:           entry.Notes,
			Date:            dateOnlyTime{Time: entry.Date},
			Type:            stockGrantTxStr,
			AccountId:       entry.AccountID,
			InstrumentID:    entry.InstrumentID,
			Quantity:        entry.Quantity,
			FairMarketValue: entry.FairMarketValue,
			AttachmentID:    entry.AttachmentID,
		}
	case accounting.StockTransfer:
		return transactionPayload{
			Id:              entry.Id,
			Description:     entry.Description,
			Notes:           entry.Notes,
			Date:            dateOnlyTime{Time: entry.Date},
			Type:            stockTransferTxStr,
			OriginAccountID: entry.SourceAccountID,
			TargetAccountID: entry.TargetAccountID,
			InstrumentID:    entry.InstrumentID,
			Quantity:        entry.Quantity,
			AttachmentID:    entry.AttachmentID,
		}
	case accounting.StockVest:
		payload := transactionPayload{
			Id:              entry.Id,
			Description:     entry.Description,
			Notes:           entry.Notes,
			Date:            dateOnlyTime{Time: entry.Date},
			Type:            stockVestTxStr,
			OriginAccountID: entry.SourceAccountID,
			TargetAccountID: entry.TargetAccountID,
			InstrumentID:    entry.InstrumentID,
			Quantity:        entry.Quantity,
			VestingPrice:    entry.VestingPrice,
			CategoryId:      entry.CategoryID,
			AttachmentID:    entry.AttachmentID,
		}
		for _, ls := range entry.LotSelections {
			payload.LotAllocations = append(payload.LotAllocations, struct {
				LotID    uint    `json:"lotId"`
				Quantity float64 `json:"quantity"`
			}{LotID: ls.LotID, Quantity: ls.Quantity})
		}
		return payload
	case accounting.StockForfeit:
		payload := transactionPayload{
			Id:           entry.Id,
			Description:  entry.Description,
			Notes:        entry.Notes,
			Date:         dateOnlyTime{Time: entry.Date},
			Type:         stockForfeitTxStr,
			AccountId:    entry.AccountID,
			InstrumentID: entry.InstrumentID,
			Quantity:     entry.Quantity,
			AttachmentID: entry.AttachmentID,
		}
		for _, ls := range entry.LotSelections {
			payload.LotAllocations = append(payload.LotAllocations, struct {
				LotID    uint    `json:"lotId"`
				Quantity float64 `json:"quantity"`
			}{LotID: ls.LotID, Quantity: ls.Quantity})
		}
		return payload
	case accounting.BalanceStatus:
		return transactionPayload{
			Id:           entry.Id,
			Description:  entry.Description,
			Notes:        entry.Notes,
			Date:         dateOnlyTime{Time: entry.Date},
			Type:         balanceStatusTxStr,
			Amount:       entry.Amount,
			AccountId:    entry.AccountID,
			AttachmentID: entry.AttachmentID,
		}
	case accounting.Revaluation:
		return transactionPayload{
			Id:           entry.Id,
			Description:  entry.Description,
			Notes:        entry.Notes,
			Date:         dateOnlyTime{Time: entry.Date},
			Type:         revaluationTxStr,
			Amount:       entry.Amount,
			Balance:      entry.Balance,
			AccountId:    entry.AccountID,
			AttachmentID: entry.AttachmentID,
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

// parseIntQueryParam parses an integer query parameter with a default value.
func parseIntQueryParam(r *http.Request, key string, defaultVal int) (int, error) {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal, nil
	}
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
		return 0, fmt.Errorf("invalid %s format", key)
	}
	return v, nil
}

// parseIntListParam parses a repeated/comma-separated integer query parameter.
func parseIntListParam(r *http.Request, key string) ([]int, error) {
	var ids []int
	for _, raw := range r.URL.Query()[key] {
		for _, s := range strings.Split(raw, ",") {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			var v int
			if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
				return nil, fmt.Errorf("invalid %s format", key)
			}
			ids = append(ids, v)
		}
	}
	return ids, nil
}

// parseUintListParam parses a repeated/comma-separated uint query parameter.
func parseUintListParam(r *http.Request, key string) ([]uint, error) {
	var ids []uint
	for _, raw := range r.URL.Query()[key] {
		for _, s := range strings.Split(raw, ",") {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			var v uint
			if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
				return nil, fmt.Errorf("invalid %s format", key)
			}
			ids = append(ids, v)
		}
	}
	return ids, nil
}

// parseStringListParam parses a repeated/comma-separated string query parameter.
func parseStringListParam(r *http.Request, key string) []string {
	var vals []string
	for _, raw := range r.URL.Query()[key] {
		for _, s := range strings.Split(raw, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				vals = append(vals, s)
			}
		}
	}
	return vals
}

func (h *Handler) ListTx() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		opts, accountIds, err := parseListTxParams(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		entries, total, err := h.Store.ListTransactions(r.Context(), opts)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to list entries: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		// Compute prior-page balance when filtering by a single account
		var priorBalance float64
		if len(accountIds) == 1 && accountIds[0] >= 0 {
			priorBalance, err = h.Store.PriorPageBalance(r.Context(), opts, uint(accountIds[0])) // #nosec G115 -- validated non-negative above
			if err != nil {
				priorBalance = 0
			}
		}

		response := listEntriesResponse{
			Items:        make([]transactionPayload, len(entries)),
			Total:        total,
			PriorBalance: priorBalance,
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

func parseListTxParams(r *http.Request) (accounting.ListOpts, []int, error) {
	now := time.Now()
	startDate, endDate, err := getDateRange(
		r.URL.Query().Get("startDate"), r.URL.Query().Get("endDate"),
		now.AddDate(0, 0, -30), now,
	)
	if err != nil {
		return accounting.ListOpts{}, nil, err
	}

	limit, err := parseIntQueryParam(r, "limit", 30)
	if err != nil {
		return accounting.ListOpts{}, nil, err
	}
	page, err := parseIntQueryParam(r, "page", 1)
	if err != nil {
		return accounting.ListOpts{}, nil, err
	}

	accountIds, err := parseIntListParam(r, "accountIds")
	if err != nil {
		return accounting.ListOpts{}, nil, err
	}

	txTypes := expandTypeGroups(parseStringListParam(r, "types"))

	categoryIds, err := parseUintListParam(r, "categoryIds")
	if err != nil {
		return accounting.ListOpts{}, nil, err
	}

	var hasAttachment *bool
	if r.URL.Query().Get("hasAttachment") == "true" {
		b := true
		hasAttachment = &b
	}

	search := r.URL.Query().Get("search")
	if len(search) > 100 {
		search = search[:100]
	}

	opts := accounting.ListOpts{
		StartDate:     startDate,
		EndDate:       endDate,
		Limit:         limit,
		AccountId:     accountIds,
		Page:          page,
		Types:         txTypes,
		CategoryIds:   categoryIds,
		HasAttachment: hasAttachment,
		Search:        search,
	}
	return opts, accountIds, nil
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
	balanceStatusTxStr  = "balancestatus"
	stockVestTxStr      = "stockvest"
	stockForfeitTxStr   = "stockforfeit"
	revaluationTxStr    = "revaluation"
)

const investmentGroupStr = "investment"

// expandTypeGroups takes user-facing type group names and returns the corresponding TxType values.
func expandTypeGroups(groups []string) []accounting.TxType {
	var types []accounting.TxType
	for _, g := range groups {
		switch strings.ToLower(strings.TrimSpace(g)) {
		case investmentGroupStr:
			types = append(types,
				accounting.StockBuyTransaction,
				accounting.StockSellTransaction,
				accounting.StockGrantTransaction,
				accounting.StockTransferTransaction,
				accounting.StockVestTransaction,
				accounting.StockForfeitTransaction,
			)
		default:
			if t := parseTxType(g); t != accounting.UnknownTransaction {
				types = append(types, t)
			}
		}
	}
	return types
}

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
	case balanceStatusTxStr:
		return accounting.BalanceStatusTransaction
	case stockVestTxStr:
		return accounting.StockVestTransaction
	case stockForfeitTxStr:
		return accounting.StockForfeitTransaction
	case revaluationTxStr:
		return accounting.RevaluationTransaction
	default:
		return accounting.UnknownTransaction
	}
}
