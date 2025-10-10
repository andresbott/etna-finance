package timeseries

import (
	"github.com/go-bumbu/testdbs"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"sort"
	"testing"
	"time"
)

func TestIngestSeries(t *testing.T) {
	tcs := []struct {
		name    string
		input   Record
		wantErr string
	}{
		{
			name: "create valid record",
			input: Record{
				Series: "btc_price",
				Time:   time.Now(),
				Value:  68000.5,
			},
		},
		{
			name: "want error on empty series name",
			input: Record{
				Time:  time.Now(),
				Value: 123.4,
			},
			wantErr: "timeseries name cannot be empty",
		},
		{
			name: "want error on zero time",
			input: Record{
				Series: "btc_price",
				Value:  100.0,
			},
			wantErr: "timeseries time value cannot be zero",
		},
		{
			name: "want error on missing series in db",
			input: Record{
				Series: "unknown_series",
				Time:   time.Now(),
				Value:  99.9,
			},
			wantErr: "failed to lookup series: record not found",
		},
	}

	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {

			dbCon := db.ConnDbName("TestSeriesIngest")
			store, err := NewRegistry(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			// Arrange: create a known series so that valid tests can use it
			existingSeries := dbTimeSeries{Name: "btc_price"}
			if err := store.db.Create(&existingSeries).Error; err != nil {
				t.Fatalf("failed to create test series: %v", err)
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					err := store.Ingest(tc.input)

					if tc.wantErr != "" {
						// Expecting an error
						if err == nil {
							t.Fatalf("expected error: %s, but got none", tc.wantErr)
						}

						if err.Error() != tc.wantErr {
							t.Errorf("expected error: %s, but got %v", tc.wantErr, err.Error())
						}

					} else {
						// No error expected
						if err != nil {
							t.Fatalf("unexpected error: %v", err)
						}

						// Verify the record is in the DB
						var got []dbRecord
						if err := store.db.Find(&got).Error; err != nil {
							t.Fatalf("failed to query db: %v", err)
						}

						if len(got) == 0 {
							t.Fatalf("expected at least 1 record, got 0")
						}

						// Find the last inserted record
						sort.Slice(got, func(i, j int) bool {
							return got[i].Id > got[j].Id
						})

						last := got[0]

						if last.SeriesId != existingSeries.ID {
							t.Errorf("expected SeriesId=%d, got %d", existingSeries.ID, last.SeriesId)
						}

						if last.Value != tc.input.Value {
							t.Errorf("expected Value=%.2f, got %.2f", tc.input.Value, last.Value)
						}
					}

				})
			}
		})
	}
}

func TestUpdateRecord(t *testing.T) {
	tcs := []struct {
		name       string
		targetID   *uint
		update     RecordUpdate
		wantErr    string
		seriesName string
		want       []Record
	}{
		{
			name:       "update value only",
			update:     RecordUpdate{Value: ptr(999.99)},
			seriesName: "btc_price",
			want: []Record{
				{Series: "btc_price", Time: getDate("2022-01-07"), Value: 999.99},
			},
		},
		{
			name:       "update time only",
			update:     RecordUpdate{Time: ptr(getDate("2022-01-08"))},
			seriesName: "eth_price",
			want: []Record{
				{Series: "eth_price", Time: getDate("2022-01-08"), Value: 100.0},
			},
		},
		{
			name:       "want error on zero id",
			targetID:   ptr(uint(0)),
			seriesName: "shiba_price",
			wantErr:    "record ID is required for update",
		},
		{
			name:       "want error on non-existing record",
			targetID:   ptr(uint(999)),
			seriesName: "banana_price",
			wantErr:    "record not found: record not found",
		},
	}

	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			dbCon := db.ConnDbName("TestSeriesUpdateRecord")
			store, err := NewRegistry(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					series := dbTimeSeries{Name: tc.seriesName}
					if err := store.db.Create(&series).Error; err != nil {
						t.Fatalf("failed to create series: %v", err)
					}

					r1 := dbRecord{
						SeriesId: series.ID,
						Time:     getDate("2022-01-07"),
						Value:    100.0,
					}
					if err := store.db.Create(&r1).Error; err != nil {
						t.Fatalf("failed to create record: %v", err)
					}

					recordId := r1.Id
					if tc.targetID != nil {
						recordId = *tc.targetID
					}
					err := store.UpdateRecord(recordId, tc.update)

					// Error expectations
					if tc.wantErr != "" {
						if err == nil {
							t.Fatalf("expected error %q, got none", tc.wantErr)
						}
						if diff := cmp.Diff(tc.wantErr, err.Error()); diff != "" {
							t.Errorf("unexpected error (-want +got):\n%s", diff)
						}
					} else {
						if err != nil {
							t.Fatalf("unexpected error: %v", err)
						}

						// Verify with ListRecords
						got, err := store.ListRecords(tc.seriesName)
						if err != nil {
							t.Fatalf("ListRecords failed: %v", err)
						}

						if diff := cmp.Diff(tc.want, got,
							cmpopts.IgnoreFields(Record{}, "Id"), // id may vary depending on DB
							cmpopts.SortSlices(func(a, b Record) bool { return a.Time.Before(b.Time) }),
						); diff != "" {
							t.Errorf("unexpected result (-want +got):\n%s", diff)
						}
					}
				})
			}
		})
	}
}
