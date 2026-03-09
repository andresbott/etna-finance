# Transaction File Attachments Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Allow attaching a single file (image or PDF) to any transaction, stored on the server filesystem with DB metadata, viewable from the transaction list.

**Architecture:** Isolated `internal/filestore` package owns file storage + `db_attachments` table. The `dbTransaction` in accounting gets a nullable `AttachmentID *uint` FK. Three new API endpoints handle upload/serve/delete. Frontend adds a paperclip icon in transaction rows and a file input in dialogs.

**Tech Stack:** Go (GORM/SQLite), gorilla/mux, Vue 3, PrimeVue, TanStack Vue Query, Axios

**Design doc:** `docs/plans/2026-03-09-transaction-attachments-design.md`

---

### Task 1: FileStore Package — Core Store and DB Model

**Files:**
- Create: `internal/filestore/filestore.go`
- Create: `internal/filestore/filestore_test.go`

**Step 1: Write the test file with tests for Save, Get, GetFilePath, Delete**

```go
package filestore

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-bumbu/testdbs"
)

// createTestFile creates a minimal multipart file for testing.
func createTestFile(t *testing.T, filename string, content []byte, contentType string) (multipart.File, *multipart.FileHeader) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	h.Set("Content-Type", contentType)
	part, err := writer.CreatePart(h)
	if err != nil {
		t.Fatal(err)
	}
	part.Write(content)
	writer.Close()

	reader := multipart.NewReader(&buf, writer.Boundary())
	form, err := reader.ReadForm(int64(len(content) + 1024))
	if err != nil {
		t.Fatal(err)
	}
	f, err := form.File["file"][0].Open()
	if err != nil {
		t.Fatal(err)
	}
	return f, form.File["file"][0]
}

func TestStore_Save(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			baseDir := t.TempDir()
			store, err := New(db.ConnDbName("TestSave"), baseDir, 10*1024*1024)
			if err != nil {
				t.Fatal(err)
			}

			f, header := createTestFile(t, "receipt.pdf", []byte("%PDF-1.4 test"), "application/pdf")
			defer f.Close()

			date := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)
			id, err := store.Save(context.Background(), date, f, header)
			if err != nil {
				t.Fatalf("Save failed: %v", err)
			}
			if id == 0 {
				t.Fatal("expected non-zero attachment ID")
			}

			// Verify file exists on disk
			att, err := store.Get(context.Background(), id)
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}
			if att.OriginalName != "receipt.pdf" {
				t.Errorf("expected original name 'receipt.pdf', got %q", att.OriginalName)
			}
			if att.MimeType != "application/pdf" {
				t.Errorf("expected mime type 'application/pdf', got %q", att.MimeType)
			}
			if !strings.HasPrefix(att.StoragePath, "2026/03/") {
				t.Errorf("expected storage path to start with '2026/03/', got %q", att.StoragePath)
			}

			fullPath, err := store.GetFilePath(context.Background(), id)
			if err != nil {
				t.Fatalf("GetFilePath failed: %v", err)
			}
			if _, err := os.Stat(fullPath); err != nil {
				t.Fatalf("file does not exist at %s: %v", fullPath, err)
			}
		})
	}
}

func TestStore_Save_InvalidMimeType(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			baseDir := t.TempDir()
			store, err := New(db.ConnDbName("TestSaveInvalid"), baseDir, 10*1024*1024)
			if err != nil {
				t.Fatal(err)
			}

			f, header := createTestFile(t, "script.sh", []byte("#!/bin/bash"), "application/x-sh")
			defer f.Close()

			_, err = store.Save(context.Background(), time.Now(), f, header)
			if err == nil {
				t.Fatal("expected error for invalid mime type")
			}
			if !strings.Contains(err.Error(), "not allowed") {
				t.Errorf("expected 'not allowed' in error, got: %v", err)
			}
		})
	}
}

func TestStore_Save_TooLarge(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			baseDir := t.TempDir()
			store, err := New(db.ConnDbName("TestSaveTooLarge"), baseDir, 100) // 100 bytes max
			if err != nil {
				t.Fatal(err)
			}

			bigContent := make([]byte, 200)
			f, header := createTestFile(t, "big.pdf", bigContent, "application/pdf")
			defer f.Close()

			_, err = store.Save(context.Background(), time.Now(), f, header)
			if err == nil {
				t.Fatal("expected error for file too large")
			}
			if !strings.Contains(err.Error(), "too large") {
				t.Errorf("expected 'too large' in error, got: %v", err)
			}
		})
	}
}

func TestStore_Delete(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			baseDir := t.TempDir()
			store, err := New(db.ConnDbName("TestDelete"), baseDir, 10*1024*1024)
			if err != nil {
				t.Fatal(err)
			}

			f, header := createTestFile(t, "receipt.png", []byte{0x89, 0x50, 0x4E, 0x47}, "image/png")
			defer f.Close()

			id, err := store.Save(context.Background(), time.Now(), f, header)
			if err != nil {
				t.Fatalf("Save failed: %v", err)
			}

			fullPath, _ := store.GetFilePath(context.Background(), id)

			err = store.Delete(context.Background(), id)
			if err != nil {
				t.Fatalf("Delete failed: %v", err)
			}

			// File should be removed from disk
			if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
				t.Error("expected file to be deleted from disk")
			}

			// DB record should be gone (soft delete)
			_, err = store.Get(context.Background(), id)
			if err == nil {
				t.Error("expected error when getting deleted attachment")
			}
		})
	}
}

func TestStore_Get_NotFound(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			baseDir := t.TempDir()
			store, err := New(db.ConnDbName("TestGetNotFound"), baseDir, 10*1024*1024)
			if err != nil {
				t.Fatal(err)
			}

			_, err = store.Get(context.Background(), 99999)
			if err == nil {
				t.Fatal("expected error for non-existent attachment")
			}
		})
	}
}
```

