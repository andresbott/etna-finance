package router

import (
	"fmt"
	"net/http"
	"strconv"

	handlrs "github.com/andresbott/etna/app/router/handlers"
	"github.com/andresbott/etna/app/router/handlers/backup"
	csvimportHandler "github.com/andresbott/etna/app/router/handlers/csvimport"
	finHandler "github.com/andresbott/etna/app/router/handlers/finance"
	mktHandler "github.com/andresbott/etna/app/router/handlers/marketdata"
	taskHandler "github.com/andresbott/etna/app/router/handlers/tasks"
	toolsDataHandler "github.com/andresbott/etna/app/router/handlers/toolsdata"
	"github.com/go-bumbu/userauth/authenticator"
	"github.com/go-bumbu/userauth/handlers/sessionauth"
	"github.com/gorilla/mux"
)

func (h *MainAppHandler) attachApiV0(r *mux.Router) error {
	var authMiddleware func(http.Handler) http.Handler
	if h.authDisabled {
		authMiddleware = authenticator.New(
			[]authenticator.AuthHandler{NewNoAuthHandler(h.defaultUser)},
			h.logger, nil, nil,
		).Middleware
	} else {
		auth := authenticator.New([]authenticator.AuthHandler{h.SessionAuth}, h.logger, nil, nil)
		authMiddleware = auth.Middleware
	}
	r.Use(authMiddleware)

	// attach api paths to api/v0
	h.settingsApi(r)
	h.accountingAPI(r)
	h.marketDataAPI(r)
	h.backupApi(r)
	h.tasksApi(r)
	if h.csvImportStore != nil {
		h.csvImportAPI(r)
	}
	h.toolsDataAPI(r)

	// send a 400 error on everything else on the API
	r.PathPrefix("").HandlerFunc(StatusErrText(http.StatusBadRequest, "wrong api call"))

	return nil
}

const settingsPath = "/settings"

func (h *MainAppHandler) settingsApi(r *mux.Router) {
	getSymbols := func() ([]string, error) { return h.marketStore.ListPriceSymbols() }
	r.Path(settingsPath).Methods(http.MethodGet).Handler(handlrs.SettingsHandlerWithMarketData(h.appSettings, getSymbols))
}

const finProviderPath = "/fin/provider"
const finAccountPath = "/fin/account"
const finEntries = "/fin/entries"
const finCategoryIncome = "/fin/category/income"
const finCategoryExpense = "/fin/category/expense"
const finInstrumentPath = "/fin/instrument"
const finPortfolio = "/fin/portfolio"
const finReport = "/fin/report"

