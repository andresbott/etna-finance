# Re-apply Category Rules — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Allow users to retroactively apply category matching rules to existing transactions across all accounts.

**Architecture:** New backend endpoints (`reapply-preview` and `reapply-submit`) reuse the existing `matchCategory()` logic from the CSV import parser. A new frontend view (`ReapplyRulesView.vue`) mirrors the import preview UX with a DataTable + checkboxes. The flow is triggered from the Category Rules page via a "Re-apply Rules" button.

**Tech Stack:** Go (gorilla/mux, gorm), Vue 3 (PrimeVue DataTable), TypeScript

---

### Task 1: Export `matchCategory` from the internal csvimport package

The `matchCategory()` function in `internal/csvimport/parser.go:892` is currently unexported. The new handler needs to call it.

**Files:**
- Modify: `internal/csvimport/parser.go:892`

**Step 1: Rename `matchCategory` to `MatchCategory`**

In `internal/csvimport/parser.go`, rename the function:

```go
// MatchCategory iterates rules in order and returns the categoryID of the first
// matching rule, or 0 if none match.
func MatchCategory(description string, rules []CategoryRule) uint {
```

**Step 2: Update the call site in the same file**

In `internal/csvimport/parser.go:870`, update the call inside `Parse()`:

```go
		parsed.CategoryID = MatchCategory(parsed.Description, rules)
```

**Step 3: Run tests to verify nothing broke**

Run: `go test ./internal/csvimport/... -v -count=1`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/csvimport/parser.go
git commit -m "$(cat <<'EOF'
refactor: export MatchCategory for reuse by reapply handler

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Add `ReapplyPreview` handler — write test first

**Files:**
- Create: `app/router/handlers/csvimport/reapply_test.go`
- Test: `app/router/handlers/csvimport/reapply_test.go`

**Step 1: Write the failing test**

Create `app/router/handlers/csvimport/reapply_test.go`:

