package e2e

import (
	"fmt"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/andresbott/etna/zarf/e2e/etna"
	"github.com/andresbott/etna/zarf/e2e/instance"
	"github.com/go-rod/rod"
)

func TestOperateBasic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic: %v\n%s", r, debug.Stack())
		}
	}()
	cfg := instance.DefaultEnvCfg()
	cfg.Settings.Instruments = true
	inst, nav := SetupE2E(t, &cfg)
	var page *rod.Page
	var err error

	t.Run("Configure accounts", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic: %v\n%s", r, debug.Stack())
			}
		}()
		page, err = nav.Navigate(etna.GetURL(inst.BaseURL, "/accounts"))
		if err != nil {
			t.Fatalf("navigate to /accounts: %v", err)
		}
		page.MustWaitLoad()

		// Use a strict timeout so failures fail fast instead of hanging
		page = page.Timeout(25 * time.Second)

		// Create 2 account providers via the UI
		for _, name := range []string{"Bank Alpha", "Bank Beta"} {
			etna.CreateProviderViaUI(t, page, name, "E2E provider "+name)
		}

		// Under Bank Alpha, create one account of each type: cash, checking, savings
		for _, spec := range []struct {
			accName   string
			typeLabel string
		}{
			{"Wallet", "Cash"},
			{"Checking", "Checking"},
			{"Savings", "Savings"},
		} {
			etna.CreateAccountViaUI(t, page, "Bank Alpha", "Alpha "+spec.accName, spec.typeLabel, "CHF")
		}

		// Under Bank Beta, create cash + checking
		for _, spec := range []struct {
			accName   string
			typeLabel string
		}{
			{"Wallet", "Cash"},
			{"Checking", "Checking"},
		} {
			etna.CreateAccountViaUI(t, page, "Bank Beta", "Beta "+spec.accName, spec.typeLabel, "CHF")
		}
	})

	t.Run("Configure categories", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic: %v\n%s", r, debug.Stack())
			}
		}()
		page, err = nav.Navigate(etna.GetURL(inst.BaseURL, "/categories"))
		if err != nil {
			t.Fatalf("navigate to /categories: %v", err)
		}
		page.MustWaitLoad()
		page = page.Timeout(30 * time.Second)

		// Expense tab is active by default — create parent categories
		etna.CreateParentCategoryViaUI(t, page, "Food", "Food expenses")
		etna.CreateParentCategoryViaUI(t, page, "Transport", "Transport expenses")
		etna.CreateParentCategoryViaUI(t, page, "Housing", "Housing expenses")

		// Create subcategories
		etna.CreateSubCategoryViaUI(t, page, "Food", "Groceries", "Grocery shopping")
		etna.CreateSubCategoryViaUI(t, page, "Food", "Dining", "Dining out")
		etna.CreateSubCategoryViaUI(t, page, "Housing", "Rent", "Monthly rent")
		etna.CreateSubCategoryViaUI(t, page, "Housing", "Utilities", "Utility bills")

		// Switch to Income tab (PrimeVue TabView renders headers as role="tab")
		page.MustElementX("//*[@role='tab' and contains(., 'Income Categories')]").MustClick()
		time.Sleep(300 * time.Millisecond) // let tab transition settle

		// Create parent income category
		etna.CreateParentCategoryViaUI(t, page, "Employment", "Employment income")

		// Create subcategory under "Employment"
		etna.CreateSubCategoryViaUI(t, page, "Employment", "Salary", "Monthly salary")
	})

	t.Run("Create Transactions", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic: %v\n%s", r, debug.Stack())
			}
		}()
		page, err = nav.Navigate(etna.GetURL(inst.BaseURL, "/entries"))
		if err != nil {
			t.Fatalf("navigate to /entries: %v", err)
		}
		page.MustWaitLoad()

		// Use a strict timeout so failures fail fast instead of hanging
		page = page.Timeout(180 * time.Second)

		// Transactions are spread across 5 different days (relative to today).
		// Ordering ensures no account balance ever goes negative.
		//
		// Day 1 (today-50d): Initial income
		//   Alpha Checking: +5000 (Salary Jan)        → 5,000
		//   Beta Checking:  +1500 (Freelance)         → 1,500
		//   Alpha Wallet:   +200  (Cash gift)         →   200
		//
		// Day 2 (today-43d): Transfer in + first expenses
		//   Alpha Checking → Alpha Wallet: 500        → AC 4,500 / AW 700
		//   Alpha Checking: -1200 (Rent Jan)          → AC 3,300
		//   Alpha Wallet:   -350  (Groceries)         → AW 350
		//
		// Day 3 (today-35d): Savings + more expenses
		//   Alpha Checking → Alpha Savings: 500       → AC 2,800 / AS 500
		//   Alpha Wallet:   -150  (Transport)         → AW 200
		//   Beta Checking:  -200  (Utilities)         → BC 1,300
		//
		// Day 4 (today-19d): Second salary + small expense
		//   Alpha Checking: +6000 (Salary Feb)        → AC 8,800
		//   Alpha Wallet:   -85.50 (Dining out)       → AW 114.50
		//
		// Day 5 (today-9d): Rent + cross-bank transfer
		//   Alpha Checking: -1200 (Rent Feb)          → AC 7,600
		//   Alpha Checking → Beta Checking: 1000      → AC 6,600 / BC 2,300
		today := time.Now()
		fmtDate := func(d time.Time) string { return d.Format("2006-01-02") }
		day1 := fmtDate(today.AddDate(0, 0, -50))
		day2 := fmtDate(today.AddDate(0, 0, -43))
		day3 := fmtDate(today.AddDate(0, 0, -35))
		day4 := fmtDate(today.AddDate(0, 0, -19))
		day5 := fmtDate(today.AddDate(0, 0, -9))

		// --- Day 1 — Income ---
		etna.CreateIncomeViaUI(t, page, "Salary Jan", "5000", "Alpha Checking (CHF)", day1, "Salary", "Employment")
		etna.CreateIncomeViaUI(t, page, "Freelance payment", "1500", "Beta Checking (CHF)", day1, "", "")
		etna.CreateIncomeViaUI(t, page, "Cash gift", "200", "Alpha Wallet (CHF)", day1, "", "")

		// --- Day 2 — Transfer + expenses ---
		etna.CreateTransferViaUI(t, page, "Cash withdrawal", "500", "Alpha Checking (CHF)", "Alpha Wallet (CHF)", day2)
		etna.CreateExpenseViaUI(t, page, "Rent Jan", "1200", "Alpha Checking (CHF)", day2, "Rent", "Housing")
		etna.CreateExpenseViaUI(t, page, "Groceries", "350", "Alpha Wallet (CHF)", day2, "Groceries", "Food")

		// --- Day 3 — Savings + expenses ---
		etna.CreateTransferViaUI(t, page, "Savings deposit", "500", "Alpha Checking (CHF)", "Alpha Savings (CHF)", day3)
		etna.CreateExpenseViaUI(t, page, "Transport", "150", "Alpha Wallet (CHF)", day3, "Transport", "")
		etna.CreateExpenseViaUI(t, page, "Utilities", "200", "Beta Checking (CHF)", day3, "Utilities", "Housing")

		// --- Day 4 — Second salary + dining ---
		etna.CreateIncomeViaUI(t, page, "Salary Feb", "6000", "Alpha Checking (CHF)", day4, "Salary", "Employment")
		etna.CreateExpenseViaUI(t, page, "Dining out", "85.50", "Alpha Wallet (CHF)", day4, "Dining", "Food")

		// --- Day 5 — Rent + cross-bank transfer ---
		etna.CreateExpenseViaUI(t, page, "Rent Feb", "1200", "Alpha Checking (CHF)", day5, "Rent", "Housing")
		etna.CreateTransferViaUI(t, page, "Cross-bank transfer", "1000", "Alpha Checking (CHF)", "Beta Checking (CHF)", day5)
	})
	t.Run("validate Dashboards", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic: %v\n%s", r, debug.Stack())
			}
		}()
		page, err = nav.Navigate(etna.GetURL(inst.BaseURL, "/reports/overview"))
		if err != nil {
			t.Fatalf("navigate to /reports/overview: %v", err)
		}
		page.MustWaitLoad()
		page = page.Timeout(30 * time.Second)

		// Wait for Account Types data to load (Cash row appears when ready)
		page.MustElementX("//span[text()='Cash']/ancestor::tr[1]")

		// --- Verify Account Types table ---
		// Expected totals (en-US formatted):
		//   Cash:     114.50
		//   Checking: 8,900.00
		//   Savings:  500.00
		//   Total:    9,514.50 CHF
		checkAccountType := func(typeName, expectedAmount string) {
			row := page.MustElementX(fmt.Sprintf("//span[text()='%s']/ancestor::tr[1]", typeName))
			actual := strings.TrimSpace(row.MustElementX(".//td[last()]//span").MustText())
			if actual != expectedAmount {
				t.Errorf("Account type %q: got %q, want %q", typeName, actual, expectedAmount)
			}
		}
		checkAccountType("Cash", "114.50")
		checkAccountType("Checking", "8,900.00")
		checkAccountType("Savings", "500.00")

		totalText := strings.TrimSpace(page.MustElement(".total-value").MustText())
		if !strings.Contains(totalText, "9,514.50") {
			t.Errorf("Total (excl. unvested): got %q, want substring %q", totalText, "9,514.50")
		}

		// --- Verify Cash Accounts card ---
		cashCard := page.MustElementX("//*[text()='Cash Accounts']/ancestor::*[contains(@class, 'p-card')][1]")

		// Verify provider names are present
		cashCard.MustElementX(".//span[text()='Bank Alpha']")
		cashCard.MustElementX(".//span[text()='Bank Beta']")

		// Verify individual account balances
		checkBalance := func(accountName, expectedBalance string) {
			accEl := cashCard.MustElementX(fmt.Sprintf(".//span[text()='%s']", accountName))
			row := accEl.MustElementX("./ancestor::div[contains(@class, 'justify-content-between')][1]")
			actual := strings.TrimSpace(row.MustElement(".font-bold").MustText())
			if actual != expectedBalance {
				t.Errorf("Account %q balance: got %q, want %q", accountName, actual, expectedBalance)
			}
		}
		checkBalance("Alpha Wallet", "114.50")
		checkBalance("Alpha Checking", "6,600.00")
		checkBalance("Alpha Savings", "500.00")
		checkBalance("Beta Wallet", "0.00")
		checkBalance("Beta Checking", "2,300.00")
	})

}
