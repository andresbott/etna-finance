package finance

import (
	"context"
	"fmt"
	"golang.org/x/text/currency"
	"testing"
	"time"

	"github.com/go-bumbu/testdbs"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var date1 = time.Date(2025, time.March, 15, 0, 0, 0, 0, time.UTC)
var date2 = time.Date(2025, time.March, 16, 0, 0, 0, 0, time.UTC)

var ignoreEntryFields = cmpopts.IgnoreFields(Entry{},
	"Id", "TargetAccountCurrency", "OriginAccountCurrency")

func TestCreateEntry(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			tcs := []struct {
				name    string
				input   Entry
				tenant  string
				wantErr string
			}{
				{
					name:   "create valid entry",
					tenant: tenant1,
					input:  Entry{Description: "Salary", Amount: 1000, Date: date1, Type: ExpenseEntry},
				},
				{
					name:    "want error on empty description",
					tenant:  tenant1,
					input:   Entry{Description: "", Amount: 1000, Date: date1, Type: ExpenseEntry},
					wantErr: "description cannot be empty",
				},
				{
					name:    "want error on zero amount",
					tenant:  tenant1,
					input:   Entry{Description: "Groceries", Amount: 0, Date: date1, Type: ExpenseEntry},
					wantErr: "amount cannot be empty",
				},
				{
					name:    "want error on zero date",
					tenant:  tenant1,
					input:   Entry{Description: "Investment", Amount: 500, Date: time.Time{}, Type: ExpenseEntry},
					wantErr: "date cannot be zero",
				},
				{
					name:    "want error on empty type",
					tenant:  tenant1,
					input:   Entry{Description: "Groceries", Amount: 0, Date: date1},
					wantErr: "entry type cannot be empty",
				},
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					dbCon := db.ConnDbName("storeCreateEntry")
					store, err := New(dbCon)
					if err != nil {
						t.Fatal(err)
					}

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
				name         string
				create       Entry
				createTenant string
				checkTenant  string
				want         Entry
				wantErr      string
			}{
				{
					name:         "get existing entry",
					createTenant: tenant1,
					create:       Entry{Description: "Salary", Amount: 1000, Date: date1, Type: ExpenseEntry, TargetAccountID: 2},
					checkTenant:  tenant1,
					want:         Entry{Description: "Salary", Amount: 1000, Date: date1, Type: ExpenseEntry, TargetAccountID: 2},
				},
				{
					name:         "want error when reading from different tenant",
					createTenant: tenant1,
					create:       Entry{Description: "Salary", Amount: 1000, Date: date1, Type: ExpenseEntry},
					checkTenant:  tenant2,
					wantErr:      "entry not found",
				},
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					dbCon := db.ConnDbName("storeGetEntry")
					store, err := New(dbCon)
					if err != nil {
						t.Fatal(err)
					}

					ctx := context.Background()
					id, err := store.CreateEntry(ctx, tc.create, tc.createTenant)
					if err != nil {
						t.Fatal(err)
					}

					got, err := store.GetEntry(ctx, id, tc.checkTenant)
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
				create       Entry
				createTenant string
				deleteID     uint
				deleteTenant string
				wantErr      string
			}{
				{
					name:         "delete existing entry",
					createTenant: tenant1,
					create:       Entry{Description: "Salary", Amount: 1000, Date: date1, Type: ExpenseEntry},
					deleteTenant: tenant1,
				},
				{
					name:         "error when deleting non-existent entry",
					createTenant: tenant1,
					create:       Entry{Description: "Salary", Amount: 1000, Date: date1, Type: ExpenseEntry},
					deleteTenant: tenant1,
					deleteID:     9999,
					wantErr:      "entry not found",
				},
				{
					name:         "error when deleting entry  for other tenant",
					createTenant: tenant1,
					create:       Entry{Description: "Salary", Amount: 1000, Date: date1, Type: ExpenseEntry},
					deleteTenant: tenant2,
					wantErr:      "entry not found",
				},
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					dbCon := db.ConnDbName("bkmStoreDeleteEntry")
					store, err := New(dbCon)
					if err != nil {
						t.Fatal(err)
					}

					ctx := context.Background()
					id, err := store.CreateEntry(ctx, tc.create, tc.createTenant)
					if err != nil {
						t.Fatal(err)
					}

					if tc.deleteID == 0 {
						tc.deleteID = id
					}

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

						_, err := store.GetEntry(ctx, tc.deleteID, tc.deleteTenant)
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
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			tcs := []struct {
				name          string
				create        Entry
				createTenant  string
				updateID      uint
				updateTenant  string
				updatePayload EntryUpdatePayload
				want          Entry
				wantErr       string
			}{
				{
					name:          "update existing entry description",
					createTenant:  tenant1,
					create:        Entry{Description: "Salary", Amount: 1000, Date: date1, Type: ExpenseEntry},
					updateTenant:  tenant1,
					updatePayload: EntryUpdatePayload{Description: ptr("Updated Entry Description")},
					want:          Entry{Description: "Updated Entry Description", Amount: 1000, Date: date1, Type: ExpenseEntry},
				},
				{
					name:          "update entry amount",
					createTenant:  tenant1,
					create:        Entry{Description: "Salary", Amount: 1000, Date: date1, Type: ExpenseEntry},
					updateTenant:  tenant1,
					updatePayload: EntryUpdatePayload{Amount: ptr(float64(200))},
					want:          Entry{Description: "Salary", Amount: 200, Date: date1, Type: ExpenseEntry},
				},
				{
					name:          "update entry description and amount",
					createTenant:  tenant1,
					create:        Entry{Description: "Salary", Amount: 1000, Date: date1, Type: ExpenseEntry},
					updateTenant:  tenant1,
					updatePayload: EntryUpdatePayload{Description: ptr("Updated Entry Description"), Amount: ptr(float64(300))},
					want:          Entry{Description: "Updated Entry Description", Amount: 300, Date: date1, Type: ExpenseEntry},
				},
				{
					name:          "update entry date",
					createTenant:  tenant1,
					create:        Entry{Description: "Salary", Amount: 1000, Date: date1, Type: ExpenseEntry},
					updateTenant:  tenant1,
					updatePayload: EntryUpdatePayload{Date: &date2},
					want:          Entry{Description: "Salary", Amount: 1000, Date: date2, Type: ExpenseEntry},
				},
				{
					name:          "error when updating non-existent entry",
					createTenant:  tenant1,
					create:        Entry{Description: "Salary", Amount: 1000, Date: date1, Type: ExpenseEntry},
					updateTenant:  tenant1,
					updateID:      9999,
					updatePayload: EntryUpdatePayload{Description: ptr("Updated Entry Description")},
					wantErr:       "entry not found",
				},
				{
					name:          "error when updating anther's tenant entry",
					createTenant:  tenant1,
					create:        Entry{Description: "Salary", Amount: 1000, Date: date1, Type: ExpenseEntry},
					updateTenant:  tenant2,
					updatePayload: EntryUpdatePayload{Description: ptr("Updated Entry Description")},
					wantErr:       "entry not found",
				},
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					dbCon := db.ConnDbName("bkmStoreUpdateEntry")
					store, err := New(dbCon)
					if err != nil {
						t.Fatal(err)
					}

					ctx := context.Background()
					id, err := store.CreateEntry(ctx, tc.create, tc.createTenant)
					if err != nil {
						t.Fatal(err)
					}

					if tc.updateID == 0 {
						tc.updateID = id
					}

					err = store.UpdateEntry(tc.updatePayload, tc.updateID, tc.updateTenant)
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

var sampleEntries = []Entry{
	{Description: "e1", Amount: 1, Type: ExpenseEntry, Date: getTime("2025-01-01 00:00:00")}, // 0
	{Description: "e2", Amount: 2, Type: ExpenseEntry, Date: getTime("2025-01-02 00:00:00"),
		TargetAccountID: 1, TargetAccountName: "acc1"},
	{Description: "e3", Amount: 3, Type: ExpenseEntry, Date: getTime("2025-01-03 00:00:00"),
		TargetAccountID: 2, TargetAccountName: "acc2"},
	{Description: "e4", Amount: 4, Type: ExpenseEntry, Date: getTime("2025-01-04 00:00:00")}, // 3
	{Description: "e5", Amount: 5, Type: ExpenseEntry, Date: getTime("2025-01-05 00:00:00"),
		TargetAccountID: 2, TargetAccountName: "acc2"},
	{Description: "e6", Amount: 6, Type: ExpenseEntry, Date: getTime("2025-01-06 00:00:00"),
		TargetAccountID: 1, TargetAccountName: "acc1"},
	{Description: "e7", Amount: 7, Type: ExpenseEntry, Date: getTime("2025-01-07 00:00:00")}, // 6
	{Description: "e8", Amount: 8, Type: ExpenseEntry, Date: getTime("2025-01-08 00:00:00"),
		TargetAccountID: 2, TargetAccountName: "acc2"},
	{Description: "e9", Amount: 9, Type: ExpenseEntry, Date: getTime("2025-01-09 00:00:00")},
	{Description: "e10", Amount: 10, Type: TransferEntry, Date: getTime("2025-01-10 00:00:00"),
		TargetAccountID: 2, TargetAccountName: "acc2", OriginAccountID: 1, OriginAccountName: "acc1"},
	{Description: "e11", Amount: 10, Type: ExpenseEntry, Date: getTime("2025-01-11 00:00:00")}, // 10
	{Description: "e12", Amount: 10, Type: ExpenseEntry, Date: getTime("2025-01-12 00:00:00")},
	{Description: "e13", Amount: 10, Type: ExpenseEntry, Date: getTime("2025-01-13 00:00:00")},
	{Description: "e14", Amount: 10, Type: ExpenseEntry, Date: getTime("2025-01-14 00:00:00")},
	{Description: "e14", Amount: 10, Type: ExpenseEntry, Date: getTime("2025-01-15 00:00:00")},
	{Description: "e15", Amount: 10, Type: ExpenseEntry, Date: getTime("2025-01-16 00:00:00")},
}

var sampleAccounts = []Account{
	{ID: 1, Name: "acc1", Currency: currency.EUR, Type: 0, AccountProviderID: 1},
	{ID: 2, Name: "acc2", Currency: currency.EUR, Type: 0, AccountProviderID: 1},
	{ID: 3, Name: "acc3", Currency: currency.EUR, Type: 0, AccountProviderID: 1},
}

func TestSearchEntries(t *testing.T) {
	// Mock database setup - assuming a function NewStore that returns a *Store
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			tcs := []struct {
				name      string
				startDate time.Time
				endDate   time.Time
				accountID *uint
				limit     int
				page      int
				tenant    string
				wantErr   string
				want      []Entry
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
					accountID: ptr(uint(2)),
					tenant:    tenant1,
					want:      []Entry{sampleEntries[4], sampleEntries[2]},
				},
				{
					name:      "search with limit",
					startDate: getTime("2025-01-01 00:00:01"),
					endDate:   getTime("2025-01-09 00:00:00"),
					accountID: ptr(uint(2)),
					tenant:    tenant1,
					limit:     2,
					want:      []Entry{sampleEntries[7], sampleEntries[4]},
				},
				{
					name:      "search with limit and page",
					startDate: getTime("2025-01-01 00:00:01"),
					endDate:   getTime("2025-01-09 00:00:00"),
					accountID: ptr(uint(2)),
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
					want:      []Entry{sampleEntries[15], sampleEntries[14]},
				},
			}

			dbCon := db.ConnDbName("bkmStoreSearchEntries")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			for _, acc := range sampleAccounts {
				_, err = store.CreateAccount(context.Background(), acc, tenant1)
				if err != nil {
					t.Fatal(err)
				}
			}

			for _, entry := range sampleEntries {
				_, err = store.CreateEntry(context.Background(), entry, tenant1)
				if err != nil {
					t.Fatal(err)
				}
			}
			for _, entry := range sampleEntries[9:16] {
				_, err = store.CreateEntry(context.Background(), entry, tenant2)
				if err != nil {
					t.Fatal(err)
				}
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					// Call the ListEntries method
					got, err := store.ListEntries(
						context.Background(),
						tc.startDate,
						tc.endDate,
						tc.accountID,
						tc.limit,
						tc.page,
						tc.tenant,
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
