package taskrunner

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-bumbu/tempo"
	"github.com/google/uuid"
)

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
	// Logger is used for task lifecycle logging. If nil, logging is disabled.
	Logger *slog.Logger
}

// NewRunner creates a new task runner with the given configuration.
func NewRunner(cfg Cfg) *Runner {
	if cfg.Parallelism <= 0 {
		cfg.Parallelism = 1
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 20
	}

	qr := tempo.NewQueueRunner(tempo.RunnerCfg{
		Parallelism: cfg.Parallelism,
		QueueSize:   cfg.QueueSize,
		HistorySize: 50,
	})

	l := cfg.Logger
	if l == nil {
		l = slog.New(slog.DiscardHandler)
	}

	return &Runner{
		queue:  qr,
		logger: l,
	}
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

// Enqueue adds a named task to the queue for execution.
func (r *Runner) Enqueue(fn func(ctx context.Context) error, name string) error {
	_, err := r.EnqueueWithID(fn, name)
	return err
}

// EnqueueWithID adds a named task to the queue and returns its execution ID.
func (r *Runner) EnqueueWithID(fn func(ctx context.Context) error, name string) (uuid.UUID, error) {
	id, err := r.queue.Add(fn, name)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to enqueue task %q: %w", name, err)
	}
	r.logger.Info("task enqueued", slog.String("component", "taskrunner"), slog.String("task", name), slog.String("id", id.String()))
	return id, nil
}

// List returns raw task info from the queue (pending, running, completed, failed).
func (r *Runner) List() []tempo.TaskInfo {
	return r.queue.List()
}

// ExecutionInfo is a DTO for a single task execution, suitable for API responses.
type ExecutionInfo struct {
	ID        uuid.UUID `json:"id"`
	TaskName  string    `json:"task_name"`
	Status    string    `json:"status"`
	QueuedAt  time.Time `json:"queued_at"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
}

// Executions returns execution history as ExecutionInfo slice.
func (r *Runner) Executions() []ExecutionInfo {
	raw := r.queue.List()
	out := make([]ExecutionInfo, len(raw))
	for i, t := range raw {
		out[i] = ExecutionInfo{
			ID:        t.ID,
			TaskName:  t.Name,
			Status:    t.Status.Str(),
			QueuedAt:  t.QueuedAt,
			StartedAt: t.StartedAt,
			EndedAt:   t.EndedAt,
		}
	}
	return out
}
