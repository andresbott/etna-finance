# Backup Attachments Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Include transaction file attachments (images, PDFs) in the backup zip so that restore recovers both the metadata and the actual files.

**Architecture:** During export, collect all attachment IDs referenced by transactions, read their metadata + file bytes from the filestore, and write them into the zip as binary files alongside a JSON manifest. During import, restore attachments first (before transactions), then remap IDs when creating transactions. The `filestore.Store` gets two new methods: `SaveRaw` (import without multipart) and `WipeData` (restore cleanup). All callers (`backup.ExportToFile`, `backup.Import`, handler, task) gain an optional `*filestore.Store` parameter — nil means skip attachments gracefully.

**Tech Stack:** Go, gorm, archive/zip, filestore, accounting store

---

### Task 1: Add `SaveRaw` method to filestore

**Files:**
- Modify: `internal/filestore/filestore.go`
- Test: `internal/filestore/filestore_test.go`

**Step 1: Write the failing test**

Add to `internal/filestore/filestore_test.go`:

```go
func TestSaveRaw(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			baseDir := t.TempDir()
			store, err := New(db.ConnDbName("TestSaveRaw"), baseDir, 10*1024*1024)
			if err != nil {
				t.Fatal(err)
			}

			// JPEG content
			content := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0x00}, 100)...)

			ctx := context.Background()
			date := time.Date(2025, 5, 20, 0, 0, 0, 0, time.UTC)

			id, err := store.SaveRaw(ctx, date, content, "receipt.jpg", "image/jpeg")
			if err != nil {
				t.Fatalf("SaveRaw failed: %v", err)
			}
			if id == 0 {
				t.Fatal("expected non-zero ID")
			}

			att, err := store.Get(ctx, id)
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}
			if att.OriginalName != "receipt.jpg" {
				t.Errorf("expected OriginalName 'receipt.jpg', got %q", att.OriginalName)
			}
			if att.MimeType != "image/jpeg" {
				t.Errorf("expected MimeType 'image/jpeg', got %q", att.MimeType)
			}
			if att.FileSize != int64(len(content)) {
				t.Errorf("expected FileSize %d, got %d", len(content), att.FileSize)
			}

			// Verify file on disk
			fp, err := store.GetFilePath(ctx, id)
			if err != nil {
				t.Fatalf("GetFilePath failed: %v", err)
			}
			diskContent := readTestFile(t, fp)
			if !bytes.Equal(diskContent, content) {
				t.Error("disk content does not match original")
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd internal/filestore && go test -run TestSaveRaw -v`
Expected: FAIL — `store.SaveRaw undefined`

**Step 3: Write minimal implementation**

Add to `internal/filestore/filestore.go`:

```go
// SaveRaw stores raw bytes as an attachment, bypassing multipart form handling.
// Used during backup restore to re-create attachments from zip content.
func (s *Store) SaveRaw(ctx context.Context, date time.Time, content []byte, originalName, mimeType string) (uint, error) {
	if int64(len(content)) > s.maxSize {
		return 0, ErrTooLarge
	}

	if !allowedMimeTypes[mimeType] {
		return 0, ErrMimeNotAllowed
	}

	ext := mimeToExt[mimeType]
	randBytes := make([]byte, 4)
	if _, err := rand.Read(randBytes); err != nil {
		return 0, fmt.Errorf("generating random name: %w", err)
	}
	randHex := hex.EncodeToString(randBytes)

	storagePath := fmt.Sprintf("%04d/%02d/%02d_%s%s",
		date.Year(), date.Month(), date.Day(), randHex, ext)

	absPath := filepath.Join(s.baseDir, storagePath)

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return 0, fmt.Errorf("creating directory: %w", err)
	}

	if err := os.WriteFile(absPath, content, 0o600); err != nil {
		return 0, fmt.Errorf("writing file: %w", err)
	}

	record := dbAttachment{
		OriginalName: originalName,
		StoragePath:  storagePath,
		MimeType:     mimeType,
		FileSize:     int64(len(content)),
	}
	if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
		_ = os.Remove(absPath)
		return 0, fmt.Errorf("inserting record: %w", err)
	}

	return record.Id, nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd internal/filestore && go test -run TestSaveRaw -v`
Expected: PASS

**Step 5: Commit**

```
git add internal/filestore/filestore.go internal/filestore/filestore_test.go
git commit -m "feat(filestore): add SaveRaw method for backup restore"
```

