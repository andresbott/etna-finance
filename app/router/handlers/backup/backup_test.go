package backup

import (
	"encoding/json"
	"github.com/andresbott/etna/internal/accounting"
	"github.com/glebarez/sqlite"
	"github.com/google/go-cmp/cmp"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"io"
	"mime/multipart"
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
					{Id: hashFilename("file1.zip"), Filename: "file1.zip", Size: 6},
					{Id: hashFilename("file2.ZIP"), Filename: "file2.ZIP", Size: 6},
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
	}{
		{
			name:        "successful deletion",
			destination: tempDir,
			id:          validID,
			expectCode:  http.StatusOK,
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

			if status := recorder.Code; status != tc.expectCode {
				t.Errorf("wrong status code: got %v want %v", status, tc.expectCode)
			}

			if tc.expectErr != "" {
				body, err := io.ReadAll(recorder.Body)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(body), tc.expectErr) {
					t.Errorf("unexpected error message: got %q want substring %q", string(body), tc.expectErr)
				}
			} else {
				// Ensure the file was actually deleted on successful deletion
				if tc.name == "successful deletion" {
					if _, err := os.Stat(zipFile); !os.IsNotExist(err) {
						t.Errorf("expected file to be deleted, but it still exists")
					}
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

func TestHandler_RestoreUpload(t *testing.T) {
	tcs := []struct {
		name       string
		setupStore func() *accounting.Store
		setupFile  func(t *testing.T) (filename string, content []byte)
		expectCode int
		expectErr  string
	}{
		{
			name: "successful restore",
			setupStore: func() *accounting.Store {
				db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
					Logger: logger.Discard,
				})
				if err != nil {
					t.Fatalf("unable to connect to sqlite: %v", err)
				}
				store, err := accounting.NewStore(db)
				if err != nil {
					t.Fatalf("unable to create store: %v", err)
				}
				return store
			},
			setupFile: func(t *testing.T) (string, []byte) {
				// Use the test backup from internal/backup/testdata
				// Try multiple paths to find the test file
				possiblePaths := []string{
					"../../../internal/backup/testdata/backup-v1.zip",
					"../../../../internal/backup/testdata/backup-v1.zip",
					"internal/backup/testdata/backup-v1.zip",
				}
				var content []byte
				var err error
				for _, path := range possiblePaths {
					content, err = os.ReadFile(path)
					if err == nil {
						break
					}
				}
				if err != nil {
					t.Skipf("skipping test: test backup file not found")
				}
				return "backup-v1.zip", content
			},
			expectCode: http.StatusOK,
		},
		{
			name: "non-zip file rejected",
			setupStore: func() *accounting.Store {
				db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
					Logger: logger.Discard,
				})
				if err != nil {
					t.Fatalf("unable to connect to sqlite: %v", err)
				}
				store, err := accounting.NewStore(db)
				if err != nil {
					t.Fatalf("unable to create store: %v", err)
				}
				return store
			},
			setupFile: func(t *testing.T) (string, []byte) {
				return "backup.txt", []byte("not a zip file")
			},
			expectCode: http.StatusBadRequest,
			expectErr:  "only .zip files are allowed",
		},
		{
			name: "invalid backup file",
			setupStore: func() *accounting.Store {
				db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
					Logger: logger.Discard,
				})
				if err != nil {
					t.Fatalf("unable to connect to sqlite: %v", err)
				}
				store, err := accounting.NewStore(db)
				if err != nil {
					t.Fatalf("unable to create store: %v", err)
				}
				return store
			},
			setupFile: func(t *testing.T) (string, []byte) {
				return "invalid.zip", []byte("not a valid zip")
			},
			expectCode: http.StatusInternalServerError,
			expectErr:  "failed to restore backup",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			h := &Handler{
				Destination: tempDir,
				Store:       tc.setupStore(),
			}

			filename, content := tc.setupFile(t)

			// Create multipart form
			body := &strings.Builder{}
			writer := createMultipartForm(t, body, filename, content)
			contentType := writer.FormDataContentType()

			recorder := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/restore", strings.NewReader(body.String()))
			req.Header.Set("Content-Type", contentType)

			handler := h.RestoreUpload()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != tc.expectCode {
				t.Errorf("wrong status code: got %v want %v", recorder.Code, tc.expectCode)
				respBody, _ := io.ReadAll(recorder.Body)
				t.Logf("response body: %s", string(respBody))
			}

			if tc.expectErr != "" {
				body, err := io.ReadAll(recorder.Body)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(body), tc.expectErr) {
					t.Errorf("unexpected error message: got %q want substring %q", string(body), tc.expectErr)
				}
			} else {
				// On success, verify that a restore file was created
				files, err := os.ReadDir(tempDir)
				if err != nil {
					t.Fatal(err)
				}

				found := false
				for _, f := range files {
					if strings.HasPrefix(f.Name(), "restore-") && strings.HasSuffix(f.Name(), ".zip") {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("uploaded restore file not found in destination")
				}
			}
		})
	}
}