**Step 2: Run the tests to verify they fail**

Run: `go test ./internal/filestore/ -v -count=1`
Expected: FAIL — package does not exist yet

**Step 3: Implement the filestore package**

Create `internal/filestore/filestore.go`:

```go
package filestore

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

var (
	ErrNotFound       = errors.New("attachment not found")
	ErrMimeNotAllowed = errors.New("file type not allowed")
	ErrTooLarge       = errors.New("file too large")
)

var allowedMimeTypes = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/webp":      ".webp",
	"application/pdf": ".pdf",
}

type dbAttachment struct {
	Id           uint           `gorm:"primaryKey"`
	OriginalName string         `gorm:"size:255;not null"`
	StoragePath  string         `gorm:"size:512;not null"`
	MimeType     string         `gorm:"size:100;not null"`
	FileSize     int64          `gorm:"not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

type Attachment struct {
	Id           uint
	OriginalName string
	StoragePath  string
	MimeType     string
	FileSize     int64
}

type Store struct {
	db      *gorm.DB
	baseDir string
	maxSize int64
}

func New(db *gorm.DB, baseDir string, maxSize int64) (*Store, error) {
	if err := db.AutoMigrate(&dbAttachment{}); err != nil {
		return nil, fmt.Errorf("filestore auto-migrate: %w", err)
	}
	return &Store{db: db, baseDir: baseDir, maxSize: maxSize}, nil
}

func (s *Store) Save(ctx context.Context, date time.Time, file multipart.File, header *multipart.FileHeader) (uint, error) {
	// Validate mime type
	ext, ok := allowedMimeTypes[header.Header.Get("Content-Type")]
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrMimeNotAllowed, header.Header.Get("Content-Type"))
	}

	// Validate size
	if header.Size > s.maxSize {
		return 0, fmt.Errorf("%w: %d bytes exceeds limit of %d", ErrTooLarge, header.Size, s.maxSize)
	}

	// Generate storage path: YYYY/MM/DD_<random>.<ext>
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		return 0, fmt.Errorf("generating random name: %w", err)
	}
	randomHex := hex.EncodeToString(randomBytes)
	relDir := date.Format("2006/01")
	relPath := filepath.Join(relDir, fmt.Sprintf("%02d_%s%s", date.Day(), randomHex, ext))

	// Create directory
	absDir := filepath.Join(s.baseDir, relDir)
	if err := os.MkdirAll(absDir, 0750); err != nil {
		return 0, fmt.Errorf("creating directory: %w", err)
	}

	// Write file to disk
	absPath := filepath.Join(s.baseDir, relPath)
	dst, err := os.Create(absPath)
	if err != nil {
		return 0, fmt.Errorf("creating file: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		os.Remove(absPath) // cleanup on error
		return 0, fmt.Errorf("writing file: %w", err)
	}

	// Insert DB record
	record := dbAttachment{
		OriginalName: header.Filename,
		StoragePath:  relPath,
		MimeType:     header.Header.Get("Content-Type"),
		FileSize:     written,
	}
	if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
		os.Remove(absPath) // cleanup on error
		return 0, fmt.Errorf("inserting attachment record: %w", err)
	}

	return record.Id, nil
}

