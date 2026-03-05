package taskrunner

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/reugn/go-quartz/quartz"
)

// TaskEnqueuer is called by the scheduler when a cron trigger fires.
// The implementation (in the app layer) should enqueue the task on the runner.
type TaskEnqueuer interface {
	EnqueueTask(ctx context.Context, taskName string) error
}

// FuncEnqueuer adapts a function to TaskEnqueuer.
type FuncEnqueuer func(ctx context.Context, taskName string) error

// EnqueueTask calls f(ctx, taskName).
func (f FuncEnqueuer) EnqueueTask(ctx context.Context, taskName string) error {
	return f(ctx, taskName)
}

// enqueueJob implements quartz.Job and calls the TaskEnqueuer when the trigger fires.
type enqueueJob struct {
	taskName string
	enqueuer TaskEnqueuer
}

func (j *enqueueJob) Execute(ctx context.Context) error {
	return j.enqueuer.EnqueueTask(ctx, j.taskName)
}

func (j *enqueueJob) Description() string {
	return fmt.Sprintf("enqueue task %q", j.taskName)
}

// Scheduler loads task schedules from the store and uses go-quartz to trigger enqueues at cron times.
type Scheduler struct {
	quartzSched quartz.Scheduler
	store       *ScheduleStore
	enqueuer    TaskEnqueuer
	logger      *slog.Logger
	mu          sync.Mutex
}

// SchedulerCfg configures the cron scheduler.
type SchedulerCfg struct {
	ScheduleStore *ScheduleStore
	Enqueuer      TaskEnqueuer
	Logger        *slog.Logger
}

// NormalizeCronExpression converts a 5-field Unix cron (minute hour dom month dow) to 6-field Quartz
// (second minute hour dom month dow) by prepending "0" for seconds. Expressions already with 6+ fields
// are returned unchanged. go-quartz requires the Quartz format.
func NormalizeCronExpression(cron string) string {
	cron = strings.TrimSpace(cron)
	parts := strings.Fields(cron)
	if len(parts) == 5 {
		return "0 " + cron
	}
	return cron
}

// ValidateCronExpression checks whether the cron expression is valid (same format used by the scheduler).
// Accepts both 5-field (Unix) and 6-field (Quartz) expressions; normalizes to Quartz before validating.
// Returns an error describing the problem if invalid. Use this before persisting a schedule (e.g. in API handlers).
func ValidateCronExpression(cron string) error {
	if cron == "" {
		return fmt.Errorf("cron expression is required")
	}
	cron = NormalizeCronExpression(cron)
	_, err := quartz.NewCronTrigger(cron)
	return err
}

// NewScheduler creates a new scheduler. Call Start to load schedules from the store and begin firing.
func NewScheduler(cfg SchedulerCfg) (*Scheduler, error) {
	if cfg.ScheduleStore == nil {
		return nil, fmt.Errorf("schedule store is required")
	}
	if cfg.Enqueuer == nil {
		return nil, fmt.Errorf("enqueuer is required")
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	qs, err := quartz.NewStdScheduler()
	if err != nil {
		return nil, fmt.Errorf("create quartz scheduler: %w", err)
	}
	return &Scheduler{
		quartzSched: qs,
		store:       cfg.ScheduleStore,
		enqueuer:    cfg.Enqueuer,
		logger:      logger,
	}, nil
}

// Start starts the quartz scheduler and loads enabled schedules from the store.
// It runs until ctx is cancelled. Call Refresh to reload schedules after DB changes.
func (s *Scheduler) Start(ctx context.Context) {
	s.quartzSched.Start(ctx)
	if err := s.loadSchedules(ctx); err != nil {
		s.logger.Error("failed to load task schedules", slog.String("component", "taskrunner"), slog.String("error", err.Error()))
	}
}

// loadSchedules clears existing scheduled jobs and reloads from the store (enabled only).
func (s *Scheduler) loadSchedules(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.quartzSched.Clear(); err != nil {
		return err
	}
	list, err := s.store.ListEnabled(ctx)
	if err != nil {
		return err
	}
	for _, sch := range list {
		expr := NormalizeCronExpression(sch.CronExpression)
		trigger, err := quartz.NewCronTrigger(expr)
		if err != nil {
			s.logger.Warn("invalid cron expression for task, skipping",
				slog.String("component", "taskrunner"),
				slog.String("task", sch.TaskName),
				slog.String("cron", sch.CronExpression),
				slog.String("error", err.Error()))
			continue
		}
		job := &enqueueJob{taskName: sch.TaskName, enqueuer: s.enqueuer}
		jobKey := quartz.NewJobKey("task:" + sch.TaskName)
		detail := quartz.NewJobDetail(job, jobKey)
		if err := s.quartzSched.ScheduleJob(detail, trigger); err != nil {
			s.logger.Warn("failed to schedule task",
				slog.String("component", "taskrunner"),
				slog.String("task", sch.TaskName),
				slog.String("error", err.Error()))
			continue
		}
		s.logger.Info("scheduled task",
			slog.String("component", "taskrunner"),
			slog.String("task", sch.TaskName),
			slog.String("cron", sch.CronExpression))
	}
	return nil
}

// Refresh reloads schedules from the store and reschedules all jobs (call after updating schedules in DB).
func (s *Scheduler) Refresh(ctx context.Context) error {
	return s.loadSchedules(ctx)
}

// Stop stops the quartz scheduler.
func (s *Scheduler) Stop() {
	s.quartzSched.Stop()
}

// Wait blocks until the scheduler stops and all jobs have finished.
func (s *Scheduler) Wait(ctx context.Context) {
	s.quartzSched.Wait(ctx)
}
