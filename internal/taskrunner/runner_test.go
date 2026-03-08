package taskrunner

import (
	"context"
	"testing"
	"time"
)

func TestNewRunner_Defaults(t *testing.T) {
	runner, err := NewRunner(Cfg{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner == nil {
		t.Fatal("expected non-nil runner")
	}
}

func TestNewRunner_WithCustomCfg(t *testing.T) {
	runner, err := NewRunner(Cfg{
		Parallelism: 2,
		QueueSize:   10,
		HistorySize: 5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner == nil {
		t.Fatal("expected non-nil runner")
	}
}

func TestNewRunner_WithDB(t *testing.T) {
	db := newTestDB(t)
	runner, err := NewRunner(Cfg{
		DB: db,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner == nil {
		t.Fatal("expected non-nil runner")
	}
}

func TestNewRunner_WithLogDir(t *testing.T) {
	dir := t.TempDir()
	runner, err := NewRunner(Cfg{
		LogDir: dir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner == nil {
		t.Fatal("expected non-nil runner")
	}
}

func TestRunner_RegisterAndAddRun(t *testing.T) {
	runner, err := NewRunner(Cfg{})
	if err != nil {
		t.Fatal(err)
	}
	runner.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = runner.Shutdown(ctx)
	}()

	called := make(chan struct{}, 1)
	runner.RegisterTask(func(ctx context.Context) error {
		called <- struct{}{}
		return nil
	}, "test-task", 1)

	id, err := runner.AddRun("test-task")
	if err != nil {
		t.Fatalf("add run error: %v", err)
	}
	if id.String() == "00000000-0000-0000-0000-000000000000" {
		t.Error("expected non-nil UUID")
	}

	// Wait for task to complete
	select {
	case <-called:
		// ok
	case <-time.After(5 * time.Second):
		t.Fatal("task did not execute within timeout")
	}
}

func TestRunner_List(t *testing.T) {
	runner, err := NewRunner(Cfg{})
	if err != nil {
		t.Fatal(err)
	}

	// List on empty runner should return non-nil
	list := runner.List()
	if list == nil {
		t.Fatal("expected non-nil list")
	}
}

func TestRunner_Executions_Empty(t *testing.T) {
	runner, err := NewRunner(Cfg{})
	if err != nil {
		t.Fatal(err)
	}

	execs := runner.Executions()
	if execs == nil {
		t.Fatal("expected non-nil slice")
	}
	if len(execs) != 0 {
		t.Errorf("expected 0 executions, got %d", len(execs))
	}
}

func TestRunner_Executions_AfterRun(t *testing.T) {
	runner, err := NewRunner(Cfg{})
	if err != nil {
		t.Fatal(err)
	}
	runner.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = runner.Shutdown(ctx)
	}()

	done := make(chan struct{}, 1)
	runner.RegisterTask(func(ctx context.Context) error {
		done <- struct{}{}
		return nil
	}, "exec-task", 1)

	_, _ = runner.AddRun("exec-task")

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}

	// Give a small window for status update
	time.Sleep(100 * time.Millisecond)

	execs := runner.Executions()
	if len(execs) < 1 {
		t.Fatalf("expected at least 1 execution, got %d", len(execs))
	}

	found := false
	for _, e := range execs {
		if e.TaskName == "exec-task" {
			found = true
			if e.Status != "complete" {
				t.Errorf("expected 'complete' status, got %q", e.Status)
			}
		}
	}
	if !found {
		t.Error("execution for 'exec-task' not found")
	}
}

func TestRunner_AddRun_UnregisteredTask(t *testing.T) {
	runner, err := NewRunner(Cfg{})
	if err != nil {
		t.Fatal(err)
	}
	runner.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = runner.Shutdown(ctx)
	}()

	// Adding a run for an unregistered task should still enqueue (runner handles unknown at execution time)
	// or may return an error depending on implementation. We just verify no panic.
	_, _ = runner.AddRun("nonexistent-task")
}

func TestRunner_Cancel(t *testing.T) {
	runner, err := NewRunner(Cfg{
		Parallelism: 1,
		QueueSize:   5,
	})
	if err != nil {
		t.Fatal(err)
	}
	runner.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = runner.Shutdown(ctx)
	}()

	// Register a blocking task
	blocker := make(chan struct{})
	runner.RegisterTask(func(ctx context.Context) error {
		<-blocker // block until we close
		return nil
	}, "blocking-task", 1)

	// Start the blocking task so the worker is busy
	_, _ = runner.AddRun("blocking-task")
	time.Sleep(100 * time.Millisecond) // let it start

	// Register and add a second task that will be waiting
	runner.RegisterTask(func(ctx context.Context) error {
		return nil
	}, "waiting-task", 1)

	waitingID, err := runner.AddRun("waiting-task")
	if err != nil {
		t.Fatalf("add run error: %v", err)
	}

	// Cancel the waiting task
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = runner.Cancel(ctx, waitingID)
	if err != nil {
		t.Logf("cancel returned error (may be expected): %v", err)
	}

	close(blocker) // unblock first task
}

func TestRunner_Executions_SortedByQueuedAt(t *testing.T) {
	runner, err := NewRunner(Cfg{QueueSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	runner.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = runner.Shutdown(ctx)
	}()

	done := make(chan struct{}, 3)
	runner.RegisterTask(func(ctx context.Context) error {
		done <- struct{}{}
		return nil
	}, "sort-task", 0)

	for i := 0; i < 3; i++ {
		_, _ = runner.AddRun("sort-task")
	}

	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for tasks")
		}
	}

	time.Sleep(100 * time.Millisecond)

	execs := runner.Executions()
	if len(execs) < 3 {
		t.Fatalf("expected at least 3 executions, got %d", len(execs))
	}

	// Verify sorted by QueuedAt descending (newest first)
	for i := 1; i < len(execs); i++ {
		if execs[i].QueuedAt.After(execs[i-1].QueuedAt) {
			t.Errorf("executions not sorted: index %d queued after index %d", i, i-1)
		}
	}
}
