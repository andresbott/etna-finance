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
					dbCon := db.ConnDbName("TestCategory")
					store, err := New(dbCon)
					if err != nil {
						t.Fatal(err)
					}

					ctx := context.Background()
					category := CategoryData{
						Name: "test",
						Type: categoryType,
					}
					cat1Id, err := store.CreateCategory(ctx, category, 0, tenant1)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					// verify that the id is mounted back
					if cat1Id == 0 {
						t.Fatalf("income category id should not be zero")
					}

					err = store.UpdateCategory(ctx, cat1Id, CategoryData{Name: "changed", Type: categoryType}, tenant1)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					cat2Id, err := store.CreateCategory(ctx, category, 0, tenant1)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					_, err = store.CreateCategory(ctx, CategoryData{Type: categoryType}, cat1Id, tenant2)
					if err == nil {
						t.Fatal("expecting error but none got")
					}

					// ===================================
					//  move items
					// ===================================
					err = store.MoveCategory(ctx, cat1Id, cat2Id, tenant1)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					got, err := store.ListDescendantCategories(ctx, 0, -1, categoryType, tenant1)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					want := []Category{
						{Id: cat2Id, ParentId: 0, CategoryData: CategoryData{Name: "test", Type: categoryType}},
						{Id: cat1Id, ParentId: cat2Id, CategoryData: CategoryData{Name: "changed", Type: categoryType}},
					}

					if diff := cmp.Diff(got, want); diff != "" {
						t.Errorf("unexpected result (-want +got):\n%s", diff)
					}

					// ===================================
					//  delete
					// ===================================
					err = store.DeleteCategoryRecursive(ctx, cat1Id, tenant1)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					got, err = store.ListDescendantCategories(ctx, 0, -1, categoryType, tenant1)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					want = []Category{
						{Id: cat2Id, ParentId: 0, CategoryData: CategoryData{Name: "test", Type: categoryType}},
					}

					if diff := cmp.Diff(got, want); diff != "" {
						t.Errorf("unexpected result (-want +got):\n%s", diff)
					}
				})
			}
		})
	}
}

func TestStore_CreateCategoryErrors(t *testing.T) {
	for _, db := range testdbs.DBs() {
		// test on all DBs
		t.Run(db.DbType(), func(t *testing.T) {

			tcs := []struct {
				name    string
				input   CategoryData
				wantErr string
			}{
				{
					name:    "expect error when mixing category types",
					input:   CategoryData{Name: "test", Type: ExpenseCategory},
					wantErr: ErrCategoryConstraintViolation.Error(),
				},
				{
					name:    "expect error when wrong category type",
					input:   CategoryData{Name: "test"},
					wantErr: ErrWrongCategoryType.Error(),
				},
			}

			dbCon := db.ConnDbName("TestCreateCategoryErrors")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			// create an income
			inCatId, err := store.CreateCategory(t.Context(), CategoryData{Name: "root income", Type: IncomeCategory}, 0, tenant1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			_ = inCatId

			expCatId, err := store.CreateCategory(t.Context(), CategoryData{Name: "root expense", Type: ExpenseCategory}, 0, tenant1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			_ = expCatId

			// test all category Types
			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					_, err = store.CreateCategory(t.Context(), tc.input, inCatId, tenant1)
					if err == nil {
						t.Fatal("expected error but none got")
					}
					if err.Error() != tc.wantErr {
						t.Errorf("expected error: %s, but got %s", tc.wantErr, err.Error())
					}
				})
			}
		})
	}
}

func TestStore_UpdateCategoryErrors(t *testing.T) {
	for _, db := range testdbs.DBs() {
		// test on all DBs
		t.Run(db.DbType(), func(t *testing.T) {

			tcs := []struct {
				name    string
				input   CategoryData
				wantErr string
			}{
				{
					name:    "expect error when mixing category types",
					input:   CategoryData{Name: "test", Type: ExpenseCategory},
					wantErr: ErrCategoryConstraintViolation.Error(),
				},
				{
					name:    "expect error when wrong category type",
					input:   CategoryData{Name: "test"},
					wantErr: ErrWrongCategoryType.Error(),
				},
			}

			dbCon := db.ConnDbName("TestCreateCategoryErrors")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			// create an income
			inCatId, err := store.CreateCategory(t.Context(), CategoryData{Name: "root income", Type: IncomeCategory}, 0, tenant1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			_ = inCatId

			expCatId, err := store.CreateCategory(t.Context(), CategoryData{Name: "root expense", Type: ExpenseCategory}, 0, tenant1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			_ = expCatId

			// test all category Types
			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					err = store.UpdateCategory(t.Context(), inCatId, tc.input, tenant1)
					if err == nil {
						t.Fatal("expected error but none got")
					}
					if err.Error() != tc.wantErr {
						t.Errorf("expected error: %s, but got %s", tc.wantErr, err.Error())
					}
				})
			}
		})
	}
}

func TestStore_MoveCategoryErrors(t *testing.T) {
	for _, db := range testdbs.DBs() {
		// test on all DBs
		t.Run(db.DbType(), func(t *testing.T) {

			tcs := []struct {
				name    string
				input   CategoryData
				wantErr string
			}{
				{
					name:    "expect error when mixing category types",
					input:   CategoryData{Name: "test", Type: ExpenseCategory},
					wantErr: ErrCategoryConstraintViolation.Error(),
				},
			}

			dbCon := db.ConnDbName("TestCreateCategoryErrors")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			// create an income
			inCatId, err := store.CreateCategory(t.Context(), CategoryData{Name: "root income", Type: IncomeCategory}, 0, tenant1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// test all category Types
			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					testCatId, err := store.CreateCategory(t.Context(), tc.input, 0, tenant1)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					err = store.MoveCategory(t.Context(), inCatId, testCatId, tenant1)
					if err == nil {
						t.Fatal("expected error but none got")
					}
					if err.Error() != tc.wantErr {
						t.Errorf("expected error: %s, but got %s", tc.wantErr, err.Error())
					}
				})
			}
		})
	}
}
