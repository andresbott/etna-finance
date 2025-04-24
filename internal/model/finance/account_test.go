package finance

import (
	"context"
	"github.com/go-bumbu/testdbs"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/text/currency"
	"testing"
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
					wantErr:     AccountProviderNotFoundErr.Error(),
				},
			}

			dbCon := db.ConnDbName("TestGetAccountProvider")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}
			sampleData(t, store) // note: test operates on one set of data

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
					wantErr:      AccountProviderNotFoundErr.Error(),
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
			sampleData(t, store) // note: test operates on one set of data

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
					wantErr:       AccountProviderNotFoundErr.Error(),
				},
				{
					name:          "error when updating another tenant",
					updateTenant:  tenant2,
					updateID:      1,
					updatePayload: AccountProviderUpdatePayload{Description: ptr("Updated description")},
					wantErr:       AccountProviderNotFoundErr.Error(),
				},
			}

			dbCon := db.ConnDbName("UpdateAccountProvider")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}
			sampleData(t, store) // note: test operates on one set of data

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
					want:        sampleAccountProviders,
				},
				{
					name:        "get account provider with prefetched accounts",
					preload:     true,
					checkTenant: tenant1,
					want: []AccountProvider{
						{Name: "provider1", Description: "provider1", Accounts: []Account{
							{ID: 1, Name: "acc1", Currency: currency.EUR, Type: 0, AccountProviderID: 1},
							{ID: 3, Name: "acc3", Currency: currency.EUR, Type: 0, AccountProviderID: 1},
							{ID: 3, Name: "acc4", Currency: currency.EUR, Type: 0, AccountProviderID: 1},
							{ID: 4, Name: "acc5", Currency: currency.EUR, Type: 0, AccountProviderID: 1},
						}},
						{Name: "provider2", Description: "provider2", Accounts: []Account{
							{ID: 2, Name: "acc2", Currency: currency.USD, Type: 0, AccountProviderID: 2},
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
			sampleData(t, store) // note: test operates on one set of data

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
					input:  Account{Name: "Main", Currency: currency.USD, Type: Stocks, AccountProviderID: 1},
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

			sampleData(t, store) // note: test operates on one set of data

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
					wantErr:     AccountNotFoundErr.Error(),
				},
			}

			dbCon := db.ConnDbName("TestGetAccount")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			sampleData(t, store) // note: test operates on one set of data

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
					wantErr:      AccountNotFoundErr.Error(),
				},
				{
					name:         "error when deleting non-existent tenant",
					deleteTenant: emptyTenant,
					deleteID:     2,
					wantErr:      AccountNotFoundErr.Error(),
				},
			}

			dbCon := db.ConnDbName("TestGetAccount")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			sampleData(t, store) // note: test operates on one set of data

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
					want:          Account{ID: 1, Name: "Updated Name", Currency: currency.EUR, Type: 0, AccountProviderID: 1},
				},
				{
					name:          "update currency",
					updateID:      2,
					updateTenant:  tenant1,
					updatePayload: AccountUpdatePayload{Currency: &currency.EUR},
					want:          Account{ID: 2, Name: "acc2", Currency: currency.EUR, Type: 0, AccountProviderID: 2},
				},
				{
					name:          "update Type",
					updateID:      3,
					updateTenant:  tenant1,
					updatePayload: AccountUpdatePayload{Type: Stocks},
					want:          Account{ID: 3, Name: "acc3", Currency: currency.EUR, Type: Stocks, AccountProviderID: 1},
				},
				{
					name:          "update Provider Id",
					updateID:      4,
					updateTenant:  tenant1,
					updatePayload: AccountUpdatePayload{ProviderID: ptr(2)},
					want:          Account{ID: 4, Name: "acc4", Currency: currency.EUR, Type: 0, AccountProviderID: 2},
				},
				{
					name:          "error when updating non-existent account",
					updateTenant:  tenant1,
					updateID:      9999,
					updatePayload: AccountUpdatePayload{Name: ptr("Updated Name")},
					wantErr:       AccountNotFoundErr.Error(),
				},
				{
					name:          "error when updating wron tenant",
					updateTenant:  emptyTenant,
					updateID:      1,
					updatePayload: AccountUpdatePayload{Name: ptr("Updated Name")},
					wantErr:       AccountNotFoundErr.Error(),
				},
			}

			dbCon := db.ConnDbName("TestUpdateAccount")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			sampleData(t, store) // note: test operates on one set of data

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					ctx := context.Background()

					err = store.UpdateAccount(tc.updatePayload, tc.updateID, tc.updateTenant)
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
					want:        []Account{},
				},
			}

			dbCon := db.ConnDbName("ListAccounts")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			sampleData(t, store) // note: test operates on one set of data

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