func (s *Store) Get(ctx context.Context, id uint) (*Attachment, error) {
	var record dbAttachment
	if err := s.db.WithContext(ctx).First(&record, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &Attachment{
		Id:           record.Id,
		OriginalName: record.OriginalName,
		StoragePath:  record.StoragePath,
		MimeType:     record.MimeType,
		FileSize:     record.FileSize,
	}, nil
}

func (s *Store) GetFilePath(ctx context.Context, id uint) (string, error) {
	att, err := s.Get(ctx, id)
	if err != nil {
		return "", err
	}
	absPath := filepath.Join(s.baseDir, att.StoragePath)
	// Prevent path traversal
	if !strings.HasPrefix(absPath, s.baseDir) {
		return "", fmt.Errorf("invalid storage path")
	}
	return absPath, nil
}

func (s *Store) Delete(ctx context.Context, id uint) error {
	att, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	// Remove file from disk
	absPath := filepath.Join(s.baseDir, att.StoragePath)
	if err := os.Remove(absPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing file: %w", err)
	}

	// Soft-delete DB record
	if err := s.db.WithContext(ctx).Delete(&dbAttachment{}, id).Error; err != nil {
		return fmt.Errorf("deleting attachment record: %w", err)
	}
	return nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/filestore/ -v -count=1`
Expected: All PASS

**Step 5: Commit**

```
feat: add filestore package for file attachments
```

---

### Task 2: Configuration — Add MaxAttachmentSizeMB

**Files:**
- Modify: `app/cmd/config.go:36-41` (AppSettings struct)
- Modify: `app/router/handlers/settings.go:8-13` (handler AppSettings struct)

**Step 1: Add field to AppSettings in config.go**

In `app/cmd/config.go`, add `MaxAttachmentSizeMB` to the `AppSettings` struct:

```go
type AppSettings struct {
	DateFormat           string
	MainCurrency         string
	AdditionalCurrencies []string
	Instruments          bool
	MaxAttachmentSizeMB  float64 // default 10.0; converted to bytes at startup
}
```

**Step 2: Add field to handler AppSettings in settings.go**

In `app/router/handlers/settings.go`, add:

```go
type AppSettings struct {
	DateFormat   string   `json:"dateFormat"`
	MainCurrency string   `json:"mainCurrency"`
	Currencies   []string `json:"currencies"`
	Instruments  bool     `json:"instruments"`
	Version      string   `json:"version"`
	MaxAttachmentSizeMB float64 `json:"maxAttachmentSizeMB"`
}
```

**Step 3: Wire the setting through in server.go**

In `app/cmd/server.go:162-168`, add the field to the `AppSettings` initialization:

```go
AppSettings: handlers.AppSettings{
	DateFormat:          cfg.Settings.DateFormat,
	MainCurrency:        cfg.Settings.MainCurrency,
	Currencies:          cfg.Settings.AllCurrencies(),
	Instruments:         cfg.Settings.Instruments,
	Version:             metainfo.Version,
	MaxAttachmentSizeMB: cfg.Settings.MaxAttachmentSizeMB,
},
```

**Step 4: Verify it compiles**

Run: `go build ./...`
Expected: Success

**Step 5: Commit**

```
feat: add MaxAttachmentSizeMB config setting
```

---

### Task 3: Data Directory and FileStore Wiring

**Files:**
- Modify: `app/cmd/server.go:374-419` (initDataDir — add attachments dir)
- Modify: `app/cmd/server.go:103-176` (runServer — create filestore, wire into router)
- Modify: `app/router/main.go:22-42` (Cfg struct — add FileStore)
- Modify: `app/router/main.go:44-63` (MainAppHandler struct — add fileStore)
- Modify: `app/router/main.go:69-94` (New function — wire fileStore)

**Step 1: Add attachments directory to initDataDir**

In `app/cmd/server.go`, add after the sessions dir block (around line 416) and before `return nil`:

```go
	// create attachments dir
	attachmentsDir := filepath.Join(absPath, "attachments")
	attachmentsInfo, err := os.Stat(attachmentsDir)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(attachmentsDir, 0750); err != nil {
			return fmt.Errorf("failed to create attachments directory: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to stat attachments path: %w", err)
	} else if !attachmentsInfo.IsDir() {
		return fmt.Errorf("attachments path is not a directory: %s", attachmentsDir)
	}
```

**Step 2: Create filestore.Store in runServer and wire into router Cfg**

In `app/cmd/server.go`, after the csvImportStore creation (around line 116), add:

```go
	// Compute max attachment size in bytes (default 10 MB)
	maxAttachmentBytes := int64(10 * 1024 * 1024)
	if cfg.Settings.MaxAttachmentSizeMB > 0 {
		maxAttachmentBytes = int64(cfg.Settings.MaxAttachmentSizeMB * 1024 * 1024)
	}
	attachmentStore, err := filestore.New(db, filepath.Join(cfg.DataDir, "attachments"), maxAttachmentBytes)
	if err != nil {
		return fmt.Errorf("filestore: %w", err)
	}
```

Add `filestore` to the import block. Then add `AttachmentStore: attachmentStore` to the `routerCfg` struct literal (around line 175).

**Step 3: Add FileStore to router Cfg and MainAppHandler**

In `app/router/main.go`, add to the `Cfg` struct:

```go
	AttachmentStore *filestore.Store
```

Add to `MainAppHandler`:

```go
	attachmentStore *filestore.Store
```

In the `New` function, add:

```go
	app.attachmentStore = cfg.AttachmentStore
```

Add `filestore` import: `"github.com/andresbott/etna/internal/filestore"`

**Step 4: Verify it compiles**

Run: `go build ./...`
Expected: Success

**Step 5: Commit**

```
feat: wire filestore into server startup and router
```

---

### Task 4: Accounting — Add AttachmentID to dbTransaction

**Files:**
- Modify: `internal/accounting/transaction.go:31-41` (dbTransaction struct)
- Modify: `internal/accounting/transaction.go:55-145` (public Transaction types — add AttachmentID)
- Modify: `internal/accounting/transaction.go:975-997` (publicTransactions / fromDb functions — pass AttachmentID)
- Modify: `internal/accounting/transaction.go:2232-2372` (ListTransactions — select and map AttachmentID)

**Step 1: Add AttachmentID to dbTransaction**

In `internal/accounting/transaction.go:31-41`, add the field:

```go
type dbTransaction struct {
	Id           uint      `gorm:"primaryKey"`
	Date         time.Time `gorm:"not null"`
	Description  string    `gorm:"size:255"`
	Type         TxType
	AttachmentID *uint
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
	Entries      []dbEntry      `gorm:"foreignKey:TransactionID"`
	Trades       []dbTrade      `gorm:"foreignKey:TransactionID"`
}
```

**Step 2: Add AttachmentID to all public Transaction types**

Add `AttachmentID *uint` field to each of: `Income`, `Expense`, `Transfer`, `StockBuy`, `StockSell`, `StockGrant`, `StockTransfer`, `BalanceStatus`. For example:

```go
type Income struct {
	Id           uint
	Description  string
	Amount       float64
	AccountID    uint
	CategoryID   uint
	Date         time.Time
	AttachmentID *uint
	baseTx
}
```

Repeat for all other transaction types.

**Step 3: Pass AttachmentID through fromDb conversion functions**

In each `*FromDb` function (e.g., `incomeFromDb`, `expenseFromDb`, `transferFromDb`, etc.), add `AttachmentID: in.AttachmentID` to the returned struct.

For example in `incomeFromDb` (around line 999):
```go
return Income{
	Description:  in.Description,
	Date:         in.Date,
	Amount:       in.Entries[0].Amount,
	AccountID:    in.Entries[0].AccountID,
	CategoryID:   in.Entries[0].CategoryID,
	Id:           in.Id,
	AttachmentID: in.AttachmentID,
}, nil
```

Do the same for all other `*FromDb` functions.

**Step 4: Add AttachmentID to ListTransactions query**

In the `ListTransactions` SQL select (around line 2236), add:

```sql
db_transactions.attachment_id,
```

In the `intermediate` struct (around line 2332), add:

```go
AttachmentID *uint
```

In the transaction construction loop (around line 2388), add `AttachmentID: item.AttachmentID` to every transaction type created.

**Step 5: Verify it compiles and tests pass**

Run: `go build ./... && go test ./internal/accounting/ -count=1`
Expected: All pass (AutoMigrate adds the column)

**Step 6: Commit**

```
feat: add AttachmentID to transaction model
```

---

### Task 5: API Handler — Attachment Upload Endpoint

**Files:**
- Modify: `app/router/handlers/finance/account.go:17-20` (Handler struct — add FileStore)
- Create or modify: `app/router/handlers/finance/attachment.go` (new handler file)
- Modify: `app/router/api_v0.go:68-70` (wire handler with filestore)
- Modify: `app/router/api_v0.go:326-340` (add attachment routes)

**Step 1: Add FileStore to Handler struct**

In `app/router/handlers/finance/account.go:17-20`:

```go
type Handler struct {
	Store           *accounting.Store
	InstrumentStore *marketdata.Store
	FileStore       *filestore.Store
}
```

Add import: `"github.com/andresbott/etna/internal/filestore"`

**Step 2: Wire FileStore in accountingAPI**

In `app/router/api_v0.go:70`, update handler construction:

```go
finHndlr := finHandler.Handler{Store: h.finStore, InstrumentStore: h.marketStore, FileStore: h.attachmentStore}
```

Add import for filestore if needed (it may not be directly imported here since it's through the handler).

**Step 3: Create the attachment handler file**

Create `app/router/handlers/finance/attachment.go`:

```go
package finance

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/andresbott/etna/internal/filestore"
)

type attachmentPayload struct {
	Id           uint   `json:"id"`
	OriginalName string `json:"originalName"`
	MimeType     string `json:"mimeType"`
	FileSize     int64  `json:"fileSize"`
}

func (h *Handler) UploadAttachment(txId uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the transaction to verify it exists and get its date
		tx, err := h.Store.GetTransaction(r.Context(), txId)
		if err != nil {
			http.Error(w, "transaction not found", http.StatusNotFound)
			return
		}

		// Extract date from transaction (all types have Date)
		txDate := transactionDate(tx)

		// Parse multipart form with configured max size
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, fmt.Sprintf("failed to parse form: %v", err), http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to get file: %v", err), http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Check if transaction already has an attachment — delete old one first
		existingAttID := transactionAttachmentID(tx)
		if existingAttID != nil {
			if err := h.FileStore.Delete(r.Context(), *existingAttID); err != nil {
				http.Error(w, fmt.Sprintf("failed to remove existing attachment: %v", err), http.StatusInternalServerError)
				return
			}
		}

		// Save the file
		attID, err := h.FileStore.Save(r.Context(), txDate, file, header)
		if err != nil {
			if errors.Is(err, filestore.ErrMimeNotAllowed) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if errors.Is(err, filestore.ErrTooLarge) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, fmt.Sprintf("failed to save file: %v", err), http.StatusInternalServerError)
			return
		}

		// Update the transaction's AttachmentID
		if err := h.Store.SetAttachmentID(r.Context(), txId, &attID); err != nil {
			// Cleanup: delete the just-saved file
			_ = h.FileStore.Delete(r.Context(), attID)
			http.Error(w, fmt.Sprintf("failed to update transaction: %v", err), http.StatusInternalServerError)
			return
		}

		// Return attachment metadata
		att, err := h.FileStore.Get(r.Context(), attID)
		if err != nil {
			http.Error(w, "attachment saved but metadata fetch failed", http.StatusInternalServerError)
			return
		}

		resp := attachmentPayload{
			Id:           att.Id,
			OriginalName: att.OriginalName,
			MimeType:     att.MimeType,
			FileSize:     att.FileSize,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
}

func (h *Handler) GetAttachment(txId uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tx, err := h.Store.GetTransaction(r.Context(), txId)
		if err != nil {
			http.Error(w, "transaction not found", http.StatusNotFound)
			return
		}

		attID := transactionAttachmentID(tx)
		if attID == nil {
			http.Error(w, "no attachment", http.StatusNotFound)
			return
		}

		att, err := h.FileStore.Get(r.Context(), *attID)
		if err != nil {
			http.Error(w, "attachment not found", http.StatusNotFound)
			return
		}

		fullPath, err := h.FileStore.GetFilePath(r.Context(), *attID)
		if err != nil {
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", att.MimeType)
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", att.OriginalName))
		http.ServeFile(w, r, fullPath)
	})
}

