package finance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/andresbott/etna/internal/accounting"
	"github.com/gorilla/mux"
)

type positionPayload struct {
	Id           uint    `json:"id"`
	AccountID    uint    `json:"accountId"`
	InstrumentID uint    `json:"instrumentId"`
	Quantity     float64 `json:"quantity"`
	CostBasis    float64 `json:"costBasis"`
	AvgCost      float64 `json:"avgCost"`
}

type lotPayload struct {
	Id           uint    `json:"id"`
	TradeID      uint    `json:"tradeId"`
	AccountID    uint    `json:"accountId"`
	InstrumentID uint    `json:"instrumentId"`
	OpenDate     string  `json:"openDate"`
	Quantity     float64 `json:"quantity"`
	OriginalQty  float64 `json:"originalQty"`
	CostPerShare float64 `json:"costPerShare"`
	CostBasis    float64 `json:"costBasis"`
	Status       int     `json:"status"`
	ClosedDate   *string `json:"closedDate,omitempty"`
}

type tradePayload struct {
	Id            uint    `json:"id"`
	TransactionID uint    `json:"transactionId"`
	AccountID     uint    `json:"accountId"`
	InstrumentID  uint    `json:"instrumentId"`
	TradeType     int     `json:"tradeType"`
	Quantity      float64 `json:"quantity"`
	PricePerShare float64 `json:"pricePerShare"`
	TotalAmount   float64 `json:"totalAmount"`
	Currency      string  `json:"currency"`
	Date          string  `json:"date"`
}

