package marketdata

import (
	"testing"
	"time"

	"github.com/andresbott/etna/internal/marketdata/importer"
	"github.com/go-bumbu/testdbs"
	"github.com/google/go-cmp/cmp"
)

func TestIngestPricesBulk(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("TestIngestPricesBulk")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			t.Run("empty symbol returns error", func(t *testing.T) {
				err := store.IngestPricesBulk(ctx, "", []PricePoint{{Time: time.Now(), Price: 1.0}})
				if err == nil {
					t.Fatal("expected error for empty symbol")
				}
			})

			t.Run("empty points is no-op", func(t *testing.T) {
				err := store.IngestPricesBulk(ctx, "NOOP", nil)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			})

			t.Run("bulk ingest multiple points", func(t *testing.T) {
				points := []PricePoint{
					{Time: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Price: 100.0},
					{Time: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), Price: 101.0},
					{Time: time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC), Price: 102.0},
				}
				err := store.IngestPricesBulk(ctx, "BULK", points)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				records, err := store.PriceHistory(ctx, "BULK", time.Time{}, time.Time{})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(records) != 3 {
					t.Fatalf("expected 3 records, got %d", len(records))
				}
			})
		})
	}
}

func TestPriceHistory(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("TestPriceHistory")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			// Seed data
			base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			for i := 0; i < 5; i++ {
				err := store.IngestPrice(ctx, "HIST", base.AddDate(0, 0, i), float64(100+i))
				if err != nil {
					t.Fatalf("seed ingest: %v", err)
				}
			}

			t.Run("empty symbol returns error", func(t *testing.T) {
				_, err := store.PriceHistory(ctx, "", time.Time{}, time.Time{})
				if err == nil {
					t.Fatal("expected error for empty symbol")
				}
			})

			t.Run("non-existent symbol returns nil", func(t *testing.T) {
				records, err := store.PriceHistory(ctx, "DOESNOTEXIST", time.Time{}, time.Time{})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if records != nil {
					t.Errorf("expected nil, got %v", records)
				}
			})

			t.Run("all records unbounded", func(t *testing.T) {
				records, err := store.PriceHistory(ctx, "HIST", time.Time{}, time.Time{})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(records) != 5 {
					t.Fatalf("expected 5 records, got %d", len(records))
				}
				for _, r := range records {
					if r.Symbol != "HIST" {
						t.Errorf("expected symbol HIST, got %q", r.Symbol)
					}
					if r.ID == 0 {
						t.Error("expected non-zero ID")
					}
				}
			})

			t.Run("bounded range", func(t *testing.T) {
				start := base.AddDate(0, 0, 1)
				end := base.AddDate(0, 0, 3)
				records, err := store.PriceHistory(ctx, "HIST", start, end)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(records) < 1 {
					t.Fatal("expected at least 1 record in range")
				}
				for _, r := range records {
					if r.Time.Before(start) || r.Time.After(end) {
						t.Errorf("record time %v outside range [%v, %v]", r.Time, start, end)
					}
				}
			})
		})
	}
}

func TestLatestPrice(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("TestLatestPrice")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			t.Run("empty symbol returns error", func(t *testing.T) {
				_, err := store.LatestPrice(ctx, "")
				if err == nil {
					t.Fatal("expected error for empty symbol")
				}
			})

			t.Run("non-existent symbol returns nil", func(t *testing.T) {
				rec, err := store.LatestPrice(ctx, "NOPE")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if rec != nil {
					t.Errorf("expected nil, got %+v", rec)
				}
			})

			t.Run("returns most recent price", func(t *testing.T) {
				base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
				_ = store.IngestPrice(ctx, "LATEST", base, 50.0)
				_ = store.IngestPrice(ctx, "LATEST", base.AddDate(0, 0, 1), 55.0)
				_ = store.IngestPrice(ctx, "LATEST", base.AddDate(0, 0, 2), 60.0)

				rec, err := store.LatestPrice(ctx, "LATEST")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if rec == nil {
					t.Fatal("expected a record, got nil")
				}
				if rec.Symbol != "LATEST" {
					t.Errorf("expected symbol LATEST, got %q", rec.Symbol)
				}
				if rec.Price != 60.0 {
					t.Errorf("expected price 60.0, got %f", rec.Price)
				}
				if rec.ID == 0 {
					t.Error("expected non-zero ID")
				}
			})
		})
	}
}

