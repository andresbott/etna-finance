package accounting

import (
	"context"
	"github.com/go-bumbu/testdbs"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/text/currency"
	"testing"
	"time"
)

func TestCreateAccountProvider(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			tcs := []struct {
				name    string
				input   AccountProvider
				tenant  string
				wantErr string
			}{
				{
					name:   "create valid account provider",
					tenant: tenant1,
					input:  AccountProvider{Name: "provider1", Description: "test provider"},
				},
				{
					name:    "want error on empty name",
					tenant:  tenant1,
					input:   AccountProvider{Name: "", Description: "test provider"},
					wantErr: "name cannot be empty",
				},
			}

			dbCon := db.ConnDbName("TestCreateAccountProvider")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					ctx := context.Background()
					id, err := store.CreateAccountProvider(ctx, tc.input, tc.tenant)

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
							t.Errorf("expected valid account ID, but got 0")
						}

						got, err := store.GetAccountProvider(ctx, id, tc.tenant)
						if err != nil {
							t.Fatalf("expected account to be found, but got error: %v", err)
						}

						if diff := cmp.Diff(got, tc.input, cmpopts.IgnoreFields(AccountProvider{}, "ID", "Accounts")); diff != "" {
							t.Errorf("unexpected result (-want +got):\n%s", diff)
						}

					}
				})
			}
		})
	}
}

func TestGetAccountProvider(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			tcs := []struct {
				name        string
				checkId     uint
				checkTenant string
				want        AccountProvider
				wantErr     string
			}{
				{
					name:        "get existing account provider",
					checkId:     1,
					checkTenant: tenant1,
					want:        sampleAccountProviders[0],
				},
				{
					name:        "want error when reading from different tenant",
					checkId:     1,
					checkTenant: tenant2,
					wantErr:     ErrAccountProviderNotFound.Error(),
				},
			}

			dbCon := db.ConnDbName("TestGetAccountProvider")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}
			accountSampleData(t, store) // note: test operates on one set of data

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					ctx := context.Background()

					got, err := store.GetAccountProvider(ctx, tc.checkId, tc.checkTenant)
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

						if diff := cmp.Diff(got, tc.want, cmpopts.IgnoreFields(AccountProvider{}, "ID")); diff != "" {
							t.Errorf("unexpected result (-want +got):\n%s", diff)
						}
					}
				})
			}
		})
	}
}

func TestDeleteAccountProvider(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			tcs := []struct {
				name         string
				deleteID     uint
				deleteTenant string
				wantErr      string
			}{
				{
					name:         "delete existing account provider",
					deleteID:     3,
					deleteTenant: tenant1,
				},
				{
					name:         "error when deleting non-existent account",
					deleteID:     9999,
					deleteTenant: tenant1,
					wantErr:      ErrAccountProviderNotFound.Error(),
				},
				{
					name:         "error when deleting while children are referenced", // expect DB constraint to prevent
					deleteID:     1,
					deleteTenant: tenant1,
					wantErr:      "account constraint violation",
				},
			}

			dbCon := db.ConnDbName("DeleteAccountProvider")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}
			accountSampleData(t, store) // note: test operates on one set of data

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					ctx := context.Background()

					err = store.DeleteAccountProvider(ctx, tc.deleteID, tc.deleteTenant)
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

						_, err := store.GetAccountProvider(ctx, tc.deleteID, tc.deleteTenant)
						if err == nil {
							t.Fatalf("expected NotFoundErr, but got account")
						}
					}
				})
			}
		})
	}
}

