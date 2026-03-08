package taskrunner

import (
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	return db
}

func TestNewScheduleStore(t *testing.T) {
	db := newTestDB(t)
	store, err := NewScheduleStore(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestNewScheduleStore_NilDB(t *testing.T) {
	_, err := NewScheduleStore(nil)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestScheduleStore_CreateAndGetByID(t *testing.T) {
	db := newTestDB(t)
	store, err := NewScheduleStore(db)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	created, err := store.Create(ctx, Schedule{
		TaskName:       "backup",
		CronExpression: "0 * * * *",
		Enabled:        true,
	})
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if created.TaskName != "backup" {
		t.Errorf("expected task name 'backup', got %q", created.TaskName)
	}

	fetched, err := store.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get by id error: %v", err)
	}
	if fetched.TaskName != "backup" {
		t.Errorf("expected task name 'backup', got %q", fetched.TaskName)
	}
	if fetched.CronExpression != "0 * * * *" {
		t.Errorf("expected cron '0 * * * *', got %q", fetched.CronExpression)
	}
}

func TestScheduleStore_GetByID_NotFound(t *testing.T) {
	db := newTestDB(t)
	store, _ := NewScheduleStore(db)

	_, err := store.GetByID(context.Background(), 9999)
	if !errors.Is(err, ErrScheduleNotFound) {
		t.Errorf("expected ErrScheduleNotFound, got %v", err)
	}
}

func TestScheduleStore_GetByTaskName(t *testing.T) {
	db := newTestDB(t)
	store, _ := NewScheduleStore(db)
	ctx := context.Background()

	_, err := store.Create(ctx, Schedule{TaskName: "sync", CronExpression: "*/5 * * * *", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}

	sch, err := store.GetByTaskName(ctx, "sync")
	if err != nil {
		t.Fatalf("get by task name error: %v", err)
	}
	if sch.TaskName != "sync" {
		t.Errorf("expected 'sync', got %q", sch.TaskName)
	}
}

func TestScheduleStore_GetByTaskName_NotFound(t *testing.T) {
	db := newTestDB(t)
	store, _ := NewScheduleStore(db)

	_, err := store.GetByTaskName(context.Background(), "nonexistent")
	if !errors.Is(err, ErrScheduleNotFound) {
		t.Errorf("expected ErrScheduleNotFound, got %v", err)
	}
}

func TestScheduleStore_List(t *testing.T) {
	db := newTestDB(t)
	store, _ := NewScheduleStore(db)
	ctx := context.Background()

	// Empty list
	list, err := store.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d", len(list))
	}

	// Create two schedules
	_, _ = store.Create(ctx, Schedule{TaskName: "a", CronExpression: "* * * * *", Enabled: true})
	_, _ = store.Create(ctx, Schedule{TaskName: "b", CronExpression: "* * * * *", Enabled: false})

	list, err = store.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 schedules, got %d", len(list))
	}
}

func TestScheduleStore_ListEnabled(t *testing.T) {
	db := newTestDB(t)
	store, _ := NewScheduleStore(db)
	ctx := context.Background()

	_, _ = store.Create(ctx, Schedule{TaskName: "enabled-task", CronExpression: "* * * * *", Enabled: true})
	// Create as enabled, then update to disabled (GORM skips zero-value bool on create due to default:true)
	disabled, _ := store.Create(ctx, Schedule{TaskName: "disabled-task", CronExpression: "* * * * *", Enabled: true})
	_ = store.Update(ctx, Schedule{ID: disabled.ID, CronExpression: "* * * * *", Enabled: false})

	list, err := store.ListEnabled(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 enabled schedule, got %d", len(list))
	}
	if list[0].TaskName != "enabled-task" {
		t.Errorf("expected 'enabled-task', got %q", list[0].TaskName)
	}
}

func TestScheduleStore_Update(t *testing.T) {
	db := newTestDB(t)
	store, _ := NewScheduleStore(db)
	ctx := context.Background()

	created, _ := store.Create(ctx, Schedule{TaskName: "upd", CronExpression: "0 * * * *", Enabled: true})

	err := store.Update(ctx, Schedule{ID: created.ID, CronExpression: "*/10 * * * *", Enabled: false})
	if err != nil {
		t.Fatalf("update error: %v", err)
	}

	fetched, _ := store.GetByID(ctx, created.ID)
	if fetched.CronExpression != "*/10 * * * *" {
		t.Errorf("expected updated cron, got %q", fetched.CronExpression)
	}
	if fetched.Enabled != false {
		t.Error("expected enabled to be false")
	}
}

func TestScheduleStore_Update_NoID(t *testing.T) {
	db := newTestDB(t)
	store, _ := NewScheduleStore(db)

	err := store.Update(context.Background(), Schedule{CronExpression: "* * * * *"})
	if err == nil {
		t.Fatal("expected error for zero ID")
	}
}

func TestScheduleStore_Delete(t *testing.T) {
	db := newTestDB(t)
	store, _ := NewScheduleStore(db)
	ctx := context.Background()

	created, _ := store.Create(ctx, Schedule{TaskName: "del", CronExpression: "* * * * *", Enabled: true})

	err := store.Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("delete error: %v", err)
	}

	// Should be soft-deleted, so GetByID should fail
	_, err = store.GetByID(ctx, created.ID)
	if !errors.Is(err, ErrScheduleNotFound) {
		t.Errorf("expected ErrScheduleNotFound after delete, got %v", err)
	}
}

