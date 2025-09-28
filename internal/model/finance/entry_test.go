package finance

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-bumbu/testdbs"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var ignoreEntryFields = cmpopts.IgnoreFields(Entry{},
	"Id", "TargetAccountCurrency", "OriginAccountCurrency")

func TestCreateEntry(t *testing.T) {
	tcs := []struct {
		name    string
		input   Entry
		tenant  string
		wantErr string
	}{
		{
			name:   "create valid entry",
			tenant: tenant1,
			input:  Entry{Description: "Salary", TargetAmount: 1000, TargetAccountID: 1, Date: date1, Type: ExpenseEntry},
		},
		{
			name:    "want error on empty description",
			tenant:  tenant1,
			input:   Entry{Description: "", TargetAmount: 1000, Date: date1, Type: ExpenseEntry},
			wantErr: "description cannot be empty",
		},
		{
			name:    "want error on zero amount",
			tenant:  tenant1,
			input:   Entry{Description: "Groceries", TargetAmount: 0, Date: date1, Type: ExpenseEntry},
			wantErr: "target amount cannot be empty",
		},
		{
			name:    "want error on zero date",
			tenant:  tenant1,
			input:   Entry{Description: "Investment", TargetAmount: 500, TargetAccountID: 1, Date: time.Time{}, Type: ExpenseEntry},
			wantErr: "date cannot be zero",
		},
		{
			name:    "want error on empty type",
			tenant:  tenant1,
			input:   Entry{Description: "Groceries", TargetAmount: 0, Date: date1},
			wantErr: "entry type cannot be empty",
		},
	}

	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {

			dbCon := db.ConnDbName("storeCreateEntry")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					ctx := context.Background()
					id, err := store.CreateEntry(ctx, tc.input, tc.tenant)

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

						if id == 0 {
							t.Errorf("expected valid entry ID, but got 0")
						}

						got, err := store.GetEntry(ctx, id, tc.tenant)
						if err != nil {
							t.Fatalf("expected entry to be found, but got error: %v", err)
						}

						if diff := cmp.Diff(got, tc.input, ignoreEntryFields); diff != "" {
							t.Errorf("unexpected result (-want +got):\n%s", diff)
						}
					}
				})
			}
		})
	}
}

func TestGetEntry(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			tcs := []struct {
				name        string
				checkTenant string
				checkId     uint
				want        Entry
				wantErr     string
			}{
				{
					name:        "get existing entry",
					checkTenant: tenant1,
					checkId:     1,
					want:        sampleEntries[0],
				},
				{
					name:        "want error when reading from different tenant",
					checkTenant: tenant2,
					wantErr:     "entry not found",
				},
			}

			dbCon := db.ConnDbName("storeGetEntry")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}
			sampleData(t, store) // note: test operates on one set of data

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					ctx := context.Background()
					got, err := store.GetEntry(ctx, tc.checkId, tc.checkTenant)
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

						if diff := cmp.Diff(got, tc.want, ignoreEntryFields); diff != "" {
							t.Errorf("unexpected result (-want +got):\n%s", diff)
						}
					}
				})
			}
		})
	}
}

func TestDeleteEntry(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			tcs := []struct {
				name         string
				deleteID     uint
				deleteTenant string
				wantErr      string
			}{
				{
					name:         "delete existing entry",
					deleteID:     1,
					deleteTenant: tenant1,
				},
				{
					name:         "error when deleting non-existent entry",
					deleteTenant: tenant1,
					deleteID:     9999,
					wantErr:      "entry not found",
				},
				{
					name:         "error when deleting entry  for other tenant",
					deleteTenant: tenant2,
					wantErr:      "entry not found",
				},
			}

			dbCon := db.ConnDbName("storeGetEntry")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}
			sampleData(t, store) // note: test operates on one set of data

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					ctx := context.Background()

					err = store.DeleteEntry(ctx, tc.deleteID, tc.deleteTenant)
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

						_, err = store.GetEntry(ctx, tc.deleteID, tc.deleteTenant)
						if err == nil {
							t.Fatalf("expected NotFoundErr, but got entry")
						}
					}
				})
			}
		})
	}
}

