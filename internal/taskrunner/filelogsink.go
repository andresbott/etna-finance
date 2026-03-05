package taskrunner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-bumbu/tempo"
	"github.com/google/uuid"
)

// TaskLogGetter returns the plain-text log for a task execution (e.g. for API or UI).
type TaskLogGetter interface {
	GetTaskLog(ctx context.Context, executionID uuid.UUID) (string, error)
}

// FileTaskLogReader reads task log files from a directory (one file per execution: {id}.log).
// Use the same directory as FileTaskLogSink (e.g. DataDir/tasklogs) so logs written by the sink can be read.
func NewFileTaskLogReader(dir string) TaskLogGetter {
	return &fileTaskLogReader{dir: dir}
}

type fileTaskLogReader struct {
	dir string
}

func (f *fileTaskLogReader) GetTaskLog(ctx context.Context, executionID uuid.UUID) (string, error) {
	path := filepath.Join(f.dir, executionID.String()+".log")
	b, err := os.ReadFile(path) //nolint:gosec // path is internal, constructed from trusted uuid
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(b), nil
}

// TaskLogCleaner is called when tasks are removed from persistence so log storage can be cleaned (e.g. delete log files).
type TaskLogCleaner interface {
	RemoveTaskLogs(ctx context.Context, ids []uuid.UUID) error
}

// FileTaskLogSink writes task logs to plain text files on disk, one file per task (taskID.log).
// It implements tempo.TaskLogSink and TaskLogCleaner. Minimum log level is defined by the runner's LogLevel (system log level).
type FileTaskLogSink struct {
	dir string
	mu  sync.Mutex
}

// NewFileTaskLogSink creates a file-based task log sink. dir is the directory for log files; it is created if missing.
func NewFileTaskLogSink(dir string) (*FileTaskLogSink, error) {
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("task log dir: %w", err)
	}
	return &FileTaskLogSink{dir: dir}, nil
}

// Append implements tempo.TaskLogSink. Writes a single plain-text line: timestamp LEVEL message.
func (f *FileTaskLogSink) Append(ctx context.Context, taskID uuid.UUID, level string, msg string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	path := filepath.Join(f.dir, taskID.String()+".log")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644) //nolint:gosec // path is internal, constructed from trusted uuid
	if err != nil {
		return err
	}
	line := fmt.Sprintf("%s %s %s\n", time.Now().UTC().Format(time.RFC3339Nano), level, msg)
	_, err = file.WriteString(line)
	_ = file.Close()
	return err
}

// RemoveTaskLogs implements TaskLogCleaner. Deletes the log file for each task ID; missing files are ignored.
func (f *FileTaskLogSink) RemoveTaskLogs(ctx context.Context, ids []uuid.UUID) error {
	for _, id := range ids {
		path := filepath.Join(f.dir, id.String()+".log")
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// Ensure FileTaskLogSink implements the interfaces at compile time.
var (
	_ tempo.TaskLogSink = (*FileTaskLogSink)(nil)
	_ TaskLogCleaner    = (*FileTaskLogSink)(nil)
)
