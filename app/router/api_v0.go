package router

import (
	"fmt"
	"github.com/andresbott/etna/app/router/handlers/backup"
	finHandler "github.com/andresbott/etna/app/router/handlers/finance"
	"github.com/go-bumbu/userauth/authenticator"
	"github.com/go-bumbu/userauth/handlers/sessionauth"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

func (h *MainAppHandler) attachApiV0(r *mux.Router) error {
	// this sub router does enforce authentication
	authHandlers := []authenticator.AuthHandler{h.SessionAuth}
	auth := authenticator.New(authHandlers, h.logger, nil, nil)

	r.Use(auth.Middleware)

	// attach api paths to api/v0
	h.accountingAPI(r)
	h.backupApi(r)

	// send a 400 error on everything else on the API
	r.PathPrefix("").HandlerFunc(StatusErrText(http.StatusBadRequest, "wrong api call"))

	return nil
}

const finProviderPath = "/fin/provider"
const finAccountPath = "/fin/account"
const finEntries = "/fin/entries"
const finCategoryIncome = "/fin/category/income"
const finCategoryExpense = "/fin/category/expense"
const finReport = "/fin/report"

// this api surface is quite inconsistent, I know....
// I haven't put too much thought into it for now and I will change it in the future
//
//nolint:gocognit,gocyclo // the function is quite big and verbose but easy to follow
func (h *MainAppHandler) accountingAPI(r *mux.Router) {

	finHndlr := finHandler.Handler{Store: h.finStore}

	// ==========================================================================
	// Account Providers
	// ==========================================================================

	r.Path(finProviderPath).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read bookmark: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		finHndlr.ListAccountProviders(userData.UserId).ServeHTTP(w, r)
	})

	r.Path(finProviderPath).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read bookmark: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		finHndlr.CreateAccountProvider(userData.UserId).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{ID}", finProviderPath)).Methods(http.MethodPut).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read bookmark: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}

		finHndlr.UpdateAccountProvider(itemId, userData.UserId).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{ID}", finProviderPath)).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read bookmark: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}

		finHndlr.DeleteAccountProvider(itemId, userData.UserId).ServeHTTP(w, r)
	})
	// ==========================================================================
	// Accounts
	// ==========================================================================

	r.Path(finAccountPath).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read bookmark: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		finHndlr.CreateAccount(userData.UserId).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{ID}", finAccountPath)).Methods(http.MethodPut).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read bookmark: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}

		finHndlr.UpdateAccount(itemId, userData.UserId).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{ID}", finAccountPath)).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read bookmark: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}

		finHndlr.DeleteAccount(itemId, userData.UserId).ServeHTTP(w, r)
	})
	// ==========================================================================
	// Entry Category
	// ==========================================================================

	catHandler := finHandler.CategoryHandler{Store: h.finStore}

	// list income categories
	r.Path(finCategoryIncome).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		catHandler.ListIncome(0, userData.UserId).ServeHTTP(w, r)
	})

	// create income categories
	r.Path(finCategoryIncome).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		catHandler.CreateIncome(userData.UserId).ServeHTTP(w, r)
	})

	// Update income categories
	r.Path(fmt.Sprintf("%s/{ID}", finCategoryIncome)).Methods(http.MethodPut).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		catHandler.UpdateIncome(itemId, userData.UserId).ServeHTTP(w, r)
	})
	// TODO Move exposed as API yet

	// delete income category
	r.Path(fmt.Sprintf("%s/{ID}", finCategoryIncome)).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		catHandler.DeleteIncome(itemId, userData.UserId).ServeHTTP(w, r)
	})

	// list expense categories
	r.Path(finCategoryExpense).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		catHandler.ListExpense(0, userData.UserId).ServeHTTP(w, r)
	})

	// create expense categories
	r.Path(finCategoryExpense).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		catHandler.CreateExpense(userData.UserId).ServeHTTP(w, r)
	})

	// Update expense categories
	r.Path(fmt.Sprintf("%s/{ID}", finCategoryExpense)).Methods(http.MethodPut).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		catHandler.UpdateExpense(itemId, userData.UserId).ServeHTTP(w, r)
	})
	// TODO Move exposed as API yet

	// delete expense category
	r.Path(fmt.Sprintf("%s/{ID}", finCategoryExpense)).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		catHandler.DeleteExpense(itemId, userData.UserId).ServeHTTP(w, r)
	})

	// ==========================================================================
	// Entries
	// ==========================================================================

	r.Path(finEntries).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		finHndlr.ListTx(userData.UserId).ServeHTTP(w, r)
	})

	r.Path(finEntries).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		finHndlr.CreateTx(userData.UserId).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{ID}", finEntries)).Methods(http.MethodPut).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}

		finHndlr.UpdateTx(itemId, userData.UserId).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{ID}", finEntries)).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}

		finHndlr.DeleteTx(itemId, userData.UserId).ServeHTTP(w, r)
	})

	// ==========================================================================
	// Report
	// ==========================================================================

	r.Path(fmt.Sprintf("%s/income-expense", finReport)).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		finHndlr.IncomeExpenseReport(userData.UserId).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/balance", finReport)).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		finHndlr.AccountBalance(userData.UserId).ServeHTTP(w, r)
	})

}

