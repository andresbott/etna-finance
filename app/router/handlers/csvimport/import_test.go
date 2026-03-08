package csvimport

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/andresbott/etna/internal/accounting"
	"github.com/andresbott/etna/internal/marketdata"
	"github.com/glebarez/sqlite"
	"golang.org/x/text/currency"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestLoadExistingTransactions_PaginatesAll(t *testing.T) {
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
	defer func() { _ = uDb.Close() }()

	mktStore, err := marketdata.NewStore(db)
	if err != nil {
		t.Fatalf("unable to create marketdata store: %v", err)
	}
	store, err := accounting.NewStore(db, mktStore)
	if err != nil {
		t.Fatalf("unable to create accounting store: %v", err)
	}

	ctx := context.Background()

	// Create a provider and account
	providerID, err := store.CreateAccountProvider(ctx, accounting.AccountProvider{Name: "test", Description: "test", Icon: "bank"})
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	accID, err := store.CreateAccount(ctx, accounting.Account{
		Name:              "test-acc",
		Currency:          currency.CHF,
		Type:              accounting.CashAccountType,
		AccountProviderID: providerID,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}

	// Create a category
	catID, err := store.CreateCategory(ctx, accounting.CategoryData{Name: "Food", Icon: "food", Type: accounting.ExpenseCategory}, 0)
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	// Insert more than MaxSearchResults (300) transactions
	totalTx := accounting.MaxSearchResults + 50
	baseDate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < totalTx; i++ {
		tx := accounting.Expense{
			Description: fmt.Sprintf("expense-%d", i),
			Amount:      float64(i + 1),
			AccountID:   accID,
			CategoryID:  catID,
			Date:        baseDate.AddDate(0, 0, i),
		}
		if _, err := store.CreateTransaction(ctx, tx); err != nil {
			t.Fatalf("create tx %d: %v", i, err)
		}
	}

	h := &ImportHandler{FinStore: store}
	req, _ := http.NewRequest("GET", "/", nil)
	existing, err := h.loadExistingTransactions(req, accID)
	if err != nil {
		t.Fatalf("loadExistingTransactions: %v", err)
	}

	if len(existing) != totalTx {
		t.Errorf("expected %d existing transactions, got %d", totalTx, len(existing))
	}
}
