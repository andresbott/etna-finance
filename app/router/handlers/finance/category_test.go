package finance

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/andresbott/etna/internal/accounting"
	"github.com/glebarez/sqlite"
	"github.com/google/go-cmp/cmp"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCreateCategory(t *testing.T) {
	tcs := []struct {
		name         string
		userId       string
		categoryType string
		payload      io.Reader
		expectErr    string
		expectCode   int
		expectIcon   string
	}{
		{
			name:         "successful income category request",
			userId:       tenant1,
			categoryType: IncomeCategoryType,
			payload:      bytes.NewBuffer([]byte(`{"name":"Salary"}`)),
			expectCode:   http.StatusOK,
		},
		{
			name:         "successful expense category request",
			userId:       tenant1,
			categoryType: ExpenseCategoryType,
			payload:      bytes.NewBuffer([]byte(`{"name":"Food"}`)),
			expectCode:   http.StatusOK,
		},
		{
			name:         "successful income category request with icon",
			userId:       tenant1,
			categoryType: IncomeCategoryType,
			payload:      bytes.NewBuffer([]byte(`{"name":"Bonus","icon":"bonus-icon"}`)),
			expectCode:   http.StatusOK,
			expectIcon:   "bonus-icon",
		},
		{
			name:         "successful expense category request with icon",
			userId:       tenant1,
			categoryType: ExpenseCategoryType,
			payload:      bytes.NewBuffer([]byte(`{"name":"Entertainment","icon":"entertainment-icon"}`)),
			expectCode:   http.StatusOK,
			expectIcon:   "entertainment-icon",
		},
		{
			name:         "assert create child category",
			userId:       tenant1,
			categoryType: ExpenseCategoryType,
			payload:      bytes.NewBuffer([]byte(`{"name":"Groceries", "parentId": 3}`)),
			expectCode:   http.StatusOK,
		},
		{
			name:         "empty payload",
			userId:       tenant1,
			categoryType: ExpenseCategoryType,
			payload:      nil,
			expectErr:    "request had empty body",
			expectCode:   http.StatusBadRequest,
		},
		{
			name:         "invalid category type",
			userId:       tenant1,
			categoryType: "invalid",
			payload:      bytes.NewBuffer([]byte(`{"name":"Food"}`)),
			expectErr:    "invalid category type: invalid",
			expectCode:   http.StatusBadRequest,
		},
		{
			name:         "malformed payload",
			userId:       tenant1,
			categoryType: ExpenseCategoryType,
			payload:      bytes.NewBuffer([]byte(`{"name":"Food`)),
			expectErr:    "unable to decode json: unexpected EOF",
			expectCode:   http.StatusBadRequest,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleCategoryHandler(t)
			defer end()

			recorder := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/category", tc.payload)
			handler := h.createCategory(tc.categoryType)
			handler.ServeHTTP(recorder, req)

			if tc.expectErr != "" {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectCode)
				}
				respText, err := io.ReadAll(recorder.Body)
				if err != nil {
					t.Fatal(err)
				}
				got := strings.TrimSuffix(string(respText), "\n")
				if got != tc.expectErr {
					t.Errorf("unexpected error message: got \"%s\" want \"%v\"", got, tc.expectErr)
				}
			} else {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectCode)
					t.Errorf("response body: %s", recorder.Body)
					return
				}

				cat := categoryPayload{}
				err := json.NewDecoder(recorder.Body).Decode(&cat)
				if err != nil {
					t.Fatal(err)
				}
				if cat.Id == 0 {
					t.Error("returned category id is empty")
				}
				if tc.expectIcon != "" && cat.Icon != tc.expectIcon {
					t.Errorf("returned icon mismatch: got %q want %q", cat.Icon, tc.expectIcon)
				}
			}
		})
	}
}