```go
package csvimport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andresbott/etna/internal/accounting"
	"github.com/andresbott/etna/internal/csvimport"
	"github.com/andresbott/etna/internal/marketdata"
	"github.com/glebarez/sqlite"
	"golang.org/x/text/currency"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestStores(t *testing.T) (*accounting.Store, *csvimport.Store) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("unable to open sqlite: %v", err)
	}
	uDb, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = uDb.Close() })

	mktStore, err := marketdata.NewStore(db)
	if err != nil {
		t.Fatalf("unable to create marketdata store: %v", err)
	}
	finStore, err := accounting.NewStore(db, mktStore)
	if err != nil {
		t.Fatalf("unable to create accounting store: %v", err)
	}
	csvStore, err := csvimport.NewStore(db)
	if err != nil {
		t.Fatalf("unable to create csvimport store: %v", err)
	}
	return finStore, csvStore
}

func TestReapplyPreview(t *testing.T) {
	finStore, csvStore := setupTestStores(t)
	ctx := context.Background()

	// Create provider, account
	providerID, err := finStore.CreateAccountProvider(ctx, accounting.AccountProvider{Name: "test", Description: "test", Icon: "bank"})
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	accID, err := finStore.CreateAccount(ctx, accounting.Account{
		Name:              "Checking",
		Currency:          currency.CHF,
		Type:              accounting.CashAccountType,
		AccountProviderID: providerID,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}

	// Create categories
	expCatID, err := finStore.CreateCategory(ctx, accounting.CategoryData{Name: "Food", Icon: "food", Type: accounting.ExpenseCategory}, 0)
	if err != nil {
		t.Fatalf("create expense category: %v", err)
	}
	wrongCatID, err := finStore.CreateCategory(ctx, accounting.CategoryData{Name: "Transport", Icon: "car", Type: accounting.ExpenseCategory}, 0)
	if err != nil {
		t.Fatalf("create wrong category: %v", err)
	}

	// Create category rule: "GROCERY" -> Food
	_, err = csvStore.CreateCategoryRule(ctx, csvimport.CategoryRule{
		Pattern:    "GROCERY",
		IsRegex:    false,
		CategoryID: expCatID,
		Position:   0,
	})
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}

	baseDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	// Tx 1: matches rule, no category (should show as changed=true)
	_, err = finStore.CreateTransaction(ctx, accounting.Expense{
		Description: "GROCERY STORE",
		Amount:      50.0,
		AccountID:   accID,
		CategoryID:  0,
		Date:        baseDate,
	})
	if err != nil {
		t.Fatalf("create tx1: %v", err)
	}

	// Tx 2: matches rule, wrong category (should show as changed=true)
	_, err = finStore.CreateTransaction(ctx, accounting.Expense{
		Description: "GROCERY MARKET",
		Amount:      30.0,
		AccountID:   accID,
		CategoryID:  wrongCatID,
		Date:        baseDate.AddDate(0, 0, 1),
	})
	if err != nil {
		t.Fatalf("create tx2: %v", err)
	}

	// Tx 3: matches rule, correct category (should show as changed=false)
	_, err = finStore.CreateTransaction(ctx, accounting.Expense{
		Description: "GROCERY DEPOT",
		Amount:      20.0,
		AccountID:   accID,
		CategoryID:  expCatID,
		Date:        baseDate.AddDate(0, 0, 2),
	})
	if err != nil {
		t.Fatalf("create tx3: %v", err)
	}

	// Tx 4: does NOT match any rule (should be excluded)
	_, err = finStore.CreateTransaction(ctx, accounting.Expense{
		Description: "RENT PAYMENT",
		Amount:      1000.0,
		AccountID:   accID,
		CategoryID:  0,
		Date:        baseDate.AddDate(0, 0, 3),
	})
	if err != nil {
		t.Fatalf("create tx4: %v", err)
	}

	handler := &ImportHandler{CsvStore: csvStore, FinStore: finStore}
	req := httptest.NewRequest(http.MethodPost, "/api/v0/import/reapply-preview", nil)
	w := httptest.NewRecorder()

	handler.ReapplyPreview().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var rows []ReapplyRow
	if err := json.NewDecoder(w.Body).Decode(&rows); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// Should have 3 rows (tx1, tx2, tx3 match; tx4 excluded)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	// Count changed vs unchanged
	changedCount := 0
	unchangedCount := 0
	for _, row := range rows {
		if row.Changed {
			changedCount++
			if row.NewCategoryID != expCatID {
				t.Errorf("changed row %d: expected newCategoryId=%d, got %d", row.TransactionID, expCatID, row.NewCategoryID)
			}
		} else {
			unchangedCount++
		}
	}

	if changedCount != 2 {
		t.Errorf("expected 2 changed rows, got %d", changedCount)
	}
	if unchangedCount != 1 {
		t.Errorf("expected 1 unchanged row, got %d", unchangedCount)
	}

	// Verify all rows have account info
	for _, row := range rows {
		if row.AccountName == "" {
			t.Errorf("row %d: expected accountName, got empty", row.TransactionID)
		}
		if row.AccountID == 0 {
			t.Errorf("row %d: expected accountId, got 0", row.TransactionID)
		}
		_ = fmt.Sprintf("row check %d", row.TransactionID) // avoid unused import
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./app/router/handlers/csvimport/... -run TestReapplyPreview -v -count=1`
Expected: FAIL — `ReapplyRow` and `ReapplyPreview` do not exist yet

---

### Task 3: Implement `ReapplyPreview` handler

**Files:**
- Create: `app/router/handlers/csvimport/reapply.go`

**Step 1: Write the handler implementation**

Create `app/router/handlers/csvimport/reapply.go`:

