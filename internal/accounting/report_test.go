package accounting

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/andresbott/etna/internal/marketdata"
	"github.com/go-bumbu/testdbs"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/text/currency"
)

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
						Id: 4, ParentId: 3, Name: "Voo", Icon: "pi-dollar",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 1600, Count: 1}, currency.USD: {}, currency.CHF: {}},
					},
					{
						Id: 3, ParentId: 1, Name: "Stock benefits", Icon: "pi-chart-line",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 6700, Count: 5},
							currency.USD: {Value: 3500, Count: 2}, currency.CHF: {},
						},
					},
					{
						Id: 1, ParentId: 0, Name: "Salary", Icon: "pi-wallet",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 9100, Count: 7},
							currency.USD: {Value: 3500, Count: 2}, currency.CHF: {},
						},
					},
					{
						Id: 5, ParentId: 3, Name: "MSFT", Icon: "pi-microsoft",
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
						Id: 2, ParentId: 0, Name: "Home", Icon: "pi-home",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 4500, Count: 9}, currency.USD: {}, currency.CHF: {},
						},
					},
					{
						Id: 6, ParentId: 2, Name: "Groceries", Icon: "pi-shopping-cart",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 600, Count: 2}, currency.USD: {}, currency.CHF: {},
						},
					},
					{
						Id: 8, ParentId: 7, Name: "Electricity", Icon: "pi-bolt",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 1600, Count: 2}, currency.USD: {}, currency.CHF: {},
						},
					},
					{
						Id: 7, ParentId: 2, Name: "Bills", Icon: "pi-file",
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
						Id: 4, ParentId: 3, Name: "Voo", Icon: "pi-dollar",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 0, Count: 0}, currency.USD: {}, currency.CHF: {},
						},
					},
					{
						Id: 3, ParentId: 1, Name: "Stock benefits", Icon: "pi-chart-line",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 2500, Count: 2}, currency.USD: {}, currency.CHF: {},
						},
					},
					{
						Id: 1, ParentId: 0, Name: "Salary", Icon: "pi-wallet",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 3900, Count: 3}, currency.USD: {}, currency.CHF: {},
						},
					},
					{
						Id: 5, ParentId: 3, Name: "MSFT", Icon: "pi-microsoft",
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
						Id: 2, ParentId: 0, Name: "Home", Icon: "pi-home",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 1200, Count: 3}, currency.USD: {}, currency.CHF: {},
						},
					},
					{
						Id: 6, ParentId: 2, Name: "Groceries", Icon: "pi-shopping-cart",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 400, Count: 1}, currency.USD: {}, currency.CHF: {},
						},
					},
					{
						Id: 8, ParentId: 7, Name: "Electricity", Icon: "pi-bolt",
						Values: map[currency.Unit]CategoryReportValues{
							currency.EUR: {Value: 0, Count: 0}, currency.USD: {}, currency.CHF: {},
						},
					},
					{
						Id: 7, ParentId: 2, Name: "Bills", Icon: "pi-file",
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

			dbCon := db.ConnDbName("TestGetCategoryReport")
			store, err := NewStore(dbCon, nil)
			if err != nil {
				t.Fatal(err)
			}

			categorySampleData(t, store, sampleCategories)
			transactionSampleData(t, store, categoryReportSamples)

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					got, err := store.ReportInOutByCategory(t.Context(), tc.startDate, tc.endDate)
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

func TestAccountBalanceSingle(t *testing.T) {
	tcs := []struct {
		name      string
		date      time.Time
		tenant    string
		accountId uint
		want      AccountBalance
		wantErr   string
	}{
		{
			name:      "asert over all time on account 1",
			accountId: 1,
			date:      getDate("2022-01-30"),
			tenant:    tenant1,
			want:      AccountBalance{Date: getDate("2022-01-30"), Sum: 604.5, Count: 5},
		},
		{
			name:      "asert over all time on account 2",
			accountId: 2,
			date:      getDate("2022-01-30"),
			tenant:    tenant1,
			want:      AccountBalance{Date: getDate("2022-01-30"), Sum: 247.2, Count: 5},
		},
		{
			name:      "asert account 1 at day 4",
			accountId: 1,
			date:      getDate("2022-01-04"),
			tenant:    tenant1,
			want:      AccountBalance{Date: getDate("2022-01-04"), Sum: 374.5, Count: 4},
		},
		{
			name:      "initial balance before any action",
			accountId: 1,
			date:      getDate("2021-01-30"),
			tenant:    tenant1,
			want:      AccountBalance{Date: getDate("2021-01-30"), Sum: 0, Count: 0},
		},
	}

	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {

			dbCon := db.ConnDbName("storeCreateEntry")
			store, err := NewStore(dbCon, nil)
			if err != nil {
				t.Fatal(err)
			}

			transactionSampleData(t, store, balanceSampleData)

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					got, err := store.AccountBalanceSingle(t.Context(), tc.accountId, tc.date)
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

						if diff := cmp.Diff(got, tc.want); diff != "" {
							t.Errorf("unexpected result (-want +got):\n%s", diff)
						}
					}
				})
			}
		})
	}
}

