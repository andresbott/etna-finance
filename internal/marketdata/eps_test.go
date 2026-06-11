package marketdata

import (
	"testing"
	"time"

	"github.com/andresbott/etna/internal/marketdata/importer"
	"github.com/go-bumbu/testdbs"
)

//nolint:gocyclo // table-driven test with many independent sub-cases; complexity is inherent and readable
func TestEPSStore(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, err := NewStore(db.ConnDbName("TestEPSStore"))
			if err != nil {
				t.Fatal(err)
			}

			t.Run("empty symbol returns error", func(t *testing.T) {
				if err := store.IngestEPS(ctx, "", EPSPoint{Time: time.Now(), Basic: 1}); err == nil {
					t.Fatal("expected error for empty symbol")
				}
			})

			t.Run("bulk ingest, history and latest round-trip", func(t *testing.T) {
				pts := []EPSPoint{
					{Time: time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC), Basic: 1.0, Diluted: 0.95},
					{Time: time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC), Basic: 1.1, Diluted: 1.05},
					{Time: time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC), Basic: 1.2, Diluted: 1.15},
				}
				if err := store.IngestEPSBulk(ctx, "AAA", pts); err != nil {
					t.Fatalf("IngestEPSBulk: %v", err)
				}

				recs, err := store.EPSHistory(ctx, "AAA", time.Time{}, time.Time{})
				if err != nil {
					t.Fatalf("EPSHistory: %v", err)
				}
				if len(recs) != 3 {
					t.Fatalf("expected 3 records, got %d", len(recs))
				}
				if recs[0].Basic != 1.0 || recs[0].Diluted != 0.95 {
					t.Errorf("record 0 mismatch: %+v", recs[0])
				}
				if recs[1].Basic != 1.1 || recs[1].Diluted != 1.05 {
					t.Errorf("record 1 mismatch: %+v", recs[1])
				}
				if recs[2].Basic != 1.2 || recs[2].Diluted != 1.15 {
					t.Errorf("record 2 mismatch: %+v", recs[2])
				}

				latest, err := store.LatestEPS(ctx, "AAA")
				if err != nil {
					t.Fatalf("LatestEPS: %v", err)
				}
				if latest == nil || latest.Basic != 1.2 {
					t.Errorf("expected latest Basic=1.2, got %+v", latest)
				}
			})

			t.Run("history of unknown symbol is nil, latest is nil", func(t *testing.T) {
				recs, err := store.EPSHistory(ctx, "NOPE", time.Time{}, time.Time{})
				if err != nil {
					t.Fatalf("EPSHistory: %v", err)
				}
				if recs != nil {
					t.Errorf("expected nil for unknown symbol, got %v", recs)
				}
				latest, err := store.LatestEPS(ctx, "NOPE")
				if err != nil {
					t.Fatalf("LatestEPS: %v", err)
				}
				if latest != nil {
					t.Errorf("expected nil latest, got %+v", latest)
				}
			})

			t.Run("restatement at same date overwrites", func(t *testing.T) {
				d := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
				if err := store.IngestEPS(ctx, "BBB", EPSPoint{Time: d, Basic: 2.0, Diluted: 1.9}); err != nil {
					t.Fatal(err)
				}
				if err := store.IngestEPS(ctx, "BBB", EPSPoint{Time: d, Basic: 2.5, Diluted: 2.4}); err != nil {
					t.Fatal(err)
				}
				recs, err := store.EPSHistory(ctx, "BBB", time.Time{}, time.Time{})
				if err != nil {
					t.Fatal(err)
				}
				if len(recs) != 1 || recs[0].Basic != 2.5 {
					t.Errorf("expected single overwritten record Basic=2.5, got %+v", recs)
				}
			})

			t.Run("edit moves the timestamp, delete removes", func(t *testing.T) {
				old := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
				moved := time.Date(2025, 3, 2, 0, 0, 0, 0, time.UTC)
				if err := store.IngestEPS(ctx, "CCC", EPSPoint{Time: old, Basic: 3.0, Diluted: 2.9}); err != nil {
					t.Fatal(err)
				}
				if err := store.EditEPS(ctx, "CCC", old, EPSPoint{Time: moved, Basic: 3.1, Diluted: 3.0}); err != nil {
					t.Fatalf("EditEPS: %v", err)
				}
				recs, err := store.EPSHistory(ctx, "CCC", time.Time{}, time.Time{})
				if err != nil {
					t.Fatal(err)
				}
				if len(recs) != 1 || !recs[0].Time.Equal(moved) || recs[0].Basic != 3.1 {
					t.Errorf("expected single moved record at %v Basic=3.1, got %+v", moved, recs)
				}
				if err := store.DeleteEPSAt(ctx, "CCC", moved); err != nil {
					t.Fatalf("DeleteEPSAt: %v", err)
				}
				recs, err = store.EPSHistory(ctx, "CCC", time.Time{}, time.Time{})
				if err != nil {
					t.Fatal(err)
				}
				if len(recs) != 0 {
					t.Errorf("expected 0 records after delete, got %d", len(recs))
				}
			})
		})
	}
}

func TestEPSPointsFromImporter(t *testing.T) {
	if got := EPSPointsFromImporter(nil); got != nil {
		t.Errorf("nil input should yield nil, got %v", got)
	}
	if got := EPSPointsFromImporter([]importer.EPSPoint{}); got != nil {
		t.Errorf("empty input should yield nil, got %v", got)
	}
	in := []importer.EPSPoint{
		{Symbol: "AAA", FiscalPeriod: "Q1", FiscalYear: "2025", Time: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), Basic: 1.5, Diluted: 1.4},
	}
	got := EPSPointsFromImporter(in)
	if len(got) != 1 {
		t.Fatalf("expected 1 point, got %d", len(got))
	}
	// Store EPSPoint carries only Time/Basic/Diluted (Symbol/FiscalPeriod/FiscalYear are intentionally dropped).
	want := EPSPoint{Time: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), Basic: 1.5, Diluted: 1.4}
	if got[0] != want {
		t.Errorf("got %+v, want %+v", got[0], want)
	}
}
