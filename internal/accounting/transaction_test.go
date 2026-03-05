package accounting

import (
	"context"
	"math"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/andresbott/etna/internal/marketdata"
	"github.com/go-bumbu/testdbs"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/text/currency"
	"gorm.io/gorm"
)

// newAccountingStoreWithMarketData creates an accounting store with a marketdata store for tests that need instruments.
func newAccountingStoreWithMarketData(t *testing.T, db *gorm.DB) (*Store, *marketdata.Store) {
	t.Helper()
	mktStore, err := marketdata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(db, mktStore)
	if err != nil {
		t.Fatal(err)
	}
	return store, mktStore
}

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
			store, _ := newAccountingStoreWithMarketData(t, dbCon)

			categorySampleData(t, store, sampleCategories)
			transactionSampleData(t, store, sampleTransactions)

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					id, err := store.CreateTransaction(t.Context(), tc.input)
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
							t.Errorf("expected valid entry id, but got 0")
						}

						got, err := store.GetTransaction(t.Context(), id)
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
			store, _ := newAccountingStoreWithMarketData(t, dbCon)

			categorySampleData(t, store, sampleCategories)
			transactionSampleData(t, store, sampleTransactions)

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					got, err := store.GetTransaction(t.Context(), tc.checkId)
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
			store, _ := newAccountingStoreWithMarketData(t, dbCon)

			categorySampleData(t, store, sampleCategories)
			transactionSampleData(t, store, sampleTransactions)

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					err := store.DeleteTransaction(t.Context(), tc.deleteID)
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

						_, err := store.GetTransaction(t.Context(), tc.deleteID)
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
			store, _ := newAccountingStoreWithMarketData(t, dbCon)

			categorySampleData(t, store, sampleCategories)
			accountSampleData(t, store) // note: test operates on one set of data

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					in := Income{Description: "description", Amount: 10, AccountID: 1, CategoryID: 1, Date: getDate("2025-01-02")}
					id, err := store.CreateTransaction(t.Context(), in)
					if err != nil {
						t.Fatalf("failed to create baseTx: %v", err)
					}

					if tc.txId != 0 { // only overwrite if the test case sets the value
						id = tc.txId
					}

					err = store.UpdateIncome(t.Context(), tc.updateInput, id)
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

						got, err := store.GetTransaction(t.Context(), id)
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
			store, _ := newAccountingStoreWithMarketData(t, dbCon)

			categorySampleData(t, store, sampleCategories)
			accountSampleData(t, store) // note: test operates on one set of data

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					in := Expense{Description: "description", Amount: 10, AccountID: 1, CategoryID: 2,
						Date: getDate("2025-01-02")}
					id, err := store.CreateTransaction(t.Context(), in)
					if err != nil {
						t.Fatalf("failed to create baseTx: %v", err)
					}

					if tc.txId != 0 { // only overwrite if the test case sets the value
						id = tc.txId
					}

					err = store.UpdateExpense(t.Context(), tc.updateInput, id)
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

						got, err := store.GetTransaction(t.Context(), id)
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
			store, _ := newAccountingStoreWithMarketData(t, dbCon)

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
					id, err := store.CreateTransaction(t.Context(), in)
					if err != nil {
						t.Fatalf("failed to create baseTx: %v", err)
					}

					if tc.txId != 0 { // only overwrite if the test case sets the value
						id = tc.txId
					}

					err = store.UpdateTransfer(t.Context(), tc.updateInput, id)

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

						got, err := store.GetTransaction(t.Context(), id)
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
			store, _ := newAccountingStoreWithMarketData(t, dbCon)

			accountSampleData(t, store)
			// note, to optimize the test, we execute all tests on a common set of pre created transactions
			transactionSampleData(t, store, listTransactionsSampleData)

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					got, err := store.ListTransactions(t.Context(), tc.opts)

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
	cmpopts.IgnoreUnexported(StockBuy{}),
	cmpopts.IgnoreUnexported(StockSell{}),
	cmpopts.IgnoreUnexported(StockGrant{}),
	cmpopts.IgnoreUnexported(StockTransfer{}),
}
var ignoreUnexportedAndIds = []cmp.Option{
	cmpopts.IgnoreUnexported(Income{}),
	cmpopts.IgnoreUnexported(Expense{}),
	cmpopts.IgnoreUnexported(Transfer{}),
	cmpopts.IgnoreFields(Income{}, "Id"),
	cmpopts.IgnoreFields(Expense{}, "Id"),
	cmpopts.IgnoreFields(Transfer{}, "Id"),
}