// this api surface is quite inconsistent, I know....
// I haven't put too much thought into it for now and I will change it in the future
//
//nolint:gocognit,gocyclo // the function is quite big and verbose but easy to follow
func (h *MainAppHandler) accountingAPI(r *mux.Router) {

	finHndlr := finHandler.Handler{Store: h.finStore, InstrumentStore: h.marketStore, FileStore: h.attachmentStore}

	// ==========================================================================
	// Account Providers
	// ==========================================================================

	r.Path(finProviderPath).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		finHndlr.ListAccountProviders().ServeHTTP(w, r)
	})

	r.Path(finProviderPath).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		finHndlr.CreateAccountProvider().ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{id}", finProviderPath)).Methods(http.MethodPut).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}

		finHndlr.UpdateAccountProvider(itemId).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{id}", finProviderPath)).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}

		finHndlr.DeleteAccountProvider(itemId).ServeHTTP(w, r)
	})
	// ==========================================================================
	// Accounts
	// ==========================================================================

	r.Path(finAccountPath).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		finHndlr.CreateAccount().ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{id}", finAccountPath)).Methods(http.MethodPut).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}

		finHndlr.UpdateAccount(itemId).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{id}", finAccountPath)).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}

		finHndlr.DeleteAccount(itemId).ServeHTTP(w, r)
	})
	// ==========================================================================
	// Entry Category
	// ==========================================================================

	catHandler := finHandler.CategoryHandler{Store: h.finStore}

	// list income categories
	r.Path(finCategoryIncome).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		catHandler.ListIncome(0).ServeHTTP(w, r)
	})

	// create income categories
	r.Path(finCategoryIncome).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		catHandler.CreateIncome().ServeHTTP(w, r)
	})

	// Update income categories
	r.Path(fmt.Sprintf("%s/{id}", finCategoryIncome)).Methods(http.MethodPut).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		catHandler.UpdateIncome(itemId).ServeHTTP(w, r)
	})
	// TODO Move exposed as API yet

	// delete income category
	r.Path(fmt.Sprintf("%s/{id}", finCategoryIncome)).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		catHandler.DeleteIncome(itemId).ServeHTTP(w, r)
	})

	// list expense categories
	r.Path(finCategoryExpense).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		catHandler.ListExpense(0).ServeHTTP(w, r)
	})

	// create expense categories
	r.Path(finCategoryExpense).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		catHandler.CreateExpense().ServeHTTP(w, r)
	})

	// Update expense categories
	r.Path(fmt.Sprintf("%s/{id}", finCategoryExpense)).Methods(http.MethodPut).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		catHandler.UpdateExpense(itemId).ServeHTTP(w, r)
	})
	// TODO Move exposed as API yet

	// delete expense category
	r.Path(fmt.Sprintf("%s/{id}", finCategoryExpense)).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		catHandler.DeleteExpense(itemId).ServeHTTP(w, r)
	})

	// ==========================================================================
	// Entries
	// ==========================================================================

	r.Path(finEntries).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		finHndlr.ListTx().ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{id}", finEntries)).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		finHndlr.GetTx(itemId).ServeHTTP(w, r)
	})

	r.Path(finEntries).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		finHndlr.CreateTx().ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{id}", finEntries)).Methods(http.MethodPut).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}

		finHndlr.UpdateTx(itemId).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{id}", finEntries)).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}

		finHndlr.DeleteTx(itemId).ServeHTTP(w, r)
	})

	// ==========================================================================
	// Attachments
	// ==========================================================================

	r.Path(fmt.Sprintf("%s/{id}/attachment", finEntries)).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}

		finHndlr.UploadAttachment(itemId).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{id}/attachment", finEntries)).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}

		finHndlr.GetAttachment(itemId).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{id}/attachment", finEntries)).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}

		finHndlr.DeleteAttachment(itemId).ServeHTTP(w, r)
	})

	// ==========================================================================
	// Instruments
	// ==========================================================================

	if h.appSettings.Instruments { //nolint:nestif // route registration for optional features is inherently nested
		r.Path(finInstrumentPath).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := sessionauth.CtxGetUserData(r)
			if err != nil {
				http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
				return
			}
			finHndlr.ListInstruments().ServeHTTP(w, r)
		})

		r.Path(finInstrumentPath).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := sessionauth.CtxGetUserData(r)
			if err != nil {
				http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
				return
			}
			finHndlr.CreateInstrument().ServeHTTP(w, r)
		})

		r.Path(fmt.Sprintf("%s/{id}", finInstrumentPath)).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := sessionauth.CtxGetUserData(r)
			if err != nil {
				http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
				return
			}
			itemId, httpErr := getId(r)
			if httpErr != nil {
				http.Error(w, httpErr.Error, httpErr.Code)
				return
			}
			finHndlr.GetInstrument(itemId).ServeHTTP(w, r)
		})

		r.Path(fmt.Sprintf("%s/{id}", finInstrumentPath)).Methods(http.MethodPut).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := sessionauth.CtxGetUserData(r)
			if err != nil {
				http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
				return
			}
			itemId, httpErr := getId(r)
			if httpErr != nil {
				http.Error(w, httpErr.Error, httpErr.Code)
				return
			}
			finHndlr.UpdateInstrument(itemId).ServeHTTP(w, r)
		})

		r.Path(fmt.Sprintf("%s/{id}", finInstrumentPath)).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := sessionauth.CtxGetUserData(r)
			if err != nil {
				http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
				return
			}
			itemId, httpErr := getId(r)
			if httpErr != nil {
				http.Error(w, httpErr.Error, httpErr.Code)
				return
			}
			finHndlr.DeleteInstrument(itemId).ServeHTTP(w, r)
		})
	} else {
		instrumentsDisabled := StatusErrText(http.StatusForbidden, "financial instruments are disabled")
		r.PathPrefix(finInstrumentPath).HandlerFunc(instrumentsDisabled)
	}

	// ==========================================================================
	// Portfolio
	// ==========================================================================

	if h.appSettings.Instruments {
		r.Path(fmt.Sprintf("%s/positions", finPortfolio)).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := sessionauth.CtxGetUserData(r); err != nil {
				http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
				return
			}
			finHndlr.ListPositions().ServeHTTP(w, r)
		})

		r.Path(fmt.Sprintf("%s/positions/{accountId}/{instrumentId}", finPortfolio)).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := sessionauth.CtxGetUserData(r); err != nil {
				http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
				return
			}
			finHndlr.GetPositionDetail().ServeHTTP(w, r)
		})

		r.Path(fmt.Sprintf("%s/lots", finPortfolio)).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := sessionauth.CtxGetUserData(r); err != nil {
				http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
				return
			}
			finHndlr.ListLots().ServeHTTP(w, r)
		})

		r.Path(fmt.Sprintf("%s/trades", finPortfolio)).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := sessionauth.CtxGetUserData(r); err != nil {
				http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
				return
			}
			finHndlr.ListPortfolioTrades().ServeHTTP(w, r)
		})
	}

	// ==========================================================================
	// Report
	// ==========================================================================

	r.Path(fmt.Sprintf("%s/income-expense", finReport)).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		finHndlr.IncomeExpenseReport().ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/balance", finReport)).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := sessionauth.CtxGetUserData(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		finHndlr.AccountBalance().ServeHTTP(w, r)
	})

}

