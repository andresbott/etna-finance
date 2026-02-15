package accounting

import (
	"errors"
	"testing"

	"github.com/go-bumbu/testdbs"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/text/currency"
)

func TestCreateInstrument(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			tcs := []struct {
				name    string
				input   Instrument
				tenant  string
				wantErr string
			}{
				{
					name:   "create valid security",
					tenant: tenant1,
					input:  Instrument{Symbol: "AAPL", Name: "Apple Inc.", Currency: currency.USD},
				},
				{
					name:   "create security with empty name",
					tenant: tenant1,
					input:  Instrument{Symbol: "GOOGL", Name: "", Currency: currency.EUR},
				},
				{
					name:    "want error on empty symbol",
					tenant:  tenant1,
					input:   Instrument{Symbol: "", Name: "Unknown", Currency: currency.USD},
					wantErr: "symbol cannot be empty",
				},
				{
					name:    "want error on empty currency",
					tenant:  tenant1,
					input:   Instrument{Symbol: "MSFT", Name: "Microsoft", Currency: currency.Unit{}},
					wantErr: "currency cannot be empty",
				},
				{
					name:    "want error on duplicate symbol for same tenant",
					tenant:  tenant1,
					input:   Instrument{Symbol: "AAPL", Name: "Another Apple", Currency: currency.USD},
					wantErr: ErrInstrumentSymbolDuplicate.Error(),
				},
			}

			dbCon := db.ConnDbName("TestCreateInstrument")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					ctx := t.Context()
					id, err := store.CreateInstrument(ctx, tc.input, tc.tenant)

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

					got, err := store.GetInstrument(ctx, id, tc.tenant)
					if err != nil {
						t.Fatalf("expected security to be found, but got error: %v", err)
					}
					if got.Symbol != tc.input.Symbol || got.Name != tc.input.Name || got.Currency.String() != tc.input.Currency.String() {
						t.Errorf("got Instrument Symbol=%q Name=%q Currency=%q, want Symbol=%q Name=%q Currency=%q",
							got.Symbol, got.Name, got.Currency.String(), tc.input.Symbol, tc.input.Name, tc.input.Currency.String())
					}
				})
			}
		})
	}
}

func TestGetInstrument(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("TestGetInstrument")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			// Create a security for tenant1
			id, err := store.CreateInstrument(ctx, Instrument{Symbol: "TEST", Name: "Test Instrument", Currency: currency.CHF}, tenant1)
			if err != nil {
				t.Fatal(err)
			}

			tcs := []struct {
				name        string
				checkId     uint
				checkTenant string
				want        Instrument
				wantErr     string
			}{
				{
					name:        "get existing security",
					checkId:     id,
					checkTenant: tenant1,
					want:        Instrument{Symbol: "TEST", Name: "Test Instrument", Currency: currency.CHF},
				},
				{
					name:        "want error when reading from different tenant",
					checkId:     id,
					checkTenant: tenant2,
					wantErr:     ErrInstrumentNotFound.Error(),
				},
				{
					name:        "want error when security does not exist",
					checkId:     99999,
					checkTenant: tenant1,
					wantErr:     ErrInstrumentNotFound.Error(),
				},
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					got, err := store.GetInstrument(ctx, tc.checkId, tc.checkTenant)
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
						t.Errorf("got Instrument Symbol=%q Name=%q Currency=%q, want Symbol=%q Name=%q Currency=%q",
							got.Symbol, got.Name, got.Currency.String(), tc.want.Symbol, tc.want.Name, tc.want.Currency.String())
					}
				})
			}
		})
	}
}