func TestUpdateCategory(t *testing.T) {
	tcs := []struct {
		name         string
		userId       string
		categoryType string
		categoryId   uint
		payload      io.Reader
		expectErr    string
		expectCode   int
	}{
		{
			name:         "successful income category update",
			userId:       tenant1,
			categoryType: IncomeCategoryType,
			categoryId:   1,
			payload:      bytes.NewBuffer([]byte(`{"name":"Updated Income"}`)),
			expectCode:   http.StatusOK,
		},
		{
			name:         "successful expense category update",
			userId:       tenant1,
			categoryType: ExpenseCategoryType,
			categoryId:   3,
			payload:      bytes.NewBuffer([]byte(`{"name":"Updated Expenses"}`)),
			expectCode:   http.StatusOK,
		},
		{
			name:         "successful income category update with icon",
			userId:       tenant1,
			categoryType: IncomeCategoryType,
			categoryId:   1,
			payload:      bytes.NewBuffer([]byte(`{"name":"Salary","icon":"salary-updated-icon"}`)),
			expectCode:   http.StatusOK,
		},
		{
			name:         "successful expense category update with icon",
			userId:       tenant1,
			categoryType: ExpenseCategoryType,
			categoryId:   3,
			payload:      bytes.NewBuffer([]byte(`{"name":"Food","icon":"food-updated-icon"}`)),
			expectCode:   http.StatusOK,
		},
		{
			name:         "empty payload",
			userId:       tenant1,
			categoryType: ExpenseCategoryType,
			payload:      nil,
			expectErr:    "request had empty body",
			expectCode:   http.StatusBadRequest,
		},
		{
			name:         "invalid category type",
			userId:       tenant1,
			categoryType: "invalid",
			payload:      bytes.NewBuffer([]byte(`{"name":"Food"}`)),
			expectErr:    "invalid category type: invalid",
			expectCode:   http.StatusBadRequest,
		},
		{
			name:         "malformed payload",
			userId:       tenant1,
			categoryType: ExpenseCategoryType,
			payload:      bytes.NewBuffer([]byte(`{"name":"Food`)),
			expectErr:    "unable to decode json: unexpected EOF",
			expectCode:   http.StatusBadRequest,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleCategoryHandler(t)
			defer end()

			req, _ := http.NewRequest("PATCH", "/api/category/"+strconv.FormatUint(uint64(tc.categoryId), 10), tc.payload)
			recorder := httptest.NewRecorder()
			handler := h.updateCategory(tc.categoryId, tc.categoryType)
			handler.ServeHTTP(recorder, req)

			if tc.expectErr != "" {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectCode)
				}
				respText, err := io.ReadAll(recorder.Body)
				if err != nil {
					t.Fatal(err)
				}
				got := strings.TrimSuffix(string(respText), "\n")
				if got != tc.expectErr {
					t.Errorf("unexpected error message: got \"%s\" want \"%v\"", got, tc.expectErr)
				}
			} else {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectCode)
					t.Errorf("response body: %s", recorder.Body)
				}
			}
		})
	}
}

