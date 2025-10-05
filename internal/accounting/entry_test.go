package accounting

import (
	"github.com/go-bumbu/testdbs"
	"testing"
	"time"
)

var sumEntriesSample = map[int]Transaction{
	1: Income{Description: "First Income", Amount: 1.1, AccountID: 1, Date: getDate("2025-01-01")},
	2: Expense{Description: "First expense", Amount: 2.2, AccountID: 1, Date: getDate("2025-01-02")},
	3: Transfer{Description: "First transfer", OriginAmount: 3.3, OriginAccountID: 1, TargetAmount: 4.4, TargetAccountID: 2, Date: getDate("2025-01-03")},
}

func TestSumEntries(t *testing.T) {
	tcs := []struct {
		name       string
		startDate  time.Time
		endDate    time.Time
		accountID  []uint
		categoryID []uint
		entityType []entryType
		tenant     string
		wantErr    string
		want       []dbEntry
	}{
		{
			name:       "sum with valid date range",
			startDate:  getDateTime("2025-01-01 00:00:01"),
			endDate:    getDateTime("2025-01-05 00:00:00"),
			tenant:     tenant1,
			entityType: []entryType{expenseEntry},
			want:       []dbEntry{},
		},
		//{
		//	name:       "ensure transfers are not accounted",
		//	startDate:  getDateTime("2025-01-08 00:00:01"),
		//	endDate:    getDateTime("2025-01-11 00:00:00"),
		//	entityType: []EntryType{expenseEntry},
		//	tenant:     tenant1,
		//	want:       []Entry{sampleEntries[8], sampleEntries[10]},
		//},
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
		//	name:       "sum with single category ID filter",
		//	startDate:  getDateTime("2025-01-01 00:00:01"),
		//	endDate:    getDateTime("2025-02-01 00:00:00"),
		//	categoryID: []uint{1},
		//	entityType: []EntryType{expenseEntry},
		//	tenant:     tenant1,
		//	want:       []Entry{sampleEntries[11]},
		//},
		//{
		//	name:       "sum with multiple category ID filter",
		//	startDate:  getDateTime("2025-01-01 00:00:01"),
		//	endDate:    getDateTime("2025-02-01 00:00:00"),
		//	categoryID: []uint{4, 1},
		//	entityType: []EntryType{expenseEntry},
		//	tenant:     tenant1,
		//	want:       []Entry{sampleEntries[13], sampleEntries[11]},
		//},
		//{
		//	name:       "sum with multiple category ID filters and account ID filter",
		//	startDate:  getDateTime("2025-01-01 00:00:01"),
		//	endDate:    getDateTime("2025-02-01 00:00:00"),
		//	categoryID: []uint{1, 3, 4},
		//	accountID:  []uint{4, 1},
		//	entityType: []EntryType{expenseEntry},
		//	tenant:     tenant1,
		//	want:       []Entry{sampleEntries[11], sampleEntries[13]},
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
							StartDate:   tc.startDate,
							EndDate:     tc.endDate,
							CategoryIds: tc.categoryID,
							//EntryType:   tc.entityType,
							Tenant: tc.tenant,
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

						var want sumResult
						for _, item := range tc.want {
							want.Sum = want.Sum + item.Amount
							want.Count++
						}

						// check values
						if got.Sum != want.Sum {
							t.Errorf("expected sum to be %f, but got %f", want.Sum, got.Sum)
						}

						// check count
						if got.Count != want.Count {
							t.Errorf("expected count to be %d, but got %d", want.Count, got.Count)
						}

					}
				})
			}
		})
	}
}
