package backup

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/andresbott/etna/internal/accounting"
	"github.com/andresbott/etna/internal/backup"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Handler struct {
	Destination string
	Store       *accounting.Store
}

type listPayload struct {
	Id       string `json:"id"`
	Filename string `json:"filename"`
}
type listResponse struct {
	Files []listPayload `json:"files"`
}

func (h *Handler) List() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		absPath, err := filepath.Abs(h.Destination)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to resolve destination: %v", err), http.StatusInternalServerError)
			return
		}

		files, err := os.ReadDir(absPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to read directory: %v", err), http.StatusInternalServerError)
			return
		}

		payloads := []listPayload{} // init empty for proper response
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".zip") {
				payloads = append(payloads, listPayload{
					Id:       hashFilename(f.Name()),
					Filename: f.Name(),
				})
			}
		}

		resp := listResponse{Files: payloads}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, fmt.Sprintf("failed to encode response: %v", err), http.StatusInternalServerError)
			return
		}
	})
}

// hashFilename generates a short, deterministic hash for a given filename.
func hashFilename(name string) string {
	sum := sha1.Sum([]byte(name))
	return hex.EncodeToString(sum[:8]) // short 8-byte hash for readability
}

func (h *Handler) Delete(id string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		absPath, err := filepath.Abs(h.Destination)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to resolve destination: %v", err), http.StatusInternalServerError)
			return
		}

		files, err := os.ReadDir(absPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to read directory: %v", err), http.StatusInternalServerError)
			return
		}

		var targetFile string
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".zip") {
				if hashFilename(f.Name()) == id {
					targetFile = filepath.Join(absPath, f.Name())
					break
				}
			}
		}

		if targetFile == "" {
			http.Error(w, fmt.Sprintf("file with id %s not found", id), http.StatusNotFound)
			return
		}

		if err := os.Remove(targetFile); err != nil {
			http.Error(w, fmt.Sprintf("failed to delete file: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"deleted": true,
			"id":      id,
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, fmt.Sprintf("failed to encode response: %v", err), http.StatusInternalServerError)
			return
		}
	})
}

func (h *Handler) CreateBackup() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		absPath, err := filepath.Abs(h.Destination)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to resolve destination: %v", err), http.StatusInternalServerError)
			return
		}

		now := time.Now().Format("2006-01-02_15-04")
		backupFile := filepath.Join(absPath, fmt.Sprintf("backup-%s.zip", now))

		// Create the backup file
		err = backup.Export(r.Context(), h.Store, backupFile)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to create backup: %v", err), http.StatusInternalServerError)
			return
		}

		// Respond with the created file name
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]string{
			"file": backupFile,
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, fmt.Sprintf("failed to encode response: %v", err), http.StatusInternalServerError)
			return
		}
	})
}
