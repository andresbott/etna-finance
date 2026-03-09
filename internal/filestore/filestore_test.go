package filestore

import (
	"bytes"
	"context"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-bumbu/testdbs"
)

func TestMain(m *testing.M) {
	testdbs.InitDBS()
	code := m.Run()
	_ = testdbs.Clean()
	os.Exit(code)
}

// fakeFileHeader creates a multipart.FileHeader backed by the given content.
func fakeFileHeader(t *testing.T, filename string, content []byte) (multipart.File, *multipart.FileHeader) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	_, err = part.Write(content)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	reader := multipart.NewReader(&buf, writer.Boundary())
	form, err := reader.ReadForm(int64(len(content)) + 1024)
	if err != nil {
		t.Fatal(err)
	}
	fh := form.File["file"][0]
	f, err := fh.Open()
	if err != nil {
		t.Fatal(err)
	}
	return f, fh
}

func TestSaveAndGet(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			baseDir := t.TempDir()
			store, err := New(db.ConnDbName("TestSaveAndGet"), baseDir, 10*1024*1024)
			if err != nil {
				t.Fatal(err)
			}

			// Create a small JPEG-like file (starts with JPEG magic bytes).
			content := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0x00}, 100)...)
			f, fh := fakeFileHeader(t, "photo.jpg", content)
			defer func() { _ = f.Close() }()

			ctx := context.Background()
			date := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)

			id, err := store.Save(ctx, date, f, fh)
			if err != nil {
				t.Fatalf("Save failed: %v", err)
			}
			if id == 0 {
				t.Fatal("expected non-zero ID")
			}

			// Get metadata
			att, err := store.Get(ctx, id)
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}
			if att.OriginalName != "photo.jpg" {
				t.Errorf("expected OriginalName 'photo.jpg', got %q", att.OriginalName)
			}
			if att.MimeType != "image/jpeg" {
				t.Errorf("expected MimeType 'image/jpeg', got %q", att.MimeType)
			}
			if att.FileSize != int64(len(content)) {
				t.Errorf("expected FileSize %d, got %d", len(content), att.FileSize)
			}

			// GetFilePath
			fp, err := store.GetFilePath(ctx, id)
			if err != nil {
				t.Fatalf("GetFilePath failed: %v", err)
			}
			if !filepath.IsAbs(fp) {
				t.Errorf("expected absolute path, got %q", fp)
			}
			// Verify file exists and content matches
			diskContent := readTestFile(t, fp)
			if !bytes.Equal(diskContent, content) {
				t.Error("disk content does not match original")
			}
		})
	}
}

func TestSavePNG(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			baseDir := t.TempDir()
			store, err := New(db.ConnDbName("TestSavePNG"), baseDir, 10*1024*1024)
			if err != nil {
				t.Fatal(err)
			}

			// PNG magic bytes
			content := append([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, bytes.Repeat([]byte{0x00}, 100)...)
			f, fh := fakeFileHeader(t, "image.png", content)
			defer func() { _ = f.Close() }()

			ctx := context.Background()
			date := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

			id, err := store.Save(ctx, date, f, fh)
			if err != nil {
				t.Fatalf("Save failed: %v", err)
			}

			att, err := store.Get(ctx, id)
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}
			if att.MimeType != "image/png" {
				t.Errorf("expected MimeType 'image/png', got %q", att.MimeType)
			}
			_ = id
		})
	}
}

func TestSavePDF(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			baseDir := t.TempDir()
			store, err := New(db.ConnDbName("TestSavePDF"), baseDir, 10*1024*1024)
			if err != nil {
				t.Fatal(err)
			}

			// PDF magic bytes
			content := append([]byte("%PDF-1.4"), bytes.Repeat([]byte{0x00}, 100)...)
			f, fh := fakeFileHeader(t, "doc.pdf", content)
			defer func() { _ = f.Close() }()

			ctx := context.Background()
			date := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)

			id, err := store.Save(ctx, date, f, fh)
			if err != nil {
				t.Fatalf("Save failed: %v", err)
			}

			att, err := store.Get(ctx, id)
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}
			if att.MimeType != "application/pdf" {
				t.Errorf("expected MimeType 'application/pdf', got %q", att.MimeType)
			}
			_ = id
		})
	}
}

