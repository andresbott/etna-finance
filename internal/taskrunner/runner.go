package taskrunner

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/go-bumbu/tempo"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrQueueFull is returned when the task queue has no capacity (re-exported from tempo).
// Handlers should use errors.Is(err, ErrQueueFull) and return 429 to the client.
var ErrQueueFull = tempo.ErrQueueFull

// Runner manages background task execution using a tempo QueueRunner.
// It is agnostic of task types; concrete tasks are defined and enqueued by the application layer.
type Runner struct {
	queue  *tempo.QueueRunner
	logger *slog.Logger
}

// Cfg holds the configuration for the task runner.
type Cfg struct {
	// Parallelism is the number of concurrent workers. Defaults to 1.
	Parallelism int
	// QueueSize is the maximum number of pending tasks. Defaults to 20.
	QueueSize int
	// HistorySize is the maximum number of completed tasks to keep when doing cleanup Defaults to 20.
	HistorySize int
	// Logger is used for task lifecycle logging. If nil, logging is disabled.
	Logger *slog.Logger
	// DB, when set, is used to persist task executions (task_executions table).
	// If nil, tasks are kept in memory only and are lost on restart.
	DB *gorm.DB
	// LogSink, when set, receives log lines from tasks (tempo.Logger(ctx).InfoContext(ctx, "msg")).
	// Use tempo.NewMemTaskLogSink() for in-memory or implement tempo.TaskLogSink for DB.
	// If LogDir is set, a FileTaskLogSink is created and used (and LogSink is ignored).
	LogSink tempo.TaskLogSink
	// LogLevel is the minimum level sent to LogSink (e.g. slog.LevelInfo). Zero is Info. Use the system log level.
	LogLevel slog.Level
	// LogDir, when set, enables a file log sink: task logs are written to plain text files under this directory (one file per task).
	// RemoveTasks deletes the corresponding log files. LogLevel is used as the minimum level for the file sink.
	LogDir string
}

// NewRunner creates a new task runner with the given configuration.
// When Cfg.DB is set, task executions are stored in the database and recovered on startup.
func NewRunner(cfg Cfg) (*Runner, error) {
	if cfg.Parallelism <= 0 {
		cfg.Parallelism = 1
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 20
	}

	if cfg.HistorySize <= 0 {
		cfg.HistorySize = 20
	}

	var logSink tempo.TaskLogSink = cfg.LogSink
	var logCleaner TaskLogCleaner
	if cfg.LogDir != "" {
		fileSink, err := NewFileTaskLogSink(cfg.LogDir)
		if err != nil {
			return nil, fmt.Errorf("task log sink: %w", err)
		}
		logSink = fileSink
		logCleaner = fileSink
	}

	var persistence tempo.TaskStatePersistence
	if cfg.DB != nil {
		l := cfg.Logger
		if l == nil {
			l = slog.New(slog.DiscardHandler)
		}
		store, err := NewTaskExecutionStore(cfg.DB, l, logCleaner)
		if err != nil {
			return nil, fmt.Errorf("task execution store: %w", err)
		}
		persistence = store
	} else {
		persistence = tempo.NewMemPersistence()
	}

	qr, err := tempo.NewQueueRunner(tempo.RunnerCfg{
		Parallelism: cfg.Parallelism,
		QueueSize:   cfg.QueueSize,
		HistorySize: cfg.HistorySize,
		Persistence: persistence,
		LogSink:     logSink,
		LogLevel:    cfg.LogLevel,
	})
	if err != nil {
		return nil, fmt.Errorf("queue runner: %w", err)
	}

	l := cfg.Logger
	if l == nil {
		l = slog.New(slog.DiscardHandler)
	}

	return &Runner{
		queue:  qr,
		logger: l,
	}, nil
}

// Start begins processing queued tasks in the background.
func (r *Runner) Start() {
	r.queue.StartBg()
	r.logger.Info("task runner started", slog.String("component", "taskrunner"))
}

