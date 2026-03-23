package accounting

import (
	"context"
	"errors"
	"fmt"
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
		{
			name:   "create valid balance status",
			tenant: tenant1,
			input: BalanceStatus{
				Description: "balance check",
				Amount:      1500.50,
				AccountID:   1,
				Date:        time.Now(),
			},
		},
		{
			name:   "balance status on investment account should fail",
			tenant: tenant1,
			input: BalanceStatus{
				Description: "balance check",
				Amount:      1500.50,
				AccountID:   5,
				Date:        time.Now(),
			},
			wantErr: "incompatible account type Investment for balance status transaction",
		},
		{
			name:   "balance status with zero account should fail",
			tenant: tenant1,
			input: BalanceStatus{
				Description: "balance check",
				Amount:      1500.50,
				AccountID:   0,
				Date:        time.Now(),
			},
			wantErr: "account id is required",
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

	// used in category filter tests (2021 date range)
	30: Income{Description: "cat-i1", Date: getDate("2021-01-01"), Amount: 1, AccountID: 1, CategoryID: 1},
	31: Expense{Description: "cat-e1", Date: getDate("2021-01-02"), Amount: 1, AccountID: 1, CategoryID: 2},
	32: Income{Description: "cat-i2", Date: getDate("2021-01-03"), Amount: 1, AccountID: 1, CategoryID: 1},
	33: Transfer{Description: "cat-t1", Date: getDate("2021-01-04"), OriginAmount: 1, OriginAccountID: 1, TargetAmount: 1, TargetAccountID: 2},
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
		{
			name:   "filter by single category",
			tenant: tenant1,
			opts: ListOpts{
				StartDate:   getDate("2021-01-01"),
				EndDate:     getDate("2021-12-31"),
				CategoryIds: []uint{1},
			},
			want: []Transaction{
				listTransactionsSampleData[32], listTransactionsSampleData[30],
			},
		},
		{
			name:   "filter by multiple categories",
			tenant: tenant1,
			opts: ListOpts{
				StartDate:   getDate("2021-01-01"),
				EndDate:     getDate("2021-12-31"),
				CategoryIds: []uint{1, 2},
			},
			want: []Transaction{
				listTransactionsSampleData[32], listTransactionsSampleData[31], listTransactionsSampleData[30],
			},
		},
	}

	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			dbCon := db.ConnDbName("TestListTransactions")
			store, _ := newAccountingStoreWithMarketData(t, dbCon)

			categorySampleData(t, store, sampleCategories)
			accountSampleData(t, store)
			// note, to optimize the test, we execute all tests on a common set of pre created transactions
			transactionSampleData(t, store, listTransactionsSampleData)

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					got, _, err := store.ListTransactions(t.Context(), tc.opts)

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

func TestStore_ListTransactions_HasAttachment(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			dbCon := db.ConnDbName("TestListTxAttachment")
			store, _ := newAccountingStoreWithMarketData(t, dbCon)
			accountSampleData(t, store)

			id1, err := store.CreateTransaction(t.Context(), Income{
				Description: "with attachment", Amount: 1, AccountID: 1, Date: getDate("2025-06-01"),
			})
			if err != nil {
				t.Fatal(err)
			}
			_, err = store.CreateTransaction(t.Context(), Income{
				Description: "without attachment", Amount: 1, AccountID: 1, Date: getDate("2025-06-02"),
			})
			if err != nil {
				t.Fatal(err)
			}

			// Set attachment_id directly
			store.db.Model(&dbTransaction{}).Where("id = ?", id1).Update("attachment_id", 999)

			hasAttachment := true
			got, _, err := store.ListTransactions(t.Context(), ListOpts{
				StartDate:     getDate("2025-06-01"),
				EndDate:       getDate("2025-06-30"),
				HasAttachment: &hasAttachment,
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 {
				t.Fatalf("expected 1 result, got %d", len(got))
			}
			if got[0].(Income).Description != "with attachment" {
				t.Errorf("expected 'with attachment', got %q", got[0].(Income).Description)
			}
		})
	}
}

func TestStore_ListTransactions_Search(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			dbCon := db.ConnDbName("TestListTxSearch")
			store, _ := newAccountingStoreWithMarketData(t, dbCon)
			accountSampleData(t, store)

			if _, err := store.CreateTransaction(t.Context(), Income{
				Description: "Grocery shopping", Amount: 50, AccountID: 1, Date: getDate("2025-07-01"),
			}); err != nil {
				t.Fatal(err)
			}
			if _, err := store.CreateTransaction(t.Context(), Income{
				Description: "Salary deposit", Notes: "monthly grocery allowance", Amount: 100, AccountID: 1, Date: getDate("2025-07-02"),
			}); err != nil {
				t.Fatal(err)
			}
			if _, err := store.CreateTransaction(t.Context(), Income{
				Description: "Gas station", Amount: 30, AccountID: 1, Date: getDate("2025-07-03"),
			}); err != nil {
				t.Fatal(err)
			}

			tcs := []struct {
				name   string
				search string
				want   int
			}{
				{"match description", "grocery", 2},
				{"match notes", "allowance", 1},
				{"case insensitive", "GROCERY", 2},
				{"partial match", "gro", 2},
				{"no match", "xyz", 0},
				{"empty search returns all", "", 3},
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					got, _, err := store.ListTransactions(t.Context(), ListOpts{
						StartDate: getDate("2025-07-01"),
						EndDate:   getDate("2025-07-31"),
						Search:    tc.search,
					})
					if err != nil {
						t.Fatal(err)
					}
					if len(got) != tc.want {
						t.Errorf("search %q: expected %d results, got %d", tc.search, tc.want, len(got))
					}
				})
			}
		})
	}
}

func TestStore_PriorPageBalance(t *testing.T) {
	// Uses the same sample data as TestStore_ListTransactions (pagination entries 20-28).
	// 2022 range entries (DESC): p9(acct3) p8(acct1) p7(acct1) p6(acct1) p5(acct2) p4(acct1) p3(acct1) p2(acct1) p1(acct1)
	// When filtered to account 1 (7 txs): p8 p7 p6 p4 p3 p2 p1

	tcs := []struct {
		name      string
		opts      ListOpts
		accountID uint
		want      float64
	}{
		{
			name: "page 1, all older entries for account 1",
			opts: ListOpts{
				StartDate: getDate("2022-01-01"),
				EndDate:   getDate("2022-12-31"),
				AccountId: []int{1},
				Limit:     2, Page: 1,
			},
			accountID: 1,
			want:      5, // 5 income entries of amount 1 on older pages
		},
		{
			name: "page 2, older entries for account 1",
			opts: ListOpts{
				StartDate: getDate("2022-01-01"),
				EndDate:   getDate("2022-12-31"),
				AccountId: []int{1},
				Limit:     2, Page: 2,
			},
			accountID: 1,
			want:      3, // 3 income entries of amount 1 on older pages
		},
		{
			name: "last page, no older entries",
			opts: ListOpts{
				StartDate: getDate("2022-01-01"),
				EndDate:   getDate("2022-12-31"),
				AccountId: []int{1},
				Limit:     2, Page: 4,
			},
			accountID: 1,
			want:      0, // last page, no older entries
		},
		{
			name: "page 1 with large limit, no older entries",
			opts: ListOpts{
				StartDate: getDate("2022-01-01"),
				EndDate:   getDate("2022-12-31"),
				AccountId: []int{1},
				Limit:     100, Page: 1,
			},
			accountID: 1,
			want:      0, // all entries fit on page 1
		},
	}

	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			dbCon := db.ConnDbName("TestPriorPageBalance")
			store, _ := newAccountingStoreWithMarketData(t, dbCon)

			categorySampleData(t, store, sampleCategories)
			accountSampleData(t, store)
			transactionSampleData(t, store, listTransactionsSampleData)

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					got, err := store.PriorPageBalance(t.Context(), tc.opts, tc.accountID)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					if got != tc.want {
						t.Errorf("PriorPageBalance = %v, want %v", got, tc.want)
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
	cmpopts.IgnoreUnexported(BalanceStatus{}),
	cmpopts.IgnoreFields(BalanceStatus{}, "Id"),
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

			list, _, err := store.ListTransactions(ctx, ListOpts{
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
func TestStore_UpdateStockBuy(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("storeUpdateStockBuy"))
			invID, cashID, instID := setupStockBuySellTest(t, ctx, store, mktStore)

			buy := StockBuy{
				Description:         "Buy AAPL",
				Date:                getDate("2025-02-01"),
				InvestmentAccountID: invID,
				CashAccountID:       cashID,
				InstrumentID:        instID,
				Quantity:            10,
				TotalAmount:         1500.0,
				StockAmount:         1850.0,
			}
			buyID, err := store.CreateStockBuy(ctx, buy)
			if err != nil {
				t.Fatalf("CreateStockBuy: %v", err)
			}

			// Update description
			err = store.UpdateStockBuy(ctx, StockBuyUpdate{Description: ptr("Updated Buy")}, buyID)
			if err != nil {
				t.Fatalf("UpdateStockBuy description: %v", err)
			}
			got, err := store.GetTransaction(ctx, buyID)
			if err != nil {
				t.Fatalf("GetTransaction: %v", err)
			}
			gotBuy, ok := got.(StockBuy)
			if !ok {
				t.Fatalf("expected StockBuy, got %T", got)
			}
			if gotBuy.Description != "Updated Buy" {
				t.Errorf("expected description 'Updated Buy', got %q", gotBuy.Description)
			}

			// Update quantity
			err = store.UpdateStockBuy(ctx, StockBuyUpdate{Quantity: ptr(20.0)}, buyID)
			if err != nil {
				t.Fatalf("UpdateStockBuy quantity: %v", err)
			}
			got, _ = store.GetTransaction(ctx, buyID)
			gotBuy = got.(StockBuy)
			if gotBuy.Quantity != 20 {
				t.Errorf("expected quantity 20, got %v", gotBuy.Quantity)
			}

			// Validation error: empty description
			err = store.UpdateStockBuy(ctx, StockBuyUpdate{Description: ptr("")}, buyID)
			if err == nil {
				t.Fatal("expected error for empty description")
			}

			// Validation error: zero quantity
			err = store.UpdateStockBuy(ctx, StockBuyUpdate{Quantity: ptr(0.0)}, buyID)
			if err == nil {
				t.Fatal("expected error for zero quantity")
			}

			// Validation error: non-existent transaction
			err = store.UpdateStockBuy(ctx, StockBuyUpdate{Description: ptr("x")}, 99999)
			if err == nil {
				t.Fatal("expected error for non-existent transaction")
			}
		})
	}
}

func TestStore_UpdateStockSell(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("storeUpdateStockSell"))
			invID, cashID, instID := setupStockBuySellTest(t, ctx, store, mktStore)

			// First create a buy to have lots available for selling
			_, err := store.CreateStockBuy(ctx, StockBuy{
				Description:         "Buy AAPL",
				Date:                getDate("2025-02-01"),
				InvestmentAccountID: invID,
				CashAccountID:       cashID,
				InstrumentID:        instID,
				Quantity:            10,
				TotalAmount:         1500.0,
				StockAmount:         1850.0,
			})
			if err != nil {
				t.Fatalf("CreateStockBuy: %v", err)
			}

			sell := StockSell{
				Description:         "Sell AAPL",
				Date:                getDate("2025-02-02"),
				InvestmentAccountID: invID,
				CashAccountID:       cashID,
				InstrumentID:        instID,
				Quantity:            3,
				TotalAmount:         465.0,
			}
			sellID, err := store.CreateStockSell(ctx, sell)
			if err != nil {
				t.Fatalf("CreateStockSell: %v", err)
			}

			// Update description
			err = store.UpdateStockSell(ctx, StockSellUpdate{Description: ptr("Updated Sell")}, sellID)
			if err != nil {
				t.Fatalf("UpdateStockSell description: %v", err)
			}
			got, err := store.GetTransaction(ctx, sellID)
			if err != nil {
				t.Fatalf("GetTransaction: %v", err)
			}
			gotSell, ok := got.(StockSell)
			if !ok {
				t.Fatalf("expected StockSell, got %T", got)
			}
			if gotSell.Description != "Updated Sell" {
				t.Errorf("expected description 'Updated Sell', got %q", gotSell.Description)
			}

			// Validation error: empty description
			err = store.UpdateStockSell(ctx, StockSellUpdate{Description: ptr("")}, sellID)
			if err == nil {
				t.Fatal("expected error for empty description")
			}

			// Validation error: non-existent transaction
			err = store.UpdateStockSell(ctx, StockSellUpdate{Description: ptr("x")}, 99999)
			if err == nil {
				t.Fatal("expected error for non-existent transaction")
			}

			// Regression: updating a sell must restore lot quantities first so the
			// recreated sell can allocate from the same lots without "insufficient quantity".
			qty := 5.0
			err = store.UpdateStockSell(ctx, StockSellUpdate{Quantity: &qty, TotalAmount: ptr(775.0)}, sellID)
			if err != nil {
				t.Fatalf("UpdateStockSell quantity change (lot restore regression): %v", err)
			}
			got, err = store.GetTransaction(ctx, sellID)
			if err != nil {
				t.Fatalf("GetTransaction after qty change: %v", err)
			}
			gotSell, ok = got.(StockSell)
			if !ok {
				t.Fatalf("expected StockSell, got %T", got)
			}
			if gotSell.Quantity != 5 {
				t.Errorf("expected quantity 5, got %v", gotSell.Quantity)
			}
		})
	}
}

func TestStore_UpdateStockGrant(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("storeUpdateStockGrant"))
			grantAccountID, _, instrumentID := setupStockGrantTransferTest(t, ctx, store, mktStore)

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

			// Update description
			err = store.UpdateStockGrant(ctx, StockGrantUpdate{Description: ptr("Updated Grant")}, grantID)
			if err != nil {
				t.Fatalf("UpdateStockGrant: %v", err)
			}
			got, err := store.GetTransaction(ctx, grantID)
			if err != nil {
				t.Fatalf("GetTransaction: %v", err)
			}
			gotGrant, ok := got.(StockGrant)
			if !ok {
				t.Fatalf("expected StockGrant, got %T", got)
			}
			if gotGrant.Description != "Updated Grant" {
				t.Errorf("expected description 'Updated Grant', got %q", gotGrant.Description)
			}

			// Update quantity
			err = store.UpdateStockGrant(ctx, StockGrantUpdate{Quantity: ptr(200.0)}, grantID)
			if err != nil {
				t.Fatalf("UpdateStockGrant quantity: %v", err)
			}
			got, _ = store.GetTransaction(ctx, grantID)
			gotGrant = got.(StockGrant)
			if gotGrant.Quantity != 200 {
				t.Errorf("expected quantity 200, got %v", gotGrant.Quantity)
			}

			// Validation error: empty description
			err = store.UpdateStockGrant(ctx, StockGrantUpdate{Description: ptr("")}, grantID)
			if err == nil {
				t.Fatal("expected error for empty description")
			}

			// Validation error: non-existent transaction
			err = store.UpdateStockGrant(ctx, StockGrantUpdate{Description: ptr("x")}, 99999)
			if err == nil {
				t.Fatal("expected error for non-existent transaction")
			}
		})
	}
}