// setupStockBuySellTest creates provider, investment account, cash account and instrument for stock buy/sell tests.
func setupStockBuySellTest(t *testing.T, ctx context.Context, store *Store, mktStore *marketdata.Store) (investmentAccountID, cashAccountID, instrumentID uint) {
	t.Helper()
	providerID, err := store.CreateAccountProvider(ctx, AccountProvider{Name: "Broker"})
	if err != nil {
		t.Fatal(err)
	}
	investmentAccountID, err = store.CreateAccount(ctx, Account{
		AccountProviderID: providerID,
		Name:              "Broker account",
		Currency:          currency.USD,
		Type:              InvestmentAccountType,
	})
	if err != nil {
		t.Fatal(err)
	}
	cashAccountID, err = store.CreateAccount(ctx, Account{
		AccountProviderID: providerID,
		Name:              "Checking",
		Currency:          currency.USD,
		Type:              CheckinAccountType,
	})
	if err != nil {
		t.Fatal(err)
	}
	instrumentID, err = mktStore.CreateInstrument(ctx, marketdata.Instrument{
		Symbol:   "AAPL",
		Name:     "Apple Inc.",
		Currency: currency.USD,
	})
	if err != nil {
		t.Fatal(err)
	}
	return investmentAccountID, cashAccountID, instrumentID
}

func verifyStockBuyResult(t *testing.T, got Transaction, want StockBuy, investmentAccountID, cashAccountID uint) {
	t.Helper()
	gotStockBuy, ok := got.(StockBuy)
	if !ok {
		t.Fatalf("expected StockBuy, got %T", got)
	}
	if gotStockBuy.Quantity != want.Quantity || gotStockBuy.TotalAmount != want.TotalAmount {
		t.Errorf("got StockBuy Quantity=%v TotalAmount=%v, want Quantity=%v TotalAmount=%v",
			gotStockBuy.Quantity, gotStockBuy.TotalAmount, want.Quantity, want.TotalAmount)
	}
	if gotStockBuy.InvestmentAccountID != investmentAccountID || gotStockBuy.CashAccountID != cashAccountID {
		t.Errorf("got StockBuy InvestmentAccountID=%v CashAccountID=%v, want %v %v",
			gotStockBuy.InvestmentAccountID, gotStockBuy.CashAccountID, investmentAccountID, cashAccountID)
	}
}

func verifyStockSellResult(t *testing.T, got Transaction, want StockSell, investmentAccountID, cashAccountID uint) {
	t.Helper()
	gotStockSell, ok := got.(StockSell)
	if !ok {
		t.Fatalf("expected StockSell, got %T", got)
	}
	if gotStockSell.Quantity != want.Quantity || gotStockSell.TotalAmount != want.TotalAmount {
		t.Errorf("got StockSell Quantity=%v TotalAmount=%v, want Quantity=%v TotalAmount=%v",
			gotStockSell.Quantity, gotStockSell.TotalAmount, want.Quantity, want.TotalAmount)
	}
	if gotStockSell.InvestmentAccountID != investmentAccountID || gotStockSell.CashAccountID != cashAccountID {
		t.Errorf("got StockSell InvestmentAccountID=%v CashAccountID=%v, want %v %v",
			gotStockSell.InvestmentAccountID, gotStockSell.CashAccountID, investmentAccountID, cashAccountID)
	}
	if gotStockSell.CostBasis <= 0 {
		t.Errorf("got StockSell CostBasis=%v, want positive", gotStockSell.CostBasis)
	}
	if math.Abs(gotStockSell.CostBasis+gotStockSell.RealizedGainLoss+gotStockSell.Fees-gotStockSell.TotalAmount) > 0.01 {
		t.Errorf("invariant violated: costBasis+realizedGainLoss+fees=%v != totalAmount=%v",
			gotStockSell.CostBasis+gotStockSell.RealizedGainLoss+gotStockSell.Fees, gotStockSell.TotalAmount)
	}
}

