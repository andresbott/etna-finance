package etna

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
)

func GetURL(baseurl, path string) string {
	sanitized := strings.TrimPrefix(path, "/")
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(baseurl, "/"), sanitized)
}

// WaitDialogClose waits for el to become invisible (dialog closed).
// If it doesn't close within the page timeout, the test fails with a descriptive message.
func WaitDialogClose(t *testing.T, el *rod.Element, action string) {
	t.Helper()
	if err := rod.Try(func() {
		el.MustWaitInvisible()
	}); err != nil {
		t.Fatalf("%s failed (dialog still open – check backend)", action)
	}
}

// TypeAmount clicks a PrimeVue InputNumber field, selects all text, and types the
// amount character by character. Rod's MustInput uses CDP insertText which doesn't
// trigger InputNumber's per-keystroke handlers, so we use Keyboard.Type instead.
func TypeAmount(page *rod.Page, el *rod.Element, amount string) {
	el.MustClick()
	el.MustSelectAllText()
	for _, c := range amount {
		page.Keyboard.MustType(input.Key(c))
	}
}

// TypeDate sets a date in a PrimeVue DatePicker input.
//
// PrimeVue DatePicker v4 renders its input as readonly (inputmode="none"): the field
// is not meant for keyboard entry. CDP key-event simulation is unreliable because
// Ctrl+A / select-all does not consistently clear the readonly field, causing stray
// characters to appear (e.g. "2026-01-052" instead of "2026-01-05").
//
// The reliable path is the native-setter + input-event approach: PrimeVue's onInput
// handler reads event.target.value, calls parseValue, and calls updateModel with no
// manualInput guard — so setting the DOM value via HTMLInputElement.prototype.value
// and dispatching a native "input" event commits the date directly.
//
// The date argument must be in the format configured for the application (e.g.
// "2026-01-17" for YYYY-MM-DD) so that parseValue can match against the dateFormat.
// Escape is pressed afterwards to close the calendar overlay.
func TypeDate(page *rod.Page, el *rod.Element, date string) {
	el.MustClick()
	el.MustEval(fmt.Sprintf(`function() {
		Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set.call(this, %q);
		this.dispatchEvent(new Event('input', { bubbles: true }));
	}`, date))
	time.Sleep(100 * time.Millisecond)
	page.Keyboard.MustType(input.Escape)
	time.Sleep(200 * time.Millisecond)
}

// ActiveTabPanel returns the currently visible tabpanel element on the page.
func ActiveTabPanel(page *rod.Page) *rod.Element {
	// PrimeVue keeps inactive tabpanels in the DOM with display:none;
	// the active one is marked with data-p-active="true".
	return page.MustElementX("//*[@role='tabpanel' and @data-p-active='true']")
}

// SelectCategoryInDialog opens the CategorySelect TreeSelect and picks the given category.
// If expandParent is non-empty, the parent node is expanded first (for selecting subcategories).
func SelectCategoryInDialog(t *testing.T, page *rod.Page, category, expandParent string) {
	t.Helper()
	dialog := page.MustElement(".entry-dialog")
	dialog.MustElement("#category-parent .p-treeselect-label-container").MustClick()
	time.Sleep(300 * time.Millisecond)

	if expandParent != "" {
		parentNode := page.MustElementX(fmt.Sprintf("//*[@role='treeitem'][@aria-label='%s']", expandParent))
		expanded, _ := parentNode.Attribute("aria-expanded")
		if expanded == nil || *expanded != "true" {
			parentNode.MustElement("button").MustClick()
			time.Sleep(200 * time.Millisecond)
		}
	}

	page.MustElementX(fmt.Sprintf("//*[@role='treeitem'][@aria-label='%s']", category)).MustClick()
	time.Sleep(200 * time.Millisecond)
}

// SelectAccountInDialog selects an account in the AccountSelector (TreeSelect) within the visible entry dialog.
// accountLabel is the displayed label e.g. "Alpha Wallet (CHF)".
func SelectAccountInDialog(t *testing.T, page *rod.Page, accountLabel string) {
	t.Helper()
	dialog := page.MustElement(".entry-dialog")
	trigger := dialog.MustElement(".account-select .p-treeselect:not(.p-disabled) .p-treeselect-label-container")
	ClickTreeSelectItem(t, page, trigger, accountLabel)
}

