package timeseries

import (
	"github.com/go-bumbu/testdbs"
	"github.com/google/go-cmp/cmp"
	"testing"
	"time"
)

var testRecords = []Record{
	{Series: "btc_price", Time: getDate("2025-01-01"), Value: 10000},
	{Series: "btc_price", Time: getDate("2025-01-02"), Value: 11000},
	{Series: "btc_price", Time: getDate("2025-01-03"), Value: 12000},
	{Series: "btc_price", Time: getDate("2025-01-04"), Value: 13000},
	{Series: "btc_price", Time: getDate("2025-01-05"), Value: 14000},
	{Series: "btc_price", Time: getDate("2025-01-06"), Value: 15000},
	{Series: "btc_price", Time: getDate("2025-01-07"), Value: 16000},
	{Series: "btc_price", Time: getDate("2025-01-08"), Value: 17000},
	{Series: "btc_price", Time: getDate("2025-01-09"), Value: 18000},
	{Series: "btc_price", Time: getDate("2025-01-10"), Value: 19000},
}

func TestRecordAt(t *testing.T) {
	tcs := []struct {
		name      string
		series    string
		queryTime time.Time
		wantValue float64
		wantErr   string
	}{
		{
			name:      "before first record",
			series:    "btc_price",
			queryTime: getDate("2024-12-31"),
			wantErr:   "record not found",
		},
		{
			name:      "exact match with 5th record",
			series:    "btc_price",
			queryTime: getDate("2025-01-05"),
			wantValue: 14000,
		},
		{
			name:      "between 6th and 7th record",
			series:    "btc_price",
			queryTime: getDate("2025-01-06").Add(12 * time.Hour),
			wantValue: 15000,
		},
		{
			name:      "after last record",
			series:    "btc_price",
			queryTime: getDate("2025-01-11"),
			wantValue: 19000,
		},
		{
			name:      "series not found",
			series:    "unknown_series",
			queryTime: getDate("2025-01-05"),
			wantErr:   "series not found",
		},
	}

	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			dbCon := db.ConnDbName("TestRecordAt")
			store, err := NewRegistry(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			// Arrange: create the series
			s := dbTimeSeries{Name: "btc_price"}
			if err := store.db.Create(&s).Error; err != nil {
				t.Fatalf("failed to create series: %v", err)
			}

			// Ingest all fixed records
			for _, r := range testRecords {
				if err := store.Ingest(r); err != nil {
					t.Fatalf("failed to ingest record %v: %v", r, err)
				}
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					got, err := store.RecordAt(tc.series, tc.queryTime)

					if tc.wantErr != "" {
						if err == nil {
							t.Fatalf("expected error %q but got none", tc.wantErr)
						}
						if diff := cmp.Diff(tc.wantErr, err.Error()); diff != "" {
							t.Errorf("unexpected error (-want +got):\n%s", diff)
						}
						return
					}

					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					if got.Value != tc.wantValue {
						t.Errorf("unexpected value: want %.2f, got %.2f", tc.wantValue, got.Value)
					}
				})
			}
		})
	}
}