func TestStore_UpdateStockTransfer(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("storeUpdateStockXfer"))

			// Create two investment accounts for transfer.
			providerID, err := store.CreateAccountProvider(ctx, AccountProvider{Name: "Broker"})
			if err != nil {
				t.Fatal(err)
			}
			invA, err := store.CreateAccount(ctx, Account{AccountProviderID: providerID, Name: "Investment A", Currency: currency.USD, Type: InvestmentAccountType})
			if err != nil {
				t.Fatal(err)
			}
			invB, err := store.CreateAccount(ctx, Account{AccountProviderID: providerID, Name: "Investment B", Currency: currency.USD, Type: InvestmentAccountType})
			if err != nil {
				t.Fatal(err)
			}
			instrumentID, err := mktStore.CreateInstrument(ctx, marketdata.Instrument{Symbol: "RSU", Name: "Company RSU", Currency: currency.USD})
			if err != nil {
				t.Fatal(err)
			}

			// First create a grant so there are lots to transfer
			_, err = store.CreateStockGrant(ctx, StockGrant{
				Description:  "RSU grant",
				Date:         getDate("2025-03-01"),
				AccountID:    invA,
				InstrumentID: instrumentID,
				Quantity:     100,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			transfer := StockTransfer{
				Description:     "transfer shares",
				Date:            getDate("2025-03-15"),
				SourceAccountID: invA,
				TargetAccountID: invB,
				InstrumentID:    instrumentID,
				Quantity:        50,
			}
			transferID, err := store.CreateStockTransfer(ctx, transfer)
			if err != nil {
				t.Fatalf("CreateStockTransfer: %v", err)
			}

			// Update description
			err = store.UpdateStockTransfer(ctx, StockTransferUpdate{Description: ptr("Updated Transfer")}, transferID)
			if err != nil {
				t.Fatalf("UpdateStockTransfer: %v", err)
			}
			got, err := store.GetTransaction(ctx, transferID)
			if err != nil {
				t.Fatalf("GetTransaction: %v", err)
			}
			gotTransfer, ok := got.(StockTransfer)
			if !ok {
				t.Fatalf("expected StockTransfer, got %T", got)
			}
			if gotTransfer.Description != "Updated Transfer" {
				t.Errorf("expected description 'Updated Transfer', got %q", gotTransfer.Description)
			}

			// Validation error: empty description
			err = store.UpdateStockTransfer(ctx, StockTransferUpdate{Description: ptr("")}, transferID)
			if err == nil {
				t.Fatal("expected error for empty description")
			}

			// Validation error: non-existent transaction
			err = store.UpdateStockTransfer(ctx, StockTransferUpdate{Description: ptr("x")}, 99999)
			if err == nil {
				t.Fatal("expected error for non-existent transaction")
			}
		})
	}
}

func TestStore_UpdateTransaction(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("storeUpdateTxDispatch"))
			categorySampleData(t, store, sampleCategories)
			accountSampleData(t, store)

			// Create an income transaction
			incomeID, err := store.CreateTransaction(ctx, Income{
				Description: "salary",
				Amount:      1000,
				AccountID:   1,
				CategoryID:  1,
				Date:        getDate("2025-01-01"),
			})
			if err != nil {
				t.Fatalf("CreateTransaction income: %v", err)
			}

			// Update income via UpdateTransaction dispatcher
			err = store.UpdateTransaction(ctx, IncomeUpdate{Description: ptr("updated salary")}, incomeID)
			if err != nil {
				t.Fatalf("UpdateTransaction income: %v", err)
			}
			got, _ := store.GetTransaction(ctx, incomeID)
			if income, ok := got.(Income); !ok || income.Description != "updated salary" {
				t.Errorf("expected income with description 'updated salary', got %+v", got)
			}

			// Create an expense transaction (CategoryID 2 = "Home" = ExpenseCategory)
			expenseID, err := store.CreateTransaction(ctx, Expense{
				Description: "groceries",
				Amount:      50,
				AccountID:   1,
				CategoryID:  2,
				Date:        getDate("2025-01-02"),
			})
			if err != nil {
				t.Fatalf("CreateTransaction expense: %v", err)
			}

			// Update expense via UpdateTransaction dispatcher
			err = store.UpdateTransaction(ctx, ExpenseUpdate{Description: ptr("updated groceries")}, expenseID)
			if err != nil {
				t.Fatalf("UpdateTransaction expense: %v", err)
			}
			got, _ = store.GetTransaction(ctx, expenseID)
			if expense, ok := got.(Expense); !ok || expense.Description != "updated groceries" {
				t.Errorf("expected expense with description 'updated groceries', got %+v", got)
			}

			// Test stock buy via UpdateTransaction dispatcher
			invID, cashID, instID := setupStockBuySellTest(t, ctx, store, mktStore)
			buyID, err := store.CreateStockBuy(ctx, StockBuy{
				Description:         "Buy shares",
				Date:                getDate("2025-02-01"),
				InvestmentAccountID: invID,
				CashAccountID:       cashID,
				InstrumentID:        instID,
				Quantity:            10,
				TotalAmount:         1500.0,
				StockAmount:         1850.0,
			})
			if err != nil {
				t.Fatalf("CreateStockBuy: %v", err)
			}

			err = store.UpdateTransaction(ctx, StockBuyUpdate{Description: ptr("Updated buy")}, buyID)
			if err != nil {
				t.Fatalf("UpdateTransaction stock buy: %v", err)
			}
			got, _ = store.GetTransaction(ctx, buyID)
			if sb, ok := got.(StockBuy); !ok || sb.Description != "Updated buy" {
				t.Errorf("expected stock buy with description 'Updated buy', got %+v", got)
			}

			// Test invalid update type
			err = store.UpdateTransaction(ctx, EmptyTransactionUpdate{}, incomeID)
			if err == nil {
				t.Fatal("expected error for invalid transaction update type")
			}
		})
	}
}

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

			// Create two investment accounts for transfer.
			providerID, err := store.CreateAccountProvider(ctx, AccountProvider{Name: "Broker"})
			if err != nil {
				t.Fatal(err)
			}
			invA, err := store.CreateAccount(ctx, Account{AccountProviderID: providerID, Name: "Investment A", Currency: currency.USD, Type: InvestmentAccountType})
			if err != nil {
				t.Fatal(err)
			}
			invB, err := store.CreateAccount(ctx, Account{AccountProviderID: providerID, Name: "Investment B", Currency: currency.USD, Type: InvestmentAccountType})
			if err != nil {
				t.Fatal(err)
			}
			instrumentID, err := mktStore.CreateInstrument(ctx, marketdata.Instrument{Symbol: "RSU", Name: "Company RSU", Currency: currency.USD})
			if err != nil {
				t.Fatal(err)
			}

			grant := StockGrant{
				Description:  "RSU grant",
				Date:         getDate("2025-03-01"),
				AccountID:    invA,
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
				Description:     "transfer to brokerage B",
				Date:            getDate("2025-03-15"),
				SourceAccountID: invA,
				TargetAccountID: invB,
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

			list, _, err := store.ListTransactions(ctx, ListOpts{
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
				{"cash account must be Cash/Checkin/Savings", StockBuy{Description: "x", Date: getDate("2025-01-01"), InvestmentAccountID: accountID, CashAccountID: accountID, InstrumentID: instrumentID, Quantity: 1, TotalAmount: 1, StockAmount: 1}, "Cash, Checkin, Savings or Lent"},
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
		{AccountProviderID: accProviderId, Name: "acc5", Currency: currency.EUR, Type: InvestmentAccountType},
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

func TestStore_BalanceStatus_RoundTrip(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			dbCon := db.ConnDbName("balanceStatusRoundTrip")
			store, _ := newAccountingStoreWithMarketData(t, dbCon)

			categorySampleData(t, store, sampleCategories)
			transactionSampleData(t, store, sampleTransactions)

			input := BalanceStatus{
				Description: "March statement balance",
				Amount:      2500.75,
				AccountID:   1,
				Date:        getDate("2025-03-01"),
			}

			id, err := store.CreateTransaction(t.Context(), input)
			if err != nil {
				t.Fatalf("unexpected error creating balance status: %v", err)
			}
			if id == 0 {
				t.Fatal("expected valid id, got 0")
			}

			got, err := store.GetTransaction(t.Context(), id)
			if err != nil {
				t.Fatalf("unexpected error getting balance status: %v", err)
			}

			bs, ok := got.(BalanceStatus)
			if !ok {
				t.Fatalf("expected BalanceStatus, got %T", got)
			}

			if bs.Id != id {
				t.Errorf("expected Id=%d, got %d", id, bs.Id)
			}
			if bs.Description != input.Description {
				t.Errorf("expected Description=%q, got %q", input.Description, bs.Description)
			}
			if bs.Amount != input.Amount {
				t.Errorf("expected Amount=%f, got %f", input.Amount, bs.Amount)
			}
			if bs.AccountID != input.AccountID {
				t.Errorf("expected AccountID=%d, got %d", input.AccountID, bs.AccountID)
			}
			if !bs.Date.Equal(input.Date) {
				t.Errorf("expected Date=%v, got %v", input.Date, bs.Date)
			}
		})
	}
}

func TestStore_UpdateBalanceStatus(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			dbCon := db.ConnDbName("updateBalanceStatus")
			store, _ := newAccountingStoreWithMarketData(t, dbCon)

			categorySampleData(t, store, sampleCategories)
			transactionSampleData(t, store, sampleTransactions)

			input := BalanceStatus{
				Description: "original balance",
				Amount:      1000.00,
				AccountID:   1,
				Date:        getDate("2025-03-01"),
			}

			id, err := store.CreateTransaction(t.Context(), input)
			if err != nil {
				t.Fatalf("unexpected error creating balance status: %v", err)
			}

			newDesc := "updated balance"
			newAmount := 2000.50
			err = store.UpdateTransaction(t.Context(), BalanceStatusUpdate{
				Description: &newDesc,
				Amount:      &newAmount,
			}, id)
			if err != nil {
				t.Fatalf("unexpected error updating balance status: %v", err)
			}

			got, err := store.GetTransaction(t.Context(), id)
			if err != nil {
				t.Fatalf("unexpected error getting balance status: %v", err)
			}

			bs, ok := got.(BalanceStatus)
			if !ok {
				t.Fatalf("expected BalanceStatus, got %T", got)
			}

			if bs.Description != newDesc {
				t.Errorf("expected Description=%q, got %q", newDesc, bs.Description)
			}
			if bs.Amount != newAmount {
				t.Errorf("expected Amount=%f, got %f", newAmount, bs.Amount)
			}
			// Unchanged fields should remain
			if bs.AccountID != input.AccountID {
				t.Errorf("expected AccountID=%d, got %d", input.AccountID, bs.AccountID)
			}
		})
	}
}

func TestStore_BalanceStatus_DoesNotAffectBalance(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			dbCon := db.ConnDbName("TestBalanceStatusDoesNotAffectBalance")
			store, _ := newAccountingStoreWithMarketData(t, dbCon)

			accountSampleData(t, store)
			ctx := t.Context()

			// Create an income entry: amount=1000, accountID=1, date 2025-01-15
			_, err := store.CreateTransaction(ctx, Income{
				Description: "salary",
				Amount:      1000,
				AccountID:   1,
				Date:        getDate("2025-01-15"),
			})
			if err != nil {
				t.Fatalf("unexpected error creating income: %v", err)
			}

			// Get balance before balance status
			balBefore, err := store.AccountBalanceSingle(ctx, 1, getDate("2025-12-31"))
			if err != nil {
				t.Fatalf("unexpected error getting balance: %v", err)
			}

			// Create a balance status: amount=5000, accountID=1, date 2025-06-15
			_, err = store.CreateTransaction(ctx, BalanceStatus{
				Description: "bank statement",
				Amount:      5000,
				AccountID:   1,
				Date:        getDate("2025-06-15"),
			})
			if err != nil {
				t.Fatalf("unexpected error creating balance status: %v", err)
			}

			// Get balance after balance status
			balAfter, err := store.AccountBalanceSingle(ctx, 1, getDate("2025-12-31"))
			if err != nil {
				t.Fatalf("unexpected error getting balance after: %v", err)
			}

			if balBefore.Sum != balAfter.Sum {
				t.Errorf("balance changed after adding balance status: before=%f, after=%f", balBefore.Sum, balAfter.Sum)
			}
		})
	}
}

func TestStore_PriorPageBalance_ExcludesBalanceStatus(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			dbCon := db.ConnDbName("TestPriorPageBalanceExcludesBalanceStatus")
			store, _ := newAccountingStoreWithMarketData(t, dbCon)

			accountSampleData(t, store)
			ctx := t.Context()

			// Create 5 income entries: amount=100 each, accountID=1
			for i := 1; i <= 5; i++ {
				_, err := store.CreateTransaction(ctx, Income{
					Description: fmt.Sprintf("income %d", i),
					Amount:      100,
					AccountID:   1,
					Date:        getDate(fmt.Sprintf("2025-01-0%d", i)),
				})
				if err != nil {
					t.Fatalf("unexpected error creating income %d: %v", i, err)
				}
			}

			// Create a balance status: amount=9999, accountID=1, date 2025-01-03
			_, err := store.CreateTransaction(ctx, BalanceStatus{
				Description: "bank statement",
				Amount:      9999,
				AccountID:   1,
				Date:        getDate("2025-01-03"),
			})
			if err != nil {
				t.Fatalf("unexpected error creating balance status: %v", err)
			}

			// Transactions sorted DESC by date, then by ID:
			// income Jan 5 (100), income Jan 4 (100), balance status Jan 3 (9999), income Jan 3 (100), income Jan 2 (100), income Jan 1 (100)
			// Page 1 (limit=2) shows: income Jan 5 + income Jan 4
			// Prior balance = sum of remaining income entries = 100 + 100 + 100 = 300
			opts := ListOpts{
				StartDate: getDate("2025-01-01"),
				EndDate:   getDate("2025-12-31"),
				AccountId: []int{1},
				Limit:     2,
				Page:      1,
			}

			got, err := store.PriorPageBalance(ctx, opts, 1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			want := 300.0
			if got != want {
				t.Errorf("PriorPageBalance = %v, want %v (balance status amount should not contribute)", got, want)
			}
		})
	}
}