func TestHandler_RestoreFromExisting(t *testing.T) {
	tcs := []struct {
		name       string
		setupStore func() *accounting.Store
		setupFile  func(t *testing.T, dir string) (filename, id string)
		expectCode int
		expectErr  string
	}{
		{
			name: "successful restore from existing",
			setupStore: func() *accounting.Store {
				db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
					Logger: logger.Discard,
				})
				if err != nil {
					t.Fatalf("unable to connect to sqlite: %v", err)
				}
				store, err := accounting.NewStore(db)
				if err != nil {
					t.Fatalf("unable to create store: %v", err)
				}
				return store
			},
			setupFile: func(t *testing.T, dir string) (string, string) {
				// Try multiple paths to find the test file
				possiblePaths := []string{
					"../../../internal/backup/testdata/backup-v1.zip",
					"../../../../internal/backup/testdata/backup-v1.zip",
					"internal/backup/testdata/backup-v1.zip",
				}
				var content []byte
				var err error
				for _, path := range possiblePaths {
					content, err = os.ReadFile(path)
					if err == nil {
						break
					}
				}
				if err != nil {
					t.Skipf("skipping test: test backup file not found")
				}
				filename := "backup-existing.zip"
				filePath := filepath.Join(dir, filename)
				if err := os.WriteFile(filePath, content, 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				return filename, hashFilename(filename)
			},
			expectCode: http.StatusOK,
		},
		{
			name: "backup file not found",
			setupStore: func() *accounting.Store {
				db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
					Logger: logger.Discard,
				})
				if err != nil {
					t.Fatalf("unable to connect to sqlite: %v", err)
				}
				store, err := accounting.NewStore(db)
				if err != nil {
					t.Fatalf("unable to create store: %v", err)
				}
				return store
			},
			setupFile: func(t *testing.T, dir string) (string, string) {
				return "", "nonexistent-id"
			},
			expectCode: http.StatusNotFound,
			expectErr:  "backup file with id nonexistent-id not found",
		},
		{
			name: "invalid backup content",
			setupStore: func() *accounting.Store {
				db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
					Logger: logger.Discard,
				})
				if err != nil {
					t.Fatalf("unable to connect to sqlite: %v", err)
				}
				store, err := accounting.NewStore(db)
				if err != nil {
					t.Fatalf("unable to create store: %v", err)
				}
				return store
			},
			setupFile: func(t *testing.T, dir string) (string, string) {
				filename := "corrupt-backup.zip"
				filePath := filepath.Join(dir, filename)
				if err := os.WriteFile(filePath, []byte("not a valid zip"), 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				return filename, hashFilename(filename)
			},
			expectCode: http.StatusInternalServerError,
			expectErr:  "failed to restore backup",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			h := &Handler{
				Destination: tempDir,
				Store:       tc.setupStore(),
			}

			_, id := tc.setupFile(t, tempDir)

			recorder := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/restore/"+id, nil)

			handler := h.RestoreFromExisting(id)
			handler.ServeHTTP(recorder, req)

			if recorder.Code != tc.expectCode {
				t.Errorf("wrong status code: got %v want %v", recorder.Code, tc.expectCode)
				respBody, _ := io.ReadAll(recorder.Body)
				t.Logf("response body: %s", string(respBody))
			}

			if tc.expectErr != "" {
				body, err := io.ReadAll(recorder.Body)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(body), tc.expectErr) {
					t.Errorf("unexpected error message: got %q want substring %q", string(body), tc.expectErr)
				}
			}
		})
	}
}

func TestGenerateRandomFilename(t *testing.T) {
	// Test that function generates valid filenames
	for i := 0; i < 10; i++ {
		filename, err := generateRandomFilename()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.HasPrefix(filename, "restore-") {
			t.Errorf("filename should start with 'restore-', got %s", filename)
		}

		if !strings.HasSuffix(filename, ".zip") {
			t.Errorf("filename should end with '.zip', got %s", filename)
		}

		// Check format: restore-YYYY-MM-DD_HH-MM-SS-XXXXXXXXXXXXXXXX.zip
		parts := strings.Split(filename, "-")
		if len(parts) < 6 {
			t.Errorf("filename should have correct format, got %s", filename)
		}
	}

	// Test uniqueness
	names := make(map[string]bool)
	for i := 0; i < 100; i++ {
		filename, err := generateRandomFilename()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if names[filename] {
			t.Errorf("generated duplicate filename: %s", filename)
		}
		names[filename] = true
	}
}

// Helper function to create multipart form data
func createMultipartForm(t *testing.T, body io.Writer, filename string, content []byte) *multipart.Writer {
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}

	if _, err := part.Write(content); err != nil {
		t.Fatalf("failed to write file content: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	return writer
}