---

### Task 2: Add `WipeData` method to filestore

**Files:**
- Modify: `internal/filestore/filestore.go`
- Test: `internal/filestore/filestore_test.go`

**Step 1: Write the failing test**

Add to `internal/filestore/filestore_test.go`:

```go
func TestWipeData(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			baseDir := t.TempDir()
			store, err := New(db.ConnDbName("TestWipeData"), baseDir, 10*1024*1024)
			if err != nil {
				t.Fatal(err)
			}

			ctx := context.Background()
			date := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)

			// Save two files
			content1 := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0x00}, 100)...)
			id1, err := store.SaveRaw(ctx, date, content1, "a.jpg", "image/jpeg")
			if err != nil {
				t.Fatalf("SaveRaw 1 failed: %v", err)
			}
			content2 := append([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, bytes.Repeat([]byte{0x00}, 100)...)
			id2, err := store.SaveRaw(ctx, date, content2, "b.png", "image/png")
			if err != nil {
				t.Fatalf("SaveRaw 2 failed: %v", err)
			}

			// Get file paths before wipe
			fp1, _ := store.GetFilePath(ctx, id1)
			fp2, _ := store.GetFilePath(ctx, id2)

			// Wipe
			err = store.WipeData(ctx)
			if err != nil {
				t.Fatalf("WipeData failed: %v", err)
			}

			// DB records gone
			_, err = store.Get(ctx, id1)
			if err != ErrNotFound {
				t.Errorf("expected ErrNotFound for id1, got %v", err)
			}
			_, err = store.Get(ctx, id2)
			if err != ErrNotFound {
				t.Errorf("expected ErrNotFound for id2, got %v", err)
			}

			// Files removed from disk
			if _, err := os.Stat(fp1); !os.IsNotExist(err) {
				t.Error("file 1 should be removed from disk")
			}
			if _, err := os.Stat(fp2); !os.IsNotExist(err) {
				t.Error("file 2 should be removed from disk")
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd internal/filestore && go test -run TestWipeData -v`
Expected: FAIL — `store.WipeData undefined`

**Step 3: Write minimal implementation**

Add to `internal/filestore/filestore.go`:

```go
// WipeData deletes all attachment files from disk and hard-deletes all DB records.
// Used during backup restore to clear existing data before importing.
func (s *Store) WipeData(ctx context.Context) error {
	// Fetch all records (including soft-deleted)
	var records []dbAttachment
	if err := s.db.WithContext(ctx).Unscoped().Find(&records).Error; err != nil {
		return fmt.Errorf("listing attachments: %w", err)
	}

	// Remove files from disk
	for _, rec := range records {
		absPath := filepath.Join(s.baseDir, rec.StoragePath)
		_ = os.Remove(absPath)
	}

	// Hard-delete all DB records
	if err := s.db.WithContext(ctx).Unscoped().Where("1 = 1").Delete(&dbAttachment{}).Error; err != nil {
		return fmt.Errorf("deleting attachment records: %w", err)
	}

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd internal/filestore && go test -run TestWipeData -v`
Expected: PASS

**Step 5: Commit**

```
git add internal/filestore/filestore.go internal/filestore/filestore_test.go
git commit -m "feat(filestore): add WipeData method for backup restore"
```

---

### Task 3: Update backup data model

**Files:**
- Modify: `internal/backup/dataV1.go`

**Step 1: Add `AttachmentID` to `TransactionV1` and add `attachmentV1` type**

In `internal/backup/dataV1.go`, add `AttachmentID` field to `TransactionV1`:

```go
// Add after the existing SourceAccountID field (line ~86):
AttachmentID *uint `json:"attachmentId,omitempty"`
```

Add new constants and type at the end of the file (before the closing of the file):

```go
const attachmentsFile = "attachments.json"
const attachmentsDir = "attachments/"

type attachmentV1 struct {
	ID           uint   `json:"id"`
	OriginalName string `json:"originalName"`
	MimeType     string `json:"mimeType"`
	FileSize     int64  `json:"fileSize"`
	ZipPath      string `json:"zipPath"`
}
```

**Step 2: Verify it compiles**

Run: `cd internal/backup && go build ./...`
Expected: success

**Step 3: Commit**

```
git add internal/backup/dataV1.go
git commit -m "feat(backup): add attachment data types to backup schema"
```

---

### Task 4: Add `writeBinaryFile` to zipWriter