// ==========================================================================
// Market Data
// ==========================================================================

const finMarketDataPath = "/fin/marketdata"
const finFXPath = "/fin/fx"

func (h *MainAppHandler) marketDataAPI(r *mux.Router) {
	mktHndlr := mktHandler.Handler{
		Store:        h.marketStore,
		MainCurrency: h.appSettings.MainCurrency,
		Currencies:   h.appSettings.Currencies,
	}

	// GET /fin/marketdata/symbols (list symbols with price data; must be before {symbol} routes)
	r.Path(fmt.Sprintf("%s/symbols", finMarketDataPath)).Methods(http.MethodGet).Handler(mktHndlr.ListSymbols())

	// GET /fin/marketdata/{symbol}/prices?start=YYYY-MM-DD&end=YYYY-MM-DD
	r.Path(fmt.Sprintf("%s/{symbol}/prices", finMarketDataPath)).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		symbol := mux.Vars(r)["symbol"]
		mktHndlr.ListPrices(symbol).ServeHTTP(w, r)
	})

	// POST /fin/marketdata/{symbol}/prices  (single point)
	r.Path(fmt.Sprintf("%s/{symbol}/prices", finMarketDataPath)).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		symbol := mux.Vars(r)["symbol"]
		mktHndlr.CreatePrice(symbol).ServeHTTP(w, r)
	})

	// POST /fin/marketdata/{symbol}/prices/bulk
	r.Path(fmt.Sprintf("%s/{symbol}/prices/bulk", finMarketDataPath)).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		symbol := mux.Vars(r)["symbol"]
		mktHndlr.CreatePricesBulk(symbol).ServeHTTP(w, r)
	})

	// GET /fin/marketdata/{symbol}/prices/latest
	r.Path(fmt.Sprintf("%s/{symbol}/prices/latest", finMarketDataPath)).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		symbol := mux.Vars(r)["symbol"]
		mktHndlr.LatestPrice(symbol).ServeHTTP(w, r)
	})

	// PUT /fin/marketdata/prices/{id}
	r.Path(fmt.Sprintf("%s/prices/{id}", finMarketDataPath)).Methods(http.MethodPut).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		mktHndlr.UpdatePrice(itemId).ServeHTTP(w, r)
	})

	// DELETE /fin/marketdata/prices/{id}
	r.Path(fmt.Sprintf("%s/prices/{id}", finMarketDataPath)).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		mktHndlr.DeletePrice(itemId).ServeHTTP(w, r)
	})

	// ==========================================================================
	// Currency exchange (FX) rates — main currency + secondary currencies from settings
	// ==========================================================================

	// GET /fin/fx/pairs (list configured pairs, e.g. CHF/USD, CHF/EUR)
	r.Path(fmt.Sprintf("%s/pairs", finFXPath)).Methods(http.MethodGet).Handler(mktHndlr.ListFXPairs())

	// GET /fin/fx/{main}/{secondary}/rates/latest (must be before /rates to avoid "latest" as segment)
	r.Path(fmt.Sprintf("%s/{main}/{secondary}/rates/latest", finFXPath)).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := mux.Vars(r)
		mktHndlr.LatestFXRate(v["main"], v["secondary"]).ServeHTTP(w, r)
	})
	// POST /fin/fx/{main}/{secondary}/rates/bulk
	r.Path(fmt.Sprintf("%s/{main}/{secondary}/rates/bulk", finFXPath)).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := mux.Vars(r)
		mktHndlr.CreateFXRatesBulk(v["main"], v["secondary"]).ServeHTTP(w, r)
	})
	// GET /fin/fx/{main}/{secondary}/rates?start=YYYY-MM-DD&end=YYYY-MM-DD
	r.Path(fmt.Sprintf("%s/{main}/{secondary}/rates", finFXPath)).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := mux.Vars(r)
		mktHndlr.ListFXRates(v["main"], v["secondary"]).ServeHTTP(w, r)
	})
	// POST /fin/fx/{main}/{secondary}/rates (single rate)
	r.Path(fmt.Sprintf("%s/{main}/{secondary}/rates", finFXPath)).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := mux.Vars(r)
		mktHndlr.CreateFXRate(v["main"], v["secondary"]).ServeHTTP(w, r)
	})
	// PUT /fin/fx/rates/{id}
	r.Path(fmt.Sprintf("%s/rates/{id}", finFXPath)).Methods(http.MethodPut).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		mktHndlr.UpdateFXRate(itemId).ServeHTTP(w, r)
	})
	// DELETE /fin/fx/rates/{id}
	r.Path(fmt.Sprintf("%s/rates/{id}", finFXPath)).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		mktHndlr.DeleteFXRate(itemId).ServeHTTP(w, r)
	})
}