var balanceSampleData = map[int]Transaction{

	// series of transactions
	// income
	1: Income{Description: "i1", Date: getDateTime("2022-01-01 00:00:00"), Amount: 1000, AccountID: 1},
	// some expenses
	2: Expense{Description: "e1", Date: getDateTime("2022-01-02 00:00:00"), Amount: 100, AccountID: 1},
	3: Expense{Description: "e2", Date: getDateTime("2022-01-03 00:00:00"), Amount: 25.5, AccountID: 1},
	// transfer to another account
	4: Transfer{Description: "t1", Date: getDateTime("2022-01-04 00:00:00"), OriginAmount: 500, OriginAccountID: 1, TargetAmount: 450, TargetAccountID: 2},
	// some expenses on the other account
	5: Expense{Description: "e3", Date: getDateTime("2022-01-05 00:00:00"), Amount: 50.5, AccountID: 2},
	6: Expense{Description: "e4", Date: getDateTime("2022-01-06 00:00:00"), Amount: 12.30, AccountID: 2},
	7: Income{Description: "i2", Date: getDateTime("2022-01-07 00:00:00"), Amount: 60, AccountID: 2},
	8: Transfer{Description: "t2", Date: getDateTime("2022-01-08 00:00:00"), OriginAmount: 200, OriginAccountID: 2, TargetAmount: 230, TargetAccountID: 1},
}