**Files:**
- Modify: `internal/backup/zip.go`

**Step 1: Add the method**

Add to `internal/backup/zip.go`:

```go
// writeBinaryFile writes raw bytes into the zip archive at the given path.
func (zw *zipWriter) writeBinaryFile(filename string, data []byte) error {
	f, err := zw.writer.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file in zip: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write binary data: %w", err)
	}

	return nil
}
```

**Step 2: Verify it compiles**

Run: `cd internal/backup && go build ./...`
Expected: success

**Step 3: Commit**

```
git add internal/backup/zip.go
git commit -m "feat(backup): add writeBinaryFile to zipWriter"
```

---

### Task 5: Update export to include attachments

This is the core export change. We need to:
- Update function signatures to accept `*filestore.Store`
- Update `writeTransactions` to include `AttachmentID`
- Add `writeAttachments` function
- Call it from `export()`

**Files:**
- Modify: `internal/backup/export.go`

**Step 1: Update function signatures**

Change `ExportToFile` and `export` signatures:

```go
// Old:
func ExportToFile(ctx context.Context, store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, zipFile string) error
func export(ctx context.Context, store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, fullPath string) error

// New:
func ExportToFile(ctx context.Context, store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, fileStore *filestore.Store, zipFile string) error
func export(ctx context.Context, store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, fileStore *filestore.Store, fullPath string) error
```

Update `ExportToFile` body to pass `fileStore`:

```go
func ExportToFile(ctx context.Context, store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, fileStore *filestore.Store, zipFile string) error {
	err := verifyZipPath(zipFile)
	if err != nil {
		return err
	}
	return export(ctx, store, mdStore, csvStore, fileStore, zipFile)
}
```

Add `"github.com/andresbott/etna/internal/filestore"` to the import block.

**Step 2: Update `writeTransactions` to return attachment IDs and include them in the JSON**

Change `writeTransactions` signature to return attachment IDs it encounters:

```go
func writeTransactions(ctx context.Context, zw *zipWriter, store *accounting.Store) ([]uint, error) {
```

For each transaction type in the switch statement, extract AttachmentID. The simplest approach: add a helper that extracts it from the transaction interface, and after appending each `TransactionV1`, set the AttachmentID. Here is the updated function — note each case now includes `AttachmentID`:

In the switch cases, add `AttachmentID: item.AttachmentID` to every `TransactionV1` literal. For example:

```go
case accounting.Transfer:
    jsonData = append(jsonData, TransactionV1{
        Id:              item.Id,
        Description:     item.Description,
        OriginAmount:    item.OriginAmount,
        OriginAccountID: item.OriginAccountID,
        TargetAmount:    item.TargetAmount,
        TargetAccountID: item.TargetAccountID,
        Date:            item.Date,
        Type:            txTypeTransfer,
        AttachmentID:    item.AttachmentID,
    })
```

Do this for ALL cases: `Transfer`, `Income`, `Expense`, `StockBuy`, `StockSell`, `StockGrant`, `StockTransfer`.

After the loop, collect unique non-nil attachment IDs:

```go
	// Collect unique attachment IDs
	seen := map[uint]bool{}
	var attachmentIDs []uint
	for _, tx := range jsonData {
		if tx.AttachmentID != nil && !seen[*tx.AttachmentID] {
			seen[*tx.AttachmentID] = true
			attachmentIDs = append(attachmentIDs, *tx.AttachmentID)
		}
	}

	if err := zw.writeJsonFile(transactionsFile, jsonData); err != nil {
		return nil, err
	}
	return attachmentIDs, nil
```

**Step 3: Add `writeAttachments` function**

