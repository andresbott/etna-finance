package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod"
)

func getURL(baseurl, path string) string {
	sanitized := strings.TrimPrefix(path, "/")
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(baseurl, "/"), sanitized)
}

func TestOperateBasic(t *testing.T) {
	inst, nav := SetupE2E(t, nil)
	page, err := nav.Navigate(getURL(inst.BaseURL, "/settings"))
	if err != nil {
		t.Fatalf("navigate to root: %v", err)
	}
	page.MustWaitLoad()

	t.Run("Configure accounts", func(t *testing.T) {
		page, err = nav.Navigate(getURL(inst.BaseURL, "/accounts"))
		if err != nil {
			t.Fatalf("navigate to /accounts: %v", err)
		}
		page.MustWaitLoad()

		// Use a strict timeout so failures fail fast instead of hanging
		page = page.Timeout(15 * time.Second)

		// Create 2 account providers via the UI
		for _, name := range []string{"Bank Alpha", "Bank Beta"} {
			createProviderViaUI(t, page, name, "E2E provider "+name)
		}

		// Under each provider, create one account of each type: cash, checking, savings
		accountSpecs := []struct {
			accName   string
			typeLabel string // UI label: Cash, Checking, Savings
		}{
			{"Wallet", "Cash"},
			{"Checking", "Checking"},
			{"Savings", "Savings"},
		}
		for _, providerName := range []string{"Bank Alpha", "Bank Beta"} {
			short := "Alpha"
			if providerName == "Bank Beta" {
				short = "Beta"
			}
			for _, spec := range accountSpecs {
				accName := fmt.Sprintf("%s %s", short, spec.accName)
				createAccountViaUI(t, page, providerName, accName, spec.typeLabel, "CHF")
				time.Sleep(100 * time.Millisecond) // dialog open animation
			}
		}
	})

	//t.Run("Create Transactions", func(t *testing.T) {
	//	page, err = nav.Navigate(getURL(inst.BaseURL, "/entries"))
	//	if err != nil {
	//		t.Fatalf("navigate to /entries: %v", err)
	//	}
	//	page.MustWaitLoad()
	//
	//	// Use a strict timeout so failures fail fast instead of hanging
	//	page = page.Timeout(15 * time.Second)
	//
	//	createIncomeViaUI(t, page, "E2E salary", "100", "Alpha Wallet (CHF)")
	//	createExpenseViaUI(t, page, "E2E groceries", "25.50", "Alpha Wallet (CHF)")
	//	createTransferViaUI(t, page, "E2E move to savings", "50", "Alpha Wallet (CHF)", "Alpha Checking (CHF)")
	//})
}

func createProviderViaUI(t *testing.T, page *rod.Page, name, description string) {
	t.Helper()
	// Use XPath to avoid MustSearch hanging when multiple matches or DOM in flux
	addBtn := page.MustElementX("//button[contains(., 'Add Account Provider')]")
	addBtn.MustClick()
	time.Sleep(300 * time.Millisecond) // dialog open animation
	page.MustElement("input[placeholder='Provider Name']").MustInput(name)
	page.MustElement("input[placeholder='Description']").MustInput(description)
	page.MustSearch("Create").MustClick()
	time.Sleep(500 * time.Millisecond) // wait for API + Vue refresh

	// If create failed (e.g. backend error), dialog stays open. Check immediately—Element() waits,
	// and on success the dialog is gone, so we must not wait for it.
	if el, _ := page.Timeout(200 * time.Millisecond).Element("input[placeholder='Provider Name']"); el != nil {
		if visible, _ := el.Visible(); visible {
			page.MustSearch("Cancel").MustClick()
			//time.Sleep(200 * time.Millisecond)
			t.Fatalf("create provider %q failed (dialog still open – check backend)", name)
		}
	}

}

func createAccountViaUI(t *testing.T, page *rod.Page, providerName, accName, typeLabel, currency string) {
	t.Helper()
	// Find the provider row and click its plus button (add account).
	// Retry: after creating a provider, the table may need a moment to refresh.
	row, err := page.ElementX(fmt.Sprintf("//span[text()='%s']/ancestor::tr[1]", providerName))
	for i := 0; (err != nil || row == nil) && i < 5; i++ {
		time.Sleep(500 * time.Millisecond)
		row, err = page.ElementX(fmt.Sprintf("//span[text()='%s']/ancestor::tr[1]", providerName))
	}
	if err != nil || row == nil {
		t.Fatalf("provider %q not found in table (create provider may have failed)", providerName)
	}
	row.MustElement("button .pi-plus").MustClick()
	time.Sleep(400 * time.Millisecond)

	// Account dialog: fill name, icon, type, currency, save
	// Icon is NOT role=combobox (IconSelect uses button); Type and Currency are comboboxes
	page.MustElement("input[placeholder='Account Name']").MustInput(accName)

	page.MustElement(".icon-select-trigger").MustClick() // Icon (button, not combobox)
	time.Sleep(250 * time.Millisecond)
	// Target the icon grid button (title=icon name)
	page.MustElement("button.icon-item[title='money-bill']").MustClick()

	combo := page.MustElements("[role='combobox']")
	combo[0].MustClick() // Type
	time.Sleep(250 * time.Millisecond)
	// Target option by aria-label so we don't match "Cash"/etc elsewhere (e.g. table rows)
	page.MustElementX(fmt.Sprintf("//li[@role='option' and @aria-label='%s']", typeLabel)).MustClick()

	combo[1].MustClick() // Currency
	time.Sleep(250 * time.Millisecond)
	page.MustElementX(fmt.Sprintf("//li[@role='option' and @aria-label='%s']", currency)).MustClick()

	page.MustSearch("Save").MustClick()
	time.Sleep(500 * time.Millisecond)

	// If save failed, dialog stays open. Check immediately—Element() waits, and on success the
	// dialog is gone, so we must not wait for it.
	if el, _ := page.Timeout(200 * time.Millisecond).Element("input[placeholder='Account Name']"); el != nil {
		if visible, _ := el.Visible(); visible {
			page.MustSearch("Cancel").MustClick()
			time.Sleep(200 * time.Millisecond)
			t.Fatalf("create account %q under %q failed (dialog still open – check backend)", accName, providerName)
		}
	}
}

