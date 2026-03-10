# Phase 1: Foundations — Type Safety & Quick Wins

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Establish proper TypeScript types across the codebase, migrate remaining JS files to TS, and fix locale inconsistency in number formatting.

**Architecture:** Three independent refactors that improve type safety and consistency. Item 9 (Entry types) provides the foundation for future refactors. Item 7 (JS→TS migration) eliminates untyped stores. Item 4 (locale fix) is a small consistency fix.

**Tech Stack:** TypeScript, Vue 3, Pinia, PrimeVue, Vitest

---

## Task 1: Complete the `Entry` TypeScript interface (Item 9)

**Files:**
- Modify: `webui/src/types/entry.ts`

**Context:** The current `Entry` interface (8 fields) is missing most fields used in templates and logic. The `UpdateEntryDTO` already hints at the full shape. Entry types are: `income`, `expense`, `transfer`, `stockbuy`, `stocksell`, `stockgrant`, `stocktransfer`, `balancestatus`.

### Step 1: Define the base entry fields and discriminated union

Replace the existing `Entry` interface in `webui/src/types/entry.ts` with a discriminated union. Keep the existing DTOs and other interfaces unchanged.

```typescript
/** Shared fields present on all entry types returned by the API. */
interface BaseEntry {
    id: string
    date: string
    description?: string
    notes?: string
    attachmentId?: number | null
    categoryId?: number
}

export interface IncomeEntry extends BaseEntry {
    type: 'income'
    accountId: string
    amount: number
    /** API returns PascalCase amount field */
    Amount: number
    targetStockAmount?: number
}

export interface ExpenseEntry extends BaseEntry {
    type: 'expense'
    accountId: string
    amount: number
    Amount: number
    targetStockAmount?: number
}

export interface TransferEntry extends BaseEntry {
    type: 'transfer'
    originAccountId: number
    targetAccountId: number
    originAmount: number
    targetAmount: number
    originStockAmount?: number
    targetStockAmount?: number
}

export interface StockBuyEntry extends BaseEntry {
    type: 'stockbuy'
    investmentAccountId: number
    cashAccountId: number
    instrumentId: number
    quantity: number
    totalAmount: number
    StockAmount: number
}

export interface StockSellEntry extends BaseEntry {
    type: 'stocksell'
    investmentAccountId: number
    cashAccountId: number
    instrumentId: number
    quantity: number
    totalAmount: number
    StockAmount: number
    costBasis?: number
    fees?: number
}

export interface StockGrantEntry extends BaseEntry {
    type: 'stockgrant'
    accountId: string
    instrumentId: number
    quantity: number
    fairMarketValue: number
}

export interface StockTransferEntry extends BaseEntry {
    type: 'stocktransfer'
    originAccountId: number
    targetAccountId: number
    instrumentId: number
    quantity: number
}

export interface BalanceStatusEntry extends BaseEntry {
    type: 'balancestatus'
    accountId: string
    amount: number
    Amount: number
}

export type Entry =
    | IncomeEntry
    | ExpenseEntry
    | TransferEntry
    | StockBuyEntry
    | StockSellEntry
    | StockGrantEntry
    | StockTransferEntry
    | BalanceStatusEntry
```

### Step 2: Run the TypeScript compiler to check for type errors

Run: `cd webui && npx vue-tsc --noEmit 2>&1 | head -50`

Expected: May show some errors in templates where fields are accessed without narrowing. These are pre-existing — the types now surface them. Note any errors but do not fix template code in this task.

### Step 3: Commit

```bash
git add webui/src/types/entry.ts
git commit -m "refactor: complete Entry type as discriminated union by entry type"
```

---

## Task 2: Migrate `settingsStore.js` to TypeScript (Item 7)

**Files:**
- Rename: `webui/src/store/settingsStore.js` → `webui/src/store/settingsStore.ts`

**Context:** 70 lines. Uses `err.message` without narrowing the `unknown` catch type. All refs are untyped. One import in `router/index.js` uses `.js` extension explicitly.

### Step 1: Rename the file

```bash
cd webui && mv src/store/settingsStore.js src/store/settingsStore.ts
```

### Step 2: Add type annotations

Add types to refs, fix the catch block, and type the return. Key changes:

