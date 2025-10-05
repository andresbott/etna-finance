package accounting

import (
	"context"
	"github.com/go-bumbu/testdbs"
	"github.com/google/go-cmp/cmp"
	"sort"
	"testing"
	"time"
)

//func TestGetAccountReport(t *testing.T) {
//	tcs := []struct {
//		name       string
//		accountIds []uint
//		endDate    time.Time
//		tenant     string
//		want       CategoryReport
//		wantErr    string
//	}{
//		{
//			name:    "simple report over all time",
//			endDate: getDateTime("2025-01-30 00:00:01"),
//			tenant:  tenant1,
//			want: CategoryReport{
//				Income: []CategoryReportItem{
//					{Id: 0, ParentId: 0, Name: "unclassified", Description: "entries without any category", Value: 2.5, Count: 1},
//					{Id: 4, ParentId: 0, Name: "in_top2", Value: 0, Count: 0},
//					{Id: 1, ParentId: 0, Name: "in_top1", Value: 553.5, Count: 2},
//					{Id: 3, ParentId: 2, Name: "in_sub2", Value: 3, Count: 1},
//					{Id: 2, ParentId: 1, Name: "in_sub1", Value: 553.5, Count: 2},
//				},
//				Expenses: []CategoryReportItem{
//					{Id: 0, ParentId: 0, Name: "unclassified", Description: "entries without any category", Value: 5317.6, Count: 11},
//					{Id: 1, ParentId: 0, Name: "ex_top1", Value: 99.6, Count: 2},
//					{Id: 3, ParentId: 2, Name: "ex_sub2", Value: 0, Count: 0},
//					{Id: 2, ParentId: 1, Name: "ex_sub1", Value: -100.4, Count: 1},
//				},
//			},
//		},
//		{
//			name:       "limit to one account ",
//			accountIds: []uint{2},
//			endDate:    getDateTime("2025-01-30 00:00:01"),
//			tenant:     tenant1,
//			want: CategoryReport{
//				Income: []CategoryReportItem{
//					{Id: 0, ParentId: 0, Name: "unclassified", Description: "entries without any category", Value: 2.5, Count: 1},
//					{Id: 4, ParentId: 0, Name: "in_top2", Value: 0, Count: 0},
//					{Id: 1, ParentId: 0, Name: "in_top1", Value: 553.5, Count: 2},
//					{Id: 3, ParentId: 2, Name: "in_sub2", Value: 3, Count: 1},
//					{Id: 2, ParentId: 1, Name: "in_sub1", Value: 553.5, Count: 2},
//				},
//				Expenses: []CategoryReportItem{
//					{Id: 0, ParentId: 0, Name: "unclassified", Description: "entries without any category", Value: 5317.6, Count: 11},
//					{Id: 1, ParentId: 0, Name: "ex_top1", Value: 99.6, Count: 2},
//					{Id: 3, ParentId: 2, Name: "ex_sub2", Value: 0, Count: 0},
//					{Id: 2, ParentId: 1, Name: "ex_sub1", Value: -100.4, Count: 1},
//				},
//			},
//		},
//	}
//
//	for _, db := range testdbs.DBs() {
//		t.Run(db.DbType(), func(t *testing.T) {
//
//			dbCon := db.ConnDbName("storeCreateEntry")
//			store, err := New(dbCon)
//			if err != nil {
//				t.Fatal(err)
//			}
//
//			sampleData(t, store)
//
//			for _, tc := range tcs {
//				t.Run(tc.name, func(t *testing.T) {
//
//					ctx := context.Background()
//					got, err := store.GetAccountReport(ctx, tc.accountIds, tc.endDate, tc.tenant)
//					if tc.wantErr != "" {
//						if err == nil {
//							t.Fatalf("expected error: %s, but got none", tc.wantErr)
//						}
//						if err.Error() != tc.wantErr {
//							t.Errorf("expected error: %s, but got %v", tc.wantErr, err.Error())
//						}
//					} else {
//						if err != nil {
//							t.Fatalf("unexpected error: %v", err)
//						}
//
//						spew.Dump(got)
//						//
//						//if diff := cmp.Diff(got, tc.want); diff != "" {
//						//	t.Errorf("unexpected result (-want +got):\n%s", diff)
//						//}
//					}
//				})
//			}
//		})
//	}
//}