```go
func writeAttachments(ctx context.Context, zw *zipWriter, fileStore *filestore.Store, attachmentIDs []uint) error {
	if fileStore == nil || len(attachmentIDs) == 0 {
		return zw.writeJsonFile(attachmentsFile, []attachmentV1{})
	}

	var manifest []attachmentV1
	for _, id := range attachmentIDs {
		att, err := fileStore.Get(ctx, id)
		if err != nil {
			// Skip missing attachments — don't fail the whole backup
			continue
		}

		filePath, err := fileStore.GetFilePath(ctx, id)
		if err != nil {
			continue
		}

		content, err := os.ReadFile(filePath) //nolint:gosec // path from trusted filestore
		if err != nil {
			continue
		}

		// Determine extension from mime type
		ext := mimeToExt(att.MimeType)
		zipPath := fmt.Sprintf("%s%d%s", attachmentsDir, att.Id, ext)

		if err := zw.writeBinaryFile(zipPath, content); err != nil {
			return fmt.Errorf("failed to write attachment %d: %w", id, err)
		}

		manifest = append(manifest, attachmentV1{
			ID:           att.Id,
			OriginalName: att.OriginalName,
			MimeType:     att.MimeType,
			FileSize:     att.FileSize,
			ZipPath:      zipPath,
		})
	}

	if manifest == nil {
		manifest = []attachmentV1{}
	}
	return zw.writeJsonFile(attachmentsFile, manifest)
}

func mimeToExt(mime string) string {
	switch mime {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "application/pdf":
		return ".pdf"
	default:
		return ".bin"
	}
}
```

**Step 4: Update `export()` to call `writeTransactions` and `writeAttachments`**

Replace the `writeTransactions` call in `export()`:

```go
	// Old:
	// err = writeTransactions(ctx, zw, store)
	// if err != nil {
	//     return err
	// }

	// New:
	attachmentIDs, err := writeTransactions(ctx, zw, store)
	if err != nil {
		return err
	}

	err = writeAttachments(ctx, zw, fileStore, attachmentIDs)
	if err != nil {
		return err
	}
```

**Step 5: Verify it compiles**

Run: `cd internal/backup && go build ./...`
Expected: compilation errors in callers (backup handler, task, tests) — that's expected, we'll fix them in later tasks.

**Step 6: Commit**

```
git add internal/backup/export.go
git commit -m "feat(backup): export attachments in backup zip"
```

---

### Task 6: Update import to restore attachments

**Files:**
- Modify: `internal/backup/import.go`

**Step 1: Update `Import` signature**

```go
// Old:
func Import(ctx context.Context, store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, file string) error

// New:
func Import(ctx context.Context, store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, fileStore *filestore.Store, file string) error
```

Add `"github.com/andresbott/etna/internal/filestore"` to the import block.

**Step 2: Add filestore wipe in `Import`**

After the existing wipes (`store.WipeData`, `mdStore.WipeData`, `csvStore.WipeData`), add:

```go
	if fileStore != nil {
		err = fileStore.WipeData(ctx)
		if err != nil {
			return err
		}
	}
```

**Step 3: Add `importAttachments` function**

```go
func importAttachments(ctx context.Context, fileStore *filestore.Store, r *zip.ReadCloser) (map[uint]uint, error) {
	attachmentsMap := map[uint]uint{}
	if fileStore == nil {
		return attachmentsMap, nil
	}

	// Load manifest — if not present, return empty map (old backup without attachments)
	manifest, err := loadAttachmentManifest(r)
	if err != nil {
		return attachmentsMap, nil // gracefully skip
	}

	for _, att := range manifest {
		content, err := readZipBinary(r, att.ZipPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read attachment %s from zip: %w", att.ZipPath, err)
		}

		// Use a fixed date for storage path — the original date doesn't matter for restore
		date := time.Now()
		newID, err := fileStore.SaveRaw(ctx, date, content, att.OriginalName, att.MimeType)
		if err != nil {
			return nil, fmt.Errorf("failed to save attachment %d: %w", att.ID, err)
		}
		attachmentsMap[att.ID] = newID
	}

	return attachmentsMap, nil
}

func loadAttachmentManifest(r *zip.ReadCloser) ([]attachmentV1, error) {
	for _, f := range r.File {
		if f.Name != attachmentsFile {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer func() { _ = rc.Close() }()

		data, err := io.ReadAll(rc)
		if err != nil {
			return nil, err
		}

		var manifest []attachmentV1
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, err
		}
		return manifest, nil
	}
	return nil, fmt.Errorf("attachments manifest not found")
}

func readZipBinary(r *zip.ReadCloser, path string) ([]byte, error) {
	for _, f := range r.File {
		if f.Name != path {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer func() { _ = rc.Close() }()
		return io.ReadAll(rc)
	}
	return nil, fmt.Errorf("file %s not found in zip", path)
}
```

**Step 4: Update `importTransactions` to accept and use attachmentsMap**

Change signature:

