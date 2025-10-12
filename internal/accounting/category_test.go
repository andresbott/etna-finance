package accounting

import (
	"context"
	"github.com/go-bumbu/testdbs"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"sort"
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
					store, err := NewStore(dbCon)
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
			store, err := NewStore(dbCon)
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
			store, err := NewStore(dbCon)
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
			store, err := NewStore(dbCon)
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

func TestGetCategory(t *testing.T) {
	tcs := []struct {
		name    string
		id      uint
		want    Category
		tenant  string
		wantErr string
	}{
		{
			name:   "get a child income",
			tenant: tenant1,
			id:     5,
			want: Category{Id: 5,
				ParentId:     0, // 3 expected  TODO for now the node returns always 0 for, check https://github.com/go-bumbu/closure-tree/issues/10
				CategoryData: CategoryData{Name: "MSFT", Type: IncomeCategory}},
		},
		{
			name:   "get a child expense",
			tenant: tenant1,
			id:     8,
			want: Category{Id: 8,
				ParentId:     0, //7 expected  TODO for now the node returns always 0 for, check https://github.com/go-bumbu/closure-tree/issues/10
				CategoryData: CategoryData{Name: "Electricity", Type: ExpenseCategory}},
		},
	}

	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {

			dbCon := db.ConnDbName("storeGetCategory")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			categorySampleData(t, store, sampleCategories)

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					got, err := store.GetCategory(t.Context(), tc.id, tc.tenant)
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

						if diff := cmp.Diff(got, tc.want, cmp.AllowUnexported(categoryIds{})); diff != "" {
							t.Errorf("unexpected result (-got +want):\n%s", diff)
						}
					}
				})
			}
		})
	}
}

var ignoreCategoryIdFields = cmpopts.IgnoreFields(Category{},
	"Id", "ParentId")

func TestGetCategoryChildren(t *testing.T) {
	tcs := []struct {
		name    string
		catType CategoryType
		tenant  string
		wantErr string
		want    []categoryIds
	}{
		{
			name:    "get income categories",
			catType: IncomeCategory,
			tenant:  tenant1,
			want: []categoryIds{
				{Category: sampleCategories[4], childrenIds: []uint{5}},
				{Category: sampleCategories[0], childrenIds: []uint{1, 3, 4, 5}},
				{Category: sampleCategories[2], childrenIds: []uint{3, 4, 5}},
				{Category: sampleCategories[3], childrenIds: []uint{4}},
			},
		},
		{
			name:    "get expense categories",
			catType: ExpenseCategory,
			tenant:  tenant1,
			want: []categoryIds{
				{Category: sampleCategories[6], childrenIds: []uint{7, 8}},       // Bills
				{Category: sampleCategories[7], childrenIds: []uint{8}},          // Electricity
				{Category: sampleCategories[5], childrenIds: []uint{6}},          // Groceries
				{Category: sampleCategories[1], childrenIds: []uint{2, 6, 7, 8}}, // Home
			},
		},
	}

	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {

			dbCon := db.ConnDbName("storeGetDescendants")
			store, err := NewStore(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			categorySampleData(t, store, sampleCategories)

			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {

					ctx := context.Background()

					got, err := store.getCategoryChildren(ctx, tc.catType, tc.tenant)
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

						// Sort by Name
						sort.Slice(got, func(i, j int) bool {
							return got[i].Name < got[j].Name
						})

						if diff := cmp.Diff(got, tc.want, ignoreCategoryIdFields, cmp.AllowUnexported(categoryIds{})); diff != "" {
							t.Errorf("unexpected result (-got +want):\n%s", diff)
						}
					}
				})
			}
		})
	}
}

var sampleCategories = []Category{
	{Id: 1, ParentId: 0, CategoryData: CategoryData{Name: "Salary", Type: IncomeCategory}},
	{Id: 2, ParentId: 0, CategoryData: CategoryData{Name: "Home", Type: ExpenseCategory}},
	{Id: 3, ParentId: 1, CategoryData: CategoryData{Name: "Stock benefits", Type: IncomeCategory}},
	{Id: 4, ParentId: 3, CategoryData: CategoryData{Name: "Voo", Type: IncomeCategory}},
	{Id: 5, ParentId: 3, CategoryData: CategoryData{Name: "MSFT", Type: IncomeCategory}},
	{Id: 6, ParentId: 2, CategoryData: CategoryData{Name: "Groceries", Type: ExpenseCategory}},
	{Id: 7, ParentId: 2, CategoryData: CategoryData{Name: "Bills", Type: ExpenseCategory}},
	{Id: 8, ParentId: 7, CategoryData: CategoryData{Name: "Electricity", Type: ExpenseCategory}},
}

func categorySampleData(t *testing.T, store *Store, categories []Category) {

	// =========================================
	// create accounts
	// =========================================

	for _, cat := range categories {
		_, err := store.CreateCategory(t.Context(), cat.CategoryData, cat.ParentId, tenant1)
		if err != nil {
			t.Fatalf("error creating account 1: %v", err)
		}
	}

}
