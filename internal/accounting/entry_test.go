package accounting

import (
	"github.com/go-bumbu/testdbs"
	"testing"
	"time"
)

var sumEntriesSample = map[int]Transaction{

	// A bunch of expenses
	100: Expense{Description: "e1", Date: getDateTime("2022-01-01 00:00:00"), Amount: 100, AccountID: 1, CategoryID: 2},
	101: Expense{Description: "e2", Date: getDateTime("2022-01-02 00:00:00"), Amount: 200, AccountID: 2, CategoryID: 6},
	102: Expense{Description: "e3", Date: getDateTime("2022-01-03 00:00:00"), Amount: 300, AccountID: 3, CategoryID: 7},
	103: Expense{Description: "e4", Date: getDateTime("2022-01-04 00:00:00"), Amount: 400, AccountID: 1, CategoryID: 2},
	104: Expense{Description: "e5", Date: getDateTime("2022-01-05 00:00:00"), Amount: 500, AccountID: 1, CategoryID: 2},
	105: Expense{Description: "e6", Date: getDateTime("2022-01-06 00:00:00"), Amount: 600, AccountID: 1, CategoryID: 2},
	106: Expense{Description: "e7", Date: getDateTime("2022-01-07 00:00:00"), Amount: 700, AccountID: 1, CategoryID: 8},
	107: Expense{Description: "e8", Date: getDateTime("2022-01-08 00:00:00"), Amount: 800, AccountID: 1, CategoryID: 2},
	108: Expense{Description: "e9", Date: getDateTime("2022-01-09 00:00:00"), Amount: 900, AccountID: 1, CategoryID: 2},

	// A bunch of incomes
	220: Income{Description: "i1", Date: getDateTime("2022-01-01 00:00:00"), Amount: 1000, AccountID: 1, CategoryID: 1},
	221: Income{Description: "i2", Date: getDateTime("2022-01-02 00:00:00"), Amount: 1100, AccountID: 1, CategoryID: 1},
	222: Income{Description: "i3", Date: getDateTime("2022-01-03 00:00:00"), Amount: 1200, AccountID: 1, CategoryID: 1},
	223: Income{Description: "i4", Date: getDateTime("2022-01-04 00:00:00"), Amount: 1300, AccountID: 1, CategoryID: 1},
	224: Income{Description: "i5", Date: getDateTime("2022-01-05 00:00:00"), Amount: 1400, AccountID: 1, CategoryID: 1},
	225: Income{Description: "i6", Date: getDateTime("2022-01-06 00:00:00"), Amount: 1500, AccountID: 2, CategoryID: 4},
	226: Income{Description: "i7", Date: getDateTime("2022-01-07 00:00:00"), Amount: 1600, AccountID: 1, CategoryID: 4},
	227: Income{Description: "i8", Date: getDateTime("2022-01-08 00:00:00"), Amount: 1700, AccountID: 2, CategoryID: 5},
	228: Income{Description: "i9", Date: getDateTime("2022-01-09 00:00:00"), Amount: 1800, AccountID: 1, CategoryID: 1},

	// A bunch of Transfers
	320: Transfer{Description: "t1", Date: getDateTime("2022-01-01 00:00:00"), OriginAmount: 10, OriginAccountID: 1, TargetAmount: 11, TargetAccountID: 2},
	321: Transfer{Description: "t2", Date: getDateTime("2022-01-02 00:00:00"), OriginAmount: 20, OriginAccountID: 1, TargetAmount: 21, TargetAccountID: 2},
	322: Transfer{Description: "t3", Date: getDateTime("2022-01-03 00:00:00"), OriginAmount: 30, OriginAccountID: 1, TargetAmount: 31, TargetAccountID: 2},
	323: Transfer{Description: "t4", Date: getDateTime("2022-01-04 00:00:00"), OriginAmount: 40, OriginAccountID: 1, TargetAmount: 41, TargetAccountID: 2},
	324: Transfer{Description: "t5", Date: getDateTime("2022-01-05 00:00:00"), OriginAmount: 50, OriginAccountID: 1, TargetAmount: 51, TargetAccountID: 2},
}

func TestSumEntriesByCategories(t *testing.T) {
	tcs := []struct {
		name       string
		startDate  time.Time
		endDate    time.Time
		accountID  []uint
		categoryID []uint
		entityType entryType
		tenant     string
		wantErr    string
		want       sumResult
	}{
		{
			name:       "sum expenses with valid date range",
			startDate:  getDate("2022-01-02"),
			endDate:    getDate("2022-01-04"),
			tenant:     tenant1,
			entityType: expenseEntry,
			want:       sumResult{Sum: 900, Count: 3}, // 101, 102, 103
		},
		{
			name:       "sum income with valid date range",
			startDate:  getDate("2022-01-07"),
			endDate:    getDate("2022-01-09"),
			tenant:     tenant1,
			entityType: incomeEntry,
			want:       sumResult{Sum: 5100, Count: 3}, // 101, 102, 103
		},
		{
			name:       "sum expenses by category ids",
			startDate:  getDate("2022-01-01"),
			endDate:    getDate("2022-01-09"),
			tenant:     tenant1,
			entityType: expenseEntry,
			categoryID: []uint{8, 7},                   // expenses 2,6,7,8
			want:       sumResult{Sum: 1000, Count: 2}, // 102, 106
		},
		{
			name:       "sum expenses by account ids",
			startDate:  getDate("2022-01-01"),
			endDate:    getDate("2022-01-09"),
			tenant:     tenant1,
			entityType: expenseEntry,
			accountID:  []uint{2, 3},
			want:       sumResult{Sum: 500, Count: 2}, // 101, 102
		},
		{
			name:       "sum expenses by account and category ids",
			startDate:  getDate("2022-01-01"),
			endDate:    getDate("2022-01-09"),
			tenant:     tenant1,
			entityType: incomeEntry,
			accountID:  []uint{2},
			categoryID: []uint{4, 5},
			want:       sumResult{Sum: 3200, Count: 2}, // 225, 227
		},
		{
			name:       "expect empty result for another tenant",
			startDate:  getDate("2022-01-01"),
			endDate:    getDate("2022-01-09"),
			tenant:     tenant2,
			entityType: expenseEntry,
			categoryID: []uint{8, 7},
			want:       sumResult{Sum: 0, Count: 0}, // 102, 106
		},
	}

	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			dbCon := db.ConnDbName("TestSumEntries")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}
			categorySampleData(t, store, sampleCategories)
			transactionSampleData(t, store, sumEntriesSample)

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					// Call the sumEntries method
					got, err := store.sumEntries(
						t.Context(),
						sumEntriesOpts{
							startDate:   tc.startDate,
							endDate:     tc.endDate,
							categoryIds: tc.categoryID,
							accountIds:  tc.accountID,
							entryType:   tc.entityType,
							tenant:      tc.tenant,
						},
					)

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

						// check values
						if got.Sum != tc.want.Sum {
							t.Errorf("expected sum to be %f, but got %f", tc.want.Sum, got.Sum)
						}

						// check count
						if got.Count != tc.want.Count {
							t.Errorf("expected count to be %d, but got %d", tc.want.Count, got.Count)
						}
					}
				})
			}
		})
	}
}
