package backup

import (
	"encoding/json"
	"github.com/andresbott/etna/internal/accounting"
	"github.com/glebarez/sqlite"
	"github.com/google/go-cmp/cmp"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandler_List(t *testing.T) {
	// Prepare temporary directories and files
	tempDir := t.TempDir()
	zipFile1 := filepath.Join(tempDir, "file1.zip")
	zipFile2 := filepath.Join(tempDir, "file2.ZIP") // test case-insensitive
	txtFile := filepath.Join(tempDir, "file3.txt")  // should be ignored

	// Create fake files
	os.WriteFile(zipFile1, []byte("dummy1"), 0644)
	os.WriteFile(zipFile2, []byte("dummy2"), 0644)
	os.WriteFile(txtFile, []byte("dummy3"), 0644)

	// Invalid directory for error case
	invalidDir := "/path/does/not/exist"

	tcs := []struct {
		name        string
		destination string
		expectCode  int
		expectErr   string
		want        listResponse
	}{
		{
			name:        "directory with zip files",
			destination: tempDir,
			expectCode:  http.StatusOK,
			want: listResponse{
				Files: []listPayload{
					{Id: hashFilename("file1.zip"), Filename: "file1.zip"},
					{Id: hashFilename("file2.ZIP"), Filename: "file2.ZIP"},
				},
			},
		},
		{
			name:        "empty directory",
			destination: t.TempDir(),
			expectCode:  http.StatusOK,
			want:        listResponse{Files: []listPayload{}},
		},
		{
			name:        "invalid directory",
			destination: invalidDir,
			expectCode:  http.StatusInternalServerError,
			expectErr:   "failed to read directory: ",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h := &Handler{Destination: tc.destination}

			recorder := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/files", nil)
			handler := h.List()
			handler.ServeHTTP(recorder, req)

			if tc.expectErr != "" {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("wrong status code: got %v want %v", status, tc.expectCode)
				}
				respText, err := io.ReadAll(recorder.Body)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(respText), tc.expectErr) {
					t.Errorf("unexpected error message: got \"%s\", want substring \"%s\"", string(respText), tc.expectErr)
				}
			} else {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("wrong status code: got %v want %v", status, tc.expectCode)
				}

				got := listResponse{}
				if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(got, tc.want); diff != "" {
					t.Errorf("unexpected response (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestHandler_Delete(t *testing.T) {
	// Prepare temporary directory and files
	tempDir := t.TempDir()
	zipFile := filepath.Join(tempDir, "file1.zip")
	nonZipFile := filepath.Join(tempDir, "file2.txt")

	os.WriteFile(zipFile, []byte("dummy"), 0644)
	os.WriteFile(nonZipFile, []byte("dummy"), 0644)

	validID := hashFilename("file1.zip")
	invalidID := "nonexistent"

	// Non-existent directory for error case
	invalidDir := "/path/does/not/exist"

	tcs := []struct {
		name        string
		destination string
		id          string
		expectCode  int
		expectErr   string
		expectResp  map[string]any
	}{
		{
			name:        "successful deletion",
			destination: tempDir,
			id:          validID,
			expectCode:  http.StatusOK,
			expectResp: map[string]any{
				"deleted": true,
				"id":      validID,
			},
		},
		{
			name:        "file not found",
			destination: tempDir,
			id:          invalidID,
			expectCode:  http.StatusNotFound,
			expectErr:   "file with id nonexistent not found",
		},
		{
			name:        "invalid directory",
			destination: invalidDir,
			id:          validID,
			expectCode:  http.StatusInternalServerError,
			expectErr:   "failed to read directory",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h := &Handler{Destination: tc.destination}

			recorder := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", "/files/"+tc.id, nil)
			handler := h.Delete(tc.id)
			handler.ServeHTTP(recorder, req)

			if tc.expectErr != "" {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("wrong status code: got %v want %v", status, tc.expectCode)
				}
				body, err := io.ReadAll(recorder.Body)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(body), tc.expectErr) {
					t.Errorf("unexpected error message: got %q want substring %q", string(body), tc.expectErr)
				}
			} else {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("wrong status code: got %v want %v", status, tc.expectCode)
				}

				got := map[string]any{}
				if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(got, tc.expectResp); diff != "" {
					t.Errorf("unexpected response (-want +got):\n%s", diff)
				}

				// Ensure the file was actually deleted
				if _, err := os.Stat(zipFile); tc.name == "successful deletion" && !os.IsNotExist(err) {
					t.Errorf("expected file to be deleted, but it still exists")
				}
			}
		})
	}
}

