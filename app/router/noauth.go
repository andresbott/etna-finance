package router

import (
	"net/http"

	"github.com/go-bumbu/userauth/authenticator"
	"github.com/go-bumbu/userauth/handlers/sessionauth"
)

const noAuthHandlerName = "noAuth"

// noAuthHandler implements authenticator.AuthHandler. When auth is disabled,
// it injects the default user into request context and allows all access.
type noAuthHandler struct {
	defaultUser string
}

// NewNoAuthHandler returns an AuthHandler that always allows access and
// injects defaultUser into the request context for CtxGetUserData.
func NewNoAuthHandler(defaultUser string) authenticator.AuthHandler {
	return &noAuthHandler{defaultUser: defaultUser}
}

func (h *noAuthHandler) Name() string {
	return noAuthHandlerName
}

func (h *noAuthHandler) HandleAuth(w http.ResponseWriter, r *http.Request) (allowAccess, stopEvaluation bool) {
	data := sessionauth.SessionData{
		UserData: sessionauth.UserData{
			UserId:          h.defaultUser,
			IsAuthenticated: true,
		},
	}
	sessionauth.CtxSetUserData(r, data)
	return true, true
}
