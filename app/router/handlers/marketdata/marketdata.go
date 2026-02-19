package marketdata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/andresbott/etna/internal/marketdata"
)

// Handler serves market data HTTP endpoints backed by a marketdata.Store.
type Handler struct {
	Store *marketdata.Store
}

type pricePayload struct {
	ID     uint    `json:"id"`
	Symbol string  `json:"symbol"`
	Time   string  `json:"time"`
	Price  float64 `json:"price"`
}

type priceCreatePayload struct {
	Time  string  `json:"time"`
	Price float64 `json:"price"`
}

type priceUpdatePayload struct {
	Time  *string  `json:"time,omitempty"`
	Price *float64 `json:"price,omitempty"`
}

type bulkCreatePayload struct {
	Points []priceCreatePayload `json:"points"`
}

const timeLayout = "2006-01-02"

// ListSymbols returns the list of instrument symbols that have price data.
func (h *Handler) ListSymbols() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		symbols, err := h.Store.ListPriceSymbols()
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to list symbols: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		type response struct {
			Symbols []string `json:"symbols"`
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response{Symbols: symbols}); err != nil {
			http.Error(w, fmt.Sprintf("error encoding JSON: %s", err.Error()), http.StatusInternalServerError)
		}
	})
}

func recordToPayload(r marketdata.PriceRecord) pricePayload {
	return pricePayload{
		ID:     r.ID,
		Symbol: r.Symbol,
		Time:   r.Time.Format(timeLayout),
		Price:  r.Price,
	}
}

// ListPrices returns the price history for the given symbol.
// Query parameters "start" and "end" (YYYY-MM-DD) bound the range.
func (h *Handler) ListPrices(symbol string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if symbol == "" {
			http.Error(w, "symbol is required", http.StatusBadRequest)
			return
		}

		var start, end time.Time
		if v := r.URL.Query().Get("start"); v != "" {
			t, err := time.Parse(timeLayout, v)
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid start date: %s", err.Error()), http.StatusBadRequest)
				return
			}
			start = t
		}
		if v := r.URL.Query().Get("end"); v != "" {
			t, err := time.Parse(timeLayout, v)
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid end date: %s", err.Error()), http.StatusBadRequest)
				return
			}
			end = t
		}

		records, err := h.Store.PriceHistory(r.Context(), symbol, start, end)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to list prices: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		out := make([]pricePayload, len(records))
		for i, rec := range records {
			out[i] = recordToPayload(rec)
		}

		type response struct {
			Items []pricePayload `json:"items"`
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response{Items: out}); err != nil {
			http.Error(w, fmt.Sprintf("error encoding JSON: %s", err.Error()), http.StatusInternalServerError)
		}
	})
}

// LatestPrice returns the most recent price record for the given symbol.
func (h *Handler) LatestPrice(symbol string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if symbol == "" {
			http.Error(w, "symbol is required", http.StatusBadRequest)
			return
		}

		rec, err := h.Store.LatestPrice(r.Context(), symbol)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to get latest price: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		if rec == nil {
			http.Error(w, "no price data found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(recordToPayload(*rec))
	})
}

// CreatePrice ingests a single price point for the given symbol.
func (h *Handler) CreatePrice(symbol string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if symbol == "" {
			http.Error(w, "symbol is required", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		var payload priceCreatePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		t, err := time.Parse(timeLayout, payload.Time)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid time: %s", err.Error()), http.StatusBadRequest)
			return
		}

		if err := h.Store.IngestPrice(r.Context(), symbol, t, payload.Price); err != nil {
			http.Error(w, fmt.Sprintf("unable to ingest price: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(pricePayload{
			Symbol: symbol,
			Time:   t.Format(timeLayout),
			Price:  payload.Price,
		})
	})
}

// CreatePricesBulk ingests multiple price points for the given symbol.
func (h *Handler) CreatePricesBulk(symbol string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if symbol == "" {
			http.Error(w, "symbol is required", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		var payload bulkCreatePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}
		if len(payload.Points) == 0 {
			http.Error(w, "no price points provided", http.StatusBadRequest)
			return
		}

		points := make([]marketdata.PricePoint, len(payload.Points))
		for i, p := range payload.Points {
			t, err := time.Parse(timeLayout, p.Time)
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid time at index %d: %s", i, err.Error()), http.StatusBadRequest)
				return
			}
			points[i] = marketdata.PricePoint{Time: t, Price: p.Price}
		}

		if err := h.Store.IngestPricesBulk(r.Context(), symbol, points); err != nil {
			http.Error(w, fmt.Sprintf("unable to bulk ingest prices: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	})
}

// UpdatePrice applies a partial update to a price record.
func (h *Handler) UpdatePrice(id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id == 0 {
			http.Error(w, "valid record id is required", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		var payload priceUpdatePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		var update marketdata.PriceUpdate
		if payload.Time != nil {
			t, err := time.Parse(timeLayout, *payload.Time)
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid time: %s", err.Error()), http.StatusBadRequest)
				return
			}
			update.Time = &t
		}
		update.Price = payload.Price

		if err := h.Store.UpdatePrice(r.Context(), id, update); err != nil {
			http.Error(w, fmt.Sprintf("unable to update price: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}

// DeletePrice removes a price record by ID.
func (h *Handler) DeletePrice(id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id == 0 {
			http.Error(w, "valid record id is required", http.StatusBadRequest)
			return
		}

		if err := h.Store.DeletePrice(r.Context(), id); err != nil {
			http.Error(w, fmt.Sprintf("unable to delete price: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}