func TestUpdateAccountProvider(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			tcs := []struct {
				name          string
				updateID      uint
				updateTenant  string
				updatePayload AccountProviderUpdatePayload
				want          AccountProvider
				wantErr       string
			}{
				{
					name:          "update existing account provider name",
					updateTenant:  tenant1,
					updateID:      1,
					updatePayload: AccountProviderUpdatePayload{Name: ptr("Updated Name")},
					want:          AccountProvider{Name: "Updated Name", Description: "provider1", Accounts: []Account{}},
				},
				{
					name:          "update description",
					updateTenant:  tenant1,
					updateID:      2,
					updatePayload: AccountProviderUpdatePayload{Description: ptr("Updated description")},
					want:          AccountProvider{Name: "provider2", Description: "Updated description", Accounts: []Account{}},
				},
				{
					name:          "error when updating non-existent account",
					updateTenant:  tenant1,
					updateID:      9999,
					updatePayload: AccountProviderUpdatePayload{Description: ptr("Updated description")},
					wantErr:       ErrAccountProviderNotFound.Error(),
				},
				{
					name:          "error when updating another tenant",
					updateTenant:  tenant2,
					updateID:      1,
					updatePayload: AccountProviderUpdatePayload{Description: ptr("Updated description")},
					wantErr:       ErrAccountProviderNotFound.Error(),
				},
			}

			dbCon := db.ConnDbName("UpdateAccountProvider")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}
			accountSampleData(t, store) // note: test operates on one set of data

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					ctx := context.Background()
					err = store.UpdateAccountProvider(tc.updatePayload, tc.updateID, tc.updateTenant)
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

						got, err := store.GetAccountProvider(ctx, tc.updateID, tc.updateTenant)
						if err != nil {
							t.Fatalf("expected account provider to be found, but got error: %v", err)
						}

						if diff := cmp.Diff(got, tc.want, cmpopts.IgnoreFields(AccountProvider{}, "ID")); diff != "" {
							t.Errorf("unexpected result (-want +got):\n%s", diff)
						}
					}
				})
			}
		})
	}
}

func TestListAccountsProvider(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			tcs := []struct {
				name        string
				checkTenant string
				preload     bool
				want        []AccountProvider
				wantErr     string
			}{
				{
					name:        "list multiple Account Providers sorted",
					checkTenant: tenant1,
					want:        []AccountProvider{sampleAccountProviders[0], sampleAccountProviders[1], sampleAccountProviders[2]},
				},
				{
					name:        "get account provider with prefetched accounts",
					preload:     true,
					checkTenant: tenant1,
					want: []AccountProvider{
						{Name: "provider1", Description: "provider1", Accounts: []Account{
							{ID: 1, Name: "acc1", Currency: currency.EUR, Type: CashAccountType, AccountProviderID: 1},
							{ID: 3, Name: "acc3", Currency: currency.EUR, Type: CashAccountType, AccountProviderID: 1},
							{ID: 4, Name: "acc4", Currency: currency.EUR, Type: UnknownAccountType, AccountProviderID: 1},
							{ID: 5, Name: "acc5", Currency: currency.EUR, Type: InvestmentAccountType, AccountProviderID: 1},
						}},
						{Name: "provider2", Description: "provider2", Accounts: []Account{
							{ID: 2, Name: "acc2", Currency: currency.USD, Type: CashAccountType, AccountProviderID: 2},
						}},
						{Name: "provider3", Description: "provider3", Accounts: []Account{}}, // 3 does not have accounts
					},
				},
				{
					name:        "want empty result when listing for different tenant",
					checkTenant: emptyTenant,
					want:        []AccountProvider{},
				},
			}

			dbCon := db.ConnDbName("TestListAccountsProvider")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}
			accountSampleData(t, store) // note: test operates on one set of data

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					ctx := context.Background()
					got, err := store.ListAccountsProvider(ctx, tc.checkTenant, tc.preload)
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
						if diff := cmp.Diff(got, tc.want, cmpopts.IgnoreFields(AccountProvider{}, "ID"), cmpopts.IgnoreFields(Account{}, "ID", "Currency")); diff != "" {
							t.Errorf("unexpected result (-want +got):\n%s", diff)
						}
					}
				})
			}
		})
	}
}