func TestScheduleStore_DeleteByTaskName(t *testing.T) {
	db := newTestDB(t)
	store, _ := NewScheduleStore(db)
	ctx := context.Background()

	_, _ = store.Create(ctx, Schedule{TaskName: "deltask", CronExpression: "* * * * *", Enabled: true})

	err := store.DeleteByTaskName(ctx, "deltask")
	if err != nil {
		t.Fatalf("delete by task name error: %v", err)
	}

	_, err = store.GetByTaskName(ctx, "deltask")
	if !errors.Is(err, ErrScheduleNotFound) {
		t.Errorf("expected ErrScheduleNotFound, got %v", err)
	}
}

func TestScheduleStore_DeleteByTaskName_NotFound(t *testing.T) {
	db := newTestDB(t)
	store, _ := NewScheduleStore(db)

	err := store.DeleteByTaskName(context.Background(), "no-such-task")
	if !errors.Is(err, ErrScheduleNotFound) {
		t.Errorf("expected ErrScheduleNotFound, got %v", err)
	}
}

func TestScheduleStore_UpsertByTaskName_Create(t *testing.T) {
	db := newTestDB(t)
	store, _ := NewScheduleStore(db)
	ctx := context.Background()

	sch, err := store.UpsertByTaskName(ctx, "new-task", "*/15 * * * *", true)
	if err != nil {
		t.Fatalf("upsert create error: %v", err)
	}
	if sch.TaskName != "new-task" {
		t.Errorf("expected 'new-task', got %q", sch.TaskName)
	}
	if sch.CronExpression != "*/15 * * * *" {
		t.Errorf("expected cron '*/15 * * * *', got %q", sch.CronExpression)
	}
}

func TestScheduleStore_UpsertByTaskName_Update(t *testing.T) {
	db := newTestDB(t)
	store, _ := NewScheduleStore(db)
	ctx := context.Background()

	// Create first
	_, _ = store.UpsertByTaskName(ctx, "upsert-task", "0 * * * *", true)

	// Upsert should update
	sch, err := store.UpsertByTaskName(ctx, "upsert-task", "*/30 * * * *", false)
	if err != nil {
		t.Fatalf("upsert update error: %v", err)
	}
	if sch.CronExpression != "*/30 * * * *" {
		t.Errorf("expected updated cron, got %q", sch.CronExpression)
	}
	if sch.Enabled != false {
		t.Error("expected enabled=false after upsert")
	}

	// Should still be only one schedule
	list, _ := store.List(ctx)
	if len(list) != 1 {
		t.Errorf("expected 1 schedule, got %d", len(list))
	}
}

func TestScheduleStore_UpsertByTaskName_RestoreSoftDeleted(t *testing.T) {
	db := newTestDB(t)
	store, _ := NewScheduleStore(db)
	ctx := context.Background()

	// Create and delete
	created, _ := store.UpsertByTaskName(ctx, "restore-task", "0 * * * *", true)
	_ = store.Delete(ctx, created.ID)

	// Upsert should restore the soft-deleted record
	sch, err := store.UpsertByTaskName(ctx, "restore-task", "*/5 * * * *", true)
	if err != nil {
		t.Fatalf("upsert restore error: %v", err)
	}
	if sch.CronExpression != "*/5 * * * *" {
		t.Errorf("expected restored cron, got %q", sch.CronExpression)
	}

	// Should be findable again
	fetched, err := store.GetByTaskName(ctx, "restore-task")
	if err != nil {
		t.Fatalf("expected to find restored schedule: %v", err)
	}
	if fetched.Enabled != true {
		t.Error("expected enabled=true after restore")
	}
}