```go
// Old:
func importTransactions(ctx context.Context, store *accounting.Store, r *zip.ReadCloser, accountsMap, incomeMap, expenseMap, instrumentsMap map[uint]uint) error

// New:
func importTransactions(ctx context.Context, store *accounting.Store, r *zip.ReadCloser, accountsMap, incomeMap, expenseMap, instrumentsMap, attachmentsMap map[uint]uint) error
```

Inside `importTransactions`, for each transaction type, after creating the transaction struct and before the `store.CreateTransaction` call, set the `AttachmentID` if present. The cleanest way: after the switch block but before `CreateTransaction`, remap and set:

```go
		// Remap AttachmentID
		var remappedAttID *uint
		if tx.AttachmentID != nil {
			if newID, ok := attachmentsMap[*tx.AttachmentID]; ok {
				remappedAttID = &newID
			}
		}
```

Then for each transaction type in the switch, set AttachmentID on the struct. Since each type is a different struct, the simplest approach is to add `AttachmentID: remappedAttID` in each case. But since `remappedAttID` is computed after the switch, we need to restructure slightly. Better approach: compute it before the switch, then include it in each case:

Actually, compute `remappedAttID` right before the switch:

```go
	for _, tx := range txs {
		// Remap AttachmentID
		var remappedAttID *uint
		if tx.AttachmentID != nil {
			if newID, ok := attachmentsMap[*tx.AttachmentID]; ok {
				remappedAttID = &newID
			}
		}

		var item accounting.Transaction
		switch tx.Type {
		case txTypeIncome:
			in := accounting.Income{
				Description: tx.Description, Amount: tx.Amount, CategoryID: tx.CategoryID, Date: tx.Date,
			}
			in.AccountID = accountsMap[tx.AccountID]
			in.CategoryID = incomeMap[tx.CategoryID]
			in.AttachmentID = remappedAttID
			item = in
		// ... same pattern for all other types
```

Add `<type>.AttachmentID = remappedAttID` to every case in the switch.

**Step 5: Update `Import()` body to call `importAttachments` and pass map to `importTransactions`**

Add the import call after instruments and before transactions:

```go
	attachmentsMap, err := importAttachments(ctx, fileStore, r)
	if err != nil {
		return err
	}

	err = importTransactions(ctx, store, r, accountsMap, inMap, exMap, instrumentsMap, attachmentsMap)
```

**Step 6: Verify it compiles**

Run: `cd internal/backup && go build ./...`
Expected: compilation errors in callers — expected.

**Step 7: Commit**

```
git add internal/backup/import.go
git commit -m "feat(backup): import attachments from backup zip"
```

---

### Task 7: Update callers (handler, task) and fix compilation

**Files:**
- Modify: `app/router/handlers/backup/backup.go`
- Modify: `app/router/api_v0.go` (line ~818)
- Modify: `app/tasks/backup.go`
- Modify: `app/cmd/server.go` (line ~313)

**Step 1: Update backup handler**

In `app/router/handlers/backup/backup.go`, add `FileStore` field and update calls:

```go
// Add to Handler struct:
type Handler struct {
	Destination string
	Store       *accounting.Store
	MdStore     *marketdata.Store
	CsvStore    *csvimport.Store
	FileStore   *filestore.Store
}
```

Add `"github.com/andresbott/etna/internal/filestore"` to imports.

Update `CreateBackup()` — change the `backup.ExportToFile` call:

```go
err = backup.ExportToFile(r.Context(), h.Store, h.MdStore, h.CsvStore, h.FileStore, backupFile)
```

Update `RestoreUpload()` and `RestoreFromExisting()` — change the `backup.Import` calls:

```go
if err := backup.Import(r.Context(), h.Store, h.MdStore, h.CsvStore, h.FileStore, dstPath); err != nil {
```

```go
if err := backup.Import(r.Context(), h.Store, h.MdStore, h.CsvStore, h.FileStore, targetFile); err != nil {
```

**Step 2: Update backup handler wiring in `api_v0.go`**

At `app/router/api_v0.go:818`, add `FileStore`:

```go
backupHndl := backup.Handler{
    Destination: h.backupDestination,
    Store:       h.finStore,
    MdStore:     h.marketStore,
    CsvStore:    h.csvImportStore,
    FileStore:   h.attachmentStore,
}
```

**Step 3: Update task**

In `app/tasks/backup.go`, update `BackupTaskCfg`, `NewBackupTaskFn`, and `newBackupFunc` to accept `*filestore.Store`:

