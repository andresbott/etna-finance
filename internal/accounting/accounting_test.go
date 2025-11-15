package accounting

import (
	"fmt"
	"github.com/go-bumbu/testdbs"
	"github.com/google/go-cmp/cmp"
	"os"
	"testing"
	"time"
)

// TestMain modifies how test are run,
// it makes sure that the needed DBs are ready and does cleanup in the end.
func TestMain(m *testing.M) {
	testdbs.InitDBS()
	// main block that runs tests
	code := m.Run()
	_ = testdbs.Clean()
	os.Exit(code)
}

const (
	tenant1     = "tenant1"
	tenant2     = "tenant2"
	emptyTenant = "tenantEmpty"
)

// returns a pointer to a specific type
func ptr[T any](v T) *T {
	return &v
}
func getDate(timeStr string) time.Time {
	// Parse the string based on the provided layout
	parsedTime, err := time.Parse("2006-01-02", timeStr)
	if err != nil {
		panic(fmt.Errorf("unable to parse time: %v", err))
	}
	return parsedTime
}
func getDateTime(timeStr string) time.Time {
	// Parse the string based on the provided layout
	parsedTime, err := time.Parse("2006-01-02 15:04:05", timeStr)
	if err != nil {
		panic(fmt.Errorf("unable to parse time: %v", err))

	}
	return parsedTime
}

func TestListTenants(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {

			t.Run("with tenant", func(t *testing.T) {
				dbCon := db.ConnDbName("TestListTenants")
				store, err := NewStore(dbCon)
				if err != nil {
					t.Fatal(err)
				}

				// Load sample data (creates tenants like tenant1, tenant2)
				accountSampleData(t, store)

				got, err := store.ListTenants(t.Context())
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				want := []string{tenant1, tenant2}

				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("unexpected result (-want +got):\n%s", diff)
				}
			})

			t.Run("without tenant", func(t *testing.T) {

				dbEmptyCon := db.ConnDbName("TestListTenants_Empty")
				storeEmpty, err := NewStore(dbEmptyCon)
				if err != nil {
					t.Fatal(err)
				}

				gotEmpty, err := storeEmpty.ListTenants(t.Context())
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if len(gotEmpty) != 0 {
					t.Errorf("expected no tenants, got %v", gotEmpty)
				}
			})

		})
	}
}

func TestWipeData(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {

			dbConn := db.ConnDbName("TestWipeData")
			store, err := NewStore(dbConn)
			if err != nil {
				t.Fatal(err)
			}
			categorySampleData(t, store, sampleCategories)
			transactionSampleData(t, store, sumEntriesSample)

			err = store.WipeData(t.Context())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			accounts, err := store.ListAccounts(t.Context(), tenant1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if accounts != nil {
				t.Errorf("expected no accounts, found %d accounbts", len(accounts))
			}

			income, err := store.ListDescendantCategories(t.Context(), 0, -1, IncomeCategory, tenant1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if income != nil {
				t.Errorf("expected no accounts, found %d income categories", len(income))
			}

			expense, err := store.ListDescendantCategories(t.Context(), 0, -1, ExpenseCategory, tenant1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if expense != nil {
				t.Errorf("expected no accounts, found %d expense categories", len(expense))
			}

			opts := ListOpts{
				StartDate: getDate("1900-01-01"),
				EndDate:   getDate("3000-01-04"),
			}
			transactions, err := store.ListTransactions(t.Context(), opts, tenant1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if transactions != nil {
				t.Errorf("expected no transactions, found %d transactions", len(transactions))
			}

		})
	}
}