func TestAccountBalance(t *testing.T) {
	tcs := []struct {
		name      string
		startDate time.Time
		endDate   time.Time
		accountId uint
		count     int
		tenant    string
		want      []AccountBalance
		wantErr   string
	}{
		{
			name:      "get 1 value with all the data",
			accountId: 1,
			endDate:   getDate("2022-01-31"),
			tenant:    tenant1,
			want: []AccountBalance{
				{Date: getDate("2022-01-31"), Sum: -120, Count: 12},
			},
		},
		{
			name:      "get 1 value with all the data with count 1",
			accountId: 1,
			endDate:   getDate("2022-01-31"),
			count:     1,
			tenant:    tenant1,
			want: []AccountBalance{
				{Date: getDate("2022-01-31"), Sum: -120, Count: 12},
			},
		},
		{
			name:      "verify different accountId",
			accountId: 2,
			endDate:   getDate("2023-01-31"),
			tenant:    tenant1,
			want: []AccountBalance{
				{Date: getDate("2023-01-31"), Sum: -5, Count: 5},
			},
		},
		{
			name:      "assert initial status",
			accountId: 1,
			endDate:   getDate("2021-01-30"),
			tenant:    tenant1,
			want: []AccountBalance{
				{Date: getDate("2021-01-30"), Sum: 0, Count: 0},
			},
		},
		{
			name:      "get balance on specific date",
			accountId: 1,
			endDate:   getDate("2022-01-26"),
			tenant:    tenant1,
			want: []AccountBalance{
				{Date: getDate("2022-01-26"), Sum: -70, Count: 7},
			},
		},
		{
			name:      "get 2 start and end on count = 2",
			accountId: 1,
			startDate: getDate("2022-01-02"),
			endDate:   getDate("2022-01-31"),
			count:     2,
			tenant:    tenant1,
			want: []AccountBalance{
				{Date: getDate("2022-01-02"), Sum: -20, Count: 2},
				{Date: getDate("2022-01-31"), Sum: -120, Count: 10},
			},
		},
		{
			name:      "get 3 items",
			accountId: 1,
			startDate: getDate("2022-01-02"),
			endDate:   getDate("2022-01-31"),
			count:     3,
			tenant:    tenant1,
			want: []AccountBalance{
				{Date: getDate("2022-01-02"), Sum: -20, Count: 2},
				{Date: getDate("2022-01-16"), Sum: -40, Count: 2},
				{Date: getDate("2022-01-31"), Sum: -120, Count: 8},
			},
		},
		{
			name:      "get 3 items no start time",
			accountId: 1,
			endDate:   getDate("2022-01-31"),
			count:     3,
			tenant:    tenant1,
			want: []AccountBalance{
				{Date: getDate("2022-01-29"), Sum: -100, Count: 10},
				{Date: getDate("2022-01-30"), Sum: -110, Count: 1},
				{Date: getDate("2022-01-31"), Sum: -120, Count: 1},
			},
		},

		{
			name:      "get 5 items",
			accountId: 1,
			startDate: getDate("2022-01-02"),
			endDate:   getDate("2022-01-31"),
			count:     5,
			tenant:    tenant1,
			want: []AccountBalance{
				{Date: getDate("2022-01-02"), Sum: -20, Count: 2},
				{Date: getDate("2022-01-09"), Sum: -40, Count: 2},
				{Date: getDate("2022-01-16"), Sum: -40, Count: 0},
				{Date: getDate("2022-01-23"), Sum: -40, Count: 0},
				{Date: getDate("2022-01-31"), Sum: -120, Count: 8},
			},
		},
		{
			name:      "get same result without data",
			accountId: 1,
			startDate: getDate("2022-02-28"),
			endDate:   getDate("2022-03-05"),
			count:     4,
			tenant:    tenant1,
			want: []AccountBalance{
				{Date: getDate("2022-02-28"), Sum: -230, Count: 23},
				{Date: getDate("2022-03-02"), Sum: -240, Count: 1},
				{Date: getDate("2022-03-03"), Sum: -240, Count: 0},
				{Date: getDate("2022-03-05"), Sum: -240, Count: 0},
			},
		},
		{
			name:      "ensure negative balance",
			accountId: 3,
			endDate:   getDate("2024-01-31"),
			tenant:    tenant1,
			want: []AccountBalance{
				{Date: getDate("2024-01-31"), Sum: -90, Count: 2},
			},
		},
		{
			name:      "ensure positive balance",
			accountId: 4,
			endDate:   getDate("2024-01-31"),
			tenant:    tenant1,
			want: []AccountBalance{
				{Date: getDate("2024-01-31"), Sum: 85, Count: 3},
			},
		},
	}

	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {

			dbCon := db.ConnDbName("TestAccountBalance")
			store, err := NewStore(dbCon, nil)
			if err != nil {
				t.Fatal(err)
			}

			transactionSampleData(t, store, balanceSampleDataProgression)

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					got, err := store.AccountBalance(t.Context(), tc.accountId, tc.count, tc.startDate, tc.endDate)
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

						if diff := cmp.Diff(got, tc.want); diff != "" {
							t.Errorf("unexpected result (+want -got):\n%s", diff)
						}
					}
				})
			}
		})
	}
}