```go
package csvimport

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/andresbott/etna/internal/accounting"
	"github.com/andresbott/etna/internal/csvimport"
)

// ReapplyRow represents a single transaction's proposed category change.
type ReapplyRow struct {
	TransactionID   uint    `json:"transactionId"`
	TransactionType string  `json:"transactionType"`
	Description     string  `json:"description"`
	Date            string  `json:"date"`
	Amount          float64 `json:"amount"`
	AccountID       uint    `json:"accountId"`
	AccountName     string  `json:"accountName"`
	CurrentCategoryID   uint   `json:"currentCategoryId"`
	CurrentCategoryName string `json:"currentCategoryName"`
	NewCategoryID       uint   `json:"newCategoryId"`
	NewCategoryName     string `json:"newCategoryName"`
	Changed             bool   `json:"changed"`
}

func (h *ImportHandler) ReapplyPreview() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Load category rules in position order
		rules, err := h.CsvStore.ListCategoryRules(ctx)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to list category rules: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		if len(rules) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("[]"))
			return
		}

		// Load account map for name resolution
		accountMap, err := h.FinStore.ListAccountsMap(ctx)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to list accounts: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		// Load category maps for name resolution
		catNameMap, err := h.buildCategoryNameMap(ctx)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to load categories: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		// Load all income and expense transactions across all accounts
		var rows []ReapplyRow
		for page := 1; ; page++ {
			opts := accounting.ListOpts{
				StartDate: time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC),
				Types:     []accounting.TxType{accounting.IncomeTransaction, accounting.ExpenseTransaction},
				Limit:     accounting.MaxSearchResults,
				Page:      page,
			}

			txs, _, err := h.FinStore.ListTransactions(ctx, opts)
			if err != nil {
				http.Error(w, fmt.Sprintf("unable to list transactions: %s", err.Error()), http.StatusInternalServerError)
				return
			}
			if len(txs) == 0 {
				break
			}

			for _, tx := range txs {
				row, ok := h.buildReapplyRow(tx, rules, accountMap, catNameMap)
				if ok {
					rows = append(rows, row)
				}
			}
		}

		if rows == nil {
			rows = []ReapplyRow{}
		}

		respJSON, err := json.Marshal(rows)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}

func (h *ImportHandler) buildReapplyRow(tx accounting.Transaction, rules []csvimport.CategoryRule, accountMap map[uint]accounting.Account, catNameMap map[uint]string) (ReapplyRow, bool) {
	var row ReapplyRow

	switch item := tx.(type) {
	case accounting.Income:
		newCatID := csvimport.MatchCategory(item.Description, rules)
		if newCatID == 0 {
			return row, false
		}
		row = ReapplyRow{
			TransactionID:   item.Id,
			TransactionType: "income",
			Description:     item.Description,
			Date:            item.Date.Format("2006-01-02"),
			Amount:          item.Amount,
			AccountID:       item.AccountID,
			CurrentCategoryID: item.CategoryID,
			NewCategoryID:     newCatID,
			Changed:           item.CategoryID != newCatID,
		}
	case accounting.Expense:
		newCatID := csvimport.MatchCategory(item.Description, rules)
		if newCatID == 0 {
			return row, false
		}
		row = ReapplyRow{
			TransactionID:   item.Id,
			TransactionType: "expense",
			Description:     item.Description,
			Date:            item.Date.Format("2006-01-02"),
			Amount:          math.Abs(item.Amount),
			AccountID:       item.AccountID,
			CurrentCategoryID: item.CategoryID,
			NewCategoryID:     newCatID,
			Changed:           item.CategoryID != newCatID,
		}
	default:
		return row, false
	}

	if acct, ok := accountMap[row.AccountID]; ok {
		row.AccountName = acct.Name
	}
	row.CurrentCategoryName = catNameMap[row.CurrentCategoryID]
	row.NewCategoryName = catNameMap[row.NewCategoryID]

	return row, true
}

func (h *ImportHandler) buildCategoryNameMap(ctx context.Context) (map[uint]string, error) {
	nameMap := make(map[uint]string)

	incomeCategories, err := h.FinStore.ListCategories(ctx, accounting.IncomeCategory, 0)
	if err != nil {
		return nil, err
	}
	flattenCategories(incomeCategories, "", nameMap)

	expenseCategories, err := h.FinStore.ListCategories(ctx, accounting.ExpenseCategory, 0)
	if err != nil {
		return nil, err
	}
	flattenCategories(expenseCategories, "", nameMap)

	return nameMap, nil
}

func flattenCategories(categories []accounting.Category, prefix string, nameMap map[uint]string) {
	for _, cat := range categories {
		path := cat.Name
		if prefix != "" {
			path = prefix + " > " + cat.Name
		}
		nameMap[cat.ID] = path
		if len(cat.Children) > 0 {
			flattenCategories(cat.Children, path, nameMap)
		}
	}
}
```

