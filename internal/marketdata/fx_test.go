package marketdata

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-bumbu/testdbs"
)

// mustRegisterPair registers an FX pair or fails the test. FX series must exist before ingest
// (ingest no longer auto-registers).
func mustRegisterPair(t *testing.T, ctx context.Context, store *Store, main, secondary string) {
	t.Helper()
	if err := store.RegisterPair(ctx, main, secondary); err != nil {
		t.Fatalf("register %s/%s: %v", main, secondary, err)
	}
}

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
				if err := store.RegisterPair(ctx, "EUR", "USD"); err != nil {
					t.Fatalf("register: %v", err)
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
				expectedRates := []float64{1.08, 1.09, 1.10}
				for i, r := range records {
					if r.Rate != expectedRates[i] {
						t.Errorf("records[%d]: expected rate %v, got %v", i, expectedRates[i], r.Rate)
					}
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
			if err := store.RegisterPair(ctx, "CHF", "USD"); err != nil {
				t.Fatalf("register: %v", err)
			}
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
				if err := store.RegisterPair(ctx, "GBP", "EUR"); err != nil {
					t.Fatalf("register: %v", err)
				}
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
			})
		})
	}
}

func TestEditRate(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("TestEditRate")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			t.Run("empty main returns error", func(t *testing.T) {
				err := store.EditRate(ctx, "", "USD", time.Time{}, RatePoint{Time: time.Now(), Rate: 1.5})
				if err == nil {
					t.Fatal("expected error for empty main")
				}
			})

			t.Run("update rate value in place", func(t *testing.T) {
				base := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
				mustRegisterPair(t, ctx, store, "JPY", "USD")
				err := store.IngestRate(ctx, "JPY", "USD", base, 0.0067)
				if err != nil {
					t.Fatal(err)
				}

				// Edit in place (same time, new rate)
				err = store.EditRate(ctx, "JPY", "USD", base, RatePoint{Time: base, Rate: 0.0070})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				updated, err := store.RateHistory(ctx, "JPY", "USD", time.Time{}, time.Time{})
				if err != nil {
					t.Fatal(err)
				}
				if len(updated) == 0 {
					t.Fatal("expected at least one record after update")
				}
				found := false
				for _, r := range updated {
					if r.Time.Equal(base) {
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

			t.Run("editing a record's date is rejected", func(t *testing.T) {
				base := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
				newTime := time.Date(2025, 7, 15, 0, 0, 0, 0, time.UTC)
				mustRegisterPair(t, ctx, store, "CAD", "USD")
				if err := store.IngestRate(ctx, "CAD", "USD", base, 0.74); err != nil {
					t.Fatal(err)
				}

				// The date is the record's identity: an edit cannot move it to a new timestamp.
				err := store.EditRate(ctx, "CAD", "USD", base, RatePoint{Time: newTime, Rate: 0.75})
				if !errors.Is(err, ErrDateImmutable) {
					t.Fatalf("EditRate with changed date err = %v, want ErrDateImmutable", err)
				}

				// Nothing changed: original rate intact at base, no record at newTime.
				atOld, err := store.RateAt(ctx, "CAD", "USD", base)
				if err != nil {
					t.Fatal(err)
				}
				if atOld == nil || atOld.Rate != 0.74 {
					t.Errorf("expected original record unchanged (rate 0.74), got %+v", atOld)
				}
				// As-of read at newTime still carries the base value forward (no new 0.75 record).
				atNew, err := store.RateAt(ctx, "CAD", "USD", newTime)
				if err != nil {
					t.Fatal(err)
				}
				if atNew == nil || atNew.Rate != 0.74 {
					t.Errorf("expected as-of rate at %v to remain 0.74 (no new record), got %+v", newTime, atNew)
				}
			})
		})
	}
}

func TestDeleteRateAt(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("TestDeleteRateAt")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			base := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
			if err := store.RegisterPair(ctx, "AUD", "USD"); err != nil {
				t.Fatalf("register: %v", err)
			}
			err = store.IngestRate(ctx, "AUD", "USD", base, 0.65)
			if err != nil {
				t.Fatal(err)
			}

			t.Run("delete existing record", func(t *testing.T) {
				err := store.DeleteRateAt(ctx, "AUD", "USD", base)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				after, err := store.RateHistory(ctx, "AUD", "USD", time.Time{}, time.Time{})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				for _, r := range after {
					if r.Time.Equal(base) {
						t.Error("record should have been deleted")
					}
				}
			})

			t.Run("delete missing record returns ErrRecordNotFound", func(t *testing.T) {
				err := store.DeleteRateAt(ctx, "AUD", "USD", base)
				if !errors.Is(err, ErrRecordNotFound) {
					t.Fatalf("err = %v, want ErrRecordNotFound", err)
				}
			})
		})
	}
}

// TestListFXPairsDetailed asserts pairs come back structured from labels.
func TestListFXPairsDetailed(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, err := NewStore(db.ConnDbName("TestListFXPairsDetailed"))
			if err != nil {
				t.Fatal(err)
			}
			if err := store.RegisterPair(ctx, "EUR", "USD"); err != nil {
				t.Fatalf("RegisterPair: %v", err)
			}
			pairs, err := store.ListFXPairsDetailed(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if len(pairs) != 1 || pairs[0] != (FXPair{Main: "EUR", Secondary: "USD"}) {
				t.Fatalf("got %+v, want [{EUR USD}]", pairs)
			}
		})
	}
}