// ClickTreeSelectItem opens a TreeSelect dropdown (via trigger) and selects the item matching accountLabel.
// It waits for the tree overlay to stabilize before selecting, avoiding stale-element races from PrimeVue re-renders.
func ClickTreeSelectItem(t *testing.T, page *rod.Page, trigger *rod.Element, accountLabel string) {
	t.Helper()
	trigger.MustClick()
	page.MustElement("[role='tree']").MustWaitStable()
	page.MustElementX(fmt.Sprintf("//*[@role='treeitem'][@aria-label='%s']", accountLabel)).MustClick()
}

func CreateParentCategoryViaUI(t *testing.T, page *rod.Page, name, description string) {
	t.Helper()
	panel := ActiveTabPanel(page)
	panel.MustElementX(".//button[contains(., 'Add new parent category')]").MustClick()
	nameInput := page.MustElement("#category-name")
	nameInput.MustWaitInteractable()
	nameInput.MustInput(name)
	page.MustElement("#category-description").MustInput(description)
	page.MustElementX("//button[contains(., 'Save')]").MustClick()
	WaitDialogClose(t, nameInput, fmt.Sprintf("create parent category %q", name))
}

func CreateSubCategoryViaUI(t *testing.T, page *rod.Page, parentName, name, description string) {
	t.Helper()
	panel := ActiveTabPanel(page)
	row, err := panel.ElementX(fmt.Sprintf(".//span[contains(., '%s')]/ancestor::tr[1]", parentName))
	if err != nil || row == nil {
		t.Fatalf("parent category %q not found in table", parentName)
	}
	row.MustElement("button .pi-plus").MustClick()
	nameInput := page.MustElement("#category-name")
	nameInput.MustWaitInteractable()
	nameInput.MustInput(name)
	page.MustElement("#category-description").MustInput(description)
	page.MustElementX("//button[contains(., 'Save')]").MustClick()
	WaitDialogClose(t, nameInput, fmt.Sprintf("create subcategory %q under %q", name, parentName))
}

func CreateProviderViaUI(t *testing.T, page *rod.Page, name, description string) {
	t.Helper()
	// Use XPath to avoid MustSearch hanging when multiple matches or DOM in flux
	addBtn := page.MustElementX("//button[contains(., 'Add Account Provider')]")
	addBtn.MustClick()
	nameInput := page.MustElement("input[placeholder='Provider Name']")
	nameInput.MustWaitInteractable()
	nameInput.MustInput(name)
	page.MustElement("input[placeholder='Description']").MustInput(description)
	page.MustSearch("Create").MustClick()
	// Wait for dialog to close (element removed from DOM or hidden)
	WaitDialogClose(t, nameInput, fmt.Sprintf("create provider %q", name))
}

func CreateAccountViaUI(t *testing.T, page *rod.Page, providerName, accName, typeLabel, currency string) {
	t.Helper()
	// Find the provider row and click its plus button (add account)
	row, err := page.ElementX(fmt.Sprintf("//span[text()='%s']/ancestor::tr[1]", providerName))
	if err != nil || row == nil {
		t.Fatalf("provider %q not found in table (create provider may have failed)", providerName)
	}
	row.MustElement("button .pi-plus").MustClick()

	// Account dialog: fill name, icon, type, currency, save
	accInput := page.MustElement("input[placeholder='Account Name']")
	accInput.MustWaitInteractable()
	accInput.MustInput(accName)

	// Icon is NOT role=combobox (IconSelect uses button); Type and Currency are comboboxes
	page.MustElement(".icon-select-trigger").MustClick()
	page.MustElement("button.icon-item[title='money-bill']").MustWaitInteractable().MustClick()

	combo := page.MustElements("[role='combobox']")
	combo[0].MustClick() // Type
	// Target option by aria-label so we don't match "Cash"/etc elsewhere (e.g. table rows)
	page.MustElementX(fmt.Sprintf("//li[@role='option' and @aria-label='%s']", typeLabel)).MustWaitInteractable().MustClick()

	combo[1].MustClick() // Currency
	page.MustElementX(fmt.Sprintf("//li[@role='option' and @aria-label='%s']", currency)).MustWaitInteractable().MustClick()

	page.MustSearch("Save").MustClick()
	// Wait for dialog to close (element removed from DOM or hidden)
	WaitDialogClose(t, accInput, fmt.Sprintf("create account %q under %q", accName, providerName))
}