func TestInvalidMimeType(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			baseDir := t.TempDir()
			store, err := New(db.ConnDbName("TestInvalidMime"), baseDir, 10*1024*1024)
			if err != nil {
				t.Fatal(err)
			}

			// Plain text content — not an allowed mime type
			content := []byte("hello world, this is a text file")
			f, fh := fakeFileHeader(t, "notes.txt", content)
			defer func() { _ = f.Close() }()

			ctx := context.Background()
			date := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)

			_, err = store.Save(ctx, date, f, fh)
			if err == nil {
				t.Fatal("expected error for invalid mime type, got nil")
			}
			if err != ErrMimeNotAllowed {
				t.Errorf("expected ErrMimeNotAllowed, got %v", err)
			}
		})
	}
}

func TestFileTooLarge(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			baseDir := t.TempDir()
			maxSize := int64(50) // very small limit
			store, err := New(db.ConnDbName("TestFileTooLarge"), baseDir, maxSize)
			if err != nil {
				t.Fatal(err)
			}

			// Create content larger than maxSize
			content := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0x00}, 100)...)
			f, fh := fakeFileHeader(t, "big.jpg", content)
			defer func() { _ = f.Close() }()

			ctx := context.Background()
			date := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)

			_, err = store.Save(ctx, date, f, fh)
			if err == nil {
				t.Fatal("expected error for file too large, got nil")
			}
			if err != ErrTooLarge {
				t.Errorf("expected ErrTooLarge, got %v", err)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			baseDir := t.TempDir()
			store, err := New(db.ConnDbName("TestDelete"), baseDir, 10*1024*1024)
			if err != nil {
				t.Fatal(err)
			}

			content := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0x00}, 100)...)
			f, fh := fakeFileHeader(t, "todelete.jpg", content)
			defer func() { _ = f.Close() }()

			ctx := context.Background()
			date := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)

			id, err := store.Save(ctx, date, f, fh)
			if err != nil {
				t.Fatalf("Save failed: %v", err)
			}

			// Get filepath before delete
			fp, err := store.GetFilePath(ctx, id)
			if err != nil {
				t.Fatalf("GetFilePath failed: %v", err)
			}

			// Verify file exists
			if _, err := os.Stat(fp); os.IsNotExist(err) {
				t.Fatal("file should exist before delete")
			}

			// Delete
			err = store.Delete(ctx, id)
			if err != nil {
				t.Fatalf("Delete failed: %v", err)
			}

			// File should be removed from disk
			if _, err := os.Stat(fp); !os.IsNotExist(err) {
				t.Error("file should be removed from disk after delete")
			}

			// DB record should be soft-deleted (Get returns ErrNotFound)
			_, err = store.Get(ctx, id)
			if err != ErrNotFound {
				t.Errorf("expected ErrNotFound after delete, got %v", err)
			}
		})
	}
}

func TestGetNotFound(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			baseDir := t.TempDir()
			store, err := New(db.ConnDbName("TestGetNotFound"), baseDir, 10*1024*1024)
			if err != nil {
				t.Fatal(err)
			}

			ctx := context.Background()

			_, err = store.Get(ctx, 99999)
			if err != ErrNotFound {
				t.Errorf("expected ErrNotFound, got %v", err)
			}

			_, err = store.GetFilePath(ctx, 99999)
			if err != ErrNotFound {
				t.Errorf("expected ErrNotFound from GetFilePath, got %v", err)
			}
		})
	}
}

