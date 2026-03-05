package finance

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"net/http"

	"github.com/andresbott/etna/internal/accounting"
	"github.com/andresbott/etna/internal/marketdata"

	"golang.org/x/text/currency"
)

type Handler struct {
	Store           *accounting.Store
	InstrumentStore *marketdata.Store
}

type accountProviderPayload struct {
	Id          uint             `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Icon        string           `json:"icon"`
	Accounts    []accountPayload `json:"accounts"`
}

var validationErr = accounting.ErrValidation("")

func (h *Handler) CreateAccountProvider() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		account := accounting.AccountProvider{
			Name:        payload.Name,
			Description: payload.Description,
			Icon:        payload.Icon,
		}

		accID, err := h.Store.CreateAccountProvider(r.Context(), account)
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
	Icon        *string `json:"icon"`
}

func (h *Handler) UpdateAccountProvider(Id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		item := accounting.AccountProviderUpdatePayload{
			Name:        payload.Name,
			Description: payload.Description,
			Icon:        payload.Icon,
		}

		err = h.Store.UpdateAccountProvider(item, Id)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else if errors.Is(err, accounting.ErrAccountProviderNotFound) {
				http.Error(w, fmt.Sprintf("unable to update account provider in DB: %s", err.Error()), http.StatusNotFound)
				return
			} else if errors.Is(err, accounting.ErrNoChanges) {
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

func (h *Handler) DeleteAccountProvider(Id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := h.Store.DeleteAccountProvider(r.Context(), Id)
		if err != nil {
			if errors.Is(err, accounting.ErrAccountProviderNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			} else if errors.Is(err, accounting.ErrAccountConstraintViolation) {
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

func (h *Handler) ListAccountProviders() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		providers, err := h.Store.ListAccountsProvider(r.Context(), true)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to update account: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		outputItems := make([]accountProviderPayload, len(providers))
		for i, b := range providers {

			accounts := make([]accountPayload, len(b.Accounts))
			for j, account := range b.Accounts {
				currencyStr := account.Currency.String()
				accounts[j] = accountPayload{
					Id:       account.ID,
					Name:     account.Name,
					Icon:     account.Icon,
					Currency: currencyStr,
					Type:     strings.ToLower(account.Type.String()),
				}
			}

			provider := accountProviderPayload{
				Id:          b.ID,
				Name:        b.Name,
				Description: b.Description,
				Icon:        b.Icon,
				Accounts:    accounts,
			}
			outputItems[i] = provider
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
	Icon       string `json:"icon"`
	Currency   string `json:"currency"`
	Type       string `json:"type"` // enum of type
}

func (h *Handler) CreateAccount() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		t := parseAccountType(payload.Type)
		if t == accounting.UnknownAccountType {
			http.Error(w, fmt.Sprintf("unable to parse account type: %s", payload.Type), http.StatusBadRequest)
			return
		}

		account := accounting.Account{
			Name:              payload.Name,
			Icon:              payload.Icon,
			AccountProviderID: payload.ProviderId,
			Type:              t,
		}
		if t.RequiresCurrency() {
			cur, err := currency.ParseISO(payload.Currency)
			if err != nil {
				http.Error(w, fmt.Sprintf("unable to parse currency: %s", err.Error()), http.StatusBadRequest)
				return
			}
			account.Currency = cur
		}
		// Investment/Unvested: ignore any currency in the request

		accID, err := h.Store.CreateAccount(r.Context(), account)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			} else {
				http.Error(w, fmt.Sprintf("unable to Store account in DB: %s", err.Error()), http.StatusInternalServerError)
				return
			}
		}

		currencyStr := ""
		if account.Type.RequiresCurrency() {
			currencyStr = account.Currency.String()
		}
		responsePayload := accountPayload{
			Id:       accID,
			Name:     account.Name,
			Icon:     account.Icon,
			Currency: currencyStr,
			Type:     account.Type.String(),
		}

		respJson, err := json.Marshal(responsePayload)
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
	Icon     *string `json:"icon"`
	Currency *string `json:"currency"`
	Type     string  `json:"type"`
}

func (h *Handler) UpdateAccount(Id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		account := accounting.AccountUpdatePayload{
			Name: payload.Name,
		}

		if payload.Icon != nil {
			account.Icon = payload.Icon
		}

		// Resolve target type for currency: from payload or current account
		targetType := accounting.UnknownAccountType
		if payload.Type != "" {
			targetType = parseAccountType(payload.Type)
		}
		if targetType == accounting.UnknownAccountType && payload.Currency != nil {
			current, err := h.Store.GetAccount(r.Context(), Id)
			if err != nil {
				http.Error(w, fmt.Sprintf("unable to get account: %s", err.Error()), http.StatusInternalServerError)
				return
			}
			targetType = current.Type
		}
		if payload.Currency != nil && targetType.RequiresCurrency() {
			cur, err := currency.ParseISO(*payload.Currency)
			if err != nil {
				http.Error(w, fmt.Sprintf("unable to parse currency: %s", err.Error()), http.StatusBadRequest)
				return
			}
			account.Currency = &cur
		}
		// Investment/Unvested: ignore currency in request (handler does not set it; store will clear if needed)

		if payload.Type != "" {
			t := parseAccountType(payload.Type)
			if t == accounting.UnknownAccountType {
				http.Error(w, fmt.Sprintf("unable to parse account type: %s", err.Error()), http.StatusBadRequest)
				return
			}
			account.Type = t
		}

		err = h.Store.UpdateAccount(r.Context(), account, Id)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else if errors.Is(err, accounting.ErrAccountNotFound) {
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

func (h *Handler) DeleteAccount(Id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := h.Store.DeleteAccount(r.Context(), Id)
		if err != nil {
			if errors.Is(err, accounting.ErrAccountNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)

			} else if errors.Is(err, accounting.ErrAccountContainsEntries) {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				http.Error(w, fmt.Sprintf("unable to delete account: %s", err.Error()), http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

const (
	cashAccountStr       = "cash"
	checkinAccountStr    = "checkin"
	savingsAccountStr    = "savings"
	investmentAccountStr = "investment"
	unvestedAccountStr   = "unvested"
)

func parseAccountType(in string) accounting.AccountType {
	in = strings.TrimSpace(in)
	in = strings.ToLower(in)
	switch in {
	case cashAccountStr:
		return accounting.CashAccountType
	case checkinAccountStr:
		return accounting.CheckinAccountType
	case savingsAccountStr:
		return accounting.SavingsAccountType
	case investmentAccountStr:
		return accounting.InvestmentAccountType
	case unvestedAccountStr:
		return accounting.UnvestedAccountType
	default:
		return accounting.UnknownAccountType
	}
}
