package finance

import (
	"context"
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
	ParentId    *uint  `json:"parentId,omitempty"`
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

		payload := categoryPayload{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		if payload.Name == "" {
			http.Error(w, "wrong payload: Name is empty", http.StatusBadRequest)
			return
		}

		var catId uint
		data := finance.CategoryData{
			Name:        payload.Name,
			Description: payload.Description,
		}
		switch categoryType {
		case IncomeCategoryType:
			data.Type = finance.IncomeCategory
		case ExpenseCategoryType:
			data.Type = finance.ExpenseCategory
		}

		catId, err = h.Store.CreateCategory(r.Context(), data, payload.ParentId, userId)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			} else {
				http.Error(w, fmt.Sprintf("unable to store data in DB: %s", err.Error()), http.StatusInternalServerError)
				return
			}
		}
		category := categoryPayload{
			Id:          catId,
			ParentId:    payload.ParentId,
			Name:        data.Name,
			Description: data.Description,
		}
		var respJson []byte
		respJson, err = json.Marshal(category)
		
		if err != nil {
			http.Error(w, fmt.Sprintf("response marshal error: %s", err.Error()), http.StatusInternalServerError)
			return
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

		err = updateCategory(r.Context(), Id, userId, categoryType, payload, h.Store)
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

// updateCategory is an internal function taking care of running the update operations and returning an error if anything goes wrong
func updateCategory(ctx context.Context, Id uint, userId, categoryType string, payload categoryUpdatePayload, store *finance.Store) error {
	var err error

	data := finance.CategoryData{
		Name:        payload.Name,
		Description: payload.Description,
	}

	switch categoryType {
	case IncomeCategoryType:
		data.Type = finance.IncomeCategory
	case ExpenseCategoryType:
		data.Type = finance.ExpenseCategory
	}

	err = store.UpdateCategory(ctx, Id, data, userId)
	if err != nil {
		return err
	}

	if payload.ParentId != nil {
		err = store.MoveCategory(ctx, Id, *payload.ParentId, data.Type, userId)
		if err != nil {
			return err
		}
	}
	return nil
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
			err = h.Store.MoveCategory(r.Context(), Id, payload.TargetParentId, finance.IncomeCategory, userId)
		} else {
			err = h.Store.MoveCategory(r.Context(), Id, payload.TargetParentId, finance.ExpenseCategory, userId)
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
			err = h.Store.DeleteCategoryRecursive(r.Context(), Id, finance.IncomeCategory, userId)
		} else {
			err = h.Store.DeleteCategoryRecursive(r.Context(), Id, finance.ExpenseCategory, userId)
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

			items, err := h.Store.ListDescendantCategories(r.Context(), Id, depth, finance.IncomeCategory, userId)
			if err != nil {
				break
			}
			outItems = make([]categoryPayload, len(items))
			for i, item := range items {
				outItems[i] = categoryPayload{
					Id:          item.Id,
					Name:        item.Name,
					Description: item.Description,
					ParentId:    item.ParentId,
				}
			}

		case ExpenseCategoryType:

			items, err := h.Store.ListDescendantCategories(r.Context(), Id, depth, finance.ExpenseCategory, userId)
			if err != nil {
				break
			}
			outItems = make([]categoryPayload, len(items))
			for i, item := range items {
				outItems[i] = categoryPayload{
					Id:          item.Id,
					Name:        item.Name,
					Description: item.Description,
					ParentId:    item.ParentId,
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

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(payload{Items: outItems}); err != nil {
			http.Error(w, fmt.Sprintf("Error encoding JSON: %s", err.Error()), http.StatusInternalServerError)
		}

	})
}