func (h *Handler) DeleteAttachment(txId uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tx, err := h.Store.GetTransaction(r.Context(), txId)
		if err != nil {
			http.Error(w, "transaction not found", http.StatusNotFound)
			return
		}

		attID := transactionAttachmentID(tx)
		if attID == nil {
			http.Error(w, "no attachment", http.StatusNotFound)
			return
		}

		if err := h.FileStore.Delete(r.Context(), *attID); err != nil {
			http.Error(w, fmt.Sprintf("failed to delete attachment: %v", err), http.StatusInternalServerError)
			return
		}

		if err := h.Store.SetAttachmentID(r.Context(), txId, nil); err != nil {
			http.Error(w, fmt.Sprintf("failed to update transaction: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}

// transactionDate extracts the date from any transaction type.
func transactionDate(tx interface{}) time.Time {
	// Use the same pattern as transactionToPayload — type-switch
	// All transaction types embed a Date field
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
	case accounting.BalanceStatus:
		return t.Date
	default:
		return time.Now()
	}
}

// transactionAttachmentID extracts the AttachmentID from any transaction type.
func transactionAttachmentID(tx interface{}) *uint {
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
	case accounting.BalanceStatus:
		return t.AttachmentID
	default:
		return nil
	}
}
```

Note: the `time` and `accounting` imports need to be added. The `filepath` import can be removed if unused.

**Step 4: Add SetAttachmentID method to accounting Store**

In `internal/accounting/transaction.go`, add near the other update methods:

```go
// SetAttachmentID updates the attachment_id on a transaction.
func (store *Store) SetAttachmentID(ctx context.Context, txId uint, attachmentID *uint) error {
	result := store.db.WithContext(ctx).
		Model(&dbTransaction{}).
		Where("id = ?", txId).
		Update("attachment_id", attachmentID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTransactionNotFound
	}
	return nil
}
```

**Step 5: Register routes in api_v0.go**

In `app/router/api_v0.go`, after the DELETE entries route (around line 340), add:

```go
	// Attachment routes
	r.Path(fmt.Sprintf("%s/{id}/attachment", finEntries)).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		finHndlr.UploadAttachment(itemId).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{id}/attachment", finEntries)).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		finHndlr.GetAttachment(itemId).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{id}/attachment", finEntries)).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		finHndlr.DeleteAttachment(itemId).ServeHTTP(w, r)
	})
