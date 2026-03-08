package csvimport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestReapplyPreview(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("unable to open sqlite: %v", err)
	}
	uDb, _ := db.DB()
	defer func() { _ = uDb.Close() }()

	mktStore, _ := marketdata.NewStore(db)
	finStore, _ := accounting.NewStore(db, mktStore)
	csvStore, _ := csvimport.NewStore(db)

	ctx := context.Background()

	// Create a provider and account
	providerID, err := finStore.CreateAccountProvider(ctx, accounting.AccountProvider{Name: "test", Description: "test", Icon: "bank"})
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	accID, err := finStore.CreateAccount(ctx, accounting.Account{
		Name:              "test-acc",
		Currency:          currency.CHF,
		Type:              accounting.CashAccountType,
		AccountProviderID: providerID,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}

	// Create two expense categories
	foodCatID, err := finStore.CreateCategory(ctx, accounting.CategoryData{Name: "Food", Icon: "food", Type: accounting.ExpenseCategory}, 0)
	if err != nil {
		t.Fatalf("create Food category: %v", err)
	}
	transportCatID, err := finStore.CreateCategory(ctx, accounting.CategoryData{Name: "Transport", Icon: "transport", Type: accounting.ExpenseCategory}, 0)
	if err != nil {
		t.Fatalf("create Transport category: %v", err)
	}

	// Create a category rule: pattern "GROCERY" -> Food category
	_, err = csvStore.CreateCategoryRule(ctx, csvimport.CategoryRule{
		Pattern:    "GROCERY",
		CategoryID: foodCatID,
		Position:   1,
	})
	if err != nil {
		t.Fatalf("create category rule: %v", err)
	}

	baseDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	// Transaction 1: Expense "GROCERY STORE" with no category -> should appear, changed=true
	_, err = finStore.CreateTransaction(ctx, accounting.Expense{
		Description: "GROCERY STORE",
		Amount:      50.00,
		AccountID:   accID,
		CategoryID:  0,
		Date:        baseDate,
	})
	if err != nil {
		t.Fatalf("create tx1: %v", err)
	}

	// Transaction 2: Expense "GROCERY MARKET" with Transport category -> should appear, changed=true
	_, err = finStore.CreateTransaction(ctx, accounting.Expense{
		Description: "GROCERY MARKET",
		Amount:      30.00,
		AccountID:   accID,
		CategoryID:  transportCatID,
		Date:        baseDate.AddDate(0, 0, 1),
	})
	if err != nil {
		t.Fatalf("create tx2: %v", err)
	}

	// Transaction 3: Expense "GROCERY DEPOT" with Food category -> should appear, changed=false
	_, err = finStore.CreateTransaction(ctx, accounting.Expense{
		Description: "GROCERY DEPOT",
		Amount:      20.00,
		AccountID:   accID,
		CategoryID:  foodCatID,
		Date:        baseDate.AddDate(0, 0, 2),
	})
	if err != nil {
		t.Fatalf("create tx3: %v", err)
	}

	// Transaction 4: Expense "RENT PAYMENT" with no category -> should NOT appear (no rule match)
	_, err = finStore.CreateTransaction(ctx, accounting.Expense{
		Description: "RENT PAYMENT",
		Amount:      1000.00,
		AccountID:   accID,
		CategoryID:  0,
		Date:        baseDate.AddDate(0, 0, 3),
	})
	if err != nil {
		t.Fatalf("create tx4: %v", err)
	}

	// Build handler and call ReapplyPreview
	handler := &ImportHandler{
		CsvStore: csvStore,
		FinStore: finStore,
	}

	req := httptest.NewRequest(http.MethodPost, "/reapply-preview", nil)
	w := httptest.NewRecorder()

	handler.ReapplyPreview().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var rows []ReapplyRow
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatalf("unable to unmarshal response: %v", err)
	}

	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	changedCount := 0
	unchangedCount := 0
	for _, row := range rows {
		if row.Changed {
			changedCount++
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
}

func TestReapplySubmit(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("unable to open sqlite: %v", err)
	}
	uDb, _ := db.DB()
	defer func() { _ = uDb.Close() }()

	mktStore, _ := marketdata.NewStore(db)
	finStore, _ := accounting.NewStore(db, mktStore)
	csvStore, _ := csvimport.NewStore(db)

	ctx := context.Background()

	// Create a provider and account
	providerID, err := finStore.CreateAccountProvider(ctx, accounting.AccountProvider{Name: "test", Description: "test", Icon: "bank"})
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	accID, err := finStore.CreateAccount(ctx, accounting.Account{
		Name:              "test-acc",
		Currency:          currency.CHF,
		Type:              accounting.CashAccountType,
		AccountProviderID: providerID,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}

	// Create expense category "Food" and income category "Salary"
	foodCatID, err := finStore.CreateCategory(ctx, accounting.CategoryData{Name: "Food", Icon: "food", Type: accounting.ExpenseCategory}, 0)
	if err != nil {
		t.Fatalf("create Food category: %v", err)
	}
	salaryCatID, err := finStore.CreateCategory(ctx, accounting.CategoryData{Name: "Salary", Icon: "salary", Type: accounting.IncomeCategory}, 0)
	if err != nil {
		t.Fatalf("create Salary category: %v", err)
	}

	// Create one expense transaction with no category
	baseDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	txID, err := finStore.CreateTransaction(ctx, accounting.Expense{
		Description: "GROCERY STORE",
		Amount:      50.00,
		AccountID:   accID,
		CategoryID:  0,
		Date:        baseDate,
	})
	if err != nil {
		t.Fatalf("create expense transaction: %v", err)
	}

	handler := &ImportHandler{CsvStore: csvStore, FinStore: finStore}

	t.Run("successful update", func(t *testing.T) {
		body := fmt.Sprintf(`[{"transactionId":%d,"transactionType":"expense","newCategoryId":%d}]`, txID, foodCatID)
		req := httptest.NewRequest(http.MethodPost, "/reapply-submit", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ReapplySubmit().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]int
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unable to unmarshal response: %v", err)
		}
		if resp["updated"] != 1 {
			t.Errorf("expected updated=1, got %d", resp["updated"])
		}
	})

	t.Run("rejects mismatched category type", func(t *testing.T) {
		body := fmt.Sprintf(`[{"transactionId":%d,"transactionType":"expense","newCategoryId":%d}]`, txID, salaryCatID)
		req := httptest.NewRequest(http.MethodPost, "/reapply-submit", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ReapplySubmit().ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
		}
	})
}
