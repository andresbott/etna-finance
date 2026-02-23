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
	Store        *marketdata.Store
	MainCurrency string   // for FX: main currency from settings
	Currencies   []string // for FX: all configured currencies (includes main)
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

// =============================================================================
// Currency exchange (FX) endpoints
// =============================================================================

type fxPairPayload struct {
	Pair string `json:"pair"` // "MAIN/SECONDARY"
}

type fxRatePayload struct {
	ID        uint    `json:"id"`
	Main      string  `json:"main"`
	Secondary string  `json:"secondary"`
	Time      string  `json:"time"`
	Rate      float64 `json:"rate"`
}

type fxRateCreatePayload struct {
	Time string  `json:"time"`
	Rate float64 `json:"rate"`
}

type fxRateUpdatePayload struct {
	Time *string  `json:"time,omitempty"`
	Rate *float64 `json:"rate,omitempty"`
}

type fxBulkCreatePayload struct {
	Points []fxRateCreatePayload `json:"points"`
}

// ListFXPairs returns configured currency pairs (main + each secondary from settings).
// Response: { "pairs": ["CHF/USD", "CHF/EUR", ...] }
func (h *Handler) ListFXPairs() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		main := h.MainCurrency
		if main == "" {
			main = "CHF"
		}
		currencies := h.Currencies
		if len(currencies) == 0 {
			currencies = []string{"CHF"}
		}
		var pairs []string
		for _, c := range currencies {
			if c != main {
				pairs = append(pairs, main+"/"+c)
			}
		}
		type response struct {
			Pairs []string `json:"pairs"`
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response{Pairs: pairs})
	})
}

func fxRecordToPayload(rec marketdata.RateRecord) fxRatePayload {
	return fxRatePayload{
		ID:        rec.ID,
		Main:      rec.Main,
		Secondary: rec.Secondary,
		Time:      rec.Time.Format(timeLayout),
		Rate:      rec.Rate,
	}
}

// ListFXRates returns rate history for a pair. Path: {main}/{secondary}/rates?start=...&end=...
func (h *Handler) ListFXRates(main, secondary string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if main == "" || secondary == "" {
			http.Error(w, "main and secondary currency are required", http.StatusBadRequest)
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
			// Treat end as inclusive of that calendar day: end of day so time <= end includes records on that date
			end = t.Add(24*time.Hour - time.Nanosecond)
		}
		records, err := h.Store.RateHistory(r.Context(), main, secondary, start, end)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to list rates: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		out := make([]fxRatePayload, len(records))
		for i, rec := range records {
			out[i] = fxRecordToPayload(rec)
		}
		type response struct {
			Items []fxRatePayload `json:"items"`
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response{Items: out})
	})
}

// LatestFXRate returns the most recent rate for the pair.
func (h *Handler) LatestFXRate(main, secondary string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if main == "" || secondary == "" {
			http.Error(w, "main and secondary currency are required", http.StatusBadRequest)
			return
		}
		rec, err := h.Store.LatestRate(r.Context(), main, secondary)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to get latest rate: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		if rec == nil {
			http.Error(w, "no rate data found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(fxRecordToPayload(*rec))
	})
}

// CreateFXRate ingests a single rate for the pair.
func (h *Handler) CreateFXRate(main, secondary string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if main == "" || secondary == "" {
			http.Error(w, "main and secondary currency are required", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}
		var payload fxRateCreatePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}
		t, err := time.Parse(timeLayout, payload.Time)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid time: %s", err.Error()), http.StatusBadRequest)
			return
		}
		if err := h.Store.IngestRate(r.Context(), main, secondary, t, payload.Rate); err != nil {
			http.Error(w, fmt.Sprintf("unable to ingest rate: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(fxRatePayload{
			Main: main, Secondary: secondary,
			Time: t.Format(timeLayout), Rate: payload.Rate,
		})
	})
}

// CreateFXRatesBulk ingests multiple rate points for the pair.
func (h *Handler) CreateFXRatesBulk(main, secondary string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if main == "" || secondary == "" {
			http.Error(w, "main and secondary currency are required", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}
		var payload fxBulkCreatePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}
		if len(payload.Points) == 0 {
			http.Error(w, "no rate points provided", http.StatusBadRequest)
			return
		}
		points := make([]marketdata.RatePoint, len(payload.Points))
		for i, p := range payload.Points {
			t, err := time.Parse(timeLayout, p.Time)
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid time at index %d: %s", i, err.Error()), http.StatusBadRequest)
				return
			}
			points[i] = marketdata.RatePoint{Time: t, Rate: p.Rate}
		}
		if err := h.Store.IngestRatesBulk(r.Context(), main, secondary, points); err != nil {
			http.Error(w, fmt.Sprintf("unable to bulk ingest rates: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})
}

// UpdateFXRate applies a partial update to a rate record.
func (h *Handler) UpdateFXRate(id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id == 0 {
			http.Error(w, "valid record id is required", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}
		var payload fxRateUpdatePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}
		var update marketdata.RateUpdate
		if payload.Time != nil {
			t, err := time.Parse(timeLayout, *payload.Time)
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid time: %s", err.Error()), http.StatusBadRequest)
				return
			}
			update.Time = &t
		}
		update.Rate = payload.Rate
		if err := h.Store.UpdateRate(r.Context(), id, update); err != nil {
			http.Error(w, fmt.Sprintf("unable to update rate: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

// DeleteFXRate removes a rate record by ID.
func (h *Handler) DeleteFXRate(id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id == 0 {
			http.Error(w, "valid record id is required", http.StatusBadRequest)
			return
		}
		if err := h.Store.DeleteRate(r.Context(), id); err != nil {
			http.Error(w, fmt.Sprintf("unable to delete rate: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