// openAddEntryAndSelect opens the Add Entry dropdown and selects the given option label (e.g. "Add Income").
func openAddEntryAndSelect(t *testing.T, page *rod.Page, optionLabel string) {
	t.Helper()
	page.MustElement(".add-entry-select [role='combobox']").MustClick()
	time.Sleep(250 * time.Millisecond)
	page.MustElementX(fmt.Sprintf("//li[@role='option' and contains(., '%s')]", optionLabel)).MustClick()
	time.Sleep(400 * time.Millisecond) // dialog open
}

// selectAccountInDialog selects an account in the AccountSelector (TreeSelect) within the visible entry dialog.
// accountLabel is the displayed label e.g. "Alpha Wallet (CHF)".
func selectAccountInDialog(page *rod.Page, accountLabel string) {
	dialog := page.MustElement(".entry-dialog")
	dialog.MustElement(".account-select [role='combobox']").MustClick()
	time.Sleep(250 * time.Millisecond)
	page.MustElementX(fmt.Sprintf("//*[@role='treeitem' and contains(., '%s')]", accountLabel)).MustClick()
	time.Sleep(150 * time.Millisecond)
}

func createIncomeViaUI(t *testing.T, page *rod.Page, description, amount, accountLabel string) {
	t.Helper()
	openAddEntryAndSelect(t, page, "Add Income")
	dialog := page.MustElement(".entry-dialog")
	dialog.MustElement("input[name='description']").MustInput(description)
	dialog.MustElement("input[name='amount']").MustInput(amount)
	selectAccountInDialog(page, accountLabel)
	page.MustSearch("Save").MustClick()
	time.Sleep(500 * time.Millisecond)
	if el, _ := page.Timeout(200 * time.Millisecond).Element(".entry-dialog input[name='description']"); el != nil {
		if visible, _ := el.Visible(); visible {
			page.MustSearch("Cancel").MustClick()
			t.Fatalf("create income %q failed (dialog still open)", description)
		}
	}
}

func createExpenseViaUI(t *testing.T, page *rod.Page, description, amount, accountLabel string) {
	t.Helper()
	openAddEntryAndSelect(t, page, "Add Expense")
	dialog := page.MustElement(".entry-dialog")
	dialog.MustElement("input[name='description']").MustInput(description)
	dialog.MustElement("input[name='amount']").MustInput(amount)
	selectAccountInDialog(page, accountLabel)
	page.MustSearch("Save").MustClick()
	time.Sleep(500 * time.Millisecond)
	if el, _ := page.Timeout(200 * time.Millisecond).Element(".entry-dialog input[name='description']"); el != nil {
		if visible, _ := el.Visible(); visible {
			page.MustSearch("Cancel").MustClick()
			t.Fatalf("create expense %q failed (dialog still open)", description)
		}
	}
}

func createTransferViaUI(t *testing.T, page *rod.Page, description, amount, originAccount, targetAccount string) {
	t.Helper()
	openAddEntryAndSelect(t, page, "Add Transfer")
	dialog := page.MustElement(".entry-dialog")
	dialog.MustElement("input[name='description']").MustInput(description)
	// Origin account (first AccountSelector in transfer dialog)
	dialog.MustElements(".account-select [role='combobox']")[0].MustClick()
	time.Sleep(250 * time.Millisecond)
	page.MustElementX(fmt.Sprintf("//*[@role='treeitem' and contains(., '%s')]", originAccount)).MustClick()
	time.Sleep(150 * time.Millisecond)
	dialog.MustElement("input[name='originAmount']").MustInput(amount)
	// Target account (second AccountSelector)
	dialog.MustElements(".account-select [role='combobox']")[1].MustClick()
	time.Sleep(250 * time.Millisecond)
	page.MustElementX(fmt.Sprintf("//*[@role='treeitem' and contains(., '%s')]", targetAccount)).MustClick()
	time.Sleep(150 * time.Millisecond)
	dialog.MustElement("input[name='targetAmount']").MustInput(amount)
	page.MustSearch("Save").MustClick()
	time.Sleep(500 * time.Millisecond)
	if el, _ := page.Timeout(200 * time.Millisecond).Element(".entry-dialog input[name='description']"); el != nil {
		if visible, _ := el.Visible(); visible {
			page.MustSearch("Cancel").MustClick()
			t.Fatalf("create transfer %q failed (dialog still open)", description)
		}
	}
}
