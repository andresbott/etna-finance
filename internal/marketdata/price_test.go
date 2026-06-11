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
				err := store.IngestPricesBulk(ctx, "", []PricePoint{{Time: time.Now(), Close: 1.0}})
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
					{Time: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Open: 99.0, High: 105.0, Low: 98.0, Close: 100.0, Volume: 1000},
					{Time: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), Open: 100.5, High: 106.0, Low: 99.5, Close: 101.0, Volume: 1200},
					{Time: time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC), Open: 101.0, High: 107.0, Low: 100.0, Close: 102.0, Volume: 1100},
				}
				if err := store.RegisterInstrument(ctx, "BULK"); err != nil {
					t.Fatalf("register: %v", err)
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

				// Verify all five OHLCV fields round-trip correctly
				r := records[0]
				if r.Open != 99.0 || r.High != 105.0 || r.Low != 98.0 || r.Close != 100.0 || r.Volume != 1000 {
					t.Errorf("record 0 OHLCV mismatch: got Open=%f High=%f Low=%f Close=%f Volume=%f",
						r.Open, r.High, r.Low, r.Close, r.Volume)
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
			if err := store.RegisterInstrument(ctx, "HIST"); err != nil {
				t.Fatalf("register: %v", err)
			}
			for i := 0; i < 5; i++ {
				price := float64(100 + i)
				err := store.IngestPrice(ctx, "HIST", PricePoint{
					Time: base.AddDate(0, 0, i), Open: price, High: price + 1, Low: price - 1, Close: price,
				})
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
				if err := store.RegisterInstrument(ctx, "LATEST"); err != nil {
					t.Fatalf("register: %v", err)
				}
				if err := store.IngestPrice(ctx, "LATEST", PricePoint{Time: base, Close: 50.0}); err != nil {
					t.Fatalf("seed ingest: %v", err)
				}
				if err := store.IngestPrice(ctx, "LATEST", PricePoint{Time: base.AddDate(0, 0, 1), Close: 55.0}); err != nil {
					t.Fatalf("seed ingest: %v", err)
				}
				if err := store.IngestPrice(ctx, "LATEST", PricePoint{Time: base.AddDate(0, 0, 2), Close: 60.0}); err != nil {
					t.Fatalf("seed ingest: %v", err)
				}

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
				if rec.Close != 60.0 {
					t.Errorf("expected close 60.0, got %f", rec.Close)
				}
			})
		})
	}
}

func TestEditPrice(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("TestEditPrice")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			t.Run("update close value", func(t *testing.T) {
				base := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
				if err := store.RegisterInstrument(ctx, "UPD"); err != nil {
					t.Fatalf("register: %v", err)
				}
				err := store.IngestPrice(ctx, "UPD", PricePoint{Time: base, Close: 200.0})
				if err != nil {
					t.Fatal(err)
				}

				// Edit in-place (same time, new close)
				err = store.EditPrice(ctx, "UPD", base, PricePoint{Time: base, Close: 250.0})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				updated, err := store.PriceHistory(ctx, "UPD", time.Time{}, time.Time{})
				if err != nil {
					t.Fatal(err)
				}
				if len(updated) == 0 {
					t.Fatal("expected records after edit")
				}
				if updated[0].Close != 250.0 {
					t.Errorf("expected close 250.0, got %f", updated[0].Close)
				}
			})

			t.Run("move to new time", func(t *testing.T) {
				base := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
				newTime := time.Date(2025, 7, 15, 0, 0, 0, 0, time.UTC)
				if err := store.RegisterInstrument(ctx, "UPDT"); err != nil {
					t.Fatalf("register: %v", err)
				}
				err := store.IngestPrice(ctx, "UPDT", PricePoint{Time: base, Close: 300.0})
				if err != nil {
					t.Fatal(err)
				}

				err = store.EditPrice(ctx, "UPDT", base, PricePoint{Time: newTime, Close: 310.0})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// Original time should be gone
				orig, err := store.PriceHistory(ctx, "UPDT", base, base)
				if err != nil {
					t.Fatal(err)
				}
				if len(orig) != 0 {
					t.Errorf("expected old record to be removed, got %d records", len(orig))
				}
			})
		})
	}
}

func TestDeletePriceAt(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("TestDeletePriceAt")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			base := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
			if err := store.RegisterInstrument(ctx, "DEL"); err != nil {
				t.Fatalf("register: %v", err)
			}
			err = store.IngestPrice(ctx, "DEL", PricePoint{Time: base, Close: 400.0})
			if err != nil {
				t.Fatal(err)
			}

			t.Run("delete existing record", func(t *testing.T) {
				err := store.DeletePriceAt(ctx, "DEL", base)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				after, err := store.PriceHistory(ctx, "DEL", time.Time{}, time.Time{})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				for _, r := range after {
					if r.Time.Equal(base) {
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
			{Time: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Open: 9.0, High: 11.0, Low: 8.5, Close: 10.0, Volume: 500},
			{Time: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), Open: 19.0, High: 21.0, Low: 18.5, Close: 20.0, Volume: 600},
			{Time: time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC), Open: 29.0, High: 31.0, Low: 28.5, Close: 30.0, Volume: 700},
		}
		want := []PricePoint{
			{Time: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Open: 9.0, High: 11.0, Low: 8.5, Close: 10.0, Volume: 500},
			{Time: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), Open: 19.0, High: 21.0, Low: 18.5, Close: 20.0, Volume: 600},
			{Time: time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC), Open: 29.0, High: 31.0, Low: 28.5, Close: 30.0, Volume: 700},
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
			if err := store.RegisterInstrument(ctx, "MAINT"); err != nil {
				t.Fatalf("register: %v", err)
			}
			for i := 0; i < 3; i++ {
				_ = store.IngestPrice(ctx, "MAINT", PricePoint{Time: base.AddDate(0, 0, i), Close: float64(100 + i)})
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