// =======================================================================================
// Account
// =======================================================================================

func TestCreateAccount(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			tcs := []struct {
				name    string
				input   Account
				tenant  string
				wantErr string
			}{
				{
					name:   "create valid account",
					tenant: tenant1,
					input:  Account{Name: "Main", Currency: currency.USD, Type: InvestmentAccountType, AccountProviderID: 1},
				},
				{
					name:    "want error on empty name",
					tenant:  tenant1,
					input:   Account{Name: "", Currency: currency.USD, AccountProviderID: 1},
					wantErr: "name cannot be empty",
				},
				{
					name:    "want error on empty Provider ID",
					tenant:  tenant1,
					input:   Account{Name: "sss", Currency: currency.USD},
					wantErr: "account provider ID cannot be empty",
				},
				{
					name:    "want error on empty currency",
					tenant:  tenant1,
					input:   Account{Name: "Main", Currency: currency.Unit{}, AccountProviderID: 1},
					wantErr: "currency cannot be empty",
				},
			}

			dbCon := db.ConnDbName("TestCreateAccount")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			accountSampleData(t, store) // note: test operates on one set of data

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					ctx := context.Background()
					id, err := store.CreateAccount(ctx, tc.input, tc.tenant)

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
							t.Errorf("expected valid account ID, but got 0")
						}

						got, err := store.GetAccount(ctx, id, tc.tenant)
						if err != nil {
							t.Fatalf("expected account to be found, but got error: %v", err)
						}

						if diff := cmp.Diff(got, tc.input, cmpopts.IgnoreFields(Account{}, "ID", "Currency")); diff != "" {
							t.Errorf("unexpected result (-want +got):\n%s", diff)
						}
						// verify currency
						if got.Currency != tc.input.Currency {
							t.Errorf("expected currency %s, but got %s", tc.input.Currency, got.Currency)
						}
					}
				})
			}
		})
	}
}

func TestGetAccount(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			tcs := []struct {
				name        string
				checkId     uint
				checkTenant string
				want        Account
				wantErr     string
			}{
				{
					name:        "get existing account",
					checkId:     1,
					checkTenant: tenant1,
					want:        sampleAccounts[0],
				},
				{
					name:        "want error when reading from different tenant",
					checkId:     1,
					checkTenant: tenant2,
					wantErr:     ErrAccountNotFound.Error(),
				},
			}

			dbCon := db.ConnDbName("TestGetAccount")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			accountSampleData(t, store) // note: test operates on one set of data

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					ctx := context.Background()

					got, err := store.GetAccount(ctx, tc.checkId, tc.checkTenant)
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

						if diff := cmp.Diff(got, tc.want, cmpopts.IgnoreFields(Account{}, "ID", "Currency")); diff != "" {
							t.Errorf("unexpected result (-want +got):\n%s", diff)
						}
						// verify currency
						if got.Currency != tc.want.Currency {
							t.Errorf("expected currency %s, but got %s", tc.want.Currency, got.Currency)
						}
					}
				})
			}
		})
	}
}

func TestDeleteAccount(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			tcs := []struct {
				name         string
				deleteID     uint
				deleteTenant string
				wantErr      string
			}{
				{
					name:         "delete existing account",
					deleteID:     1,
					deleteTenant: tenant1,
				},
				{
					name:         "error when deleting non-existent account",
					deleteTenant: tenant1,
					deleteID:     9999,
					wantErr:      ErrAccountNotFound.Error(),
				},
				{
					name:         "error when deleting non-existent tenant",
					deleteTenant: emptyTenant,
					deleteID:     2,
					wantErr:      ErrAccountNotFound.Error(),
				},
				{
					name:         "error when deleting account that has entries referenced",
					deleteTenant: tenant1,
					deleteID:     3,
					wantErr:      "cannot delete account: 1 entries still reference it",
				},
			}

			dbCon := db.ConnDbName("TestGetAccount")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			// note: all test operates on one set of data
			accountSampleData(t, store)
			// add one entry to trigger error on delete
			_, err = store.CreateTransaction(t.Context(),
				Income{Description: "test", Amount: 1, AccountID: 3, Date: time.Now()}, tenant1)
			if err != nil {
				t.Fatal(err)
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					ctx := context.Background()

					err = store.DeleteAccount(ctx, tc.deleteID, tc.deleteTenant)
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
						_, err := store.GetAccount(ctx, tc.deleteID, tc.deleteTenant)
						if err == nil {
							t.Fatalf("expected NotFoundErr, but got account")
						}
					}
				})
			}
		})
	}
}

