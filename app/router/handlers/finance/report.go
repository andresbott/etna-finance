package finance

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

func (h *Handler) GetReport(userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to list entries: user not provided", http.StatusBadRequest)
			return
		}

		startDate, endDate, err := getDateRange(r.URL.Query().Get("startDate"), r.URL.Query().Get("endDate"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		entries, err := h.Store.GetReport(r.Context(), startDate, endDate, userId)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to list entries: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		response := listEntriesResponse{
			Items: make([]entryPayload, len(entries)),
		}

		for i, entry := range entries {
			response.Items[i] = entryPayload{
				Id:          entry.Id,
				Description: entry.Description,
				Date:        entry.Date,
				Type:        entry.Type.String(),

				StockAmount: entry.StockAmount,

				TargetAmount:          entry.TargetAmount,
				TargetAccountID:       entry.TargetAccountID,
				TargetAccountName:     entry.TargetAccountName,
				TargetAccountCurrency: entry.TargetAccountCurrency.String(),

				OriginAmount:          entry.OriginAmount,
				OriginAccountID:       entry.OriginAccountID,
				OriginAccountName:     entry.OriginAccountName,
				OriginAccountCurrency: entry.OriginAccountCurrency.String(),

				CategoryId: entry.CategoryId,
			}
		}

		respJson, err := json.Marshal(response)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJson)
	})
}

func getDateRange(startDateStr, endDateStr string) (time.Time, time.Time, error) {

	// Set default date range to last 30 days if not provided
	now := time.Now()
	endDate := now
	startDate := now.AddDate(0, 0, -30) // 30 days ago

	// Parse dates if provided
	if startDateStr != "" {
		var err error
		startDate, err = time.Parse(time.DateOnly, startDateStr)
		if err != nil {
			return startDate, endDate, errors.New("invalid startDate format")
		}
	}
	// set the start time to midnight
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())

	if endDateStr != "" {
		var err error
		endDate, err = time.Parse(time.DateOnly, endDateStr)
		if err != nil {
			return startDate, endDate, errors.New("invalid endDate format")
		}
	}
	// set the endDate time to midnight of the next day
	endDate = endDate.AddDate(0, 0, 1)
	endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location())

	return startDate, endDate, nil

}
