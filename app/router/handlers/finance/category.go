package finance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/andresbott/etna/internal/accounting"
)

type CategoryHandler struct {
	Store *accounting.Store
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
	Icon        string `json:"icon"`
}

type categoryUpdatePayload struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	ParentId    *uint  `json:"parentId,omitempty"`
}

type categoryMovePayload struct {
	TargetParentId uint `json:"targetParentId"`
}

func (h *CategoryHandler) CreateIncome() http.Handler {
	return h.createCategory(IncomeCategoryType)
}

func (h *CategoryHandler) CreateExpense() http.Handler {
	return h.createCategory(ExpenseCategoryType)
}

func (h *CategoryHandler) createCategory(categoryType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		data := accounting.CategoryData{
			Name:        payload.Name,
			Description: payload.Description,
			Icon:        payload.Icon,
		}
		switch categoryType {
		case IncomeCategoryType:
			data.Type = accounting.IncomeCategory
		case ExpenseCategoryType:
			data.Type = accounting.ExpenseCategory
		}

		catId, err = h.Store.CreateCategory(r.Context(), data, payload.ParentId)
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
			Icon:        data.Icon,
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

func (h *CategoryHandler) UpdateIncome(Id uint) http.Handler {
	return h.updateCategory(Id, IncomeCategoryType)
}

func (h *CategoryHandler) UpdateExpense(Id uint) http.Handler {
	return h.updateCategory(Id, ExpenseCategoryType)
}

func (h *CategoryHandler) updateCategory(Id uint, categoryType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		err = updateCategory(r.Context(), Id, categoryType, payload, h.Store)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			} else if errors.Is(err, accounting.ErrCategoryNotFound) {
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
func updateCategory(ctx context.Context, Id uint, categoryType string, payload categoryUpdatePayload, store *accounting.Store) error {
	var err error

	data := accounting.CategoryData{
		Name:        payload.Name,
		Description: payload.Description,
		Icon:        payload.Icon,
	}

	switch categoryType {
	case IncomeCategoryType:
		data.Type = accounting.IncomeCategory
	case ExpenseCategoryType:
		data.Type = accounting.ExpenseCategory
	}

	err = store.UpdateCategory(ctx, Id, data)
	if err != nil {
		return err
	}

	if payload.ParentId != nil {
		err = store.MoveCategory(ctx, Id, *payload.ParentId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *CategoryHandler) MoveIncome(Id uint) http.Handler {
	return h.moveCategory(Id, IncomeCategoryType)
}

func (h *CategoryHandler) MoveExpense(Id uint) http.Handler {
	return h.moveCategory(Id, ExpenseCategoryType)
}

func (h *CategoryHandler) moveCategory(Id uint, categoryType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		err = h.Store.MoveCategory(r.Context(), Id, payload.TargetParentId)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			} else if errors.Is(err, accounting.ErrCategoryNotFound) {
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

func (h *CategoryHandler) DeleteIncome(Id uint) http.Handler {
	return h.deleteRecurseCategory(Id, IncomeCategoryType)
}

func (h *CategoryHandler) DeleteExpense(Id uint) http.Handler {
	return h.deleteRecurseCategory(Id, ExpenseCategoryType)
}

func (h *CategoryHandler) deleteRecurseCategory(Id uint, categoryType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if categoryType != IncomeCategoryType && categoryType != ExpenseCategoryType {
			http.Error(w, fmt.Sprintf("invalid category type: %s", categoryType), http.StatusBadRequest)
			return
		}

		err := h.Store.DeleteCategoryRecursive(r.Context(), Id)
		if err != nil {
			if errors.Is(err, accounting.ErrCategoryNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			} else if errors.Is(err, accounting.ErrCategoryConstraintViolation) {
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

func (h *CategoryHandler) ListIncome(Id uint) http.Handler {
	return h.listCategory(Id, IncomeCategoryType)
}

func (h *CategoryHandler) ListExpense(Id uint) http.Handler {
	return h.listCategory(Id, ExpenseCategoryType)
}

// listCategory lists all categories of type categoryType for the given parent id
func (h *CategoryHandler) listCategory(Id uint, categoryType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

			items, err := h.Store.ListDescendantCategories(r.Context(), Id, depth, accounting.IncomeCategory)
			if err != nil {
				break
			}
			outItems = make([]categoryPayload, len(items))
			for i, item := range items {
				outItems[i] = categoryPayload{
					Id:          item.Id,
					Name:        item.Name,
					Description: item.Description,
					Icon:        item.Icon,
					ParentId:    item.ParentId,
				}
			}

		case ExpenseCategoryType:

			items, err := h.Store.ListDescendantCategories(r.Context(), Id, depth, accounting.ExpenseCategory)
			if err != nil {
				break
			}
			outItems = make([]categoryPayload, len(items))
			for i, item := range items {
				outItems[i] = categoryPayload{
					Id:          item.Id,
					Name:        item.Name,
					Description: item.Description,
					Icon:        item.Icon,
					ParentId:    item.ParentId,
				}
			}

		default:
			http.Error(w, fmt.Sprintf("invalid category type: %s", categoryType), http.StatusBadRequest)
			return
		}

		if err != nil {
			if errors.Is(err, accounting.ErrCategoryNotFound) {
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