func TestStore_ListTransactions_CombinedFilters(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			dbCon := db.ConnDbName("TestListTxCombined")
			store, _ := newAccountingStoreWithMarketData(t, dbCon)
			categorySampleData(t, store, sampleCategories)
			accountSampleData(t, store)

			if _, err := store.CreateTransaction(t.Context(), Income{
				Description: "Grocery income", Amount: 1, AccountID: 1, CategoryID: 1, Date: getDate("2025-08-01"),
			}); err != nil {
				t.Fatal(err)
			}
			if _, err := store.CreateTransaction(t.Context(), Expense{
				Description: "Grocery expense", Amount: 1, AccountID: 1, CategoryID: 2, Date: getDate("2025-08-02"),
			}); err != nil {
				t.Fatal(err)
			}
			if _, err := store.CreateTransaction(t.Context(), Income{
				Description: "Salary", Amount: 1, AccountID: 1, CategoryID: 1, Date: getDate("2025-08-03"),
			}); err != nil {
				t.Fatal(err)
			}

			got, _, err := store.ListTransactions(t.Context(), ListOpts{
				StartDate:   getDate("2025-08-01"),
				EndDate:     getDate("2025-08-31"),
				CategoryIds: []uint{1},
				Search:      "grocery",
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 {
				t.Errorf("expected 1 result, got %d", len(got))
			}
		})
	}
}

func TestStore_PriorPageBalance_IgnoresNewFilters(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			dbCon := db.ConnDbName("TestPriorBalNewFilters")
			store, _ := newAccountingStoreWithMarketData(t, dbCon)
			accountSampleData(t, store)

			for i := 1; i <= 3; i++ {
				if _, err := store.CreateTransaction(t.Context(), Income{
					Description: fmt.Sprintf("income %d", i),
					Amount:      10,
					AccountID:   1,
					CategoryID:  0,
					Date:        getDate(fmt.Sprintf("2025-09-%02d", i)),
				}); err != nil {
					t.Fatal(err)
				}
			}

			hasAttachment := true
			baseOpts := ListOpts{
				StartDate: getDate("2025-09-01"),
				EndDate:   getDate("2025-09-30"),
				AccountId: []int{1},
				Limit:     1, Page: 1,
			}
			basePrior, err := store.PriorPageBalance(t.Context(), baseOpts, 1)
			if err != nil {
				t.Fatal(err)
			}

			filteredOpts := ListOpts{
				StartDate:     getDate("2025-09-01"),
				EndDate:       getDate("2025-09-30"),
				AccountId:     []int{1},
				Limit:         1, Page: 1,
				CategoryIds:   []uint{999},
				HasAttachment: &hasAttachment,
				Search:        "nonexistent",
			}
			filteredPrior, err := store.PriorPageBalance(t.Context(), filteredOpts, 1)
			if err != nil {
				t.Fatal(err)
			}

			if basePrior != filteredPrior {
				t.Errorf("PriorPageBalance should ignore new filters: base=%f filtered=%f", basePrior, filteredPrior)
			}
		})
	}
}

// setupCreateStockVest creates a grant and vest, returning the vest ID and category ID for verification.
func setupCreateStockVest(t *testing.T, ctx context.Context, store *Store,
	unvestedID, investmentID, instrumentID uint,
) (vestID uint, categoryID uint) {
	t.Helper()
	_, err := store.CreateStockGrant(ctx, StockGrant{
		Description:     "RSU Grant 100 shares",
		Date:            getDate("2025-01-15"),
		AccountID:       unvestedID,
		InstrumentID:    instrumentID,
		Quantity:        100,
		FairMarketValue: 50.0,
	})
	if err != nil {
		t.Fatalf("CreateStockGrant: %v", err)
	}

	lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
	if err != nil {
		t.Fatalf("ListLots: %v", err)
	}
	if len(lots) != 1 {
		t.Fatalf("expected 1 lot in unvested account, got %d", len(lots))
	}
	lotID := lots[0].Id

	categoryID, err = store.CreateCategory(ctx, CategoryData{Name: "RSU Income", Type: IncomeCategory}, 0)
	if err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}

	vestID, err = store.CreateStockVest(ctx, StockVest{
		Description:     "Vest 60 shares",
		Date:            getDate("2025-06-15"),
		SourceAccountID: unvestedID,
		TargetAccountID: investmentID,
		InstrumentID:    instrumentID,
		VestingPrice:    75.0,
		CategoryID:      categoryID,
		LotSelections:   []LotSelection{{LotID: lotID, Quantity: 60}},
	})
	if err != nil {
		t.Fatalf("CreateStockVest: %v", err)
	}
	if vestID == 0 {
		t.Fatal("expected non-zero transaction id")
	}
	return vestID, categoryID
}

func verifyStockVestGetTransaction(t *testing.T, ctx context.Context, store *Store,
	vestID, unvestedID, investmentID, instrumentID, categoryID uint,
) {
	t.Helper()
	got, err := store.GetTransaction(ctx, vestID)
	if err != nil {
		t.Fatalf("GetTransaction: %v", err)
	}
	gotVest, ok := got.(StockVest)
	if !ok {
		t.Fatalf("expected StockVest, got %T", got)
	}
	if gotVest.SourceAccountID != unvestedID {
		t.Errorf("SourceAccountID: got %d, want %d", gotVest.SourceAccountID, unvestedID)
	}
	if gotVest.TargetAccountID != investmentID {
		t.Errorf("TargetAccountID: got %d, want %d", gotVest.TargetAccountID, investmentID)
	}
	if gotVest.InstrumentID != instrumentID {
		t.Errorf("InstrumentID: got %d, want %d", gotVest.InstrumentID, instrumentID)
	}
	if gotVest.VestingPrice != 75.0 {
		t.Errorf("VestingPrice: got %v, want 75", gotVest.VestingPrice)
	}
	if gotVest.CategoryID != categoryID {
		t.Errorf("CategoryID: got %d, want %d", gotVest.CategoryID, categoryID)
	}
	if gotVest.Description != "Vest 60 shares" {
		t.Errorf("Description: got %q, want %q", gotVest.Description, "Vest 60 shares")
	}
}

func verifyStockVestLots(t *testing.T, ctx context.Context, store *Store,
	unvestedID, investmentID, instrumentID uint,
) {
	t.Helper()
	targetLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: investmentID, InstrumentID: instrumentID})
	if err != nil {
		t.Fatalf("ListLots(target): %v", err)
	}
	if len(targetLots) != 1 {
		t.Fatalf("expected 1 lot in target account, got %d", len(targetLots))
	}
	if targetLots[0].CostPerShare != 75.0 {
		t.Errorf("target lot CostPerShare: got %v, want 75", targetLots[0].CostPerShare)
	}
	if targetLots[0].Quantity != 60 {
		t.Errorf("target lot Quantity: got %v, want 60", targetLots[0].Quantity)
	}

	sourceLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
	if err != nil {
		t.Fatalf("ListLots(source): %v", err)
	}
	totalRemaining := 0.0
	for _, l := range sourceLots {
		totalRemaining += l.Quantity
	}
	if totalRemaining != 40 {
		t.Errorf("source remaining quantity: got %v, want 40", totalRemaining)
	}
}

func verifyStockVestListTransactions(t *testing.T, ctx context.Context, store *Store,
	vestID, unvestedID, investmentID, instrumentID, categoryID uint,
) {
	t.Helper()
	list, count, err := store.ListTransactions(ctx, ListOpts{
		StartDate: getDate("2025-06-01"),
		EndDate:   getDate("2025-06-30"),
		Types:     []TxType{StockVestTransaction},
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}
	if count != 1 {
		t.Fatalf("ListTransactions count: got %d, want 1", count)
	}
	if len(list) != 1 {
		t.Fatalf("ListTransactions len: got %d, want 1", len(list))
	}
	listedVest, ok := list[0].(StockVest)
	if !ok {
		t.Fatalf("expected StockVest from ListTransactions, got %T", list[0])
	}
	if listedVest.Id != vestID {
		t.Errorf("ListTransactions Id: got %d, want %d", listedVest.Id, vestID)
	}
	if listedVest.Description != "Vest 60 shares" {
		t.Errorf("ListTransactions Description: got %q, want %q", listedVest.Description, "Vest 60 shares")
	}
	if listedVest.SourceAccountID != unvestedID {
		t.Errorf("ListTransactions SourceAccountID: got %d, want %d", listedVest.SourceAccountID, unvestedID)
	}
	if listedVest.TargetAccountID != investmentID {
		t.Errorf("ListTransactions TargetAccountID: got %d, want %d", listedVest.TargetAccountID, investmentID)
	}
	if listedVest.InstrumentID != instrumentID {
		t.Errorf("ListTransactions InstrumentID: got %d, want %d", listedVest.InstrumentID, instrumentID)
	}
	if listedVest.VestingPrice != 75.0 {
		t.Errorf("ListTransactions VestingPrice: got %v, want 75", listedVest.VestingPrice)
	}
	if listedVest.CategoryID != categoryID {
		t.Errorf("ListTransactions CategoryID: got %d, want %d", listedVest.CategoryID, categoryID)
	}
}

func TestStore_CreateStockVest(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("storeCreateStockVest"))
			unvestedID, investmentID, instrumentID := setupStockGrantTransferTest(t, ctx, store, mktStore)

			vestID, categoryID := setupCreateStockVest(t, ctx, store, unvestedID, investmentID, instrumentID)

			t.Run("verify GetTransaction", func(t *testing.T) {
				verifyStockVestGetTransaction(t, ctx, store, vestID, unvestedID, investmentID, instrumentID, categoryID)
			})
			t.Run("verify lots", func(t *testing.T) {
				verifyStockVestLots(t, ctx, store, unvestedID, investmentID, instrumentID)
			})
			t.Run("verify ListTransactions", func(t *testing.T) {
				verifyStockVestListTransactions(t, ctx, store, vestID, unvestedID, investmentID, instrumentID, categoryID)
			})
		})
	}
}

func TestStore_UpdateStockVest(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("storeUpdateStockVest"))
			unvestedID, investmentID, instrumentID := setupStockGrantTransferTest(t, ctx, store, mktStore)

			// 1. Create a grant (100 shares, FMV $50)
			_, err := store.CreateStockGrant(ctx, StockGrant{
				Description:     "RSU Grant 100 shares",
				Date:            getDate("2025-01-15"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			// 2. List lots to get the lot ID
			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			if len(lots) != 1 {
				t.Fatalf("expected 1 lot in unvested account, got %d", len(lots))
			}
			lotID := lots[0].Id

			// 3. Create income category
			categoryID, err := store.CreateCategory(ctx, CategoryData{Name: "RSU Income", Type: IncomeCategory}, 0)
			if err != nil {
				t.Fatalf("CreateCategory: %v", err)
			}

			// 4. Create StockVest (60 shares at $75)
			vest := StockVest{
				Description:     "Vest 60 shares",
				Date:            getDate("2025-06-15"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    75.0,
				CategoryID:      categoryID,
				LotSelections: []LotSelection{
					{LotID: lotID, Quantity: 60},
				},
			}
			vestID, err := store.CreateStockVest(ctx, vest)
			if err != nil {
				t.Fatalf("CreateStockVest: %v", err)
			}

			// 5. Update: change vesting price to $80
			newPrice := 80.0
			err = store.UpdateStockVest(ctx, StockVestUpdate{
				VestingPrice: &newPrice,
			}, vestID)
			if err != nil {
				t.Fatalf("UpdateStockVest: %v", err)
			}

			// 6. Verify via GetTransaction that VestingPrice is now $80
			got, err := store.GetTransaction(ctx, vestID)
			if err != nil {
				t.Fatalf("GetTransaction after update: %v", err)
			}
			gotVest, ok := got.(StockVest)
			if !ok {
				t.Fatalf("expected StockVest, got %T", got)
			}
			if gotVest.VestingPrice != 80.0 {
				t.Errorf("VestingPrice: got %v, want 80", gotVest.VestingPrice)
			}

			// 7. Verify target lot costPerShare is $80
			targetLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: investmentID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(target) after update: %v", err)
			}
			if len(targetLots) != 1 {
				t.Fatalf("expected 1 lot in target account, got %d", len(targetLots))
			}
			if targetLots[0].CostPerShare != 80.0 {
				t.Errorf("target lot CostPerShare: got %v, want 80", targetLots[0].CostPerShare)
			}
		})
	}
}

// setupVestAndDelete creates a grant, vests, verifies the vest, then deletes the vest.
func setupVestAndDelete(t *testing.T, ctx context.Context, store *Store,
	unvestedID, investmentID, instrumentID uint,
) {
	t.Helper()
	vestID, _ := setupCreateStockVest(t, ctx, store, unvestedID, investmentID, instrumentID)

	srcLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
	if err != nil {
		t.Fatalf("ListLots(source) before delete: %v", err)
	}
	srcTotal := 0.0
	for _, l := range srcLots {
		srcTotal += l.Quantity
	}
	if srcTotal != 40 {
		t.Fatalf("source lots before delete: got %v, want 40", srcTotal)
	}

	if err := store.DeleteTransaction(ctx, vestID); err != nil {
		t.Fatalf("DeleteTransaction: %v", err)
	}
}

func verifyDeleteVestSourceLots(t *testing.T, ctx context.Context, store *Store,
	unvestedID, instrumentID uint,
) {
	t.Helper()
	srcLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
	if err != nil {
		t.Fatalf("ListLots(source) after delete: %v", err)
	}
	if len(srcLots) != 1 {
		t.Fatalf("expected 1 source lot after delete, got %d", len(srcLots))
	}
	if srcLots[0].Quantity != 100 {
		t.Errorf("source lot Quantity after delete: got %v, want 100", srcLots[0].Quantity)
	}
	if srcLots[0].CostPerShare != 50.0 {
		t.Errorf("source lot CostPerShare after delete: got %v, want 50", srcLots[0].CostPerShare)
	}
	if srcLots[0].Status != LotOpen {
		t.Errorf("source lot Status after delete: got %v, want %v (Open)", srcLots[0].Status, LotOpen)
	}
}

func verifyDeleteVestPositions(t *testing.T, ctx context.Context, store *Store,
	unvestedID, investmentID, instrumentID uint,
) {
	t.Helper()
	srcPos, err := store.GetPosition(ctx, unvestedID, instrumentID)
	if err != nil {
		t.Fatalf("GetPosition(source) after delete: %v", err)
	}
	if srcPos.Quantity != 100 {
		t.Errorf("source position Quantity: got %v, want 100", srcPos.Quantity)
	}
	targetPositions, err := store.ListPositions(ctx, ListPositionsOpts{AccountID: investmentID})
	if err != nil {
		t.Fatalf("ListPositions(target) after delete: %v", err)
	}
	for _, p := range targetPositions {
		if p.InstrumentID == instrumentID && p.Quantity != 0 {
			t.Errorf("target position Quantity: got %v, want 0", p.Quantity)
		}
	}
}