func (h *Handler) ListPositions() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var opts accounting.ListPositionsOpts

		if accountIdStr := r.URL.Query().Get("accountId"); accountIdStr != "" {
			id, err := strconv.ParseUint(accountIdStr, 10, 64)
			if err != nil {
				http.Error(w, "invalid accountId", http.StatusBadRequest)
				return
			}
			opts.AccountID = uint(id)
		}

		positions, err := h.Store.ListPositions(r.Context(), opts)
		if err != nil {
			http.Error(w, fmt.Sprintf("error listing positions: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		payload := make([]positionPayload, len(positions))
		for i, p := range positions {
			payload[i] = positionPayload{
				Id:           p.Id,
				AccountID:    p.AccountID,
				InstrumentID: p.InstrumentID,
				Quantity:     p.Quantity,
				CostBasis:    p.CostBasis,
				AvgCost:      p.AvgCost,
			}
		}

		respJSON, err := json.Marshal(map[string]interface{}{"items": payload})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}

func (h *Handler) GetPositionDetail() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		accountId, err := strconv.ParseUint(vars["accountId"], 10, 64)
		if err != nil {
			http.Error(w, "invalid accountId", http.StatusBadRequest)
			return
		}
		instrumentId, err := strconv.ParseUint(vars["instrumentId"], 10, 64)
		if err != nil {
			http.Error(w, "invalid instrumentId", http.StatusBadRequest)
			return
		}

		pos, err := h.Store.GetPosition(r.Context(), uint(accountId), uint(instrumentId))
		if err != nil {
			http.Error(w, fmt.Sprintf("error getting position: %s", err.Error()), http.StatusNotFound)
			return
		}

		// Also fetch lots for detail
		openStatus := accounting.LotOpen
		lots, err := h.Store.ListLots(r.Context(), accounting.ListLotsOpts{
			AccountID:    uint(accountId),
			InstrumentID: uint(instrumentId),
			Status:       &openStatus,
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("error listing lots: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		lotPayloads := make([]lotPayload, len(lots))
		for i, l := range lots {
			lp := lotPayload{
				Id:           l.Id,
				TradeID:      l.TradeID,
				AccountID:    l.AccountID,
				InstrumentID: l.InstrumentID,
				OpenDate:     l.OpenDate.Format("2006-01-02"),
				Quantity:     l.Quantity,
				OriginalQty:  l.OriginalQty,
				CostPerShare: l.CostPerShare,
				CostBasis:    l.CostBasis,
				Status:       int(l.Status),
			}
			if l.ClosedDate != nil {
				s := l.ClosedDate.Format("2006-01-02")
				lp.ClosedDate = &s
			}
			lotPayloads[i] = lp
		}

		resp := map[string]interface{}{
			"position": positionPayload{
				Id:           pos.Id,
				AccountID:    pos.AccountID,
				InstrumentID: pos.InstrumentID,
				Quantity:     pos.Quantity,
				CostBasis:    pos.CostBasis,
				AvgCost:      pos.AvgCost,
			},
			"lots": lotPayloads,
		}

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

func (h *Handler) ListLots() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var opts accounting.ListLotsOpts

		if s := r.URL.Query().Get("accountId"); s != "" {
			id, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				http.Error(w, "invalid accountId", http.StatusBadRequest)
				return
			}
			opts.AccountID = uint(id)
		}
		if s := r.URL.Query().Get("instrumentId"); s != "" {
			id, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				http.Error(w, "invalid instrumentId", http.StatusBadRequest)
				return
			}
			opts.InstrumentID = uint(id)
		}

		lots, err := h.Store.ListLots(r.Context(), opts)
		if err != nil {
			http.Error(w, fmt.Sprintf("error listing lots: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		payload := make([]lotPayload, len(lots))
		for i, l := range lots {
			lp := lotPayload{
				Id:           l.Id,
				TradeID:      l.TradeID,
				AccountID:    l.AccountID,
				InstrumentID: l.InstrumentID,
				OpenDate:     l.OpenDate.Format("2006-01-02"),
				Quantity:     l.Quantity,
				OriginalQty:  l.OriginalQty,
				CostPerShare: l.CostPerShare,
				CostBasis:    l.CostBasis,
				Status:       int(l.Status),
			}
			if l.ClosedDate != nil {
				s := l.ClosedDate.Format("2006-01-02")
				lp.ClosedDate = &s
			}
			payload[i] = lp
		}

		respJSON, err := json.Marshal(map[string]interface{}{"items": payload})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}

func (h *Handler) ListPortfolioTrades() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var opts accounting.ListTradesOpts

		if s := r.URL.Query().Get("accountId"); s != "" {
			id, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				http.Error(w, "invalid accountId", http.StatusBadRequest)
				return
			}
			opts.AccountID = uint(id)
		}
		if s := r.URL.Query().Get("instrumentId"); s != "" {
			id, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				http.Error(w, "invalid instrumentId", http.StatusBadRequest)
				return
			}
			opts.InstrumentID = uint(id)
		}

		// Parse optional date range
		if s := r.URL.Query().Get("startDate"); s != "" {
			startDate, err := parseDateParam(s)
			if err != nil {
				http.Error(w, "invalid startDate", http.StatusBadRequest)
				return
			}
			opts.StartDate = startDate
		}
		if s := r.URL.Query().Get("endDate"); s != "" {
			endDate, err := parseDateParam(s)
			if err != nil {
				http.Error(w, "invalid endDate", http.StatusBadRequest)
				return
			}
			opts.EndDate = endDate
		}

		trades, err := h.Store.ListTrades(r.Context(), opts)
		if err != nil {
			http.Error(w, fmt.Sprintf("error listing trades: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		payload := make([]tradePayload, len(trades))
		for i, t := range trades {
			payload[i] = tradePayload{
				Id:            t.Id,
				TransactionID: t.TransactionID,
				AccountID:     t.AccountID,
				InstrumentID:  t.InstrumentID,
				TradeType:     int(t.TradeType),
				Quantity:      t.Quantity,
				PricePerShare: t.PricePerShare,
				TotalAmount:   t.TotalAmount,
				Currency:      t.Currency,
				Date:          t.Date.Format("2006-01-02"),
			}
		}

		respJSON, err := json.Marshal(map[string]interface{}{"items": payload})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}

func parseDateParam(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	return time.Parse("2006-01-02", s)
}
