package marketdata

import (
	"context"
	"encoding/json"
	"errors"
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
	Symbol string  `json:"symbol"`
	Time   string  `json:"time"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}

type priceCreatePayload struct {
	Time   string  `json:"time"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}

type bulkCreatePayload struct {
	Points []priceCreatePayload `json:"points"`
}

const timeLayout = "2006-01-02"

func (p priceCreatePayload) toPoint() (marketdata.PricePoint, error) {
	t, err := time.Parse(timeLayout, p.Time)
	if err != nil {
		return marketdata.PricePoint{}, err
	}
	return marketdata.PricePoint{Time: t, Open: p.Open, High: p.High, Low: p.Low, Close: p.Close, Volume: p.Volume}, nil
}

// ListSymbols returns the list of instrument symbols that have price data.
func (h *Handler) ListSymbols() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		symbols, err := h.Store.ListPriceSymbols(r.Context())
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
		Symbol: r.Symbol,
		Time:   r.Time.Format(timeLayout),
		Open:   r.Open, High: r.High, Low: r.Low, Close: r.Close, Volume: r.Volume,
	}
}

// ListPrices returns the price history for the given symbol.
// Query parameters "start" and "end" (YYYY-MM-DD) bound the range.
//
//nolint:dupl // intentionally parallel to ListEPS; the price and EPS read handlers mirror each other
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
			// Treat end as inclusive of that calendar day
			end = t.Add(24*time.Hour - time.Nanosecond)
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

		pt, err := payload.toPoint()
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid time: %s", err.Error()), http.StatusBadRequest)
			return
		}

		if err := h.Store.IngestPrice(r.Context(), symbol, pt); err != nil {
			http.Error(w, fmt.Sprintf("unable to ingest price: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(recordToPayload(marketdata.PriceRecord{Symbol: symbol, Time: pt.Time, Open: pt.Open, High: pt.High, Low: pt.Low, Close: pt.Close, Volume: pt.Volume}))
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
			pt, err := p.toPoint()
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid time at index %d: %s", i, err.Error()), http.StatusBadRequest)
				return
			}
			points[i] = pt
		}

		if err := h.Store.IngestPricesBulk(r.Context(), symbol, points); err != nil {
			http.Error(w, fmt.Sprintf("unable to bulk ingest prices: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	})
}

// editTimeseriesRecord is the shared body of EditPrice/EditEPS. It validates the path params, parses
// {origDate}, decodes the body into a payload, converts it to a point and applies the store edit.
// recordNoun ("price"/"EPS") and immutableMsg tailor the user-facing errors; the date is the
// record's identity, so a body time that differs from {origDate} surfaces as ErrDateImmutable → 400.
func editTimeseriesRecord[T, P any](
	w http.ResponseWriter,
	r *http.Request,
	symbol, origDate, recordNoun, immutableMsg string,
	toPoint func(T) (P, error),
	edit func(context.Context, string, time.Time, P) error,
) {
	if symbol == "" || origDate == "" {
		http.Error(w, "symbol and date are required", http.StatusBadRequest)
		return
	}
	oldTime, err := time.Parse(timeLayout, origDate)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid date: %s", err.Error()), http.StatusBadRequest)
		return
	}
	if r.Body == nil {
		http.Error(w, "request had empty body", http.StatusBadRequest)
		return
	}
	var payload T
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
		return
	}
	pt, err := toPoint(payload)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid time: %s", err.Error()), http.StatusBadRequest)
		return
	}
	if err := edit(r.Context(), symbol, oldTime, pt); err != nil {
		if errors.Is(err, marketdata.ErrDateImmutable) {
			http.Error(w, immutableMsg, http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("unable to update %s: %s", recordNoun, err.Error()), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// EditPrice upserts the candle for {symbol} at {date}. The body carries the full candle; its time
// must match {date} — the date is the record's identity and an edit cannot change it (returns 400).
func (h *Handler) EditPrice(symbol, origDate string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		editTimeseriesRecord(w, r, symbol, origDate, "price",
			"a price record's date cannot be changed; delete it and create a new one",
			priceCreatePayload.toPoint, h.Store.EditPrice)
	})
}

// deleteTimeseriesRecord is the shared body of DeletePrice/DeleteEPS. It validates the path params,
// parses {date} and applies the store delete. recordNoun ("price"/"EPS") and notFoundMsg tailor the
// user-facing errors; a missing record surfaces as ErrRecordNotFound → 404.
func deleteTimeseriesRecord(
	w http.ResponseWriter,
	r *http.Request,
	symbol, date, recordNoun, notFoundMsg string,
	del func(context.Context, string, time.Time) error,
) {
	if symbol == "" || date == "" {
		http.Error(w, "symbol and date are required", http.StatusBadRequest)
		return
	}
	t, err := time.Parse(timeLayout, date)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid date: %s", err.Error()), http.StatusBadRequest)
		return
	}
	if err := del(r.Context(), symbol, t); err != nil {
		if errors.Is(err, marketdata.ErrRecordNotFound) {
			http.Error(w, notFoundMsg, http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("unable to delete %s: %s", recordNoun, err.Error()), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// DeletePrice removes the candle for {symbol} at the given {date}.
func (h *Handler) DeletePrice(symbol, date string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deleteTimeseriesRecord(w, r, symbol, date, "price", "no price data found", h.Store.DeletePriceAt)
	})
}

// =============================================================================
// EPS endpoints (mirrors the price endpoints; powers the TTM P·E chart line)
// =============================================================================

type epsPayload struct {
	Time       string  `json:"time"`
	EPSBasic   float64 `json:"eps_basic"`
	EPSDiluted float64 `json:"eps_diluted"`
}

func (p epsPayload) toPoint() (marketdata.EPSPoint, error) {
	t, err := time.Parse(timeLayout, p.Time)
	if err != nil {
		return marketdata.EPSPoint{}, err
	}
	return marketdata.EPSPoint{Time: t, Basic: p.EPSBasic, Diluted: p.EPSDiluted}, nil
}

type epsBulkCreatePayload struct {
	Points []epsPayload `json:"points"`
}

func epsRecordToPayload(r marketdata.EPSRecord) epsPayload {
	return epsPayload{Time: r.Time.Format(timeLayout), EPSBasic: r.Basic, EPSDiluted: r.Diluted}
}

// ListEPS returns the EPS history for the given symbol. Query params "start"/"end" (YYYY-MM-DD) bound the range.
//
//nolint:dupl // intentionally parallel to ListPrices; the price and EPS read handlers mirror each other
func (h *Handler) ListEPS(symbol string) http.Handler {
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
			end = t.Add(24*time.Hour - time.Nanosecond) // inclusive of that calendar day
		}
		records, err := h.Store.EPSHistory(r.Context(), symbol, start, end)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to list EPS: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		out := make([]epsPayload, len(records))
		for i, rec := range records {
			out[i] = epsRecordToPayload(rec)
		}
		type response struct {
			Items []epsPayload `json:"items"`
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response{Items: out}); err != nil {
			http.Error(w, fmt.Sprintf("error encoding JSON: %s", err.Error()), http.StatusInternalServerError)
		}
	})
}

