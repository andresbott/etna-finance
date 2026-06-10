package finance

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/andresbott/etna/internal/marketdata"
	"github.com/andresbott/etna/internal/marketdata/importer"
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
	Notes    string `json:"notes"`
	Type     string `json:"type"`
	Exchange string `json:"exchange"`
}

type instrumentCreatePayload struct {
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
	Notes    string `json:"notes"`
	Type     string `json:"type"`
	Exchange string `json:"exchange"`
}

func instrumentToPayload(s marketdata.Instrument) instrumentPayload {
	return instrumentPayload{
		ID:       s.ID,
		Symbol:   s.Symbol,
		Name:     s.Name,
		Currency: s.Currency.String(),
		Notes:    s.Notes,
		Type:     s.Type,
		Exchange: s.Exchange,
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

		if payload.Type == "" {
			http.Error(w, "type cannot be empty", http.StatusBadRequest)
			return
		}
		if payload.Exchange == "" {
			http.Error(w, "exchange cannot be empty", http.StatusBadRequest)
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
			Notes:    payload.Notes,
			Type:     payload.Type,
			Exchange: payload.Exchange,
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
			Notes:    item.Notes,
			Type:     item.Type,
			Exchange: item.Exchange,
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
	Notes    *string `json:"notes,omitempty"`
	Type     *string `json:"type,omitempty"`
	Exchange *string `json:"exchange,omitempty"`
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
		if payload.Type != nil && *payload.Type == "" {
			http.Error(w, "type cannot be empty", http.StatusBadRequest)
			return
		}
		if payload.Exchange != nil && *payload.Exchange == "" {
			http.Error(w, "exchange cannot be empty", http.StatusBadRequest)
			return
		}
		item := marketdata.InstrumentUpdatePayload{
			Symbol:   payload.Symbol,
			Name:     payload.Name,
			Currency: payload.Currency,
			Notes:    payload.Notes,
			Type:     payload.Type,
			Exchange: payload.Exchange,
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

// ---------------------------------------------------------------------------
// Instrument lookup (reference provider)
// ---------------------------------------------------------------------------

type instrumentLookupResponse struct {
	Name     string `json:"name"`
	Currency string `json:"currency"`
	Type     string `json:"type"`
	Exchange string `json:"exchange"`
	Notes    string `json:"notes"`
}

// massiveTypeToAppType maps raw Massive ticker type codes to the app's Type values.
// Anything not listed passes through unchanged (shown via the dialog's "Other" field).
var massiveTypeToAppType = map[string]string{
	"CS":   "Stock",
	"ADRC": "Stock",
	"ETF":  "ETF",
	"BOND": "Bond",
}

// massiveMicToExchange maps Massive primary-exchange MIC codes to the app's exchange names.
// Anything not listed passes through unchanged (shown via the dialog's "Other" field).
var massiveMicToExchange = map[string]string{
	"XNYS": "NYSE",
	"XNAS": "NASDAQ",
	"XLON": "LSE",
	"XTSE": "TSX",
	"XPAR": "Euronext",
	"XAMS": "Euronext",
	"XBRU": "Euronext",
	"XETR": "XETRA",
	"XSWX": "SIX",
	"XTKS": "JPX (Tokyo)",
	"XHKG": "HKEX",
	"XASX": "ASX",
	"XMAD": "BME (Madrid)",
	"XMIL": "Borsa Italiana",
	"XSTO": "Nasdaq Nordic",
}

func mapTickerDetails(d importer.TickerDetails) instrumentLookupResponse {
	t := d.Type
	if mapped, ok := massiveTypeToAppType[d.Type]; ok {
		t = mapped
	}
	ex := d.Exchange
	if mapped, ok := massiveMicToExchange[d.Exchange]; ok {
		ex = mapped
	}
	return instrumentLookupResponse{
		Name:     d.Name,
		Currency: d.Currency,
		Type:     t,
		Exchange: ex,
		Notes:    d.Notes,
	}
}

// LookupInstrument returns suggested instrument details for a symbol from the reference
// provider. Returns 204 when no provider is configured or the symbol is not found.
func (h *Handler) LookupInstrument() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		if symbol == "" {
			http.Error(w, "symbol query parameter is required", http.StatusBadRequest)
			return
		}
		if h.Reference == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		details, err := h.Reference.GetTickerDetails(r.Context(), symbol)
		if err != nil || !details.Found {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mapTickerDetails(details))
	})
}