**Note:** The `context` import is needed — ensure `import "context"` is at the top if not auto-imported. Also verify `ListCategories` signature exists. If it doesn't, check the actual method name — it might be `ListIncomeCategories` / `ListExpenseCategories` or similar. Adapt accordingly by reading `internal/accounting/category.go`.

**Step 2: Run the test**

Run: `go test ./app/router/handlers/csvimport/... -run TestReapplyPreview -v -count=1`
Expected: PASS

**Step 3: Run all tests to verify nothing broke**

Run: `go test ./... -count=1 2>&1 | tail -20`
Expected: All PASS

**Step 4: Run lint**

Run: `make lint`
Expected: 0 issues

**Step 5: Commit**

```bash
git add app/router/handlers/csvimport/reapply.go app/router/handlers/csvimport/reapply_test.go
git commit -m "$(cat <<'EOF'
feat: add reapply-preview endpoint for category rule re-application

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Add `ReapplySubmit` handler — test first

**Files:**
- Modify: `app/router/handlers/csvimport/reapply_test.go`

**Step 1: Write the failing test**

Append to `app/router/handlers/csvimport/reapply_test.go`:

```go
func TestReapplySubmit(t *testing.T) {
	finStore, csvStore := setupTestStores(t)
	ctx := context.Background()

	// Create provider, account, category
	providerID, _ := finStore.CreateAccountProvider(ctx, accounting.AccountProvider{Name: "test", Description: "test", Icon: "bank"})
	accID, _ := finStore.CreateAccount(ctx, accounting.Account{
		Name:              "Checking",
		Currency:          currency.CHF,
		Type:              accounting.CashAccountType,
		AccountProviderID: providerID,
	})
	expCatID, _ := finStore.CreateCategory(ctx, accounting.CategoryData{Name: "Food", Icon: "food", Type: accounting.ExpenseCategory}, 0)
	incCatID, _ := finStore.CreateCategory(ctx, accounting.CategoryData{Name: "Salary", Icon: "money", Type: accounting.IncomeCategory}, 0)

	baseDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	// Create an expense with no category
	txID1, _ := finStore.CreateTransaction(ctx, accounting.Expense{
		Description: "GROCERY STORE",
		Amount:      50.0,
		AccountID:   accID,
		CategoryID:  0,
		Date:        baseDate,
	})

	handler := &ImportHandler{CsvStore: csvStore, FinStore: finStore}

	t.Run("successful update", func(t *testing.T) {
		body := fmt.Sprintf(`[{"transactionId":%d,"transactionType":"expense","newCategoryId":%d}]`, txID1, expCatID)
		req := httptest.NewRequest(http.MethodPost, "/api/v0/import/reapply-submit", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ReapplySubmit().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]int
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp["updated"] != 1 {
			t.Errorf("expected updated=1, got %d", resp["updated"])
		}
	})

	t.Run("rejects mismatched category type", func(t *testing.T) {
		// Try to assign income category to an expense transaction
		body := fmt.Sprintf(`[{"transactionId":%d,"transactionType":"expense","newCategoryId":%d}]`, txID1, incCatID)
		req := httptest.NewRequest(http.MethodPost, "/api/v0/import/reapply-submit", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ReapplySubmit().ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for mismatched category, got %d: %s", w.Code, w.Body.String())
		}
	})
}
```

Also add `"strings"` to the imports at the top of the file.

**Step 2: Run test to verify it fails**

Run: `go test ./app/router/handlers/csvimport/... -run TestReapplySubmit -v -count=1`
Expected: FAIL — `ReapplySubmit` method does not exist

---

### Task 5: Implement `ReapplySubmit` handler

**Files:**
- Modify: `app/router/handlers/csvimport/reapply.go`

**Step 1: Add the handler implementation**

Append to `app/router/handlers/csvimport/reapply.go`:

```go
type reapplySubmitItem struct {
	TransactionID   uint   `json:"transactionId"`
	TransactionType string `json:"transactionType"`
	NewCategoryID   uint   `json:"newCategoryId"`
}