func TestListInstruments(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("TestListInstruments")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			// Create instruments for tenant1
			_, _ = store.CreateInstrument(ctx, Instrument{Symbol: "A", Name: "First", Currency: currency.USD}, tenant1)
			_, _ = store.CreateInstrument(ctx, Instrument{Symbol: "B", Name: "Second", Currency: currency.EUR}, tenant1)
			_, _ = store.CreateInstrument(ctx, Instrument{Symbol: "C", Name: "Third", Currency: currency.CHF}, tenant1)

			tcs := []struct {
				name        string
				tenant      string
				wantCount   int
				wantSymbols []string
			}{
				{
					name:        "list multiple instruments sorted by id",
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
					got, err := store.ListInstruments(ctx, tc.tenant)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					if len(got) != tc.wantCount {
						t.Errorf("ListInstruments: got %d items, want %d", len(got), tc.wantCount)
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

func TestUpdateInstrument(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("TestUpdateInstrument")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			id, err := store.CreateInstrument(ctx, Instrument{Symbol: "OLD", Name: "Old Name", Currency: currency.USD}, tenant1)
			if err != nil {
				t.Fatal(err)
			}
			_, err = store.CreateInstrument(ctx, Instrument{Symbol: "TAKEN", Name: "Other", Currency: currency.EUR}, tenant1)
			if err != nil {
				t.Fatal(err)
			}

			tcs := []struct {
				name    string
				id      uint
				tenant  string
				payload InstrumentUpdatePayload
				wantErr string
				want    Instrument
			}{
				{
					name:   "update name only",
					id:     id,
					tenant: tenant1,
					payload: InstrumentUpdatePayload{
						Name: ptr("New Name"),
					},
					want: Instrument{ID: id, Symbol: "OLD", Name: "New Name", Currency: currency.USD},
				},
				{
					name:   "update symbol and currency",
					id:     id,
					tenant: tenant1,
					payload: InstrumentUpdatePayload{
						Symbol:   ptr("NEW"),
						Currency: ptr("EUR"),
					},
					want: Instrument{ID: id, Symbol: "NEW", Name: "New Name", Currency: currency.EUR},
				},
				{
					name:    "empty symbol rejected",
					id:      id,
					tenant:  tenant1,
					payload: InstrumentUpdatePayload{Symbol: ptr("")},
					wantErr: "symbol cannot be empty",
				},
				{
					name:    "empty currency rejected",
					id:      id,
					tenant:  tenant1,
					payload: InstrumentUpdatePayload{Currency: ptr("")},
					wantErr: "currency cannot be empty",
				},
				{
					name:    "no changes",
					id:      id,
					tenant:  tenant1,
					payload: InstrumentUpdatePayload{},
					wantErr: ErrNoChanges.Error(),
				},
				{
					name:    "not found wrong tenant",
					id:      id,
					tenant:  tenant2,
					payload: InstrumentUpdatePayload{Name: ptr("X")},
					wantErr: ErrInstrumentNotFound.Error(),
				},
				{
					name:    "not found wrong id",
					id:      99999,
					tenant:  tenant1,
					payload: InstrumentUpdatePayload{Name: ptr("X")},
					wantErr: ErrInstrumentNotFound.Error(),
				},
				{
					name:    "duplicate symbol rejected",
					id:      id,
					tenant:  tenant1,
					payload: InstrumentUpdatePayload{Symbol: ptr("TAKEN")},
					wantErr: ErrInstrumentSymbolDuplicate.Error(),
				},
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					err := store.UpdateInstrument(ctx, tc.id, tc.tenant, tc.payload)
					if tc.wantErr != "" {
						if err == nil {
							t.Fatalf("expected error: %s, but got none", tc.wantErr)
						}
						if err.Error() != tc.wantErr {
							t.Errorf("expected error: %s, got %v", tc.wantErr, err.Error())
						}
						return
					}
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					got, err := store.GetInstrument(ctx, tc.id, tc.tenant)
					if err != nil {
						t.Fatalf("get after update: %v", err)
					}
					if got.Symbol != tc.want.Symbol || got.Name != tc.want.Name || got.Currency.String() != tc.want.Currency.String() {
						t.Errorf("got Symbol=%q Name=%q Currency=%q, want Symbol=%q Name=%q Currency=%q",
							got.Symbol, got.Name, got.Currency.String(), tc.want.Symbol, tc.want.Name, tc.want.Currency.String())
					}
				})
			}
		})
	}
}

func TestDeleteInstrument(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			dbCon := db.ConnDbName("TestDeleteInstrument")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			id, err := store.CreateInstrument(ctx, Instrument{Symbol: "DEL", Name: "To Delete", Currency: currency.USD}, tenant1)
			if err != nil {
				t.Fatal(err)
			}

			tcs := []struct {
				name    string
				id      uint
				tenant  string
				wantErr string
			}{
				{
					name:   "delete existing",
					id:     id,
					tenant: tenant1,
				},
				{
					name:    "delete again returns not found",
					id:      id,
					tenant:  tenant1,
					wantErr: ErrInstrumentNotFound.Error(),
				},
				{
					name:    "delete wrong tenant",
					id:      id,
					tenant:  tenant2,
					wantErr: ErrInstrumentNotFound.Error(),
				},
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					err := store.DeleteInstrument(ctx, tc.id, tc.tenant)
					if tc.wantErr != "" {
						if err == nil {
							t.Fatalf("expected error: %s, but got none", tc.wantErr)
						}
						if err.Error() != tc.wantErr {
							t.Errorf("expected error: %s, got %v", tc.wantErr, err.Error())
						}
						return
					}
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					_, err = store.GetInstrument(ctx, tc.id, tc.tenant)
					if err == nil {
						t.Error("expected security to be deleted (get should fail)")
					}
					if !errors.Is(err, ErrInstrumentNotFound) {
						t.Errorf("expected ErrInstrumentNotFound after delete, got %v", err)
					}
				})
			}
		})
	}
}