// Shutdown gracefully stops the runner, waiting for active tasks to complete.
func (r *Runner) Shutdown(ctx context.Context) error {
	r.logger.Info("task runner shutting down", slog.String("component", "taskrunner"))
	return r.queue.ShutDown(ctx)
}

// RegisterTask registers the task function for the name (if not already or overwrites).
// maxParallelism limits concurrent runs of this task (0 = no limit). Use AddRun(name) to enqueue a run.
// The runner logs when the task starts, finishes, or fails.
func (r *Runner) RegisterTask(fn func(ctx context.Context) error, name string, maxParallelism int) {
	wrapped := r.wrapTaskRun(name, fn)
	r.queue.RegisterTask(tempo.TaskDef{Name: name, Run: wrapped, MaxParallelism: maxParallelism})
	r.logger.Info("task registered", slog.String("component", "taskrunner"), slog.String("task", name))
}

// wrapTaskRun returns a function that logs start/finish/error then runs fn.
func (r *Runner) wrapTaskRun(name string, fn func(ctx context.Context) error) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		r.logger.Info("task started", slog.String("component", "taskrunner"), slog.String("task", name))
		err := fn(ctx)
		if err != nil {
			r.logger.Error("task failed", slog.String("component", "taskrunner"), slog.String("task", name), slog.String("error", err.Error()))
			return err
		}
		r.logger.Info("task finished", slog.String("component", "taskrunner"), slog.String("task", name))
		return nil
	}
}

// AddRun enqueues one run of the task by name. Returns the execution ID or an error (e.g. ErrQueueFull).
// The task must have been registered previously with RegisterTask.
func (r *Runner) AddRun(name string) (uuid.UUID, error) {
	id, err := r.queue.Add(name)
	if err != nil {
		return uuid.Nil, fmt.Errorf("enqueue task %q: %w", name, err)
	}
	return id, nil
}

// List returns raw task info from the queue (pending, running, completed, failed).
func (r *Runner) List() []tempo.TaskInfo {
	return r.queue.List()
}

// ExecutionInfo is a DTO for a single task execution, suitable for API responses.
// StartedAt is omitted when the task never ran (e.g. canceled while waiting), so duration is empty for those.
type ExecutionInfo struct {
	ID        uuid.UUID  `json:"id"`
	TaskName  string     `json:"task_name"`
	Status    string     `json:"status"`
	QueuedAt  time.Time  `json:"queued_at"`
	StartedAt *time.Time `json:"started_at,omitempty"`
	EndedAt   time.Time  `json:"ended_at"`
}

// Executions returns execution history as ExecutionInfo slice, ordered by queue time only:
// newest QueuedAt first, no grouping by status.
func (r *Runner) Executions() []ExecutionInfo {
	raw := r.queue.List()
	out := make([]ExecutionInfo, len(raw))
	for i, t := range raw {
		var startedAt *time.Time
		if !t.StartedAt.IsZero() {
			startedAt = &t.StartedAt
		}
		out[i] = ExecutionInfo{
			ID:        t.ID,
			TaskName:  t.Name,
			Status:    t.Status.Str(),
			QueuedAt:  t.QueuedAt,
			StartedAt: startedAt,
			EndedAt:   t.EndedAt,
		}
	}
	slices.SortFunc(out, func(a, b ExecutionInfo) int {
		if c := b.QueuedAt.Compare(a.QueuedAt); c != 0 {
			return c
		}
		return strings.Compare(a.ID.String(), b.ID.String())
	})
	return out
}

// Cancel requests cancellation of the task execution with the given ID.
// It works for waiting and running tasks. For a running task, the runner waits for the task
// to observe context cancellation (or a short timeout). Returns an error if the ID is not
// found or the task is not in a cancelable state.
func (r *Runner) Cancel(ctx context.Context, id uuid.UUID) error {
	err := r.queue.Cancel(ctx, id)
	if err != nil {
		return fmt.Errorf("cancel task %s: %w", id, err)
	}
	r.logger.Info("task canceled", slog.String("component", "taskrunner"), slog.String("id", id.String()))
	return nil
}