func TestGetCategoryReport(t *testing.T) {
	tcs := []struct {
		name      string
		startDate time.Time
		endDate   time.Time
		tenant    string
		want      CategoryReport
		wantErr   string
	}{
		{
			name:      "simple report over all time",
			startDate: getDateTime("2024-01-01 00:00:01"),
			endDate:   getDateTime("2025-01-30 00:00:01"),
			tenant:    tenant1,
			want: CategoryReport{
				Income: []CategoryReportItem{
					{Id: 0, ParentId: 0, Name: "unclassified", Description: "entries without any category", Value: 2.5, Count: 1},
					{Id: 4, ParentId: 0, Name: "in_top2", Value: 0, Count: 0},
					{Id: 1, ParentId: 0, Name: "in_top1", Value: 553.5, Count: 2},
					{Id: 3, ParentId: 2, Name: "in_sub2", Value: 3, Count: 1},
					{Id: 2, ParentId: 1, Name: "in_sub1", Value: 553.5, Count: 2},
				},
				Expenses: []CategoryReportItem{
					{Id: 0, ParentId: 0, Name: "unclassified", Description: "entries without any category", Value: 5317.6, Count: 11},
					{Id: 1, ParentId: 0, Name: "ex_top1", Value: 99.6, Count: 2},
					{Id: 3, ParentId: 2, Name: "ex_sub2", Value: 0, Count: 0},
					{Id: 2, ParentId: 1, Name: "ex_sub1", Value: -100.4, Count: 1},
				},
			},
		},
		{
			name:      "get entries with no category", // use time filter to get a smaller sample
			startDate: getDateTime("2025-01-15 00:00:00"),
			endDate:   getDateTime("2025-01-30 00:00:01"),
			tenant:    tenant1,
			want: CategoryReport{
				Income: []CategoryReportItem{
					{Id: 0, ParentId: 0, Name: "unclassified", Description: "entries without any category", Value: 2.5, Count: 1},
					{Id: 4, ParentId: 0, Name: "in_top2", Value: 0, Count: 0},
					{Id: 1, ParentId: 0, Name: "in_top1", Value: 550.5, Count: 1},
					{Id: 3, ParentId: 2, Name: "in_sub2", Value: 0, Count: 0},
					{Id: 2, ParentId: 1, Name: "in_sub1", Value: 550.5, Count: 1},
				},
				Expenses: []CategoryReportItem{
					{Id: 0, ParentId: 0, Name: "unclassified", Description: "entries without any category", Value: 5000, Count: 2},
					{Id: 1, ParentId: 0, Name: "ex_top1"},
					{Id: 3, ParentId: 2, Name: "ex_sub2"},
					{Id: 2, ParentId: 1, Name: "ex_sub1"},
				},
			},
		},
	}

	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {

			dbCon := db.ConnDbName("storeCreateEntry")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			// todo add sample data
			//sampleData(t, store)

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					ctx := context.Background()
					got, err := store.GetCategoryReport(ctx, tc.startDate, tc.endDate, tc.tenant)
					if tc.wantErr != "" {
						if err == nil {
							t.Fatalf("expected error: %s, but got none", tc.wantErr)
						}
						if err.Error() != tc.wantErr {
							t.Errorf("expected error: %s, but got %v", tc.wantErr, err.Error())
						}
					} else {
						if err != nil {
							t.Fatalf("unexpected error: %v", err)
						}

						// sort by name
						sort.Slice(got.Income, func(i, j int) bool {
							return got.Income[i].Name >= got.Income[j].Name
						})
						sort.Slice(got.Expenses, func(i, j int) bool {
							return got.Expenses[i].Name >= got.Expenses[j].Name
						})

						if diff := cmp.Diff(got, tc.want); diff != "" {
							t.Errorf("unexpected result (-want +got):\n%s", diff)
						}
					}
				})
			}
		})
	}
}
