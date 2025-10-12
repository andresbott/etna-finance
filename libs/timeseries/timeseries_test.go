package timeseries

import (
	"fmt"
	"github.com/go-bumbu/testdbs"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"os"
	"sort"
	"testing"
	"time"
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
						Sampling: []SamplingPolicy{
							{Precision: time.Hour, Retention: 7 * 24 * time.Hour, AggregationFn: "avg"},
							{Precision: 24 * time.Hour, Retention: 30 * 24 * time.Hour, AggregationFn: "avg"},
						},
					},
					expected: TimeSeries{
						Name: "btc_price",
						Sampling: []SamplingPolicy{
							{Precision: time.Hour, Retention: 7 * 24 * time.Hour, AggregationFn: "avg"},
							{Precision: 24 * time.Hour, Retention: 30 * 24 * time.Hour, AggregationFn: "avg"},
						},
					},
				},
				{
					name: "idempotent registration",
					initial: &TimeSeries{
						Name: "btc_price",
						Sampling: []SamplingPolicy{
							{Precision: time.Hour, Retention: 7 * 24 * time.Hour, AggregationFn: "avg"},
							{Precision: 24 * time.Hour, Retention: 30 * 24 * time.Hour, AggregationFn: "avg"},
						},
					},
					input: TimeSeries{
						Name: "btc_price",
						Sampling: []SamplingPolicy{
							{Precision: time.Hour, Retention: 7 * 24 * time.Hour, AggregationFn: "avg"},
							{Precision: 24 * time.Hour, Retention: 30 * 24 * time.Hour, AggregationFn: "avg"},
						},
					},
					expected: TimeSeries{
						Name: "btc_price",
						Sampling: []SamplingPolicy{
							{Precision: time.Hour, Retention: 7 * 24 * time.Hour, AggregationFn: "avg"},
							{Precision: 24 * time.Hour, Retention: 30 * 24 * time.Hour, AggregationFn: "avg"},
						},
					},
				},
				{
					name: "update retention and policy",
					initial: &TimeSeries{
						Name: "btc_price",
						Sampling: []SamplingPolicy{
							{Precision: time.Hour, Retention: 7 * 24 * time.Hour, AggregationFn: "avg"},
						},
					},
					input: TimeSeries{
						Name: "btc_price",
						Sampling: []SamplingPolicy{
							{Precision: time.Hour, Retention: 10 * 24 * time.Hour, AggregationFn: "avg"},
						},
					},
					expected: TimeSeries{
						Name: "btc_price",
						Sampling: []SamplingPolicy{
							{Precision: time.Hour, Retention: 10 * 24 * time.Hour, AggregationFn: "avg"},
						},
					},
				},
				{
					name: "delete policy",
					initial: &TimeSeries{
						Name: "btc_price",
						Sampling: []SamplingPolicy{
							{Precision: time.Hour, Retention: 7 * 24 * time.Hour, AggregationFn: "avg"},
							{Precision: 24 * time.Hour, Retention: 30 * 24 * time.Hour, AggregationFn: "avg"},
						},
					},
					input: TimeSeries{
						Name: "btc_price",
						Sampling: []SamplingPolicy{
							{Precision: 24 * time.Hour, Retention: 30 * 24 * time.Hour, AggregationFn: "avg"},
						},
					},
					expected: TimeSeries{
						Name: "btc_price",
						Sampling: []SamplingPolicy{
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
						t.Fatalf("getTimeSeries failed: %v", err)
					}

					//sort to have comparable results
					sort.Slice(got.Sampling, func(i, j int) bool {
						return got.Sampling[i].Precision < got.Sampling[j].Precision
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

				Sampling: []SamplingPolicy{
					{Precision: time.Hour, Retention: 7 * 24 * time.Hour, AggregationFn: "avg"},
					{Precision: 24 * time.Hour, Retention: 30 * 24 * time.Hour, AggregationFn: "avg"},
				},
			}
			s2 := TimeSeries{
				Name: "eth_price",

				Sampling: []SamplingPolicy{
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
			//	sort.Slice(got[i].Sampling, func(a, b int) bool {
			//		return got[i].Sampling[a].Name > got[i].Sampling[b].Name
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
