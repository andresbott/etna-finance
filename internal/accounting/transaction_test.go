package accounting

import (
	"sort"
	"testing"
	"time"

	"github.com/go-bumbu/testdbs"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/text/currency"
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
				CategoryID:  1,
				Date:        time.Now(),
			},
		},
		{
			name:   "create valid income without a category",
			tenant: tenant1,
			input: Income{
				Description: "income 1",
				Amount:      2.5,
				AccountID:   1,
				CategoryID:  0,
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
				CategoryID:  2,
				Date:        time.Now(),
			},
		},
		{
			name:   "create valid expense without a category",
			tenant: tenant1,
			input: Expense{
				Description: "income 1",
				Amount:      2.5,
				AccountID:   1,
				CategoryID:  0,
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
				Amount:     2.5,
				AccountID:  1,
				CategoryID: 2,
				Date:       time.Now(),
			},
			wantErr: "description cannot be empty",
		},
		{
			name:   "want error on wrong category type",
			tenant: tenant1,
			input: Income{
				Description: "income 1",
				Amount:      2.5,
				AccountID:   1,
				CategoryID:  2,
				Date:        time.Now(),
			},
			wantErr: "incompatible category type for Income transaction",
		},
		{
			name:   "want error on zero amount",
			tenant: tenant1,
			input: Expense{
				Description: "income 1",
				AccountID:   1,
				CategoryID:  2,
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
				CategoryID:  2,
			}, wantErr: "date cannot be zero",
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
			transactionSampleData(t, store, sampleTransactions)

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
							t.Errorf("unexpected result (+want -got):\n%s", diff)
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
					name:        "get existing baseTx",
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

			dbCon := db.ConnDbName("storeGetTransaction")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			categorySampleData(t, store, sampleCategories)
			transactionSampleData(t, store, sampleTransactions)

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
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			categorySampleData(t, store, sampleCategories)
			transactionSampleData(t, store, sampleTransactions)

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
							t.Fatalf("expected item to not exist, but got we got a baseTx")
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
				CategoryID:  1,
				Date:        getDate("2025-01-02"),
			},
		},
		{
			name:         "update date",
			updateTenant: tenant1,
			updateInput:  IncomeUpdate{Date: ptr(getDate("2025-01-03"))},
			want: Income{
				Description: "description",
				Amount:      10,
				AccountID:   1,
				CategoryID:  1,
				Date:        getDate("2025-01-03"),
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
				CategoryID:  1,
				Date:        getDate("2025-01-02"),
			},
		},
		{
			name:         "update account id",
			updateTenant: tenant1,
			updateInput:  IncomeUpdate{AccountID: ptr(uint(2))}, // valid CashAccountType account
			want: Income{
				Description: "description",
				Amount:      10,
				AccountID:   2,
				CategoryID:  1,
				Date:        getDate("2025-01-02"),
			},
		},
		{
			name:         "update category id",
			updateTenant: tenant1,
			updateInput:  IncomeUpdate{CategoryID: ptr(uint(3))}, // valid CashAccountType account
			want: Income{
				Description: "description",
				Amount:      10,
				AccountID:   1,
				CategoryID:  3,
				Date:        getDate("2025-01-02"),
			},
		},
		{
			name:         "unset category id",
			updateTenant: tenant1,
			updateInput:  IncomeUpdate{CategoryID: ptr(uint(0))}, // valid CashAccountType account
			want: Income{
				Description: "description",
				Amount:      10,
				AccountID:   1,
				CategoryID:  0,
				Date:        getDate("2025-01-02"),
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
			wantErr:      "incompatible account type 'Investment' for transaction",
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
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			categorySampleData(t, store, sampleCategories)
			accountSampleData(t, store) // note: test operates on one set of data

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					in := Income{Description: "description", Amount: 10, AccountID: 1, CategoryID: 1, Date: getDate("2025-01-02")}
					id, err := store.CreateTransaction(t.Context(), in, tenant1)
					if err != nil {
						t.Fatalf("failed to create baseTx: %v", err)
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
				CategoryID:  2,
				Date:        getDate("2025-01-02"),
			},
		},
		{
			name:         "update date",
			updateTenant: tenant1,
			updateInput:  ExpenseUpdate{Date: ptr(getDate("2025-01-03"))},
			want: Expense{
				Description: "description",
				Amount:      10,
				AccountID:   1,
				CategoryID:  2,
				Date:        getDate("2025-01-03"),
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
				CategoryID:  2,
				Date:        getDate("2025-01-02"),
			},
		},
		{
			name:         "update account id",
			updateTenant: tenant1,
			updateInput:  ExpenseUpdate{AccountID: ptr(uint(2))}, // valid CashAccountType account
			want: Expense{
				Description: "description",
				Amount:      10,
				AccountID:   2,
				CategoryID:  2,
				Date:        getDate("2025-01-02"),
			},
		},
		{
			name:         "update category id",
			updateTenant: tenant1,
			updateInput:  ExpenseUpdate{CategoryID: ptr(uint(6))}, // valid CashAccountType account
			want: Expense{
				Description: "description",
				Amount:      10,
				AccountID:   1,
				CategoryID:  6,
				Date:        getDate("2025-01-02"),
			},
		},
		{
			name:         "unset category id",
			updateTenant: tenant1,
			updateInput:  ExpenseUpdate{CategoryID: ptr(uint(0))}, // valid CashAccountType account
			want: Expense{
				Description: "description",
				Amount:      10,
				AccountID:   1,
				CategoryID:  0,
				Date:        getDate("2025-01-02"),
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
			wantErr:      "incompatible account type 'Investment' for transaction",
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
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			categorySampleData(t, store, sampleCategories)
			accountSampleData(t, store) // note: test operates on one set of data

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					in := Expense{Description: "description", Amount: 10, AccountID: 1, CategoryID: 2,
						Date: getDate("2025-01-02")}
					id, err := store.CreateTransaction(t.Context(), in, tenant1)
					if err != nil {
						t.Fatalf("failed to create baseTx: %v", err)
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
				Date:            getDate("2025-01-02"),
				OriginAmount:    10,
				OriginAccountID: 1,
				TargetAmount:    11,
				TargetAccountID: 2,
			},
		},
		{
			name:         "update date",
			updateTenant: tenant1,
			updateInput:  TransferUpdate{Date: ptr(getDate("2025-01-03"))},
			want: Transfer{
				Description:     "desc",
				Date:            getDate("2025-01-03"),
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
				Date:            getDate("2025-01-02"),
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
				Date:            getDate("2025-01-02"),
				OriginAmount:    30.1,
				OriginAccountID: 1,
				TargetAmount:    11,
				TargetAccountID: 2,
			},
		},
		{
			name:         "update target account id",
			updateTenant: tenant1,
			updateInput:  TransferUpdate{TargetAccountID: ptr(uint(3))}, // valid CashAccountType account
			want: Transfer{
				Description:     "desc",
				Date:            getDate("2025-01-02"),
				OriginAmount:    10,
				OriginAccountID: 1,
				TargetAmount:    11,
				TargetAccountID: 3,
			},
		},
		{
			name:         "update origin account id",
			updateTenant: tenant1,
			updateInput:  TransferUpdate{OriginAccountID: ptr(uint(3))}, // valid CashAccountType account
			want: Transfer{
				Description:     "desc",
				Date:            getDate("2025-01-02"),
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
			wantErr:      "incompatible account type 'Investment' for transaction",
		},
		{
			name:         "non-cash origin account error",
			updateTenant: tenant1,
			updateInput:  TransferUpdate{OriginAccountID: ptr(uint(5))},
			wantErr:      "incompatible account type 'Investment' for transaction",
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
			store, err := NewStore(dbCon)
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
						Date:            getDate("2025-01-02"),
					}
					id, err := store.CreateTransaction(t.Context(), in, tenant1)
					if err != nil {
						t.Fatalf("failed to create baseTx: %v", err)
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
							t.Fatalf("expected baseTx but got error: %v", err)
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

var listTransactionsSampleData = map[int]Transaction{
	// used in test 1
	1: Income{Description: "First Income", Amount: 1.1, AccountID: 1, Date: getDate("2025-01-01")},
	2: Expense{Description: "First expense", Amount: 2.2, AccountID: 1, Date: getDate("2025-01-02")},
	3: Transfer{Description: "First transfer", OriginAmount: 3.3, OriginAccountID: 1, TargetAmount: 4.4, TargetAccountID: 2, Date: getDate("2025-01-03")},
	// used in date based filter test
	4: Income{Description: "1", Date: getDateTime("2023-12-31 23:58:00"), Amount: 1, AccountID: 1},
	5: Income{Description: "2", Date: getDateTime("2024-01-01 00:00:00"), Amount: 1, AccountID: 1},
	6: Income{Description: "3", Date: getDateTime("2024-01-02 00:00:00"), Amount: 1, AccountID: 2},
	7: Income{Description: "4", Date: getDateTime("2024-01-03 00:00:00"), Amount: 1, AccountID: 1},
	8: Income{Description: "5", Date: getDateTime("2024-01-04 00:00:00"), Amount: 1, AccountID: 1},

	// used in filter by type
	9:  Income{Description: "i1", Date: getDate("2023-01-01"), Amount: 1.1, AccountID: 1},
	10: Expense{Description: "e1", Date: getDate("2023-01-02"), Amount: 2.2, AccountID: 1},
	11: Transfer{Description: "t1", Date: getDate("2023-01-03"), OriginAmount: 3.3, OriginAccountID: 1, TargetAmount: 4.4, TargetAccountID: 2},
	12: Income{Description: "i2", Date: getDate("2023-01-04"), Amount: 1.1, AccountID: 1},
	13: Expense{Description: "e2", Date: getDate("2023-01-05"), Amount: 2.2, AccountID: 1},
	14: Transfer{Description: "t2", Date: getDate("2023-01-06"), OriginAmount: 3.3, OriginAccountID: 2, TargetAmount: 4.4, TargetAccountID: 1},

	// used in pagination tests
	20: Income{Description: "p1", Date: getDateTime("2022-01-01 00:00:00"), Amount: 1, AccountID: 1},
	21: Income{Description: "p2", Date: getDateTime("2022-01-02 00:00:00"), Amount: 1, AccountID: 1},
	22: Income{Description: "p3", Date: getDateTime("2022-01-03 00:00:00"), Amount: 1, AccountID: 1},
	23: Income{Description: "p4", Date: getDateTime("2022-01-04 00:00:00"), Amount: 1, AccountID: 1},
	24: Income{Description: "p5", Date: getDateTime("2022-01-05 00:00:00"), Amount: 1, AccountID: 2},
	25: Income{Description: "p6", Date: getDateTime("2022-01-06 00:00:00"), Amount: 1, AccountID: 1},
	26: Income{Description: "p7", Date: getDateTime("2022-01-07 00:00:00"), Amount: 1, AccountID: 1},
	27: Income{Description: "p8", Date: getDateTime("2022-01-08 00:00:00"), Amount: 1, AccountID: 1},
	28: Income{Description: "p9", Date: getDateTime("2022-01-09 00:00:00"), Amount: 1, AccountID: 3},
}

func TestStore_ListTransactions(t *testing.T) {
	tcs := []struct {
		name   string
		tenant string

		opts    ListOpts
		want    []Transaction
		wantErr string
	}{
		{
			name:   "list basic transactions",
			tenant: tenant1,
			opts: ListOpts{
				StartDate: getDate("2025-01-01"),
				EndDate:   getDate("2025-01-04"),
			},
			want: []Transaction{
				listTransactionsSampleData[3], listTransactionsSampleData[2], listTransactionsSampleData[1],
			},
		},
		{
			name:   "filter by date",
			tenant: tenant1,
			opts: ListOpts{
				StartDate: getDate("2024-01-01"),
				EndDate:   getDate("2024-01-03"),
			},
			want: []Transaction{
				listTransactionsSampleData[7], listTransactionsSampleData[6], listTransactionsSampleData[5],
			},
		},
		{
			name:   "filter by type income",
			tenant: tenant1,
			opts: ListOpts{
				StartDate: getDate("2023-01-01"),
				EndDate:   getDate("2023-12-31"),
				Types:     []TxType{IncomeTransaction},
			},
			want: []Transaction{
				listTransactionsSampleData[4], listTransactionsSampleData[12], listTransactionsSampleData[9],
			},
		},
		{
			name:   "filter by type income and expense",
			tenant: tenant1,
			opts: ListOpts{
				StartDate: getDate("2023-01-01"),
				EndDate:   getDate("2023-12-31"),
				Types:     []TxType{IncomeTransaction, ExpenseTransaction},
			},
			want: []Transaction{
				listTransactionsSampleData[4], listTransactionsSampleData[13], listTransactionsSampleData[12],
				listTransactionsSampleData[10], listTransactionsSampleData[9],
			},
		},
		{
			name:   "limit the responses",
			tenant: tenant1,
			opts: ListOpts{
				StartDate: getDate("2022-01-01"),
				EndDate:   getDate("2022-12-31"),
				Limit:     2, Page: 1,
			},
			want: []Transaction{
				listTransactionsSampleData[28], listTransactionsSampleData[27],
			},
		},
		{
			name:   "limit the responses page 2",
			tenant: tenant1,
			opts: ListOpts{
				StartDate: getDate("2022-01-01"),
				EndDate:   getDate("2022-12-31"),
				Limit:     2, Page: 2,
			},
			want: []Transaction{
				listTransactionsSampleData[26], listTransactionsSampleData[25],
			},
		},
		{
			name:   "limit the responses last page",
			tenant: tenant1,
			opts: ListOpts{
				StartDate: getDate("2022-01-01"),
				EndDate:   getDate("2022-12-31"),
				Limit:     7, Page: 2,
			},
			want: []Transaction{
				listTransactionsSampleData[21], listTransactionsSampleData[20],
			},
		},
		{
			name:   "limit the responses to several accounts",
			tenant: tenant1,
			opts: ListOpts{
				StartDate: getDate("2022-01-01"),
				EndDate:   getDate("2025-01-04"),
				AccountId: []int{2, 3},
				Types:     []TxType{IncomeTransaction, ExpenseTransaction},
			},
			want: []Transaction{
				listTransactionsSampleData[6],
				listTransactionsSampleData[28],
				listTransactionsSampleData[24],
			},
		},
		{
			name:   "limit the responses to single account id and type transfer",
			tenant: tenant1,
			opts: ListOpts{
				StartDate: getDate("2022-01-01"),
				EndDate:   getDate("2025-01-04"),
				AccountId: []int{2},
				Types:     []TxType{TransferTransaction},
			},
			want: []Transaction{
				listTransactionsSampleData[3],
				listTransactionsSampleData[14],
				listTransactionsSampleData[11],
			},
		},
		{
			name:   "limit the responses to single account id",
			tenant: tenant1,
			opts: ListOpts{
				StartDate: getDate("2022-01-01"),
				EndDate:   getDate("2025-01-04"),
				AccountId: []int{2},
			},
			want: []Transaction{
				listTransactionsSampleData[3],
				listTransactionsSampleData[6],
				listTransactionsSampleData[14],
				listTransactionsSampleData[11],
				listTransactionsSampleData[24],
			},
		},
	}

	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			dbCon := db.ConnDbName("TestListTransactions")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			accountSampleData(t, store)
			// note, to optimize the test, we execute all tests on a common set of pre created transactions
			transactionSampleData(t, store, listTransactionsSampleData)

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					got, err := store.ListTransactions(t.Context(), tc.opts, tenant1)

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
						if diff := cmp.Diff(got, tc.want, ignoreUnexportedAndIds...); diff != "" {
							t.Errorf("unexpected result (-got +want):\n%s", diff)
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
var ignoreUnexportedAndIds = []cmp.Option{
	cmpopts.IgnoreUnexported(Income{}),
	cmpopts.IgnoreUnexported(Expense{}),
	cmpopts.IgnoreUnexported(Transfer{}),
	cmpopts.IgnoreFields(Income{}, "Id"),
	cmpopts.IgnoreFields(Expense{}, "Id"),
	cmpopts.IgnoreFields(Transfer{}, "Id"),
}

var sampleTransactions = map[int]Transaction{
	1: Income{Description: "First Income", Amount: 1.1, AccountID: 1, CategoryID: 1,
		Date: getDate("2025-01-01")},
	2: Expense{Description: "First expense", Amount: 2.2, AccountID: 1, CategoryID: 2,
		Date: getDate("2025-01-02")},
	3: Transfer{Description: "First transfer", OriginAmount: 3.3, OriginAccountID: 1,
		TargetAmount: 4.4, TargetAccountID: 2, Date: getDate("2025-01-03")},
	4: Income{Description: "income without category", Amount: 1.1, AccountID: 1, CategoryID: 0,
		Date: getDate("2025-01-04")},
	5: Expense{Description: "expense without category", Amount: 1.1, AccountID: 1, CategoryID: 0,
		Date: getDate("2025-01-05")},
}

func transactionSampleData(t *testing.T, store *Store, data map[int]Transaction) {

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
		{AccountProviderID: accProviderId, Name: "acc1", Currency: currency.EUR, Type: CashAccountType},
		{AccountProviderID: accProviderId, Name: "acc2", Currency: currency.USD, Type: CashAccountType},
		{AccountProviderID: accProviderId, Name: "acc3", Currency: currency.CHF, Type: CashAccountType},
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

	// transform the map into a sorted array to have predictable test results
	var dataKeys []int
	for k := range data {
		dataKeys = append(dataKeys, k)
	}
	sort.Ints(dataKeys)

	var dataAr = make([]Transaction, len(data))
	for i, k := range dataKeys {
		dataAr[i] = data[k]
	}

	for _, tx := range dataAr {
		_, err = store.CreateTransaction(t.Context(), tx, tenant1)
		if err != nil {
			t.Fatalf("error creating account: %v", err)
		}
	}
}