var balanceSampleDataProgression = map[int]Transaction{
	100: Expense{Description: "e1", Date: getDateTime("2022-01-01 00:00:00"), Amount: 10, AccountID: 1},
	101: Expense{Description: "e2", Date: getDateTime("2022-01-02 00:01:00"), Amount: 10, AccountID: 1},

	102: Expense{Description: "e3", Date: getDateTime("2022-01-03 00:00:00"), Amount: 10, AccountID: 1},
	103: Expense{Description: "e4", Date: getDateTime("2022-01-04 23:59:59"), Amount: 10, AccountID: 1},
	123: Expense{Description: "e24", Date: getDateTime("2022-01-24 00:00:00"), Amount: 10, AccountID: 1},
	124: Expense{Description: "e25", Date: getDateTime("2022-01-25 00:00:00"), Amount: 10, AccountID: 1},
	125: Expense{Description: "e26", Date: getDateTime("2022-01-26 00:00:00"), Amount: 10, AccountID: 1},
	126: Expense{Description: "e27", Date: getDateTime("2022-01-27 00:00:00"), Amount: 10, AccountID: 1},
	127: Expense{Description: "e28", Date: getDateTime("2022-01-28 00:00:00"), Amount: 10, AccountID: 1},
	128: Expense{Description: "e29", Date: getDateTime("2022-01-29 00:00:00"), Amount: 10, AccountID: 1},
	129: Expense{Description: "e30", Date: getDateTime("2022-01-30 00:00:00"), Amount: 10, AccountID: 1},
	130: Expense{Description: "e31", Date: getDateTime("2022-01-31 00:00:00"), Amount: 10, AccountID: 1},

	131: Expense{Description: "e32", Date: getDateTime("2022-02-01 00:00:00"), Amount: 10, AccountID: 1},
	132: Expense{Description: "e33", Date: getDateTime("2022-02-02 00:00:00"), Amount: 10, AccountID: 1},
	133: Expense{Description: "e34", Date: getDateTime("2022-02-03 00:00:00"), Amount: 10, AccountID: 1},
	134: Expense{Description: "e35", Date: getDateTime("2022-02-04 00:00:00"), Amount: 10, AccountID: 1},
	135: Expense{Description: "e36", Date: getDateTime("2022-02-05 00:00:00"), Amount: 10, AccountID: 1},
	136: Expense{Description: "e37", Date: getDateTime("2022-02-06 00:00:00"), Amount: 10, AccountID: 1},
	137: Expense{Description: "e38", Date: getDateTime("2022-02-07 00:00:00"), Amount: 10, AccountID: 1},
	138: Expense{Description: "e39", Date: getDateTime("2022-02-08 00:00:00"), Amount: 10, AccountID: 1},
	139: Expense{Description: "e40", Date: getDateTime("2022-02-09 00:00:00"), Amount: 10, AccountID: 1},
	157: Expense{Description: "e58", Date: getDateTime("2022-02-27 00:00:00"), Amount: 10, AccountID: 1},

	158: Expense{Description: "e59", Date: getDateTime("2022-02-28 00:00:00"), Amount: 10, AccountID: 1},
	159: Expense{Description: "e60", Date: getDateTime("2022-03-01 00:00:00"), Amount: 10, AccountID: 1},

	201: Expense{Description: "e1", Date: getDateTime("2023-01-01 00:00:00"), Amount: 10, AccountID: 2},
	202: Expense{Description: "e2", Date: getDateTime("2023-01-01 00:10:00"), Amount: 10, AccountID: 2},
	203: Income{Description: "e2", Date: getDateTime("2023-01-01 00:20:00"), Amount: 10, AccountID: 2},
	204: Transfer{Description: "e2", Date: getDateTime("2023-01-02 00:20:00"), OriginAmount: 10, OriginAccountID: 2, TargetAmount: 10, TargetAccountID: 1},
	205: Transfer{Description: "e2", Date: getDateTime("2023-01-02 00:20:00"), OriginAmount: 12, OriginAccountID: 1, TargetAmount: 15, TargetAccountID: 2},

	301: Income{Description: "i1_acc3", Date: getDateTime("2024-01-01 00:20:00"), Amount: 10, AccountID: 3},
	302: Expense{Description: "e1_acc3", Date: getDateTime("2024-01-02 00:00:00"), Amount: 100, AccountID: 3},

	401: Income{Description: "i1_acc4", Date: getDateTime("2024-01-01 00:20:00"), Amount: 100, AccountID: 4},
	402: Expense{Description: "e1_acc4", Date: getDateTime("2024-01-02 00:00:00"), Amount: 10, AccountID: 4},
	403: Expense{Description: "e1_acc4", Date: getDateTime("2024-01-03 00:00:00"), Amount: 5, AccountID: 4},
}

