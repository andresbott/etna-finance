package finance

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/andresbott/etna/internal/model/finance"
)

type entryPayload struct {
	Id              uint      `json:"id"`
	Description     string    `json:"description"`
	Amount          float64   `json:"amount"`
	StockAmount     float64   `json:"StockAmount"`
	Date            time.Time `json:"date"`
	Type            string    `json:"type"`
	TargetAccountID uint      `json:"targetAccountId"`
	OriginAccountID uint      `json:"originAccountId"`
	CategoryId      uint      `json:"CategoryId"`
}

func (h *Handler) CreateEntry(userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to create entry: user not provided", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		payload := entryPayload{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		entry := finance.Entry{
			Description:     payload.Description,
			Amount:          payload.Amount,
			StockAmount:     payload.StockAmount,
			Date:            payload.Date,
			TargetAccountID: payload.TargetAccountID,
			OriginAccountID: payload.OriginAccountID,
			CategoryId:      payload.CategoryId,
		}

		t, err := finance.ParseEntryType(payload.Type)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to parse entry type: %s", err.Error()), http.StatusBadRequest)
			return
		}
		entry.Type = t

		entryID, err := h.Store.CreateEntry(r.Context(), entry, userId)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				http.Error(w, fmt.Sprintf("unable to store entry in DB: %s", err.Error()), http.StatusInternalServerError)
				return
			}
		}

		entry.Id = entryID
		respJson, err := json.Marshal(entry)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJson)
	})
}

type entryUpdatePayload struct {
	Description     *string    `json:"description"`
	Amount          *float64   `json:"amount"`
	Date            *time.Time `json:"date"`
	TargetAccountID *uint      `json:"target_account_id"`
	OriginAccountID *uint      `json:"origin_account_id"`
	CategoryId      *uint      `json:"category_id"`
}

func (h *Handler) UpdateEntry(Id uint, userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to update entry: user not provided", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		payload := entryUpdatePayload{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		updatePayload := finance.EntryUpdatePayload{
			Description:     payload.Description,
			Amount:          payload.Amount,
			Date:            payload.Date,
			TargetAccountID: payload.TargetAccountID,
			OriginAccountID: payload.OriginAccountID,
			CategoryId:      payload.CategoryId,
		}

		err = h.Store.UpdateEntry(updatePayload, Id, userId)
		if err != nil {
			if errors.Is(err, finance.EntryNotFoundErr) {
				http.Error(w, "entry not found", http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("unable to update entry: %s", err.Error()), http.StatusInternalServerError)
			}
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}

func (h *Handler) DeleteEntry(Id uint, userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to delete entry: user not provided", http.StatusBadRequest)
			return
		}

		err := h.Store.DeleteEntry(r.Context(), Id, userId)
		if err != nil {
			if errors.Is(err, finance.EntryNotFoundErr) {
				http.Error(w, "entry not found", http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("unable to delete entry: %s", err.Error()), http.StatusInternalServerError)
			}
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}

type listEntriesResponse struct {
	Items []entryPayload `json:"items"`
}

func (h *Handler) ListEntries(userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to list entries: user not provided", http.StatusBadRequest)
			return
		}

		// Parse query parameters

		// Set default date range to last 30 days if not provided
		now := time.Now()
		endDate := now
		startDate := now.AddDate(0, 0, -30) // 30 days ago

		// Parse dates if provided
		startDateStr := r.URL.Query().Get("startDate")
		if startDateStr != "" {
			var err error
			startDate, err = time.Parse(time.DateOnly, startDateStr)
			if err != nil {
				http.Error(w, "invalid startDate format", http.StatusBadRequest)
				return
			}
		}
		// set the start time to midnight
		startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())

		endDateStr := r.URL.Query().Get("endDate")
		if endDateStr != "" {
			var err error
			endDate, err = time.Parse(time.DateOnly, endDateStr)
			if err != nil {
				http.Error(w, "invalid endDate format", http.StatusBadRequest)
				return
			}
		}
		// set the endDate time to midnight of the next day
		endDate = endDate.AddDate(0, 0, 1)
		endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location())

		// Parse account ID if provided
		var accountID *uint
		accountIDStr := r.URL.Query().Get("accountId")
		if accountIDStr != "" {
			var id uint
			if _, err := fmt.Sscanf(accountIDStr, "%d", &id); err != nil {
				http.Error(w, "invalid account_id format", http.StatusBadRequest)
				return
			}
			accountID = &id
		}

		// Parse pagination parameters
		limitStr := r.URL.Query().Get("limit")
		limit := 30 // default
		if limitStr != "" {
			if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil {
				http.Error(w, "invalid limit format", http.StatusBadRequest)
				return
			}
		}

		page := 1 // default
		pageStr := r.URL.Query().Get("page")
		if pageStr != "" {
			if _, err := fmt.Sscanf(pageStr, "%d", &page); err != nil {
				http.Error(w, "invalid page format", http.StatusBadRequest)
				return
			}
		}

		entries, err := h.Store.ListEntries(r.Context(), startDate, endDate, accountID, limit, page, userId)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to list entries: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		response := listEntriesResponse{
			Items: make([]entryPayload, len(entries)),
		}

		for i, entry := range entries {
			response.Items[i] = entryPayload{
				Id:              entry.Id,
				Description:     entry.Description,
				Amount:          entry.Amount,
				StockAmount:     entry.StockAmount,
				Date:            entry.Date,
				Type:            entry.Type.String(),
				TargetAccountID: entry.TargetAccountID,
				OriginAccountID: entry.OriginAccountID,
				CategoryId:      entry.CategoryId,
			}
		}

		if len(response.Items) == 0 {
			response.Items = []entryPayload{
				{
					Id:              1,
					Description:     "income 1",
					Amount:          1000.0,
					Date:            time.Now(),
					Type:            "income",
					TargetAccountID: 1,
					CategoryId:      1,
				},
				{
					Id:              2,
					Description:     "expense 2",
					Amount:          25.5,
					StockAmount:     0,
					Date:            time.Time{},
					Type:            "expense",
					TargetAccountID: 0,
					OriginAccountID: 0,
					CategoryId:      0,
				},
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

func (h *Handler) LockEntries(userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to lock entries: user not provided", http.StatusBadRequest)
			return
		}

		dateStr := r.URL.Query().Get("date")
		if dateStr == "" {
			http.Error(w, "date parameter is required", http.StatusBadRequest)
			return
		}

		date, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			http.Error(w, "invalid date format", http.StatusBadRequest)
			return
		}

		err = h.Store.LockEntries(r.Context(), date)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to lock entries: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}
