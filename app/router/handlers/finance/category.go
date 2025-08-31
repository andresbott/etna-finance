package finance

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/andresbott/etna/internal/model/finance"
	"net/http"
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
	ParentId    uint   `json:"parentId,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type categoryUpdatePayload struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type categoryMovePayload struct {
	TargetParentId uint `json:"targetParentId"`
}

func (h *CategoryHandler) CreateIncome(userId string) http.Handler {
	return h.createCategory(userId, IncomeCategoryType)
}

func (h *CategoryHandler) CreateExpense(userId string) http.Handler {
	return h.createCategory(userId, ExpenseCategoryType)
}

func (h *CategoryHandler) createCategory(userId, categoryType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to create category: user not provided", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		if categoryType != IncomeCategoryType && categoryType != ExpenseCategoryType {
			http.Error(w, fmt.Sprintf("invalid category type: %s", categoryType), http.StatusBadRequest)
			return
		}

		var (
			respJson []byte
			err      error
		)

		payload := categoryPayload{}
		err = json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		if payload.Name == "" {
			http.Error(w, fmt.Sprintf("wrong payload: Name is empty"), http.StatusBadRequest)
			return
		}

		switch categoryType {
		case IncomeCategoryType:
			category := &finance.IncomeCategory{
				Name:        payload.Name,
				Description: payload.Description,
			}
			err = h.Store.CreateIncomeCategory(r.Context(), category, payload.ParentId, userId)
			if err != nil {
				break
			}
			respJson, err = json.Marshal(category)

		case ExpenseCategoryType:
			category := &finance.ExpenseCategory{
				Name:        payload.Name,
				Description: payload.Description,
			}
			err = h.Store.CreateExpenseCategory(r.Context(), category, payload.ParentId, userId)
			if err != nil {
				break
			}
			respJson, err = json.Marshal(category)

		default:
			http.Error(w, fmt.Sprintf("invalid category type: %s", categoryType), http.StatusBadRequest)
			return
		}

		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			} else {
				http.Error(w, fmt.Sprintf("unable to store category in DB: %s", err.Error()), http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJson)
	})
}

func (h *CategoryHandler) UpdateIncome(Id uint, userId string) http.Handler {
	return h.updateCategory(Id, userId, IncomeCategoryType)
}

func (h *CategoryHandler) UpdateExpense(Id uint, userId string) http.Handler {
	return h.updateCategory(Id, userId, ExpenseCategoryType)
}

func (h *CategoryHandler) updateCategory(Id uint, userId, categoryType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to update category: user not provided", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

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
				Name:        payload.Name,
				Description: payload.Description,
			}
			err = h.Store.UpdateIncomeCategory(r.Context(), Id, category, userId)
		} else {
			category := finance.ExpenseCategory{
				Name:        payload.Name,
				Description: payload.Description,
			}
			err = h.Store.UpdateExpenseCategory(r.Context(), Id, category, userId)
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

func (h *CategoryHandler) MoveIncome(Id uint, userId string) http.Handler {
	return h.moveCategory(Id, userId, IncomeCategoryType)
}

func (h *CategoryHandler) MoveExpense(Id uint, userId string) http.Handler {
	return h.moveCategory(Id, userId, ExpenseCategoryType)
}

func (h *CategoryHandler) moveCategory(Id uint, userId, categoryType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to move category: user not provided", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

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
			err = h.Store.MoveIncomeCategory(r.Context(), Id, payload.TargetParentId, userId)
		} else {
			err = h.Store.MoveExpenseCategory(r.Context(), Id, payload.TargetParentId, userId)
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

func (h *CategoryHandler) DeleteIncome(Id uint, userId string) http.Handler {
	return h.deleteRecurseCategory(Id, userId, IncomeCategoryType)
}

func (h *CategoryHandler) DeleteExpense(Id uint, userId string) http.Handler {
	return h.deleteRecurseCategory(Id, userId, ExpenseCategoryType)
}

func (h *CategoryHandler) deleteRecurseCategory(Id uint, userId, categoryType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to delete category: user not provided", http.StatusBadRequest)
			return
		}

		if categoryType != IncomeCategoryType && categoryType != ExpenseCategoryType {
			http.Error(w, fmt.Sprintf("invalid category type: %s", categoryType), http.StatusBadRequest)
			return
		}

		var err error
		if categoryType == IncomeCategoryType {
			err = h.Store.DeleteRecurseIncomeCategory(r.Context(), Id, userId)
		} else {
			err = h.Store.DeleteRecurseExpenseCategory(r.Context(), Id, userId)
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

func (h *CategoryHandler) ListIncome(Id uint, userId string) http.Handler {
	return h.listCategory(Id, userId, IncomeCategoryType)
}

func (h *CategoryHandler) ListExpense(Id uint, userId string) http.Handler {
	return h.listCategory(Id, userId, ExpenseCategoryType)
}

// listCategory lists all categories of type categoryType belonging to user userId and parent id
func (h *CategoryHandler) listCategory(Id uint, userId, categoryType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to delete category: user not provided", http.StatusBadRequest)
			return
		}

		depth := 4 // TODO expose as parameter

		type payload struct {
			Items []categoryPayload `json:"items"`
		}

		var (
			outItems []categoryPayload
			err      error
		)

		switch categoryType {
		case IncomeCategoryType:
			var items []finance.IncomeCategory
			err = h.Store.DescendantsIncomeCategory(r.Context(), Id, depth, userId, &items)
			if err != nil {
				break
			}
			outItems = make([]categoryPayload, len(items))
			for i, item := range items {
				outItems[i] = categoryPayload{
					Id:          item.Id(),
					Name:        item.Name,
					Description: item.Description,
					ParentId:    item.Parent(),
				}
			}

		case ExpenseCategoryType:
			var items []finance.ExpenseCategory
			err = h.Store.DescendantsExpenseCategory(r.Context(), Id, depth, userId, &items)
			if err != nil {
				break
			}
			outItems = make([]categoryPayload, len(items))
			for i, item := range items {
				outItems[i] = categoryPayload{
					Id:          item.Id(),
					Name:        item.Name,
					Description: item.Description,
					ParentId:    item.Parent(),
				}
			}

		default:
			http.Error(w, fmt.Sprintf("invalid category type: %s", categoryType), http.StatusBadRequest)
			return
		}

		if err != nil {
			if errors.Is(err, finance.ErrCategoryNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("unable to delete category: %s", err.Error()), http.StatusInternalServerError)
			}
			return
		}

		if len(outItems) == 0 {
			outItems = append(outItems, categoryPayload{
				Id:          1,
				ParentId:    0,
				Name:        "First Parent",
				Description: "This is  a sample parent",
			})
			outItems = append(outItems, categoryPayload{
				Id:          2,
				ParentId:    1,
				Name:        "First Child",
				Description: "This is  a sample child",
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(payload{Items: outItems}); err != nil {
			http.Error(w, fmt.Sprintf("Error encoding JSON: %s", err.Error()), http.StatusInternalServerError)
		}

	})
}