// LatestEPS returns the most recent EPS record for the given symbol.
func (h *Handler) LatestEPS(symbol string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if symbol == "" {
			http.Error(w, "symbol is required", http.StatusBadRequest)
			return
		}
		rec, err := h.Store.LatestEPS(r.Context(), symbol)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to get latest EPS: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		if rec == nil {
			http.Error(w, "no EPS data found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(epsRecordToPayload(*rec))
	})
}

// CreateEPS ingests a single EPS observation for the given symbol.
func (h *Handler) CreateEPS(symbol string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if symbol == "" {
			http.Error(w, "symbol is required", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}
		var payload epsPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}
		pt, err := payload.toPoint()
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid time: %s", err.Error()), http.StatusBadRequest)
			return
		}
		// Adding the first EPS point is how the series is introduced for a symbol that was not
		// auto-defined (e.g. a manually annotated non-stock), so ensure the series exists.
		if err := h.Store.RegisterEPSSeries(r.Context(), symbol); err != nil {
			http.Error(w, fmt.Sprintf("unable to register EPS series: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		if err := h.Store.IngestEPS(r.Context(), symbol, pt); err != nil {
			http.Error(w, fmt.Sprintf("unable to ingest EPS: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(epsRecordToPayload(marketdata.EPSRecord{Symbol: symbol, Time: pt.Time, Basic: pt.Basic, Diluted: pt.Diluted}))
	})
}

// CreateEPSBulk ingests multiple EPS observations for the given symbol.
func (h *Handler) CreateEPSBulk(symbol string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if symbol == "" {
			http.Error(w, "symbol is required", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}
		var payload epsBulkCreatePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}
		if len(payload.Points) == 0 {
			http.Error(w, "no EPS points provided", http.StatusBadRequest)
			return
		}
		points := make([]marketdata.EPSPoint, len(payload.Points))
		for i, p := range payload.Points {
			pt, err := p.toPoint()
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid time at index %d: %s", i, err.Error()), http.StatusBadRequest)
				return
			}
			points[i] = pt
		}
		// Adding the first EPS point is how the series is introduced for a symbol that was not
		// auto-defined (e.g. a manually annotated non-stock), so ensure the series exists.
		if err := h.Store.RegisterEPSSeries(r.Context(), symbol); err != nil {
			http.Error(w, fmt.Sprintf("unable to register EPS series: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		if err := h.Store.IngestEPSBulk(r.Context(), symbol, points); err != nil {
			http.Error(w, fmt.Sprintf("unable to bulk ingest EPS: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})
}

// EditEPS upserts the EPS observation for {symbol} at {date}. The body carries the full point; its
// time must match {date} — the date is the record's identity and an edit cannot change it (400).
// Mirrors EditPrice.
func (h *Handler) EditEPS(symbol, origDate string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		editTimeseriesRecord(w, r, symbol, origDate, "EPS",
			"an EPS record's date cannot be changed; delete it and create a new one",
			epsPayload.toPoint, h.Store.EditEPS)
	})
}

// DeleteEPS removes the EPS observation for {symbol} at the given {date}. Mirrors DeletePrice.
func (h *Handler) DeleteEPS(symbol, date string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deleteTimeseriesRecord(w, r, symbol, date, "EPS", "no EPS data found", h.Store.DeleteEPSAt)
	})
}

// =============================================================================
// Currency exchange (FX) endpoints
// =============================================================================

type fxRatePayload struct {
	Main      string  `json:"main"`
	Secondary string  `json:"secondary"`
	Time      string  `json:"time"`
	Rate      float64 `json:"rate"`
}

type fxRateCreatePayload struct {
	Time string  `json:"time"`
	Rate float64 `json:"rate"`
}

func (p fxRateCreatePayload) toPoint() (marketdata.RatePoint, error) {
	t, err := time.Parse(timeLayout, p.Time)
	if err != nil {
		return marketdata.RatePoint{}, err
	}
	return marketdata.RatePoint{Time: t, Rate: p.Rate}, nil
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
		pt, err := payload.toPoint()
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid time: %s", err.Error()), http.StatusBadRequest)
			return
		}
		// Adding a rate is how an FX pair is first introduced via the API, so ensure the series exists.
		if err := h.Store.RegisterPair(r.Context(), main, secondary); err != nil {
			http.Error(w, fmt.Sprintf("unable to register pair: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		if err := h.Store.IngestRate(r.Context(), main, secondary, pt.Time, pt.Rate); err != nil {
			http.Error(w, fmt.Sprintf("unable to ingest rate: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(fxRatePayload{
			Main: main, Secondary: secondary,
			Time: pt.Time.Format(timeLayout), Rate: pt.Rate,
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
			pt, err := p.toPoint()
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid time at index %d: %s", i, err.Error()), http.StatusBadRequest)
				return
			}
			points[i] = pt
		}
		// Adding rates is how an FX pair is first introduced via the API, so ensure the series exists.
		if err := h.Store.RegisterPair(r.Context(), main, secondary); err != nil {
			http.Error(w, fmt.Sprintf("unable to register pair: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		if err := h.Store.IngestRatesBulk(r.Context(), main, secondary, points); err != nil {
			http.Error(w, fmt.Sprintf("unable to bulk ingest rates: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})
}

// EditFXRate upserts the rate for {main}/{secondary} at the given original {date}. Body carries the
// full rate point; its time must match {date} — the date is the record's identity and an edit
// cannot change it (returns 400). Mirrors EditPrice.
func (h *Handler) EditFXRate(main, secondary, origDate string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if main == "" || secondary == "" || origDate == "" {
			http.Error(w, "main, secondary and date are required", http.StatusBadRequest)
			return
		}
		oldTime, err := time.Parse(timeLayout, origDate)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid date: %s", err.Error()), http.StatusBadRequest)
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
		pt, err := payload.toPoint()
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid time: %s", err.Error()), http.StatusBadRequest)
			return
		}
		if err := h.Store.EditRate(r.Context(), main, secondary, oldTime, pt); err != nil {
			if errors.Is(err, marketdata.ErrDateImmutable) {
				http.Error(w, "a rate record's date cannot be changed; delete it and create a new one", http.StatusBadRequest)
				return
			}
			http.Error(w, fmt.Sprintf("unable to update rate: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

// DeleteFXRate removes the rate record for {main}/{secondary} at the given {date}. Mirrors DeletePrice.
func (h *Handler) DeleteFXRate(main, secondary, date string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if main == "" || secondary == "" || date == "" {
			http.Error(w, "main, secondary and date are required", http.StatusBadRequest)
			return
		}
		t, err := time.Parse(timeLayout, date)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid date: %s", err.Error()), http.StatusBadRequest)
			return
		}
		if err := h.Store.DeleteRateAt(r.Context(), main, secondary, t); err != nil {
			if errors.Is(err, marketdata.ErrRecordNotFound) {
				http.Error(w, "no rate data found", http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to delete rate: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