```go
type BackupTaskCfg struct {
	Store       *accounting.Store
	MdStore     *marketdata.Store
	CsvStore    *csvimport.Store
	FileStore   *filestore.Store
	Destination string
	Interval    time.Duration
	Logger      *slog.Logger
}

func NewBackupTaskFn(store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, fileStore *filestore.Store, destination string, l *slog.Logger) func(ctx context.Context) error {
	return newBackupFunc(store, mdStore, csvStore, fileStore, destination, l)
}

func newBackupFunc(store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, fileStore *filestore.Store, destination string, l *slog.Logger) func(ctx context.Context) error {
```

Add `"github.com/andresbott/etna/internal/filestore"` to imports.

Update the `backup.ExportToFile` call inside `newBackupFunc`:

```go
err := backup.ExportToFile(ctx, store, mdStore, csvStore, fileStore, zipFile)
```

**Step 4: Update task registration in `server.go`**

At `app/cmd/server.go:313`, add `attachmentStore`:

```go
runner.RegisterTask(tasks.NewBackupTaskFn(finStore, marketStore, csvImportStore, attachmentStore, backupDest, l), tasks.BackupTaskName, 0)
```

**Step 5: Verify everything compiles**

Run: `go build ./...`
Expected: success (tests may still fail but compilation should work)

**Step 6: Commit**

```
git add app/router/handlers/backup/backup.go app/router/api_v0.go app/tasks/backup.go app/cmd/server.go
git commit -m "feat(backup): wire filestore through backup handler and task"
```

---

### Task 8: Update backup tests

**Files:**
- Modify: `internal/backup/export_test.go`
- Modify: `internal/backup/import_test.go`
- Modify: `app/router/handlers/backup/backup_test.go` (if it calls Export/Import directly)

**Step 1: Update `export_test.go`**

Add filestore setup to `TestExport`. After creating stores and before `sampleData`, add:

```go
	fileStore, err := filestore.New(db, filepath.Join(t.TempDir(), "attachments"), 10*1024*1024)
	if err != nil {
		t.Fatalf("unable to create filestore: %v", err)
	}
```

Add `"github.com/andresbott/etna/internal/filestore"` to imports.

Update `sampleData` to accept a `*filestore.Store` parameter and create an attachment on one transaction. Update signature:

```go
func sampleData(t *testing.T, store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, fileStore *filestore.Store) {
```

At the end of `sampleData`, after creating transactions, add an attachment to the first transaction (income i1, id=1):

```go
	// =========================================
	// Attach a file to the first transaction
	// =========================================
	if fileStore != nil {
		jpegContent := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0x00}, 50)...)
		attID, err := fileStore.SaveRaw(t.Context(), getDate("2022-01-20"), jpegContent, "receipt.jpg", "image/jpeg")
		if err != nil {
			t.Fatalf("error saving attachment: %v", err)
		}
		err = store.SetAttachmentID(t.Context(), 1, &attID)
		if err != nil {
			t.Fatalf("error setting attachment ID: %v", err)
		}
	}
```

Add `"bytes"` to imports if not present.

Update all `sampleData` call sites to pass the filestore (or `nil` where not available).

Update `export()` call in `TestExport`:

```go
err = export(t.Context(), store, mdStore, csvStore, fileStore, target)
```

Update `backupPayload` struct to include attachments:

```go
type backupPayload struct {
	// ... existing fields ...
	Attachments []attachmentV1
}
```

Update `readFromZip` to handle the new files — add a case for `attachmentsFile`:

```go
	case attachmentsFile:
		payload.Attachments, err = unmarshalJSON[[]attachmentV1](data)
```

Update the expected `want` in `TestExport` — the first transaction should now have an `AttachmentID`:

```go
{Id: 1, Description: "i1", Amount: 12.5, AccountID: 1, CategoryID: 1, Date: getDate("2022-01-20"), Type: txTypeIncome, AttachmentID: uintPtr(1)},
```

Add helper:

```go
func uintPtr(v uint) *uint { return &v }
```

Add attachment expectation:

```go
Attachments: []attachmentV1{
    {ID: 1, OriginalName: "receipt.jpg", MimeType: "image/jpeg", FileSize: 54, ZipPath: "attachments/1.jpg"},
},
```

Note: `FileSize` = 4 magic bytes + 50 zero bytes = 54.

Also verify that the binary file exists in the zip by checking that `readFromZip` doesn't error. For a more thorough check, add a binary content assertion after `readFromZip` — or keep it simple and just trust the manifest.

