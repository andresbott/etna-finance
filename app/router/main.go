package router

import (
	_ "embed"
	"fmt"
	handlrs "github.com/andresbott/etna/app/router/handlers"
	"github.com/andresbott/etna/app/router/handlers/backup"
	"github.com/andresbott/etna/app/spa"
	"github.com/andresbott/etna/internal/accounting"
	"github.com/go-bumbu/http/middleware"
	"github.com/go-bumbu/userauth"
	"github.com/go-bumbu/userauth/handlers/sessionauth"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
	"log/slog"
	"net/http"
)

type Cfg struct {
	Db                *gorm.DB
	SessionAuth       *sessionauth.Manager
	UserMngr          userauth.LoginHandler
	Logger            *slog.Logger
	BackupDestination string
	ProductionMode    bool
}

// MainAppHandler is the entrypoint http handler for the whole application
type MainAppHandler struct {
	router            *mux.Router
	db                *gorm.DB
	finStore          *accounting.Store
	SessionAuth       *sessionauth.Manager
	userMngr          userauth.LoginHandler
	logger            *slog.Logger
	backupDestination string
	productionMode    bool
}

func (h *MainAppHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

var financeStore *accounting.Store

func New(cfg Cfg) (*MainAppHandler, error) {
	r := mux.NewRouter()
	app := MainAppHandler{
		router:            r,
		db:                cfg.Db,
		SessionAuth:       cfg.SessionAuth,
		userMngr:          cfg.UserMngr,
		logger:            cfg.Logger,
		backupDestination: cfg.BackupDestination,
		productionMode:    cfg.ProductionMode,
	}

	fineStore, err := accounting.NewStore(cfg.Db)
	if err != nil {
		return nil, fmt.Errorf("unable to create accounting Store :%v", err)
	}
	app.finStore = fineStore

	prodMid := middleware.New(middleware.Cfg{
		JsonErrors:  false,
		GenericErrs: cfg.ProductionMode,
		Logger:      cfg.Logger,
		Histogram:   middleware.NewPromHistogram("", nil, nil),
	})
	r.Use(prodMid.Middleware)

	app.attachUserAuth(app.router.PathPrefix("/auth").Subrouter())
	// backup/restore
	app.attachBackup(app.router.PathPrefix("/backup").Subrouter())

	// add a handler for /api/v0, this includes authentication on tasks
	err = app.attachApiV0(app.router.PathPrefix("/api/v0").Subrouter())
	if err != nil {
		return nil, err
	}

	// add the spa to path /
	err = app.attachSpa(app.router.PathPrefix("/").Subrouter(), "/")
	if err != nil {
		return nil, err
	}

	return &app, nil
}

func (h *MainAppHandler) attachSpa(r *mux.Router, path string) error {
	// if you want to serve the spa from the root, pass "/" to the spa handler and the path prefix
	// note that the SPA base and route needs to be adjusted accordingly
	spaHandler, err := spa.App(path)
	if err != nil {
		return err
	}
	r.Methods(http.MethodGet).PathPrefix(path).Handler(spaHandler)
	return nil
}

func (h *MainAppHandler) attachUserAuth(r *mux.Router) {

	//  LOGIN
	r.Path("/login").Methods(http.MethodPost).Handler(h.SessionAuth.JsonAuthHandler(h.userMngr))
	r.Path("/login").Methods(http.MethodOptions).Handler(
		http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {}))
	// TODO add a basic form login here to the GET method
	r.Path("/login").HandlerFunc(StatusErr(http.StatusMethodNotAllowed))

	// LOGOUT
	r.Path("/logout").Handler(h.SessionAuth.LogoutHandler("/"))

	// STATUS
	r.Path("/status").Methods(http.MethodGet).Handler(handlrs.UserStatusHandler(h.SessionAuth))
	r.Path("/status").HandlerFunc(StatusErr(http.StatusMethodNotAllowed))

	// OPTIONS
	//r.Path("/user/options").Methods(http.MethodGet).Handler(handlers.StatusErr(http.StatusNotImplemented))
	r.Path("/user/options").HandlerFunc(StatusErr(http.StatusMethodNotAllowed))
}

func (h *MainAppHandler) attachBackup(r *mux.Router) {

	backupHndl := backup.Handler{
		Destination: h.backupDestination,
		Store:       h.finStore,
	}
	r.Path("/list").Methods(http.MethodGet).Handler(backupHndl.List())
	r.Path("/create").Methods(http.MethodPut).Handler(backupHndl.CreateBackup())
	r.Path("/delete/{ID}").Methods(http.MethodPut).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		backupHndl.Delete(itemId)
	})

	r.Path("/download/{ID}").Methods(http.MethodPut).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "not implemented", http.StatusNotImplemented)
		return
	})

}

func StatusErr(status int) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(status), status)
	}
}
func StatusErrText(status int, text string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, text, status)
	}
}