func TestHandler_CreateBackup(t *testing.T) {
	tcs := []struct {
		name       string
		store      func() *accounting.Store
		expectCode int
		expectErr  string
	}{
		{
			name: "successful backup",
			store: func() *accounting.Store {

				db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
					Logger: logger.Discard,
				})
				if err != nil {
					t.Fatalf("unable to connect to sqlite: %v", err)
				}

				store, err := accounting.NewStore(db)
				if err != nil {
					t.Fatalf("unable to connect to finance: %v", err)
				}
				return store
			},
			expectCode: http.StatusOK,
		},
		{
			name: "backup.ExportToFile returns error",
			store: func() *accounting.Store {
				return nil
			},
			expectCode: http.StatusInternalServerError,
			expectErr:  "failed to create backup: finance store was not initialized\n",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			h := &Handler{Destination: tempDir, Store: tc.store()}

			recorder := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/backup", nil)
			handler := h.CreateBackup()
			handler.ServeHTTP(recorder, req)

			if tc.expectErr != "" {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("wrong status code: got %v want %v", status, tc.expectCode)
				}
				body, err := io.ReadAll(recorder.Body)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(body), tc.expectErr) {
					t.Errorf("unexpected error message: got %q want substring %q", string(body), tc.expectErr)
				}
			} else {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("wrong status code: got %v want %v", status, tc.expectCode)
					respText, err := io.ReadAll(recorder.Body)
					if err != nil {
						t.Fatal(err)
					}
					t.Fatalf("error response: %s", string(respText))
				}

				// decode JSON response to get backup file path
				var resp map[string]string
				if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
					t.Fatal(err)
				}

				filePath := resp["file"]
				if filePath == "" {
					t.Fatalf("response does not contain file path")
				}

				// check that file exists
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Errorf("expected backup file %s to exist, but it does not", filePath)
				}
			}

		})
	}
}

func TestHandler_Download(t *testing.T) {
	tcs := []struct {
		name       string
		filename   string
		expectCode int
		expectErr  string
	}{
		{
			name:       "successful download",
			filename:   "backup.zip",
			expectCode: http.StatusOK,
		},
		{
			name:       "file not found",
			filename:   "nonexistent",
			expectCode: http.StatusNotFound,
			expectErr:  "file not found",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			fileName := "backup.zip"
			filePath := filepath.Join(tempDir, fileName)
			if err := os.WriteFile(filePath, []byte("test content"), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}
			fileID := hashFilename(tc.filename)

			h := &Handler{Destination: tempDir}

			recorder := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/download/"+fileID, nil)

			handler := h.Download(fileID)
			handler.ServeHTTP(recorder, req)

			if recorder.Code != tc.expectCode {
				t.Fatalf("unexpected status code: got %v, want %v", recorder.Code, tc.expectCode)
			}

			if tc.expectErr != "" {
				body, _ := io.ReadAll(recorder.Body)
				if !strings.Contains(string(body), tc.expectErr) {
					t.Errorf("unexpected error message: got %q, want substring %q", string(body), tc.expectErr)
				}
			} else {
				// Successful download: check headers and content
				contentDisposition := recorder.Header().Get("Content-Disposition")
				if !strings.Contains(contentDisposition, "attachment") {
					t.Errorf("missing or incorrect Content-Disposition header: %s", contentDisposition)
				}

				contentType := recorder.Header().Get("Content-Type")
				if contentType != "application/zip" {
					t.Errorf("expected Content-Type application/zip, got %s", contentType)
				}

				body, _ := io.ReadAll(recorder.Body)
				if string(body) != "test content" {
					t.Errorf("unexpected file content: %s", string(body))
				}
			}
		})
	}
}
