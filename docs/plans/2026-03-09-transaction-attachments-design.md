# Transaction File Attachments — Design

## Overview

Allow attaching a single file (image or PDF) to any cash-type transaction. Files are stored on the server filesystem with metadata in a separate DB table. The transaction table holds a nullable FK to the attachment. A configurable max file size defaults to 10 MB.

## Decisions

- **Single file per transaction** — keeps UI and data model simple.
- **Allowed types:** JPEG, PNG, WebP, PDF.
- **FK direction:** `db_transactions.attachment_id` points to `db_attachments.id`. The FileStore package has no knowledge of transactions.
- **Separate endpoints** for file operations (not embedded in transaction JSON payloads).
- **Browser-native viewing** — clicking the attachment icon opens the file in a new tab.
- **File input in dialogs** — attachment is managed inside existing create/edit dialogs.
- **Config:** `MaxAttachmentSizeMB float64` in `AppSettings`, default `10.0`.

## Architecture

### FileStore Package (`internal/filestore`)

Fully isolated package. Owns the `db_attachments` table and filesystem operations. No concept of transactions.

**DB Model — `db_attachments`:**

| Column         | Type           | Notes                                      |
|----------------|----------------|--------------------------------------------|
| `id`           | uint (PK)      |                                            |
| `original_name`| string         | User's original filename                   |
| `storage_path` | string         | Relative path: `YYYY/MM/DD_<random>.<ext>` |
| `mime_type`    | string         | e.g. `image/jpeg`, `application/pdf`       |
| `file_size`    | int64          | Bytes                                      |
| `created_at`   | timestamp      |                                            |
| `updated_at`   | timestamp      |                                            |
| `deleted_at`   | timestamp      | Soft delete                                |

**Store struct:**

```go
type Store struct {
    db      *gorm.DB
    baseDir string // absolute path: <dataDir>/attachments
    maxSize int64  // bytes, converted from config MB
}
```

**Methods:**

- `Save(ctx, date time.Time, file multipart.File, header *multipart.FileHeader) (uint, error)` — validate mime type and size, generate storage path from date, write file to disk, insert DB row, return attachment ID.
- `Get(ctx, id uint) (*Attachment, error)` — return metadata.
- `GetFilePath(ctx, id uint) (string, error)` — resolve absolute disk path for serving.
- `Delete(ctx, id uint) error` — remove file from disk, soft-delete DB row.

**File path generation:**

```
<baseDir>/2026/03/09_a8f3c1b2.pdf
```

Format: `YYYY/MM/DD_<8-char-random-hex>.<ext>`. Year and month are subdirectories. The date comes from the transaction date passed during `Save()`.

**Allowed MIME types:** `image/jpeg`, `image/png`, `image/webp`, `application/pdf`.

### Accounting Store Changes

**`dbTransaction` change:**

Add `AttachmentID *uint` — nullable column. GORM AutoMigrate adds it with NULL default; existing transactions are unaffected.

**Lifecycle:**

- **Create with attachment:** Create transaction, then call `fileStore.Save()`, then update transaction's `AttachmentID`.
- **Delete transaction:** If `AttachmentID` is set, call `fileStore.Delete()` before/after deleting the transaction.
- **Replace attachment:** Delete old via `fileStore.Delete()`, save new via `fileStore.Save()`, update `AttachmentID`.
- **Remove attachment:** Call `fileStore.Delete()`, set `AttachmentID` to nil.

### API Endpoints

| Method   | Path                              | Description                  |
|----------|-----------------------------------|------------------------------|
| `POST`   | `/fin/entries/{id}/attachment`    | Upload file (multipart form) |
| `GET`    | `/fin/entries/{id}/attachment`    | Serve file inline            |
| `DELETE` | `/fin/entries/{id}/attachment`    | Remove attachment            |

**POST** — Multipart form with `file` field. Validates the transaction exists, checks for existing attachment (reject or replace), calls `fileStore.Save()`, updates `AttachmentID` on the transaction. Returns attachment metadata JSON.

**GET** — Looks up `AttachmentID` on the transaction, resolves the file path via `fileStore.GetFilePath()`, serves with `Content-Type` from mime type and `Content-Disposition: inline`.

**DELETE** — Looks up `AttachmentID`, calls `fileStore.Delete()`, sets `AttachmentID` to nil on the transaction.

### Frontend Changes

**Transaction list (EntriesTable.vue / AccountEntriesTable.vue):**

- API response includes `attachmentId` (nullable) per transaction.
- Paperclip icon (`pi pi-paperclip`) in actions column, visible when `attachmentId` is set.
- Click opens `GET /fin/entries/{id}/attachment` in a new browser tab via `window.open`.

**Dialogs (IncomeExpenseDialog, TransferDialog, BalanceStatusDialog):**

- File input field at the bottom of each cash-type dialog.
- HTML file input with `accept=".jpg,.jpeg,.png,.webp,.pdf"`.
- Shows selected filename + remove button.
- When editing with existing attachment: shows filename with view/replace/remove options.
- Upload is a follow-up request after transaction create/update succeeds.

**API client (`webui/src/lib/api/Attachment.ts`):**

- `uploadAttachment(txId: number, file: File): Promise<AttachmentMeta>` — POST multipart.
- `getAttachmentUrl(txId: number): string` — returns URL string for `window.open`.
- `deleteAttachment(txId: number): Promise<void>` — DELETE.

### Configuration

**`AppSettings` change:**

```go
MaxAttachmentSizeMB float64 // default 10.0
```

**Config YAML:**

```yaml
Settings:
  MaxAttachmentSizeMB: 10
```

Converted to bytes at startup: `int64(cfg.MaxAttachmentSizeMB * 1024 * 1024)`.

**Data directory:**

Add `attachments/` subdirectory in `initDataDir()`:

```
data/
├── carbon.db
├── attachments/    <-- new
├── backup/
├── sessions/
└── tasklogs/
```

### Wiring

- `filestore.Store` constructed in server startup with DB, `<dataDir>/attachments`, and max size.
- Injected into the finance handler alongside `accounting.Store`.
- Finance handler uses both stores to coordinate transaction + attachment lifecycle.