const tasksPath = "/tasks"
const tasksExecutionsPath = "/tasks/executions"

func (h *MainAppHandler) tasksApi(r *mux.Router) {
	if h.taskRunner == nil {
		r.PathPrefix(tasksPath).HandlerFunc(StatusErr(http.StatusServiceUnavailable))
		return
	}
	th := taskHandler.Handler{
		Runner:         h.taskRunner,
		ScheduleStore:  h.scheduleStore,
		Scheduler:      h.scheduler,
		TaskLogGetter:  h.taskLogGetter,
		ProductionMode: h.productionMode,
	}
	r.Path(tasksPath).Methods(http.MethodGet).Handler(th.ListTasks())
	r.Path(tasksExecutionsPath).Methods(http.MethodGet).Handler(th.ListExecutions())
	r.Path(fmt.Sprintf("%s/executions/{id}/logs", tasksPath)).Methods(http.MethodGet).Handler(th.GetExecutionLog())
	r.Path(fmt.Sprintf("%s/executions/{id}/cancel", tasksPath)).Methods(http.MethodPost).Handler(th.CancelExecution())
	r.Path(fmt.Sprintf("%s/{name}/trigger", tasksPath)).Methods(http.MethodPost).Handler(th.TriggerTask())
	r.Path(fmt.Sprintf("%s/{name}", tasksPath)).Methods(http.MethodGet).Handler(th.GetTask())
	r.Path(fmt.Sprintf("%s/{name}", tasksPath)).Methods(http.MethodPut).Handler(th.UpsertTask())
	r.Path(fmt.Sprintf("%s/{name}", tasksPath)).Methods(http.MethodPatch).Handler(th.PatchTask())
	r.Path(fmt.Sprintf("%s/{name}", tasksPath)).Methods(http.MethodDelete).Handler(th.DeleteTaskSchedule())
}

