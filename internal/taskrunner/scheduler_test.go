package taskrunner

import (
	"context"
	"testing"
)

func TestNormalizeCronExpression(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{name: "5 fields gets 0 prepended", input: "*/5 * * * *", expect: "0 */5 * * * *"},
		{name: "6 fields unchanged", input: "0 */5 * * * *", expect: "0 */5 * * * *"},
		{name: "7 fields unchanged", input: "0 */5 * * * * 2024", expect: "0 */5 * * * * 2024"},
		{name: "leading/trailing spaces trimmed", input: "  */5 * * * *  ", expect: "0 */5 * * * *"},
		{name: "single field unchanged", input: "*", expect: "*"},
		{name: "empty string", input: "", expect: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeCronExpression(tt.input)
			if got != tt.expect {
				t.Errorf("NormalizeCronExpression(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}

func TestValidateCronExpression(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid 5-field", input: "*/5 * * * *", wantErr: false},
		{name: "valid 6-field", input: "0 */5 * * * *", wantErr: false},
		{name: "empty string", input: "", wantErr: true},
		{name: "invalid expression", input: "not a cron", wantErr: true},
		{name: "every minute", input: "* * * * *", wantErr: false},
		{name: "specific time", input: "30 14 * * 1-5", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCronExpression(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCronExpression(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestFuncEnqueuer(t *testing.T) {
	var calledTask string
	f := FuncEnqueuer(func(ctx context.Context, taskName string) error {
		calledTask = taskName
		return nil
	})

	err := f.EnqueueTask(context.Background(), "my-task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calledTask != "my-task" {
		t.Errorf("expected task name %q, got %q", "my-task", calledTask)
	}
}

func TestFuncEnqueuer_Error(t *testing.T) {
	f := FuncEnqueuer(func(ctx context.Context, taskName string) error {
		return context.Canceled
	})

	err := f.EnqueueTask(context.Background(), "failing-task")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEnqueueJob(t *testing.T) {
	var calledTask string
	enqueuer := FuncEnqueuer(func(ctx context.Context, taskName string) error {
		calledTask = taskName
		return nil
	})

	job := &enqueueJob{taskName: "test-task", enqueuer: enqueuer}

	if desc := job.Description(); desc == "" {
		t.Error("expected non-empty description")
	}

	err := job.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calledTask != "test-task" {
		t.Errorf("expected %q, got %q", "test-task", calledTask)
	}
}
