package finance

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/andresbott/etna/internal/model/finance"
	"golang.org/x/text/currency"
	"net/http"
)

type Handler struct {
	Store *finance.Store
}

type accountPayload struct {
	Id       uint   `json:"id"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
	Type     string `json:"type"` // enum of type
}

var validationErr = finance.ValidationErr("")

func (h *Handler) CreateAccount(userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to create account: user not provided", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		payload := accountPayload{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		account := finance.Account{
			Name: payload.Name,
		}
		cur, err := currency.ParseISO(payload.Currency)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to parse currency: %s", err.Error()), http.StatusBadRequest)
			return
		}
		account.Currency = cur

		t, err := finance.ParseAccountType(payload.Type)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to parse account type: %s", err.Error()), http.StatusBadRequest)
			return
		}
		account.Type = t

		accID, err := h.Store.CreateAccount(r.Context(), account, userId)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				http.Error(w, fmt.Sprintf("unable to store account in DB: %s", err.Error()), http.StatusInternalServerError)
				return
			}
		}

		account.ID = accID
		respJson, err := json.Marshal(account)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJson)
	})
}

type accountUpdatePayload struct {
	Id       uint    `json:"id"`
	Name     *string `json:"name"`
	Currency *string `json:"currency"`
	Type     string  `json:"type"`
}

func (h *Handler) UpdateAccount(Id uint, userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to update account: user not provided", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		payload := accountUpdatePayload{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		account := finance.AccountUpdatePayload{
			Name: payload.Name,
		}

		if payload.Currency != nil {
			cur, err := currency.ParseISO(*payload.Currency)
			if err != nil {
				http.Error(w, fmt.Sprintf("unable to parse currency: %s", err.Error()), http.StatusBadRequest)
				return
			}
			account.Currency = &cur
		}

		if payload.Type != "" {
			t, err := finance.ParseAccountType(payload.Type)
			if err != nil {
				http.Error(w, fmt.Sprintf("unable to parse account type: %s", err.Error()), http.StatusBadRequest)
				return
			}
			account.Type = t
		}

		err = h.Store.UpdateAccount(account, Id, userId)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else if errors.Is(err, finance.AccountNotFoundErr) {
				http.Error(w, fmt.Sprintf("unable to update account in DB: %s", err.Error()), http.StatusNotFound)
				return
			} else {
				http.Error(w, fmt.Sprintf("unable to update account in DB: %s", err.Error()), http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	})
}

func (h *Handler) DeleteAccount(Id uint, userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if userId == "" {
			http.Error(w, "unable to get account: user not provided", http.StatusBadRequest)
			return
		}

		err := h.Store.DeleteAccount(r.Context(), Id, userId)
		if err != nil {
			if errors.Is(err, finance.AccountNotFoundErr) {
				http.Error(w, err.Error(), http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("unable to delete account: %s", err.Error()), http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

type listResponse struct {
	Items []accountPayload `json:"items"`
}

func (h *Handler) ListAccounts(userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		if userId == "" {
			http.Error(w, "unable to list accounts: user not provided", http.StatusBadRequest)
			return
		}

		accounts, err := h.Store.ListAccounts(r.Context(), userId)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to update account: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		outputItems := []accountPayload{}
		for _, b := range accounts {

			add := accountPayload{
				Id:       b.ID,
				Name:     b.Name,
				Currency: b.Currency.String(),
				Type:     b.Type.String(),
			}
			outputItems = append(outputItems, add)
		}

		output := listResponse{
			Items: outputItems,
		}

		respJson, err := json.Marshal(output)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJson)

		w.WriteHeader(http.StatusOK)
	})
}
