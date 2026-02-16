package handlrs

import (
	"encoding/json"
	"net/http"
)

type AppSettings struct {
	DateFormat   string   `json:"dateFormat"`
	MainCurrency string   `json:"mainCurrency"`
	Currencies   []string `json:"currencies"`
	Instruments  bool     `json:"instruments"`
}

func SettingsHandler(settings AppSettings) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(settings); err != nil {
			http.Error(w, "failed to encode settings", http.StatusInternalServerError)
		}
	})
}
