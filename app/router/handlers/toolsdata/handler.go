package toolsdata

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/andresbott/etna/internal/toolsdata"
)

type Handler struct {
	Store *toolsdata.Store
}

type casePayload struct {
	ID                   uint            `json:"id"`
	ToolType             string          `json:"toolType"`
	Name                 string          `json:"name"`
	Description          string          `json:"description"`
	ExpectedAnnualReturn float64         `json:"expectedAnnualReturn"`
	Params               json.RawMessage `json:"params"`
	CreatedAt            string          `json:"createdAt"`
	UpdatedAt            string          `json:"updatedAt"`
}

func toPayload(cs toolsdata.CaseStudy) casePayload {
	return casePayload{
		ID:                   cs.ID,
		ToolType:             cs.ToolType,
		Name:                 cs.Name,
		Description:          cs.Description,
		ExpectedAnnualReturn: cs.ExpectedAnnualReturn,
		Params:               cs.Params,
		CreatedAt:            cs.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:            cs.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func (h *Handler) ListCases(toolType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		items, err := h.Store.List(r.Context(), toolType)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to list case studies: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		payloads := make([]casePayload, len(items))
		for i, cs := range items {
			payloads[i] = toPayload(cs)
		}
		respJSON, err := json.Marshal(payloads)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}

func (h *Handler) GetCase(toolType string, id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cs, err := h.Store.Get(r.Context(), toolType, id)
		if err != nil {
			if errors.Is(err, toolsdata.ErrCaseStudyNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to get case study: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		respJSON, err := json.Marshal(toPayload(cs))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}

func (h *Handler) CreateCase(toolType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}
		var payload casePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		cs := toolsdata.CaseStudy{
			ToolType:             toolType,
			Name:                 payload.Name,
			Description:          payload.Description,
			ExpectedAnnualReturn: payload.ExpectedAnnualReturn,
			Params:               payload.Params,
		}

		created, err := h.Store.Create(r.Context(), cs)
		if err != nil {
			var target toolsdata.ErrValidation
			if errors.As(err, &target) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, fmt.Sprintf("unable to create case study: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		respJSON, err := json.Marshal(toPayload(created))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}

func (h *Handler) UpdateCase(toolType string, id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}
		var payload casePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		cs := toolsdata.CaseStudy{
			Name:                 payload.Name,
			Description:          payload.Description,
			ExpectedAnnualReturn: payload.ExpectedAnnualReturn,
			Params:               payload.Params,
		}

		updated, err := h.Store.Update(r.Context(), toolType, id, cs)
		if err != nil {
			var target toolsdata.ErrValidation
			if errors.As(err, &target) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if errors.Is(err, toolsdata.ErrCaseStudyNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to update case study: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		respJSON, err := json.Marshal(toPayload(updated))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}

func (h *Handler) DeleteCase(toolType string, id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := h.Store.Delete(r.Context(), toolType, id)
		if err != nil {
			if errors.Is(err, toolsdata.ErrCaseStudyNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to delete case study: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