```typescript
const error = ref<string | null>(null)
const dateFormat = ref<string>('')
const mainCurrency = ref<string>('')
const currencies = ref<string[]>([])
const instruments = ref<boolean>(false)
const marketDataSymbols = ref<string[]>([])
const version = ref<string>('')

// In catch block (line 34):
} catch (err: unknown) {
    console.error('Failed to fetch application settings:', err)
    error.value = err instanceof Error ? err.message : 'Failed to load settings'
}
```

### Step 3: Update the import in router/index.js

In `webui/src/router/index.js`, line 3 imports with explicit `.js` extension:
```typescript
// Before:
import { useSettingsStore } from '@/store/settingsStore.js'
// After:
import { useSettingsStore } from '@/store/settingsStore'
```

### Step 4: Verify build

Run: `cd webui && npx vue-tsc --noEmit 2>&1 | grep settingsStore`
Expected: No errors related to settingsStore.

### Step 5: Commit

```bash
git add webui/src/store/settingsStore.ts webui/src/router/index.js
git commit -m "refactor: migrate settingsStore to TypeScript"
```

---

## Task 3: Migrate `uiStore.js` to TypeScript (Item 7)

**Files:**
- Rename: `webui/src/store/uiStore.js` → `webui/src/store/uiStore.ts`

**Context:** 64 lines. Simple UI state (drawer visibility, screen width check). No catch blocks or complex types. Straightforward rename + annotation.

### Step 1: Rename and add types

```bash
cd webui && mv src/store/uiStore.js src/store/uiStore.ts
```

No code changes needed — the file uses `ref(false)` and `ref(true)` which TypeScript can infer. Just rename.

### Step 2: Verify build

Run: `cd webui && npx vue-tsc --noEmit 2>&1 | grep uiStore`
Expected: No errors.

### Step 3: Commit

```bash
git add webui/src/store/uiStore.ts
git commit -m "refactor: migrate uiStore to TypeScript"
```

---

## Task 4: Migrate `userStore.js` to TypeScript (Item 7)

**Files:**
- Rename: `webui/src/store/userStore.js` → `webui/src/store/userStore.ts`

**Context:** 161 lines. `login()` has 4 untyped parameters. `logoutCallbacks` is an untyped array. Commented-out code can be removed.

### Step 1: Rename the file

```bash
cd webui && mv src/store/userStore.js src/store/userStore.ts
```

### Step 2: Add type annotations

Key changes:

```typescript
const logoutCallbacks: Array<() => void> = []

const registerLogoutAction = (callback: () => void) => { ... }

const login = (
    user: string,
    pass: string,
    keepMeLoggedIn: boolean | undefined,
    onSuccessNavigate?: () => void
) => { ... }
```

Remove the commented-out toast code (lines 101-110).

### Step 3: Verify build

Run: `cd webui && npx vue-tsc --noEmit 2>&1 | grep userStore`
Expected: No errors.

### Step 4: Commit

```bash
git add webui/src/store/userStore.ts
git commit -m "refactor: migrate userStore to TypeScript"
```

---

## Task 5: Migrate `useDateFormat.js` to TypeScript (Item 7)

**Files:**
- Rename: `webui/src/composables/useDateFormat.js` → `webui/src/composables/useDateFormat.ts`

**Context:** 129 lines. Pure functions with string/Date parameters. Uses `z` from zod.

### Step 1: Rename the file

```bash
cd webui && mv src/composables/useDateFormat.js src/composables/useDateFormat.ts
```

### Step 2: Add type annotations to function parameters

```typescript
function toPrimeVueDateFormat(backendFormat: string | undefined): string { ... }

function formatDisplayDate(date: Date | string | null | undefined, format?: string): string { ... }

function formatTime(date: Date | string | null | undefined): string { ... }

function parseDateString(value: unknown, format?: string): Date | unknown { ... }
```

The return types of `useDateFormat()` composable are already inferred by Vue's `computed`.

### Step 3: Verify build

Run: `cd webui && npx vue-tsc --noEmit 2>&1 | grep useDateFormat`
Expected: No errors.

### Step 4: Commit

```bash
git add webui/src/composables/useDateFormat.ts
git commit -m "refactor: migrate useDateFormat to TypeScript"
```

---

## Task 6: Migrate `router/index.js` to TypeScript (Item 7)

**Files:**
- Rename: `webui/src/router/index.js` → `webui/src/router/index.ts`