func TestUpdateEntry(t *testing.T) {
	tcs := []struct {
		name          string
		updateID      uint
		updateTenant  string
		updatePayload EntryUpdatePayload
		want          Entry
		wantErr       string
	}{
		{
			name:          "update existing entry description",
			updateID:      1,
			updateTenant:  tenant1,
			updatePayload: EntryUpdatePayload{Description: ptr("Updated Entry Description")},
			want:          Entry{Description: "Updated Entry Description", TargetAmount: 1, TargetAccountID: 1, Type: ExpenseEntry, Date: getTime("2025-01-01 00:00:00")},
		},
		{
			name:          "update entry target amount",
			updateID:      2,
			updateTenant:  tenant1,
			updatePayload: EntryUpdatePayload{TargetAmount: ptr(float64(200))},
			want: Entry{Description: "e1", TargetAmount: 200, Type: ExpenseEntry, Date: getTime("2025-01-02 00:00:00"),
				TargetAccountID: 1},
		},
		{
			name:          "update entry description and target amount",
			updateID:      3,
			updateTenant:  tenant1,
			updatePayload: EntryUpdatePayload{Description: ptr("Updated Entry Description"), TargetAmount: ptr(float64(300))},
			want: Entry{Description: "Updated Entry Description", TargetAmount: 300, Type: IncomeEntry, Date: getTime("2025-01-03 00:00:00"),
				TargetAccountID: 2, CategoryId: 3},
		},
		{
			name:          "update entry date",
			updateID:      4,
			updateTenant:  tenant1,
			updatePayload: EntryUpdatePayload{Date: &date2},
			want:          Entry{Description: "e3", TargetAmount: -4.1, TargetAccountID: 1, Type: ExpenseEntry, Date: date2},
		},
		{
			name:          "error when updating non-existent entry",
			updateTenant:  tenant1,
			updateID:      9999,
			updatePayload: EntryUpdatePayload{Description: ptr("Updated Entry Description")},
			wantErr:       "entry not found",
		},
		{
			name:          "error when updating another tenant's entry",
			updateTenant:  tenant2,
			updateID:      1,
			updatePayload: EntryUpdatePayload{Description: ptr("Updated Entry Description")},
			wantErr:       "entry not found",
		},
	}

	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {

			dbCon := db.ConnDbName("TestUpdateEntry")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}
			sampleData(t, store)

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					ctx := context.Background()

					err = store.UpdateEntry(ctx, tc.updatePayload, tc.updateID, tc.updateTenant)
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

						got, err := store.GetEntry(ctx, tc.updateID, tc.updateTenant)
						if err != nil {
							t.Fatalf("expected entry to be found, but got error: %v", err)
						}

						if diff := cmp.Diff(got, tc.want, ignoreEntryFields); diff != "" {
							t.Errorf("unexpected result (-want +got):\n%s", diff)
						}
					}
				})
			}
		})
	}
}

func getTime(timeStr string) time.Time {
	// Parse the string based on the provided layout
	parsedTime, err := time.Parse("2006-01-02 15:04:05", timeStr)
	if err != nil {
		panic(fmt.Errorf("unable to parse time: %v", err))

	}
	return parsedTime
}