// OpenAddEntryAndSelect opens the Add Entry dropdown and selects the given option label (e.g. "Add Income").
func OpenAddEntryAndSelect(t *testing.T, page *rod.Page, optionLabel string) {
	t.Helper()
	page.MustElement(".add-entry-select [role='combobox']").MustClick()
	page.MustElementX(fmt.Sprintf("//li[@role='option' and contains(., '%s')]", optionLabel)).MustWaitInteractable().MustClick()
	// Callers wait for dialog elements to become interactable
}

func CreateIncomeViaUI(t *testing.T, page *rod.Page, description, amount, accountLabel, date, category, expandParent string) {
	t.Helper()
	OpenAddEntryAndSelect(t, page, "Add Income")
	dialog := page.MustElement(".entry-dialog")
	descInput := dialog.MustElement("input[name='description']")
	descInput.MustWaitInteractable()
	descInput.MustInput(description)
	// Select account first — this waits for async account loading to finish.
	SelectAccountInDialog(t, page, accountLabel)
	TypeAmount(page, dialog.MustElement("input[name='amount']"), amount)
	if date != "" {
		TypeDate(page, dialog.MustElement("input[name='date']"), date)
	}
	if category != "" {
		SelectCategoryInDialog(t, page, category, expandParent)
	}
	page.MustSearch("Save").MustClick()
	WaitDialogClose(t, descInput, fmt.Sprintf("create income %q", description))
}

func CreateExpenseViaUI(t *testing.T, page *rod.Page, description, amount, accountLabel, date, category, expandParent string) {
	t.Helper()
	OpenAddEntryAndSelect(t, page, "Add Expense")
	dialog := page.MustElement(".entry-dialog")
	descInput := dialog.MustElement("input[name='description']")
	descInput.MustWaitInteractable()
	descInput.MustInput(description)
	// Select account first — this waits for async account loading to finish.
	SelectAccountInDialog(t, page, accountLabel)
	TypeAmount(page, dialog.MustElement("input[name='amount']"), amount)
	if date != "" {
		TypeDate(page, dialog.MustElement("input[name='date']"), date)
	}
	if category != "" {
		SelectCategoryInDialog(t, page, category, expandParent)
	}
	page.MustSearch("Save").MustClick()
	WaitDialogClose(t, descInput, fmt.Sprintf("create expense %q", description))
}

func CreateTransferViaUI(t *testing.T, page *rod.Page, description, amount, originAccount, targetAccount, date string) {
	t.Helper()
	OpenAddEntryAndSelect(t, page, "Add Transfer")
	dialog := page.MustElement(".entry-dialog")
	descInput := dialog.MustElement("input[name='description']")
	descInput.MustWaitInteractable()
	descInput.MustInput(description)
	if date != "" {
		TypeDate(page, dialog.MustElement("input[name='date']"), date)
	}
	// Origin and target account selectors — click the visible label containers.
	accountSelects := dialog.MustElements(".account-select")
	// Origin account (first AccountSelector in transfer dialog)
	originTrigger := accountSelects[0].MustElement(".p-treeselect:not(.p-disabled) .p-treeselect-label-container")
	ClickTreeSelectItem(t, page, originTrigger, originAccount)
	TypeAmount(page, dialog.MustElement("input[name='originAmount']"), amount)
	// Target account (second AccountSelector)
	targetTrigger := accountSelects[1].MustElement(".p-treeselect:not(.p-disabled) .p-treeselect-label-container")
	ClickTreeSelectItem(t, page, targetTrigger, targetAccount)
	TypeAmount(page, dialog.MustElement("input[name='targetAmount']"), amount)
	page.MustSearch("Save").MustClick()
	WaitDialogClose(t, descInput, fmt.Sprintf("create transfer %q", description))
}