// setupInvestmentBalanceTest creates a broker with an investment account, buys stock, and ingests prices.
func setupInvestmentBalanceTest(t *testing.T, ctx context.Context, store *Store, mktStore *marketdata.Store) (investmentAccountID, cashAccountID uint) {
	t.Helper()

	providerID, err := store.CreateAccountProvider(ctx, AccountProvider{Name: "Broker"})
	if err != nil {
		t.Fatal(err)
	}
	investmentAccountID, err = store.CreateAccount(ctx, Account{
		AccountProviderID: providerID,
		Name:              "Investments",
		Currency:          currency.USD,
		Type:              InvestmentAccountType,
	})
	if err != nil {
		t.Fatal(err)
	}
	cashAccountID, err = store.CreateAccount(ctx, Account{
		AccountProviderID: providerID,
		Name:              "Cash",
		Currency:          currency.USD,
		Type:              CheckinAccountType,
	})
	if err != nil {
		t.Fatal(err)
	}

	instrumentID, err := mktStore.CreateInstrument(ctx, marketdata.Instrument{
		Symbol:   "TEST",
		Name:     "Test Corp",
		Currency: currency.USD,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Buy 10 shares at $100 on Jan 15 (total $1000)
	_, err = store.CreateTransaction(ctx, StockBuy{
		Description:         "buy TEST",
		Date:                getDateTime("2025-01-15 10:00:00"),
		Quantity:            10,
		TotalAmount:         1000,
		StockAmount:         1000,
		InstrumentID:        instrumentID,
		InvestmentAccountID: investmentAccountID,
		CashAccountID:       cashAccountID,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Ingest prices: $100 on Jan 15, $120 on Feb 1, $150 on Mar 1
	for _, pp := range []struct {
		date  string
		price float64
	}{
		{"2025-01-15", 100},
		{"2025-02-01", 120},
		{"2025-03-01", 150},
	} {
		if err := mktStore.IngestPrice(ctx, "TEST", getDate(pp.date), pp.price); err != nil {
			t.Fatal(err)
		}
	}

	return investmentAccountID, cashAccountID
}

func TestAccountBalance_InvestmentMarketValue(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			dbCon := db.ConnDbName("TestInvBalance")
			store, mktStore := newAccountingStoreWithMarketData(t, dbCon)
			ctx := t.Context()

			investmentAccountID, _ := setupInvestmentBalanceTest(t, ctx, store, mktStore)

			t.Run("single step returns market value at date", func(t *testing.T) {
				got, err := store.AccountBalance(ctx, investmentAccountID, 1, time.Time{}, getDate("2025-03-01"))
				if err != nil {
					t.Fatal(err)
				}
				if len(got) != 1 {
					t.Fatalf("expected 1 result, got %d", len(got))
				}
				// 10 shares × $150 = $1500 (same currency, no FX conversion)
				want := 1500.0
				if got[0].Sum != want {
					t.Errorf("got Sum=%v, want %v", got[0].Sum, want)
				}
			})

			t.Run("multiple steps show value evolution", func(t *testing.T) {
				// Use dates aligned with price ingestion points
				got, err := store.AccountBalance(ctx, investmentAccountID, 3, getDate("2025-01-01"), getDate("2025-03-01"))
				if err != nil {
					t.Fatal(err)
				}
				if len(got) != 3 {
					t.Fatalf("expected 3 results, got %d", len(got))
				}
				// Step 1 (Jan 1): no shares yet (bought Jan 15), value = 0
				if got[0].Sum != 0 {
					t.Errorf("step 0: got Sum=%v, want 0", got[0].Sum)
				}
				// Step 2 (~Jan 31): 10 shares, last known price is Jan 15 = $100 → $1000
				if got[1].Sum != 1000 {
					t.Errorf("step 1: got Sum=%v, want 1000", got[1].Sum)
				}
				// Step 3 (Mar 1): 10 shares, price at Mar 1 = $150 → $1500
				if got[2].Sum != 1500 {
					t.Errorf("step 2: got Sum=%v, want 1500", got[2].Sum)
				}
			})

			t.Run("before any purchase returns zero", func(t *testing.T) {
				got, err := store.AccountBalance(ctx, investmentAccountID, 0, time.Time{}, getDate("2025-01-01"))
				if err != nil {
					t.Fatal(err)
				}
				if got[0].Sum != 0 {
					t.Errorf("got Sum=%v, want 0", got[0].Sum)
				}
			})
		})
	}
}

func TestAccountBalance_InvestmentWithFX(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			dbCon := db.ConnDbName("TestInvBalanceFX")
			store, mktStore := newAccountingStoreWithMarketData(t, dbCon)
			ctx := t.Context()

			// Set main currency to CHF; instrument is in USD
			store.mainCurrency = "CHF"

			investmentAccountID, _ := setupInvestmentBalanceTest(t, ctx, store, mktStore)

			// Ingest CHF/USD FX rate: 0.90 (1 CHF = 0.90 USD, so $1 = 1/0.90 CHF)
			if err := mktStore.IngestRate(ctx, "CHF", "USD", getDate("2025-01-01"), 0.90); err != nil {
				t.Fatal(err)
			}
			if err := mktStore.IngestRate(ctx, "CHF", "USD", getDate("2025-03-01"), 0.85); err != nil {
				t.Fatal(err)
			}

			t.Run("investment value converted via FX", func(t *testing.T) {
				got, err := store.AccountBalance(ctx, investmentAccountID, 1, time.Time{}, getDate("2025-03-01"))
				if err != nil {
					t.Fatal(err)
				}
				// 10 shares × $150 = $1500 USD / 0.85 rate = 1764.71 CHF
				want := 1500.0 / 0.85
				if diff := got[0].Sum - want; diff > 0.01 || diff < -0.01 {
					t.Errorf("got Sum=%v, want ~%v", got[0].Sum, want)
				}
				if got[0].Unconverted {
					t.Error("expected Unconverted=false")
				}
			})
		})
	}
}