func TestStore_CreateStockBuy_CreateStockSell(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("storeCreateStock"))
			investmentAccountID, cashAccountID, instrumentID := setupStockBuySellTest(t, ctx, store, mktStore)

			buy := StockBuy{
				Description:         "Buy AAPL",
				Date:                getDate("2025-02-01"),
				InvestmentAccountID: investmentAccountID,
				CashAccountID:       cashAccountID,
				InstrumentID:        instrumentID,
				Quantity:            10,
				TotalAmount:         1500.0,
				StockAmount:         1850.0,
			}
			buyID, err := store.CreateStockBuy(ctx, buy)
			if err != nil {
				t.Fatalf("CreateStockBuy: %v", err)
			}
			if buyID == 0 {
				t.Fatal("expected non-zero transaction id")
			}
			gotBuy, err := store.GetTransaction(ctx, buyID)
			if err != nil {
				t.Fatalf("GetTransaction(buy): %v", err)
			}
			verifyStockBuyResult(t, gotBuy, buy, investmentAccountID, cashAccountID)

			sell := StockSell{
				Description:         "Sell AAPL",
				Date:                getDate("2025-02-02"),
				InvestmentAccountID: investmentAccountID,
				CashAccountID:       cashAccountID,
				InstrumentID:        instrumentID,
				Quantity:            3,
				TotalAmount:         465.0,
			}
			sellID, err := store.CreateStockSell(ctx, sell)
			if err != nil {
				t.Fatalf("CreateStockSell: %v", err)
			}
			if sellID == 0 {
				t.Fatal("expected non-zero transaction id")
			}
			gotSell, err := store.GetTransaction(ctx, sellID)
			if err != nil {
				t.Fatalf("GetTransaction(sell): %v", err)
			}
			verifyStockSellResult(t, gotSell, sell, investmentAccountID, cashAccountID)

			list, err := store.ListTransactions(ctx, ListOpts{
				StartDate: getDate("2025-02-01"),
				EndDate:   getDate("2025-02-28"),
				Types:     []TxType{StockBuyTransaction, StockSellTransaction},
				Limit:     10,
			})
			if err != nil {
				t.Fatalf("ListTransactions: %v", err)
			}
			if len(list) != 2 {
				t.Errorf("ListTransactions: got %d stock transactions, want 2", len(list))
			}
		})
	}
}

func TestStore_CreateStockSell_costAverage(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("storeCostAvg"))
			invID, cashID, instID := setupStockBuySellTest(t, ctx, store, mktStore)
			buy := StockBuy{
				Description:         "Buy 10 @ 1500",
				Date:                getDate("2025-02-01"),
				InvestmentAccountID: invID,
				CashAccountID:       cashID,
				InstrumentID:        instID,
				Quantity:            10,
				TotalAmount:         1500,
				StockAmount:         1500,
			}
			if _, err := store.CreateStockBuy(ctx, buy); err != nil {
				t.Fatalf("CreateStockBuy: %v", err)
			}
			sell := StockSell{
				Description:         "Sell 4",
				Date:                getDate("2025-02-02"),
				InvestmentAccountID: invID,
				CashAccountID:       cashID,
				InstrumentID:        instID,
				Quantity:            4,
				TotalAmount:         700,
			}
			sellID, err := store.CreateStockSell(ctx, sell)
			if err != nil {
				t.Fatalf("CreateStockSell: %v", err)
			}
			got, err := store.GetTransaction(ctx, sellID)
			if err != nil {
				t.Fatalf("GetTransaction: %v", err)
			}
			s := got.(StockSell)
			if s.CostBasis != 600 {
				t.Errorf("costBasis got %v want 600 (4/10 * 1500)", s.CostBasis)
			}
			if s.RealizedGainLoss != 100 {
				t.Errorf("realizedGainLoss got %v want 100", s.RealizedGainLoss)
			}
		})
	}
}

