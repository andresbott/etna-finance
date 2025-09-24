package finance

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/andresbott/etna/internal/model/finance"
)

type entryPayload struct {
	Id          uint      `json:"id"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	Type        string    `json:"type"`

	StockAmount float64 `json:"StockAmount"`

	TargetAmount          float64 `json:"targetAmount"`
	TargetAccountID       uint    `json:"targetAccountId"`
	TargetAccountName     string  `json:"targetAccountName"`
	TargetAccountCurrency string  `json:"targetAccountCurrency"`

	OriginAmount          float64 `json:"originAmount"`
	OriginAccountID       uint    `json:"originAccountId"`
	OriginAccountName     string  `json:"originAccountName"`
	OriginAccountCurrency string  `json:"originAccountCurrency"`

	CategoryId uint `json:"CategoryId"`
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
			Description: payload.Description,
			Date:        payload.Date,

			StockAmount: payload.StockAmount,

			TargetAmount:    payload.TargetAmount,
			TargetAccountID: payload.TargetAccountID,

			OriginAmount:    payload.OriginAmount,
			OriginAccountID: payload.OriginAccountID,

			CategoryId: payload.CategoryId,
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
				return
			} else {
				http.Error(w, fmt.Sprintf("unable to Store entry in DB: %s", err.Error()), http.StatusInternalServerError)
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
	Description *string    `json:"description"`
	Date        *time.Time `json:"date"`

	StockAmount *float64 `json:"stockAmount"`

	TargetAmount    *float64 `json:"targetAmount"`
	TargetAccountID *uint    `json:"targetAccountId"`

	OriginAmount    *float64 `json:"originAmount"`
	OriginAccountID *uint    `json:"originIccountId"`

	CategoryId *uint `json:"categoryId"`
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
			Description: payload.Description,
			Date:        payload.Date,

			StockAmount: payload.StockAmount,

			TargetAmount:    payload.TargetAmount,
			TargetAccountID: payload.TargetAccountID,

			OriginAmount:    payload.OriginAmount,
			OriginAccountID: payload.OriginAccountID,

			CategoryId: payload.CategoryId,
		}

		err = h.Store.UpdateEntry(r.Context(), updatePayload, Id, userId)
		if err != nil {
			if errors.Is(err, finance.ErrEntryNotFound) {
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
			if errors.Is(err, finance.ErrEntryNotFound) {
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

		startDate, endDate, err := getDateRange(r.URL.Query().Get("startDate"), r.URL.Query().Get("endDate"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Parse account IDs if provided
		accountIds := []int{}
		ids := r.URL.Query()["accountIds"]
		for _, idStr := range ids {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				http.Error(w, "invalid accountId format", http.StatusBadRequest)
				return
			}
			accountIds = append(accountIds, id)
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

		entries, err := h.Store.ListEntries(r.Context(), finance.ListOpts{StartDate: startDate, EndDate: endDate,
			AccountIds: accountIds, Limit: limit, Page: page, Tenant: userId})
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
