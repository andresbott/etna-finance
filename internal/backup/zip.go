package backup

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

// zipWriter wraps a zip.Writer and its underlying file
type zipWriter struct {
	file   *os.File
	buffer *bytes.Buffer // optional: non-nil if writing in memory
	writer *zip.Writer
}

// Close closes both the zip writer and the underlying file.
func (zw *zipWriter) Close() error {
	if err := zw.writer.Close(); err != nil {
		return err
	}
	return zw.file.Close()
}

// createZipFile initializes the zipWriter
func createZipFile(dest string) (*zipWriter, error) {
	f, err := os.Create(dest)
	if err != nil {
		return nil, fmt.Errorf("failed to create zip file: %w", err)
	}

	zw := &zipWriter{
		file:   f,
		writer: zip.NewWriter(f),
	}
	return zw, nil
}

// createZipMemory creates an in-memory ZIP writer
func createZipMemory() *zipWriter {
	buf := new(bytes.Buffer)
	return &zipWriter{
		buffer: buf,
		writer: zip.NewWriter(buf),
	}
}

// writeJsonFile writes a JSON file into the provided zip.Writer.
func (zw *zipWriter) writeJsonFile(filename string, data any) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	f, err := zw.writer.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file in zip: %w", err)
	}

	if _, err := f.Write(jsonData); err != nil {
		return fmt.Errorf("failed to write JSON data: %w", err)
	}

	return nil
}