const importProfilePath = "/import/profiles"
const importCategoryRuleGroupPath = "/import/category-rule-groups"
const importParsePath = "/import/parse"
const importSubmitPath = "/import/submit"
const importPreviewPath = "/import/preview"
const importReapplyPreviewPath = "/import/reapply-preview"
const importReapplySubmitPath = "/import/reapply-submit"

func (h *MainAppHandler) csvImportAPI(r *mux.Router) {
	profileHndlr := csvimportHandler.ProfileHandler{Store: h.csvImportStore}
	ruleGroupHndlr := csvimportHandler.CategoryRuleGroupHandler{Store: h.csvImportStore}
	importHndlr := csvimportHandler.ImportHandler{CsvStore: h.csvImportStore, FinStore: h.finStore}

	registerCrudRoutes(r, importProfilePath, crudHandlers{
		list:   profileHndlr.ListProfiles,
		create: profileHndlr.CreateProfile,
		update: profileHndlr.UpdateProfile,
		delete: profileHndlr.DeleteProfile,
	})

	registerCrudRoutes(r, importCategoryRuleGroupPath, crudHandlers{
		list:   ruleGroupHndlr.ListCategoryRuleGroups,
		create: ruleGroupHndlr.CreateCategoryRuleGroup,
		update: ruleGroupHndlr.UpdateCategoryRuleGroup,
		delete: ruleGroupHndlr.DeleteCategoryRuleGroup,
	})

	h.csvImportRulePatternRoutes(r, ruleGroupHndlr)
	h.csvImportParseRoutes(r, importHndlr)
	h.csvImportReapplyRoutes(r, importHndlr)
}

type crudHandlers struct {
	list   func() http.Handler
	create func() http.Handler
	update func(uint) http.Handler
	delete func(uint) http.Handler
}

func registerCrudRoutes(r *mux.Router, basePath string, h crudHandlers) {
	r.Path(basePath).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		h.list().ServeHTTP(w, r)
	})

	r.Path(basePath).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		h.create().ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{id}", basePath)).Methods(http.MethodPut).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		h.update(itemId).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{id}", basePath)).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		h.delete(itemId).ServeHTTP(w, r)
	})
}

func (h *MainAppHandler) csvImportRulePatternRoutes(r *mux.Router, ruleGroupHndlr csvimportHandler.CategoryRuleGroupHandler) {
	r.Path(fmt.Sprintf("%s/{groupId}/patterns", importCategoryRuleGroupPath)).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		groupId, httpErr := getVarId(r, "groupId")
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		ruleGroupHndlr.CreateCategoryRulePattern(groupId).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{groupId}/patterns/{id}", importCategoryRuleGroupPath)).Methods(http.MethodPut).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		groupId, httpErr := getVarId(r, "groupId")
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		patternId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		ruleGroupHndlr.UpdateCategoryRulePattern(groupId, patternId).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{groupId}/patterns/{id}", importCategoryRuleGroupPath)).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		groupId, httpErr := getVarId(r, "groupId")
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		patternId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		ruleGroupHndlr.DeleteCategoryRulePattern(groupId, patternId).ServeHTTP(w, r)
	})
}

func (h *MainAppHandler) csvImportParseRoutes(r *mux.Router, importHndlr csvimportHandler.ImportHandler) {
	r.Path(importParsePath).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		importHndlr.ParseCSV().ServeHTTP(w, r)
	})

	r.Path(importSubmitPath).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		importHndlr.SubmitImport().ServeHTTP(w, r)
	})

	r.Path(importPreviewPath).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		importHndlr.PreviewCSV().ServeHTTP(w, r)
	})
}

func (h *MainAppHandler) csvImportReapplyRoutes(r *mux.Router, importHndlr csvimportHandler.ImportHandler) {
	r.Path(importReapplyPreviewPath).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		importHndlr.ReapplyPreview().ServeHTTP(w, r)
	})

	r.Path(importReapplySubmitPath).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		importHndlr.ReapplySubmit().ServeHTTP(w, r)
	})
}

const backupPath = "/backup"
const restorePath = "/restore"