func TestStore_CreateStockSell_realizedLoss(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("storeLoss"))
			invID, cashID, instID := setupStockBuySellTest(t, ctx, store, mktStore)
			buy := StockBuy{
				Description:         "Buy 10 @ 1000",
				Date:                getDate("2025-02-01"),
				InvestmentAccountID: invID,
				CashAccountID:       cashID,
				InstrumentID:        instID,
				Quantity:            10,
				TotalAmount:         1000,
				StockAmount:         1000,
			}
			if _, err := store.CreateStockBuy(ctx, buy); err != nil {
				t.Fatalf("CreateStockBuy: %v", err)
			}
			sell := StockSell{
				Description:         "Sell 5 @ 400",
				Date:                getDate("2025-02-02"),
				InvestmentAccountID: invID,
				CashAccountID:       cashID,
				InstrumentID:        instID,
				Quantity:            5,
				TotalAmount:         400,
			}
			sellID, err := store.CreateStockSell(ctx, sell)
			if err != nil {
				t.Fatalf("CreateStockSell: %v", err)
			}
			got, err := store.GetTransaction(ctx, sellID)
			if err != nil {
				t.Fatalf("GetTransaction: %v", err)
			}
			s := got.(StockSell)
			if s.CostBasis != 500 {
				t.Errorf("costBasis got %v want 500", s.CostBasis)
			}
			if s.RealizedGainLoss != -100 {
				t.Errorf("realizedGainLoss got %v want -100", s.RealizedGainLoss)
			}
		})
	}
}

func TestStore_CreateStockSell_fullCloseReopen(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("storeFullClose"))
			invID, cashID, instID := setupStockBuySellTest(t, ctx, store, mktStore)
			buy1 := StockBuy{
				Description:         "Buy 10",
				Date:                getDate("2025-02-01"),
				InvestmentAccountID: invID,
				CashAccountID:       cashID,
				InstrumentID:        instID,
				Quantity:            10,
				TotalAmount:         1000,
				StockAmount:         1000,
			}
			if _, err := store.CreateStockBuy(ctx, buy1); err != nil {
				t.Fatalf("CreateStockBuy: %v", err)
			}
			sellAll := StockSell{
				Description:         "Sell all 10",
				Date:                getDate("2025-02-02"),
				InvestmentAccountID: invID,
				CashAccountID:       cashID,
				InstrumentID:        instID,
				Quantity:            10,
				TotalAmount:         900,
			}
			if _, err := store.CreateStockSell(ctx, sellAll); err != nil {
				t.Fatalf("CreateStockSell: %v", err)
			}
			buy2 := StockBuy{
				Description:         "Buy 5 @ 200",
				Date:                getDate("2025-02-03"),
				InvestmentAccountID: invID,
				CashAccountID:       cashID,
				InstrumentID:        instID,
				Quantity:            5,
				TotalAmount:         200,
				StockAmount:         200,
			}
			if _, err := store.CreateStockBuy(ctx, buy2); err != nil {
				t.Fatalf("CreateStockBuy: %v", err)
			}
			sell2 := StockSell{
				Description:         "Sell 2",
				Date:                getDate("2025-02-04"),
				InvestmentAccountID: invID,
				CashAccountID:       cashID,
				InstrumentID:        instID,
				Quantity:            2,
				TotalAmount:         100,
			}
			sellID, err := store.CreateStockSell(ctx, sell2)
			if err != nil {
				t.Fatalf("CreateStockSell: %v", err)
			}
			got, err := store.GetTransaction(ctx, sellID)
			if err != nil {
				t.Fatalf("GetTransaction: %v", err)
			}
			s := got.(StockSell)
			if s.CostBasis != 80 {
				t.Errorf("costBasis got %v want 80 (2/5 * 200), full close should reset cost basis", s.CostBasis)
			}
		})
	}
}