func TestUpdatePrice(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("TestUpdatePrice")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			t.Run("zero id returns error", func(t *testing.T) {
				err := store.UpdatePrice(ctx, 0, PriceUpdate{Price: ptr(99.0)})
				if err == nil {
					t.Fatal("expected error for zero id")
				}
			})

			t.Run("update price value", func(t *testing.T) {
				base := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
				err := store.IngestPrice(ctx, "UPD", base, 200.0)
				if err != nil {
					t.Fatal(err)
				}

				records, err := store.PriceHistory(ctx, "UPD", time.Time{}, time.Time{})
				if err != nil {
					t.Fatal(err)
				}
				if len(records) == 0 {
					t.Fatal("expected at least one record")
				}
				recID := records[0].ID

				newPrice := 250.0
				err = store.UpdatePrice(ctx, recID, PriceUpdate{Price: &newPrice})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				updated, err := store.PriceHistory(ctx, "UPD", time.Time{}, time.Time{})
				if err != nil {
					t.Fatal(err)
				}
				found := false
				for _, r := range updated {
					if r.ID == recID {
						found = true
						if r.Price != 250.0 {
							t.Errorf("expected price 250.0, got %f", r.Price)
						}
					}
				}
				if !found {
					t.Error("updated record not found")
				}
			})

			t.Run("update time", func(t *testing.T) {
				base := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
				err := store.IngestPrice(ctx, "UPDT", base, 300.0)
				if err != nil {
					t.Fatal(err)
				}

				records, err := store.PriceHistory(ctx, "UPDT", time.Time{}, time.Time{})
				if err != nil {
					t.Fatal(err)
				}
				if len(records) == 0 {
					t.Fatal("expected at least one record")
				}
				recID := records[0].ID

				newTime := time.Date(2025, 7, 15, 0, 0, 0, 0, time.UTC)
				err = store.UpdatePrice(ctx, recID, PriceUpdate{Time: &newTime})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			})
		})
	}
}

func TestDeletePrice(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("TestDeletePrice")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			base := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
			err = store.IngestPrice(ctx, "DEL", base, 400.0)
			if err != nil {
				t.Fatal(err)
			}

			records, err := store.PriceHistory(ctx, "DEL", time.Time{}, time.Time{})
			if err != nil {
				t.Fatal(err)
			}
			if len(records) == 0 {
				t.Fatal("expected at least one record")
			}
			recID := records[0].ID

			t.Run("delete existing record", func(t *testing.T) {
				err := store.DeletePrice(ctx, recID)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				after, err := store.PriceHistory(ctx, "DEL", time.Time{}, time.Time{})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				for _, r := range after {
					if r.ID == recID {
						t.Error("record should have been deleted")
					}
				}
			})
		})
	}
}

func TestPricePointsFromImporter(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result := PricePointsFromImporter(nil)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("empty slice returns nil", func(t *testing.T) {
		result := PricePointsFromImporter([]importer.PricePoint{})
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("converts points correctly", func(t *testing.T) {
		input := []importer.PricePoint{
			{Time: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Price: 10.0},
			{Time: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), Price: 20.0},
			{Time: time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC), Price: 30.0},
		}
		want := []PricePoint{
			{Time: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Price: 10.0},
			{Time: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), Price: 20.0},
			{Time: time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC), Price: 30.0},
		}
		got := PricePointsFromImporter(input)
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestMaintenance(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("TestMaintenance")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			// Seed some data so maintenance has something to work with
			base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			for i := 0; i < 3; i++ {
				_ = store.IngestPrice(ctx, "MAINT", base.AddDate(0, 0, i), float64(100+i))
			}

			t.Run("maintenance runs without error", func(t *testing.T) {
				err := store.Maintenance(ctx)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			})

			t.Run("maintenance on empty store", func(t *testing.T) {
				dbCon2 := db.ConnDbName("TestMaintenanceEmpty")
				store2, err := NewStore(dbCon2)
				if err != nil {
					t.Fatal(err)
				}
				err = store2.Maintenance(ctx)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			})
		})
	}
}