func TestUpdateAccount(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			tcs := []struct {
				name          string
				updateID      uint
				updateTenant  string
				updatePayload AccountUpdatePayload
				want          Account
				wantErr       string
			}{
				{
					name:          "update existing account name",
					updateID:      1,
					updateTenant:  tenant1,
					updatePayload: AccountUpdatePayload{Name: ptr("Updated Name")},
					want:          Account{ID: 1, Name: "Updated Name", Currency: currency.EUR, Type: CashAccountType, AccountProviderID: 1},
				},
				{
					name:          "update currency",
					updateID:      2,
					updateTenant:  tenant1,
					updatePayload: AccountUpdatePayload{Currency: &currency.EUR},
					want:          Account{ID: 2, Name: "acc2", Currency: currency.EUR, Type: CashAccountType, AccountProviderID: 2},
				},
				{
					name:          "update Type",
					updateID:      3,
					updateTenant:  tenant1,
					updatePayload: AccountUpdatePayload{Type: InvestmentAccountType},
					want:          Account{ID: 3, Name: "acc3", Currency: currency.EUR, Type: InvestmentAccountType, AccountProviderID: 1},
				},
				{
					name:          "update Provider Id",
					updateID:      4,
					updateTenant:  tenant1,
					updatePayload: AccountUpdatePayload{ProviderID: ptr(uint(2))},
					want:          Account{ID: 4, Name: "acc4", Currency: currency.EUR, Type: 0, AccountProviderID: 2},
				},
				{
					name:          "error when updating non-existent account",
					updateTenant:  tenant1,
					updateID:      9999,
					updatePayload: AccountUpdatePayload{Name: ptr("Updated Name")},
					wantErr:       ErrAccountNotFound.Error(),
				},
				{
					name:          "error when updating wron tenant",
					updateTenant:  emptyTenant,
					updateID:      1,
					updatePayload: AccountUpdatePayload{Name: ptr("Updated Name")},
					wantErr:       ErrAccountNotFound.Error(),
				},
			}

			dbCon := db.ConnDbName("TestUpdateAccount")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			accountSampleData(t, store) // note: test operates on one set of data

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					ctx := context.Background()

					err = store.UpdateAccount(t.Context(), tc.updatePayload, tc.updateID, tc.updateTenant)
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

						got, err := store.GetAccount(ctx, tc.updateID, tc.updateTenant)
						if err != nil {
							t.Fatalf("expected account to be found, but got error: %v", err)
						}

						if diff := cmp.Diff(got, tc.want, cmpopts.IgnoreFields(Account{}, "ID", "Currency")); diff != "" {
							t.Errorf("unexpected result (-want +got):\n%s", diff)
						}
						// verify currency
						if got.Currency != tc.want.Currency {
							t.Errorf("expected currency %s, but got %s", tc.want.Currency, got.Currency)
						}

					}
				})
			}
		})
	}
}

