package accounting

import (
	"context"
	"testing"

	"github.com/go-bumbu/testdbs"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/text/currency"
)

func TestCreateSecurity(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			tcs := []struct {
				name    string
				input   Security
				tenant  string
				wantErr string
			}{
				{
					name:   "create valid security",
					tenant: tenant1,
					input:  Security{Symbol: "AAPL", Name: "Apple Inc.", Currency: currency.USD},
				},
				{
					name:   "create security with empty name",
					tenant: tenant1,
					input:  Security{Symbol: "GOOGL", Name: "", Currency: currency.EUR},
				},
				{
					name:    "want error on empty symbol",
					tenant:  tenant1,
					input:   Security{Symbol: "", Name: "Unknown", Currency: currency.USD},
					wantErr: "symbol cannot be empty",
				},
				{
					name:    "want error on empty currency",
					tenant:  tenant1,
					input:   Security{Symbol: "MSFT", Name: "Microsoft", Currency: currency.Unit{}},
					wantErr: "currency cannot be empty",
				},
			}

			dbCon := db.ConnDbName("TestCreateSecurity")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					ctx := context.Background()
					id, err := store.CreateSecurity(ctx, tc.input, tc.tenant)

					if tc.wantErr != "" {
						if err == nil {
							t.Fatalf("expected error: %s, but got none", tc.wantErr)
						}
						if err.Error() != tc.wantErr {
							t.Errorf("expected error: %s, but got %v", tc.wantErr, err.Error())
						}
						return
					}
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					if id == 0 {
						t.Errorf("expected valid security id, but got 0")
					}

					got, err := store.GetSecurity(ctx, id, tc.tenant)
					if err != nil {
						t.Fatalf("expected security to be found, but got error: %v", err)
					}
					if got.Symbol != tc.input.Symbol || got.Name != tc.input.Name || got.Currency.String() != tc.input.Currency.String() {
						t.Errorf("got Security Symbol=%q Name=%q Currency=%q, want Symbol=%q Name=%q Currency=%q",
							got.Symbol, got.Name, got.Currency.String(), tc.input.Symbol, tc.input.Name, tc.input.Currency.String())
					}
				})
			}
		})
	}
}

func TestGetSecurity(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := context.Background()
			dbCon := db.ConnDbName("TestGetSecurity")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			// Create a security for tenant1
			id, err := store.CreateSecurity(ctx, Security{Symbol: "TEST", Name: "Test Security", Currency: currency.CHF}, tenant1)
			if err != nil {
				t.Fatal(err)
			}

			tcs := []struct {
				name        string
				checkId     uint
				checkTenant string
				want        Security
				wantErr     string
			}{
				{
					name:        "get existing security",
					checkId:     id,
					checkTenant: tenant1,
					want:        Security{Symbol: "TEST", Name: "Test Security", Currency: currency.CHF},
				},
				{
					name:        "want error when reading from different tenant",
					checkId:     id,
					checkTenant: tenant2,
					wantErr:     ErrSecurityNotFound.Error(),
				},
				{
					name:        "want error when security does not exist",
					checkId:     99999,
					checkTenant: tenant1,
					wantErr:     ErrSecurityNotFound.Error(),
				},
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					got, err := store.GetSecurity(ctx, tc.checkId, tc.checkTenant)
					if tc.wantErr != "" {
						if err == nil {
							t.Fatalf("expected error: %s, but got none", tc.wantErr)
						}
						if err.Error() != tc.wantErr {
							t.Errorf("expected error: %s, but got %v", tc.wantErr, err.Error())
						}
						return
					}
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					if got.Symbol != tc.want.Symbol || got.Name != tc.want.Name || got.Currency.String() != tc.want.Currency.String() {
						t.Errorf("got Security Symbol=%q Name=%q Currency=%q, want Symbol=%q Name=%q Currency=%q",
							got.Symbol, got.Name, got.Currency.String(), tc.want.Symbol, tc.want.Name, tc.want.Currency.String())
					}
				})
			}
		})
	}
}

func TestListSecurities(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := context.Background()
			dbCon := db.ConnDbName("TestListSecurities")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			// Create securities for tenant1
			_, _ = store.CreateSecurity(ctx, Security{Symbol: "A", Name: "First", Currency: currency.USD}, tenant1)
			_, _ = store.CreateSecurity(ctx, Security{Symbol: "B", Name: "Second", Currency: currency.EUR}, tenant1)
			_, _ = store.CreateSecurity(ctx, Security{Symbol: "C", Name: "Third", Currency: currency.CHF}, tenant1)

			tcs := []struct {
				name        string
				tenant      string
				wantCount   int
				wantSymbols []string
			}{
				{
					name:        "list multiple securities sorted by id",
					tenant:      tenant1,
					wantCount:   3,
					wantSymbols: []string{"A", "B", "C"},
				},
				{
					name:        "want empty result for different tenant",
					tenant:      tenant2,
					wantCount:   0,
					wantSymbols: nil,
				},
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					got, err := store.ListSecurities(ctx, tc.tenant)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					if len(got) != tc.wantCount {
						t.Errorf("ListSecurities: got %d items, want %d", len(got), tc.wantCount)
					}
					if tc.wantSymbols != nil {
						symbols := make([]string, len(got))
						for i, s := range got {
							symbols[i] = s.Symbol
						}
						if diff := cmp.Diff(symbols, tc.wantSymbols); diff != "" {
							t.Errorf("unexpected symbols (-want +got):\n%s", diff)
						}
					}
				})
			}
		})
	}
}