```

**Step 6: Verify it compiles**

Run: `go build ./...`
Expected: Success

**Step 7: Commit**

```
feat: add attachment upload/serve/delete API endpoints
```

---

### Task 6: API — Add AttachmentID to Transaction Payload and Delete Cleanup

**Files:**
- Modify: `app/router/handlers/finance/transaction.go:37-75` (transactionPayload — add AttachmentID)
- Modify: `app/router/handlers/finance/transaction.go:410-505` (transactionToPayload — map AttachmentID)
- Modify: `app/router/handlers/finance/transaction.go:387-401` (DeleteTx — cleanup attachment)

**Step 1: Add AttachmentID to transactionPayload**

In `app/router/handlers/finance/transaction.go:37-75`, add:

```go
AttachmentID *uint `json:"attachmentId,omitempty"`
```

**Step 2: Map AttachmentID in transactionToPayload**

In each case of the `transactionToPayload` switch (lines 410-505), add `AttachmentID: entry.AttachmentID` to the returned payload. For example:

```go
case accounting.Income:
	return transactionPayload{
		Id:           entry.Id,
		Description:  entry.Description,
		Date:         dateOnlyTime{Time: entry.Date},
		Type:         incomeTxStr,
		Amount:       entry.Amount,
		AccountId:    entry.AccountID,
		CategoryId:   entry.CategoryID,
		AttachmentID: entry.AttachmentID,
	}
