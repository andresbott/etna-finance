package timeseries

import (
	"fmt"
	"os"
	"sort"
	"testing"
	"testing/synctest"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-bumbu/testdbs"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// TestMain modifies how test are run,
// it makes sure that the needed DBs are ready and does cleanup in the end.
func TestMain(m *testing.M) {
	testdbs.InitDBS()
	// main block that runs tests
	code := m.Run()
	_ = testdbs.Clean()
	os.Exit(code)
}

func TestRegisterSeries(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {

			tcs := []struct {
				name     string
				initial  *TimeSeries // optional initial insert
				input    TimeSeries  // input to RegisterSeries
				expected TimeSeries  // expected result after registration
			}{
				{
					name: "create new series",
					input: TimeSeries{
						Name: "btc_price",
						DownSampling: []SamplingPolicy{
							{Precision: time.Hour, Retention: 7 * 24 * time.Hour, AggregationFn: "avg"},
							{Precision: 24 * time.Hour, Retention: 30 * 24 * time.Hour, AggregationFn: "avg"},
						},
					},
					expected: TimeSeries{
						Name: "btc_price",
						DownSampling: []SamplingPolicy{
							{Precision: time.Hour, Retention: 7 * 24 * time.Hour, AggregationFn: "avg"},
							{Precision: 24 * time.Hour, Retention: 30 * 24 * time.Hour, AggregationFn: "avg"},
						},
					},
				},
				{
					name: "idempotent registration", // policies are not duplicated
					initial: &TimeSeries{
						Name: "btc_price",
						DownSampling: []SamplingPolicy{
							{Precision: time.Hour, Retention: 7 * 24 * time.Hour, AggregationFn: "avg"},
							{Precision: 24 * time.Hour, Retention: 30 * 24 * time.Hour, AggregationFn: "avg"},
						},
					},
					input: TimeSeries{
						Name: "btc_price",
						DownSampling: []SamplingPolicy{
							{Precision: time.Hour, Retention: 7 * 24 * time.Hour, AggregationFn: "avg"},
							{Precision: 24 * time.Hour, Retention: 30 * 24 * time.Hour, AggregationFn: "avg"},
						},
					},
					expected: TimeSeries{
						Name: "btc_price",
						DownSampling: []SamplingPolicy{
							{Precision: time.Hour, Retention: 7 * 24 * time.Hour, AggregationFn: "avg"},
							{Precision: 24 * time.Hour, Retention: 30 * 24 * time.Hour, AggregationFn: "avg"},
						},
					},
				},
				{
					name: "update retention and policy",
					initial: &TimeSeries{
						Name: "btc_price",
						DownSampling: []SamplingPolicy{
							{Precision: time.Hour, Retention: 7 * 24 * time.Hour, AggregationFn: "avg"},
						},
					},
					input: TimeSeries{
						Name: "btc_price",
						DownSampling: []SamplingPolicy{
							{Precision: time.Hour, Retention: 10 * 24 * time.Hour, AggregationFn: "avg"},
						},
					},
					expected: TimeSeries{
						Name: "btc_price",
						DownSampling: []SamplingPolicy{
							{Precision: time.Hour, Retention: 10 * 24 * time.Hour, AggregationFn: "avg"},
						},
					},
				},
				{
					name: "delete policy",
					initial: &TimeSeries{
						Name: "btc_price",
						DownSampling: []SamplingPolicy{
							{Precision: time.Hour, Retention: 7 * 24 * time.Hour, AggregationFn: "avg"},
							{Precision: 24 * time.Hour, Retention: 30 * 24 * time.Hour, AggregationFn: "avg"},
						},
					},
					input: TimeSeries{
						Name: "btc_price",
						DownSampling: []SamplingPolicy{
							{Precision: 24 * time.Hour, Retention: 30 * 24 * time.Hour, AggregationFn: "avg"},
						},
					},
					expected: TimeSeries{
						Name: "btc_price",
						DownSampling: []SamplingPolicy{
							{Precision: 24 * time.Hour, Retention: 30 * 24 * time.Hour, AggregationFn: "avg"},
						},
					},
				},
			}

			dbCon := db.ConnDbName("TestRegisterSeries")
			store, err := NewRegistry(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					if tc.initial != nil {
						if err := store.RegisterSeries(*tc.initial); err != nil {
							t.Fatalf("setup initial failed: %v", err)
						}
					}

					if err := store.RegisterSeries(tc.input); err != nil {
						t.Fatalf("RegisterSeries failed: %v", err)
					}

					got, err := store.GetSeries(tc.input.Name)
					if err != nil {
						t.Fatalf("GetSeries failed: %v", err)
					}

					//sort to have comparable results
					sort.Slice(got.DownSampling, func(i, j int) bool {
						return got.DownSampling[i].Precision < got.DownSampling[j].Precision
					})

					if diff := cmp.Diff(tc.expected, got); diff != "" {
						t.Errorf("unexpected series state (-want +got):\n%s", diff)
					}
				})
			}
		})
	}
}

