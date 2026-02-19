package jobs

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-bumbu/tempo"
)

// Runner manages background jobs using a tempo QueueRunner.
type Runner struct {
	queue  *tempo.QueueRunner
	logger *slog.Logger
}

// Cfg holds the configuration for the job runner.
type Cfg struct {
	// Parallelism is the number of concurrent workers. Defaults to 1.
	Parallelism int
	// QueueSize is the maximum number of pending jobs. Defaults to 20.
	QueueSize int
	// Logger is used for job lifecycle logging. If nil, logging is disabled.
	Logger *slog.Logger
}

// NewRunner creates a new job runner with the given configuration.
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

// Start begins processing queued jobs in the background.
func (r *Runner) Start() {
	r.queue.StartBg()
	r.logger.Info("job runner started", slog.String("component", "jobs"))
}

// Shutdown gracefully stops the runner, waiting for active jobs to complete.
func (r *Runner) Shutdown(ctx context.Context) error {
	r.logger.Info("job runner shutting down", slog.String("component", "jobs"))
	return r.queue.ShutDown(ctx)
}

// Enqueue adds a named job to the queue for execution.
func (r *Runner) Enqueue(fn func(ctx context.Context) error, name string) error {
	_, err := r.queue.Add(fn, name)
	if err != nil {
		return fmt.Errorf("failed to enqueue job %q: %w", name, err)
	}
	r.logger.Info("job enqueued", slog.String("component", "jobs"), slog.String("job", name))
	return nil
}

// List returns info about all jobs (pending, running, completed, failed).
func (r *Runner) List() []tempo.TaskInfo {
	return r.queue.List()
}
