package finance

import (
	"github.com/go-bumbu/testdbs"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/text/currency"
	"testing"
	"time"
)

func TestStore_CreateTransaction(t *testing.T) {
	tcs := []struct {
		name    string
		input   Transaction
		tenant  string
		wantErr string
	}{
		{
			name:   "create valid income",
			tenant: tenant1,
			input: Income{
				Description: "income 1",
				Amount:      2.5,
				AccountID:   1,
				Date:        time.Now(),
			},
		},
		{
			name:   "create valid expense",
			tenant: tenant1,
			input: Expense{
				Description: "expense 1",
				Amount:      2.5,
				AccountID:   1,
				Date:        time.Now(),
			},
		},
		{
			name:   "create valid Transfer",
			tenant: tenant1,
			input: Transfer{
				Description:     "transfer 1",
				OriginAmount:    1.1,
				OriginAccountID: 1,
				TargetAmount:    2.2,
				TargetAccountID: 2,
				Date:            time.Now(),
			},
		},
		{
			name:   "want error on empty description",
			tenant: tenant1,
			input: Expense{
				Amount:    2.5,
				AccountID: 1,
				Date:      time.Now(),
			},
			wantErr: "description cannot be empty",
		},
		{
			name:   "want error on zero amount",
			tenant: tenant1,
			input: Expense{
				Description: "income 1",
				AccountID:   1,
				Date:        time.Now(),
			},
			wantErr: "amount cannot be zero",
		},
		{
			name:   "want error on zero date",
			tenant: tenant1,
			input: Expense{
				Description: "income 1",
				Amount:      2.5,
				AccountID:   1,
			}, wantErr: "date cannot be zero",
		},
	}

	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {

			dbCon := db.ConnDbName("storeCreateEntry")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			transactionSampleData(t, store)

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					id, err := store.CreateTransaction(t.Context(), tc.input, tc.tenant)
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

						got, err := store.GetTransaction(t.Context(), id, tc.tenant)
						if err != nil {
							t.Fatalf("expected entry to be found, but got error: %v", err)
						}

						if diff := cmp.Diff(got, tc.input, ignoreUnexportedTxFields...); diff != "" {
							t.Errorf("unexpected result (-want +got):\n%s", diff)
						}
					}
				})
			}
		})
	}
}

func TestStore_GetTransaction(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			tcs := []struct {
				name        string
				checkTenant string
				checkId     uint
				want        Transaction
				wantErr     string
			}{
				{
					name:        "get existing transaction",
					checkTenant: tenant1,
					checkId:     2,
					want:        sampleTransactions[2],
				},
				{
					name:        "want error when reading from different tenant",
					checkTenant: tenant2,
					wantErr:     ErrTransactionNotFound.Error(),
				},
			}

			dbCon := db.ConnDbName("storeGetEntry")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			transactionSampleData(t, store)

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					got, err := store.GetTransaction(t.Context(), tc.checkId, tc.checkTenant)
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

						if diff := cmp.Diff(got, tc.want, ignoreUnexportedTxFields...); diff != "" {
							t.Errorf("unexpected result (-want +got):\n%s", diff)
						}
					}
				})
			}
		})
	}
}

func TestStore_DeleteTransaction(t *testing.T) {
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
					wantErr:      "transaction not found",
				},
				{
					name:         "error when deleting entry  for other tenant",
					deleteTenant: tenant2,
					wantErr:      "transaction not found",
				},
			}

			dbCon := db.ConnDbName("storeGetEntry")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			transactionSampleData(t, store)

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					err = store.DeleteTransaction(t.Context(), tc.deleteID, tc.deleteTenant)
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

						_, err := store.GetTransaction(t.Context(), tc.deleteID, tc.deleteTenant)
						if err == nil {
							t.Fatalf("expected item to not exist, but got we got a transaction")
						}

						var entries []dbEntry
						if err := store.db.Where("transaction_id = ?", tc.deleteID).Find(&entries).Error; err != nil {
							t.Fatalf("unexpected err, failed to retrieve entries: %v", err)
						}
						if len(entries) > 0 {
							t.Errorf("expected no entries, but got %d entries", len(entries))
						}

					}
				})
			}
		})
	}
}