func TestMoveCategory(t *testing.T) {
	tcs := []struct {
		name         string
		userId       string
		categoryType string
		payload      io.Reader
		expectErr    string
		expectCode   int
	}{
		{
			name:         "successful income category move",
			userId:       tenant1,
			categoryType: IncomeCategoryType,
			payload:      bytes.NewBuffer([]byte(`{"targetParentId": 2}`)),
			expectCode:   http.StatusOK,
		},
		{
			name:         "successful expense category move",
			userId:       tenant1,
			categoryType: ExpenseCategoryType,
			payload:      bytes.NewBuffer([]byte(`{"targetParentId": 2}`)),
			expectCode:   http.StatusOK,
		},
		{
			name:         "empty payload",
			userId:       tenant1,
			categoryType: ExpenseCategoryType,
			payload:      nil,
			expectErr:    "request had empty body",
			expectCode:   http.StatusBadRequest,
		},
		{
			name:         "invalid category type",
			userId:       tenant1,
			categoryType: "invalid",
			payload:      bytes.NewBuffer([]byte(`{"targetParentId": 2}`)),
			expectErr:    "invalid category type: invalid",
			expectCode:   http.StatusBadRequest,
		},
		{
			name:         "malformed payload",
			userId:       tenant1,
			categoryType: ExpenseCategoryType,
			payload:      bytes.NewBuffer([]byte(`{"targetParentId": 2`)),
			expectErr:    "unable to decode json: unexpected EOF",
			expectCode:   http.StatusBadRequest,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleCategoryHandler(t)
			defer end()

			// Use a pre-existing category or create one
			var categoryId uint = 1

			req, _ := http.NewRequest("PATCH", "/api/category/"+strconv.FormatUint(uint64(categoryId), 10)+"/move", tc.payload)
			recorder := httptest.NewRecorder()
			handler := h.moveCategory(categoryId, tc.categoryType)
			handler.ServeHTTP(recorder, req)

			if tc.expectErr != "" {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectCode)
				}
				respText, err := io.ReadAll(recorder.Body)
				if err != nil {
					t.Fatal(err)
				}
				got := strings.TrimSuffix(string(respText), "\n")
				if got != tc.expectErr {
					t.Errorf("unexpected error message: got \"%s\" want \"%v\"", got, tc.expectErr)
				}
			} else {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectCode)
				}
			}
		})
	}
}

func TestDeleteRecurseCategory(t *testing.T) {
	tcs := []struct {
		name         string
		userId       string
		categoryType string
		expectErr    string
		expectCode   int
	}{
		{
			name:         "successful income category delete",
			userId:       tenant1,
			categoryType: IncomeCategoryType,
			expectCode:   http.StatusOK,
		},
		{
			name:         "successful expense category delete",
			userId:       tenant1,
			categoryType: ExpenseCategoryType,
			expectCode:   http.StatusOK,
		},
		{
			name:         "invalid category type",
			userId:       tenant1,
			categoryType: "invalid",
			expectErr:    "invalid category type: invalid",
			expectCode:   http.StatusBadRequest,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleCategoryHandler(t)
			defer end()

			var categoryId uint = 1

			req, _ := http.NewRequest("DELETE", "/api/category/"+strconv.FormatUint(uint64(categoryId), 10), nil)
			recorder := httptest.NewRecorder()
			handler := h.deleteRecurseCategory(categoryId, tc.categoryType)
			handler.ServeHTTP(recorder, req)

			if tc.expectErr != "" {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectCode)
				}
				respText, err := io.ReadAll(recorder.Body)
				if err != nil {
					t.Fatal(err)
				}
				got := strings.TrimSuffix(string(respText), "\n")
				if got != tc.expectErr {
					t.Errorf("unexpected error message: got \"%s\" want \"%v\"", got, tc.expectErr)
				}
			} else {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectCode)
				}
			}
		})
	}
}