func TestDeleteStockVest_RestoresSourceLots(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("storeDeleteVestRestore"))
			unvestedID, investmentID, instrumentID := setupStockGrantTransferTest(t, ctx, store, mktStore)

			setupVestAndDelete(t, ctx, store, unvestedID, investmentID, instrumentID)

			t.Run("verify source lots restored", func(t *testing.T) {
				verifyDeleteVestSourceLots(t, ctx, store, unvestedID, instrumentID)
			})
			t.Run("verify target lots removed", func(t *testing.T) {
				targetLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: investmentID, InstrumentID: instrumentID})
				if err != nil {
					t.Fatalf("ListLots(target) after delete: %v", err)
				}
				if len(targetLots) != 0 {
					t.Errorf("expected 0 target lots after delete, got %d", len(targetLots))
				}
			})
			t.Run("verify positions", func(t *testing.T) {
				verifyDeleteVestPositions(t, ctx, store, unvestedID, investmentID, instrumentID)
			})
		})
	}
}

func TestUpdateStockVest_BlockedWhenSharesSold(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("storeUpdateVestBlocked"))
			unvestedID, investmentID, instrumentID := setupStockGrantTransferTest(t, ctx, store, mktStore)

			// We also need a cash account for the sell
			providerID, err := store.CreateAccountProvider(ctx, AccountProvider{Name: "Cash Broker"})
			if err != nil {
				t.Fatalf("CreateAccountProvider: %v", err)
			}
			cashID, err := store.CreateAccount(ctx, Account{
				AccountProviderID: providerID,
				Name:              "Cash",
				Currency:          currency.USD,
				Type:              CheckinAccountType,
			})
			if err != nil {
				t.Fatalf("CreateAccount(cash): %v", err)
			}

			// 1. Create grant (100 shares, FMV $50) in unvested account
			_, err = store.CreateStockGrant(ctx, StockGrant{
				Description:     "RSU Grant 100 shares",
				Date:            getDate("2025-01-15"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			// 2. List lots to get lot ID
			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			if len(lots) != 1 {
				t.Fatalf("expected 1 lot in unvested account, got %d", len(lots))
			}
			lotID := lots[0].Id

			// 3. Create income category
			categoryID, err := store.CreateCategory(ctx, CategoryData{Name: "RSU Income", Type: IncomeCategory}, 0)
			if err != nil {
				t.Fatalf("CreateCategory: %v", err)
			}

			// 4. Vest 60 shares at $75
			vest := StockVest{
				Description:     "Vest 60 shares",
				Date:            getDate("2025-06-15"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    75.0,
				CategoryID:      categoryID,
				LotSelections: []LotSelection{
					{LotID: lotID, Quantity: 60},
				},
			}
			vestID, err := store.CreateStockVest(ctx, vest)
			if err != nil {
				t.Fatalf("CreateStockVest: %v", err)
			}

			// 5. Sell 20 of the vested shares
			_, err = store.CreateStockSell(ctx, StockSell{
				Description:         "Sell 20 vested shares",
				Date:                getDate("2025-07-01"),
				InvestmentAccountID: investmentID,
				CashAccountID:       cashID,
				InstrumentID:        instrumentID,
				Quantity:            20,
				TotalAmount:         2000, // $100/share
			})
			if err != nil {
				t.Fatalf("CreateStockSell: %v", err)
			}

			// 6. Try to UpdateStockVest — should return an error
			newPrice := 80.0
			err = store.UpdateStockVest(ctx, StockVestUpdate{
				VestingPrice: &newPrice,
			}, vestID)
			if err == nil {
				t.Fatal("expected error when updating vest with sold shares, got nil")
			}
			var validationErr ErrValidation
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected ErrValidation, got %T: %v", err, err)
			}
			if !strings.Contains(string(validationErr), "some vested shares have been sold") {
				t.Errorf("unexpected error message: %q", validationErr)
			}

			// 7. Verify all lots remain unchanged
			// Source: should still be 40
			srcLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(source): %v", err)
			}
			srcTotal := 0.0
			for _, l := range srcLots {
				srcTotal += l.Quantity
			}
			if srcTotal != 40 {
				t.Errorf("source lots after blocked update: got %v, want 40", srcTotal)
			}

			// Target: should still be 40 (60 vested - 20 sold)
			targetLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: investmentID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(target): %v", err)
			}
			targetTotal := 0.0
			for _, l := range targetLots {
				targetTotal += l.Quantity
			}
			if targetTotal != 40 {
				t.Errorf("target lots after blocked update: got %v, want 40", targetTotal)
			}
		})
	}
}

func TestUpdateStockVest_RejectsInvalidMerge(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("vestUpdateInvalid"))
			unvestedID, investmentID, instrumentID := setupStockGrantTransferTest(t, ctx, store, mktStore)

			// Create a grant (100 shares, FMV $50) in the unvested account.
			_, err := store.CreateStockGrant(ctx, StockGrant{
				Description:     "RSU Grant 100 shares",
				Date:            getDate("2025-01-15"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			// List lots to get the lot ID.
			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			if len(lots) != 1 {
				t.Fatalf("expected 1 lot, got %d", len(lots))
			}
			lotID := lots[0].Id

			// Create an income category.
			categoryID, err := store.CreateCategory(ctx, CategoryData{Name: "RSU Income", Type: IncomeCategory}, 0)
			if err != nil {
				t.Fatalf("CreateCategory: %v", err)
			}

			// Create a valid vest.
			vestID, err := store.CreateStockVest(ctx, StockVest{
				Description:     "Vest 60 shares",
				Date:            getDate("2025-06-15"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    75.0,
				CategoryID:      categoryID,
				LotSelections:   []LotSelection{{LotID: lotID, Quantity: 60}},
			})
			if err != nil {
				t.Fatalf("CreateStockVest: %v", err)
			}

			// Try to update with VestingPrice = 0 — should be rejected.
			zeroPrice := 0.0
			err = store.UpdateStockVest(ctx, StockVestUpdate{
				VestingPrice: &zeroPrice,
			}, vestID)
			if err == nil {
				t.Fatal("expected validation error for VestingPrice=0, got nil")
			}
			var validationErr ErrValidation
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected ErrValidation for VestingPrice=0, got %T: %v", err, err)
			}
			if !strings.Contains(string(validationErr), "vesting price") {
				t.Errorf("unexpected error message for VestingPrice=0: %q", validationErr)
			}

			// Try to update with SourceAccountID = TargetAccountID — should be rejected.
			err = store.UpdateStockVest(ctx, StockVestUpdate{
				SourceAccountID: &investmentID,
			}, vestID)
			if err == nil {
				t.Fatal("expected validation error for same source/target, got nil")
			}
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected ErrValidation for same source/target, got %T: %v", err, err)
			}
			if !strings.Contains(string(validationErr), "source and target must be different") {
				t.Errorf("unexpected error message for same source/target: %q", validationErr)
			}
		})
	}
}

func TestCreateStockVest_RejectsZeroQuantity(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("vestZeroQty"))
			unvestedID, investmentID, instrumentID := setupStockGrantTransferTest(t, ctx, store, mktStore)

			// Create a grant (100 shares, FMV $50) in the unvested account.
			_, err := store.CreateStockGrant(ctx, StockGrant{
				Description:     "RSU Grant 100 shares",
				Date:            getDate("2025-01-15"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			// List lots to get the lot ID.
			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			if len(lots) != 1 {
				t.Fatalf("expected 1 lot, got %d", len(lots))
			}
			lotID := lots[0].Id

			// Create an income category.
			categoryID, err := store.CreateCategory(ctx, CategoryData{Name: "RSU Income", Type: IncomeCategory}, 0)
			if err != nil {
				t.Fatalf("CreateCategory: %v", err)
			}

			// Try to create a vest with a LotSelection of quantity 0.
			_, err = store.CreateStockVest(ctx, StockVest{
				Description:     "Zero quantity vest",
				Date:            getDate("2025-06-15"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    75.0,
				CategoryID:      categoryID,
				LotSelections:   []LotSelection{{LotID: lotID, Quantity: 0}},
			})
			if err == nil {
				t.Fatal("expected validation error for zero quantity, got nil")
			}
			var validationErr ErrValidation
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected ErrValidation for zero quantity, got %T: %v", err, err)
			}
			if !strings.Contains(string(validationErr), "total vesting quantity must be greater than 0") {
				t.Errorf("unexpected error message: %q", validationErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RSU Lifecycle Edge-Case Tests
// ---------------------------------------------------------------------------

// rsuTestSetup is a helper that creates unvested + investment + cash accounts,
// an instrument, and an income category. It returns all IDs needed for RSU tests.
func rsuTestSetup(t *testing.T, ctx context.Context, store *Store, mktStore *marketdata.Store) (
	unvestedID, investmentID, cashID, instrumentID, categoryID uint,
) {
	t.Helper()
	unvestedID, investmentID, instrumentID = setupStockGrantTransferTest(t, ctx, store, mktStore)

	// Create a cash account for sells
	providerID, err := store.CreateAccountProvider(ctx, AccountProvider{Name: "Cash Provider"})
	if err != nil {
		t.Fatal(err)
	}
	cashID, err = store.CreateAccount(ctx, Account{
		AccountProviderID: providerID,
		Name:              "Checking",
		Currency:          currency.USD,
		Type:              CheckinAccountType,
	})
	if err != nil {
		t.Fatal(err)
	}

	categoryID, err = store.CreateCategory(ctx, CategoryData{Name: "RSU Income", Type: IncomeCategory}, 0)
	if err != nil {
		t.Fatal(err)
	}
	return
}

// sumLotQuantity returns the total quantity across all lots returned by ListLots.
func sumLotQuantity(t *testing.T, lots []Lot) float64 {
	t.Helper()
	total := 0.0
	for _, l := range lots {
		total += l.Quantity
	}
	return total
}

// TestRSU_Scenario1_DeleteGrantAfterVest verifies that deleting a grant
// that has already been partially vested is blocked to prevent orphaned
// vested lots in the investment account.
func TestRSU_Scenario1_DeleteGrantAfterVest(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("rsuScenario1"))
			unvestedID, investmentID, _, instrumentID, categoryID := rsuTestSetup(t, ctx, store, mktStore)

			// Grant 100 shares at FMV $50
			grantID, err := store.CreateStockGrant(ctx, StockGrant{
				Description:     "RSU Grant 100",
				Date:            getDate("2025-01-01"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			// Get the lot ID for vesting
			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			lotID := lots[0].Id

			// Vest 60 shares at $75
			_, err = store.CreateStockVest(ctx, StockVest{
				Description:     "Vest 60",
				Date:            getDate("2025-06-01"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    75.0,
				CategoryID:      categoryID,
				LotSelections:   []LotSelection{{LotID: lotID, Quantity: 60}},
			})
			if err != nil {
				t.Fatalf("CreateStockVest: %v", err)
			}

			// Delete the grant -- should be blocked because lots have been vested
			err = store.DeleteTransaction(ctx, grantID)
			if err == nil {
				t.Fatal("expected error when deleting grant with vested lots, got nil")
			}

			// Verify data is still consistent: unvested=40, investment=60
			srcLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(source): %v", err)
			}
			if got := sumLotQuantity(t, srcLots); got != 40 {
				t.Errorf("source lots after blocked delete: got %v, want 40", got)
			}

			tgtLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: investmentID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(target): %v", err)
			}
			if got := sumLotQuantity(t, tgtLots); got != 60 {
				t.Errorf("target lots after blocked delete: got %v, want 60", got)
			}
		})
	}
}

// TestRSU_Scenario2_UpdateGrantAfterVest verifies that editing a grant's
// quantity below the already-vested amount is blocked.
func TestRSU_Scenario2_UpdateGrantAfterVest(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("rsuScenario2"))
			unvestedID, investmentID, _, instrumentID, categoryID := rsuTestSetup(t, ctx, store, mktStore)

			// Grant 100 shares
			grantID, err := store.CreateStockGrant(ctx, StockGrant{
				Description:     "RSU Grant 100",
				Date:            getDate("2025-01-01"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			lotID := lots[0].Id

			// Vest 60 shares
			_, err = store.CreateStockVest(ctx, StockVest{
				Description:     "Vest 60",
				Date:            getDate("2025-06-01"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    75.0,
				CategoryID:      categoryID,
				LotSelections:   []LotSelection{{LotID: lotID, Quantity: 60}},
			})
			if err != nil {
				t.Fatalf("CreateStockVest: %v", err)
			}

			// Try to reduce grant to 50 shares -- should fail since 60 already vested
			newQty := 50.0
			err = store.UpdateStockGrant(ctx, StockGrantUpdate{Quantity: &newQty}, grantID)
			if err == nil {
				t.Fatal("expected error when reducing grant below vested amount, got nil")
			}

			// Verify data is still consistent
			srcLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(source): %v", err)
			}
			if got := sumLotQuantity(t, srcLots); got != 40 {
				t.Errorf("source lots after blocked update: got %v, want 40", got)
			}
			tgtLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: investmentID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(target): %v", err)
			}
			if got := sumLotQuantity(t, tgtLots); got != 60 {
				t.Errorf("target lots after blocked update: got %v, want 60", got)
			}
		})
	}
}

// TestRSU_Scenario3_DeleteVestThenGrant verifies the full cleanup:
// grant 100, vest all 100, delete vest, delete grant -> all lots gone.
func TestRSU_Scenario3_DeleteVestThenGrant(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("rsuScenario3"))
			unvestedID, investmentID, _, instrumentID, categoryID := rsuTestSetup(t, ctx, store, mktStore)

			grantID, err := store.CreateStockGrant(ctx, StockGrant{
				Description:     "RSU Grant 100",
				Date:            getDate("2025-01-01"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			lotID := lots[0].Id

			vestID, err := store.CreateStockVest(ctx, StockVest{
				Description:     "Vest 100",
				Date:            getDate("2025-06-01"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    75.0,
				CategoryID:      categoryID,
				LotSelections:   []LotSelection{{LotID: lotID, Quantity: 100}},
			})
			if err != nil {
				t.Fatalf("CreateStockVest: %v", err)
			}

			// Delete the vest
			if err := store.DeleteTransaction(ctx, vestID); err != nil {
				t.Fatalf("DeleteTransaction(vest): %v", err)
			}

			// Source lot should be back to 100
			srcLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(source) after vest delete: %v", err)
			}
			if len(srcLots) != 1 || srcLots[0].Quantity != 100 {
				t.Fatalf("source lots after vest delete: got qty=%v, want 100", sumLotQuantity(t, srcLots))
			}

			// No target lots
			tgtLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: investmentID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(target) after vest delete: %v", err)
			}
			if len(tgtLots) != 0 {
				t.Errorf("expected 0 target lots, got %d", len(tgtLots))
			}

			// Now delete the grant
			if err := store.DeleteTransaction(ctx, grantID); err != nil {
				t.Fatalf("DeleteTransaction(grant): %v", err)
			}

			// No lots anywhere
			allSrcLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(source) after grant delete: %v", err)
			}
			if len(allSrcLots) != 0 {
				t.Errorf("expected 0 source lots after full cleanup, got %d", len(allSrcLots))
			}

			// Positions should be zero
			srcPositions, err := store.ListPositions(ctx, ListPositionsOpts{AccountID: unvestedID})
			if err != nil {
				t.Fatalf("ListPositions(source): %v", err)
			}
			for _, p := range srcPositions {
				if p.InstrumentID == instrumentID && p.Quantity != 0 {
					t.Errorf("source position qty: got %v, want 0", p.Quantity)
				}
			}
		})
	}
}

