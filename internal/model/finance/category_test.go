package finance

import (
	"context"
	"github.com/go-bumbu/testdbs"
	"github.com/google/go-cmp/cmp"
	"testing"
)

// since almost all the logic is delegated to the closure-tree library
// this test is just a simple smoke test
func TestStore_CategorySmoke(t *testing.T) {
	for _, db := range testdbs.DBs() {
		// test on all DBs
		t.Run(db.DbType(), func(t *testing.T) {

			categoryTypes := map[string]CategoryType{
				"income":  IncomeCategory,
				"expense": ExpenseCategory,
			}

			// test all category Types
			for catStr, categoryType := range categoryTypes {
				t.Run(catStr, func(t *testing.T) {
					dbCon := db.ConnDbName("TestIncomeCategory")
					store, err := New(dbCon)
					if err != nil {
						t.Fatal(err)
					}

					ctx := context.Background()
					category := CategoryData{
						Name: "test",
						Type: categoryType,
					}
					catId, err := store.CreateCategory(ctx, category, 0, tenant1)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					// verify that the id is mounted back
					if catId == 0 {
						t.Fatalf("income category id should not be zero")
					}

					err = store.UpdateCategory(ctx, 1, CategoryData{Name: "changed", Type: categoryType}, tenant1)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					_, err = store.CreateCategory(ctx, category, 0, tenant1)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					_, err = store.CreateCategory(ctx, CategoryData{Type: categoryType}, 1, tenant2)
					if err == nil {
						t.Fatal("expecting error but none got")
					}

					// ===================================
					//  move items
					// ===================================
					err = store.MoveCategory(ctx, 1, 2, categoryType, tenant1)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					got, err := store.ListDescendantCategories(ctx, 0, -1, categoryType, tenant1)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					want := []Category{
						{Id: 2, ParentId: 0, CategoryData: CategoryData{Name: "test", Type: categoryType}},
						{Id: 1, ParentId: 2, CategoryData: CategoryData{Name: "changed", Type: categoryType}},
					}

					if diff := cmp.Diff(got, want); diff != "" {
						t.Errorf("unexpected result (-want +got):\n%s", diff)
					}

					// ===================================
					//  delete
					// ===================================
					err = store.DeleteCategoryRecursive(ctx, 1, categoryType, tenant1)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					got, err = store.ListDescendantCategories(ctx, 0, -1, categoryType, tenant1)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					want = []Category{
						{Id: 2, ParentId: 0, CategoryData: CategoryData{Name: "test", Type: categoryType}},
					}

					if diff := cmp.Diff(got, want); diff != "" {
						t.Errorf("unexpected result (-want +got):\n%s", diff)
					}
				})
			}

		})
	}
}
