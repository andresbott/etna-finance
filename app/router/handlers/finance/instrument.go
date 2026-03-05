package finance

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/andresbott/etna/internal/marketdata"
	"golang.org/x/text/currency"
)

// ---------------------------------------------------------------------------
// Instrument (marketdata)
// ---------------------------------------------------------------------------

type instrumentPayload struct {
	ID       uint   `json:"id"`
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
}

type instrumentCreatePayload struct {
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
}

func instrumentToPayload(s marketdata.Instrument) instrumentPayload {
	return instrumentPayload{
		ID:       s.ID,
		Symbol:   s.Symbol,
		Name:     s.Name,
		Currency: s.Currency.String(),
	}
}

var instrumentValidationErr = marketdata.ErrValidation("")

// checkBody writes 400 if r.Body is nil and returns false; otherwise returns true.
func checkBody(w http.ResponseWriter, r *http.Request) bool {
	if r.Body == nil {
		http.Error(w, "request had empty body", http.StatusBadRequest)
		return false
	}
	return true
}

func (h *Handler) ListInstruments() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.InstrumentStore == nil {
			http.Error(w, "instruments not available", http.StatusServiceUnavailable)
			return
		}

		items, err := h.InstrumentStore.ListInstruments(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to list instruments: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		out := make([]instrumentPayload, len(items))
		for i, item := range items {
			out[i] = instrumentToPayload(item)
		}

		type response struct {
			Items []instrumentPayload `json:"items"`
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response{Items: out}); err != nil {
			http.Error(w, fmt.Sprintf("error encoding JSON: %s", err.Error()), http.StatusInternalServerError)
		}
	})
}

func (h *Handler) CreateInstrument() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}
		if h.InstrumentStore == nil {
			http.Error(w, "instruments not available", http.StatusServiceUnavailable)
			return
		}

		var payload instrumentCreatePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		curr, err := currency.ParseISO(payload.Currency)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid currency: %s", payload.Currency), http.StatusBadRequest)
			return
		}

		item := marketdata.Instrument{
			Symbol:   payload.Symbol,
			Name:     payload.Name,
			Currency: curr,
		}

		id, err := h.InstrumentStore.CreateInstrument(r.Context(), item)
		if err != nil {
			if errors.As(err, &instrumentValidationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if errors.Is(err, marketdata.ErrInstrumentSymbolDuplicate) {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			http.Error(w, fmt.Sprintf("unable to create instrument: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		out := instrumentPayload{
			ID:       id,
			Symbol:   item.Symbol,
			Name:     item.Name,
			Currency: item.Currency.String(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(out)
	})
}

func (h *Handler) GetInstrument(id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.InstrumentStore == nil {
			http.Error(w, "instruments not available", http.StatusServiceUnavailable)
			return
		}

		item, err := h.InstrumentStore.GetInstrument(r.Context(), id)
		if err != nil {
			if errors.Is(err, marketdata.ErrInstrumentNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to get instrument: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(instrumentToPayload(item))
	})
}

type instrumentUpdatePayload struct {
	Symbol   *string `json:"symbol,omitempty"`
	Name     *string `json:"name,omitempty"`
	Currency *string `json:"currency,omitempty"`
}

func (h *Handler) UpdateInstrument(id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !checkBody(w, r) {
			return
		}
		if h.InstrumentStore == nil {
			http.Error(w, "instruments not available", http.StatusServiceUnavailable)
			return
		}
		var payload instrumentUpdatePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}
		item := marketdata.InstrumentUpdatePayload{
			Symbol:   payload.Symbol,
			Name:     payload.Name,
			Currency: payload.Currency,
		}
		err := h.InstrumentStore.UpdateInstrument(r.Context(), id, item)
		if err != nil {
			if errors.As(err, &instrumentValidationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if errors.Is(err, marketdata.ErrNoChanges) {
				http.Error(w, "no changes applied", http.StatusBadRequest)
				return
			}
			if errors.Is(err, marketdata.ErrInstrumentNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			if errors.Is(err, marketdata.ErrInstrumentSymbolDuplicate) {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			http.Error(w, fmt.Sprintf("unable to update instrument: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func (h *Handler) DeleteInstrument(id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.InstrumentStore == nil {
			http.Error(w, "instruments not available", http.StatusServiceUnavailable)
			return
		}

		err := h.InstrumentStore.DeleteInstrument(r.Context(), id)
		if err != nil {
			if errors.Is(err, marketdata.ErrInstrumentNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to delete instrument: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
