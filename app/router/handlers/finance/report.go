package finance

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/andresbott/etna/internal/accounting"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type incomeExpenseResponse struct {
	Incomes  []incomeExpenseEntry `json:"income"`
	Expenses []incomeExpenseEntry `json:"expenses"`
}
type incomeExpenseEntry struct {
	Id          uint                           `json:"id"`
	ParentId    uint                           `json:"ParentId"`
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	Values      map[string]incomeExpenseValues `json:"values"`
}
type incomeExpenseValues struct {
	Value float64 `json:"amount"`
	Count uint    `json:"count"`
}

func (h *Handler) IncomeExpenseReport(userId string) http.Handler {
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

		report, err := h.Store.ReportInOutByCategory(r.Context(), startDate, endDate, userId)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to list entries: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		response := incomeExpenseResponse{
			Incomes:  make([]incomeExpenseEntry, len(report.Income)),
			Expenses: make([]incomeExpenseEntry, len(report.Expenses)),
		}

		for i, income := range report.Income {
			values := make(map[string]incomeExpenseValues)
			for k, v := range income.Values {
				values[k.String()] = incomeExpenseValues{
					Value: v.Value,
					Count: v.Count,
				}
			}
			response.Incomes[i] = incomeExpenseEntry{
				Id:          income.Id,
				ParentId:    income.ParentId,
				Name:        income.Name,
				Description: income.Description,
				Values:      values,
			}
		}

		for i, expense := range report.Expenses {
			values := make(map[string]incomeExpenseValues)
			for k, v := range expense.Values {
				values[k.String()] = incomeExpenseValues{
					Value: v.Value,
					Count: v.Count,
				}
			}
			response.Expenses[i] = incomeExpenseEntry{
				Id:          expense.Id,
				ParentId:    expense.ParentId,
				Name:        expense.Name,
				Description: expense.Description,
				Values:      values,
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

type accountBalancesResponse struct {
	Accounts map[uint]accountBalance `json:"accounts"`
}

type accountBalance struct {
	Date  time.Time
	Sum   float64
	Count uint
}

func (h *Handler) AccountBalance(userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to list entries: user not provided", http.StatusBadRequest)
			return
		}

		accountIds := r.URL.Query().Get("accountIds")
		ids := strings.Split(accountIds, ",")

		response := accountBalancesResponse{
			Accounts: map[uint]accountBalance{},
		}
		for _, accountId := range ids {
			id, err := strconv.ParseUint(accountId, 10, 64)
			if err != nil {
				http.Error(w, fmt.Sprintf("unable to parse account IDs: %s", err.Error()), http.StatusBadRequest)
				return
			}
			data, err := h.Store.AccountBalance(r.Context(), uint(id), time.Now(), userId)
			if err != nil {
				if errors.Is(err, accounting.ErrAccountNotFound) {
					http.Error(w, fmt.Sprintf("account id not found: %d", id), http.StatusBadRequest)
					return
				} else {
					http.Error(w, fmt.Sprintf("unable to get account balance: %s", err.Error()), http.StatusInternalServerError)
					return
				}
			}
			response.Accounts[uint(id)] = accountBalance{
				Date:  data.Date,
				Sum:   data.Sum,
				Count: data.Count,
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