func TestStore_CreateStockSell_insufficientQuantity(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("storeInsufficient"))
			invID, cashID, instID := setupStockBuySellTest(t, ctx, store, mktStore)
			buy := StockBuy{
				Description:         "Buy 3",
				Date:                getDate("2025-02-01"),
				InvestmentAccountID: invID,
				CashAccountID:       cashID,
				InstrumentID:        instID,
				Quantity:            3,
				TotalAmount:         300,
				StockAmount:         300,
			}
			if _, err := store.CreateStockBuy(ctx, buy); err != nil {
				t.Fatalf("CreateStockBuy: %v", err)
			}
			sell := StockSell{
				Description:         "Sell 5",
				Date:                getDate("2025-02-02"),
				InvestmentAccountID: invID,
				CashAccountID:       cashID,
				InstrumentID:        instID,
				Quantity:            5,
				TotalAmount:         500,
			}
			_, err := store.CreateStockSell(ctx, sell)
			if err == nil {
				t.Fatal("expected error for insufficient quantity")
			}
			if !strings.Contains(err.Error(), "insufficient") && !strings.Contains(err.Error(), "quantity") {
				t.Errorf("error should mention insufficient quantity, got: %v", err)
			}
		})
	}
}

func TestStore_CreateStockSell_breakEven(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("storeBreakEven"))
			invID, cashID, instID := setupStockBuySellTest(t, ctx, store, mktStore)
			buy := StockBuy{
				Description:         "Buy 10 @ 1000",
				Date:                getDate("2025-02-01"),
				InvestmentAccountID: invID,
				CashAccountID:       cashID,
				InstrumentID:        instID,
				Quantity:            10,
				TotalAmount:         1000,
				StockAmount:         1000,
			}
			if _, err := store.CreateStockBuy(ctx, buy); err != nil {
				t.Fatalf("CreateStockBuy: %v", err)
			}
			sell := StockSell{
				Description:         "Sell 5 @ 500 (exact cost)",
				Date:                getDate("2025-02-02"),
				InvestmentAccountID: invID,
				CashAccountID:       cashID,
				InstrumentID:        instID,
				Quantity:            5,
				TotalAmount:         500,
			}
			sellID, err := store.CreateStockSell(ctx, sell)
			if err != nil {
				t.Fatalf("CreateStockSell: %v", err)
			}
			got, err := store.GetTransaction(ctx, sellID)
			if err != nil {
				t.Fatalf("GetTransaction: %v", err)
			}
			s := got.(StockSell)
			if s.CostBasis != 500 {
				t.Errorf("costBasis got %v want 500", s.CostBasis)
			}
			if s.RealizedGainLoss != 0 {
				t.Errorf("realizedGainLoss got %v want 0 (break-even)", s.RealizedGainLoss)
			}
			var entries []dbEntry
			if err := store.db.WithContext(ctx).Where("transaction_id = ?", sellID).Find(&entries).Error; err != nil {
				t.Fatalf("query entries: %v", err)
			}
			if len(entries) != 2 {
				t.Errorf("break-even sell should have 2 entries (no P&L), got %d", len(entries))
			}
		})
	}
}

func TestStore_CreateStockSell_withFees(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("storeFees"))
			invID, cashID, instID := setupStockBuySellTest(t, ctx, store, mktStore)
			buy := StockBuy{
				Description:         "Buy 10 @ 1000",
				Date:                getDate("2025-02-01"),
				InvestmentAccountID: invID,
				CashAccountID:       cashID,
				InstrumentID:        instID,
				Quantity:            10,
				TotalAmount:         1000,
				StockAmount:         1000,
			}
			if _, err := store.CreateStockBuy(ctx, buy); err != nil {
				t.Fatalf("CreateStockBuy: %v", err)
			}
			sell := StockSell{
				Description:         "Sell 5 @ 600 gross, fees 10",
				Date:                getDate("2025-02-02"),
				InvestmentAccountID: invID,
				CashAccountID:       cashID,
				InstrumentID:        instID,
				Quantity:            5,
				TotalAmount:         600,
				Fees:                10,
			}
			sellID, err := store.CreateStockSell(ctx, sell)
			if err != nil {
				t.Fatalf("CreateStockSell: %v", err)
			}
			got, err := store.GetTransaction(ctx, sellID)
			if err != nil {
				t.Fatalf("GetTransaction: %v", err)
			}
			s := got.(StockSell)
			if s.CostBasis != 500 {
				t.Errorf("costBasis got %v want 500", s.CostBasis)
			}
			if s.RealizedGainLoss != 90 {
				t.Errorf("realizedGainLoss got %v want 90 (600-500-10)", s.RealizedGainLoss)
			}
			if s.Fees != 10 {
				t.Errorf("fees got %v want 10", s.Fees)
			}
			var entries []dbEntry
			store.db.WithContext(ctx).Where("transaction_id = ?", sellID).Find(&entries)
			if len(entries) != 4 {
				t.Errorf("sell with gain and fees should have 4 entries, got %d", len(entries))
			}
		})
	}
}

