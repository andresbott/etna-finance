package backup

import (
	"path/filepath"
	"testing"

	"github.com/andresbott/etna/internal/accounting"
	"github.com/andresbott/etna/internal/csvimport"
	"github.com/andresbott/etna/internal/filestore"
	"github.com/andresbott/etna/internal/marketdata"
	"github.com/andresbott/etna/internal/taskrunner"
	"github.com/andresbott/etna/internal/toolsdata"
	"github.com/glebarez/sqlite"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// scheduleTestStores bundles the stores needed for a backup round-trip including schedules.
type scheduleTestStores struct {
	accounting *accounting.Store
	marketdata *marketdata.Store
	csvimport  *csvimport.Store
	filestore  *filestore.Store
	toolsdata  *toolsdata.Store
	schedules  *taskrunner.ScheduleStore
}

func newScheduleTestStores(t *testing.T, dsn string) scheduleTestStores {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("unable to connect to sqlite: %v", err)
	}
	mdStore, err := marketdata.NewStore(db)
	if err != nil {
		t.Fatalf("unable to create marketdata store: %v", err)
	}
	store, err := accounting.NewStore(db, mdStore)
	if err != nil {
		t.Fatalf("unable to connect to finance: %v", err)
	}
	csvStore, err := csvimport.NewStore(db)
	if err != nil {
		t.Fatalf("unable to create csvimport store: %v", err)
	}
	tdStore, err := toolsdata.NewStore(db)
	if err != nil {
		t.Fatalf("unable to create toolsdata store: %v", err)
	}
	fileStore, err := filestore.New(db, filepath.Join(t.TempDir(), "attachments"), 10*1024*1024)
	if err != nil {
		t.Fatalf("unable to create filestore: %v", err)
	}
	schStore, err := taskrunner.NewScheduleStore(db)
	if err != nil {
		t.Fatalf("unable to create schedule store: %v", err)
	}
	return scheduleTestStores{
		accounting: store, marketdata: mdStore, csvimport: csvStore,
		filestore: fileStore, toolsdata: tdStore, schedules: schStore,
	}
}

// TestScheduleRoundTrip verifies that task schedules survive an export -> import cycle.
func TestScheduleRoundTrip(t *testing.T) {
	src := newScheduleTestStores(t, "file:schedSource?mode=memory&cache=shared")

	if _, err := src.schedules.Create(t.Context(), taskrunner.Schedule{
		TaskName: "backup", CronExpression: "0 3 * * *", Enabled: true,
	}); err != nil {
		t.Fatalf("create enabled schedule: %v", err)
	}
	if _, err := src.schedules.Create(t.Context(), taskrunner.Schedule{
		TaskName: "sync", CronExpression: "0 */6 * * *", Enabled: false,
	}); err != nil {
		t.Fatalf("create disabled schedule: %v", err)
	}

	target := filepath.Join(t.TempDir(), "sched.zip")
	if err := export(t.Context(), src.accounting, src.marketdata, src.csvimport, src.filestore, src.toolsdata, src.schedules, target); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	dst := newScheduleTestStores(t, "file:schedDest?mode=memory&cache=shared")
	// pre-existing schedule that must be wiped before restore
	if _, err := dst.schedules.Create(t.Context(), taskrunner.Schedule{
		TaskName: "stale", CronExpression: "0 0 * * *", Enabled: true,
	}); err != nil {
		t.Fatalf("create stale schedule: %v", err)
	}

	if err := Import(t.Context(), dst.accounting, dst.marketdata, dst.csvimport, dst.filestore, dst.toolsdata, dst.schedules, target); err != nil {
		t.Fatalf("import failed: %v", err)
	}

	got, err := dst.schedules.List(t.Context())
	if err != nil {
		t.Fatalf("list schedules: %v", err)
	}

	want := []taskrunner.Schedule{
		{TaskName: "backup", CronExpression: "0 3 * * *", Enabled: true},
		{TaskName: "sync", CronExpression: "0 */6 * * *", Enabled: false},
	}
	if diff := cmp.Diff(want, got,
		cmpopts.IgnoreFields(taskrunner.Schedule{}, "ID", "CreatedAt", "UpdatedAt"),
		cmpopts.SortSlices(func(a, b taskrunner.Schedule) bool { return a.TaskName < b.TaskName }),
	); diff != "" {
		t.Errorf("unexpected schedules after restore (-want +got):\n%s", diff)
	}
}