func (h *ImportHandler) ReapplySubmit() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		var items []reapplySubmitItem
		if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		if len(items) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"updated":0}`))
			return
		}

		ctx := r.Context()
		updated := 0
		for _, item := range items {
			catID := item.NewCategoryID

			var update accounting.TransactionUpdate
			switch item.TransactionType {
			case "expense":
				update = accounting.ExpenseUpdate{CategoryID: &catID}
			case "income":
				update = accounting.IncomeUpdate{CategoryID: &catID}
			default:
				http.Error(w, fmt.Sprintf("unsupported transaction type: %s", item.TransactionType), http.StatusBadRequest)
				return
			}

			if err := h.FinStore.UpdateTransaction(ctx, update, item.TransactionID); err != nil {
				var valErr accounting.ErrValidation
				if errors.As(err, &valErr) {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				http.Error(w, fmt.Sprintf("error updating transaction %d: %s", item.TransactionID, err.Error()), http.StatusInternalServerError)
				return
			}
			updated++
		}

		resp := map[string]int{"updated": updated}
		respJSON, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}
```

Ensure `"errors"` is in the import block of `reapply.go`. Also note: `accounting.ErrValidation` is imported via `"github.com/andresbott/etna/internal/accounting"`. The `errors` package is from the standard library.

Also remove the `"math"` import if the linter complains (only needed in preview, not submit).

**Step 2: Run the test**

Run: `go test ./app/router/handlers/csvimport/... -run TestReapplySubmit -v -count=1`
Expected: PASS

**Step 3: Run all tests**

Run: `go test ./... -count=1 2>&1 | tail -20`
Expected: All PASS

**Step 4: Run lint**

Run: `make lint`
Expected: 0 issues

**Step 5: Commit**

```bash
git add app/router/handlers/csvimport/reapply.go app/router/handlers/csvimport/reapply_test.go
git commit -m "$(cat <<'EOF'
feat: add reapply-submit endpoint for category rule re-application

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: Register new routes

**Files:**
- Modify: `app/router/api_v0.go:710-737`

**Step 1: Add route constants and handler registration**

In `app/router/api_v0.go`, add two new route constants after line 611 (near the other import constants):

```go
const importReapplyPreviewPath = "/import/reapply-preview"
const importReapplySubmitPath = "/import/reapply-submit"
```

Then in the `csvImportAPI` function, after the CSV Parse & Submit section (after line 736), add:

```go
	// ==========================================================================
	// Re-apply Category Rules
	// ==========================================================================

	r.Path(importReapplyPreviewPath).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		importHndlr.ReapplyPreview().ServeHTTP(w, r)
	})

	r.Path(importReapplySubmitPath).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		importHndlr.ReapplySubmit().ServeHTTP(w, r)
	})
```

**Step 2: Run tests**

Run: `go test ./app/router/... -count=1 2>&1 | tail -10`
Expected: PASS

**Step 3: Run lint**

Run: `make lint`
Expected: 0 issues

**Step 4: Commit**

```bash
git add app/router/api_v0.go
git commit -m "$(cat <<'EOF'
feat: register reapply-preview and reapply-submit routes

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

### Task 7: Add frontend API client functions and types

**Files:**
- Modify: `webui/src/lib/api/CsvImport.ts`
- Modify: `webui/src/types/csvimport.ts`

**Step 1: Add the `ReapplyRow` type**

Append to `webui/src/types/csvimport.ts`:

```typescript
export interface ReapplyRow {
  transactionId: number
  transactionType: 'income' | 'expense'
  description: string
  date: string
  amount: number
  accountId: number
  accountName: string
  currentCategoryId: number
  currentCategoryName: string
  newCategoryId: number
  newCategoryName: string
  changed: boolean
}

export interface ReapplySubmitItem {
  transactionId: number
  transactionType: 'income' | 'expense'
  newCategoryId: number
}
```

**Step 2: Add API functions**

Append to `webui/src/lib/api/CsvImport.ts`:

```typescript
import type { ImportProfile, CategoryRule, ParsedRow, PreviewResult, ReapplyRow, ReapplySubmitItem } from '@/types/csvimport'
```

Update the existing import line at the top to include the new types. Then append at the bottom:

```typescript
// Reapply category rules
export const reapplyPreview = () =>
  apiClient.post<ReapplyRow[]>('/import/reapply-preview').then(r => r.data)

export const reapplySubmit = (items: ReapplySubmitItem[]) =>
  apiClient.post<{ updated: number }>('/import/reapply-submit', items).then(r => r.data)
```

**Step 3: Verify frontend builds**

Run: `cd webui && npm run build 2>&1 | tail -5`
Expected: Build succeeds (or only unrelated warnings)

**Step 4: Commit**

```bash
git add webui/src/types/csvimport.ts webui/src/lib/api/CsvImport.ts
git commit -m "$(cat <<'EOF'
feat: add reapply API client functions and types

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

### Task 8: Create `ReapplyRulesView.vue`

**Files:**
- Create: `webui/src/views/csvimport/ReapplyRulesView.vue`

**Step 1: Create the view component**

Create `webui/src/views/csvimport/ReapplyRulesView.vue`:

```vue
<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'

import Button from 'primevue/button'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Card from 'primevue/card'
import Checkbox from 'primevue/checkbox'
import ProgressSpinner from 'primevue/progressspinner'

import { reapplyPreview, reapplySubmit } from '@/lib/api/CsvImport'
import { useDateFormat } from '@/composables/useDateFormat'
import { getEntryTypeIcon } from '@/utils/entryDisplay'
import { getApiErrorMessage } from '@/utils/apiError'

const router = useRouter()
const toast = useToast()
const { formatDate } = useDateFormat()

const formatAmount = (n) =>
    n != null && !Number.isNaN(n)
        ? n.toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
        : '0.00'

/* --- State --- */
const rows = ref(null) // null = loading, array = loaded
const isLoading = ref(false)
const isSubmitting = ref(false)
const checkedRows = ref({}) // transactionId -> boolean

/* --- Summary --- */
const summary = computed(() => {
    if (!rows.value) return { changedCount: 0, unchangedCount: 0 }
    let changedCount = 0
    let unchangedCount = 0
    for (const row of rows.value) {
        if (row.changed) changedCount++
        else unchangedCount++
    }
    return { changedCount, unchangedCount }
})

/* --- Checked count --- */
const checkedCount = computed(() => {
    if (!rows.value) return 0
    return rows.value.filter((r) => checkedRows.value[r.transactionId]).length
})

/* --- Row class --- */
const getRowClass = (data) => ({
    'expense-row': data.transactionType === 'expense',
    'income-row': data.transactionType === 'income',
    'unchanged-row': !data.changed
})

/* --- Load preview --- */
const loadPreview = async () => {
    isLoading.value = true
    try {
        const result = await reapplyPreview()
        rows.value = result
        // Initialize checked state: changed=true checked, changed=false unchecked
        const checked = {}
        for (const row of result) {
            checked[row.transactionId] = row.changed
        }
        checkedRows.value = checked
    } catch (err) {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to load preview: ' + getApiErrorMessage(err), life: 5000 })
        rows.value = []
    } finally {
        isLoading.value = false
    }
}

/* --- Submit --- */
const handleSubmit = async () => {
    if (!rows.value) return
    const selected = rows.value
        .filter((r) => checkedRows.value[r.transactionId])
        .map((r) => ({
            transactionId: r.transactionId,
            transactionType: r.transactionType,
            newCategoryId: r.newCategoryId
        }))

    if (selected.length === 0) {
        toast.add({ severity: 'warn', summary: 'No rows selected', detail: 'Select at least one transaction to update.', life: 3000 })
        return
    }

    isSubmitting.value = true
    try {
        const result = await reapplySubmit(selected)
        toast.add({
            severity: 'success',
            summary: 'Categories updated',
            detail: `${result.updated} transactions updated successfully.`,
            life: 4000
        })
        router.push('/setup/category-rules')
    } catch (err) {
        toast.add({
            severity: 'error',
            summary: 'Update failed',
            detail: getApiErrorMessage(err),
            life: 5000
        })
    } finally {
        isSubmitting.value = false
    }
}

/* --- Navigation --- */
const handleBack = () => {
    router.push('/setup/category-rules')
}

onMounted(() => {
    loadPreview()
})
</script>