// setupStockGrantTransferTest creates provider, unvested account, investment account and instrument for stock grant and transfer tests.
func setupStockGrantTransferTest(t *testing.T, ctx context.Context, store *Store, mktStore *marketdata.Store) (grantAccountID, investmentAccountID, instrumentID uint) {
	t.Helper()
	providerID, err := store.CreateAccountProvider(ctx, AccountProvider{Name: "Broker"})
	if err != nil {
		t.Fatal(err)
	}
	grantAccountID, err = store.CreateAccount(ctx, Account{
		AccountProviderID: providerID,
		Name:              "RSU Unvested",
		Currency:          currency.USD,
		Type:              UnvestedAccountType,
	})
	if err != nil {
		t.Fatal(err)
	}
	investmentAccountID, err = store.CreateAccount(ctx, Account{
		AccountProviderID: providerID,
		Name:              "Broker vested",
		Currency:          currency.USD,
		Type:              InvestmentAccountType,
	})
	if err != nil {
		t.Fatal(err)
	}
	instrumentID, err = mktStore.CreateInstrument(ctx, marketdata.Instrument{
		Symbol:   "RSU",
		Name:     "Company RSU",
		Currency: currency.USD,
	})
	if err != nil {
		t.Fatal(err)
	}
	return grantAccountID, investmentAccountID, instrumentID
}

func verifyStockGrantResult(t *testing.T, got Transaction, want StockGrant) {
	t.Helper()
	gotStockGrant, ok := got.(StockGrant)
	if !ok {
		t.Fatalf("expected StockGrant, got %T", got)
	}
	if gotStockGrant.Quantity != want.Quantity || gotStockGrant.AccountID != want.AccountID {
		t.Errorf("got StockGrant Quantity=%v AccountID=%v, want Quantity=%v AccountID=%v",
			gotStockGrant.Quantity, gotStockGrant.AccountID, want.Quantity, want.AccountID)
	}
}

func verifyStockTransferResult(t *testing.T, got Transaction, want StockTransfer) {
	t.Helper()
	gotStockTransfer, ok := got.(StockTransfer)
	if !ok {
		t.Fatalf("expected StockTransfer, got %T", got)
	}
	if gotStockTransfer.Quantity != want.Quantity || gotStockTransfer.SourceAccountID != want.SourceAccountID || gotStockTransfer.TargetAccountID != want.TargetAccountID {
		t.Errorf("got StockTransfer Quantity=%v Source=%v Target=%v, want Quantity=%v Source=%v Target=%v",
			gotStockTransfer.Quantity, gotStockTransfer.SourceAccountID, gotStockTransfer.TargetAccountID,
			want.Quantity, want.SourceAccountID, want.TargetAccountID)
	}
}

func TestStore_CreateStockGrant_CreateStockTransfer(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("storeCreateStockGrantTransfer"))
			grantAccountID, investmentAccountID, instrumentID := setupStockGrantTransferTest(t, ctx, store, mktStore)

			grant := StockGrant{
				Description:  "RSU grant",
				Date:         getDate("2025-03-01"),
				AccountID:    grantAccountID,
				InstrumentID: instrumentID,
				Quantity:     100,
			}
			grantID, err := store.CreateStockGrant(ctx, grant)
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}
			if grantID == 0 {
				t.Fatal("expected non-zero transaction id")
			}
			gotGrant, err := store.GetTransaction(ctx, grantID)
			if err != nil {
				t.Fatalf("GetTransaction(grant): %v", err)
			}
			verifyStockGrantResult(t, gotGrant, grant)

			transfer := StockTransfer{
				Description:     "RSU vest to brokerage",
				Date:            getDate("2025-03-15"),
				SourceAccountID: grantAccountID,
				TargetAccountID: investmentAccountID,
				InstrumentID:    instrumentID,
				Quantity:        50,
			}
			transferID, err := store.CreateStockTransfer(ctx, transfer)
			if err != nil {
				t.Fatalf("CreateStockTransfer: %v", err)
			}
			if transferID == 0 {
				t.Fatal("expected non-zero transaction id")
			}
			gotTransfer, err := store.GetTransaction(ctx, transferID)
			if err != nil {
				t.Fatalf("GetTransaction(transfer): %v", err)
			}
			verifyStockTransferResult(t, gotTransfer, transfer)

			list, err := store.ListTransactions(ctx, ListOpts{
				StartDate: getDate("2025-03-01"),
				EndDate:   getDate("2025-03-31"),
				Types:     []TxType{StockGrantTransaction, StockTransferTransaction},
				Limit:     10,
			})
			if err != nil {
				t.Fatalf("ListTransactions: %v", err)
			}
			if len(list) != 2 {
				t.Errorf("ListTransactions: got %d grant/transfer transactions, want 2", len(list))
			}
		})
	}
}