func TestAccountBalance_CashWithFX(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			dbCon := db.ConnDbName("TestCashBalanceFX")
			store, mktStore := newAccountingStoreWithMarketData(t, dbCon)
			ctx := t.Context()

			store.mainCurrency = "CHF"

			providerID, err := store.CreateAccountProvider(ctx, AccountProvider{Name: "Bank"})
			if err != nil {
				t.Fatal(err)
			}
			accountID, err := store.CreateAccount(ctx, Account{
				AccountProviderID: providerID,
				Name:              "USD Savings",
				Currency:          currency.USD,
				Type:              SavingsAccountType,
			})
			if err != nil {
				t.Fatal(err)
			}

			// Deposit $1000 on Jan 15
			_, err = store.CreateTransaction(ctx, Income{
				Description: "salary",
				Date:        getDateTime("2025-01-15 10:00:00"),
				Amount:      1000,
				AccountID:   accountID,
			})
			if err != nil {
				t.Fatal(err)
			}

			// FX rates: Jan = 0.90, Mar = 0.85
			if err := mktStore.IngestRate(ctx, "CHF", "USD", getDate("2025-01-01"), 0.90); err != nil {
				t.Fatal(err)
			}
			if err := mktStore.IngestRate(ctx, "CHF", "USD", getDate("2025-03-01"), 0.85); err != nil {
				t.Fatal(err)
			}

			t.Run("cash balance uses current FX rate on full balance", func(t *testing.T) {
				got, err := store.AccountBalance(ctx, accountID, 1, time.Time{}, getDate("2025-03-01"))
				if err != nil {
					t.Fatal(err)
				}
				// $1000 / 0.85 = 1176.47 CHF (converted at Mar rate, not Jan rate)
				want := 1000.0 / 0.85
				if diff := got[0].Sum - want; diff > 0.01 || diff < -0.01 {
					t.Errorf("got Sum=%v, want ~%v", got[0].Sum, want)
				}
			})

			t.Run("multi-step reflects FX changes on full balance", func(t *testing.T) {
				got, err := store.AccountBalance(ctx, accountID, 2, getDate("2025-01-20"), getDate("2025-03-01"))
				if err != nil {
					t.Fatal(err)
				}
				if len(got) != 2 {
					t.Fatalf("expected 2 results, got %d", len(got))
				}
				// Step 1 (Jan 20): $1000 / 0.90 = 1111.11 CHF
				wantJan := 1000.0 / 0.90
				if diff := got[0].Sum - wantJan; diff > 0.01 || diff < -0.01 {
					t.Errorf("step 0: got Sum=%v, want ~%v", got[0].Sum, wantJan)
				}
				// Step 2 (Mar 1): $1000 / 0.85 = 1176.47 CHF
				wantMar := 1000.0 / 0.85
				if diff := got[1].Sum - wantMar; diff > 0.01 || diff < -0.01 {
					t.Errorf("step 1: got Sum=%v, want ~%v", got[1].Sum, wantMar)
				}
			})
		})
	}
}

