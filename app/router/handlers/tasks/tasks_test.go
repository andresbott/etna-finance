package tasks

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andresbott/etna/internal/taskrunner"
	"github.com/go-bumbu/http/middleware"
	"github.com/gorilla/mux"
)

// TestTriggerTask_ResponseNotWrappedByMiddleware verifies that the trigger endpoint's
// 202 Accepted success response survives the production middleware with its body intact.
//
// The app wires middleware.New(Cfg{JsonErrors: true}) (see app/router/main.go). Older
// versions of go-bumbu/http wrapped the body of ANY non-200 response into
// {"error": <body>, "code": <status>}, which buried execution_id for clients that read it
// (e.g. the backup page). go-bumbu/http v0.5.0+ only wraps genuine error responses
// (status < 200 || >= 400), so a 202 is forwarded unchanged. This test guards that.
func TestTriggerTask_ResponseNotWrappedByMiddleware(t *testing.T) {
	runner, err := taskrunner.NewRunner(taskrunner.Cfg{})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	// Register a no-op under the "backup" task id so AddRun succeeds. We do not start
	// the runner; we only need the trigger to enqueue and return an execution id.
	runner.RegisterTask(func(ctx context.Context) error { return nil }, "backup", 0)

	h := &Handler{Runner: runner}

	// Wrap with the production middleware (JsonErrors: true), as in app/router/main.go.
	mid := middleware.New(middleware.Cfg{JsonErrors: true, Logger: slog.New(slog.DiscardHandler)})
	handler := mid.Middleware(h.TriggerTask())

	req := httptest.NewRequest(http.MethodPost, "/tasks/backup/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "backup"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}

	var body struct {
		ExecutionID string `json:"execution_id"`
		Error       string `json:"error"`
		Code        int    `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	if body.Error != "" || body.Code != 0 {
		t.Fatalf("response was wrapped by the middleware into an error envelope: %q", rec.Body.String())
	}
	if body.ExecutionID == "" {
		t.Fatalf("execution_id missing/empty in response: %q", rec.Body.String())
	}
}