<template>
    <div class="main-app-content">
        <div class="reapply-content">
            <!-- Header -->
            <div class="toolbar">
                <div class="toolbar-left">
                    <Button icon="pi pi-arrow-left" text rounded @click="handleBack" v-tooltip.bottom="'Back to rules'" class="mr-2" />
                    <h2 class="page-title">Re-apply Category Rules</h2>
                </div>
            </div>

            <!-- Loading -->
            <div v-if="isLoading" class="loading-section">
                <ProgressSpinner />
                <p>Analyzing transactions...</p>
            </div>

            <!-- Results -->
            <div v-else-if="rows" class="preview-section">
                <!-- Summary Bar -->
                <div class="summary-bar">
                    <span class="summary-item summary-changed">
                        <i class="pi pi-sync"></i> {{ summary.changedCount }} to update
                    </span>
                    <span class="summary-item summary-unchanged">
                        <i class="pi pi-check-circle"></i> {{ summary.unchangedCount }} already correct
                    </span>
                </div>

                <!-- Preview Table -->
                <Card>
                    <template #content>
                        <DataTable
                            class="datatable-compact"
                            :value="rows"
                            stripedRows
                            style="width: 100%"
                            :rowClass="getRowClass"
                            :paginator="rows.length > 25"
                            :rows="25"
                        >
                            <template #empty>
                                <div class="empty-state">
                                    <i class="pi pi-check-circle"></i>
                                    <p>No transactions match any category rules</p>
                                </div>
                            </template>

                            <!-- Checkbox column -->
                            <Column header="" style="width: 50px">
                                <template #body="{ data }">
                                    <Checkbox
                                        v-model="checkedRows[data.transactionId]"
                                        :binary="true"
                                    />
                                </template>
                            </Column>

                            <!-- Type icon -->
                            <Column header="" style="width: 40px">
                                <template #body="{ data }">
                                    <i :class="getEntryTypeIcon(data.transactionType)" style="font-size: 0.8rem" />
                                </template>
                            </Column>

                            <!-- Description -->
                            <Column field="description" header="Description" />

                            <!-- Date -->
                            <Column field="date" header="Date" style="width: 120px">
                                <template #body="{ data }">
                                    {{ formatDate(data.date) }}
                                </template>
                            </Column>

                            <!-- Amount -->
                            <Column field="amount" header="Amount" bodyStyle="text-align: right" style="width: 120px">
                                <template #body="{ data }">
                                    <div class="amount" :class="data.transactionType === 'expense' ? 'expense' : 'income'">
                                        <template v-if="data.transactionType === 'expense'">-</template>
                                        <template v-else>+</template>
                                        {{ formatAmount(data.amount) }}
                                    </div>
                                </template>
                            </Column>

                            <!-- Account -->
                            <Column field="accountName" header="Account" style="width: 150px" />

                            <!-- Current Category -->
                            <Column header="Current Category" style="width: 180px">
                                <template #body="{ data }">
                                    {{ data.currentCategoryName || '—' }}
                                </template>
                            </Column>

                            <!-- New Category -->
                            <Column header="New Category" style="width: 180px">
                                <template #body="{ data }">
                                    <span :class="{ 'category-changed': data.changed }">
                                        {{ data.newCategoryName || '—' }}
                                    </span>
                                </template>
                            </Column>
                        </DataTable>
                    </template>
                </Card>

                <!-- Action buttons -->
                <div class="preview-actions">
                    <Button
                        :label="`Apply Selected (${checkedCount})`"
                        icon="pi pi-check"
                        :loading="isSubmitting"
                        :disabled="checkedCount === 0"
                        @click="handleSubmit"
                    />
                    <Button
                        label="Cancel"
                        severity="secondary"
                        icon="pi pi-times"
                        @click="handleBack"
                    />
                </div>
            </div>
        </div>
    </div>
</template>

<style scoped>
.main-app-content {
    display: flex;
    flex-direction: column;
    height: 100%;
}

.reapply-content {
    display: flex;
    flex-direction: column;
    flex: 1;
    overflow: auto;
}

.toolbar {
    display: flex;
    align-items: center;
    padding: 1rem;
    background-color: var(--surface-ground);
    border-bottom: 1px solid var(--surface-border);
}

