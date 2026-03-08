package csvimport

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/andresbott/etna/internal/csvimport"
)

type CategoryRuleHandler struct {
	Store *csvimport.Store
}

type categoryRulePayload struct {
	ID         uint   `json:"id"`
	Pattern    string `json:"pattern"`
	IsRegex    bool   `json:"isRegex"`
	CategoryID uint   `json:"categoryId"`
	Position   int    `json:"position"`
}

func (h *CategoryRuleHandler) ListCategoryRules() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rules, err := h.Store.ListCategoryRules(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to list category rules: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		items := make([]categoryRulePayload, len(rules))
		for i, rule := range rules {
			items[i] = categoryRulePayload{
				ID:         rule.ID,
				Pattern:    rule.Pattern,
				IsRegex:    rule.IsRegex,
				CategoryID: rule.CategoryID,
				Position:   rule.Position,
			}
		}

		respJSON, err := json.Marshal(items)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}

func (h *CategoryRuleHandler) CreateCategoryRule() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		var payload categoryRulePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		rule := csvimport.CategoryRule{
			Pattern:    payload.Pattern,
			IsRegex:    payload.IsRegex,
			CategoryID: payload.CategoryID,
			Position:   payload.Position,
		}

		id, err := h.Store.CreateCategoryRule(r.Context(), rule)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, fmt.Sprintf("unable to create category rule: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		payload.ID = id
		respJSON, err := json.Marshal(payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}

func (h *CategoryRuleHandler) UpdateCategoryRule(id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		var payload categoryRulePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		rule := csvimport.CategoryRule{
			Pattern:    payload.Pattern,
			IsRegex:    payload.IsRegex,
			CategoryID: payload.CategoryID,
			Position:   payload.Position,
		}

		err := h.Store.UpdateCategoryRule(r.Context(), id, rule)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if errors.Is(err, csvimport.ErrCategoryRuleNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to update category rule: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func (h *CategoryRuleHandler) DeleteCategoryRule(id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := h.Store.DeleteCategoryRule(r.Context(), id)
		if err != nil {
			if errors.Is(err, csvimport.ErrCategoryRuleNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to delete category rule: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