// TestRSU_Scenario4_DeleteGrantAfterVestAndSell verifies that deleting a grant
// after vest and sell is blocked because it would orphan both vested and sold lots.
func TestRSU_Scenario4_DeleteGrantAfterVestAndSell(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("rsuScenario4"))
			unvestedID, investmentID, cashID, instrumentID, categoryID := rsuTestSetup(t, ctx, store, mktStore)

			// Grant 100 shares
			grantID, err := store.CreateStockGrant(ctx, StockGrant{
				Description:     "RSU Grant 100",
				Date:            getDate("2025-01-01"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			lotID := lots[0].Id

			// Vest 60 shares
			_, err = store.CreateStockVest(ctx, StockVest{
				Description:     "Vest 60",
				Date:            getDate("2025-06-01"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    75.0,
				CategoryID:      categoryID,
				LotSelections:   []LotSelection{{LotID: lotID, Quantity: 60}},
			})
			if err != nil {
				t.Fatalf("CreateStockVest: %v", err)
			}

			// Sell 30 shares
			_, err = store.CreateStockSell(ctx, StockSell{
				Description:         "Sell 30",
				Date:                getDate("2025-07-01"),
				InvestmentAccountID: investmentID,
				CashAccountID:       cashID,
				InstrumentID:        instrumentID,
				Quantity:            30,
				TotalAmount:         3000,
			})
			if err != nil {
				t.Fatalf("CreateStockSell: %v", err)
			}

			// Delete the grant -- should be blocked
			err = store.DeleteTransaction(ctx, grantID)
			if err == nil {
				t.Fatal("expected error when deleting grant with vested+sold lots, got nil")
			}

			// Verify data is still consistent
			srcLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(source): %v", err)
			}
			if got := sumLotQuantity(t, srcLots); got != 40 {
				t.Errorf("source lots: got %v, want 40", got)
			}

			tgtLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: investmentID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(target): %v", err)
			}
			if got := sumLotQuantity(t, tgtLots); got != 30 {
				t.Errorf("target lots: got %v, want 30 (60 vested - 30 sold)", got)
			}
		})
	}
}

// TestRSU_Scenario5_DeleteSellThenVest verifies the full restoration chain:
// grant 100, vest 60, sell 30, delete sell, delete vest -> source lot = 100.
func TestRSU_Scenario5_DeleteSellThenVest(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("rsuScenario5"))
			unvestedID, investmentID, cashID, instrumentID, categoryID := rsuTestSetup(t, ctx, store, mktStore)

			// Grant 100
			_, err := store.CreateStockGrant(ctx, StockGrant{
				Description:     "RSU Grant 100",
				Date:            getDate("2025-01-01"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			lotID := lots[0].Id

			// Vest 60
			vestID, err := store.CreateStockVest(ctx, StockVest{
				Description:     "Vest 60",
				Date:            getDate("2025-06-01"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    75.0,
				CategoryID:      categoryID,
				LotSelections:   []LotSelection{{LotID: lotID, Quantity: 60}},
			})
			if err != nil {
				t.Fatalf("CreateStockVest: %v", err)
			}

			// Sell 30
			sellID, err := store.CreateStockSell(ctx, StockSell{
				Description:         "Sell 30",
				Date:                getDate("2025-07-01"),
				InvestmentAccountID: investmentID,
				CashAccountID:       cashID,
				InstrumentID:        instrumentID,
				Quantity:            30,
				TotalAmount:         3000,
			})
			if err != nil {
				t.Fatalf("CreateStockSell: %v", err)
			}

			// Delete the sell
			if err := store.DeleteTransaction(ctx, sellID); err != nil {
				t.Fatalf("DeleteTransaction(sell): %v", err)
			}

			// After sell delete: target lots should be back to 60
			tgtLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: investmentID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(target) after sell delete: %v", err)
			}
			if got := sumLotQuantity(t, tgtLots); got != 60 {
				t.Errorf("target lots after sell delete: got %v, want 60", got)
			}

			// Delete the vest
			if err := store.DeleteTransaction(ctx, vestID); err != nil {
				t.Fatalf("DeleteTransaction(vest): %v", err)
			}

			// After vest delete: source should be 100, no target lots
			srcLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(source) after vest delete: %v", err)
			}
			if got := sumLotQuantity(t, srcLots); got != 100 {
				t.Errorf("source lots after vest delete: got %v, want 100", got)
			}

			tgtLots, err = store.ListLots(ctx, ListLotsOpts{AccountID: investmentID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(target) after vest delete: %v", err)
			}
			if len(tgtLots) != 0 {
				t.Errorf("expected 0 target lots after vest delete, got %d (qty=%v)", len(tgtLots), sumLotQuantity(t, tgtLots))
			}

			// Source position should be 100
			srcPos, err := store.GetPosition(ctx, unvestedID, instrumentID)
			if err != nil {
				t.Fatalf("GetPosition(source): %v", err)
			}
			if srcPos.Quantity != 100 {
				t.Errorf("source position: got %v, want 100", srcPos.Quantity)
			}
		})
	}
}

// TestRSU_Scenario7_MultiGrantVest verifies that vesting from multiple grants
// handles lot selection correctly.
func TestRSU_Scenario7_MultiGrantVest(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("rsuScenario7"))
			unvestedID, investmentID, _, instrumentID, categoryID := rsuTestSetup(t, ctx, store, mktStore)

			// Grant A: 50 shares
			_, err := store.CreateStockGrant(ctx, StockGrant{
				Description:     "Grant A",
				Date:            getDate("2025-01-01"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        50,
				FairMarketValue: 40.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant A: %v", err)
			}

			// Grant B: 50 shares
			_, err = store.CreateStockGrant(ctx, StockGrant{
				Description:     "Grant B",
				Date:            getDate("2025-02-01"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        50,
				FairMarketValue: 45.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant B: %v", err)
			}

			// Get lot IDs
			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			if len(lots) != 2 {
				t.Fatalf("expected 2 lots, got %d", len(lots))
			}
			// lots are ordered by open_date ASC
			lotAID := lots[0].Id
			lotBID := lots[1].Id

			// Vest 70 shares from both grants: 50 from A, 20 from B
			_, err = store.CreateStockVest(ctx, StockVest{
				Description:     "Vest 70",
				Date:            getDate("2025-06-01"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    75.0,
				CategoryID:      categoryID,
				LotSelections: []LotSelection{
					{LotID: lotAID, Quantity: 50},
					{LotID: lotBID, Quantity: 20},
				},
			})
			if err != nil {
				t.Fatalf("CreateStockVest: %v", err)
			}

			// Verify source: Grant A lot should be closed (0), Grant B lot should have 30
			srcLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(source): %v", err)
			}
			totalSrc := sumLotQuantity(t, srcLots)
			if totalSrc != 30 {
				t.Errorf("source lots total: got %v, want 30", totalSrc)
			}

			// Verify target: should have 70 shares at $75/share
			tgtLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: investmentID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(target): %v", err)
			}
			totalTgt := sumLotQuantity(t, tgtLots)
			if totalTgt != 70 {
				t.Errorf("target lots total: got %v, want 70", totalTgt)
			}
			// All target lots should have CostPerShare = 75
			for _, l := range tgtLots {
				if l.CostPerShare != 75.0 {
					t.Errorf("target lot %d CostPerShare: got %v, want 75", l.Id, l.CostPerShare)
				}
			}

			// Verify positions
			srcPos, err := store.GetPosition(ctx, unvestedID, instrumentID)
			if err != nil {
				t.Fatalf("GetPosition(source): %v", err)
			}
			if srcPos.Quantity != 30 {
				t.Errorf("source position: got %v, want 30", srcPos.Quantity)
			}
			tgtPos, err := store.GetPosition(ctx, investmentID, instrumentID)
			if err != nil {
				t.Fatalf("GetPosition(target): %v", err)
			}
			if tgtPos.Quantity != 70 {
				t.Errorf("target position: got %v, want 70", tgtPos.Quantity)
			}
		})
	}
}

// TestRSU_Scenario8_DeleteVestFromMultiGrant verifies that deleting a vest
// that drew from multiple grants correctly restores BOTH source grants.
func TestRSU_Scenario8_DeleteVestFromMultiGrant(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("rsuScenario8"))
			unvestedID, investmentID, _, instrumentID, categoryID := rsuTestSetup(t, ctx, store, mktStore)

			// Grant A: 50 shares
			_, err := store.CreateStockGrant(ctx, StockGrant{
				Description:     "Grant A",
				Date:            getDate("2025-01-01"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        50,
				FairMarketValue: 40.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant A: %v", err)
			}

			// Grant B: 50 shares
			_, err = store.CreateStockGrant(ctx, StockGrant{
				Description:     "Grant B",
				Date:            getDate("2025-02-01"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        50,
				FairMarketValue: 45.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant B: %v", err)
			}

			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			lotAID := lots[0].Id
			lotBID := lots[1].Id

			// Vest 70 from both
			vestID, err := store.CreateStockVest(ctx, StockVest{
				Description:     "Vest 70",
				Date:            getDate("2025-06-01"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    75.0,
				CategoryID:      categoryID,
				LotSelections: []LotSelection{
					{LotID: lotAID, Quantity: 50},
					{LotID: lotBID, Quantity: 20},
				},
			})
			if err != nil {
				t.Fatalf("CreateStockVest: %v", err)
			}

			// Delete the vest
			if err := store.DeleteTransaction(ctx, vestID); err != nil {
				t.Fatalf("DeleteTransaction(vest): %v", err)
			}

			// Source: both lots should be fully restored
			srcLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(source) after delete: %v", err)
			}
			// Sort by open_date to get A first
			sort.Slice(srcLots, func(i, j int) bool { return srcLots[i].OpenDate.Before(srcLots[j].OpenDate) })

			if len(srcLots) < 2 {
				t.Fatalf("expected at least 2 source lots, got %d", len(srcLots))
			}

			// Grant A lot should be 50
			if srcLots[0].Quantity != 50 {
				t.Errorf("Grant A lot after delete: got %v, want 50", srcLots[0].Quantity)
			}
			if srcLots[0].Status != LotOpen {
				t.Errorf("Grant A lot status: got %v, want Open", srcLots[0].Status)
			}

			// Grant B lot should be 50
			if srcLots[1].Quantity != 50 {
				t.Errorf("Grant B lot after delete: got %v, want 50", srcLots[1].Quantity)
			}
			if srcLots[1].Status != LotOpen {
				t.Errorf("Grant B lot status: got %v, want Open", srcLots[1].Status)
			}

			// No target lots
			tgtLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: investmentID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(target) after delete: %v", err)
			}
			if len(tgtLots) != 0 {
				t.Errorf("expected 0 target lots, got %d", len(tgtLots))
			}
		})
	}
}

// TestRSU_Scenario10_DeleteVestAfterSell verifies that deleting a vest
// when some vested shares have been sold is blocked.
func TestRSU_Scenario10_DeleteVestAfterSell(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("rsuScenario10"))
			unvestedID, investmentID, cashID, instrumentID, categoryID := rsuTestSetup(t, ctx, store, mktStore)

			// Grant 100
			_, err := store.CreateStockGrant(ctx, StockGrant{
				Description:     "RSU Grant 100",
				Date:            getDate("2025-01-01"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			lotID := lots[0].Id

			// Vest 60
			vestID, err := store.CreateStockVest(ctx, StockVest{
				Description:     "Vest 60",
				Date:            getDate("2025-06-01"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    75.0,
				CategoryID:      categoryID,
				LotSelections:   []LotSelection{{LotID: lotID, Quantity: 60}},
			})
			if err != nil {
				t.Fatalf("CreateStockVest: %v", err)
			}

			// Sell 30
			_, err = store.CreateStockSell(ctx, StockSell{
				Description:         "Sell 30",
				Date:                getDate("2025-07-01"),
				InvestmentAccountID: investmentID,
				CashAccountID:       cashID,
				InstrumentID:        instrumentID,
				Quantity:            30,
				TotalAmount:         3000,
			})
			if err != nil {
				t.Fatalf("CreateStockSell: %v", err)
			}

			// Delete the vest -- should be blocked because some vested shares were sold
			err = store.DeleteTransaction(ctx, vestID)
			if err == nil {
				t.Fatal("expected error when deleting vest with sold shares, got nil")
			}

			// Verify data is still consistent: source=40, target=30 (60-30 sold)
			srcLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(source): %v", err)
			}
			if got := sumLotQuantity(t, srcLots); got != 40 {
				t.Errorf("source lots: got %v, want 40", got)
			}

			tgtLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: investmentID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(target): %v", err)
			}
			if got := sumLotQuantity(t, tgtLots); got != 30 {
				t.Errorf("target lots: got %v, want 30", got)
			}
		})
	}
}

// TestRSU_Scenario11_UpdateVestPrice verifies that updating the vesting price
// also updates the income entry amount.
func TestRSU_Scenario11_UpdateVestPrice(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("rsuScenario11"))
			unvestedID, investmentID, _, instrumentID, categoryID := rsuTestSetup(t, ctx, store, mktStore)

			// Grant 100 shares
			_, err := store.CreateStockGrant(ctx, StockGrant{
				Description:     "RSU Grant 100",
				Date:            getDate("2025-01-01"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			lotID := lots[0].Id

			// Vest 60 at $75 => income = 60 * 75 = $4500
			vestID, err := store.CreateStockVest(ctx, StockVest{
				Description:     "Vest 60",
				Date:            getDate("2025-06-01"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    75.0,
				CategoryID:      categoryID,
				LotSelections:   []LotSelection{{LotID: lotID, Quantity: 60}},
			})
			if err != nil {
				t.Fatalf("CreateStockVest: %v", err)
			}

			// Verify income entry before update
			var entriesBefore []dbEntry
			if err := store.db.Where("transaction_id = ? AND entry_type = ?", vestID, stockVestIncomeEntry).Find(&entriesBefore).Error; err != nil {
				t.Fatalf("query entries: %v", err)
			}
			if len(entriesBefore) != 1 {
				t.Fatalf("expected 1 income entry, got %d", len(entriesBefore))
			}
			if entriesBefore[0].Amount != 4500 {
				t.Errorf("income entry before update: got %v, want 4500", entriesBefore[0].Amount)
			}

			// Update vesting price to $80 => income should be 60 * 80 = $4800
			newPrice := 80.0
			if err := store.UpdateStockVest(ctx, StockVestUpdate{VestingPrice: &newPrice}, vestID); err != nil {
				t.Fatalf("UpdateStockVest: %v", err)
			}

			// Verify income entry after update
			var entriesAfter []dbEntry
			if err := store.db.Where("transaction_id = ? AND entry_type = ?", vestID, stockVestIncomeEntry).Find(&entriesAfter).Error; err != nil {
				t.Fatalf("query entries after update: %v", err)
			}
			if len(entriesAfter) != 1 {
				t.Fatalf("expected 1 income entry after update, got %d", len(entriesAfter))
			}
			if entriesAfter[0].Amount != 4800 {
				t.Errorf("income entry after update: got %v, want 4800", entriesAfter[0].Amount)
			}

			// Verify target lot cost per share updated
			tgtLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: investmentID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(target): %v", err)
			}
			if len(tgtLots) != 1 {
				t.Fatalf("expected 1 target lot, got %d", len(tgtLots))
			}
			if tgtLots[0].CostPerShare != 80 {
				t.Errorf("target lot CostPerShare: got %v, want 80", tgtLots[0].CostPerShare)
			}
		})
	}
}

