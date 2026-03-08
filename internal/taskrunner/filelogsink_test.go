package taskrunner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNewFileTaskLogSink(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "logs")

	sink, err := NewFileTaskLogSink(subDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sink == nil {
		t.Fatal("expected non-nil sink")
	}

	// Directory should exist
	info, err := os.Stat(subDir)
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected directory")
	}
}

func TestFileTaskLogSink_AppendAndRead(t *testing.T) {
	dir := t.TempDir()
	sink, err := NewFileTaskLogSink(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	id := uuid.New()
	ctx := context.Background()

	// Append multiple lines
	if err := sink.Append(ctx, id, "INFO", "first message"); err != nil {
		t.Fatalf("append error: %v", err)
	}
	if err := sink.Append(ctx, id, "ERROR", "second message"); err != nil {
		t.Fatalf("append error: %v", err)
	}

	// Read back with FileTaskLogReader
	reader := NewFileTaskLogReader(dir)
	log, err := reader.GetTaskLog(ctx, id)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	if !strings.Contains(log, "first message") {
		t.Errorf("log should contain 'first message', got: %s", log)
	}
	if !strings.Contains(log, "second message") {
		t.Errorf("log should contain 'second message', got: %s", log)
	}
	if !strings.Contains(log, "INFO") {
		t.Errorf("log should contain 'INFO', got: %s", log)
	}
	if !strings.Contains(log, "ERROR") {
		t.Errorf("log should contain 'ERROR', got: %s", log)
	}

	// Each append should produce a separate line
	lines := strings.Split(strings.TrimSpace(log), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestFileTaskLogReader_NonExistentLog(t *testing.T) {
	dir := t.TempDir()
	reader := NewFileTaskLogReader(dir)

	log, err := reader.GetTaskLog(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if log != "" {
		t.Errorf("expected empty string for non-existent log, got: %q", log)
	}
}

func TestFileTaskLogSink_RemoveTaskLogs(t *testing.T) {
	dir := t.TempDir()
	sink, err := NewFileTaskLogSink(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	id1 := uuid.New()
	id2 := uuid.New()

	// Write logs for two tasks
	if err := sink.Append(ctx, id1, "INFO", "task1 log"); err != nil {
		t.Fatal(err)
	}
	if err := sink.Append(ctx, id2, "INFO", "task2 log"); err != nil {
		t.Fatal(err)
	}

	// Remove only id1
	if err := sink.RemoveTaskLogs(ctx, []uuid.UUID{id1}); err != nil {
		t.Fatalf("remove error: %v", err)
	}

	// id1 log file should be gone
	path1 := filepath.Join(dir, id1.String()+".log")
	if _, err := os.Stat(path1); !os.IsNotExist(err) {
		t.Error("expected log file for id1 to be deleted")
	}

	// id2 log file should still exist
	path2 := filepath.Join(dir, id2.String()+".log")
	if _, err := os.Stat(path2); err != nil {
		t.Errorf("expected log file for id2 to exist: %v", err)
	}
}

func TestFileTaskLogSink_RemoveNonExistent(t *testing.T) {
	dir := t.TempDir()
	sink, err := NewFileTaskLogSink(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Removing non-existent logs should not error
	err = sink.RemoveTaskLogs(context.Background(), []uuid.UUID{uuid.New()})
	if err != nil {
		t.Fatalf("unexpected error removing non-existent logs: %v", err)
	}
}
