package marketdata

import (
	"testing"
	"time"

	"github.com/go-bumbu/testdbs"
)

func TestIngestRatesBulk(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("TestIngestRatesBulk")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			t.Run("empty main currency returns error", func(t *testing.T) {
				err := store.IngestRatesBulk(ctx, "", "USD", []RatePoint{{Time: time.Now(), Rate: 1.0}})
				if err == nil {
					t.Fatal("expected error for empty main currency")
				}
			})

			t.Run("empty secondary currency returns error", func(t *testing.T) {
				err := store.IngestRatesBulk(ctx, "EUR", "", []RatePoint{{Time: time.Now(), Rate: 1.0}})
				if err == nil {
					t.Fatal("expected error for empty secondary currency")
				}
			})

			t.Run("empty points is no-op", func(t *testing.T) {
				err := store.IngestRatesBulk(ctx, "EUR", "USD", nil)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			})

			t.Run("bulk ingest multiple points", func(t *testing.T) {
				points := []RatePoint{
					{Time: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Rate: 1.08},
					{Time: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), Rate: 1.09},
					{Time: time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC), Rate: 1.10},
				}
				err := store.IngestRatesBulk(ctx, "EUR", "USD", points)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				records, err := store.RateHistory(ctx, "EUR", "USD", time.Time{}, time.Time{})
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

func TestRateHistory(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("TestRateHistory")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			// Seed data
			base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			for i := 0; i < 5; i++ {
				err := store.IngestRate(ctx, "CHF", "USD", base.AddDate(0, 0, i), 1.0+float64(i)*0.01)
				if err != nil {
					t.Fatalf("seed ingest: %v", err)
				}
			}

			t.Run("empty main returns error", func(t *testing.T) {
				_, err := store.RateHistory(ctx, "", "USD", time.Time{}, time.Time{})
				if err == nil {
					t.Fatal("expected error for empty main")
				}
			})

			t.Run("empty secondary returns error", func(t *testing.T) {
				_, err := store.RateHistory(ctx, "CHF", "", time.Time{}, time.Time{})
				if err == nil {
					t.Fatal("expected error for empty secondary")
				}
			})

			t.Run("non-existent pair returns nil", func(t *testing.T) {
				records, err := store.RateHistory(ctx, "XXX", "YYY", time.Time{}, time.Time{})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if records != nil {
					t.Errorf("expected nil, got %v", records)
				}
			})

			t.Run("all records unbounded", func(t *testing.T) {
				records, err := store.RateHistory(ctx, "CHF", "USD", time.Time{}, time.Time{})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(records) != 5 {
					t.Fatalf("expected 5 records, got %d", len(records))
				}
				for _, r := range records {
					if r.Main != "CHF" || r.Secondary != "USD" {
						t.Errorf("expected CHF/USD, got %s/%s", r.Main, r.Secondary)
					}
					if r.ID == 0 {
						t.Error("expected non-zero ID")
					}
				}
			})

			t.Run("bounded range", func(t *testing.T) {
				start := base.AddDate(0, 0, 1)
				end := base.AddDate(0, 0, 3)
				records, err := store.RateHistory(ctx, "CHF", "USD", start, end)
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

func TestLatestRate(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("TestLatestRate")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			t.Run("empty main returns error", func(t *testing.T) {
				_, err := store.LatestRate(ctx, "", "USD")
				if err == nil {
					t.Fatal("expected error for empty main")
				}
			})

			t.Run("empty secondary returns error", func(t *testing.T) {
				_, err := store.LatestRate(ctx, "EUR", "")
				if err == nil {
					t.Fatal("expected error for empty secondary")
				}
			})

			t.Run("non-existent pair returns nil", func(t *testing.T) {
				rec, err := store.LatestRate(ctx, "ZZZ", "QQQ")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if rec != nil {
					t.Errorf("expected nil, got %+v", rec)
				}
			})

			t.Run("returns most recent rate", func(t *testing.T) {
				base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
				_ = store.IngestRate(ctx, "GBP", "EUR", base, 1.15)
				_ = store.IngestRate(ctx, "GBP", "EUR", base.AddDate(0, 0, 1), 1.16)
				_ = store.IngestRate(ctx, "GBP", "EUR", base.AddDate(0, 0, 2), 1.17)

				rec, err := store.LatestRate(ctx, "GBP", "EUR")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if rec == nil {
					t.Fatal("expected a record, got nil")
				}
				if rec.Main != "GBP" || rec.Secondary != "EUR" {
					t.Errorf("expected GBP/EUR, got %s/%s", rec.Main, rec.Secondary)
				}
				if rec.Rate != 1.17 {
					t.Errorf("expected rate 1.17, got %f", rec.Rate)
				}
				if rec.ID == 0 {
					t.Error("expected non-zero ID")
				}
			})
		})
	}
}

func TestUpdateRate(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("TestUpdateRate")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			t.Run("zero id returns error", func(t *testing.T) {
				err := store.UpdateRate(ctx, 0, RateUpdate{Rate: ptr(1.5)})
				if err == nil {
					t.Fatal("expected error for zero id")
				}
			})

			t.Run("update rate value", func(t *testing.T) {
				base := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
				err := store.IngestRate(ctx, "JPY", "USD", base, 0.0067)
				if err != nil {
					t.Fatal(err)
				}

				records, err := store.RateHistory(ctx, "JPY", "USD", time.Time{}, time.Time{})
				if err != nil {
					t.Fatal(err)
				}
				if len(records) == 0 {
					t.Fatal("expected at least one record")
				}
				recID := records[0].ID

				newRate := 0.0070
				err = store.UpdateRate(ctx, recID, RateUpdate{Rate: &newRate})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				updated, err := store.RateHistory(ctx, "JPY", "USD", time.Time{}, time.Time{})
				if err != nil {
					t.Fatal(err)
				}
				found := false
				for _, r := range updated {
					if r.ID == recID {
						found = true
						if r.Rate != 0.0070 {
							t.Errorf("expected rate 0.0070, got %f", r.Rate)
						}
					}
				}
				if !found {
					t.Error("updated record not found")
				}
			})

			t.Run("update time", func(t *testing.T) {
				base := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
				err := store.IngestRate(ctx, "CAD", "USD", base, 0.74)
				if err != nil {
					t.Fatal(err)
				}

				records, err := store.RateHistory(ctx, "CAD", "USD", time.Time{}, time.Time{})
				if err != nil {
					t.Fatal(err)
				}
				if len(records) == 0 {
					t.Fatal("expected at least one record")
				}
				recID := records[0].ID

				newTime := time.Date(2025, 7, 15, 0, 0, 0, 0, time.UTC)
				err = store.UpdateRate(ctx, recID, RateUpdate{Time: &newTime})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			})
		})
	}
}

func TestDeleteRate(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("TestDeleteRate")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			base := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
			err = store.IngestRate(ctx, "AUD", "USD", base, 0.65)
			if err != nil {
				t.Fatal(err)
			}

			records, err := store.RateHistory(ctx, "AUD", "USD", time.Time{}, time.Time{})
			if err != nil {
				t.Fatal(err)
			}
			if len(records) == 0 {
				t.Fatal("expected at least one record")
			}
			recID := records[0].ID

			t.Run("delete existing record", func(t *testing.T) {
				err := store.DeleteRate(ctx, recID)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				after, err := store.RateHistory(ctx, "AUD", "USD", time.Time{}, time.Time{})
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