**Context:** 304 lines. Vue Router setup with `beforeEach` guard. Uses `useUserStore` and `useSettingsStore`. The `navigate` inner function has untyped params.

### Step 1: Rename the file

```bash
cd webui && mv src/router/index.js src/router/index.ts
```

### Step 2: Add type annotations

```typescript
import type { RouteLocationNormalized, NavigationGuardNext } from 'vue-router'

// Line 265:
const navigate = function (to: RouteLocationNormalized, next: NavigationGuardNext) { ... }
```

### Step 3: Update the import in `main.js`

Check if `main.js` imports router with explicit `.js` extension. If so, remove the extension.

### Step 4: Verify build

Run: `cd webui && npx vue-tsc --noEmit 2>&1 | grep router`
Expected: No errors.

### Step 5: Commit

```bash
git add webui/src/router/index.ts
git commit -m "refactor: migrate router to TypeScript"
```

---

## Task 7: Migrate `main.js` to TypeScript (Item 7)

**Files:**
- Rename: `webui/src/main.js` → `webui/src/main.ts`

**Context:** 103 lines. App bootstrap file. Imports `CustomTheme` from `@/theme.js`. May need vite config update if the entry point is hardcoded.

### Step 1: Check vite config for entry point

Run: `cd webui && grep -n 'main' vite.config.* index.html`

If `index.html` references `src/main.js`, update to `src/main.ts`.

### Step 2: Rename the file

```bash
cd webui && mv src/main.js src/main.ts
```

### Step 3: Update index.html if needed

```html
<!-- Before: -->
<script type="module" src="/src/main.js"></script>
<!-- After: -->
<script type="module" src="/src/main.ts"></script>
```

### Step 4: Verify build

Run: `cd webui && npx vue-tsc --noEmit 2>&1 | head -20`
Expected: No new errors.

### Step 5: Commit

```bash
git add webui/src/main.ts webui/index.html
git commit -m "refactor: migrate main.js to TypeScript"
```

---

## Task 8: Fix locale inconsistency in number formatting (Item 4)

**Files:**
- Modify: `webui/src/utils/currency.ts`
- Modify: `webui/src/views/entries/EntriesTable.vue` (~15 occurrences)
- Modify: `webui/src/views/entries/AccountEntriesTable.vue` (~12 occurrences)

**Context:** `EntriesTable.vue` and `AccountEntriesTable.vue` hardcode `'es-ES'` locale in many `toLocaleString` calls. `currency.ts` defaults to `'en-US'`. Neither matches a user setting.

### Step 1: Update `currency.ts` to remove hardcoded locale

The browser's default locale (`undefined` in `toLocaleString`) respects the user's OS/browser language setting, which is the most sensible default for a finance app:

```typescript
export function formatCurrency(
    amount: number,
    minimumFractionDigits: number = 2,
    maximumFractionDigits: number = 2
): string {
    return amount.toLocaleString(undefined, {
        minimumFractionDigits,
        maximumFractionDigits
    })
}

export function formatAmount(amount: number): string {
    return formatCurrency(amount, 2, 2)
}
```

### Step 2: Replace all `toLocaleString('es-ES', ...)` in `EntriesTable.vue`

Replace every occurrence of `someValue.toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 })` with `formatAmount(someValue)`.

Add the import at the top of the script:
```typescript
import { formatAmount } from '@/utils/currency'
```

For cases with different fraction digits (e.g., `minimumFractionDigits: 2` only), use `formatCurrency(value, 2, 6)` or similar.

### Step 3: Replace all `toLocaleString('es-ES', ...)` in `AccountEntriesTable.vue`

Same approach. Replace the local `formatPrice` and `formatAmount` functions (lines 71-72) with imports from `@/utils/currency`.

### Step 4: Verify build

Run: `cd webui && npx vue-tsc --noEmit 2>&1 | head -20`
Expected: No errors.

### Step 5: Verify no remaining hardcoded locales

Run: `cd webui && grep -rn "es-ES\|en-US" src/`
Expected: No matches in table components. `currency.ts` should have no locale strings.

### Step 6: Commit

```bash
git add webui/src/utils/currency.ts webui/src/views/entries/EntriesTable.vue webui/src/views/entries/AccountEntriesTable.vue
git commit -m "fix: use browser locale for number formatting instead of hardcoded es-ES/en-US"
```
