package finance

import (
	"context"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-bumbu/testdbs"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/text/currency"
	"testing"
	"time"
)

func TestGetReport(t *testing.T) {
	tcs := []struct {
		name      string
		startDate time.Time
		endDate   time.Time
		tenant    string
		wantErr   string
	}{
		{
			name:      "create valid entry",
			startDate: getTime("2020-01-01 00:00:01"),
			endDate:   getTime("2020-01-30 00:00:01"),
			tenant:    tenant1,
		},
	}

	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {

			dbCon := db.ConnDbName("storeCreateEntry")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}
			err = populateSampleData(store)
			if err != nil {
				t.Fatal(err)
			}

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					ctx := context.Background()

					got, err := store.GetReport(ctx, tc.startDate, tc.endDate, tc.tenant)
					if err != nil {
						t.Fatalf("expected results, but got error: %v", err)
					}

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

						spew.Dump(got)
						_ = cmp.Diff(got, tc.startDate)

						//if diff := cmp.Diff(got, tc.input, ignoreEntryFields); diff != "" {
						//	t.Errorf("unexpected result (-want +got):\n%s", diff)
						//}
					}
				})
			}
		})
	}
}

func populateSampleData(store *Store) error {
	ctx := context.Background()

	// Insert categories
	incomeCats := []struct {
		IncomeCategory
		parent uint
	}{
		{IncomeCategory: IncomeCategory{Name: "in_top1"}, parent: 0}, // id 1
		{IncomeCategory: IncomeCategory{Name: "in_sub1"}, parent: 1}, // id 2
		{IncomeCategory: IncomeCategory{Name: "in_sub2"}, parent: 2}, // id 3
		{IncomeCategory: IncomeCategory{Name: "in_top2"}, parent: 0}, // id 4
	}

	for _, category := range incomeCats {
		err := store.CreateIncomeCategory(ctx, &category.IncomeCategory, category.parent, tenant1)
		if err != nil {
			return fmt.Errorf("failed to create expense category: %w", err)
		}
	}
	// insert account
	_, err := store.CreateAccountProvider(ctx, AccountProvider{Name: "Provider1"}, tenant1)
	if err != nil {
		return fmt.Errorf("failed to create Account: %w", err)
	}

	_, err = store.CreateAccount(ctx, Account{Name: "account1", AccountProviderID: 1, Currency: currency.USD, Type: Cash}, tenant1)
	if err != nil {
		return fmt.Errorf("failed to create Account: %w", err)
	}

	// insert Entries
	entries := []Entry{
		{
			Description:     "in_sub2_2020",
			Date:            getTime("2020-01-01 00:00:01"),
			Type:            IncomeEntry,
			TargetAmount:    10,
			TargetAccountID: 1,
			CategoryId:      2,
		},
		{
			Description:     "in_sub1_2020",
			Date:            getTime("2020-01-02 00:00:01"),
			Type:            IncomeEntry,
			TargetAmount:    100,
			TargetAccountID: 1,
			CategoryId:      1,
		},
	}

	// Insert entries
	for _, entry := range entries {
		_, err := store.CreateEntry(ctx, entry, tenant1)
		if err != nil {
			return fmt.Errorf("failed to create entry: %w", err)
		}
	}

	return nil
}