// TestRSU_Scenario12_DeleteVestCleansIncomeEntry verifies that deleting a vest
// also removes the income entry.
func TestRSU_Scenario12_DeleteVestCleansIncomeEntry(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("rsuScenario12"))
			unvestedID, investmentID, _, instrumentID, categoryID := rsuTestSetup(t, ctx, store, mktStore)

			_, err := store.CreateStockGrant(ctx, StockGrant{
				Description:     "RSU Grant 100",
				Date:            getDate("2025-01-01"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			lotID := lots[0].Id

			vestID, err := store.CreateStockVest(ctx, StockVest{
				Description:     "Vest 60",
				Date:            getDate("2025-06-01"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    75.0,
				CategoryID:      categoryID,
				LotSelections:   []LotSelection{{LotID: lotID, Quantity: 60}},
			})
			if err != nil {
				t.Fatalf("CreateStockVest: %v", err)
			}

			// Delete the vest
			if err := store.DeleteTransaction(ctx, vestID); err != nil {
				t.Fatalf("DeleteTransaction(vest): %v", err)
			}

			// Verify no income entries remain for this transaction
			var entries []dbEntry
			if err := store.db.Where("transaction_id = ?", vestID).Find(&entries).Error; err != nil {
				t.Fatalf("query entries: %v", err)
			}
			if len(entries) != 0 {
				t.Errorf("expected 0 entries after vest delete, got %d", len(entries))
			}

			// Verify the transaction itself is deleted
			_, err = store.GetTransaction(ctx, vestID)
			if err == nil {
				t.Error("expected transaction not found, got nil error")
			}
		})
	}
}

// TestRSU_Scenario13_TwoVestsFromSameGrant verifies that two sequential vests
// from the same grant lot correctly see the reduced quantity.
func TestRSU_Scenario13_TwoVestsFromSameGrant(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("rsuScenario13"))
			unvestedID, investmentID, _, instrumentID, categoryID := rsuTestSetup(t, ctx, store, mktStore)

			// Grant 100 shares
			_, err := store.CreateStockGrant(ctx, StockGrant{
				Description:     "RSU Grant 100",
				Date:            getDate("2025-01-01"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			lotID := lots[0].Id

			// Vest 30 shares at $70
			_, err = store.CreateStockVest(ctx, StockVest{
				Description:     "Vest 30",
				Date:            getDate("2025-06-01"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    70.0,
				CategoryID:      categoryID,
				LotSelections:   []LotSelection{{LotID: lotID, Quantity: 30}},
			})
			if err != nil {
				t.Fatalf("CreateStockVest(1): %v", err)
			}

			// Vest 40 more shares at $80 (from same lot, now has 70 remaining)
			_, err = store.CreateStockVest(ctx, StockVest{
				Description:     "Vest 40",
				Date:            getDate("2025-09-01"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    80.0,
				CategoryID:      categoryID,
				LotSelections:   []LotSelection{{LotID: lotID, Quantity: 40}},
			})
			if err != nil {
				t.Fatalf("CreateStockVest(2): %v", err)
			}

			// Source should have 30 remaining (100 - 30 - 40)
			srcLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(source): %v", err)
			}
			if got := sumLotQuantity(t, srcLots); got != 30 {
				t.Errorf("source lots: got %v, want 30", got)
			}

			// Target should have 70 total (30 + 40) in 2 lots
			tgtLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: investmentID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots(target): %v", err)
			}
			if got := sumLotQuantity(t, tgtLots); got != 70 {
				t.Errorf("target lots: got %v, want 70", got)
			}
			if len(tgtLots) != 2 {
				t.Errorf("expected 2 target lots (one per vest), got %d", len(tgtLots))
			}

			// Verify different cost bases
			sort.Slice(tgtLots, func(i, j int) bool { return tgtLots[i].Quantity < tgtLots[j].Quantity })
			if tgtLots[0].CostPerShare != 70.0 || tgtLots[0].Quantity != 30 {
				t.Errorf("first vest lot: CostPerShare=%v Qty=%v, want 70/30", tgtLots[0].CostPerShare, tgtLots[0].Quantity)
			}
			if tgtLots[1].CostPerShare != 80.0 || tgtLots[1].Quantity != 40 {
				t.Errorf("second vest lot: CostPerShare=%v Qty=%v, want 80/40", tgtLots[1].CostPerShare, tgtLots[1].Quantity)
			}

			// Trying to vest 40 more should fail (only 30 remaining)
			_, err = store.CreateStockVest(ctx, StockVest{
				Description:     "Vest too many",
				Date:            getDate("2025-12-01"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    90.0,
				CategoryID:      categoryID,
				LotSelections:   []LotSelection{{LotID: lotID, Quantity: 40}},
			})
			if err == nil {
				t.Fatal("expected error when vesting more than available, got nil")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Second pass RSU tests (TestRSU2_*)
// ---------------------------------------------------------------------------

// TestRSU2_DeleteStockTransferRestoresSourceLots verifies that deleting a
// StockTransfer restores the consumed source lots (bug: previously source
// lots were never restored on transfer deletion).
func TestRSU2_DeleteStockTransferRestoresSourceLots(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("rsu2DeleteTransfer"))

			// Create two investment accounts for transfer.
			providerID, err := store.CreateAccountProvider(ctx, AccountProvider{Name: "Broker"})
			if err != nil {
				t.Fatal(err)
			}
			invA, err := store.CreateAccount(ctx, Account{AccountProviderID: providerID, Name: "Investment A", Currency: currency.USD, Type: InvestmentAccountType})
			if err != nil {
				t.Fatal(err)
			}
			invB, err := store.CreateAccount(ctx, Account{AccountProviderID: providerID, Name: "Investment B", Currency: currency.USD, Type: InvestmentAccountType})
			if err != nil {
				t.Fatal(err)
			}
			instrumentID, err := mktStore.CreateInstrument(ctx, marketdata.Instrument{Symbol: "RSU", Name: "Company RSU", Currency: currency.USD})
			if err != nil {
				t.Fatal(err)
			}

			// Grant 100 shares into investment A
			_, err = store.CreateStockGrant(ctx, StockGrant{
				Description:     "Grant 100",
				Date:            getDate("2025-01-01"),
				AccountID:       invA,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			// Transfer 60 from invA to invB
			transferID, err := store.CreateStockTransfer(ctx, StockTransfer{
				Description:     "Transfer 60",
				Date:            getDate("2025-06-01"),
				SourceAccountID: invA,
				TargetAccountID: invB,
				InstrumentID:    instrumentID,
				Quantity:        60,
			})
			if err != nil {
				t.Fatalf("CreateStockTransfer: %v", err)
			}

			// Verify: invA=40, invB=60
			srcLots, _ := store.ListLots(ctx, ListLotsOpts{AccountID: invA, InstrumentID: instrumentID})
			if got := sumLotQuantity(t, srcLots); got != 40 {
				t.Fatalf("before delete: source lots = %v, want 40", got)
			}
			tgtLots, _ := store.ListLots(ctx, ListLotsOpts{AccountID: invB, InstrumentID: instrumentID})
			if got := sumLotQuantity(t, tgtLots); got != 60 {
				t.Fatalf("before delete: target lots = %v, want 60", got)
			}

			// Delete the transfer
			if err := store.DeleteTransaction(ctx, transferID); err != nil {
				t.Fatalf("DeleteTransaction(transfer): %v", err)
			}

			// Source should be back to 100
			srcLots, _ = store.ListLots(ctx, ListLotsOpts{AccountID: invA, InstrumentID: instrumentID})
			if got := sumLotQuantity(t, srcLots); got != 100 {
				t.Errorf("after delete: source lots = %v, want 100", got)
			}

			// Target should be 0
			tgtLots, _ = store.ListLots(ctx, ListLotsOpts{AccountID: invB, InstrumentID: instrumentID})
			if got := sumLotQuantity(t, tgtLots); got != 0 {
				t.Errorf("after delete: target lots = %v, want 0", got)
			}

			// Positions should match lots
			srcPos, _ := store.GetPosition(ctx, invA, instrumentID)
			if srcPos.Quantity != 100 {
				t.Errorf("after delete: source position = %v, want 100", srcPos.Quantity)
			}
		})
	}
}

// TestRSU2_UpdateStockTransferRestoresSourceLots verifies that updating a
// StockTransfer (e.g. changing quantity) correctly restores source lots
// and reapplies the transfer.
func TestRSU2_UpdateStockTransferRestoresSourceLots(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("rsu2UpdateTransfer"))

			// Create two investment accounts for transfer.
			providerID, err := store.CreateAccountProvider(ctx, AccountProvider{Name: "Broker"})
			if err != nil {
				t.Fatal(err)
			}
			invA, err := store.CreateAccount(ctx, Account{AccountProviderID: providerID, Name: "Investment A", Currency: currency.USD, Type: InvestmentAccountType})
			if err != nil {
				t.Fatal(err)
			}
			invB, err := store.CreateAccount(ctx, Account{AccountProviderID: providerID, Name: "Investment B", Currency: currency.USD, Type: InvestmentAccountType})
			if err != nil {
				t.Fatal(err)
			}
			instrumentID, err := mktStore.CreateInstrument(ctx, marketdata.Instrument{Symbol: "RSU", Name: "Company RSU", Currency: currency.USD})
			if err != nil {
				t.Fatal(err)
			}

			// Grant 100 into investment A
			_, err = store.CreateStockGrant(ctx, StockGrant{
				Description:     "Grant 100",
				Date:            getDate("2025-01-01"),
				AccountID:       invA,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			// Transfer 60 from invA to invB
			transferID, err := store.CreateStockTransfer(ctx, StockTransfer{
				Description:     "Transfer 60",
				Date:            getDate("2025-06-01"),
				SourceAccountID: invA,
				TargetAccountID: invB,
				InstrumentID:    instrumentID,
				Quantity:        60,
			})
			if err != nil {
				t.Fatalf("CreateStockTransfer: %v", err)
			}

			// Update transfer quantity from 60 to 30
			newQty := 30.0
			err = store.UpdateStockTransfer(ctx, StockTransferUpdate{Quantity: &newQty}, transferID)
			if err != nil {
				t.Fatalf("UpdateStockTransfer: %v", err)
			}

			// Source should be 70 (100-30), target should be 30
			srcLots, _ := store.ListLots(ctx, ListLotsOpts{AccountID: invA, InstrumentID: instrumentID})
			if got := sumLotQuantity(t, srcLots); got != 70 {
				t.Errorf("after update: source lots = %v, want 70", got)
			}
			tgtLots, _ := store.ListLots(ctx, ListLotsOpts{AccountID: invB, InstrumentID: instrumentID})
			if got := sumLotQuantity(t, tgtLots); got != 30 {
				t.Errorf("after update: target lots = %v, want 30", got)
			}
		})
	}
}

// TestRSU2_VestCategoryMustBeIncome verifies that creating a vest with an
// expense category is rejected.
func TestRSU2_VestCategoryMustBeIncome(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("rsu2VestCategory"))
			unvestedID, investmentID, _, instrumentID, _ := rsuTestSetup(t, ctx, store, mktStore)

			// Create an expense category
			expenseCatID, err := store.CreateCategory(ctx, CategoryData{Name: "Expense Cat", Type: ExpenseCategory}, 0)
			if err != nil {
				t.Fatalf("CreateCategory: %v", err)
			}

			// Grant 100
			_, err = store.CreateStockGrant(ctx, StockGrant{
				Description:     "Grant 100",
				Date:            getDate("2025-01-01"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			lots, _ := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			lotID := lots[0].Id

			// Try to vest with expense category - should fail
			_, err = store.CreateStockVest(ctx, StockVest{
				Description:     "Vest 50",
				Date:            getDate("2025-06-01"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    75.0,
				CategoryID:      expenseCatID,
				LotSelections:   []LotSelection{{LotID: lotID, Quantity: 50}},
			})
			if err == nil {
				t.Fatal("expected error when vesting with expense category, got nil")
			}
			if !strings.Contains(err.Error(), "income category") {
				t.Errorf("expected income category error, got: %v", err)
			}
		})
	}
}

// TestRSU2_DeleteVestBlockedWhenSharesTransferred verifies that deleting
// a vest is blocked when vested shares have been moved by a downstream
// StockTransfer (not just sells).
func TestRSU2_DeleteVestBlockedWhenSharesTransferred(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("rsu2VestTransferGuard"))
			unvestedID, investmentID, _, instrumentID, categoryID := rsuTestSetup(t, ctx, store, mktStore)

			// Create a second investment account
			providerID, _ := store.CreateAccountProvider(ctx, AccountProvider{Name: "Broker2"})
			investmentID2, err := store.CreateAccount(ctx, Account{
				AccountProviderID: providerID,
				Name:              "Second Investment",
				Currency:          currency.USD,
				Type:              InvestmentAccountType,
			})
			if err != nil {
				t.Fatalf("CreateAccount: %v", err)
			}

			// Grant 100
			_, err = store.CreateStockGrant(ctx, StockGrant{
				Description:     "Grant 100",
				Date:            getDate("2025-01-01"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			lots, _ := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			lotID := lots[0].Id

			// Vest 50 to investment
			vestID, err := store.CreateStockVest(ctx, StockVest{
				Description:     "Vest 50",
				Date:            getDate("2025-06-01"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    75.0,
				CategoryID:      categoryID,
				LotSelections:   []LotSelection{{LotID: lotID, Quantity: 50}},
			})
			if err != nil {
				t.Fatalf("CreateStockVest: %v", err)
			}

			// Transfer 25 from investment to investmentID2
			_, err = store.CreateStockTransfer(ctx, StockTransfer{
				Description:     "Transfer 25",
				Date:            getDate("2025-07-01"),
				SourceAccountID: investmentID,
				TargetAccountID: investmentID2,
				InstrumentID:    instrumentID,
				Quantity:        25,
			})
			if err != nil {
				t.Fatalf("CreateStockTransfer: %v", err)
			}

			// Try to delete the vest - should be blocked because shares were transferred
			err = store.DeleteTransaction(ctx, vestID)
			if err == nil {
				t.Fatal("expected error when deleting vest with transferred shares, got nil")
			}
			if !strings.Contains(err.Error(), "downstream") {
				t.Errorf("expected downstream error, got: %v", err)
			}

			// Verify data is still consistent
			srcLots, _ := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if got := sumLotQuantity(t, srcLots); got != 50 {
				t.Errorf("source lots = %v, want 50", got)
			}
		})
	}
}

// TestRSU2_DeleteTransferBlockedWhenSharesSold verifies that deleting a
// StockTransfer is blocked when transferred lots have been sold.
func TestRSU2_DeleteTransferBlockedWhenSharesSold(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("rsu2TransferSellGuard"))

			// Create two investment accounts + cash account for sells.
			providerID, err := store.CreateAccountProvider(ctx, AccountProvider{Name: "Broker"})
			if err != nil {
				t.Fatal(err)
			}
			invA, err := store.CreateAccount(ctx, Account{AccountProviderID: providerID, Name: "Investment A", Currency: currency.USD, Type: InvestmentAccountType})
			if err != nil {
				t.Fatal(err)
			}
			invB, err := store.CreateAccount(ctx, Account{AccountProviderID: providerID, Name: "Investment B", Currency: currency.USD, Type: InvestmentAccountType})
			if err != nil {
				t.Fatal(err)
			}
			cashID, err := store.CreateAccount(ctx, Account{AccountProviderID: providerID, Name: "Checking", Currency: currency.USD, Type: CheckinAccountType})
			if err != nil {
				t.Fatal(err)
			}
			instrumentID, err := mktStore.CreateInstrument(ctx, marketdata.Instrument{Symbol: "RSU", Name: "Company RSU", Currency: currency.USD})
			if err != nil {
				t.Fatal(err)
			}

			// Grant 100 into investment A
			_, err = store.CreateStockGrant(ctx, StockGrant{
				Description:     "Grant 100",
				Date:            getDate("2025-01-01"),
				AccountID:       invA,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			// Transfer 60 from invA to invB
			transferID, err := store.CreateStockTransfer(ctx, StockTransfer{
				Description:     "Transfer 60",
				Date:            getDate("2025-06-01"),
				SourceAccountID: invA,
				TargetAccountID: invB,
				InstrumentID:    instrumentID,
				Quantity:        60,
			})
			if err != nil {
				t.Fatalf("CreateStockTransfer: %v", err)
			}

			// Sell 20 from invB
			_, err = store.CreateStockSell(ctx, StockSell{
				Description:         "Sell 20",
				Date:                getDate("2025-07-01"),
				InvestmentAccountID: invB,
				CashAccountID:       cashID,
				InstrumentID:        instrumentID,
				Quantity:            20,
				TotalAmount:         2000,
			})
			if err != nil {
				t.Fatalf("CreateStockSell: %v", err)
			}

			// Try to delete the transfer - should be blocked because shares were sold
			err = store.DeleteTransaction(ctx, transferID)
			if err == nil {
				t.Fatal("expected error when deleting transfer with sold shares, got nil")
			}
		})
	}
}

