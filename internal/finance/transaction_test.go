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

func TestStore_UpdateIncome(t *testing.T) {
	tcs := []struct {
		name         string
		updateTenant string
		updateInput  IncomeUpdate
		txId         uint
		want         Transaction
		wantErr      string
	}{
		// ✅ Happy path cases
		{
			name:         "update description",
			updateTenant: tenant1,
			updateInput:  IncomeUpdate{Description: ptr("changed")},
			want: Income{
				Description: "changed",
				Amount:      10,
				AccountID:   1,
				Date:        getTime("2025-01-02"),
			},
		},
		{
			name:         "update date",
			updateTenant: tenant1,
			updateInput:  IncomeUpdate{Date: ptr(getTime("2025-01-03"))},
			want: Income{
				Description: "description",
				Amount:      10,
				AccountID:   1,
				Date:        getTime("2025-01-03"),
			},
		},
		{
			name:         "update amount",
			updateTenant: tenant1,
			updateInput:  IncomeUpdate{Amount: ptr(5.5)},
			want: Income{
				Description: "description",
				Amount:      5.5,
				AccountID:   1,
				Date:        getTime("2025-01-02"),
			},
		},
		{
			name:         "update account id",
			updateTenant: tenant1,
			updateInput:  IncomeUpdate{AccountID: ptr(uint(2))}, // valid Cash account
			want: Income{
				Description: "description",
				Amount:      10,
				AccountID:   2,
				Date:        getTime("2025-01-02"),
			},
		},

		// 🚨 Validation Errors
		{
			name:         "empty description error",
			updateTenant: tenant1,
			updateInput:  IncomeUpdate{Description: ptr("")},
			wantErr:      "description cannot be empty",
		},
		{
			name:         "zero date error",
			updateTenant: tenant1,
			updateInput:  IncomeUpdate{Date: ptr(time.Time{})},
			wantErr:      "date cannot be zero",
		},
		{
			name:         "zero amount error",
			updateTenant: tenant1,
			updateInput:  IncomeUpdate{Amount: ptr(float64(0))},
			wantErr:      "amount cannot be zero",
		},
		{
			name:         "zero account id error",
			updateTenant: tenant1,
			updateInput:  IncomeUpdate{AccountID: ptr(uint(0))},
			wantErr:      "account cannot be zero",
		},
		{
			name:         "non-cash account error",
			updateTenant: tenant1,
			updateInput:  IncomeUpdate{AccountID: ptr(uint(5))},
			wantErr:      "Incompatible account type for Income transaction",
		},

		// 🚨 No-op
		{
			name:         "no changes",
			updateTenant: tenant1,
			updateInput:  IncomeUpdate{},
			wantErr:      ErrNoChanges.Error(),
		},

		// 🚨 Not found / Wrong tenant
		{
			name:         "wrong tenant",
			updateTenant: tenant2,
			updateInput:  IncomeUpdate{Description: ptr("changed")},
			wantErr:      "error updating transaction: transaction not found",
		},
		{
			name:         "non-existing transaction",
			updateTenant: tenant1,
			updateInput:  IncomeUpdate{Description: ptr("changed")},
			txId:         9999,
			wantErr:      "error updating transaction: transaction not found",
		},
	}

	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {

			dbCon := db.ConnDbName("TestUpdateEntry")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			accountSampleData(t, store) // note: test operates on one set of data

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					in := Income{Description: "description", Amount: 10, AccountID: 1, Date: getTime("2025-01-02")}
					id, err := store.CreateTransaction(t.Context(), in, tenant1)
					if err != nil {
						t.Fatalf("failed to create transaction: %v", err)
					}

					if tc.txId != 0 { // only overwrite if the test case sets the value
						id = tc.txId
					}

					err = store.UpdateIncome(t.Context(), tc.updateInput, id, tc.updateTenant)
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

						got, err := store.GetTransaction(t.Context(), id, tc.updateTenant)
						if err != nil {
							t.Fatalf("expected entry to be found, but got error: %v", err)
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
func TestStore_UpdateExpense(t *testing.T) {
	tcs := []struct {
		name         string
		updateTenant string
		updateInput  ExpenseUpdate
		txId         uint
		want         Transaction
		wantErr      string
	}{
		// ✅ Happy path cases
		{
			name:         "update description",
			updateTenant: tenant1,
			updateInput:  ExpenseUpdate{Description: ptr("changed")},
			want: Expense{
				Description: "changed",
				Amount:      10,
				AccountID:   1,
				Date:        getTime("2025-01-02"),
			},
		},
		{
			name:         "update date",
			updateTenant: tenant1,
			updateInput:  ExpenseUpdate{Date: ptr(getTime("2025-01-03"))},
			want: Expense{
				Description: "description",
				Amount:      10,
				AccountID:   1,
				Date:        getTime("2025-01-03"),
			},
		},
		{
			name:         "update amount",
			updateTenant: tenant1,
			updateInput:  ExpenseUpdate{Amount: ptr(5.5)},
			want: Expense{
				Description: "description",
				Amount:      5.5,
				AccountID:   1,
				Date:        getTime("2025-01-02"),
			},
		},
		{
			name:         "update account id",
			updateTenant: tenant1,
			updateInput:  ExpenseUpdate{AccountID: ptr(uint(2))}, // valid Cash account
			want: Expense{
				Description: "description",
				Amount:      10,
				AccountID:   2,
				Date:        getTime("2025-01-02"),
			},
		},

		// 🚨 Validation Errors
		{
			name:         "empty description error",
			updateTenant: tenant1,
			updateInput:  ExpenseUpdate{Description: ptr("")},
			wantErr:      "description cannot be empty",
		},
		{
			name:         "zero date error",
			updateTenant: tenant1,
			updateInput:  ExpenseUpdate{Date: ptr(time.Time{})},
			wantErr:      "date cannot be zero",
		},
		{
			name:         "zero amount error",
			updateTenant: tenant1,
			updateInput:  ExpenseUpdate{Amount: ptr(float64(0))},
			wantErr:      "amount cannot be zero",
		},
		{
			name:         "zero account id error",
			updateTenant: tenant1,
			updateInput:  ExpenseUpdate{AccountID: ptr(uint(0))},
			wantErr:      "account cannot be zero",
		},
		{
			name:         "non-cash account error",
			updateTenant: tenant1,
			updateInput:  ExpenseUpdate{AccountID: ptr(uint(5))},
			wantErr:      "Incompatible account type for Expense transaction",
		},

		// 🚨 No-op
		{
			name:         "no changes",
			updateTenant: tenant1,
			updateInput:  ExpenseUpdate{},
			wantErr:      ErrNoChanges.Error(),
		},

		// 🚨 Not found / Wrong tenant
		{
			name:         "wrong tenant",
			updateTenant: tenant2,
			updateInput:  ExpenseUpdate{Description: ptr("changed")},
			wantErr:      "error updating transaction: transaction not found",
		},
		{
			name:         "non-existing transaction",
			updateTenant: tenant1,
			updateInput:  ExpenseUpdate{Description: ptr("changed")},
			txId:         9999,
			wantErr:      "error updating transaction: transaction not found",
		},
	}

	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {

			dbCon := db.ConnDbName("TestUpdateEntry")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			accountSampleData(t, store) // note: test operates on one set of data

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					in := Expense{Description: "description", Amount: 10, AccountID: 1, Date: getTime("2025-01-02")}
					id, err := store.CreateTransaction(t.Context(), in, tenant1)
					if err != nil {
						t.Fatalf("failed to create transaction: %v", err)
					}

					if tc.txId != 0 { // only overwrite if the test case sets the value
						id = tc.txId
					}

					err = store.UpdateExpense(t.Context(), tc.updateInput, id, tc.updateTenant)
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

						got, err := store.GetTransaction(t.Context(), id, tc.updateTenant)
						if err != nil {
							t.Fatalf("expected entry to be found, but got error: %v", err)
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

func TestStore_UpdateTransfer(t *testing.T) {
	tcs := []struct {
		name         string
		updateTenant string
		updateInput  TransferUpdate
		txId         uint
		want         Transfer
		wantErr      string
	}{
		{
			name:         "update description",
			updateTenant: tenant1,
			updateInput:  TransferUpdate{Description: ptr("changed")},
			want: Transfer{
				Description:     "changed",
				Date:            getTime("2025-01-02"),
				OriginAmount:    10,
				OriginAccountID: 1,
				TargetAmount:    11,
				TargetAccountID: 2,
			},
		},
		{
			name:         "update date",
			updateTenant: tenant1,
			updateInput:  TransferUpdate{Date: ptr(getTime("2025-01-03"))},
			want: Transfer{
				Description:     "desc",
				Date:            getTime("2025-01-03"),
				OriginAmount:    10,
				OriginAccountID: 1,
				TargetAmount:    11,
				TargetAccountID: 2,
			},
		},
		{
			name:         "update target amount",
			updateTenant: tenant1,
			updateInput:  TransferUpdate{TargetAmount: ptr(20.1)},
			want: Transfer{
				Description:     "desc",
				Date:            getTime("2025-01-02"),
				OriginAmount:    10,
				OriginAccountID: 1,
				TargetAmount:    20.1,
				TargetAccountID: 2,
			},
		},
		{
			name:         "update origin amount",
			updateTenant: tenant1,
			updateInput:  TransferUpdate{OriginAmount: ptr(30.1)},
			want: Transfer{
				Description:     "desc",
				Date:            getTime("2025-01-02"),
				OriginAmount:    30.1,
				OriginAccountID: 1,
				TargetAmount:    11,
				TargetAccountID: 2,
			},
		},
		{
			name:         "update target account id",
			updateTenant: tenant1,
			updateInput:  TransferUpdate{TargetAccountID: ptr(uint(3))}, // valid Cash account
			want: Transfer{
				Description:     "desc",
				Date:            getTime("2025-01-02"),
				OriginAmount:    10,
				OriginAccountID: 1,
				TargetAmount:    11,
				TargetAccountID: 3,
			},
		},
		{
			name:         "update origin account id",
			updateTenant: tenant1,
			updateInput:  TransferUpdate{OriginAccountID: ptr(uint(3))}, // valid Cash account
			want: Transfer{
				Description:     "desc",
				Date:            getTime("2025-01-02"),
				OriginAmount:    10,
				OriginAccountID: 3,
				TargetAmount:    11,
				TargetAccountID: 2,
			},
		},

		// 🚨 Validation Errors
		{
			name:         "empty description error",
			updateTenant: tenant1,
			updateInput:  TransferUpdate{Description: ptr("")},
			wantErr:      "description cannot be empty",
		},
		{
			name:         "zero date error",
			updateTenant: tenant1,
			updateInput:  TransferUpdate{Date: ptr(time.Time{})},
			wantErr:      "date cannot be zero",
		},
		{
			name:         "zero target amount error",
			updateTenant: tenant1,
			updateInput:  TransferUpdate{TargetAmount: ptr(float64(0))},
			wantErr:      "amount cannot be zero",
		},
		{
			name:         "zero origin amount error",
			updateTenant: tenant1,
			updateInput:  TransferUpdate{OriginAmount: ptr(float64(0))},
			wantErr:      "amount cannot be zero",
		},
		{
			name:         "zero target account id error",
			updateTenant: tenant1,
			updateInput:  TransferUpdate{TargetAccountID: ptr(uint(0))},
			wantErr:      "amount cannot be zero",
		},
		{
			name:         "zero origin account id error",
			updateTenant: tenant1,
			updateInput:  TransferUpdate{OriginAccountID: ptr(uint(0))},
			wantErr:      "amount cannot be zero",
		},
		{
			name:         "non-cash target account error",
			updateTenant: tenant1,
			updateInput:  TransferUpdate{TargetAccountID: ptr(uint(5))},
			wantErr:      "Incompatible account type for Income transaction",
		},
		{
			name:         "non-cash origin account error",
			updateTenant: tenant1,
			updateInput:  TransferUpdate{OriginAccountID: ptr(uint(5))},
			wantErr:      "Incompatible account type for Income transaction",
		},

		// 🚨 No-op
		{
			name:         "no changes",
			updateTenant: tenant1,
			updateInput:  TransferUpdate{},
			wantErr:      ErrNoChanges.Error(),
		},

		// 🚨 Not found / Wrong tenant
		{
			name:         "wrong tenant",
			updateTenant: tenant2,
			updateInput:  TransferUpdate{Description: ptr("changed")},
			wantErr:      "error updating transaction: transaction not found",
		},
		{
			name:         "non-existing transaction",
			updateTenant: tenant1,
			updateInput:  TransferUpdate{Description: ptr("changed")},
			txId:         9999,
			wantErr:      "error updating transaction: transaction not found",
		},
	}

	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			dbCon := db.ConnDbName("TestUpdateTransfer")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			accountSampleData(t, store)

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					// Arrange: create base transfer
					in := Transfer{
						Description:     "desc",
						OriginAmount:    10,
						OriginAccountID: 1,
						TargetAmount:    11,
						TargetAccountID: 2,
						Date:            getTime("2025-01-02"),
					}
					id, err := store.CreateTransaction(t.Context(), in, tenant1)
					if err != nil {
						t.Fatalf("failed to create transaction: %v", err)
					}

					if tc.txId != 0 { // only overwrite if the test case sets the value
						id = tc.txId
					}

					err = store.UpdateTransfer(t.Context(), tc.updateInput, id, tc.updateTenant)

					if tc.wantErr != "" {
						if err == nil {
							t.Fatalf("expected error %s but got none", tc.wantErr)
						}
						if err.Error() != tc.wantErr {
							t.Errorf("expected error %s but got %s", tc.wantErr, err.Error())
						}
						return
					} else {
						if err != nil {
							t.Fatalf("unexpected error: %v", err)
						}

						got, err := store.GetTransaction(t.Context(), id, tc.updateTenant)
						if err != nil {
							t.Fatalf("expected transaction but got error: %v", err)
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

var ignoreUnexportedTxFields = []cmp.Option{
	cmpopts.IgnoreUnexported(Income{}),
	cmpopts.IgnoreUnexported(Expense{}),
	cmpopts.IgnoreUnexported(Transfer{}),
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
