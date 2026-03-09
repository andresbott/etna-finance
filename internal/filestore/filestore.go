package filestore

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

var (
	ErrNotFound       = errors.New("attachment not found")
	ErrMimeNotAllowed = errors.New("mime type not allowed")
	ErrTooLarge       = errors.New("file too large")
)

var allowedMimeTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/webp":      true,
	"application/pdf": true,
}

// mimeToExt maps allowed MIME types to file extensions.
var mimeToExt = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/webp":      ".webp",
	"application/pdf": ".pdf",
}

type dbAttachment struct {
	Id           uint   `gorm:"primaryKey"`
	OriginalName string `gorm:"size:255;not null"`
	StoragePath  string `gorm:"size:512;not null"`
	MimeType     string `gorm:"size:100;not null"`
	FileSize     int64  `gorm:"not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// Attachment is the public metadata struct returned to callers.
type Attachment struct {
	Id           uint
	OriginalName string
	StoragePath  string
	MimeType     string
	FileSize     int64
}

// Store manages file attachments on disk and in the database.
type Store struct {
	db      *gorm.DB
	baseDir string
	maxSize int64
}

// New creates a new Store, running AutoMigrate for the attachment table.
func New(db *gorm.DB, baseDir string, maxSize int64) (*Store, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}
	if baseDir == "" {
		return nil, fmt.Errorf("baseDir cannot be empty")
	}

	err := db.AutoMigrate(&dbAttachment{})
	if err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	return &Store{
		db:      db,
		baseDir: baseDir,
		maxSize: maxSize,
	}, nil
}

// Save validates and stores a file, returning the attachment ID.
// The date parameter determines the storage directory structure (YYYY/MM/).
func (s *Store) Save(ctx context.Context, date time.Time, file multipart.File, header *multipart.FileHeader) (uint, error) {
	// Check file size from header
	if header.Size > s.maxSize {
		return 0, ErrTooLarge
	}

	// Read file content to detect MIME type and write to disk
	content, err := io.ReadAll(file)
	if err != nil {
		return 0, fmt.Errorf("reading file: %w", err)
	}

	// Double-check actual size
	if int64(len(content)) > s.maxSize {
		return 0, ErrTooLarge
	}

	// Detect MIME type from content
	mimeType := detectMimeType(content)
	if !allowedMimeTypes[mimeType] {
		return 0, ErrMimeNotAllowed
	}

	// Generate storage path: YYYY/MM/DD_<8-char-hex>.<ext>
	ext := mimeToExt[mimeType]
	randBytes := make([]byte, 4)
	if _, err := rand.Read(randBytes); err != nil {
		return 0, fmt.Errorf("generating random name: %w", err)
	}
	randHex := hex.EncodeToString(randBytes)

	storagePath := fmt.Sprintf("%04d/%02d/%02d_%s%s",
		date.Year(), date.Month(), date.Day(), randHex, ext)

	absPath := filepath.Join(s.baseDir, storagePath)

	// Create directories
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return 0, fmt.Errorf("creating directory: %w", err)
	}

	// Write file to disk
	if err := os.WriteFile(absPath, content, 0o600); err != nil {
		return 0, fmt.Errorf("writing file: %w", err)
	}

	// Insert DB record
	record := dbAttachment{
		OriginalName: header.Filename,
		StoragePath:  storagePath,
		MimeType:     mimeType,
		FileSize:     int64(len(content)),
	}
	if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
		// Clean up file on DB error
		_ = os.Remove(absPath)
		return 0, fmt.Errorf("inserting record: %w", err)
	}

	return record.Id, nil
}

// Get returns the attachment metadata for the given ID.
func (s *Store) Get(ctx context.Context, id uint) (*Attachment, error) {
	var record dbAttachment
	err := s.db.WithContext(ctx).First(&record, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("querying attachment: %w", err)
	}

	return &Attachment{
		Id:           record.Id,
		OriginalName: record.OriginalName,
		StoragePath:  record.StoragePath,
		MimeType:     record.MimeType,
		FileSize:     record.FileSize,
	}, nil
}

// GetFilePath returns the absolute file path for the given attachment ID.
// It prevents path traversal by verifying the resolved path is within baseDir.
func (s *Store) GetFilePath(ctx context.Context, id uint) (string, error) {
	att, err := s.Get(ctx, id)
	if err != nil {
		return "", err
	}

	absPath := filepath.Join(s.baseDir, att.StoragePath)
	absPath, err = filepath.Abs(absPath)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	// Prevent path traversal
	absBase, err := filepath.Abs(s.baseDir)
	if err != nil {
		return "", fmt.Errorf("resolving base: %w", err)
	}
	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) {
		return "", fmt.Errorf("path traversal detected")
	}

	return absPath, nil
}

// detectMimeType detects the MIME type from file content.
// It extends http.DetectContentType with WebP support (RIFF....WEBP signature).
func detectMimeType(content []byte) string {
	// Check for WebP: starts with "RIFF" and has "WEBP" at offset 8
	if len(content) >= 12 &&
		string(content[0:4]) == "RIFF" &&
		string(content[8:12]) == "WEBP" {
		return "image/webp"
	}
	return http.DetectContentType(content)
}

// Delete removes the file from disk and soft-deletes the DB record.
func (s *Store) Delete(ctx context.Context, id uint) error {
	var record dbAttachment
	err := s.db.WithContext(ctx).First(&record, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("querying attachment: %w", err)
	}

	// Remove file from disk (ignore error if already missing)
	absPath := filepath.Join(s.baseDir, record.StoragePath)
	_ = os.Remove(absPath)

	// Soft-delete DB record
	if err := s.db.WithContext(ctx).Delete(&record).Error; err != nil {
		return fmt.Errorf("deleting record: %w", err)
	}

	return nil
}
