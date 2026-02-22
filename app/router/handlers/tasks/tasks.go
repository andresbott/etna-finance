package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/andresbott/etna/app/tasks"
	"github.com/andresbott/etna/internal/taskrunner"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Handler serves task list, trigger, executions, and schedules API using the task runner and app task definitions.
type Handler struct {
	Runner         *taskrunner.Runner
	Enqueuers      map[string]func() (uuid.UUID, error)
	ScheduleStore  *taskrunner.ScheduleStore
	Scheduler      *taskrunner.Scheduler
	ProductionMode bool // when true, dev-only tasks (e.g. log-only) are hidden from list and not runnable
}

// TaskWithSchedule is a task definition with its schedule (if any) for the combined API.
type TaskWithSchedule struct {
	tasks.TaskDef
	Schedule *taskrunner.Schedule `json:"schedule,omitempty"`
}

// ListTasks returns a handler that lists available tasks with their schedules combined.
func (h *Handler) ListTasks() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		avail := tasks.AvailableTaskDefs(h.ProductionMode)
		out := make([]TaskWithSchedule, 0, len(avail))
		var scheduleMap map[string]taskrunner.Schedule
		if h.ScheduleStore != nil {
			list, err := h.ScheduleStore.List(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			scheduleMap = make(map[string]taskrunner.Schedule, len(list))
			for _, s := range list {
				scheduleMap[s.TaskName] = s
			}
		}
		for _, t := range avail {
			ent := TaskWithSchedule{TaskDef: t}
			if s, ok := scheduleMap[t.ID]; ok {
				ent.Schedule = &s
			}
			out = append(out, ent)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string][]TaskWithSchedule{"tasks": out}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

// TriggerTask returns a handler that triggers a task by name (from path var "name").
func (h *Handler) TriggerTask() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := mux.Vars(r)["name"]
		if name == "" {
			http.Error(w, "task name required", http.StatusBadRequest)
			return
		}
		enqueue, ok := h.Enqueuers[name]
		if !ok {
			http.Error(w, "unknown task: "+name, http.StatusNotFound)
			return
		}
		id, err := enqueue()
		if err != nil {
			if errors.Is(err, taskrunner.ErrQueueFull) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "Task queue is full. Try again later."})
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{"execution_id": id.String()})
	})
}

// ListExecutions returns a handler that lists task executions (history).
func (h *Handler) ListExecutions() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		executions := h.Runner.Executions()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string][]taskrunner.ExecutionInfo{"executions": executions}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

// CancelExecution returns a handler that cancels a task execution by ID (path var "id").
// The ID must be a valid UUID of a waiting or running execution.
func (h *Handler) CancelExecution() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idStr := mux.Vars(r)["id"]
		if idStr == "" {
			http.Error(w, "execution id required", http.StatusBadRequest)
			return
		}
		id, err := uuid.Parse(idStr)
		if err != nil {
			http.Error(w, "invalid execution id: "+err.Error(), http.StatusBadRequest)
			return
		}
		ctx := r.Context()
		if err := h.Runner.Cancel(ctx, id); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