func TestStore_CreateStockBuy_validationErrors(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("storeCreateStockValidation")
			store, mktStore := newAccountingStoreWithMarketData(t, dbCon)
			providerID, _ := store.CreateAccountProvider(ctx, AccountProvider{Name: "p"})
			accountID, _ := store.CreateAccount(ctx, Account{AccountProviderID: providerID, Name: "inv", Currency: currency.USD, Type: InvestmentAccountType})
			instrumentID, _ := mktStore.CreateInstrument(ctx, marketdata.Instrument{Symbol: "X", Name: "X", Currency: currency.USD})

			cashAccountID, _ := store.CreateAccount(ctx, Account{AccountProviderID: providerID, Name: "cash", Currency: currency.USD, Type: CashAccountType})

			tcs := []struct {
				name    string
				input   StockBuy
				wantErr string
			}{
				{"investment account must be Investment", StockBuy{Description: "x", Date: getDate("2025-01-01"), InvestmentAccountID: cashAccountID, CashAccountID: cashAccountID, InstrumentID: instrumentID, Quantity: 1, TotalAmount: 1, StockAmount: 1}, "Investment"},
				{"cash account must be Cash/Checkin/Savings", StockBuy{Description: "x", Date: getDate("2025-01-01"), InvestmentAccountID: accountID, CashAccountID: accountID, InstrumentID: instrumentID, Quantity: 1, TotalAmount: 1, StockAmount: 1}, "Cash, Checkin or Savings"},
				{"instrument not found", StockBuy{Description: "x", Date: getDate("2025-01-01"), InvestmentAccountID: accountID, CashAccountID: cashAccountID, InstrumentID: 99999, Quantity: 1, TotalAmount: 1, StockAmount: 1}, "instrument not found"},
				{"quantity must be positive", StockBuy{Description: "x", Date: getDate("2025-01-01"), InvestmentAccountID: accountID, CashAccountID: cashAccountID, InstrumentID: instrumentID, Quantity: 0, TotalAmount: 1, StockAmount: 1}, "quantity must be positive"},
				{"total amount must be positive", StockBuy{Description: "x", Date: getDate("2025-01-01"), InvestmentAccountID: accountID, CashAccountID: cashAccountID, InstrumentID: instrumentID, Quantity: 1, TotalAmount: 0, StockAmount: 1}, "total amount must be positive"},
			}
			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					_, err := store.CreateStockBuy(ctx, tc.input)
					if err == nil {
						t.Fatalf("expected error containing %q", tc.wantErr)
					}
					if err.Error() != tc.wantErr && !strings.Contains(err.Error(), tc.wantErr) {
						t.Errorf("got error %v", err)
					}
				})
			}
		})
	}
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

	accProviderId, err := store.CreateAccountProvider(t.Context(), AccountProvider{Name: "p1"})
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
		{AccountProviderID: accProviderId, Name: "acc4", Currency: currency.CHF, Type: CashAccountType},
	}
	for _, acc := range Accs {
		_, err = store.CreateAccount(t.Context(), acc)
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
		_, err = store.CreateTransaction(t.Context(), tx)
		if err != nil {
			t.Fatalf("error creating account: %v", err)
		}
	}
}
