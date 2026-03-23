package handlrs

import (
	"encoding/json"
	"net/http"
)

type AppSettings struct {
	DateFormat          string   `json:"dateFormat"`
	MainCurrency        string   `json:"mainCurrency"`
	Currencies          []string `json:"currencies"`
	Instruments         bool     `json:"instruments"`
	Rsu                 bool     `json:"rsu"`
	Tools               bool     `json:"tools"`
	MaxAttachmentSizeMB float64  `json:"maxAttachmentSizeMB"`
	Version             string   `json:"version"`
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
