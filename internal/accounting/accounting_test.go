package accounting

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-bumbu/testdbs"
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

func TestWipeData(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {

			dbConn := db.ConnDbName("TestWipeData")
			store, err := NewStore(dbConn, nil)
			if err != nil {
				t.Fatal(err)
			}
			categorySampleData(t, store, sampleCategories)
			transactionSampleData(t, store, sumEntriesSample)

			err = store.WipeData(t.Context())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			accounts, err := store.ListAccounts(t.Context())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if accounts != nil {
				t.Errorf("expected no accounts, found %d accounbts", len(accounts))
			}

			income, err := store.ListDescendantCategories(t.Context(), 0, -1, IncomeCategory)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if income != nil {
				t.Errorf("expected no accounts, found %d income categories", len(income))
			}

			expense, err := store.ListDescendantCategories(t.Context(), 0, -1, ExpenseCategory)
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
			transactions, err := store.ListTransactions(t.Context(), opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if transactions != nil {
				t.Errorf("expected no transactions, found %d transactions", len(transactions))
			}

		})
	}
}