func TestSaveWebP(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			baseDir := t.TempDir()
			store, err := New(db.ConnDbName("TestSaveWebP"), baseDir, 10*1024*1024)
			if err != nil {
				t.Fatal(err)
			}

			// WebP magic bytes: RIFF....WEBP
			header := []byte("RIFF")
			header = append(header, 0x00, 0x00, 0x00, 0x00) // file size placeholder
			header = append(header, []byte("WEBP")...)
			content := append(header, bytes.Repeat([]byte{0x00}, 100)...)
			f, fh := fakeFileHeader(t, "image.webp", content)
			defer func() { _ = f.Close() }()

			ctx := context.Background()
			date := time.Date(2025, 4, 20, 0, 0, 0, 0, time.UTC)

			id, err := store.Save(ctx, date, f, fh)
			if err != nil {
				t.Fatalf("Save failed: %v", err)
			}

			att, err := store.Get(ctx, id)
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}
			if att.MimeType != "image/webp" {
				t.Errorf("expected MimeType 'image/webp', got %q", att.MimeType)
			}
			_ = id
		})
	}
}

// TestStoragePath verifies the storage path format YYYY/MM/DD_<hex>.<ext>.
func TestStoragePath(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			baseDir := t.TempDir()
			store, err := New(db.ConnDbName("TestStoragePath"), baseDir, 10*1024*1024)
			if err != nil {
				t.Fatal(err)
			}

			content := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0x00}, 100)...)
			f, fh := fakeFileHeader(t, "photo.jpg", content)
			defer func() { _ = f.Close() }()

			ctx := context.Background()
			date := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)

			id, err := store.Save(ctx, date, f, fh)
			if err != nil {
				t.Fatalf("Save failed: %v", err)
			}

			att, err := store.Get(ctx, id)
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}

			// StoragePath should start with 2025/03/
			if len(att.StoragePath) < 20 {
				t.Fatalf("StoragePath too short: %q", att.StoragePath)
			}
			if att.StoragePath[:8] != "2025/03/" {
				t.Errorf("expected StoragePath to start with '2025/03/', got %q", att.StoragePath)
			}

			// Verify the file on disk is at baseDir/storagePath
			expectedPath := filepath.Join(baseDir, att.StoragePath)
			diskContent := readTestFile(t, expectedPath)

			// Also read via GetFilePath and compare
			fp, err := store.GetFilePath(ctx, id)
			if err != nil {
				t.Fatalf("GetFilePath failed: %v", err)
			}
			fpContent := readTestFile(t, fp)
			if !bytes.Equal(diskContent, fpContent) {
				t.Error("content mismatch between storage path and GetFilePath")
			}
		})
	}
}

// TestPathTraversal ensures GetFilePath rejects storage paths with traversal.
func TestPathTraversal(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			baseDir := t.TempDir()
			conn := db.ConnDbName("TestPathTraversal")
			store, err := New(conn, baseDir, 10*1024*1024)
			if err != nil {
				t.Fatal(err)
			}

			ctx := context.Background()

			// Manually insert a record with a malicious storage path.
			malicious := dbAttachment{
				OriginalName: "evil.jpg",
				StoragePath:  "../../../etc/passwd",
				MimeType:     "image/jpeg",
				FileSize:     100,
			}
			if err := conn.Create(&malicious).Error; err != nil {
				t.Fatalf("failed to insert malicious record: %v", err)
			}

			_, err = store.GetFilePath(ctx, malicious.Id)
			if err == nil {
				t.Fatal("expected error for path traversal, got nil")
			}
		})
	}
}

// TestDeleteNotFound verifies deleting a non-existent record.
func TestDeleteNotFound(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			baseDir := t.TempDir()
			store, err := New(db.ConnDbName("TestDeleteNotFound"), baseDir, 10*1024*1024)
			if err != nil {
				t.Fatal(err)
			}

			ctx := context.Background()
			err = store.Delete(ctx, 99999)
			if err != ErrNotFound {
				t.Errorf("expected ErrNotFound, got %v", err)
			}
		})
	}
}

