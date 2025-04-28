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

type accountProviderPayload struct {
	Id          uint             `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Accounts    []accountPayload `json:"accounts"`
}

func (h *Handler) CreateAccountProvider(userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to create account: user not provided", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		payload := accountProviderPayload{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		account := finance.AccountProvider{
			Name:        payload.Name,
			Description: payload.Description,
		}

		accID, err := h.Store.CreateAccountProvider(r.Context(), account, userId)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			} else {
				http.Error(w, fmt.Sprintf("unable to Store account in DB: %s", err.Error()), http.StatusInternalServerError)
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

type accountProviderUpdatePayload struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func (h *Handler) UpdateAccountProvider(Id uint, userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userId == "" {
			http.Error(w, "unable to update account: user not provided", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		payload := accountProviderUpdatePayload{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		item := finance.AccountProviderUpdatePayload{
			Name:        payload.Name,
			Description: payload.Description,
			//Accounts    []Account
		}

		err = h.Store.UpdateAccountProvider(item, Id, userId)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else if errors.Is(err, finance.ErrAccountProviderNotFound) {
				http.Error(w, fmt.Sprintf("unable to update account provider in DB: %s", err.Error()), http.StatusNotFound)
				return
			} else if errors.Is(err, finance.ErrNoChanges) {
				http.Error(w, "no changes applied", http.StatusBadRequest)
				return
			} else {
				http.Error(w, fmt.Sprintf("unable to update account provider in DB: %s", err.Error()), http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	})
}

func (h *Handler) DeleteAccountProvider(Id uint, userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if userId == "" {
			http.Error(w, "unable to get account: user not provided", http.StatusBadRequest)
			return
		}

		err := h.Store.DeleteAccountProvider(r.Context(), Id, userId)
		if err != nil {
			if errors.Is(err, finance.ErrAccountProviderNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			} else if errors.Is(err, finance.ErrAccountConstraintViolation) {
				http.Error(w, fmt.Sprintf("unable to delete account provider: %s", err.Error()), http.StatusConflict)
				return
			} else {
				http.Error(w, fmt.Sprintf("unable to delete account provider: %s", err.Error()), http.StatusInternalServerError)
				return
			}

		}
		w.WriteHeader(http.StatusOK)
	})
}

type listResponse struct {
	Items []accountProviderPayload `json:"items"`
}

func (h *Handler) ListAccountProviders(userId string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		if userId == "" {
			http.Error(w, "unable to list accounts: user not provided", http.StatusBadRequest)
			return
		}

		providers, err := h.Store.ListAccountsProvider(r.Context(), userId, true)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to update account: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		outputItems := make([]accountProviderPayload, len(providers))
		for i, b := range providers {

			accounts := make([]accountPayload, len(b.Accounts))
			for j, account := range b.Accounts {
				accounts[j] = accountPayload{
					Id:       account.ID,
					Name:     account.Name,
					Currency: account.Currency.String(),
					Type:     account.Type.String(),
				}
			}

			provider := accountProviderPayload{
				Id:          b.ID,
				Name:        b.Name,
				Description: b.Description,
				Accounts:    accounts,
			}
			outputItems[i] = provider
		}

		if len(outputItems) == 0 {
			outputItems = []accountProviderPayload{
				{
					Id:          3,
					Name:        "test",
					Description: "test description",
					Accounts: []accountPayload{
						{
							Id:       4,
							Name:     "acc1",
							Currency: "USD",
							Type:     finance.BankAccount,
						},
					},
				},
			}
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

// =======================================================================================
// Account
// =======================================================================================

type accountPayload struct {
	Id         uint   `json:"id"`
	ProviderId uint   `json:"providerId,omitempty"`
	Name       string `json:"name"`
	Currency   string `json:"currency"`
	Type       string `json:"type"` // enum of type
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
			Name:              payload.Name,
			AccountProviderID: payload.ProviderId,
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
				return
			} else {
				http.Error(w, fmt.Sprintf("unable to Store account in DB: %s", err.Error()), http.StatusInternalServerError)
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
			} else if errors.Is(err, finance.ErrAccountNotFound) {
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
			if errors.Is(err, finance.ErrAccountNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("unable to delete account: %s", err.Error()), http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
