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

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					dbCon := db.ConnDbName("bkmStoreCreateAccount")
					store, err := New(dbCon)
					if err != nil {
						t.Fatal(err)
					}

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
				name         string
				create       AccountProvider
				createTenant string
				checkTenant  string
				want         AccountProvider
				wantErr      string
			}{
				{
					name:         "get existing account provider",
					createTenant: tenant1,
					create:       AccountProvider{Name: "Main", Description: "test provider"},
					checkTenant:  tenant1,
					want:         AccountProvider{Name: "Main", Description: "test provider", Accounts: []Account{}},
				},
				{
					name:         "want error when reading from different tenant",
					createTenant: tenant1,
					create:       AccountProvider{Name: "Main", Description: "test provider"},
					checkTenant:  tenant2,
					wantErr:      AccountProviderNotFoundErr.Error(),
				},
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					dbCon := db.ConnDbName("bkmStoreGetAccount")
					store, err := New(dbCon)
					if err != nil {
						t.Fatal(err)
					}

					ctx := context.Background()
					id, err := store.CreateAccountProvider(ctx, tc.create, tc.createTenant)
					if err != nil {
						t.Fatal(err)
					}

					got, err := store.GetAccountProvider(ctx, id, tc.checkTenant)
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
				create       AccountProvider
				createTenant string
				deleteID     uint
				deleteTenant string
				wantErr      string
			}{
				{
					name:         "delete existing account provider",
					createTenant: tenant1,
					create:       AccountProvider{Name: "Main"},
					deleteTenant: tenant1,
				},
				{
					name:         "error when deleting non-existent account",
					createTenant: tenant1,
					create:       AccountProvider{Name: "Main"},
					deleteTenant: tenant1,
					deleteID:     9999,
					wantErr:      AccountProviderNotFoundErr.Error(),
				},
				{
					name:         "error when deleting while children are referenced", // expect DB constraint to prevent
					createTenant: tenant1,
					create:       AccountProvider{Name: "Main", Accounts: []Account{{Name: "test", Currency: currency.USD}}},
					deleteTenant: tenant1,
					wantErr:      "account constraint violation",
				},
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					dbCon := db.ConnDbName("bkmStoreDeleteAccount")
					store, err := New(dbCon)
					if err != nil {
						t.Fatal(err)
					}

					ctx := context.Background()
					id, err := store.CreateAccountProvider(ctx, tc.create, tc.createTenant)
					if err != nil {
						t.Fatal(err)
					}

					if len(tc.create.Accounts) > 0 {
						a := tc.create.Accounts[0]
						a.AccountProviderID = id

						_, err = store.CreateAccount(ctx, a, tc.createTenant)
						if err != nil {
							t.Fatal(err)
						}

					}

					if tc.deleteID == 0 {
						tc.deleteID = id
					}

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
				create        AccountProvider
				createTenant  string
				updateID      uint
				updateTenant  string
				updatePayload AccountProviderUpdatePayload
				want          AccountProvider
				wantErr       string
			}{
				{
					name:          "update existing account name",
					createTenant:  tenant1,
					create:        AccountProvider{Name: "Main"},
					updateTenant:  tenant1,
					updatePayload: AccountProviderUpdatePayload{Name: ptr("Updated Name")},
					want:          AccountProvider{Name: "Updated Name", Accounts: []Account{}},
				},
				{
					name:          "update description",
					createTenant:  tenant1,
					create:        AccountProvider{Name: "Main"},
					updateTenant:  tenant1,
					updatePayload: AccountProviderUpdatePayload{Description: ptr("Updated description")},
					want:          AccountProvider{Name: "Main", Description: "Updated description", Accounts: []Account{}},
				},
				{
					name:          "error when updating non-existent account",
					createTenant:  tenant1,
					create:        AccountProvider{Name: "Main"},
					updateTenant:  tenant1,
					updateID:      9999,
					updatePayload: AccountProviderUpdatePayload{Description: ptr("Updated description")},
					wantErr:       AccountProviderNotFoundErr.Error(),
				},
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					dbCon := db.ConnDbName("bkmStoreUpdateAccount")
					store, err := New(dbCon)
					if err != nil {
						t.Fatal(err)
					}

					ctx := context.Background()
					id, err := store.CreateAccountProvider(ctx, tc.create, tc.createTenant)
					if err != nil {
						t.Fatal(err)
					}

					if tc.updateID == 0 {
						tc.updateID = id
					}

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
					want:        []AccountProvider{{Name: "Savings", Accounts: []Account{}}, {Name: "Main", Accounts: []Account{}}},
				},
				{
					name:        "get account provider with prefetched accounts",
					preload:     true,
					checkTenant: tenant1,
					want:        []AccountProvider{{Name: "Savings", Accounts: []Account{sampleAccounts[0]}}, {Name: "Main", Accounts: []Account{}}},
				},
				{
					name:        "want empty result when listing for different tenant",
					checkTenant: tenant2,
					want:        []AccountProvider{},
				},
			}

			dbCon := db.ConnDbName("TestListAccountsProvider")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			create := []AccountProvider{{Name: "Savings", Accounts: []Account{sampleAccounts[0]}}, {Name: "Main"}}

			ctx := context.Background()
			for _, accPrv := range create {
				prId, err := store.CreateAccountProvider(ctx, accPrv, tenant1)
				if err != nil {
					t.Fatal(err)
				}
				if len(accPrv.Accounts) > 0 {
					acc := accPrv.Accounts[0]
					acc.AccountProviderID = prId
					_, err = store.CreateAccount(ctx, acc, tenant1)
					if err != nil {
						t.Fatal(err)
					}
					// insert another faulty account to ensure correct tenant isolation
					_, err = store.CreateAccount(ctx, acc, tenant2)
					if err != nil {
						t.Fatal(err)
					}
				}
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

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

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					dbCon := db.ConnDbName("bkmStoreCreateAccount")
					store, err := New(dbCon)
					if err != nil {
						t.Fatal(err)
					}

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
				name         string
				create       Account
				createTenant string
				checkTenant  string
				want         Account
				wantErr      string
			}{
				{
					name:         "get existing account",
					createTenant: tenant1,
					create:       Account{Name: "Main", Currency: currency.USD, AccountProviderID: 2},
					checkTenant:  tenant1,
					want:         Account{Name: "Main", Currency: currency.USD, AccountProviderID: 2},
				},
				{
					name:         "want error when reading from different tenant",
					createTenant: tenant1,
					create:       Account{Name: "Main", Currency: currency.USD, AccountProviderID: 2},
					checkTenant:  tenant2,
					wantErr:      AccountNotFoundErr.Error(),
				},
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					dbCon := db.ConnDbName("bkmStoreGetAccount")
					store, err := New(dbCon)
					if err != nil {
						t.Fatal(err)
					}

					ctx := context.Background()
					id, err := store.CreateAccount(ctx, tc.create, tc.createTenant)
					if err != nil {
						t.Fatal(err)
					}

					got, err := store.GetAccount(ctx, id, tc.checkTenant)
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
				create       Account
				createTenant string
				deleteID     uint
				deleteTenant string
				wantErr      string
			}{
				{
					name:         "delete existing account",
					createTenant: tenant1,
					create:       Account{Name: "Main", Currency: currency.USD, AccountProviderID: 2},
					deleteTenant: tenant1,
				},
				{
					name:         "error when deleting non-existent account",
					createTenant: tenant1,
					create:       Account{Name: "Main", Currency: currency.USD, AccountProviderID: 2},
					deleteTenant: tenant1,
					deleteID:     9999,
					wantErr:      AccountNotFoundErr.Error(),
				},
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					dbCon := db.ConnDbName("bkmStoreDeleteAccount")
					store, err := New(dbCon)
					if err != nil {
						t.Fatal(err)
					}

					ctx := context.Background()
					id, err := store.CreateAccount(ctx, tc.create, tc.createTenant)
					if err != nil {
						t.Fatal(err)
					}

					if tc.deleteID == 0 {
						tc.deleteID = id
					}

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
				create        Account
				createTenant  string
				updateID      uint
				updateTenant  string
				updatePayload AccountUpdatePayload
				want          Account
				wantErr       string
			}{
				{
					name:          "update existing account name",
					createTenant:  tenant1,
					create:        Account{Name: "Main", Currency: currency.USD, Type: Cash, AccountProviderID: 1},
					updateTenant:  tenant1,
					updatePayload: AccountUpdatePayload{Name: ptr("Updated Name")},
					want:          Account{Name: "Updated Name", Currency: currency.USD, Type: Cash, AccountProviderID: 1},
				},
				{
					name:          "update currency",
					createTenant:  tenant1,
					create:        Account{Name: "Main", Currency: currency.USD, AccountProviderID: 1},
					updateTenant:  tenant1,
					updatePayload: AccountUpdatePayload{Currency: &currency.EUR},
					want:          Account{Name: "Main", Currency: currency.EUR, Type: Unknown, AccountProviderID: 1},
				},
				{
					name:          "update Type",
					createTenant:  tenant1,
					create:        Account{Name: "Main", Currency: currency.USD, AccountProviderID: 1},
					updateTenant:  tenant1,
					updatePayload: AccountUpdatePayload{Type: Stocks},
					want:          Account{Name: "Main", Currency: currency.USD, Type: Stocks, AccountProviderID: 1},
				},
				{
					name:          "update Provider Id",
					createTenant:  tenant1,
					create:        Account{Name: "Main", Currency: currency.USD, AccountProviderID: 1},
					updateTenant:  tenant1,
					updatePayload: AccountUpdatePayload{ProviderID: ptr(2)},
					want:          Account{Name: "Main", Currency: currency.USD, AccountProviderID: 2},
				},
				{
					name:          "error when updating non-existent account",
					createTenant:  tenant1,
					create:        Account{Name: "Main", Currency: currency.USD, AccountProviderID: 1},
					updateTenant:  tenant1,
					updateID:      9999,
					updatePayload: AccountUpdatePayload{Name: ptr("Updated Name")},
					wantErr:       AccountNotFoundErr.Error(),
				},
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					dbCon := db.ConnDbName("bkmStoreUpdateAccount")
					store, err := New(dbCon)
					if err != nil {
						t.Fatal(err)
					}

					ctx := context.Background()
					id, err := store.CreateAccount(ctx, tc.create, tc.createTenant)
					if err != nil {
						t.Fatal(err)
					}

					if tc.updateID == 0 {
						tc.updateID = id
					}

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
				name         string
				create       []Account
				createTenant string
				checkTenant  string
				want         []Account
				wantErr      string
			}{
				{
					name:         "list multiple accounts sorted",
					createTenant: tenant1,
					create:       []Account{{Name: "Savings", Currency: currency.USD, AccountProviderID: 1}, {Name: "Main", Currency: currency.EUR, AccountProviderID: 1}},
					checkTenant:  tenant1,
					want:         []Account{{Name: "Savings", Currency: currency.EUR, AccountProviderID: 1}, {Name: "Main", Currency: currency.USD, AccountProviderID: 1}},
				},
				{
					name:         "want empty result when listing for different tenant",
					createTenant: tenant1,
					create:       []Account{{Name: "Main", Currency: currency.USD, AccountProviderID: 1}},
					checkTenant:  tenant2,
					want:         []Account{},
				},
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					dbCon := db.ConnDbName("bkmStoreListAccounts")
					store, err := New(dbCon)
					if err != nil {
						t.Fatal(err)
					}

					ctx := context.Background()
					for _, acc := range tc.create {
						_, err := store.CreateAccount(ctx, acc, tc.createTenant)
						if err != nil {
							t.Fatal(err)
						}
					}

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