func TestListEntries(t *testing.T) {

	tcs := []struct {
		name       string
		startDate  time.Time
		endDate    time.Time
		accountID  []int
		categoryID []int
		limit      int
		page       int
		tenant     string
		wantErr    string
		want       []Entry
	}{
		{
			name:      "search with valid date range",
			startDate: getTime("2025-01-01 00:00:01"),
			endDate:   getTime("2025-01-03 00:00:00"),
			tenant:    tenant1,
			want:      []Entry{sampleEntries[2], sampleEntries[1]},
		},
		{
			name:      "verify transfer type is correct",
			startDate: getTime("2025-01-09 00:00:01"),
			endDate:   getTime("2025-01-11 00:00:00"),
			tenant:    tenant1,
			limit:     2,
			want:      []Entry{sampleEntries[10], sampleEntries[9]},
		},
		{
			name:      "search with account ID filter",
			startDate: getTime("2025-01-01 00:00:01"),
			endDate:   getTime("2025-01-07 00:00:00"),
			accountID: []int{2},
			tenant:    tenant1,
			want:      []Entry{sampleEntries[4], sampleEntries[2]},
		},
		{
			name:      "search with multiple account IDs filter",
			startDate: getTime("2023-01-01 00:00:01"),
			endDate:   getTime("2026-01-07 00:00:00"),
			accountID: []int{4, 5},
			tenant:    tenant1,
			want:      []Entry{sampleEntries[11], sampleEntries[10]},
		},
		{
			name:       "search with single category ID filter",
			startDate:  getTime("2025-01-01 00:00:01"),
			endDate:    getTime("2025-02-01 00:00:00"),
			categoryID: []int{1},
			tenant:     tenant1,
			want:       []Entry{sampleEntries[11]},
		},
		{
			name:       "search with multiple category ID filter",
			startDate:  getTime("2025-01-01 00:00:01"),
			endDate:    getTime("2025-02-01 00:00:00"),
			categoryID: []int{4, 3},
			tenant:     tenant1,
			want:       []Entry{sampleEntries[13], sampleEntries[9], sampleEntries[2]},
		},

		{
			name:       "search with multiple category ID filters and account ID filter",
			startDate:  getTime("2025-01-01 00:00:01"),
			endDate:    getTime("2025-02-01 00:00:00"),
			categoryID: []int{1, 2, 3},
			accountID:  []int{2},
			tenant:     tenant1,
			want:       []Entry{sampleEntries[9], sampleEntries[2]},
		},
		{
			name:      "search with limit",
			startDate: getTime("2025-01-01 00:00:01"),
			endDate:   getTime("2025-01-09 00:00:00"),
			accountID: []int{2},
			tenant:    tenant1,
			limit:     2,
			want:      []Entry{sampleEntries[7], sampleEntries[4]},
		},
		{
			name:      "search with limit and page",
			startDate: getTime("2025-01-01 00:00:01"),
			endDate:   getTime("2025-01-09 00:00:00"),
			accountID: []int{2},
			tenant:    tenant1,
			limit:     2,
			page:      2,
			want:      []Entry{sampleEntries[2]},
		},
		{
			name:      "search for another tenant",
			startDate: getTime("2025-01-14 00:00:01"),
			endDate:   getTime("2025-01-20 00:00:00"),
			tenant:    tenant2,
			limit:     2,
			want:      []Entry{sampleEntries2[3], sampleEntries2[2]},
		},
	}

	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			dbCon := db.ConnDbName("TestSearchEntries")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}
			sampleData(t, store) // note: test operates on one set of data

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					// Call the ListEntries method
					got, err := store.ListEntries(
						context.Background(),
						ListOpts{
							StartDate:   tc.startDate,
							EndDate:     tc.endDate,
							AccountIds:  tc.accountID,
							CategoryIds: tc.categoryID,
							Limit:       tc.limit,
							Page:        tc.page,
							Tenant:      tc.tenant,
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

						if diff := cmp.Diff(got, tc.want, ignoreEntryFields); diff != "" {
							t.Errorf("unexpected result (-want +got):\n%s", diff)
						}
					}

				})
			}
		})
	}
}

