package finance

import (
	"context"
	closuretree "github.com/go-bumbu/closure-tree"
	"github.com/go-bumbu/testdbs"
	"github.com/google/go-cmp/cmp"
	"testing"
)

// since almost all the logic is delegated to the closure-tree library
// this test is just a simple smoke test
func TestStore_IncomeCategory(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {

			dbCon := db.ConnDbName("TestIncomeCategory")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			ctx := context.Background()
			incomeCat := IncomeCategory{
				Node: closuretree.Node{},
				Name: "test",
			}
			err = store.CreateIncomeCategory(ctx, &incomeCat, 0, tenant1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			err = store.UpdateIncomeCategory(ctx, 1, IncomeCategory{Name: "changed"}, tenant1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			err = store.CreateIncomeCategory(ctx, &incomeCat, 0, tenant1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			err = store.CreateIncomeCategory(ctx, &IncomeCategory{}, 1, tenant2)
			if err == nil {
				t.Fatal("expecting error but none got")
			}

			// ===================================
			//  move items
			// ===================================
			err = store.MoveIncomeCategory(ctx, 1, 2, tenant1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var got []IncomeCategory
			err = store.DescendantsIncomeCategory(ctx, 0, -1, tenant1, &got)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			want := []IncomeCategory{
				{
					Node: closuretree.Node{NodeId: 2, ParentId: 0, Tenant: tenant1},
					Name: "test",
				},
				{
					Node: closuretree.Node{NodeId: 1, ParentId: 2, Tenant: tenant1},
					Name: "changed",
				},
			}

			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("unexpected result (-want +got):\n%s", diff)
			}

			// ===================================
			//  delete
			// ===================================
			err = store.DeleteRecurseIncomeCategory(ctx, 1, tenant1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got = []IncomeCategory{}
			err = store.DescendantsIncomeCategory(ctx, 0, -1, tenant1, &got)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			want = []IncomeCategory{
				{
					Node: closuretree.Node{NodeId: 2, ParentId: 0, Tenant: tenant1},
					Name: "test",
				},
			}

			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("unexpected result (-want +got):\n%s", diff)
			}

		})
	}
}

// since almost all the logic is delegated to the closure-tree library
// this test is just a simple smoke test
func TestStore_ExpenseCategory(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {

			dbCon := db.ConnDbName("TestExpenseCategory")
			store, err := New(dbCon)
			if err != nil {
				t.Fatal(err)
			}

			ctx := context.Background()
			ExpenseCat := ExpenseCategory{
				Node: closuretree.Node{},
				Name: "test",
			}
			err = store.CreateExpenseCategory(ctx, &ExpenseCat, 0, tenant1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			err = store.UpdateExpenseCategory(ctx, 1, ExpenseCategory{Name: "changed"}, tenant1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			err = store.CreateExpenseCategory(ctx, &ExpenseCat, 0, tenant1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			err = store.CreateExpenseCategory(ctx, &ExpenseCategory{}, 1, tenant2)
			if err == nil {
				t.Fatal("expecting error but none got")
			}

			// ===================================
			//  move items
			// ===================================
			err = store.MoveExpenseCategory(ctx, 1, 2, tenant1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var got []ExpenseCategory
			err = store.DescendantsExpenseCategory(ctx, 0, -1, tenant1, &got)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			want := []ExpenseCategory{
				{
					Node: closuretree.Node{NodeId: 2, ParentId: 0, Tenant: tenant1},
					Name: "test",
				},
				{
					Node: closuretree.Node{NodeId: 1, ParentId: 2, Tenant: tenant1},
					Name: "changed",
				},
			}

			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("unexpected result (-want +got):\n%s", diff)
			}

			// ===================================
			//  delete
			// ===================================
			err = store.DeleteRecurseExpenseCategory(ctx, 1, tenant1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got = []ExpenseCategory{}
			err = store.DescendantsExpenseCategory(ctx, 0, -1, tenant1, &got)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			want = []ExpenseCategory{
				{
					Node: closuretree.Node{NodeId: 2, ParentId: 0, Tenant: tenant1},
					Name: "test",
				},
			}

			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("unexpected result (-want +got):\n%s", diff)
			}

		})
	}
}
