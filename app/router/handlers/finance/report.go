package finance

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type reportResponse struct {
	Incomes  []reportEntry `json:"income"`
	Expenses []reportEntry `json:"expenses"`
}
type reportEntry struct {
	Id          uint    `json:"id"`
	ParentId    uint    `json:"ParentId"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Value       float64 `json:"amount"`
}

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

		report, err := h.Store.GetReport(r.Context(), startDate, endDate, userId)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to list entries: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		response := reportResponse{
			Incomes:  make([]reportEntry, len(report.Income)),
			Expenses: make([]reportEntry, len(report.Expenses)),
		}

		for i, income := range report.Income {
			response.Incomes[i] = reportEntry{
				Id:          income.Id,
				ParentId:    income.ParentId,
				Name:        income.Name,
				Description: income.Description,
				Value:       income.Value,
			}
		}

		for i, expense := range report.Expenses {
			response.Incomes[i] = reportEntry{
				Id:          expense.Id,
				ParentId:    expense.ParentId,
				Name:        expense.Name,
				Description: expense.Description,
				Value:       expense.Value,
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