func TestStore_UpdateTransaction(t *testing.T) {
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
		//{
		//	name:          "update entry target amount",
		//	updateID:      2,
		//	updateTenant:  tenant1,
		//	updatePayload: EntryUpdatePayload{TargetAmount: ptr(float64(200))},
		//	want: Entry{Description: "e1", TargetAmount: 200, Type: ExpenseEntry, Date: getTime("2025-01-02 00:00:00"),
		//		TargetAccountID: 1},
		//},
		//{
		//	name:          "update entry description and target amount",
		//	updateID:      3,
		//	updateTenant:  tenant1,
		//	updatePayload: EntryUpdatePayload{Description: ptr("Updated Entry Description"), TargetAmount: ptr(float64(300))},
		//	want: Entry{Description: "Updated Entry Description", TargetAmount: 300, Type: IncomeEntry, Date: getTime("2025-01-03 00:00:00"),
		//		TargetAccountID: 2, CategoryId: 3},
		//},
		//{
		//	name:          "update entry date",
		//	updateID:      4,
		//	updateTenant:  tenant1,
		//	updatePayload: EntryUpdatePayload{Date: &date2},
		//	want:          Entry{Description: "e3", TargetAmount: -4.1, TargetAccountID: 1, Type: ExpenseEntry, Date: date2},
		//},
		//{
		//	name:          "error when updating non-existent entry",
		//	updateTenant:  tenant1,
		//	updateID:      9999,
		//	updatePayload: EntryUpdatePayload{Description: ptr("Updated Entry Description")},
		//	wantErr:       "entry not found",
		//},
		//{
		//	name:          "error when updating another tenant's entry",
		//	updateTenant:  tenant2,
		//	updateID:      1,
		//	updatePayload: EntryUpdatePayload{Description: ptr("Updated Entry Description")},
		//	wantErr:       "entry not found",
		//},
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

var ignoreUnexportedTxFields = []cmp.Option{
	cmpopts.IgnoreUnexported(Income{}),
	cmpopts.IgnoreUnexported(Expense{}),
	cmpopts.IgnoreUnexported(Transfer{}),
	cmpopts.IgnoreFields(Income{}, "Date"),
	cmpopts.IgnoreFields(Expense{}, "Date"),
	cmpopts.IgnoreFields(Transfer{}, "Date"),
}

var sampleTransactions = map[int]Transaction{
	1: Income{Description: "First Income", Amount: 1.1, AccountID: 1, Date: time.Now()},
	2: Expense{Description: "First expense", Amount: 2.2, AccountID: 1, Date: time.Now()},
	3: Transfer{
		Description:     "First transfer",
		OriginAmount:    3.3,
		OriginAccountID: 1,
		TargetAmount:    4.4,
		TargetAccountID: 2,
		Date:            time.Now(),
		transaction:     transaction{},
	},
}

func transactionSampleData(t *testing.T, store *Store) {

	// =========================================
	// create accounts providers
	// =========================================

	accProviderId, err := store.CreateAccountProvider(t.Context(), AccountProvider{Name: "p1"}, tenant1)
	if err != nil {
		t.Fatalf("error creating provider 1: %v", err)
	}
	// =========================================
	// create accounts
	// =========================================
	Accs := []Account{
		{AccountProviderID: accProviderId, Name: "acc1", Currency: currency.EUR, Type: Cash},
		{AccountProviderID: accProviderId, Name: "acc2", Currency: currency.USD, Type: Cash},
	}
	for _, acc := range Accs {
		_, err = store.CreateAccount(t.Context(), acc, tenant1)
		if err != nil {
			t.Fatalf("error creating account 1: %v", err)
		}
	}
	// =========================================
	// Create Transactions
	// =========================================
	for _, tx := range sampleTransactions {
		_, err = store.CreateTransaction(t.Context(), tx, tenant1)
		if err != nil {
			t.Fatalf("error creating account 1: %v", err)
		}
	}
}