// TestRSU2_VestLotInstrumentMismatch verifies that vestLots rejects lots
// whose instrument doesn't match the vest instrument.
func TestRSU2_VestLotInstrumentMismatch(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("rsu2LotInstrument"))
			unvestedID, investmentID, _, instrumentID, categoryID := rsuTestSetup(t, ctx, store, mktStore)

			// Create a second instrument
			instrumentID2, err := mktStore.CreateInstrument(ctx, marketdata.Instrument{
				Symbol:   "OTHER",
				Name:     "Other Stock",
				Currency: currency.USD,
			})
			if err != nil {
				t.Fatalf("CreateInstrument: %v", err)
			}

			// Grant 100 of instrument 1
			_, err = store.CreateStockGrant(ctx, StockGrant{
				Description:     "Grant Inst1",
				Date:            getDate("2025-01-01"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant(inst1): %v", err)
			}

			lots, _ := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			lotIDInst1 := lots[0].Id

			// Try to vest using instrument 2 but selecting lots from instrument 1
			_, err = store.CreateStockVest(ctx, StockVest{
				Description:     "Cross-instrument vest",
				Date:            getDate("2025-06-01"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID2,
				VestingPrice:    75.0,
				CategoryID:      categoryID,
				LotSelections:   []LotSelection{{LotID: lotIDInst1, Quantity: 50}},
			})
			if err == nil {
				t.Fatal("expected error when vesting with mismatched instrument, got nil")
			}
			if !strings.Contains(err.Error(), "instrument") {
				t.Errorf("expected instrument mismatch error, got: %v", err)
			}
		})
	}
}

// TestRSU2_TwoVestsDeleteFirstRestoresCorrectly verifies that when two
// vests come from the same grant, deleting the first one restores exactly
// the right number of shares (scenario 15).
func TestRSU2_TwoVestsDeleteFirstRestoresCorrectly(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("rsu2TwoVestsDelete"))
			unvestedID, investmentID, _, instrumentID, categoryID := rsuTestSetup(t, ctx, store, mktStore)

			// Grant 100
			_, err := store.CreateStockGrant(ctx, StockGrant{
				Description:     "Grant 100",
				Date:            getDate("2025-01-01"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			lots, _ := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			lotID := lots[0].Id

			// Vest A: 30 shares
			vestA, err := store.CreateStockVest(ctx, StockVest{
				Description:     "Vest A 30",
				Date:            getDate("2025-03-01"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    60.0,
				CategoryID:      categoryID,
				LotSelections:   []LotSelection{{LotID: lotID, Quantity: 30}},
			})
			if err != nil {
				t.Fatalf("CreateStockVest A: %v", err)
			}

			// Vest B: 40 shares
			_, err = store.CreateStockVest(ctx, StockVest{
				Description:     "Vest B 40",
				Date:            getDate("2025-06-01"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    75.0,
				CategoryID:      categoryID,
				LotSelections:   []LotSelection{{LotID: lotID, Quantity: 40}},
			})
			if err != nil {
				t.Fatalf("CreateStockVest B: %v", err)
			}

			// Verify: source=30, target=70
			srcLots, _ := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if got := sumLotQuantity(t, srcLots); got != 30 {
				t.Fatalf("after both vests: source = %v, want 30", got)
			}

			// Delete vest A
			if err := store.DeleteTransaction(ctx, vestA); err != nil {
				t.Fatalf("DeleteTransaction(vestA): %v", err)
			}

			// Source should be 60 (30 restored from vest A, 40 still vested by B)
			srcLots, _ = store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if got := sumLotQuantity(t, srcLots); got != 60 {
				t.Errorf("after delete vest A: source = %v, want 60", got)
			}
			// Target should be 40 (only vest B remains)
			tgtLots, _ := store.ListLots(ctx, ListLotsOpts{AccountID: investmentID, InstrumentID: instrumentID})
			if got := sumLotQuantity(t, tgtLots); got != 40 {
				t.Errorf("after delete vest A: target = %v, want 40", got)
			}
		})
	}
}

// TestRSU2_GrantTransferVestDelete verifies the complex sequence:
// Grant 100 to unvested, Vest 50 to investment A, Transfer 30 from A to B, delete transfer.
// Transfer deletion should succeed because the transferred lots haven't been sold.
func TestRSU2_GrantTransferVestDelete(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("rsu2GrantTransferVest"))
			unvestedID, investmentID, _, instrumentID, categoryID := rsuTestSetup(t, ctx, store, mktStore)

			// Create a second investment account for the transfer target.
			providerID, err := store.CreateAccountProvider(ctx, AccountProvider{Name: "Broker2"})
			if err != nil {
				t.Fatal(err)
			}
			investmentB, err := store.CreateAccount(ctx, Account{
				AccountProviderID: providerID,
				Name:              "Investment B",
				Currency:          currency.USD,
				Type:              InvestmentAccountType,
			})
			if err != nil {
				t.Fatal(err)
			}

			// Grant 100 to unvested
			_, err = store.CreateStockGrant(ctx, StockGrant{
				Description:     "Grant 100",
				Date:            getDate("2025-01-01"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			// Vest 50 from unvested to investment A
			lots, _ := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if len(lots) == 0 || sumLotQuantity(t, lots) != 100 {
				t.Fatalf("expected 100 unvested shares, got %v", sumLotQuantity(t, lots))
			}
			lotID := lots[0].Id

			_, err = store.CreateStockVest(ctx, StockVest{
				Description:     "Vest 50",
				Date:            getDate("2025-03-01"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    75.0,
				CategoryID:      categoryID,
				LotSelections:   []LotSelection{{LotID: lotID, Quantity: 50}},
			})
			if err != nil {
				t.Fatalf("CreateStockVest: %v", err)
			}

			// Transfer 30 from investment A to investment B
			transferID, err := store.CreateStockTransfer(ctx, StockTransfer{
				Description:     "Transfer 30",
				Date:            getDate("2025-06-01"),
				SourceAccountID: investmentID,
				TargetAccountID: investmentB,
				InstrumentID:    instrumentID,
				Quantity:        30,
			})
			if err != nil {
				t.Fatalf("CreateStockTransfer: %v", err)
			}

			// Try to delete the transfer - it should succeed because the
			// transferred lots haven't been sold or further transferred
			err = store.DeleteTransaction(ctx, transferID)
			if err != nil {
				t.Fatalf("DeleteTransaction(transfer): expected success, got %v", err)
			}

			// After deleting transfer: unvested=50, investmentA=50, investmentB=0
			srcLots, _ := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if got := sumLotQuantity(t, srcLots); got != 50 {
				t.Errorf("unvested = %v, want 50", got)
			}
			invALots, _ := store.ListLots(ctx, ListLotsOpts{AccountID: investmentID, InstrumentID: instrumentID})
			if got := sumLotQuantity(t, invALots); got != 50 {
				t.Errorf("investmentA = %v, want 50", got)
			}
			invBLots, _ := store.ListLots(ctx, ListLotsOpts{AccountID: investmentB, InstrumentID: instrumentID})
			if got := sumLotQuantity(t, invBLots); got != 0 {
				t.Errorf("investmentB = %v, want 0", got)
			}
		})
	}
}

// TestRSU2_UpdateVestCategoryValidation verifies that updating a vest
// to an expense category is rejected.
func TestRSU2_UpdateVestCategoryValidation(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("rsu2UpdateVestCat"))
			unvestedID, investmentID, _, instrumentID, categoryID := rsuTestSetup(t, ctx, store, mktStore)

			// Create an expense category
			expenseCatID, err := store.CreateCategory(ctx, CategoryData{Name: "Expense Cat", Type: ExpenseCategory}, 0)
			if err != nil {
				t.Fatalf("CreateCategory: %v", err)
			}

			// Grant 100
			_, err = store.CreateStockGrant(ctx, StockGrant{
				Description:     "Grant 100",
				Date:            getDate("2025-01-01"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			lots, _ := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			lotID := lots[0].Id

			// Create vest with valid income category
			vestID, err := store.CreateStockVest(ctx, StockVest{
				Description:     "Vest 50",
				Date:            getDate("2025-06-01"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    75.0,
				CategoryID:      categoryID,
				LotSelections:   []LotSelection{{LotID: lotID, Quantity: 50}},
			})
			if err != nil {
				t.Fatalf("CreateStockVest: %v", err)
			}

			// Try to update vest to expense category - should fail
			err = store.UpdateStockVest(ctx, StockVestUpdate{CategoryID: &expenseCatID}, vestID)
			if err == nil {
				t.Fatal("expected error when updating vest to expense category, got nil")
			}
			if !strings.Contains(err.Error(), "income category") {
				t.Errorf("expected income category error, got: %v", err)
			}
		})
	}
}

// TestRSU_PositionConsistency verifies that after create/delete/update sequences,
// dbPosition always matches the sum of open/partial lots.
func TestRSU_PositionConsistency(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("rsuPositionConsistency"))
			unvestedID, investmentID, cashID, instrumentID, categoryID := rsuTestSetup(t, ctx, store, mktStore)

			// Helper to check position matches lots
			checkConsistency := func(label string) {
				t.Helper()
				for _, accID := range []uint{unvestedID, investmentID} {
					lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: accID, InstrumentID: instrumentID})
					if err != nil {
						t.Fatalf("%s: ListLots(%d): %v", label, accID, err)
					}
					lotQty := 0.0
					lotCost := 0.0
					for _, l := range lots {
						if l.Status != LotClosed {
							lotQty += l.Quantity
							lotCost += l.CostBasis
						}
					}
					pos, err := store.GetPosition(ctx, accID, instrumentID)
					if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
						t.Fatalf("%s: GetPosition(%d): %v", label, accID, err)
					}
					if math.Abs(pos.Quantity-lotQty) > 0.001 {
						t.Errorf("%s: account %d position.Quantity=%v != lots sum=%v", label, accID, pos.Quantity, lotQty)
					}
					if math.Abs(pos.CostBasis-roundMoney(lotCost)) > 0.01 {
						t.Errorf("%s: account %d position.CostBasis=%v != lots sum=%v", label, accID, pos.CostBasis, roundMoney(lotCost))
					}
				}
			}

			// Grant 100
			_, err := store.CreateStockGrant(ctx, StockGrant{
				Description:     "RSU Grant 100",
				Date:            getDate("2025-01-01"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}
			checkConsistency("after grant")

			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			lotID := lots[0].Id

			// Vest 60
			vestID, err := store.CreateStockVest(ctx, StockVest{
				Description:     "Vest 60",
				Date:            getDate("2025-06-01"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    75.0,
				CategoryID:      categoryID,
				LotSelections:   []LotSelection{{LotID: lotID, Quantity: 60}},
			})
			if err != nil {
				t.Fatalf("CreateStockVest: %v", err)
			}
			checkConsistency("after vest")

			// Sell 30
			sellID, err := store.CreateStockSell(ctx, StockSell{
				Description:         "Sell 30",
				Date:                getDate("2025-07-01"),
				InvestmentAccountID: investmentID,
				CashAccountID:       cashID,
				InstrumentID:        instrumentID,
				Quantity:            30,
				TotalAmount:         3000,
			})
			if err != nil {
				t.Fatalf("CreateStockSell: %v", err)
			}
			checkConsistency("after sell")

			// Delete sell
			if err := store.DeleteTransaction(ctx, sellID); err != nil {
				t.Fatalf("DeleteTransaction(sell): %v", err)
			}
			checkConsistency("after delete sell")

			// Delete vest
			if err := store.DeleteTransaction(ctx, vestID); err != nil {
				t.Fatalf("DeleteTransaction(vest): %v", err)
			}
			checkConsistency("after delete vest")
		})
	}
}

