package e2e

import (
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
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
	t.Run("Edit Entries", func(t *testing.T) {
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
		page = page.Timeout(60 * time.Second)

		// clickEdit finds the description row and clicks the edit (pencil) button.
		clickEdit := func(desc string) {
			row := page.MustElementX(fmt.Sprintf("//span[text()='%s']/ancestor::tr[1]", desc))
			row.MustElement(".pi-pencil").MustClick()
		}

		// openDialog opens the edit dialog for desc and waits until it is ready.
		openDialog := func(desc string) *rod.Element {
			clickEdit(desc)
			dialog := page.MustElement(".entry-dialog")
			dialog.MustElement("input[name='description']").MustWaitInteractable()
			return dialog
		}

		// fieldVal reads the value property of the named input inside the dialog.
		fieldVal := func(dialog *rod.Element, name string) string {
			return strings.TrimSpace(
				dialog.MustElement("input[name='" + name + "']").MustProperty("value").Str(),
			)
		}

		// parseAmt converts a locale-formatted amount string (e.g. "6,000.00" or
		// "6.000,00") to float64. Handles en-US, de-DE and plain formats.
		parseAmt := func(s string) float64 {
			lastDot := strings.LastIndex(s, ".")
			lastComma := strings.LastIndex(s, ",")
			if lastDot > lastComma {
				s = strings.ReplaceAll(s, ",", "")
			} else if lastComma > lastDot {
				s = strings.ReplaceAll(s, ".", "")
				s = strings.ReplaceAll(s, ",", ".")
			}
			v, _ := strconv.ParseFloat(s, 64)
			return v
		}

		// treeLabel reads the display text of a TreeSelect widget.
		// cssSelector scopes to the widget's container (e.g. ".account-select").
		treeLabel := func(ctx *rod.Element, cssSelector string) string {
			return strings.TrimSpace(ctx.MustElement(cssSelector + " .p-treeselect-label").MustText())
		}

		// saveDialog clicks Save and waits for the dialog to close.
		saveDialog := func(t *testing.T, dialog *rod.Element) {
			t.Helper()
			descInput := dialog.MustElement("input[name='description']")
			page.MustSearch("Save").MustClick()
			etna.WaitDialogClose(t, descInput, "save edit dialog")
			time.Sleep(400 * time.Millisecond) // allow query invalidation to settle
		}

		// cancelDialog clicks Cancel and waits for the dialog to close.
		cancelDialog := func(t *testing.T, dialog *rod.Element) {
			t.Helper()
			descInput := dialog.MustElement("input[name='description']")
			dialog.MustElementX(".//button[contains(.,'Cancel')]").MustClick()
			etna.WaitDialogClose(t, descInput, "cancel edit dialog")
		}

		// ── Income: Salary Feb ────────────────────────────────────────────────
		// Original: 6000, Alpha Checking (CHF), Salary / Employment
		// Edit: amount 6000 → 6500
		t.Run("edit income", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v\n%s", r, debug.Stack())
				}
			}()
			dialog := openDialog("Salary Feb")

			if got := fieldVal(dialog, "description"); got != "Salary Feb" {
				t.Errorf("income desc: got %q, want %q", got, "Salary Feb")
			}
			if got := parseAmt(fieldVal(dialog, "amount")); got != 6000 {
				t.Errorf("income amount: got %g, want 6000", got)
			}
			if label := treeLabel(dialog, ".account-select"); !strings.Contains(label, "Alpha Checking") {
				t.Errorf("income account: got %q, expected to contain 'Alpha Checking'", label)
			}
			// Category: look for the expected text anywhere in the category field.
			// rod.Try catches the MustElementX panic when the text is absent (e.g. shows
			// "Root Category" fallback because categoryId prop is not bound in edit dialog).
			if err := rod.Try(func() {
				dialog.MustElementX(".//label[text()='Category']/..//*[contains(text(),'Salary')]")
			}); err != nil {
				t.Errorf("income category: 'Salary' not found in category selector — edit dialog shows wrong category (root fallback?)")
			}

			etna.TypeAmount(page, dialog.MustElement("input[name='amount']"), "6500")
			saveDialog(t, dialog)

			dialog = openDialog("Salary Feb")
			if got := parseAmt(fieldVal(dialog, "amount")); got != 6500 {
				t.Errorf("income amount after edit: got %g, want 6500", got)
			}
			if label := treeLabel(dialog, ".account-select"); !strings.Contains(label, "Alpha Checking") {
				t.Errorf("income account after edit: got %q", label)
			}
			cancelDialog(t, dialog)
		})

		// ── Expense: Utilities ────────────────────────────────────────────────
		// Original: 200, Beta Checking (CHF), Utilities / Housing
		// Edit: amount 200 → 150
		t.Run("edit expense", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v\n%s", r, debug.Stack())
				}
			}()
			dialog := openDialog("Utilities")

			if got := fieldVal(dialog, "description"); got != "Utilities" {
				t.Errorf("expense desc: got %q, want %q", got, "Utilities")
			}
			if got := parseAmt(fieldVal(dialog, "amount")); got != 200 {
				t.Errorf("expense amount: got %g, want 200", got)
			}
			if label := treeLabel(dialog, ".account-select"); !strings.Contains(label, "Beta Checking") {
				t.Errorf("expense account: got %q, expected to contain 'Beta Checking'", label)
			}
			if err := rod.Try(func() {
				dialog.MustElementX(".//label[text()='Category']/..//*[contains(text(),'Utilities')]")
			}); err != nil {
				t.Errorf("expense category: 'Utilities' not found in category selector — edit dialog shows wrong category (root fallback?)")
			}

			etna.TypeAmount(page, dialog.MustElement("input[name='amount']"), "150")
			saveDialog(t, dialog)

			dialog = openDialog("Utilities")
			if got := parseAmt(fieldVal(dialog, "amount")); got != 150 {
				t.Errorf("expense amount after edit: got %g, want 150", got)
			}
			cancelDialog(t, dialog)
		})

		// ── Transfer: Savings deposit ─────────────────────────────────────────
		// Original: 500, Alpha Checking → Alpha Savings
		// Edit: amounts 500 → 600
		t.Run("edit transfer", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v\n%s", r, debug.Stack())
				}
			}()
			dialog := openDialog("Savings deposit")

			if got := fieldVal(dialog, "description"); got != "Savings deposit" {
				t.Errorf("transfer desc: got %q, want %q", got, "Savings deposit")
			}
			if got := parseAmt(fieldVal(dialog, "originAmount")); got != 500 {
				t.Errorf("transfer originAmount: got %g, want 500", got)
			}
			if got := parseAmt(fieldVal(dialog, "targetAmount")); got != 500 {
				t.Errorf("transfer targetAmount: got %g, want 500", got)
			}
			// Two AccountSelectors in order: origin (From), target (To)
			accts := dialog.MustElements(".account-select")
			originLabel := strings.TrimSpace(accts[0].MustElement(".p-treeselect-label").MustText())
			targetLabel := strings.TrimSpace(accts[1].MustElement(".p-treeselect-label").MustText())
			if !strings.Contains(originLabel, "Alpha Checking") {
				t.Errorf("transfer origin account: got %q, expected to contain 'Alpha Checking'", originLabel)
			}
			if !strings.Contains(targetLabel, "Alpha Savings") {
				t.Errorf("transfer target account: got %q, expected to contain 'Alpha Savings'", targetLabel)
			}

			etna.TypeAmount(page, dialog.MustElement("input[name='originAmount']"), "600")
			etna.TypeAmount(page, dialog.MustElement("input[name='targetAmount']"), "600")
			saveDialog(t, dialog)

			dialog = openDialog("Savings deposit")
			if got := parseAmt(fieldVal(dialog, "originAmount")); got != 600 {
				t.Errorf("transfer originAmount after edit: got %g, want 600", got)
			}
			if got := parseAmt(fieldVal(dialog, "targetAmount")); got != 600 {
				t.Errorf("transfer targetAmount after edit: got %g, want 600", got)
			}
			cancelDialog(t, dialog)
		})
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
		// After edits: Salary Feb 6000→6500, Utilities 200→150, Savings deposit 500→600
		checkAccountType("Cash", "114.50")
		checkAccountType("Checking", "9,350.00")
		checkAccountType("Savings", "600.00")

		totalText := strings.TrimSpace(page.MustElement(".total-value").MustText())
		if !strings.Contains(totalText, "10,064.50") {
			t.Errorf("Total (excl. unvested): got %q, want substring %q", totalText, "10,064.50")
		}

		// --- Verify Cash Accounts card ---
		// Wait for balance report to load (async fetch — card shows providers only after data arrives)
		page.MustElementX("//span[text()='Bank Alpha']")
		cashCard := page.MustElementX("//*[text()='Cash Accounts']/ancestor::*[@data-pc-name='card'][1]")

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
		checkBalance("Alpha Checking", "7,000.00")
		checkBalance("Alpha Savings", "600.00")
		checkBalance("Beta Wallet", "0.00")
		checkBalance("Beta Checking", "2,350.00")
	})

	t.Run("validate income expense report", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic: %v\n%s", r, debug.Stack())
			}
		}()
		page, err = nav.Navigate(etna.GetURL(inst.BaseURL, "/reports/income-expense"))
		if err != nil {
			t.Fatalf("navigate to /reports/income-expense: %v", err)
		}
		page.MustWaitLoad()
		page = page.Timeout(90 * time.Second)

		now := time.Now()
		fmtDate := func(d time.Time) string { return d.Format("2006-01-02") }

		// Re-find inputs each time to avoid stale-element issues after Vue re-renders.
		startInput := func() *rod.Element { return page.MustElement("input[placeholder='Start date']") }
		endInput := func() *rod.Element { return page.MustElement("input[placeholder='End date']") }

		// getRowCHF finds the category row within the named report section ("Income" or
		// "Expenses") and returns the text of its CHF amount cell.
		// Uses a short per-call timeout so a missing row fails immediately with a clear
		// error rather than blocking for the full page context budget.
		getRowCHF := func(section, categoryName string) string {
			var result string
			if err := rod.Try(func() {
				p := page.Timeout(5 * time.Second)
				nameTd := p.MustElementX(fmt.Sprintf(
					"//div[contains(@class,'report-section') and .//h2[.='%s']]//td[.//span[contains(@class,'category-name') and contains(.,'%s')]]",
					section, categoryName,
				))
				amountTd := nameTd.MustElementX("./following-sibling::td[contains(@class,'amount-column')]")
				result = strings.TrimSpace(amountTd.MustText())
			}); err != nil {
				t.Errorf("report %s/%s: row not found (category missing or data not loaded for this range)", section, categoryName)
			}
			return result
		}

		// ── Range 1: last 30 days ─────────────────────────────────────────────
		// Covers Day 4 (today-19d) and Day 5 (today-9d) only.
		//   Income:  Employment 6,000.00  (Salary Feb)
		//   Expense: Housing    1,200.00  (Rent Feb)
		//            Food          85.50  (Dining out)
		// Note: Transport, Utilities, Groceries are from days 2-3 (today-43d / today-35d)
		// and fall outside this range — only Housing and Food appear.
		etna.TypeDate(page, startInput(), fmtDate(now.AddDate(0, 0, -30)))
		etna.TypeDate(page, endInput(), fmtDate(now))
		// Wait for the data to refresh. Housing changes from its default-range value
		// (includes Utilities) to 1,200.00 (Rent Feb only). A short sleep is sufficient
		// for a local backend.
		time.Sleep(800 * time.Millisecond)

		if got := getRowCHF("Income", "Employment"); got != "6,500.00" {
			t.Errorf("30d income Employment: got %q, want 6,500.00", got)
		}
		if got := getRowCHF("Expenses", "Housing"); got != "1,200.00" {
			t.Errorf("30d expense Housing: got %q, want 1,200.00", got)
		}
		if got := getRowCHF("Expenses", "Food"); got != "85.50" {
			t.Errorf("30d expense Food: got %q, want 85.50", got)
		}

		// ── Range 2: last 1 year ──────────────────────────────────────────────
		// Covers all 5 days.
		//   Income:  Employment  11,000.00  (Salary Jan 5,000 + Salary Feb 6,000)
		//   Expense: Housing      2,600.00  (Rent Jan 1,200 + Utilities 200 + Rent Feb 1,200)
		//            Food           435.50  (Groceries 350 + Dining 85.50)
		//            Transport      150.00
		etna.TypeDate(page, startInput(), fmtDate(now.AddDate(-1, 0, 0)))
		etna.TypeDate(page, endInput(), fmtDate(now))
		// Transport only appears in the 1-year range (Day 3, today-35d). Waiting for its
		// row to appear confirms the new data has loaded before we assert.
		page.MustElementX("//span[contains(@class,'category-name') and contains(.,'Transport')]")

		if got := getRowCHF("Income", "Employment"); got != "11,500.00" {
			t.Errorf("1yr income Employment: got %q, want 11,500.00", got)
		}
		if got := getRowCHF("Expenses", "Housing"); got != "2,550.00" {
			t.Errorf("1yr expense Housing: got %q, want 2,550.00", got)
		}
		if got := getRowCHF("Expenses", "Food"); got != "435.50" {
			t.Errorf("1yr expense Food: got %q, want 435.50", got)
		}
		if got := getRowCHF("Expenses", "Transport"); got != "150.00" {
			t.Errorf("1yr expense Transport: got %q, want 150.00", got)
		}
	})

	// manual review pause — keeps the browser open for inspection.
	// Only active when running non-headless with WAIT=true:
	//   E2E=true HEADLESS=false WAIT=true go test -v
	t.Run("manual review pause", func(t *testing.T) {
		headless := os.Getenv("HEADLESS") != "false" && os.Getenv("HEADLESS") != "0"
		wait := os.Getenv("WAIT") == "true" || os.Getenv("WAIT") == "1"
		if headless || !wait {
			t.Skip("skipped (set HEADLESS=false WAIT=true to pause for manual review)")
		}
		t.Log("pausing for 1h — review the browser, then kill the test to stop")
		time.Sleep(time.Hour)
	})
}