Update the `cmp.Diff` options to handle attachments:

```go
cmpopts.SortSlices(func(a, b attachmentV1) bool { return a.ID < b.ID }),
```

**Step 2: Update `TestRoundTrip`**

Update both DB setups to include filestore. Update `sampleData`, `export`, and `Import` calls:

```go
	// Source
	fileStore1, err := filestore.New(db1, filepath.Join(t.TempDir(), "att1"), 10*1024*1024)
	// ...
	sampleData(t, store1, mdStore1, csvStore1, fileStore1)
	// ...
	err = export(t.Context(), store1, mdStore1, csvStore1, fileStore1, target1)
	// ...

	// Destination
	fileStore2, err := filestore.New(db2, filepath.Join(t.TempDir(), "att2"), 10*1024*1024)
	// ...
	err = Import(t.Context(), store2, mdStore2, csvStore2, fileStore2, target1)
	// ...
	err = export(t.Context(), store2, mdStore2, csvStore2, fileStore2, target2)
```

Add attachment fields to the `cmpopts.IgnoreFields` for round-trip comparison:

```go
cmpopts.IgnoreFields(TransactionV1{}, "Id", "AccountID", "CategoryID", "OriginAccountID", "TargetAccountID", "InvestmentAccountID", "CashAccountID", "SourceAccountID", "InstrumentID", "AttachmentID"),
cmpopts.IgnoreFields(attachmentV1{}, "ID", "ZipPath"),
```

**Step 3: Update `import_test.go`**

Update `setupImportTest` to include filestore:

```go
type testStores struct {
	accounting *accounting.Store
	marketdata *marketdata.Store
	csvimport  *csvimport.Store
	filestore  *filestore.Store
}
```

Create filestore in setup and pass to `Import`:

```go
	fileStore, err := filestore.New(db, filepath.Join(t.TempDir(), "attachments"), 10*1024*1024)
	// ...
	err = Import(t.Context(), store, mdStore, csvStore, fileStore, backupFile)
	// ...
	return testStores{accounting: store, marketdata: mdStore, csvimport: csvStore, filestore: fileStore}
```

Update `sampleDataNoise` calls to pass `nil` for filestore.

NOTE: The `testdata/backup-v1.zip` fixture needs to be regenerated. Uncomment the `copyFile(target)` line in `TestExport`, run the test, then copy the generated file to `internal/backup/testdata/backup-v1.zip`.

**Step 4: Update `backup_test.go` handler test**

Check `app/router/handlers/backup/backup_test.go` for any direct calls to `backup.ExportToFile` or `backup.Import` and add the `fileStore` parameter (likely `nil` is fine for handler-level tests).

**Step 5: Run all tests**

Run: `go test ./internal/backup/... -v`
Run: `go test ./internal/filestore/... -v`
Run: `go test ./app/... -v`
Expected: all PASS

**Step 6: Regenerate test fixture**

In `TestExport`, temporarily uncomment `copyFile(target)`, run the test, copy the output to `internal/backup/testdata/backup-v1.zip`, then re-comment the line.

**Step 7: Run tests again with new fixture**

Run: `go test ./internal/backup/... -v`
Expected: all PASS

**Step 8: Commit**

```
git add internal/backup/ internal/filestore/ app/
git commit -m "test(backup): update tests for attachment backup/restore"
```

---

### Task 9: Final verification

**Step 1: Run full test suite**

Run: `go test ./... -count=1`
Expected: all PASS

**Step 2: Verify no lint issues**

Run: `make lint` (or whatever linter command the project uses)

**Step 3: Commit any final fixes**

---

## Key Design Decisions

1. **Nil filestore = skip attachments**: Both export and import gracefully handle `fileStore == nil`, meaning existing callers that don't have a filestore won't break.

2. **Old backups import fine**: The `attachments.json` manifest uses `omitempty` on `TransactionV1.AttachmentID`. Old zips without `attachments.json` are handled — `loadAttachmentManifest` returns an error that is swallowed, returning an empty map.

3. **Missing files don't fail export**: If an attachment file is missing on disk during export, we skip it with a `continue` rather than failing the entire backup.

4. **Binary files in zip**: Attachment files are stored as `attachments/<id>.<ext>` in the zip, alongside the JSON manifest at `attachments.json`.

5. **Schema version stays `"1.0.0"`**: No breaking changes — new fields are additive with `omitempty`.
