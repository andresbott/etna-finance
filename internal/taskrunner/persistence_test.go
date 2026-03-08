package taskrunner

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/go-bumbu/tempo"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestExecDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	return db
}

func TestNewTaskExecutionStore(t *testing.T) {
	db := newTestExecDB(t)
	l := slog.New(slog.DiscardHandler)
	store, err := NewTaskExecutionStore(db, l, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestNewTaskExecutionStore_NilDB(t *testing.T) {
	store, err := NewTaskExecutionStore(nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error for nil db: %v", err)
	}
	if store != nil {
		t.Fatal("expected nil store for nil db")
	}
}

func TestNewTaskExecutionStore_NilLogger(t *testing.T) {
	db := newTestExecDB(t)
	store, err := NewTaskExecutionStore(db, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestTaskExecutionStore_SaveAndList(t *testing.T) {
	db := newTestExecDB(t)
	store, _ := NewTaskExecutionStore(db, nil, nil)

	ctx := context.Background()
	now := time.Now()
	id1 := uuid.New()
	id2 := uuid.New()

	task1 := tempo.TaskInfo{
		ID:       id1,
		Name:     "task-a",
		Status:   tempo.TaskStatusComplete,
		QueuedAt: now.Add(-2 * time.Minute),
		EndedAt:  now.Add(-1 * time.Minute),
	}
	task2 := tempo.TaskInfo{
		ID:       id2,
		Name:     "task-b",
		Status:   tempo.TaskStatusWaiting,
		QueuedAt: now,
	}

	if err := store.SaveTask(ctx, task1); err != nil {
		t.Fatalf("save task1 error: %v", err)
	}
	if err := store.SaveTask(ctx, task2); err != nil {
		t.Fatalf("save task2 error: %v", err)
	}

	list, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(list))
	}

	// List is ordered by queued_at ASC
	if list[0].ID != id1 {
		t.Errorf("expected first task to be id1")
	}
	if list[1].ID != id2 {
		t.Errorf("expected second task to be id2")
	}
	if list[0].Status != tempo.TaskStatusComplete {
		t.Errorf("expected status complete, got %v", list[0].Status)
	}
}

func TestTaskExecutionStore_SaveTask_Upsert(t *testing.T) {
	db := newTestExecDB(t)
	store, _ := NewTaskExecutionStore(db, nil, nil)
	ctx := context.Background()

	id := uuid.New()
	now := time.Now()

	// Save as waiting
	task := tempo.TaskInfo{
		ID:       id,
		Name:     "upsert-task",
		Status:   tempo.TaskStatusWaiting,
		QueuedAt: now,
	}
	if err := store.SaveTask(ctx, task); err != nil {
		t.Fatal(err)
	}

	// Update to running
	task.Status = tempo.TaskStatusRunning
	task.StartedAt = now.Add(time.Second)
	if err := store.SaveTask(ctx, task); err != nil {
		t.Fatal(err)
	}

	list, _ := store.List(ctx)
	if len(list) != 1 {
		t.Fatalf("expected 1 task after upsert, got %d", len(list))
	}
	if list[0].Status != tempo.TaskStatusRunning {
		t.Errorf("expected running status, got %v", list[0].Status)
	}
}

func TestTaskExecutionStore_RemoveTasks(t *testing.T) {
	db := newTestExecDB(t)
	store, _ := NewTaskExecutionStore(db, nil, nil)
	ctx := context.Background()

	id1 := uuid.New()
	id2 := uuid.New()
	now := time.Now()

	_ = store.SaveTask(ctx, tempo.TaskInfo{ID: id1, Name: "t1", Status: tempo.TaskStatusComplete, QueuedAt: now})
	_ = store.SaveTask(ctx, tempo.TaskInfo{ID: id2, Name: "t2", Status: tempo.TaskStatusComplete, QueuedAt: now})

	// Remove id1 only
	if err := store.RemoveTasks(ctx, []uuid.UUID{id1}); err != nil {
		t.Fatalf("remove error: %v", err)
	}

	list, _ := store.List(ctx)
	if len(list) != 1 {
		t.Fatalf("expected 1 task remaining, got %d", len(list))
	}
	if list[0].ID != id2 {
		t.Errorf("expected id2 to remain")
	}
}

func TestTaskExecutionStore_RemoveTasks_Empty(t *testing.T) {
	db := newTestExecDB(t)
	store, _ := NewTaskExecutionStore(db, nil, nil)

	// Removing empty list should not error
	if err := store.RemoveTasks(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskExecutionStore_RemoveTasks_WithCleaner(t *testing.T) {
	db := newTestExecDB(t)
	dir := t.TempDir()
	sink, _ := NewFileTaskLogSink(dir)

	store, _ := NewTaskExecutionStore(db, nil, sink)
	ctx := context.Background()

	id := uuid.New()
	now := time.Now()

	// Save a task and write a log
	_ = store.SaveTask(ctx, tempo.TaskInfo{ID: id, Name: "logged-task", Status: tempo.TaskStatusComplete, QueuedAt: now})
	_ = sink.Append(ctx, id, "INFO", "some log line")

	// Remove should also clean up log files
	if err := store.RemoveTasks(ctx, []uuid.UUID{id}); err != nil {
		t.Fatalf("remove error: %v", err)
	}

	// Verify log file is gone
	reader := NewFileTaskLogReader(dir)
	log, _ := reader.GetTaskLog(ctx, id)
	if log != "" {
		t.Errorf("expected empty log after remove, got %q", log)
	}
}

func TestTaskExecutionStore_MarksRunningAsFailedOnStartup(t *testing.T) {
	db := newTestExecDB(t)

	// Manually create the table and insert a running task
	if err := db.AutoMigrate(&dbTaskExecution{}); err != nil {
		t.Fatal(err)
	}
	id := uuid.New()
	row := dbTaskExecution{
		ID:        id.String(),
		Name:      "crashed-task",
		Status:    int(tempo.TaskStatusRunning),
		QueuedAt:  time.Now().Add(-5 * time.Minute),
		StartedAt: time.Now().Add(-4 * time.Minute),
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}

	// Creating the store should mark running tasks as failed
	store, err := NewTaskExecutionStore(db, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	list, _ := store.List(context.Background())
	if len(list) != 1 {
		t.Fatalf("expected 1 task, got %d", len(list))
	}
	if list[0].Status != tempo.TaskStatusFailed {
		t.Errorf("expected failed status, got %v", list[0].Status)
	}
}