func TestSumEntries(t *testing.T) {
	tcs := []struct {
		name       string
		startDate  time.Time
		endDate    time.Time
		accountID  []int
		categoryID []uint
		entityType EntryType
		tenant     string
		wantErr    string
		want       []Entry
	}{
		{
			name:       "sum with valid date range",
			startDate:  getTime("2025-01-01 00:00:01"),
			endDate:    getTime("2025-01-05 00:00:00"),
			tenant:     tenant1,
			entityType: ExpenseEntry,
			want:       []Entry{sampleEntries[1], sampleEntries[3], sampleEntries[4]},
		},
		{
			name:       "ensure transfers are not accounted",
			startDate:  getTime("2025-01-08 00:00:01"),
			endDate:    getTime("2025-01-11 00:00:00"),
			entityType: ExpenseEntry,
			tenant:     tenant1,
			want:       []Entry{sampleEntries[8], sampleEntries[10]},
		},
		{
			name:       "sum with account ID filter",
			startDate:  getTime("2025-01-01 00:00:01"),
			endDate:    getTime("2025-01-08 00:00:00"),
			accountID:  []int{2},
			entityType: ExpenseEntry,
			tenant:     tenant1,
			want:       []Entry{sampleEntries[7], sampleEntries[4]},
		},
		{
			name:       "sum with multiple account IDs filter",
			startDate:  getTime("2023-01-01 00:00:01"),
			endDate:    getTime("2026-01-07 00:00:00"),
			accountID:  []int{4, 5},
			entityType: ExpenseEntry,
			tenant:     tenant1,
			want:       []Entry{sampleEntries[11], sampleEntries[10]},
		},
		{
			name:       "sum with single category ID filter",
			startDate:  getTime("2025-01-01 00:00:01"),
			endDate:    getTime("2025-02-01 00:00:00"),
			categoryID: []uint{1},
			entityType: ExpenseEntry,
			tenant:     tenant1,
			want:       []Entry{sampleEntries[11]},
		},
		{
			name:       "sum with multiple category ID filter",
			startDate:  getTime("2025-01-01 00:00:01"),
			endDate:    getTime("2025-02-01 00:00:00"),
			categoryID: []uint{4, 1},
			entityType: ExpenseEntry,
			tenant:     tenant1,
			want:       []Entry{sampleEntries[13], sampleEntries[11]},
		},
		{
			name:       "sum with multiple category ID filters and account ID filter",
			startDate:  getTime("2025-01-01 00:00:01"),
			endDate:    getTime("2025-02-01 00:00:00"),
			categoryID: []uint{1, 3, 4},
			accountID:  []int{4, 1},
			entityType: ExpenseEntry,
			tenant:     tenant1,
			want:       []Entry{sampleEntries[11], sampleEntries[13]},
		},
		{
			name:       "sum for another tenant",
			startDate:  getTime("2025-01-14 00:00:01"),
			endDate:    getTime("2025-01-20 00:00:00"),
			entityType: ExpenseEntry,
			tenant:     tenant2,
			want:       []Entry{sampleEntries2[3], sampleEntries2[2]},
		},
		{
			name:       "expect No entries error",
			startDate:  getTime("2024-01-14 00:00:01"),
			endDate:    getTime("2024-01-20 00:00:00"),
			entityType: ExpenseEntry,
			tenant:     tenant1,
			wantErr:    ErrEntryNotFound.Error(),
		},
	}

	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			dbCon := db.ConnDbName("TestSumEntries")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}
			sampleData(t, store) // note: test operates on one set of data

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					// Call the SumEntries method
					got, err := store.SumEntries(
						context.Background(),
						SumOpts{
							StartDate:   tc.startDate,
							EndDate:     tc.endDate,
							AccountIds:  tc.accountID,
							CategoryIds: tc.categoryID,
							EntryType:   tc.entityType,
							Tenant:      tc.tenant,
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

						var want float64
						for _, item := range tc.want {
							want = want + item.TargetAmount
						}

						if got != want {
							t.Errorf("expected sum to be %f, but got %f", want, got)
						}
					}
				})
			}
		})
	}
}