func (h *MainAppHandler) backupApi(r *mux.Router) {

	backupHndl := backup.Handler{
		Destination:    h.backupDestination,
		Store:          h.finStore,
		MdStore:        h.marketStore,
		CsvStore:       h.csvImportStore,
		ToolsDataStore: h.toolsDataStore,
	}
	r.Path(backupPath).Methods(http.MethodGet).Handler(backupHndl.List())
	r.Path(backupPath).Methods(http.MethodPost).Handler(backupHndl.CreateBackup())
	r.Path(fmt.Sprintf("%s/{id}", backupPath)).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		itemId, ok := vars["id"]
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

	r.Path(fmt.Sprintf("%s/{id}", backupPath)).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		itemId, ok := vars["id"]
		if !ok {
			http.Error(w, "could not extract tag id from request context", http.StatusInternalServerError)
			return
		}
		if itemId == "" {
			http.Error(w, "no id provided", http.StatusBadRequest)
			return
		}
		backupHndl.Download(itemId).ServeHTTP(w, r)
	})
	// Restore from uploaded file
	r.Path(restorePath).Methods(http.MethodPost).Handler(backupHndl.RestoreUpload())

	// Restore from existing backup id
	r.Path(fmt.Sprintf("%s/{id}", restorePath)).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		itemId, ok := vars["id"]
		if !ok {
			http.Error(w, "could not extract tag id from request context", http.StatusInternalServerError)
			return
		}
		if itemId == "" {
			http.Error(w, "no id provided", http.StatusBadRequest)
			return
		}
		backupHndl.RestoreFromExisting(itemId).ServeHTTP(w, r)
	})

}

// Extract the id from the request url. based on gorilla url path vars
func getId(r *http.Request) (uint, *httpError) {
	vars := mux.Vars(r)
	tagId, ok := vars["id"]
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

func getVarId(r *http.Request, name string) (uint, *httpError) {
	vars := mux.Vars(r)
	val, ok := vars[name]
	if !ok {
		return 0, &httpError{
			Error: fmt.Sprintf("could not extract %s from request context", name),
			Code:  http.StatusInternalServerError,
		}
	}
	if val == "" {
		return 0, &httpError{
			Error: fmt.Sprintf("no %s provided", name),
			Code:  http.StatusBadRequest,
		}
	}
	u64, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, &httpError{
			Error: fmt.Sprintf("unable to convert %s to number", name),
			Code:  http.StatusBadRequest,
		}
	}
	return uint(u64), nil
}

type httpError struct {
	Error string
	Code  int
}

const toolsDataPath = "/tools/{toolType:[a-z0-9-]+}/cases"

func (h *MainAppHandler) toolsDataAPI(r *mux.Router) {
	if h.toolsDataStore == nil {
		return
	}
	tdHandler := toolsDataHandler.Handler{Store: h.toolsDataStore, FileStore: h.attachmentStore}

	r.Path(toolsDataPath).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		toolType := mux.Vars(r)["toolType"]
		tdHandler.ListCases(toolType).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{id}", toolsDataPath)).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		toolType := mux.Vars(r)["toolType"]
		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		tdHandler.GetCase(toolType, itemId).ServeHTTP(w, r)
	})

	r.Path(toolsDataPath).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		toolType := mux.Vars(r)["toolType"]
		tdHandler.CreateCase(toolType).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{id}", toolsDataPath)).Methods(http.MethodPut).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		toolType := mux.Vars(r)["toolType"]
		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		tdHandler.UpdateCase(toolType, itemId).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{id}", toolsDataPath)).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		toolType := mux.Vars(r)["toolType"]
		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		tdHandler.DeleteCase(toolType, itemId).ServeHTTP(w, r)
	})

	// Attachment routes
	attPath := fmt.Sprintf("%s/{id}/attachment", toolsDataPath)

	r.Path(attPath).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		toolType := mux.Vars(r)["toolType"]
		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		tdHandler.UploadAttachment(toolType, itemId).ServeHTTP(w, r)
	})

	r.Path(attPath).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		toolType := mux.Vars(r)["toolType"]
		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		tdHandler.GetAttachment(toolType, itemId).ServeHTTP(w, r)
	})

	r.Path(attPath).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		toolType := mux.Vars(r)["toolType"]
		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		tdHandler.DeleteAttachment(toolType, itemId).ServeHTTP(w, r)
	})
}