// GetTask returns a handler that returns a single task by name with its schedule (if any).
func (h *Handler) GetTask() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := mux.Vars(r)["name"]
		if name == "" {
			http.Error(w, "task name required", http.StatusBadRequest)
			return
		}
		if !tasks.TaskNameExists(name, h.ProductionMode) {
			http.Error(w, "unknown task: "+name, http.StatusNotFound)
			return
		}
		var def tasks.TaskDef
		for _, t := range tasks.AvailableTaskDefs(h.ProductionMode) {
			if t.ID == name {
				def = t
				break
			}
		}
		out := TaskWithSchedule{TaskDef: def}
		if h.ScheduleStore != nil {
			sch, err := h.ScheduleStore.GetByTaskName(r.Context(), name)
			if err == nil {
				out.Schedule = &sch
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(out); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

// UpsertTaskRequest is the body for PUT /tasks/:name (create or update task schedule).
type UpsertTaskRequest struct {
	CronExpression string `json:"cron_expression"`
	Enabled        *bool  `json:"enabled"`
}

// UpsertTask creates or updates the schedule for a task by name; returns the task with schedule.
func (h *Handler) UpsertTask() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.ScheduleStore == nil {
			http.Error(w, "schedules not available", http.StatusServiceUnavailable)
			return
		}
		name := mux.Vars(r)["name"]
		if name == "" {
			http.Error(w, "task name required", http.StatusBadRequest)
			return
		}
		if !tasks.TaskNameExists(name, h.ProductionMode) {
			http.Error(w, "unknown task: "+name, http.StatusNotFound)
			return
		}
		var body UpsertTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if body.CronExpression == "" {
			http.Error(w, "cron_expression required", http.StatusBadRequest)
			return
		}
		cronExpr := taskrunner.NormalizeCronExpression(body.CronExpression)
		if err := taskrunner.ValidateCronExpression(cronExpr); err != nil {
			http.Error(w, "invalid cron_expression: "+err.Error(), http.StatusBadRequest)
			return
		}
		enabled := true
		if body.Enabled != nil {
			enabled = *body.Enabled
		}
		sch, err := h.ScheduleStore.UpsertByTaskName(r.Context(), name, cronExpr, enabled)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if h.Scheduler != nil {
			_ = h.Scheduler.Refresh(context.Background())
		}
		var def tasks.TaskDef
		for _, t := range tasks.AvailableTaskDefs(h.ProductionMode) {
			if t.ID == name {
				def = t
				break
			}
		}
		out := TaskWithSchedule{TaskDef: def, Schedule: &sch}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(out); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

// PatchTaskRequest is the body for PATCH /tasks/:name (all fields optional).
type PatchTaskRequest struct {
	CronExpression *string `json:"cron_expression"`
	Enabled        *bool   `json:"enabled"`
}

// PatchTask partially updates the schedule for a task (404 if task has no schedule); returns task with schedule.
func (h *Handler) PatchTask() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.ScheduleStore == nil {
			http.Error(w, "schedules not available", http.StatusServiceUnavailable)
			return
		}
		name := mux.Vars(r)["name"]
		if name == "" {
			http.Error(w, "task name required", http.StatusBadRequest)
			return
		}
		if !tasks.TaskNameExists(name, h.ProductionMode) {
			http.Error(w, "unknown task: "+name, http.StatusNotFound)
			return
		}
		sch, err := h.ScheduleStore.GetByTaskName(r.Context(), name)
		if err != nil {
			if errors.Is(err, taskrunner.ErrScheduleNotFound) {
				http.Error(w, "schedule not found for task: "+name, http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var body PatchTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if body.CronExpression != nil {
			cronExpr := taskrunner.NormalizeCronExpression(*body.CronExpression)
			if err := taskrunner.ValidateCronExpression(cronExpr); err != nil {
				http.Error(w, "invalid cron_expression: "+err.Error(), http.StatusBadRequest)
				return
			}
			sch.CronExpression = cronExpr
		}
		if body.Enabled != nil {
			sch.Enabled = *body.Enabled
		}
		if err := h.ScheduleStore.Update(r.Context(), sch); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		sch, _ = h.ScheduleStore.GetByTaskName(r.Context(), name)
		if h.Scheduler != nil {
			_ = h.Scheduler.Refresh(context.Background())
		}
		var def tasks.TaskDef
		for _, t := range tasks.AvailableTaskDefs(h.ProductionMode) {
			if t.ID == name {
				def = t
				break
			}
		}
		out := TaskWithSchedule{TaskDef: def, Schedule: &sch}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(out); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

// DeleteTaskSchedule removes the schedule for a task by name (task definition remains).
func (h *Handler) DeleteTaskSchedule() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.ScheduleStore == nil {
			http.Error(w, "schedules not available", http.StatusServiceUnavailable)
			return
		}
		name := mux.Vars(r)["name"]
		if name == "" {
			http.Error(w, "task name required", http.StatusBadRequest)
			return
		}
		if !tasks.TaskNameExists(name, h.ProductionMode) {
			http.Error(w, "unknown task: "+name, http.StatusNotFound)
			return
		}
		if err := h.ScheduleStore.DeleteByTaskName(r.Context(), name); err != nil {
			if errors.Is(err, taskrunner.ErrScheduleNotFound) {
				http.Error(w, "schedule not found for task: "+name, http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if h.Scheduler != nil {
			_ = h.Scheduler.Refresh(context.Background())
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
