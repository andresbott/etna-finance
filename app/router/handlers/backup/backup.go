package backup

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/andresbott/etna/internal/accounting"
	"github.com/andresbott/etna/internal/backup"
	"github.com/davecgh/go-spew/spew"
	"io"
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
	Size     int64  `json:"size"`
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
				fullPath := filepath.Join(absPath, f.Name())
				finfo, err := os.Stat(fullPath)
				if err != nil {
					http.Error(w, fmt.Sprintf("failed to read file: %v", err), http.StatusInternalServerError)
					return
				}

				payloads = append(payloads, listPayload{
					Id:       hashFilename(f.Name()),
					Filename: f.Name(),
					Size:     finfo.Size(),
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

func (h *Handler) Download(id string) http.Handler {
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
			if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".zip") && hashFilename(f.Name()) == id {
				targetFile = f.Name()
				break
			}
		}
		if targetFile == "" {
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}

		filePath := filepath.Join(absPath, targetFile)

		// Set headers for download
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", targetFile))
		w.Header().Set("Content-Type", "application/zip")

		// Serve the file
		http.ServeFile(w, r, filePath)
	})
}

func (h *Handler) Upload() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Limit the upload size (optional, e.g., 100MB)
		r.Body = http.MaxBytesReader(w, r.Body, 100<<20) // 100 MB

		if err := r.ParseMultipartForm(100 << 20); err != nil {
			http.Error(w, fmt.Sprintf("failed to parse form: %v", err), http.StatusBadRequest)
			return
		}

		file, handler, err := r.FormFile("file") // expects form field "file"
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to get file: %v", err), http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Only allow .zip files
		if !strings.HasSuffix(strings.ToLower(handler.Filename), ".zip") {
			http.Error(w, "only .zip files are allowed", http.StatusBadRequest)
			return
		}

		absPath, err := filepath.Abs(h.Destination)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to resolve destination: %v", err), http.StatusInternalServerError)
			return
		}

		dstPath := filepath.Join(absPath, handler.Filename)
		dstFile, err := os.Create(dstPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to create file: %v", err), http.StatusInternalServerError)
			return
		}
		defer dstFile.Close()

		// Copy uploaded file to destination
		if _, err := io.Copy(dstFile, file); err != nil {
			http.Error(w, fmt.Sprintf("failed to save file: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, "file uploaded successfully: %s", handler.Filename)
	})
}

func (h *Handler) Delete(id string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		absPath, err := filepath.Abs(h.Destination)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to resolve destination: %v", err), http.StatusInternalServerError)
			return
		}
		spew.Dump(absPath)

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

		spew.Dump(targetFile)

		if targetFile == "" {
			http.Error(w, fmt.Sprintf("file with id %s not found", id), http.StatusNotFound)
			return
		}

		if err := os.Remove(targetFile); err != nil {
			http.Error(w, fmt.Sprintf("failed to delete file: %v", err), http.StatusInternalServerError)
			return
		}

		spew.Dump(targetFile)

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
		err = backup.ExportToFile(r.Context(), h.Store, backupFile)
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
