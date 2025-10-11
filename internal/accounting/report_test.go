package accounting

import (
	"github.com/go-bumbu/testdbs"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/text/currency"
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
//			store, err := NewStore(dbCon)
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
			startDate: getDate("2022-01-01"),
			endDate:   getDate("2022-01-30"),
			tenant:    tenant1,
			want: CategoryReport{
				Income: []CategoryReportItem{
					{
						Id: 0, Name: "unclassified", Description: "entries without any category",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 1900, Count: 1}, currency.USD: {}, currency.CHF: {},
						},
					},
					{
						Id: 4, ParentId: 3, Name: "Voo",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 1600, Count: 1}, currency.USD: {}, currency.CHF: {}},
					},
					{
						Id: 3, ParentId: 1, Name: "Stock benefits",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 6700, Count: 5},
							currency.USD: {Value: 3500, Count: 2}, currency.CHF: {},
						},
					},
					{
						Id: 1, ParentId: 0, Name: "Salary",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 9100, Count: 7},
							currency.USD: {Value: 3500, Count: 2}, currency.CHF: {},
						},
					},
					{
						Id: 5, ParentId: 3, Name: "MSFT",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 1300, Count: 1},
							currency.USD: {Value: 3500, Count: 2}, currency.CHF: {},
						},
					},
				},
				Expenses: []CategoryReportItem{
					{
						Id: 0, Name: "unclassified", Description: "entries without any category",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 0, Count: 0}, currency.USD: {}, currency.CHF: {},
						},
					},
					{
						Id: 2, ParentId: 0, Name: "Home",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 4500, Count: 9}, currency.USD: {}, currency.CHF: {},
						},
					},
					{
						Id: 6, ParentId: 2, Name: "Groceries",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 600, Count: 2}, currency.USD: {}, currency.CHF: {},
						},
					},
					{
						Id: 8, ParentId: 7, Name: "Electricity",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 1600, Count: 2}, currency.USD: {}, currency.CHF: {},
						},
					},
					{
						Id: 7, ParentId: 2, Name: "Bills",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 3200, Count: 5}, currency.USD: {}, currency.CHF: {},
						},
					},
				},
			},
		},
		{
			name:      "limit results by time",
			startDate: getDate("2022-01-03"),
			endDate:   getDate("2022-01-05"),
			tenant:    tenant1,
			want: CategoryReport{
				Income: []CategoryReportItem{
					{
						Id: 0, Name: "unclassified", Description: "entries without any category",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 0, Count: 0}, currency.USD: {}, currency.CHF: {},
						},
					},
					{
						Id: 4, ParentId: 3, Name: "Voo",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 0, Count: 0}, currency.USD: {}, currency.CHF: {},
						},
					},
					{
						Id: 3, ParentId: 1, Name: "Stock benefits",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 2500, Count: 2}, currency.USD: {}, currency.CHF: {},
						},
					},
					{
						Id: 1, ParentId: 0, Name: "Salary",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 3900, Count: 3}, currency.USD: {}, currency.CHF: {},
						},
					},
					{
						Id: 5, ParentId: 3, Name: "MSFT",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 1300, Count: 1}, currency.USD: {}, currency.CHF: {},
						},
					},
				},
				Expenses: []CategoryReportItem{
					{
						Id: 0, Name: "unclassified", Description: "entries without any category",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 0, Count: 0}, currency.USD: {}, currency.CHF: {},
						},
					},
					{
						Id: 2, ParentId: 0, Name: "Home",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 1200, Count: 3}, currency.USD: {}, currency.CHF: {},
						},
					},
					{
						Id: 6, ParentId: 2, Name: "Groceries",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 400, Count: 1}, currency.USD: {}, currency.CHF: {},
						},
					},
					{
						Id: 8, ParentId: 7, Name: "Electricity",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 0, Count: 0}, currency.USD: {}, currency.CHF: {},
						},
					},
					{
						Id: 7, ParentId: 2, Name: "Bills",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 800, Count: 2}, currency.USD: {}, currency.CHF: {},
						},
					},
				},
			},
		},
	}

	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {

			dbCon := db.ConnDbName("storeCreateEntry")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			categorySampleData(t, store, sampleCategories)
			transactionSampleData(t, store, categoryReportSamples)

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					got, err := store.ReportOnCategories(t.Context(), tc.startDate, tc.endDate, tc.tenant)
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

var categoryReportSamples = map[int]Transaction{

	// A bunch of expenses
	100: Expense{Description: "e1", Date: getDateTime("2022-01-01 00:00:00"), Amount: 100, AccountID: 1, CategoryID: 2},
	101: Expense{Description: "e2", Date: getDateTime("2022-01-02 00:00:00"), Amount: 200, AccountID: 1, CategoryID: 6},
	102: Expense{Description: "e3", Date: getDateTime("2022-01-03 00:00:00"), Amount: 300, AccountID: 1, CategoryID: 7},
	103: Expense{Description: "e4", Date: getDateTime("2022-01-04 00:00:00"), Amount: 400, AccountID: 1, CategoryID: 6},
	104: Expense{Description: "e5", Date: getDateTime("2022-01-05 00:00:00"), Amount: 500, AccountID: 1, CategoryID: 7},
	105: Expense{Description: "e6", Date: getDateTime("2022-01-06 00:00:00"), Amount: 600, AccountID: 1, CategoryID: 2},
	106: Expense{Description: "e7", Date: getDateTime("2022-01-07 00:00:00"), Amount: 700, AccountID: 1, CategoryID: 8},
	107: Expense{Description: "e8", Date: getDateTime("2022-01-08 00:00:00"), Amount: 800, AccountID: 1, CategoryID: 7},
	108: Expense{Description: "e9", Date: getDateTime("2022-01-09 00:00:00"), Amount: 900, AccountID: 1, CategoryID: 8},

	// A bunch of incomes
	220: Income{Description: "i1", Date: getDateTime("2022-01-01 00:00:00"), Amount: 1000, AccountID: 1, CategoryID: 1},
	221: Income{Description: "i2", Date: getDateTime("2022-01-02 00:00:00"), Amount: 1100, AccountID: 1, CategoryID: 3},
	222: Income{Description: "i3", Date: getDateTime("2022-01-03 00:00:00"), Amount: 1200, AccountID: 1, CategoryID: 3},
	223: Income{Description: "i4", Date: getDateTime("2022-01-04 00:00:00"), Amount: 1300, AccountID: 1, CategoryID: 5},
	224: Income{Description: "i5", Date: getDateTime("2022-01-05 00:00:00"), Amount: 1400, AccountID: 1, CategoryID: 1},
	225: Income{Description: "i6", Date: getDateTime("2022-01-06 00:00:00"), Amount: 1500, AccountID: 1, CategoryID: 3},
	226: Income{Description: "i7", Date: getDateTime("2022-01-07 00:00:00"), Amount: 1600, AccountID: 1, CategoryID: 4},
	227: Income{Description: "i8", Date: getDateTime("2022-01-08 00:00:00"), Amount: 1700, AccountID: 2, CategoryID: 5},
	228: Income{Description: "i9", Date: getDateTime("2022-01-09 00:00:00"), Amount: 1800, AccountID: 2, CategoryID: 5},
	229: Income{Description: "i9", Date: getDateTime("2022-01-10 00:00:00"), Amount: 1900, AccountID: 1, CategoryID: 0},

	// A bunch of Transfers
	320: Transfer{Description: "t1", Date: getDateTime("2022-01-01 00:00:00"), OriginAmount: 10, OriginAccountID: 1, TargetAmount: 11, TargetAccountID: 2},
	321: Transfer{Description: "t2", Date: getDateTime("2022-01-02 00:00:00"), OriginAmount: 20, OriginAccountID: 1, TargetAmount: 21, TargetAccountID: 2},
	322: Transfer{Description: "t3", Date: getDateTime("2022-01-03 00:00:00"), OriginAmount: 30, OriginAccountID: 1, TargetAmount: 31, TargetAccountID: 2},
	323: Transfer{Description: "t4", Date: getDateTime("2022-01-04 00:00:00"), OriginAmount: 40, OriginAccountID: 1, TargetAmount: 41, TargetAccountID: 2},
	324: Transfer{Description: "t5", Date: getDateTime("2022-01-05 00:00:00"), OriginAmount: 50, OriginAccountID: 1, TargetAmount: 51, TargetAccountID: 2},
}
