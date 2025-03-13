package router

import (
	_ "embed"
	handlrs "github.com/andresbott/etna/app/router/handlers"
	"github.com/andresbott/etna/app/spa"
	"github.com/go-bumbu/http/middleware"
	"github.com/go-bumbu/userauth"
	"github.com/go-bumbu/userauth/handlers/sessionauth"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
	"log/slog"
	"net/http"
)

type Cfg struct {
	Db             *gorm.DB
	SessionAuth    *sessionauth.Manager
	UserMngr       userauth.LoginHandler
	Logger         *slog.Logger
	ProductionMode bool
}

// MainAppHandler is the entrypoint http handler for the whole application
type MainAppHandler struct {
	router         *mux.Router
	db             *gorm.DB
	SessionAuth    *sessionauth.Manager
	userMngr       userauth.LoginHandler
	logger         *slog.Logger
	productionMode bool
}

func (h *MainAppHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func New(cfg Cfg) (*MainAppHandler, error) {
	r := mux.NewRouter()
	app := MainAppHandler{
		router:         r,
		db:             cfg.Db,
		SessionAuth:    cfg.SessionAuth,
		userMngr:       cfg.UserMngr,
		logger:         cfg.Logger,
		productionMode: cfg.ProductionMode,
	}

	prodMid := middleware.New(middleware.Cfg{
		JsonErrors:  false,
		GenericErrs: cfg.ProductionMode,
		Logger:      cfg.Logger,
		Histogram:   middleware.NewPromHistogram("", nil, nil),
	})
	r.Use(prodMid.Middleware)

	app.attachUserAuth(app.router.PathPrefix("/auth").Subrouter())

	// add a handler for /api/v0, this includes authentication on tasks
	//app.attachApiV0(app.router.PathPrefix("/api/v0").Subrouter())

	// add the spa to path /
	err := app.attachSpa(app.router.PathPrefix("/").Subrouter(), "/")
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

func StatusErr(status int) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(status), status)
	}
}
