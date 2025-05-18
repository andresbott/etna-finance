package finance

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/andresbott/etna/internal/model/finance"
)

type CategoryHandler struct {
	Store *finance.Store
}

// Category types
const (
	IncomeCategoryType  = "income"
	ExpenseCategoryType = "expense"
)

// Using validationErr from account.go

// Common category payloads
type categoryPayload struct {
	Id          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ParentId    *uint  `json:"parentId,omitempty"`
}

type categoryUpdatePayload struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type categoryMovePayload struct {
	TargetParentId uint `json:"targetParentId"`
}

func (h *CategoryHandler) CreateCategory(userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to create category: user not provided", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		// Get category type from query parameter
		categoryType := r.URL.Query().Get("type")
		if categoryType != IncomeCategoryType && categoryType != ExpenseCategoryType {
			http.Error(w, fmt.Sprintf("invalid category type: %s", categoryType), http.StatusBadRequest)
			return
		}

		payload := categoryPayload{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		var parentId uint
		if payload.ParentId != nil {
			parentId = *payload.ParentId
		}

		var respJson []byte
		if categoryType == IncomeCategoryType {
			category := &finance.IncomeCategory{
				Name: payload.Name,
			}

			err = h.Store.CreateIncomeCategory(category, parentId, userId)
			if err != nil {
				if errors.As(err, &validationErr) {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				} else {
					http.Error(w, fmt.Sprintf("unable to store income category in DB: %s", err.Error()), http.StatusInternalServerError)
					return
				}
			}

			respJson, err = json.Marshal(category)
		} else {
			category := &finance.ExpenseCategory{
				Name: payload.Name,
			}

			err = h.Store.CreateExpenseCategory(category, parentId, userId)
			if err != nil {
				if errors.As(err, &validationErr) {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				} else {
					http.Error(w, fmt.Sprintf("unable to store expense category in DB: %s", err.Error()), http.StatusInternalServerError)
					return
				}
			}

			respJson, err = json.Marshal(category)
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJson)
	})
}

func (h *CategoryHandler) UpdateCategory(Id uint, userId string) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to update category: user not provided", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		// Get category type from query parameter
		categoryType := r.URL.Query().Get("type")
		if categoryType != IncomeCategoryType && categoryType != ExpenseCategoryType {
			http.Error(w, fmt.Sprintf("invalid category type: %s", categoryType), http.StatusBadRequest)
			return
		}

		payload := categoryUpdatePayload{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		if categoryType == IncomeCategoryType {
			category := finance.IncomeCategory{
				Name: payload.Name,
			}
			err = h.Store.UpdateIncomeCategory(Id, category, userId)
		} else {
			category := finance.ExpenseCategory{
				Name: payload.Name,
			}
			err = h.Store.UpdateExpenseCategory(Id, category, userId)
		}

		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			} else if errors.Is(err, finance.ErrCategoryNotFound) {
				http.Error(w, fmt.Sprintf("unable to update category in DB: %s", err.Error()), http.StatusNotFound)
				return
			} else {
				http.Error(w, fmt.Sprintf("unable to update category in DB: %s", err.Error()), http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	})
}

func (h *CategoryHandler) MoveCategory(Id uint, userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to move category: user not provided", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		// Get category type from query parameter
		categoryType := r.URL.Query().Get("type")
		if categoryType != IncomeCategoryType && categoryType != ExpenseCategoryType {
			http.Error(w, fmt.Sprintf("invalid category type: %s", categoryType), http.StatusBadRequest)
			return
		}

		payload := categoryMovePayload{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		if categoryType == IncomeCategoryType {
			err = h.Store.MoveIncomeCategory(Id, payload.TargetParentId, userId)
		} else {
			err = h.Store.MoveExpenseCategory(Id, payload.TargetParentId, userId)
		}

		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			} else if errors.Is(err, finance.ErrCategoryNotFound) {
				http.Error(w, fmt.Sprintf("unable to move category in DB: %s", err.Error()), http.StatusNotFound)
				return
			} else {
				http.Error(w, fmt.Sprintf("unable to move category in DB: %s", err.Error()), http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	})
}

func (h *CategoryHandler) DeleteRecurseCategory(Id uint, userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to delete category: user not provided", http.StatusBadRequest)
			return
		}

		// Get category type from query parameter
		categoryType := r.URL.Query().Get("type")
		if categoryType != IncomeCategoryType && categoryType != ExpenseCategoryType {
			http.Error(w, fmt.Sprintf("invalid category type: %s", categoryType), http.StatusBadRequest)
			return
		}

		var err error
		if categoryType == IncomeCategoryType {
			err = h.Store.DeleteRecurseIncomeCategory(Id, userId)
		} else {
			err = h.Store.DeleteRecurseExpenseCategory(Id, userId)
		}

		if err != nil {
			if errors.Is(err, finance.ErrCategoryNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			} else if errors.Is(err, finance.ErrCategoryConstraintViolation) {
				http.Error(w, fmt.Sprintf("unable to delete category: %s", err.Error()), http.StatusConflict)
				return
			} else {
				http.Error(w, fmt.Sprintf("unable to delete category: %s", err.Error()), http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	})
}