const backupPath = "/backup"
const restorePath = "/restore"

func (h *MainAppHandler) backupApi(r *mux.Router) {

	backupHndl := backup.Handler{
		Destination: h.backupDestination,
		Store:       h.finStore,
	}
	r.Path(backupPath).Methods(http.MethodGet).Handler(backupHndl.List())
	r.Path(backupPath).Methods(http.MethodPost).Handler(backupHndl.CreateBackup())
	r.Path(fmt.Sprintf("%s/{ID}", backupPath)).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		itemId, ok := vars["ID"]
		if !ok {
			http.Error(w, "could not extract tag id from request context", http.StatusInternalServerError)
			return
		}
		if itemId == "" {
			http.Error(w, "no id provided", http.StatusBadRequest)
			return
		}
		backupHndl.Delete(itemId).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{ID}", backupPath)).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		itemId, ok := vars["ID"]
		if !ok {
			http.Error(w, "could not extract tag id from request context", http.StatusInternalServerError)
			return
		}
		if itemId == "" {
			http.Error(w, "no id provided", http.StatusBadRequest)
			return
		}
		backupHndl.Download(itemId).ServeHTTP(w, r)
		return
	})
	// TODO restore from uploaded file
	r.Path(restorePath).Methods(http.MethodPost).Handler(backupHndl.CreateBackup())

	r.Path(fmt.Sprintf("%s/{ID}", restorePath)).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		itemId, ok := vars["ID"]
		if !ok {
			http.Error(w, "could not extract tag id from request context", http.StatusInternalServerError)
			return
		}
		if itemId == "" {
			http.Error(w, "no id provided", http.StatusBadRequest)
			return
		}
		// TODO restore from existing Id
	})

}

// Extract the ID from the request url. based on gorilla url path vars
func getId(r *http.Request) (uint, *httpError) {
	vars := mux.Vars(r)
	tagId, ok := vars["ID"]
	if !ok {
		return 0, &httpError{
			Error: "could not extract tag id from request context",
			Code:  http.StatusInternalServerError,
		}
	}
	if tagId == "" {
		return 0, &httpError{
			Error: "no tag id provided",
			Code:  http.StatusBadRequest,
		}
	}

	u64, err := strconv.ParseUint(tagId, 10, 64)
	if err != nil {
		return 0, &httpError{
			Error: "unable to convert id to number",
			Code:  http.StatusBadRequest,
		}
	}
	return uint(u64), nil
}

type httpError struct {
	Error string
	Code  int
}
