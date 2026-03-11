package csvimport

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/andresbott/etna/internal/csvimport"
)

type CategoryRuleGroupHandler struct {
	Store *csvimport.Store
}

type ruleGroupPayload struct {
	ID         uint                 `json:"id"`
	Name       string               `json:"name"`
	CategoryID uint                 `json:"categoryId"`
	Priority   int                  `json:"priority"`
	Patterns   []rulePatternPayload `json:"patterns"`
}

type rulePatternPayload struct {
	ID      uint   `json:"id"`
	Pattern string `json:"pattern"`
	IsRegex bool   `json:"isRegex"`
}

func (h *CategoryRuleGroupHandler) ListCategoryRuleGroups() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		groups, err := h.Store.ListCategoryRuleGroups(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to list category rule groups: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		items := make([]ruleGroupPayload, len(groups))
		for i, g := range groups {
			patterns := make([]rulePatternPayload, len(g.Patterns))
			for j, p := range g.Patterns {
				patterns[j] = rulePatternPayload{
					ID:      p.ID,
					Pattern: p.Pattern,
					IsRegex: p.IsRegex,
				}
			}
			items[i] = ruleGroupPayload{
				ID:         g.ID,
				Name:       g.Name,
				CategoryID: g.CategoryID,
				Priority:   g.Priority,
				Patterns:   patterns,
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

func (h *CategoryRuleGroupHandler) CreateCategoryRuleGroup() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		var payload ruleGroupPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		group := csvimport.CategoryRuleGroup{
			Name:       payload.Name,
			CategoryID: payload.CategoryID,
			Priority:   payload.Priority,
		}
		for _, p := range payload.Patterns {
			group.Patterns = append(group.Patterns, csvimport.CategoryRulePattern{
				Pattern: p.Pattern,
				IsRegex: p.IsRegex,
			})
		}

		id, err := h.Store.CreateCategoryRuleGroup(r.Context(), group)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, fmt.Sprintf("unable to create category rule group: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		// Re-fetch to get generated pattern IDs
		created, err := h.Store.GetCategoryRuleGroup(r.Context(), id)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to fetch created group: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		respPayload := ruleGroupPayload{
			ID:         created.ID,
			Name:       created.Name,
			CategoryID: created.CategoryID,
			Priority:   created.Priority,
		}
		for _, p := range created.Patterns {
			respPayload.Patterns = append(respPayload.Patterns, rulePatternPayload{
				ID:      p.ID,
				Pattern: p.Pattern,
				IsRegex: p.IsRegex,
			})
		}

		respJSON, err := json.Marshal(respPayload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}

func (h *CategoryRuleGroupHandler) UpdateCategoryRuleGroup(id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		var payload ruleGroupPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		group := csvimport.CategoryRuleGroup{
			Name:       payload.Name,
			CategoryID: payload.CategoryID,
			Priority:   payload.Priority,
		}

		err := h.Store.UpdateCategoryRuleGroup(r.Context(), id, group)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if errors.Is(err, csvimport.ErrCategoryRuleGroupNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to update category rule group: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func (h *CategoryRuleGroupHandler) DeleteCategoryRuleGroup(id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := h.Store.DeleteCategoryRuleGroup(r.Context(), id)
		if err != nil {
			if errors.Is(err, csvimport.ErrCategoryRuleGroupNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to delete category rule group: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func (h *CategoryRuleGroupHandler) CreateCategoryRulePattern(groupID uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		var payload rulePatternPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		pattern := csvimport.CategoryRulePattern{
			Pattern: payload.Pattern,
			IsRegex: payload.IsRegex,
		}

		id, err := h.Store.CreateCategoryRulePattern(r.Context(), groupID, pattern)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if errors.Is(err, csvimport.ErrCategoryRuleGroupNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to create pattern: %s", err.Error()), http.StatusInternalServerError)
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

func (h *CategoryRuleGroupHandler) UpdateCategoryRulePattern(groupID, patternID uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		var payload rulePatternPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		pattern := csvimport.CategoryRulePattern{
			Pattern: payload.Pattern,
			IsRegex: payload.IsRegex,
		}

		err := h.Store.UpdateCategoryRulePattern(r.Context(), groupID, patternID, pattern)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if errors.Is(err, csvimport.ErrCategoryRulePatternNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to update pattern: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func (h *CategoryRuleGroupHandler) DeleteCategoryRulePattern(groupID, patternID uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := h.Store.DeleteCategoryRulePattern(r.Context(), groupID, patternID)
		if err != nil {
			if errors.Is(err, csvimport.ErrCategoryRulePatternNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to delete pattern: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
