package handlrs

import (
	"encoding/json"
	"net/http"

	"github.com/go-bumbu/userauth/handlers/sessionauth"
)

type userStatus struct {
	User     string `json:"username"`
	LoggedIn bool   `json:"logged-in"`
}

func UserStatusHandler(session *sessionauth.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		data, err := session.GetSessData(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonData := userStatus{
			User:     data.UserId,
			LoggedIn: data.IsAuthenticated,
		}
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(jsonData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

	})
}

// AuthStatusHandler returns an HTTP handler for GET /auth/status. When authDisabled is true,
// it always returns {username: defaultUser, logged-in: true} without checking the session.
func AuthStatusHandler(session *sessionauth.Manager, authDisabled bool, defaultUser string) http.Handler {
	if authDisabled {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			jsonData := userStatus{
				User:     defaultUser,
				LoggedIn: true,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(jsonData)
		})
	}
	return UserStatusHandler(session)
}