.toolbar-left {
    display: flex;
    align-items: center;
}

.page-title {
    margin: 0;
    font-size: 1.5rem;
    font-weight: 600;
    color: var(--c-primary-700);
}

.loading-section {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 4rem 2rem;
    color: var(--text-color-secondary);
}

.preview-section {
    padding: 1rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
}

.summary-bar {
    display: flex;
    gap: 1.5rem;
    padding: 0.75rem 1rem;
    background-color: var(--surface-card);
    border: 1px solid var(--surface-border);
    border-radius: var(--border-radius);
    font-weight: 600;
}

.summary-item {
    display: flex;
    align-items: center;
    gap: 0.4rem;
}

.summary-changed {
    color: var(--blue-600);
}

.summary-unchanged {
    color: var(--green-600);
}

.preview-actions {
    display: flex;
    gap: 0.75rem;
    padding-top: 0.5rem;
}

.amount.expense {
    color: var(--red-500);
}

.amount.income {
    color: var(--green-500);
}

.category-changed {
    font-weight: 600;
    color: var(--blue-600);
}

.empty-state {
    text-align: center;
    padding: 3rem 1rem;
    color: var(--text-color-secondary);
}

.empty-state i {
    font-size: 3rem;
    margin-bottom: 1rem;
    opacity: 0.5;
}

:deep(.unchanged-row) {
    opacity: 0.6;
}
</style>
```

**Step 2: Verify frontend builds**

Run: `cd webui && npm run build 2>&1 | tail -5`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add webui/src/views/csvimport/ReapplyRulesView.vue
git commit -m "$(cat <<'EOF'
feat: add ReapplyRulesView component for category rule re-application

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

### Task 9: Add route and button to CategoryRulesView

**Files:**
- Modify: `webui/src/router/index.js` (add route around line 238, before the category-rules route)
- Modify: `webui/src/views/csvimport/CategoryRulesView.vue` (add button)

**Step 1: Add the route**

In `webui/src/router/index.js`, add a new route before the catch-all 404 route (around line 238):

```javascript
        {
            path: '/setup/reapply-rules',
            name: 'reapply-rules',
            meta: {
                requiresAuth: true
            },
            component: () => import('@/views/csvimport/ReapplyRulesView.vue')
        },
```

**Step 2: Add the button to CategoryRulesView**

In `webui/src/views/csvimport/CategoryRulesView.vue`, add `useRouter` import and the button.

Add to the `<script setup>` section, after the existing imports:

```javascript
import { useRouter } from 'vue-router'
```

And after `const toast = useToast()`:

```javascript
const router = useRouter()
```

In the template, change the button area (around line 179) from:

```html
                    <Button
                        label="New Rule"
                        icon="pi pi-plus"
                        @click="openCreateDialog"
                    />
```

To:

```html
                    <div class="flex gap-2">
                        <Button
                            label="Re-apply Rules"
                            icon="pi pi-sync"
                            severity="secondary"
                            @click="router.push('/setup/reapply-rules')"
                        />
                        <Button
                            label="New Rule"
                            icon="pi pi-plus"
                            @click="openCreateDialog"
                        />
                    </div>
```

**Step 3: Verify frontend builds**

Run: `cd webui && npm run build 2>&1 | tail -5`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add webui/src/router/index.js webui/src/views/csvimport/CategoryRulesView.vue
git commit -m "$(cat <<'EOF'
feat: add reapply-rules route and button on category rules page

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

### Task 10: Final verification

**Step 1: Run all backend tests**

Run: `go test ./... -count=1 2>&1 | tail -20`
Expected: All PASS

**Step 2: Run lint**

Run: `make lint`
Expected: 0 issues

**Step 3: Run frontend build**

Run: `cd webui && npm run build 2>&1 | tail -5`
Expected: Build succeeds

**Step 4: Manual smoke test (optional)**

If the app can be started locally:
1. Navigate to `/setup/category-rules`
2. Verify "Re-apply Rules" button appears
3. Click it — should navigate to `/setup/reapply-rules`
4. Should show loading spinner, then a table of matching transactions
5. Select some, click "Apply Selected"
6. Should show success toast and navigate back
