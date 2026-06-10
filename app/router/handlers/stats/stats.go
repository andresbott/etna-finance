package stats

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/andresbott/etna/internal/marketdata"
	"gorm.io/gorm"
)

// Handler serves aggregate application statistics (storage volume).
type Handler struct {
	DB          *gorm.DB
	MarketStore *marketdata.Store
}

type statsResponse struct {
	DBSizeBytes int64 `json:"dbSizeBytes"`
	PriceSeries int   `json:"priceSeries"`
	PricePoints int   `json:"pricePoints"`
	FXSeries    int   `json:"fxSeries"`
	FXPoints    int   `json:"fxPoints"`
}

// Stats returns storage statistics: database size and market data / FX volume.
func (h *Handler) Stats() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ds, err := h.MarketStore.Stats(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to gather market data stats: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		size, err := dbSizeBytes(r.Context(), h.DB)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to measure database size: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		resp := statsResponse{
			DBSizeBytes: size,
			PriceSeries: ds.PriceSeries,
			PricePoints: ds.PricePoints,
			FXSeries:    ds.FXSeries,
			FXPoints:    ds.FXPoints,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, fmt.Sprintf("error encoding JSON: %s", err.Error()), http.StatusInternalServerError)
		}
	})
}

// dbSizeBytes returns the SQLite main database file size (page_count * page_size).
// This excludes the WAL file, which is checkpointed back into the main file.
func dbSizeBytes(ctx context.Context, db *gorm.DB) (int64, error) {
	var pageCount, pageSize int64
	if err := db.WithContext(ctx).Raw("PRAGMA page_count").Scan(&pageCount).Error; err != nil {
		return 0, fmt.Errorf("read page_count: %w", err)
	}
	if err := db.WithContext(ctx).Raw("PRAGMA page_size").Scan(&pageSize).Error; err != nil {
		return 0, fmt.Errorf("read page_size: %w", err)
	}
	return pageCount * pageSize, nil
}
