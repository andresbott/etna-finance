---
name: e2e-test
description: Guide for writing Rod-based e2e tests that interact with the Vue/PrimeVue frontend
---

# Writing E2E Tests

E2E tests live in `zarf/e2e/` and use [go-rod/rod](https://github.com/go-rod/rod) (v0.116.2) to drive a real browser against a running instance of the app.

## Running tests

```bash
make e2e-test                                          # headed (HEADLESS=false)
E2E=true HEADLESS=false go test -v ./zarf/e2e          # headed manually (preferred)
E2E=true go test -v ./zarf/e2e                         # headless
```

**Always use `HEADLESS=false`** when running tests during development or debugging — it lets you see what the browser is doing and catch visual issues.

**Run tests early and in the background**: Whenever you start working on an e2e test (writing new code, debugging a failure, or investigating selectors), immediately launch the test run with `HEADLESS=false` as a **background task** so the user can watch the browser in real time. Do this *before* diving into code exploration or making fixes — don't wait until you're "done". Use `run_in_background: true` on the Bash tool:

```
E2E=true HEADLESS=false go test -v ./zarf/e2e --run TestName -timeout 120s
```

This way the user sees the browser open and can observe the failure visually while you're still reading code, checking selectors, or planning fixes.

## Project structure

| Path | Purpose |
|------|---------|
| `zarf/e2e/suite_test.go` | `SetupE2E(t, cfg)` — starts app instance + browser, registers cleanup |
| `zarf/e2e/instance/instance.go` | `EnvCfg`, `DefaultEnvCfg()`, `InitInstance()` — app lifecycle |
| `zarf/e2e/browser/browser.go` | `Browser` wrapper — `New()`, `Start()`, `Navigate()` |
| `zarf/e2e/*_test.go` | Actual test files |

## Test setup boilerplate

```go
func TestSomething(t *testing.T) {
    defer func() {
        if r := recover(); r != nil {
            t.Fatalf("panic: %v\n%s", r, debug.Stack())
        }
    }()

    // Use DefaultEnvCfg() and override what you need
    cfg := instance.DefaultEnvCfg()
    cfg.Settings.Instruments = true // enable if test needs instruments
    inst, nav := SetupE2E(t, &cfg)

    page, err := nav.Navigate(getURL(inst.BaseURL, "/your-page"))
    if err != nil {
        t.Fatalf("navigate: %v", err)
    }
    page.MustWaitLoad()
    page = page.Timeout(15 * time.Second) // strict timeout — fail fast

    t.Run("sub test", func(t *testing.T) {
        // ...
    })
}
```

`EnvCfg` fields: `Settings.DateFormat`, `Settings.MainCurrency`, `Settings.AdditionalCurrencies`, `Settings.Instruments`, `MarketDataImporters.Massive.ApiKeys`.

Auth is always disabled in e2e (`ETNA_AUTH_ENABLED=false`).

## Rod API — key methods

All `Must*` methods panic on failure (caught by the defer/recover in the test).

### Finding elements

```go
// CSS selector (preferred for classes, attributes, roles)
page.MustElement("input[name='description']")
page.MustElement(".entry-dialog")
page.MustElement("[role='combobox']")

// Multiple elements by CSS
page.MustElements(".account-select [role='combobox']")

// XPath (for text content matching or complex traversals)
page.MustElementX("//button[contains(., 'Add Account Provider')]")
page.MustElementX("//li[@role='option' and @aria-label='Cash']")
page.MustElementX("//span[text()='Bank Alpha']/ancestor::tr[1]")

// Scoped: find inside another element
dialog := page.MustElement(".entry-dialog")
dialog.MustElement("input[name='amount']")

// Text search (searches visible text — may hang if ambiguous, prefer XPath)
page.MustSearch("Save")
```

### Interacting

```go
el.MustClick()
el.MustInput("some text")            // types into input
el.MustWaitInteractable()            // waits until pointer-events != none
el.Visible()                         // (bool, error) — check visibility
```

### Waiting and timing

```go
page.MustWaitLoad()                  // wait for page load event
page = page.Timeout(15 * time.Second) // set strict timeout on page
el.MustWaitInteractable()            // wait for element to be clickable

// Manual sleeps for animations/transitions (keep minimal)
time.Sleep(250 * time.Millisecond)   // dropdown open animation
time.Sleep(300 * time.Millisecond)   // dialog open animation
time.Sleep(500 * time.Millisecond)   // API call + Vue re-render
```

### Short-timeout existence check (non-blocking)

```go
if el, _ := page.Timeout(200 * time.Millisecond).Element("selector"); el != nil {
    if visible, _ := el.Visible(); visible {
        // element exists and is visible
    }
}
```

## UI component selectors

The frontend uses **Vue 3 + PrimeVue 4**. When writing selectors, look at the `.vue` files in `webui/src/` to find the actual class names and attributes.

### Dialogs

All entry/account dialogs use PrimeVue `<Dialog>` with class `entry-dialog`:

```go
dialog := page.MustElement(".entry-dialog")
```

Wider variants: `.entry-dialog--wide`, `.entry-dialog--xwide`.

### Form inputs

Inputs use PrimeVue `InputText` / `InputNumber` with `name` attributes:

```go
dialog.MustElement("input[name='description']")
dialog.MustElement("input[name='amount']")
dialog.MustElement("input[name='originAmount']")
dialog.MustElement("input[name='targetAmount']")
// or by placeholder
page.MustElement("input[placeholder='Provider Name']")
page.MustElement("input[placeholder='Account Name']")
```

### Select / Dropdown (PrimeVue Select)

PrimeVue selects render with `role="combobox"`. Options render as `<li role="option">`:

```go
// Open dropdown
page.MustElement("[role='combobox']").MustClick()
time.Sleep(250 * time.Millisecond)

// Pick option by aria-label (preferred — avoids text collisions)
page.MustElementX("//li[@role='option' and @aria-label='Cash']").MustClick()

// Pick option by text content
page.MustElementX("//li[@role='option' and contains(., 'Add Income')]").MustClick()
```

When there are multiple dropdowns, use `MustElements`:

```go
combos := page.MustElements("[role='combobox']")
combos[0].MustClick() // first dropdown
combos[1].MustClick() // second dropdown
```

### Account selector (TreeSelect)

The `AccountSelector.vue` component wraps PrimeVue `TreeSelect`. It is **disabled while loading accounts** (`pointer-events: none`). Always wait:

```go
dialog := page.MustElement(".entry-dialog")
combobox := dialog.MustElement(".account-select [role='combobox']")
combobox.MustWaitInteractable()  // REQUIRED — waits for accounts API to finish
combobox.MustClick()
time.Sleep(250 * time.Millisecond)

// Options are tree items — match by displayed label "Name (Currency)"
page.MustElementX("//*[@role='treeitem' and contains(., 'Alpha Wallet (CHF)')]").MustClick()
```

Account labels follow the pattern: `AccountName (Currency)`, e.g. `"Alpha Wallet (CHF)"`.

### Icon selector

Custom component, not a combobox:

```go
page.MustElement(".icon-select-trigger").MustClick()
time.Sleep(250 * time.Millisecond)
page.MustElement("button.icon-item[title='money-bill']").MustClick()
```

### Buttons

```go
// By text (XPath)
page.MustElementX("//button[contains(., 'Add Account Provider')]").MustClick()

// By text (MustSearch — searches visible text, can be ambiguous)
page.MustSearch("Save").MustClick()
page.MustSearch("Cancel").MustClick()
page.MustSearch("Create").MustClick()

// By icon class inside a scoped element
row.MustElement("button .pi-plus").MustClick()
```

### Entry type menu

On the entries page, selecting an entry type to create:

```go
page.MustElement(".add-entry-select [role='combobox']").MustClick()
time.Sleep(250 * time.Millisecond)
page.MustElementX("//li[@role='option' and contains(., 'Add Income')]").MustClick()
time.Sleep(400 * time.Millisecond) // dialog open animation
```

Options: `"Add Income"`, `"Add Expense"`, `"Add Transfer"`, `"Buy/Sell"`, etc.

### Tables

PrimeVue DataTable. Find rows by content:

```go
row, err := page.ElementX("//span[text()='Bank Alpha']/ancestor::tr[1]")
```

## Detecting dialog success/failure

After clicking Save, check if the dialog closed. If it's still open, the operation failed:

```go
page.MustSearch("Save").MustClick()
time.Sleep(500 * time.Millisecond)

if el, _ := page.Timeout(200 * time.Millisecond).Element(".entry-dialog input[name='description']"); el != nil {
    if visible, _ := el.Visible(); visible {
        page.MustSearch("Cancel").MustClick()
        t.Fatalf("operation failed (dialog still open)")
    }
}
```

## Common pitfalls

1. **Loading states**: `AccountSelector` (and other components using `useAccounts`/`useQuery`) disable their inputs while fetching data. Always call `MustWaitInteractable()` before clicking.
2. **Dialog animation**: PrimeVue dialogs have open/close transitions. Sleep 300-400ms after triggering a dialog.
3. **Dropdown animation**: Sleep 250ms after clicking a combobox before selecting an option.
4. **`MustSearch` ambiguity**: If multiple elements match the text, `MustSearch` may pick the wrong one or hang. Prefer XPath with `//button[contains(., 'text')]` for buttons.
5. **Strict timeouts**: Always set `page = page.Timeout(15 * time.Second)` to fail fast instead of hanging for 30+ seconds on missing elements.
6. **Feature flags**: If the test needs financial instruments or stocks, set `cfg.Settings.Instruments = true` in `EnvCfg`.

## Finding selectors for new components

1. Look at the `.vue` file in `webui/src/views/` or `webui/src/components/`
2. Check the `<template>` section for:
   - Custom classes (`.entry-dialog`, `.account-select`, `.add-entry-select`)
   - PrimeVue component props: `name`, `placeholder`, `role`
   - ARIA attributes: `role="combobox"`, `role="option"`, `role="treeitem"`
3. PrimeVue components render predictable DOM:
   - `Select` / `TreeSelect` -> `[role='combobox']` trigger, `[role='option']` / `[role='treeitem']` items
   - `InputText` / `InputNumber` -> `<input>` with `name` attribute
   - `Dialog` -> container with the class you put on the component
   - `Button` -> `<button>` with label text inside
   - `DataTable` -> `<table>` with `.p-datatable` class

## Debugging with Playwright MCP

When reading source code and Rod selectors isn't enough to diagnose a failing test, use the **Playwright MCP browser tools** to connect to a live app instance and interactively inspect the real DOM.

### 1. Start a standalone instance

Start the app the same way `instance.go` does, but on a fixed port:

```bash
mkdir -p /tmp/etna-debug
ETNA_DATADIR=/tmp/etna-debug \
ETNA_SERVER_PORT=9876 \
ETNA_SERVER_BINDIP=127.0.0.1 \
ETNA_AUTH_ENABLED=false \
ETNA_SETTINGS_DATEFORMAT=YYYY-MM-DD \
ETNA_SETTINGS_MAINCURRENCY=CHF \
ETNA_SETTINGS_INSTRUMENTS=true \
  go run main.go start -c /tmp/etna-debug/config.yaml &
```

Wait for it to be ready: `curl -s http://127.0.0.1:9876/api/v0/settings`

### 2. Seed test data via the API

Create providers and accounts so the UI has data to work with:

```bash
# Create provider
curl -s -X POST http://127.0.0.1:9876/api/v0/fin/provider \
  -H 'Content-Type: application/json' \
  -d '{"name":"Bank Alpha","description":"debug provider"}'

# Create accounts (note: type values are lowercase: "cash", "checkin", "savings", "investment")
curl -s -X POST http://127.0.0.1:9876/api/v0/fin/account \
  -H 'Content-Type: application/json' \
  -d '{"name":"Alpha Wallet","providerID":1,"type":"cash","currency":"CHF","icon":"money-bill"}'
```

### 3. Navigate with Playwright MCP and inspect

Use `browser_navigate` to open the page, then `browser_snapshot` to get the accessibility tree:

- `browser_navigate` -> `http://127.0.0.1:9876/entries`
- `browser_snapshot` -> shows all elements with refs, roles, and text
- `browser_click` -> click elements by ref to open dialogs/dropdowns and re-snapshot
- `browser_evaluate` -> run JS to inspect the actual DOM (class names, attributes, etc.)

### 4. What to look for

- **Accessibility tree** (`browser_snapshot`): shows `role`, `aria-label`, `aria-expanded`, element hierarchy. Use this to verify XPath/CSS selectors and understand the component structure.
- **Raw HTML** (`browser_evaluate`): check actual CSS classes (`.p-treeselect`, `.p-disabled`, `.account-select`), `name` attributes on inputs, and PrimeVue-generated structure.
- **Stale elements**: click through a multi-step flow (open dropdown -> select item -> open next dropdown) and snapshot at each step to see if elements get recreated.
- **InputNumber fields**: PrimeVue `InputNumber` renders as `<input role="spinbutton">` with a default value like `"0.00"`. Rod's `MustInput()` inserts text at the cursor without clearing — use `MustSelectAllText().MustInput(value)` to replace the default.

### 5. Cleanup

```bash
pkill -f "etna.*start"
rm -rf /tmp/etna-debug
```
