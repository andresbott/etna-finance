package handlrs

import (
	"encoding/json"
	"net/http"
)

type AppSettings struct {
	DateFormat            string   `json:"dateFormat"`
	MainCurrency          string   `json:"mainCurrency"`
	Currencies            []string `json:"currencies"`
	InvestmentInstruments bool     `json:"investmentInstruments"`
	Rsu                   bool     `json:"rsu"`
	FinancialSimulator    bool     `json:"financialSimulator"`
	MaxAttachmentSizeMB   float64  `json:"maxAttachmentSizeMB"`
	Version               string   `json:"version"`
	// AutoEnabled lists feature keys ("rsu", "investmentInstruments") that the server
	// turned on at startup despite the config disabling them, because the database
	// contained data requiring them. Effective state still lives in the booleans above;
	// this only conveys provenance so the UI can label them "Auto-enabled".
	AutoEnabled []string `json:"autoEnabled,omitempty"`
}

// SettingsResponse is the payload for GET /settings. It extends AppSettings with
// marketDataSymbols when provided by getSymbols.
type SettingsResponse struct {
	AppSettings
	MarketDataSymbols []string `json:"marketDataSymbols,omitempty"`
}

func SettingsHandler(settings AppSettings) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(settings); err != nil {
			http.Error(w, "failed to encode settings", http.StatusInternalServerError)
		}
	})
}

// SettingsHandlerWithMarketData returns a handler that encodes app settings plus
// the list of symbols with price data from getSymbols. If getSymbols is nil or
// returns an error, marketDataSymbols is omitted or empty.
func SettingsHandlerWithMarketData(settings AppSettings, getSymbols func() ([]string, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SettingsResponse{AppSettings: settings}
		if getSymbols != nil {
			if symbols, err := getSymbols(); err == nil {
				resp.MarketDataSymbols = symbols
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, "failed to encode settings", http.StatusInternalServerError)
		}
	})
}