```

Do the same for all cases.

**Step 3: Update DeleteTx to clean up attachments**

Modify `DeleteTx` in `app/router/handlers/finance/transaction.go:387-401`:

```go
func (h *Handler) DeleteTx(Id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get transaction to check for attachment before deleting
		tx, err := h.Store.GetTransaction(r.Context(), Id)
		if err == nil {
			if attID := transactionAttachmentID(tx); attID != nil && h.FileStore != nil {
				_ = h.FileStore.Delete(r.Context(), *attID)
			}
		}

		err = h.Store.DeleteTransaction(r.Context(), Id)
		if err != nil {
			if errors.Is(err, accounting.ErrEntryNotFound) {
				http.Error(w, "entry not found", http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("unable to delete entry: %s", err.Error()), http.StatusInternalServerError)
			}
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}
```

Note: `transactionAttachmentID` is defined in `attachment.go` from Task 5.

**Step 4: Verify it compiles and tests pass**

Run: `go build ./... && go test ./... -count=1 2>&1 | tail -20`
Expected: All pass

**Step 5: Commit**

```
feat: include attachmentId in transaction API responses and cleanup on delete
```

---

### Task 7: Frontend — Attachment API Client

**Files:**
- Create: `webui/src/lib/api/Attachment.ts`

**Step 1: Create the API client module**

```typescript
import { apiClient } from '@/lib/api/client'

const API_BASE_URL = import.meta.env.VITE_SERVER_URL_V0

export interface AttachmentMeta {
    id: number
    originalName: string
    mimeType: string
    fileSize: number
}

export const uploadAttachment = async (txId: number, file: File): Promise<AttachmentMeta> => {
    const formData = new FormData()
    formData.append('file', file)
    const { data } = await apiClient.post<AttachmentMeta>(`/fin/entries/${txId}/attachment`, formData, {
        headers: { 'Content-Type': 'multipart/form-data' }
    })
    return data
}

export const getAttachmentUrl = (txId: number): string => {
    return `${API_BASE_URL}/fin/entries/${txId}/attachment`
}

export const deleteAttachment = async (txId: number): Promise<void> => {
    await apiClient.delete(`/fin/entries/${txId}/attachment`)
}
```

**Step 2: Commit**

```
feat: add attachment API client module
```

---

### Task 8: Frontend — Paperclip Icon in Transaction Tables

**Files:**
- Modify: `webui/src/views/entries/EntriesTable.vue:229-237` and `378-409` (both Actions columns)
- Modify: `webui/src/views/entries/AccountEntriesTable.vue:337-365` (Actions column)

**Step 1: Add paperclip button to EntriesTable financial view Actions column**

In `webui/src/views/entries/EntriesTable.vue`, in the first Actions column (around line 231), add before the edit button:

```vue
<Button
    v-if="data.attachmentId"
    icon="pi pi-paperclip"
    text
    rounded
    class="p-1"
    @click="openAttachment(data)"
    v-tooltip.bottom="'View Attachment'"
/>
```

**Step 2: Add the same to the second Actions column (around line 380)**

Same button added before the edit button in the non-financial Actions column.

**Step 3: Add the openAttachment function**

In the `<script setup>` section, import and add the function:

```typescript
import { getAttachmentUrl } from '@/lib/api/Attachment'

const openAttachment = (data) => {
    window.open(getAttachmentUrl(data.id), '_blank')
}
```

**Step 4: Do the same for AccountEntriesTable.vue**

In `webui/src/views/entries/AccountEntriesTable.vue`, add the same paperclip button in the Actions column (around line 341, inside the `v-else` div), and add the same import and function.

**Step 5: Commit**

```
feat: add paperclip icon in transaction tables for attachments
```

---

### Task 9: Frontend — File Input in IncomeExpenseDialog

**Files:**
- Modify: `webui/src/views/entries/dialogs/IncomeExpenseDialog.vue`

**Step 1: Add file state and attachment handling**

In the `<script setup>` section, add:

```typescript
import { uploadAttachment, deleteAttachment, getAttachmentUrl } from '@/lib/api/Attachment'

const selectedFile = ref<File | null>(null)
const existingAttachmentId = ref<number | null>(null)
const attachmentPendingDelete = ref(false)
```

Add a prop for the existing attachment:
```typescript
attachmentId: { type: Number, default: null },
```

Watch props to track existing attachment:
```typescript
watch(props, (newProps) => {
    // ... existing watches ...
    existingAttachmentId.value = newProps.attachmentId || null
    selectedFile.value = null
    attachmentPendingDelete.value = false
})
```

**Step 2: Update handleSubmit to upload/delete attachment after transaction save**

After the successful `createEntry` or `updateEntry` call, add:

```typescript
const savedId = props.isEdit ? props.entryId : result.id

// Handle attachment changes
if (attachmentPendingDelete.value && existingAttachmentId.value) {
    try {
        await deleteAttachment(savedId)
    } catch (e) {
        console.error('Failed to delete attachment:', e)
    }
}
if (selectedFile.value) {
    try {
        await uploadAttachment(savedId, selectedFile.value)
    } catch (e) {
        console.error('Failed to upload attachment:', e)
    }
}
```

Note: capture the return value of `createEntry` as `result` to get the new ID.

**Step 3: Add file input in the template**

After the CategorySelect, before the action buttons, add:

```vue
<!-- Attachment -->
<div>
    <label class="form-label">Attachment</label>
    <div v-if="existingAttachmentId && !attachmentPendingDelete && !selectedFile" class="flex align-items-center gap-2">
        <Button
            icon="pi pi-paperclip"
            label="View attachment"
            text
            size="small"
            @click="window.open(getAttachmentUrl(entryId), '_blank')"
        />
        <Button
            icon="pi pi-trash"
            text
            rounded
            severity="danger"
            size="small"
            @click="attachmentPendingDelete = true"
            v-tooltip.bottom="'Remove attachment'"
        />
    </div>
    <div v-else>
        <input
            type="file"
            accept=".jpg,.jpeg,.png,.webp,.pdf"
            @change="(e) => selectedFile = e.target.files?.[0] || null"
        />
        <div v-if="selectedFile" class="flex align-items-center gap-2 mt-1">
            <span class="text-sm">{{ selectedFile.name }}</span>
            <Button
                icon="pi pi-times"
                text
                rounded
                severity="danger"
                size="small"
                @click="selectedFile = null"
            />
        </div>
    </div>
</div>
```

**Step 4: Verify it works in the browser**

Start the dev server and test creating an income/expense with a file attached.

**Step 5: Commit**

```
feat: add file attachment input to income/expense dialog
```

---

### Task 10: Frontend — File Input in TransferDialog and BalanceStatusDialog

**Files:**
- Modify: `webui/src/views/entries/dialogs/TransferDialog.vue`
- Modify: `webui/src/views/entries/dialogs/BalanceStatusDialog.vue`

**Step 1: Apply the same pattern from Task 9 to TransferDialog**

Add the same imports, state refs, props, submit logic, and template file input as IncomeExpenseDialog. The file input goes before the action buttons div.

**Step 2: Apply the same pattern to BalanceStatusDialog**

Same changes.

**Step 3: Pass attachmentId prop from parent**

Check the parent views that open these dialogs (`EntriesView.vue`, `AccountEntriesView.vue`) and ensure `attachmentId` is passed when editing. The data already comes from the API response (added in Task 6), so bind `:attachmentId="editData.attachmentId"` on the dialog components.

**Step 4: Test in browser**

Test creating/editing transfers and balance status entries with attachments.

**Step 5: Commit**

```
feat: add file attachment input to transfer and balance status dialogs
```

---

### Task 11: Integration Testing and Cleanup

**Step 1: Run full backend test suite**

Run: `go test ./... -count=1`
Expected: All pass

**Step 2: Run frontend lint/build**

Run: `cd webui && npm run build`
Expected: Success with no errors

**Step 3: Manual end-to-end test**

Test the following flows:
1. Create an income transaction with a PDF attachment → verify paperclip shows in list → click it → file opens in new tab
2. Edit that transaction → see existing attachment → replace it with an image
3. Delete the transaction → verify attachment file is cleaned up
4. Create a transfer with an attachment
5. Remove an attachment from an existing transaction via edit dialog

**Step 4: Commit any fixes**

```
fix: address integration testing issues
```
