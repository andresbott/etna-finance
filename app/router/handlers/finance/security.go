package finance

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/andresbott/etna/internal/accounting"
	"golang.org/x/text/currency"
)

type securityPayload struct {
	ID       uint   `json:"id"`
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
}

type securityCreatePayload struct {
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
}

type securityUpdatePayload struct {
	Symbol   *string `json:"symbol,omitempty"`
	Name     *string `json:"name,omitempty"`
	Currency *string `json:"currency,omitempty"`
}

func securityToPayload(s accounting.Security) securityPayload {
	return securityPayload{
		ID:       s.ID,
		Symbol:   s.Symbol,
		Name:     s.Name,
		Currency: s.Currency.String(),
	}
}

func (h *Handler) ListSecurities(userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to list securities: user not provided", http.StatusBadRequest)
			return
		}

		items, err := h.Store.ListSecurities(r.Context(), userId)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to list securities: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		out := make([]securityPayload, len(items))
		for i, item := range items {
			out[i] = securityToPayload(item)
		}

		type response struct {
			Items []securityPayload `json:"items"`
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response{Items: out}); err != nil {
			http.Error(w, fmt.Sprintf("error encoding JSON: %s", err.Error()), http.StatusInternalServerError)
		}
	})
}

func (h *Handler) CreateSecurity(userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to create security: user not provided", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		var payload securityCreatePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		curr, err := currency.ParseISO(payload.Currency)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid currency: %s", payload.Currency), http.StatusBadRequest)
			return
		}

		item := accounting.Security{
			Symbol:   payload.Symbol,
			Name:     payload.Name,
			Currency: curr,
		}

		id, err := h.Store.CreateSecurity(r.Context(), item, userId)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, fmt.Sprintf("unable to create security: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		out := securityPayload{
			ID:       id,
			Symbol:   item.Symbol,
			Name:     item.Name,
			Currency: item.Currency.String(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(out)
	})
}

func (h *Handler) GetSecurity(id uint, userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to get security: user not provided", http.StatusBadRequest)
			return
		}

		item, err := h.Store.GetSecurity(r.Context(), id, userId)
		if err != nil {
			if errors.Is(err, accounting.ErrSecurityNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to get security: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(securityToPayload(item))
	})
}

func (h *Handler) UpdateSecurity(id uint, userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to update security: user not provided", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		var payload securityUpdatePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		item := accounting.SecurityUpdatePayload{
			Symbol:   payload.Symbol,
			Name:     payload.Name,
			Currency: payload.Currency,
		}

		err := h.Store.UpdateSecurity(r.Context(), id, userId, item)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if errors.Is(err, accounting.ErrNoChanges) {
				http.Error(w, "no changes applied", http.StatusBadRequest)
				return
			}
			if errors.Is(err, accounting.ErrSecurityNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to update security: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func (h *Handler) DeleteSecurity(id uint, userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to delete security: user not provided", http.StatusBadRequest)
			return
		}

		err := h.Store.DeleteSecurity(r.Context(), id, userId)
		if err != nil {
			if errors.Is(err, accounting.ErrSecurityNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to delete security: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
