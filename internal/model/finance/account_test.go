package finance

import (
	"context"
	"github.com/go-bumbu/testdbs"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/text/currency"
	"testing"
)

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
					input:  Account{Name: "Main", Currency: currency.USD, Type: Stocks},
				},
				{
					name:    "want error on empty name",
					tenant:  tenant1,
					input:   Account{Name: "", Currency: currency.USD},
					wantErr: "name cannot be empty",
				},
				{
					name:    "want error on empty currency",
					tenant:  tenant1,
					input:   Account{Name: "Main", Currency: currency.Unit{}},
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
					create:       Account{Name: "Main", Currency: currency.USD},
					checkTenant:  tenant1,
					want:         Account{Name: "Main", Currency: currency.USD},
				},
				{
					name:         "want error when reading from different tenant",
					createTenant: tenant1,
					create:       Account{Name: "Main", Currency: currency.USD},
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
					create:        Account{Name: "Main", Currency: currency.USD, Type: Cash},
					updateTenant:  tenant1,
					updatePayload: AccountUpdatePayload{Name: ptr("Updated Name")},
					want:          Account{Name: "Updated Name", Currency: currency.USD, Type: Cash},
				},
				{
					name:          "update currency",
					createTenant:  tenant1,
					create:        Account{Name: "Main", Currency: currency.USD},
					updateTenant:  tenant1,
					updatePayload: AccountUpdatePayload{Currency: &currency.EUR},
					want:          Account{Name: "Main", Currency: currency.EUR, Type: Unknown},
				},
				{
					name:          "update Type",
					createTenant:  tenant1,
					create:        Account{Name: "Main", Currency: currency.USD},
					updateTenant:  tenant1,
					updatePayload: AccountUpdatePayload{Type: Stocks},
					want:          Account{Name: "Main", Currency: currency.USD, Type: Stocks},
				},
				{
					name:          "error when updating non-existent account",
					createTenant:  tenant1,
					create:        Account{Name: "Main", Currency: currency.USD},
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
					create:       Account{Name: "Main", Currency: currency.USD},
					deleteTenant: tenant1,
				},
				{
					name:         "error when deleting non-existent account",
					createTenant: tenant1,
					create:       Account{Name: "Main", Currency: currency.USD},
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
					create:       []Account{{Name: "Savings", Currency: currency.USD}, {Name: "Main", Currency: currency.EUR}},
					checkTenant:  tenant1,
					want:         []Account{{Name: "Savings", Currency: currency.EUR}, {Name: "Main", Currency: currency.USD}},
				},
				{
					name:         "want empty result when listing for different tenant",
					createTenant: tenant1,
					create:       []Account{{Name: "Main", Currency: currency.USD}},
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
