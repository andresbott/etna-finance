package csvimport

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/andresbott/etna/internal/csvimport"
)

type ProfileHandler struct {
	Store *csvimport.Store
}

type profilePayload struct {
	ID                uint   `json:"id"`
	Name              string `json:"name"`
	CsvSeparator      string `json:"csvSeparator"`
	SkipRows          int    `json:"skipRows"`
	DateColumn        string `json:"dateColumn"`
	DateFormat        string `json:"dateFormat"`
	DescriptionColumn string `json:"descriptionColumn"`
	AmountColumn      string `json:"amountColumn"`
}

var validationErr = csvimport.ErrValidation("")

func (h *ProfileHandler) ListProfiles() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		profiles, err := h.Store.ListProfiles(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to list profiles: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		items := make([]profilePayload, len(profiles))
		for i, p := range profiles {
			items[i] = profilePayload{
				ID:                p.ID,
				Name:              p.Name,
				CsvSeparator:      p.CsvSeparator,
				SkipRows:          p.SkipRows,
				DateColumn:        p.DateColumn,
				DateFormat:        p.DateFormat,
				DescriptionColumn: p.DescriptionColumn,
				AmountColumn:      p.AmountColumn,
			}
		}

		respJSON, err := json.Marshal(items)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}

func (h *ProfileHandler) CreateProfile() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		var payload profilePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		profile := csvimport.ImportProfile{
			Name:              payload.Name,
			CsvSeparator:      payload.CsvSeparator,
			SkipRows:          payload.SkipRows,
			DateColumn:        payload.DateColumn,
			DateFormat:        payload.DateFormat,
			DescriptionColumn: payload.DescriptionColumn,
			AmountColumn:      payload.AmountColumn,
		}

		id, err := h.Store.CreateProfile(r.Context(), profile)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, fmt.Sprintf("unable to create profile: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		payload.ID = id
		respJSON, err := json.Marshal(payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}

func (h *ProfileHandler) UpdateProfile(id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		var payload profilePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		profile := csvimport.ImportProfile{
			Name:              payload.Name,
			CsvSeparator:      payload.CsvSeparator,
			SkipRows:          payload.SkipRows,
			DateColumn:        payload.DateColumn,
			DateFormat:        payload.DateFormat,
			DescriptionColumn: payload.DescriptionColumn,
			AmountColumn:      payload.AmountColumn,
		}

		err := h.Store.UpdateProfile(r.Context(), id, profile)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if errors.Is(err, csvimport.ErrProfileNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to update profile: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func (h *ProfileHandler) DeleteProfile(id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := h.Store.DeleteProfile(r.Context(), id)
		if err != nil {
			if errors.Is(err, csvimport.ErrProfileNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to delete profile: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
