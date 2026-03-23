package finance

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/andresbott/etna/internal/accounting"
	"github.com/andresbott/etna/internal/filestore"
)

type attachmentPayload struct {
	Id           uint   `json:"id"`
	OriginalName string `json:"originalName"`
	MimeType     string `json:"mimeType"`
	FileSize     int64  `json:"fileSize"`
}

// UploadAttachment handles POST /fin/entries/{id}/attachment
func (h *Handler) UploadAttachment(txId uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.FileStore == nil {
			http.Error(w, "attachments not configured", http.StatusServiceUnavailable)
			return
		}

		tx, err := h.Store.GetTransaction(r.Context(), txId)
		if err != nil {
			if errors.Is(err, accounting.ErrTransactionNotFound) {
				http.Error(w, "transaction not found", http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("unable to get transaction: %v", err), http.StatusInternalServerError)
			}
			return
		}

		// Parse multipart form (32 MB max memory)
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

		// If existing attachment, delete old one first
		if attID := transactionAttachmentID(tx); attID != nil {
			_ = h.FileStore.Delete(r.Context(), *attID)
		}

		date := transactionDate(tx)
		attID, err := h.FileStore.Save(r.Context(), date, file, header)
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

		// Update the transaction's AttachmentID
		if err := h.Store.SetAttachmentID(r.Context(), txId, &attID); err != nil {
			// Cleanup: delete the just-saved file
			_ = h.FileStore.Delete(r.Context(), attID)
			http.Error(w, fmt.Sprintf("failed to update transaction: %v", err), http.StatusInternalServerError)
			return
		}

		att, err := h.FileStore.Get(r.Context(), attID)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to read attachment metadata: %v", err), http.StatusInternalServerError)
			return
		}

		payload := attachmentPayload{
			Id:           att.Id,
			OriginalName: att.OriginalName,
			MimeType:     att.MimeType,
			FileSize:     att.FileSize,
		}
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

// GetAttachment handles GET /fin/entries/{id}/attachment
func (h *Handler) GetAttachment(txId uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.FileStore == nil {
			http.Error(w, "attachments not configured", http.StatusServiceUnavailable)
			return
		}

		tx, err := h.Store.GetTransaction(r.Context(), txId)
		if err != nil {
			if errors.Is(err, accounting.ErrTransactionNotFound) {
				http.Error(w, "transaction not found", http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("unable to get transaction: %v", err), http.StatusInternalServerError)
			}
			return
		}

		attID := transactionAttachmentID(tx)
		if attID == nil {
			http.Error(w, "no attachment", http.StatusNotFound)
			return
		}

		att, err := h.FileStore.Get(r.Context(), *attID)
		if err != nil {
			if errors.Is(err, filestore.ErrNotFound) {
				http.Error(w, "attachment file not found", http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("unable to get attachment: %v", err), http.StatusInternalServerError)
			}
			return
		}

		filePath, err := h.FileStore.GetFilePath(r.Context(), *attID)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to resolve file path: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", att.MimeType)
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", att.OriginalName))
		http.ServeFile(w, r, filePath)
	})
}

// DeleteAttachment handles DELETE /fin/entries/{id}/attachment
func (h *Handler) DeleteAttachment(txId uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.FileStore == nil {
			http.Error(w, "attachments not configured", http.StatusServiceUnavailable)
			return
		}

		tx, err := h.Store.GetTransaction(r.Context(), txId)
		if err != nil {
			if errors.Is(err, accounting.ErrTransactionNotFound) {
				http.Error(w, "transaction not found", http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("unable to get transaction: %v", err), http.StatusInternalServerError)
			}
			return
		}

		attID := transactionAttachmentID(tx)
		if attID == nil {
			http.Error(w, "no attachment", http.StatusNotFound)
			return
		}

		if err := h.FileStore.Delete(r.Context(), *attID); err != nil && !errors.Is(err, filestore.ErrNotFound) {
			http.Error(w, fmt.Sprintf("unable to delete attachment: %v", err), http.StatusInternalServerError)
			return
		}

		if err := h.Store.SetAttachmentID(r.Context(), txId, nil); err != nil {
			http.Error(w, fmt.Sprintf("failed to update transaction: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}

// transactionDate extracts the Date field from any transaction type.
func transactionDate(tx accounting.Transaction) time.Time {
	switch t := tx.(type) {
	case accounting.Income:
		return t.Date
	case accounting.Expense:
		return t.Date
	case accounting.Transfer:
		return t.Date
	case accounting.StockBuy:
		return t.Date
	case accounting.StockSell:
		return t.Date
	case accounting.StockGrant:
		return t.Date
	case accounting.StockTransfer:
		return t.Date
	case accounting.StockVest:
		return t.Date
	case accounting.StockForfeit:
		return t.Date
	case accounting.BalanceStatus:
		return t.Date
	default:
		return time.Now()
	}
}

// transactionAttachmentID extracts the AttachmentID field from any transaction type.
func transactionAttachmentID(tx accounting.Transaction) *uint {
	switch t := tx.(type) {
	case accounting.Income:
		return t.AttachmentID
	case accounting.Expense:
		return t.AttachmentID
	case accounting.Transfer:
		return t.AttachmentID
	case accounting.StockBuy:
		return t.AttachmentID
	case accounting.StockSell:
		return t.AttachmentID
	case accounting.StockGrant:
		return t.AttachmentID
	case accounting.StockTransfer:
		return t.AttachmentID
	case accounting.StockVest:
		return t.AttachmentID
	case accounting.StockForfeit:
		return t.AttachmentID
	case accounting.BalanceStatus:
		return t.AttachmentID
	default:
		return nil
	}
}
