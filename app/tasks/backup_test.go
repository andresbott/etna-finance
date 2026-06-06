package tasks

import (
	"testing"
	"time"
)

func TestBackupFileName(t *testing.T) {
	ts := time.Date(2026, 6, 6, 14, 23, 45, 0, time.UTC)
	got := backupFileName(ts)
	want := "etna-finance-backup-2026-06-06_14-23-45.zip"
	if got != want {
		t.Errorf("backupFileName() = %q, want %q", got, want)
	}
}
