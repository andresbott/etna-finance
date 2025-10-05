package accounting

import (
	"github.com/go-bumbu/testdbs"
	"testing"
	"time"
)

var sumEntriesSample = map[int]Transaction{

	// A bunch of expenses
	100: Expense{Description: "e1", Date: getDateTime("2022-01-01 00:00:00"), Amount: 100, AccountID: 1},
	101: Expense{Description: "e2", Date: getDateTime("2022-01-02 00:00:00"), Amount: 200, AccountID: 1},
	102: Expense{Description: "e3", Date: getDateTime("2022-01-03 00:00:00"), Amount: 300, AccountID: 1},
	103: Expense{Description: "e4", Date: getDateTime("2022-01-04 00:00:00"), Amount: 400, AccountID: 1},
	104: Expense{Description: "e5", Date: getDateTime("2022-01-05 00:00:00"), Amount: 500, AccountID: 1},
	105: Expense{Description: "e6", Date: getDateTime("2022-01-06 00:00:00"), Amount: 600, AccountID: 1},
	106: Expense{Description: "e7", Date: getDateTime("2022-01-07 00:00:00"), Amount: 700, AccountID: 1},
	107: Expense{Description: "e8", Date: getDateTime("2022-01-08 00:00:00"), Amount: 800, AccountID: 1},
	108: Expense{Description: "e9", Date: getDateTime("2022-01-09 00:00:00"), Amount: 900, AccountID: 1},

	// A bunch of incomes
	220: Income{Description: "i1", Date: getDateTime("2022-01-01 00:00:00"), Amount: 1000, AccountID: 1},
	221: Income{Description: "i2", Date: getDateTime("2022-01-02 00:00:00"), Amount: 1100, AccountID: 1},
	222: Income{Description: "i3", Date: getDateTime("2022-01-03 00:00:00"), Amount: 1200, AccountID: 1},
	223: Income{Description: "i4", Date: getDateTime("2022-01-04 00:00:00"), Amount: 1300, AccountID: 1},
	224: Income{Description: "i5", Date: getDateTime("2022-01-05 00:00:00"), Amount: 1400, AccountID: 1},
	225: Income{Description: "i6", Date: getDateTime("2022-01-06 00:00:00"), Amount: 1500, AccountID: 1},
	226: Income{Description: "i7", Date: getDateTime("2022-01-07 00:00:00"), Amount: 1600, AccountID: 1},
	227: Income{Description: "i8", Date: getDateTime("2022-01-08 00:00:00"), Amount: 1700, AccountID: 1},
	228: Income{Description: "i9", Date: getDateTime("2022-01-09 00:00:00"), Amount: 1800, AccountID: 1},

	// A bunch of Transfers
	320: Income{Description: "t1", Date: getDateTime("2022-01-01 00:00:00"), Amount: 10, AccountID: 1},
	321: Income{Description: "t2", Date: getDateTime("2022-01-02 00:00:00"), Amount: 20, AccountID: 1},
	322: Income{Description: "t3", Date: getDateTime("2022-01-03 00:00:00"), Amount: 30, AccountID: 1},
	323: Income{Description: "t4", Date: getDateTime("2022-01-04 00:00:00"), Amount: 40, AccountID: 1},
	324: Income{Description: "t5", Date: getDateTime("2022-01-05 00:00:00"), Amount: 50, AccountID: 1},
}

func TestSumEntries(t *testing.T) {
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
			want:       sumResult{Sum: 5100, Count: 3}, // 101, 102, 103
		},

		//{
		//	name:       "sum with account ID filter",
		//	startDate:  getDateTime("2025-01-01 00:00:01"),
		//	endDate:    getDateTime("2025-01-08 00:00:00"),
		//	accountID:  []uint{2},
		//	entityType: []EntryType{expenseEntry},
		//	tenant:     tenant1,
		//	want:       []Entry{sampleEntries[7], sampleEntries[4]},
		//},
		//{
		//	name:       "sum with multiple account IDs filter",
		//	startDate:  getDateTime("2023-01-01 00:00:01"),
		//	endDate:    getDateTime("2026-01-07 00:00:00"),
		//	accountID:  []uint{4, 5},
		//	entityType: []EntryType{expenseEntry},
		//	tenant:     tenant1,
		//	want:       []Entry{sampleEntries[11], sampleEntries[10]},
		//},
		//{
		//	name:       "sum for another tenant",
		//	startDate:  getDateTime("2025-01-14 00:00:01"),
		//	endDate:    getDateTime("2025-01-20 00:00:00"),
		//	entityType: []EntryType{expenseEntry},
		//	tenant:     tenant2,
		//	want:       []Entry{sampleEntries2[3], sampleEntries2[2]},
		//},
		//{
		//	name:       "expect No entries error",
		//	startDate:  getDateTime("2024-01-14 00:00:01"),
		//	endDate:    getDateTime("2024-01-20 00:00:00"),
		//	entityType: []EntryType{expenseEntry},
		//	tenant:     tenant1,
		//	wantErr:    ErrEntryNotFound.Error(),
		//},
	}

	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			dbCon := db.ConnDbName("TestSumEntries")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}
			transactionSampleData(t, store, sumEntriesSample)

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					// Call the sumEntryByCategories method
					got, err := store.sumEntryByCategories(
						t.Context(),
						sumByCategoryOpts{
							startDate:   tc.startDate,
							endDate:     tc.endDate,
							categoryIds: tc.categoryID,
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
