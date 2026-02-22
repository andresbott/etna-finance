package taskrunner

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-bumbu/tempo"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// dbTaskExecution is the GORM model for persisted task executions.
// It implements tempo's persistence by storing task state in the database.
type dbTaskExecution struct {
	ID        string    `gorm:"primaryKey;type:text;column:id"`
	Name      string    `gorm:"not null;index;column:name"`
	Status    int       `gorm:"not null;column:status"`
	QueuedAt  time.Time `gorm:"not null;index;column:queued_at"`
	StartedAt time.Time `gorm:"column:started_at"`
	EndedAt   time.Time `gorm:"column:ended_at"`
}

func (dbTaskExecution) TableName() string { return "task_executions" }

// TaskExecutionStore persists task executions to the database and implements
// tempo.TaskStatePersistence and tempo.RecoverablePersistence so the queue
// can mirror state and recover after restart.
type TaskExecutionStore struct {
	db      *gorm.DB
	logger  *slog.Logger
	cleaner TaskLogCleaner // optional; called from RemoveTasks to delete task log files
}

// NewTaskExecutionStore creates a store and runs AutoMigrate for task_executions.
// On startup, any task left in Running status (e.g. after a crash) is marked Failed
// so it can be cleared and does not block stop/cancel.
// Logger is used for startup warnings; if nil, a discard logger is used.
// cleaner, when set, is called from RemoveTasks so task log files (e.g. FileTaskLogSink) can be deleted.
func NewTaskExecutionStore(db *gorm.DB, logger *slog.Logger, cleaner TaskLogCleaner) (*TaskExecutionStore, error) {
	if db == nil {
		return nil, nil
	}
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	if err := db.AutoMigrate(&dbTaskExecution{}); err != nil {
		return nil, err
	}
	ctx := context.Background()
	now := time.Now()
	res := db.WithContext(ctx).Model(&dbTaskExecution{}).
		Where("status = ?", int(tempo.TaskStatusRunning)).
		Updates(map[string]interface{}{
			"status":   int(tempo.TaskStatusFailed),
			"ended_at": now,
		})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected > 0 {
		logger.Warn("marked leftover running task(s) as failed on startup (e.g. after crash)",
			slog.String("component", "taskrunner"),
			slog.Int64("count", res.RowsAffected))
	}
	return &TaskExecutionStore{db: db, logger: logger, cleaner: cleaner}, nil
}

// SaveTask implements tempo.TaskStatePersistence. Upserts by id.
func (s *TaskExecutionStore) SaveTask(ctx context.Context, task tempo.TaskInfo) error {
	row := dbTaskExecution{
		ID:        task.ID.String(),
		Name:      task.Name,
		Status:    int(task.Status),
		QueuedAt:  task.QueuedAt,
		StartedAt: task.StartedAt,
		EndedAt:   task.EndedAt,
	}
	return s.db.WithContext(ctx).Save(&row).Error
}

// RemoveTasks implements tempo.TaskStatePersistence. Also calls the optional TaskLogCleaner to delete task log files.
func (s *TaskExecutionStore) RemoveTasks(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	s.logger.Debug("RemoveTasks called", slog.String("component", "taskrunner"), slog.Int("count", len(ids)), slog.Any("ids", ids))
	strIDs := make([]string, len(ids))
	for i, id := range ids {
		strIDs[i] = id.String()
	}
	if err := s.db.WithContext(ctx).Where("id IN ?", strIDs).Delete(&dbTaskExecution{}).Error; err != nil {
		return err
	}
	if s.cleaner != nil {
		if err := s.cleaner.RemoveTaskLogs(ctx, ids); err != nil {
			return err
		}
	}
	return nil
}

// List implements tempo.RecoverablePersistence. Returns all stored tasks for queue recovery.
func (s *TaskExecutionStore) List(ctx context.Context) ([]tempo.TaskInfo, error) {
	var rows []dbTaskExecution
	if err := s.db.WithContext(ctx).Order("queued_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]tempo.TaskInfo, len(rows))
	for i := range rows {
		id, _ := uuid.Parse(rows[i].ID)
		out[i] = tempo.TaskInfo{
			ID:        id,
			Name:      rows[i].Name,
			Status:    tempo.TaskStatus(rows[i].Status),
			QueuedAt:  rows[i].QueuedAt,
			StartedAt: rows[i].StartedAt,
			EndedAt:   rows[i].EndedAt,
		}
	}
	return out, nil
}

// Ensure TaskExecutionStore implements the interfaces at compile time.
var (
	_ tempo.TaskStatePersistence   = (*TaskExecutionStore)(nil)
	_ tempo.RecoverablePersistence = (*TaskExecutionStore)(nil)
)