func TestAccountBalance_InvestmentAfterPartialSell(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			dbCon := db.ConnDbName("TestInvPartialSell")
			store, mktStore := newAccountingStoreWithMarketData(t, dbCon)
			ctx := t.Context()

			investmentAccountID, cashAccountID := setupInvestmentBalanceTest(t, ctx, store, mktStore)

			// Sell 4 shares on Feb 15 at $120 each
			_, err := store.CreateTransaction(ctx, StockSell{
				Description:         "sell TEST",
				Date:                getDateTime("2025-02-15 10:00:00"),
				Quantity:            4,
				PricePerShare:       120,
				TotalAmount:         480,
				InstrumentID:        1, // TEST instrument
				InvestmentAccountID: investmentAccountID,
				CashAccountID:       cashAccountID,
			})
			if err != nil {
				t.Fatal(err)
			}

			t.Run("value reflects remaining shares after sell", func(t *testing.T) {
				got, err := store.AccountBalance(ctx, investmentAccountID, 1, time.Time{}, getDate("2025-03-01"))
				if err != nil {
					t.Fatal(err)
				}
				// 6 remaining shares × $150 = $900
				want := 900.0
				if got[0].Sum != want {
					t.Errorf("got Sum=%v, want %v", got[0].Sum, want)
				}
			})

			t.Run("value before sell shows all shares", func(t *testing.T) {
				got, err := store.AccountBalance(ctx, investmentAccountID, 1, time.Time{}, getDate("2025-02-01"))
				if err != nil {
					t.Fatal(err)
				}
				// 10 shares × $120 = $1200
				want := 1200.0
				if got[0].Sum != want {
					t.Errorf("got Sum=%v, want %v", got[0].Sum, want)
				}
			})
		})
	}
}
