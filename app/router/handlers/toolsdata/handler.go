package toolsdata

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/andresbott/etna/internal/filestore"
	"github.com/andresbott/etna/internal/toolsdata"
)

type Handler struct {
	Store     *toolsdata.Store
	FileStore *filestore.Store
}

type casePayload struct {
	ID                   uint            `json:"id"`
	ToolType             string          `json:"toolType"`
	Name                 string          `json:"name"`
	Description          string          `json:"description"`
	ExpectedAnnualReturn float64         `json:"expectedAnnualReturn"`
	Params               json.RawMessage `json:"params"`
	AttachmentID         *uint           `json:"attachmentId,omitempty"`
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
		AttachmentID:         cs.AttachmentID,
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
		// Clean up attachment if present
		if h.FileStore != nil {
			cs, err := h.Store.Get(r.Context(), toolType, id)
			if err == nil && cs.AttachmentID != nil {
				_ = h.FileStore.Delete(r.Context(), *cs.AttachmentID)
			}
		}

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

type attachmentPayload struct {
	Id           uint   `json:"id"`
	OriginalName string `json:"originalName"`
	MimeType     string `json:"mimeType"`
	FileSize     int64  `json:"fileSize"`
}

func (h *Handler) UploadAttachment(toolType string, id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.FileStore == nil {
			http.Error(w, "attachments not configured", http.StatusServiceUnavailable)
			return
		}

		cs, err := h.Store.Get(r.Context(), toolType, id)
		if err != nil {
			if errors.Is(err, toolsdata.ErrCaseStudyNotFound) {
				http.Error(w, "case study not found", http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("unable to get case study: %v", err), http.StatusInternalServerError)
			}
			return
		}

		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, fmt.Sprintf("unable to parse multipart form: %v", err), http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read file from form: %v", err), http.StatusBadRequest)
			return
		}
		defer func() { _ = file.Close() }()

		// Delete old attachment if present
		if cs.AttachmentID != nil {
			_ = h.FileStore.Delete(r.Context(), *cs.AttachmentID)
		}

		attID, err := h.FileStore.Save(r.Context(), time.Now(), file, header)
		if err != nil {
			if errors.Is(err, filestore.ErrMimeNotAllowed) {
				http.Error(w, "file type not allowed", http.StatusBadRequest)
				return
			}
			if errors.Is(err, filestore.ErrTooLarge) {
				http.Error(w, "file too large", http.StatusBadRequest)
				return
			}
			http.Error(w, fmt.Sprintf("unable to save file: %v", err), http.StatusInternalServerError)
			return
		}

		if err := h.Store.SetAttachmentID(r.Context(), toolType, id, &attID); err != nil {
			_ = h.FileStore.Delete(r.Context(), attID)
			http.Error(w, fmt.Sprintf("failed to update case study: %v", err), http.StatusInternalServerError)
			return
		}

		att, err := h.FileStore.Get(r.Context(), attID)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to read attachment metadata: %v", err), http.StatusInternalServerError)
			return
		}

		respJSON, err := json.Marshal(attachmentPayload{
			Id:           att.Id,
			OriginalName: att.OriginalName,
			MimeType:     att.MimeType,
			FileSize:     att.FileSize,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}

func (h *Handler) GetAttachment(toolType string, id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.FileStore == nil {
			http.Error(w, "attachments not configured", http.StatusServiceUnavailable)
			return
		}

		cs, err := h.Store.Get(r.Context(), toolType, id)
		if err != nil {
			if errors.Is(err, toolsdata.ErrCaseStudyNotFound) {
				http.Error(w, "case study not found", http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("unable to get case study: %v", err), http.StatusInternalServerError)
			}
			return
		}

		if cs.AttachmentID == nil {
			http.Error(w, "no attachment", http.StatusNotFound)
			return
		}

		att, err := h.FileStore.Get(r.Context(), *cs.AttachmentID)
		if err != nil {
			if errors.Is(err, filestore.ErrNotFound) {
				http.Error(w, "attachment file not found", http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("unable to get attachment: %v", err), http.StatusInternalServerError)
			}
			return
		}

		filePath, err := h.FileStore.GetFilePath(r.Context(), *cs.AttachmentID)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to resolve file path: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", att.MimeType)
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", att.OriginalName))
		http.ServeFile(w, r, filePath)
	})
}

func (h *Handler) DeleteAttachment(toolType string, id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.FileStore == nil {
			http.Error(w, "attachments not configured", http.StatusServiceUnavailable)
			return
		}

		cs, err := h.Store.Get(r.Context(), toolType, id)
		if err != nil {
			if errors.Is(err, toolsdata.ErrCaseStudyNotFound) {
				http.Error(w, "case study not found", http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("unable to get case study: %v", err), http.StatusInternalServerError)
			}
			return
		}

		if cs.AttachmentID == nil {
			http.Error(w, "no attachment", http.StatusNotFound)
			return
		}

		if err := h.FileStore.Delete(r.Context(), *cs.AttachmentID); err != nil && !errors.Is(err, filestore.ErrNotFound) {
			http.Error(w, fmt.Sprintf("unable to delete attachment: %v", err), http.StatusInternalServerError)
			return
		}

		if err := h.Store.SetAttachmentID(r.Context(), toolType, id, nil); err != nil {
			http.Error(w, fmt.Sprintf("failed to update case study: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}