func TestListAccounts(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			tcs := []struct {
				name string

				checkTenant string
				want        []Account
				wantErr     string
			}{
				{
					name:        "list multiple accounts sorted",
					checkTenant: tenant1,
					want:        sampleAccounts,
				},
				{
					name:        "want empty result when listing for different tenant",
					checkTenant: emptyTenant,
					want:        nil,
				},
			}

			dbCon := db.ConnDbName("ListAccounts")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			accountSampleData(t, store) // note: test operates on one set of data

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					ctx := context.Background()
					got, err := store.ListAccounts(ctx, tc.checkTenant)

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

						if diff := cmp.Diff(got, tc.want, cmpopts.IgnoreFields(Account{}, "ID", "Currency")); diff != "" {
							t.Errorf("unexpected result (-want +got):\n%s", diff)
						}
					}
				})
			}
		})
	}
}

var sampleAccountProviders = []AccountProvider{
	{Name: "provider1", Description: "provider1", Accounts: []Account{}},           // 1
	{Name: "provider2", Description: "provider2", Accounts: []Account{}},           // 2
	{Name: "provider3", Description: "provider3", Accounts: []Account{}},           // 3 does not have accounts
	{Name: "provider4_tenant2", Description: "provider4t2", Accounts: []Account{}}, // 3 does not have accounts
}

var sampleAccounts = []Account{
	{ID: 1, Name: "acc1", Currency: currency.EUR, Type: CashAccountType, AccountProviderID: 1},
	{ID: 2, Name: "acc2", Currency: currency.USD, Type: CashAccountType, AccountProviderID: 2},
	{ID: 3, Name: "acc3", Currency: currency.EUR, Type: CashAccountType, AccountProviderID: 1},
	{ID: 4, Name: "acc4", Currency: currency.EUR, Type: UnknownAccountType, AccountProviderID: 1},
	{ID: 5, Name: "acc5", Currency: currency.CHF, Type: InvestmentAccountType, AccountProviderID: 1},
}

var sampleAccounts2 = []Account{
	{ID: 6, Name: "acc1tenant2", Currency: currency.EUR, Type: 0, AccountProviderID: 4},
}

func accountSampleData(t *testing.T, store *Store) {
	ctx := context.Background()
	// =========================================
	// create accounts providers
	// =========================================

	provider1, err := store.CreateAccountProvider(ctx, sampleAccountProviders[0], tenant1)
	if err != nil {
		t.Fatalf("error creating provider 1: %v", err)
	}
	provider2, err := store.CreateAccountProvider(ctx, sampleAccountProviders[1], tenant1)
	if err != nil {
		t.Fatalf("error creating provider 2: %v", err)
	}
	provider3, err := store.CreateAccountProvider(ctx, sampleAccountProviders[2], tenant1)
	if err != nil {
		t.Fatalf("error creating provider 2: %v", err)
	}
	_ = provider3

	provider4, err := store.CreateAccountProvider(ctx, sampleAccountProviders[3], tenant2)
	if err != nil {
		t.Fatalf("error creating provider 2: %v", err)
	}

	// =========================================
	// create accounts
	// =========================================

	acc := sampleAccounts[0]
	acc.AccountProviderID = provider1
	account1, err := store.CreateAccount(ctx, acc, tenant1)
	if err != nil {
		t.Fatalf("error creating account 1: %v", err)
	}
	_ = account1

	acc = sampleAccounts[1]
	acc.AccountProviderID = provider2
	account2, err := store.CreateAccount(ctx, acc, tenant1)
	if err != nil {
		t.Fatalf("error creating account 2: %v", err)
	}
	_ = account2

	for i := 2; i < len(sampleAccounts); i++ {
		acc = sampleAccounts[i]
		acc.AccountProviderID = provider1
		_, err = store.CreateAccount(ctx, acc, tenant1)
		if err != nil {
			t.Fatalf("error creating account 1: %v", err)
		}
	}
	for i := 0; i < len(sampleAccounts2); i++ {
		acc2 := sampleAccounts2[i]
		acc2.AccountProviderID = provider4
		_, err := store.CreateAccount(ctx, acc2, tenant2)
		if err != nil {
			t.Fatalf("error creating account 1: %v", err)
		}
	}
}