func TestListSeries(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			dbCon := db.ConnDbName("TestListSeries")
			store, err := NewRegistry(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			// Arrange: Create some sample data
			s1 := TimeSeries{
				Name: "btc_price",
				DownSampling: []SamplingPolicy{
					{Precision: time.Hour, Retention: 7 * 24 * time.Hour, AggregationFn: "avg"},
					{Precision: 24 * time.Hour, Retention: 30 * 24 * time.Hour, AggregationFn: "avg"},
				},
			}
			s2 := TimeSeries{
				Name: "eth_price",
				DownSampling: []SamplingPolicy{
					{Precision: time.Hour, Retention: 14 * 24 * time.Hour, AggregationFn: "avg"},
				},
			}

			if err := store.RegisterSeries(s1); err != nil {
				t.Fatalf("failed to insert s1: %v", err)
			}
			if err := store.RegisterSeries(s2); err != nil {
				t.Fatalf("failed to insert s2: %v", err)
			}

			got, err := store.ListSeries()
			if err != nil {
				t.Fatalf("ListSeries failed: %v", err)
			}

			// Sort the policies for comparison (in case DB doesn't guarantee order)
			//for i := range got {
			//	sort.Slice(got[i].DownSampling, func(a, b int) bool {
			//		return got[i].DownSampling[a].Name > got[i].DownSampling[b].Name
			//	})
			//}

			want := []TimeSeries{s1, s2}

			// Assert
			if diff := cmp.Diff(want, got,
				cmpopts.SortSlices(func(a, b TimeSeries) bool { return a.Name < b.Name }),
			); diff != "" {
				t.Errorf("unexpected ListSeries result (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCleanupSeries(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {

			tcs := []struct {
				name     string
				input    []Record
				progress time.Duration
				expected []Record
			}{
				{
					name: "ensue data is purged",
					input: []Record{
						{Time: getdatetime("2000-01-01 01:00:00"), Value: 10.50},
						{Time: getdatetime("2000-01-02 01:00:00"), Value: 11.50},
						{Time: getdatetime("2000-01-03 01:00:00"), Value: 12.50},
					},
					progress: 30 * 24 * time.Hour, // 30 days
					expected: []Record{
						{Time: getdatetime("2000-01-03 01:00:00"), Value: 12.50},
					},
				},
			}

			dbCon := db.ConnDbName("TestCleanupSeries")
			store, err := NewRegistry(dbCon)
			if err != nil {
				t.Fatal(err)
			}
			i := 0

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					synctest.Test(t, func(t *testing.T) {
						i++
						seriesName := fmt.Sprintf("btc_price_%d", i)
						series := TimeSeries{
							Name: seriesName,
							DownSampling: []SamplingPolicy{
								// hour precision for 7 days
								{Precision: time.Hour, Retention: 7 * 24 * time.Hour, AggregationFn: "avg"},
								// day precision for 30 days
								{Precision: 24 * time.Hour, Retention: 30 * 24 * time.Hour, AggregationFn: "avg"},
							},
						}

						if err := store.RegisterSeries(series); err != nil {
							t.Fatalf("RegisterSeries failed: %v", err)
						}

						for _, record := range tc.input {
							record.Series = seriesName
							err = store.Ingest(record)
							if err != nil {
								t.Fatalf("ingest failed: %v", err)
							}
						}

						// progress the time
						time.Sleep(tc.progress)
						spew.Dump(time.Now())

						err := store.Cleanup(t.Context())
						if err != nil {
							t.Fatalf("Cleanup failed: %v", err)
						}

						got, err := store.ListRecords(seriesName)
						if err != nil {
							t.Fatalf("ListSeries failed: %v", err)
						}
						spew.Dump(got)

						if diff := cmp.Diff(tc.expected, got); diff != "" {
							t.Errorf("unexpected series state (-want +got):\n%s", diff)
						}
					})
				})
			}
		})
	}
}

// returns a pointer to a specific type
func ptr[T any](v T) *T {
	return &v
}
func getDate(timeStr string) time.Time {
	// Parse the string based on the provided layout
	parsedTime, err := time.Parse("2006-01-02", timeStr)
	if err != nil {
		panic(fmt.Errorf("unable to parse time: %v", err))
	}
	return parsedTime
}

func getdatetime(datetimeStr string) time.Time {
	parsedTime, err := time.Parse("2006-01-02 15:04:05", datetimeStr)
	if err != nil {
		panic(fmt.Errorf("unable to parse datetime: %v", err))
	}
	return parsedTime.UTC()
}
