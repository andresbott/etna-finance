package marketdata

import (
	"errors"
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

			// EPS ingest no longer auto-registers; the series is defined at instrument creation for
			// stocks. Define it here for the symbols these sub-cases ingest into (NOPE stays absent).
			for _, sym := range []string{"AAA", "BBB", "CCC", "DDD", "EEE"} {
				if err := store.RegisterEPSSeries(ctx, sym); err != nil {
					t.Fatalf("RegisterEPSSeries %s: %v", sym, err)
				}
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

			t.Run("non-UTC times are normalized to UTC", func(t *testing.T) {
				cet := time.FixedZone("CET", 60*60)
				local := time.Date(2024, 5, 1, 1, 0, 0, 0, cet) // == 00:00 UTC
				if err := store.IngestEPSBulk(ctx, "EEE", []EPSPoint{{Time: local, Basic: 1.0, Diluted: 0.95}}); err != nil {
					t.Fatalf("unexpected error ingesting non-UTC time: %v", err)
				}
				recs, err := store.EPSHistory(ctx, "EEE", time.Time{}, time.Time{})
				if err != nil {
					t.Fatalf("EPSHistory: %v", err)
				}
				if len(recs) != 1 {
					t.Fatalf("expected 1 record, got %d", len(recs))
				}
				if !recs[0].Time.Equal(local) || recs[0].Time.Location() != time.UTC {
					t.Errorf("expected UTC instant %v, got %v (%q)", local, recs[0].Time, recs[0].Time.Location())
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

			t.Run("edit overwrites in place, delete removes", func(t *testing.T) {
				at := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
				if err := store.IngestEPS(ctx, "CCC", EPSPoint{Time: at, Basic: 3.0, Diluted: 2.9}); err != nil {
					t.Fatal(err)
				}
				// Editing at the same timestamp overwrites the values in place.
				if err := store.EditEPS(ctx, "CCC", at, EPSPoint{Time: at, Basic: 3.1, Diluted: 3.0}); err != nil {
					t.Fatalf("EditEPS: %v", err)
				}
				recs, err := store.EPSHistory(ctx, "CCC", time.Time{}, time.Time{})
				if err != nil {
					t.Fatal(err)
				}
				if len(recs) != 1 || !recs[0].Time.Equal(at) || recs[0].Basic != 3.1 {
					t.Errorf("expected single overwritten record at %v Basic=3.1, got %+v", at, recs)
				}
				if err := store.DeleteEPSAt(ctx, "CCC", at); err != nil {
					t.Fatalf("DeleteEPSAt: %v", err)
				}
				recs, err = store.EPSHistory(ctx, "CCC", time.Time{}, time.Time{})
				if err != nil {
					t.Fatal(err)
				}
				if len(recs) != 0 {
					t.Errorf("expected 0 records after delete, got %d", len(recs))
				}
				// Deleting the now-absent record reports not-found rather than success.
				if err := store.DeleteEPSAt(ctx, "CCC", at); !errors.Is(err, ErrRecordNotFound) {
					t.Fatalf("DeleteEPSAt on missing record err = %v, want ErrRecordNotFound", err)
				}
			})

			t.Run("editing a record's date is rejected", func(t *testing.T) {
				seeded := time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)
				moved := time.Date(2025, 5, 2, 0, 0, 0, 0, time.UTC)
				if err := store.IngestEPS(ctx, "DDD", EPSPoint{Time: seeded, Basic: 4.0, Diluted: 3.9}); err != nil {
					t.Fatal(err)
				}
				// The date is the record's identity: an edit cannot move it to a new timestamp.
				err := store.EditEPS(ctx, "DDD", seeded, EPSPoint{Time: moved, Basic: 4.1, Diluted: 4.0})
				if !errors.Is(err, ErrDateImmutable) {
					t.Fatalf("EditEPS with changed date err = %v, want ErrDateImmutable", err)
				}
				recs, err := store.EPSHistory(ctx, "DDD", time.Time{}, time.Time{})
				if err != nil {
					t.Fatal(err)
				}
				// Only the original record remains, unchanged; no phantom at `moved`.
				if len(recs) != 1 || !recs[0].Time.Equal(seeded) || recs[0].Basic != 4.0 {
					t.Fatalf("expected only the original record (Basic=4.0) to remain, got %+v", recs)
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