func TestListCategory(t *testing.T) {
	tcs := []struct {
		name         string
		userId       string
		categoryType string
		parentId     uint
		expectErr    string
		expectCode   int
		expectBody   string
	}{
		{
			name:         "list top level expenses categories",
			userId:       tenant1,
			categoryType: ExpenseCategoryType,
			parentId:     0,
			expectCode:   http.StatusOK,
			expectBody:   `{"items":[{"id":3,"name":"Food","description":"","icon":"food-icon"},{"id":4,"name":"Transportation","description":"","icon":"transport-icon"},{"id":5,"parentId":3,"name":"Groceries","description":"","icon":"groceries-icon"}]}`,
		},
		{
			name:         "list child level expenses categories",
			userId:       tenant1,
			categoryType: ExpenseCategoryType,
			parentId:     3,
			expectCode:   http.StatusOK,
			expectBody:   `{"items":[{"id":5,"parentId":3,"name":"Groceries","description":"","icon":"groceries-icon"}]}`,
		},
		{
			name:         "list top level income categories",
			userId:       tenant1,
			categoryType: IncomeCategoryType,
			parentId:     0,
			expectCode:   http.StatusOK,
			expectBody:   `{"items":[{"id":1,"name":"Salary","description":"","icon":"salary-icon"},{"id":2,"name":"Investments","description":"","icon":"investments-icon"}]}`,
		},
		{
			name:         "error on wrong category type",
			userId:       tenant1,
			categoryType: "banana",
			expectCode:   http.StatusBadRequest,
			expectBody:   "banana",
			expectErr:    "invalid category type: banana",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, end := SampleCategoryHandler(t)
			defer end()

			req, _ := http.NewRequest("GET", "/api/category/", nil)
			recorder := httptest.NewRecorder()
			handler := h.listCategory(tc.parentId, tc.categoryType)
			handler.ServeHTTP(recorder, req)

			respBody, err := io.ReadAll(recorder.Body)
			if err != nil {
				t.Fatal(err)
			}
			gotBody := strings.TrimSuffix(string(respBody), "\n")

			if tc.expectErr != "" {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectCode)
				}
				if diff := cmp.Diff(tc.expectErr, gotBody); diff != "" {
					t.Errorf("unexpected error message (-want +got):\n%s", diff)
				}
			} else {
				if status := recorder.Code; status != tc.expectCode {
					t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectCode)
				}
				if diff := cmp.Diff(tc.expectBody, gotBody); diff != "" {
					t.Errorf("unexpected response body (+want -got):\n%s", diff)
				}
			}
		})
	}
}

// Sample category handler for testing
func SampleCategoryHandler(t *testing.T) (*CategoryHandler, func()) {
	db, err := gorm.Open(sqlite.Open(inMemorySqlite), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("unable to connect to sqlite: %v", err)
	}

	store, err := accounting.NewStore(db, nil)
	if err != nil {
		t.Fatalf("unable to connect to finance: %v", err)
	}

	createTestCategories(t, store)

	ch := CategoryHandler{
		Store: store,
	}

	closeFn := func() {
		uDb, err := db.DB()
		if err != nil {
			t.Fatalf("unable to get underlying DB: %v", err)
		}

		err = uDb.Close()
		if err != nil {
			t.Fatalf("unable to close underlying DB: %v", err)
		}
	}
	return &ch, closeFn
}

// addTask helper functions to create test categories
func createTestCategories(t *testing.T, store *accounting.Store) {

	// Create some income categories with icons
	incomeCategory1 := accounting.CategoryData{Name: "Salary", Icon: "salary-icon", Type: accounting.IncomeCategory}
	_, err := store.CreateCategory(t.Context(), incomeCategory1, 0)
	if err != nil {
		t.Fatalf("error creating income category: %v", err)
	}

	incomeCategory2 := accounting.CategoryData{Name: "Investments", Icon: "investments-icon", Type: accounting.IncomeCategory}
	_, err = store.CreateCategory(t.Context(), incomeCategory2, 0)
	if err != nil {
		t.Fatalf("error creating income category: %v", err)
	}

	// Create some expense categories with icons
	expenseCategory1 := accounting.CategoryData{Name: "Food", Icon: "food-icon", Type: accounting.ExpenseCategory}
	expense1Id, err := store.CreateCategory(t.Context(), expenseCategory1, 0)
	if err != nil {
		t.Fatalf("error creating expense category: %v", err)
	}

	expenseCategory2 := accounting.CategoryData{Name: "Transportation", Icon: "transport-icon", Type: accounting.ExpenseCategory}
	_, err = store.CreateCategory(t.Context(), expenseCategory2, 0)
	if err != nil {
		t.Fatalf("error creating expense category: %v", err)
	}

	// Create a subcategory with icon
	expenseSubcategory := accounting.CategoryData{Name: "Groceries", Icon: "groceries-icon", Type: accounting.ExpenseCategory}
	_, err = store.CreateCategory(t.Context(), expenseSubcategory, expense1Id)
	if err != nil {
		t.Fatalf("error creating expense subcategory: %v", err)
	}
}