// ---------------------------------------------------------------------------
// Stock Forfeit Tests
// ---------------------------------------------------------------------------

// setupCreateStockForfeit creates a grant and forfeits 40 shares, returning the forfeit ID and lot ID.
func setupCreateStockForfeit(t *testing.T, ctx context.Context, store *Store,
	unvestedID, instrumentID uint,
) (forfeitID uint, lotID uint) {
	t.Helper()
	_, err := store.CreateStockGrant(ctx, StockGrant{
		Description:     "RSU Grant 100 shares",
		Date:            getDate("2025-01-15"),
		AccountID:       unvestedID,
		InstrumentID:    instrumentID,
		Quantity:        100,
		FairMarketValue: 50.0,
	})
	if err != nil {
		t.Fatalf("CreateStockGrant: %v", err)
	}

	lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
	if err != nil {
		t.Fatalf("ListLots: %v", err)
	}
	if len(lots) != 1 {
		t.Fatalf("expected 1 lot in unvested account, got %d", len(lots))
	}
	lotID = lots[0].Id

	forfeitID, err = store.CreateStockForfeit(ctx, StockForfeit{
		Description:   "Forfeit 40 shares",
		Date:          getDate("2025-06-15"),
		AccountID:     unvestedID,
		InstrumentID:  instrumentID,
		LotSelections: []LotSelection{{LotID: lotID, Quantity: 40}},
	})
	if err != nil {
		t.Fatalf("CreateStockForfeit: %v", err)
	}
	if forfeitID == 0 {
		t.Fatal("expected non-zero transaction id")
	}
	return forfeitID, lotID
}

func verifyStockForfeitGetTransaction(t *testing.T, ctx context.Context, store *Store,
	forfeitID, unvestedID, instrumentID, lotID uint,
) {
	t.Helper()
	got, err := store.GetTransaction(ctx, forfeitID)
	if err != nil {
		t.Fatalf("GetTransaction: %v", err)
	}
	gotForfeit, ok := got.(StockForfeit)
	if !ok {
		t.Fatalf("expected StockForfeit, got %T", got)
	}
	if gotForfeit.AccountID != unvestedID {
		t.Errorf("AccountID: got %d, want %d", gotForfeit.AccountID, unvestedID)
	}
	if gotForfeit.InstrumentID != instrumentID {
		t.Errorf("InstrumentID: got %d, want %d", gotForfeit.InstrumentID, instrumentID)
	}
	if gotForfeit.Quantity != 40 {
		t.Errorf("Quantity: got %v, want 40", gotForfeit.Quantity)
	}
	if gotForfeit.Description != "Forfeit 40 shares" {
		t.Errorf("Description: got %q, want %q", gotForfeit.Description, "Forfeit 40 shares")
	}
	if len(gotForfeit.LotSelections) != 1 {
		t.Fatalf("LotSelections: got %d, want 1", len(gotForfeit.LotSelections))
	}
	if gotForfeit.LotSelections[0].LotID != lotID || gotForfeit.LotSelections[0].Quantity != 40 {
		t.Errorf("LotSelections[0]: got {%d, %v}, want {%d, 40}", gotForfeit.LotSelections[0].LotID, gotForfeit.LotSelections[0].Quantity, lotID)
	}
}

func verifyStockForfeitLotsAndPosition(t *testing.T, ctx context.Context, store *Store,
	unvestedID, instrumentID uint,
) {
	t.Helper()
	sourceLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
	if err != nil {
		t.Fatalf("ListLots: %v", err)
	}
	totalRemaining := 0.0
	for _, l := range sourceLots {
		totalRemaining += l.Quantity
	}
	if totalRemaining != 60 {
		t.Errorf("remaining quantity: got %v, want 60", totalRemaining)
	}
	pos, err := store.GetPosition(ctx, unvestedID, instrumentID)
	if err != nil {
		t.Fatalf("GetPosition: %v", err)
	}
	if pos.Quantity != 60 {
		t.Errorf("position quantity: got %v, want 60", pos.Quantity)
	}
}

func verifyStockForfeitListTransactions(t *testing.T, ctx context.Context, store *Store,
	forfeitID, unvestedID, instrumentID uint,
) {
	t.Helper()
	list, count, err := store.ListTransactions(ctx, ListOpts{
		StartDate: getDate("2025-06-01"),
		EndDate:   getDate("2025-06-30"),
		Types:     []TxType{StockForfeitTransaction},
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}
	if count != 1 {
		t.Fatalf("ListTransactions count: got %d, want 1", count)
	}
	if len(list) != 1 {
		t.Fatalf("ListTransactions len: got %d, want 1", len(list))
	}
	listedForfeit, ok := list[0].(StockForfeit)
	if !ok {
		t.Fatalf("expected StockForfeit from ListTransactions, got %T", list[0])
	}
	if listedForfeit.Id != forfeitID {
		t.Errorf("ListTransactions Id: got %d, want %d", listedForfeit.Id, forfeitID)
	}
	if listedForfeit.AccountID != unvestedID {
		t.Errorf("ListTransactions AccountID: got %d, want %d", listedForfeit.AccountID, unvestedID)
	}
	if listedForfeit.InstrumentID != instrumentID {
		t.Errorf("ListTransactions InstrumentID: got %d, want %d", listedForfeit.InstrumentID, instrumentID)
	}
	if listedForfeit.Quantity != 40 {
		t.Errorf("ListTransactions Quantity: got %v, want 40", listedForfeit.Quantity)
	}
}

func TestStore_CreateStockForfeit(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("storeCreateStockForfeit"))
			unvestedID, _, instrumentID := setupStockGrantTransferTest(t, ctx, store, mktStore)

			forfeitID, lotID := setupCreateStockForfeit(t, ctx, store, unvestedID, instrumentID)

			t.Run("verify GetTransaction", func(t *testing.T) {
				verifyStockForfeitGetTransaction(t, ctx, store, forfeitID, unvestedID, instrumentID, lotID)
			})
			t.Run("verify lots and position", func(t *testing.T) {
				verifyStockForfeitLotsAndPosition(t, ctx, store, unvestedID, instrumentID)
			})
			t.Run("verify ListTransactions", func(t *testing.T) {
				verifyStockForfeitListTransactions(t, ctx, store, forfeitID, unvestedID, instrumentID)
			})
		})
	}
}

func TestStore_CreateStockForfeit_PartialVest(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("storeForfeitPartialVest"))
			unvestedID, investmentID, instrumentID := setupStockGrantTransferTest(t, ctx, store, mktStore)

			// 1. Grant 100 shares at FMV $50
			_, err := store.CreateStockGrant(ctx, StockGrant{
				Description:     "RSU Grant 100 shares",
				Date:            getDate("2025-01-15"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			// 2. Get lot ID
			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			lotID := lots[0].Id

			// 3. Create income category
			categoryID, err := store.CreateCategory(ctx, CategoryData{Name: "RSU Income", Type: IncomeCategory}, 0)
			if err != nil {
				t.Fatalf("CreateCategory: %v", err)
			}

			// 4. Vest 60 shares at $75
			_, err = store.CreateStockVest(ctx, StockVest{
				Description:     "Vest 60 shares",
				Date:            getDate("2025-03-15"),
				SourceAccountID: unvestedID,
				TargetAccountID: investmentID,
				InstrumentID:    instrumentID,
				VestingPrice:    75.0,
				CategoryID:      categoryID,
				LotSelections: []LotSelection{
					{LotID: lotID, Quantity: 60},
				},
			})
			if err != nil {
				t.Fatalf("CreateStockVest: %v", err)
			}

			// 5. Verify 40 shares remaining in unvested
			unvestedLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots (unvested): %v", err)
			}
			unvestedQty := 0.0
			for _, l := range unvestedLots {
				unvestedQty += l.Quantity
			}
			if unvestedQty != 40 {
				t.Fatalf("unvested quantity after vest: got %v, want 40", unvestedQty)
			}

			// 6. Forfeit remaining 40 shares
			_, err = store.CreateStockForfeit(ctx, StockForfeit{
				Description:  "Forfeit remaining",
				Date:         getDate("2025-06-15"),
				AccountID:    unvestedID,
				InstrumentID: instrumentID,
				LotSelections: []LotSelection{
					{LotID: lotID, Quantity: 40},
				},
			})
			if err != nil {
				t.Fatalf("CreateStockForfeit: %v", err)
			}

			// 7. Verify 0 shares in unvested
			unvestedPos, err := store.GetPosition(ctx, unvestedID, instrumentID)
			if err != nil {
				// Position with 0 shares may not exist
				unvestedPos = Position{Quantity: 0}
			}
			if unvestedPos.Quantity != 0 {
				t.Errorf("unvested position after forfeit: got %v, want 0", unvestedPos.Quantity)
			}

			// 8. Verify 60 shares still in investment (vest was not affected)
			investmentPos, err := store.GetPosition(ctx, investmentID, instrumentID)
			if err != nil {
				t.Fatalf("GetPosition (investment): %v", err)
			}
			if investmentPos.Quantity != 60 {
				t.Errorf("investment position: got %v, want 60", investmentPos.Quantity)
			}
		})
	}
}

func TestDeleteStockForfeit_RestoresLots(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("storeDeleteStockForfeit"))
			unvestedID, _, instrumentID := setupStockGrantTransferTest(t, ctx, store, mktStore)

			// 1. Grant 100 shares
			_, err := store.CreateStockGrant(ctx, StockGrant{
				Description:     "RSU Grant 100 shares",
				Date:            getDate("2025-01-15"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			// 2. Get lot ID
			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			lotID := lots[0].Id

			// 3. Forfeit 40 shares
			forfeitID, err := store.CreateStockForfeit(ctx, StockForfeit{
				Description:  "Forfeit 40 shares",
				Date:         getDate("2025-06-15"),
				AccountID:    unvestedID,
				InstrumentID: instrumentID,
				LotSelections: []LotSelection{
					{LotID: lotID, Quantity: 40},
				},
			})
			if err != nil {
				t.Fatalf("CreateStockForfeit: %v", err)
			}

			// 4. Verify 60 shares remaining
			pos, err := store.GetPosition(ctx, unvestedID, instrumentID)
			if err != nil {
				t.Fatalf("GetPosition: %v", err)
			}
			if pos.Quantity != 60 {
				t.Errorf("position after forfeit: got %v, want 60", pos.Quantity)
			}

			// 5. Delete the forfeit
			err = store.DeleteTransaction(ctx, forfeitID)
			if err != nil {
				t.Fatalf("DeleteTransaction: %v", err)
			}

			// 6. Verify 100 shares restored
			pos, err = store.GetPosition(ctx, unvestedID, instrumentID)
			if err != nil {
				t.Fatalf("GetPosition after delete: %v", err)
			}
			if pos.Quantity != 100 {
				t.Errorf("position after delete: got %v, want 100", pos.Quantity)
			}

			// 7. Verify lot is fully restored
			lotsAfter, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots after delete: %v", err)
			}
			if len(lotsAfter) != 1 {
				t.Fatalf("expected 1 lot, got %d", len(lotsAfter))
			}
			if lotsAfter[0].Quantity != 100 {
				t.Errorf("lot quantity after delete: got %v, want 100", lotsAfter[0].Quantity)
			}
			if lotsAfter[0].Status != LotOpen {
				t.Errorf("lot status after delete: got %v, want LotOpen", lotsAfter[0].Status)
			}
		})
	}
}

func TestStore_UpdateStockForfeit(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("storeUpdateStockForfeit"))
			unvestedID, _, instrumentID := setupStockGrantTransferTest(t, ctx, store, mktStore)

			// 1. Grant 100 shares
			_, err := store.CreateStockGrant(ctx, StockGrant{
				Description:     "RSU Grant 100 shares",
				Date:            getDate("2025-01-15"),
				AccountID:       unvestedID,
				InstrumentID:    instrumentID,
				Quantity:        100,
				FairMarketValue: 50.0,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			// 2. Get lot ID
			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			lotID := lots[0].Id

			// 3. Forfeit 40 shares
			forfeitID, err := store.CreateStockForfeit(ctx, StockForfeit{
				Description:  "Forfeit 40 shares",
				Date:         getDate("2025-06-15"),
				AccountID:    unvestedID,
				InstrumentID: instrumentID,
				LotSelections: []LotSelection{
					{LotID: lotID, Quantity: 40},
				},
			})
			if err != nil {
				t.Fatalf("CreateStockForfeit: %v", err)
			}

			// 4. Verify 60 remaining
			pos, err := store.GetPosition(ctx, unvestedID, instrumentID)
			if err != nil {
				t.Fatalf("GetPosition: %v", err)
			}
			if pos.Quantity != 60 {
				t.Errorf("position after initial forfeit: got %v, want 60", pos.Quantity)
			}

			// 5. Update to forfeit only 30 shares
			newDesc := "Forfeit 30 shares"
			err = store.UpdateTransaction(ctx, StockForfeitUpdate{
				Description: &newDesc,
				LotSelections: []LotSelection{
					{LotID: lotID, Quantity: 30},
				},
			}, forfeitID)
			if err != nil {
				t.Fatalf("UpdateTransaction: %v", err)
			}

			// 6. Verify 70 shares remaining
			pos, err = store.GetPosition(ctx, unvestedID, instrumentID)
			if err != nil {
				t.Fatalf("GetPosition after update: %v", err)
			}
			if pos.Quantity != 70 {
				t.Errorf("position after update: got %v, want 70", pos.Quantity)
			}

			// 7. Verify via GetTransaction
			got, err := store.GetTransaction(ctx, forfeitID)
			if err != nil {
				t.Fatalf("GetTransaction: %v", err)
			}
			gotForfeit, ok := got.(StockForfeit)
			if !ok {
				t.Fatalf("expected StockForfeit, got %T", got)
			}
			if gotForfeit.Description != "Forfeit 30 shares" {
				t.Errorf("Description: got %q, want %q", gotForfeit.Description, "Forfeit 30 shares")
			}
			if gotForfeit.Quantity != 30 {
				t.Errorf("Quantity: got %v, want 30", gotForfeit.Quantity)
			}

			// 8. Verify lot state
			lotsAfter, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instrumentID})
			if err != nil {
				t.Fatalf("ListLots after update: %v", err)
			}
			totalQty := 0.0
			for _, l := range lotsAfter {
				totalQty += l.Quantity
			}
			if totalQty != 70 {
				t.Errorf("lots total quantity after update: got %v, want 70", totalQty)
			}
		})
	}
}