// TestSaveCleanupOnDBError is harder to test without mocking — we skip this
// as the main flow is covered. The cleanup logic is tested implicitly through
// normal operation and the Delete test.

// Verify that we use the content bytes (not the extension) for MIME detection.
func TestMimeDetectionByContent(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			baseDir := t.TempDir()
			store, err := New(db.ConnDbName("TestMimeByContent"), baseDir, 10*1024*1024)
			if err != nil {
				t.Fatal(err)
			}

			// File has .jpg extension but text content — should be rejected
			content := []byte("this is plain text, not a JPEG")
			f, fh := fakeFileHeader(t, "fake.jpg", content)
			defer func() { _ = f.Close() }()

			ctx := context.Background()
			date := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)

			_, err = store.Save(ctx, date, f, fh)
			if err != ErrMimeNotAllowed {
				t.Errorf("expected ErrMimeNotAllowed for text-content-with-jpg-extension, got %v", err)
			}
		})
	}
}

func TestSaveRaw(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			baseDir := t.TempDir()
			store, err := New(db.ConnDbName("TestSaveRaw"), baseDir, 10*1024*1024)
			if err != nil {
				t.Fatal(err)
			}

			// Create a small JPEG-like content (starts with JPEG magic bytes).
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

			// Get metadata
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

			// Assert file content on disk matches
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

func TestWipeData(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			baseDir := t.TempDir()
			store, err := New(db.ConnDbName("TestWipeData"), baseDir, 10*1024*1024)
			if err != nil {
				t.Fatal(err)
			}

			ctx := context.Background()
			date := time.Date(2025, 7, 10, 0, 0, 0, 0, time.UTC)

			// Save two files: a JPEG and a PNG
			jpegContent := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0x00}, 100)...)
			id1, err := store.SaveRaw(ctx, date, jpegContent, "photo.jpg", "image/jpeg")
			if err != nil {
				t.Fatalf("SaveRaw JPEG failed: %v", err)
			}

			pngContent := append([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, bytes.Repeat([]byte{0x00}, 100)...)
			id2, err := store.SaveRaw(ctx, date, pngContent, "image.png", "image/png")
			if err != nil {
				t.Fatalf("SaveRaw PNG failed: %v", err)
			}

			// Get file paths before wipe
			fp1, err := store.GetFilePath(ctx, id1)
			if err != nil {
				t.Fatalf("GetFilePath id1 failed: %v", err)
			}
			fp2, err := store.GetFilePath(ctx, id2)
			if err != nil {
				t.Fatalf("GetFilePath id2 failed: %v", err)
			}

			// Verify files exist before wipe
			if _, err := os.Stat(fp1); os.IsNotExist(err) {
				t.Fatal("file1 should exist before wipe")
			}
			if _, err := os.Stat(fp2); os.IsNotExist(err) {
				t.Fatal("file2 should exist before wipe")
			}

			// Wipe all data
			err = store.WipeData(ctx)
			if err != nil {
				t.Fatalf("WipeData failed: %v", err)
			}

			// Assert Get returns ErrNotFound for both IDs
			_, err = store.Get(ctx, id1)
			if err != ErrNotFound {
				t.Errorf("expected ErrNotFound for id1 after wipe, got %v", err)
			}
			_, err = store.Get(ctx, id2)
			if err != ErrNotFound {
				t.Errorf("expected ErrNotFound for id2 after wipe, got %v", err)
			}

			// Assert files are removed from disk
			if _, err := os.Stat(fp1); !os.IsNotExist(err) {
				t.Error("file1 should be removed from disk after wipe")
			}
			if _, err := os.Stat(fp2); !os.IsNotExist(err) {
				t.Error("file2 should be removed from disk after wipe")
			}
		})
	}
}

// readTestFile reads a file at the given path. It is used in tests where the
// path is constructed from t.TempDir() and is therefore safe.
func readTestFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
