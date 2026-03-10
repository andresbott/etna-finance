# Frontend Unit Tests Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add comprehensive Vitest unit tests across all frontend logic layers (utils, API wrappers, stores, composables) to prevent regressions and improve refactoring confidence.

**Architecture:** Colocated test files (`foo.test.ts` next to `foo.ts`). Four tiers of increasing mock complexity: pure functions (no mocks), API wrappers (mock apiClient), stores (mock axios), composables (mock API modules + renderComposable helper). Each tier is independently valuable.

**Tech Stack:** Vitest, @vue/test-utils, @tanstack/vue-query, pinia, zod (all already installed)

**Design doc:** `docs/plans/2026-03-10-frontend-unit-tests-design.md`

---

## Task 1: Test Infrastructure — Shared Helper

**Files:**
- Create: `webui/src/test/helpers.ts`

**Step 1: Create the test helper file**

```ts
// webui/src/test/helpers.ts
import { createApp, type App } from 'vue'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia, setActivePinia } from 'pinia'

export function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: Infinity },
    },
  })
}

export function renderComposable<T>(
  composable: () => T,
  options?: { queryClient?: QueryClient }
): { result: T; unmount: () => void; app: App } {
  const pinia = createPinia()
  setActivePinia(pinia)

  let result: T
  const app = createApp({
    setup() {
      result = composable()
      return () => {}
    },
  })
  const qc = options?.queryClient ?? createTestQueryClient()
  app.use(VueQueryPlugin, { queryClient: qc })
  app.use(pinia)
  app.mount(document.createElement('div'))

  return { result: result!, unmount: () => app.unmount(), app }
}
```

**Step 2: Verify the helper compiles**

Run: `cd webui && npx vitest run --passWithNoTests 2>&1 | tail -5`
Expected: No compilation errors

**Step 3: Commit**

```bash
git add webui/src/test/helpers.ts
git commit -m "test: add shared test helper for composable testing"
```

---

## Task 2: Tier 1 — `utils/currency.ts`

**Files:**
- Create: `webui/src/utils/currency.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect } from 'vitest'
import { formatCurrency, formatAmount } from './currency'

describe('formatCurrency', () => {
  it('formats positive number with defaults', () => {
    expect(formatCurrency(1234.5)).toBe('1,234.50')
  })

  it('formats zero', () => {
    expect(formatCurrency(0)).toBe('0.00')
  })

  it('formats negative number', () => {
    expect(formatCurrency(-500.1)).toBe('-500.10')
  })

  it('respects custom fraction digits', () => {
    expect(formatCurrency(1.5, 'en-US', 0, 0)).toBe('2')
  })

  it('respects locale', () => {
    const result = formatCurrency(1234.5, 'de-DE')
    expect(result).toContain('1.234,50')
  })
})

describe('formatAmount', () => {
  it('delegates to formatCurrency with defaults', () => {
    expect(formatAmount(42)).toBe('42.00')
  })

  it('formats large numbers', () => {
    expect(formatAmount(1000000)).toBe('1,000,000.00')
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/utils/currency.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/utils/currency.test.ts
git commit -m "test: add unit tests for currency formatting utils"
```

---

## Task 3: Tier 1 — `utils/format.ts`

**Files:**
- Create: `webui/src/utils/format.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect } from 'vitest'
import { formatPct, getChangeSeverity } from './format'

describe('formatPct', () => {
  it('formats positive percentage with + prefix', () => {
    expect(formatPct(2.5)).toBe('+2.50%')
  })

  it('formats negative percentage', () => {
    expect(formatPct(-1.75)).toBe('-1.75%')
  })

  it('formats zero', () => {
    expect(formatPct(0)).toMatch(/0\.00%/)
  })

  it('returns dash for null', () => {
    expect(formatPct(null)).toBe('-')
  })

  it('returns dash for undefined', () => {
    expect(formatPct(undefined)).toBe('-')
  })
})

describe('getChangeSeverity', () => {
  it('returns success for positive', () => {
    expect(getChangeSeverity(1)).toBe('success')
  })

  it('returns danger for negative', () => {
    expect(getChangeSeverity(-1)).toBe('danger')
  })

  it('returns secondary for zero', () => {
    expect(getChangeSeverity(0)).toBe('secondary')
  })

  it('returns secondary for null', () => {
    expect(getChangeSeverity(null)).toBe('secondary')
  })

  it('returns secondary for undefined', () => {
    expect(getChangeSeverity(undefined)).toBe('secondary')
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/utils/format.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/utils/format.test.ts
git commit -m "test: add unit tests for format utils (formatPct, getChangeSeverity)"
```

---

## Task 4: Tier 1 — `utils/date.ts`

**Files:**
- Create: `webui/src/utils/date.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect } from 'vitest'
import { toLocalDateString } from './date'

describe('toLocalDateString', () => {
  it('formats Date to YYYY-MM-DD', () => {
    const d = new Date(2026, 2, 10) // March 10, 2026
    expect(toLocalDateString(d)).toBe('2026-03-10')
  })

  it('handles string input', () => {
    expect(toLocalDateString('2026-03-10')).toBe('2026-03-10')
  })

  it('returns today for null', () => {
    const today = new Date()
    const y = today.getFullYear()
    const m = String(today.getMonth() + 1).padStart(2, '0')
    const day = String(today.getDate()).padStart(2, '0')
    expect(toLocalDateString(null)).toBe(`${y}-${m}-${day}`)
  })

  it('returns today for undefined', () => {
    const result = toLocalDateString(undefined)
    expect(result).toMatch(/^\d{4}-\d{2}-\d{2}$/)
  })

  it('pads single-digit months and days', () => {
    const d = new Date(2026, 0, 5) // Jan 5
    expect(toLocalDateString(d)).toBe('2026-01-05')
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/utils/date.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/utils/date.test.ts
git commit -m "test: add unit tests for date utils (toLocalDateString)"
```

---

## Task 5: Tier 1 — `utils/dateRange.ts`

**Files:**
- Create: `webui/src/utils/dateRange.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect, vi, afterEach } from 'vitest'
import { lastDaysRange, rangeToStartEnd } from './dateRange'

describe('lastDaysRange', () => {
  afterEach(() => vi.useRealTimers())

  it('returns start and end spanning N days', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 2, 10)) // March 10, 2026

    const range = lastDaysRange(30)
    expect(range.end).toBe('2026-03-10')
    expect(range.start).toBe('2026-02-08')
  })

  it('returns same day for 0 days', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 2, 10))

    const range = lastDaysRange(0)
    expect(range.start).toBe('2026-03-10')
    expect(range.end).toBe('2026-03-10')
  })
})

describe('rangeToStartEnd', () => {
  afterEach(() => vi.useRealTimers())

  it('returns 6-month range for "6m"', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 2, 10))

    const range = rangeToStartEnd('6m')
    expect(range.end).toBe('2026-03-10')
    expect(range.start).toBe('2025-09-10')
  })

  it('returns 10-year range for "max"', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 2, 10))

    const range = rangeToStartEnd('max')
    expect(range.end).toBe('2026-03-10')
    expect(range.start).toBe('2016-03-10')
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/utils/dateRange.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/utils/dateRange.test.ts
git commit -m "test: add unit tests for dateRange utils"
```

---

## Task 6: Tier 1 — `utils/apiError.ts`

**Files:**
- Create: `webui/src/utils/apiError.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect } from 'vitest'
import { getApiErrorMessage } from './apiError'

describe('getApiErrorMessage', () => {
  it('extracts message from response.data.message', () => {
    const err = { response: { data: { message: 'Custom error' }, status: 400 } }
    expect(getApiErrorMessage(err)).toBe('Custom error')
  })

  it('extracts message from response.data.error', () => {
    const err = { response: { data: { error: 'Error string' }, status: 400 } }
    expect(getApiErrorMessage(err)).toBe('Error string')
  })

  it('returns generic message for 400', () => {
    const err = { response: { data: {}, status: 400 } }
    expect(getApiErrorMessage(err)).toBe('Invalid request. Please check your input.')
  })

  it('returns auth message for 401', () => {
    const err = { response: { data: {}, status: 401 } }
    expect(getApiErrorMessage(err)).toBe('You are not authorized.')
  })

  it('returns permission message for 403', () => {
    const err = { response: { data: {}, status: 403 } }
    expect(getApiErrorMessage(err)).toBe('You do not have permission.')
  })

  it('returns not found message for 404', () => {
    const err = { response: { data: {}, status: 404 } }
    expect(getApiErrorMessage(err)).toBe('The resource was not found.')
  })

  it('returns server error for 500', () => {
    const err = { response: { data: {}, status: 500 } }
    expect(getApiErrorMessage(err)).toBe('Server error. Please try again later.')
  })

  it('returns server error for 503', () => {
    const err = { response: { data: {}, status: 503 } }
    expect(getApiErrorMessage(err)).toBe('Server error. Please try again later.')
  })

  it('falls back to err.message for non-response errors', () => {
    const err = new Error('Network failure')
    expect(getApiErrorMessage(err)).toBe('Network failure')
  })

  it('returns generic fallback for unknown shape', () => {
    expect(getApiErrorMessage({})).toBe('An error occurred')
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/utils/apiError.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/utils/apiError.test.ts
git commit -m "test: add unit tests for apiError utils"
```

---

## Task 7: Tier 1 — `utils/convertToTree.ts`

**Files:**
- Create: `webui/src/utils/convertToTree.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect } from 'vitest'
import { buildTree, buildTreeForTable } from './convertToTree'

describe('buildTree', () => {
  it('builds flat items into tree', () => {
    const items = [
      { id: 1, parentId: null, name: 'Root' },
      { id: 2, parentId: 1, name: 'Child' },
      { id: 3, parentId: 1, name: 'Child2' },
    ]
    const tree = buildTree(items)
    expect(tree).toHaveLength(1)
    expect(tree[0].children).toHaveLength(2)
  })

  it('handles multiple roots', () => {
    const items = [
      { id: 1, parentId: null, name: 'Root1' },
      { id: 2, parentId: null, name: 'Root2' },
    ]
    const tree = buildTree(items)
    expect(tree).toHaveLength(2)
  })

  it('handles empty array', () => {
    expect(buildTree([])).toEqual([])
  })

  it('handles deeply nested items', () => {
    const items = [
      { id: 1, parentId: null, name: 'L1' },
      { id: 2, parentId: 1, name: 'L2' },
      { id: 3, parentId: 2, name: 'L3' },
    ]
    const tree = buildTree(items)
    expect(tree[0].children[0].children[0].name).toBe('L3')
  })
})

describe('buildTreeForTable', () => {
  it('returns PrimeVue TreeTable format', () => {
    const items = [
      { id: 1, parentId: null, name: 'Root' },
      { id: 2, parentId: 1, name: 'Child' },
    ]
    const tree = buildTreeForTable(items)
    expect(tree[0]).toHaveProperty('key')
    expect(tree[0]).toHaveProperty('data')
    expect(tree[0].children).toHaveLength(1)
  })

  it('handles undefined input', () => {
    expect(buildTreeForTable(undefined)).toEqual([])
  })

  it('handles empty array', () => {
    expect(buildTreeForTable([])).toEqual([])
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/utils/convertToTree.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/utils/convertToTree.test.ts
git commit -m "test: add unit tests for convertToTree utils"
```

---

## Task 8: Tier 1 — `utils/categoryUtils.ts` (pure functions only)

**Files:**
- Create: `webui/src/utils/categoryUtils.test.ts`

**Step 1: Write tests**

Note: Only test `findNodeById` and `buildCategoryPath` (pure functions). Skip `useCategoryUtils` (composable, needs mocks — Tier 4).

```ts
import { describe, it, expect } from 'vitest'
import { findNodeById, buildCategoryPath } from './categoryUtils'

const sampleTree = [
  {
    key: '1',
    label: 'Food',
    data: { id: 1, parentId: undefined, name: 'Food', path: 'Food' },
    children: [
      {
        key: '2',
        label: 'Groceries',
        data: { id: 2, parentId: 1, name: 'Groceries', path: 'Food > Groceries' },
        children: [],
      },
    ],
  },
  {
    key: '3',
    label: 'Transport',
    data: { id: 3, parentId: undefined, name: 'Transport', path: 'Transport' },
    children: [],
  },
]

describe('findNodeById', () => {
  it('finds root node by id', () => {
    const node = findNodeById(sampleTree, 1)
    expect(node?.label).toBe('Food')
  })

  it('finds nested node by id', () => {
    const node = findNodeById(sampleTree, 2)
    expect(node?.label).toBe('Groceries')
  })

  it('finds node by string id (matches key)', () => {
    const node = findNodeById(sampleTree, '3')
    expect(node?.label).toBe('Transport')
  })

  it('returns null for missing id', () => {
    expect(findNodeById(sampleTree, 999)).toBeNull()
  })

  it('returns null for empty tree', () => {
    expect(findNodeById([], 1)).toBeNull()
  })
})

describe('buildCategoryPath', () => {
  it('returns path from data.path when available', () => {
    const path = buildCategoryPath(sampleTree, 2)
    expect(path).toBe('Food > Groceries')
  })

  it('returns root name for root node', () => {
    const path = buildCategoryPath(sampleTree, 1)
    expect(path).toBe('Food')
  })

  it('returns empty string for missing id', () => {
    expect(buildCategoryPath(sampleTree, 999)).toBe('')
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/utils/categoryUtils.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/utils/categoryUtils.test.ts
git commit -m "test: add unit tests for categoryUtils pure functions"
```

---

## Task 9: Tier 1 — `utils/accountUtils.ts` (pure function only)

**Files:**
- Create: `webui/src/utils/accountUtils.test.ts`

**Step 1: Write tests**

Note: Only test `findAccountById` (pure). Skip `useAccountUtils` (composable).

```ts
import { describe, it, expect } from 'vitest'
import { findAccountById } from './accountUtils'

const providers = [
  {
    id: 1,
    name: 'Bank Alpha',
    accounts: [
      { id: 10, name: 'Checking', currency: 'CHF', type: 'checkin' },
      { id: 11, name: 'Savings', currency: 'CHF', type: 'savings' },
    ],
  },
  {
    id: 2,
    name: 'Broker',
    accounts: [
      { id: 20, name: 'Investment', currency: 'USD', type: 'investment' },
    ],
  },
]

describe('findAccountById', () => {
  it('finds account in first provider', () => {
    const acc = findAccountById(providers as any, 10)
    expect(acc?.name).toBe('Checking')
  })

  it('finds account in second provider', () => {
    const acc = findAccountById(providers as any, 20)
    expect(acc?.name).toBe('Investment')
  })

  it('finds account by string id', () => {
    const acc = findAccountById(providers as any, '11')
    expect(acc?.name).toBe('Savings')
  })

  it('returns null for missing id', () => {
    expect(findAccountById(providers as any, 999)).toBeNull()
  })

  it('returns null for empty providers', () => {
    expect(findAccountById([], 1)).toBeNull()
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/utils/accountUtils.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/utils/accountUtils.test.ts
git commit -m "test: add unit tests for accountUtils (findAccountById)"
```

---

## Task 10: Tier 1 — `utils/entryDisplay.ts`

**Files:**
- Create: `webui/src/utils/entryDisplay.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect } from 'vitest'
import { getEntryTypeIcon, ENTRY_TYPE_ICONS } from './entryDisplay'

describe('getEntryTypeIcon', () => {
  it('returns icon for expense', () => {
    expect(getEntryTypeIcon('expense')).toContain('pi-minus')
  })

  it('returns icon for income', () => {
    expect(getEntryTypeIcon('income')).toContain('pi-plus')
  })

  it('returns icon for transfer', () => {
    expect(getEntryTypeIcon('transfer')).toContain('pi-arrow-right-arrow-left')
  })

  it('returns icon for stockbuy', () => {
    expect(getEntryTypeIcon('stockbuy')).toContain('pi-chart-line')
  })

  it('returns fallback for unknown type', () => {
    expect(getEntryTypeIcon('unknown')).toContain('pi-question-circle')
  })

  it('returns fallback for null', () => {
    expect(getEntryTypeIcon(null)).toContain('pi-question-circle')
  })

  it('returns fallback for undefined', () => {
    expect(getEntryTypeIcon(undefined)).toContain('pi-question-circle')
  })
})

describe('ENTRY_TYPE_ICONS', () => {
  it('has entries for all known types', () => {
    const knownTypes = ['expense', 'income', 'transfer', 'stockbuy', 'stocksell', 'stockgrant', 'stocktransfer', 'balancestatus', 'opening-balance']
    for (const t of knownTypes) {
      expect(ENTRY_TYPE_ICONS).toHaveProperty(t)
    }
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/utils/entryDisplay.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/utils/entryDisplay.test.ts
git commit -m "test: add unit tests for entryDisplay utils"
```

---

## Task 11: Tier 1 — `utils/entryValidation.ts`

**Files:**
- Create: `webui/src/utils/entryValidation.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect } from 'vitest'
import { accountValidation } from './entryValidation'

describe('accountValidation', () => {
  it('rejects null', () => {
    const result = accountValidation.safeParse(null)
    expect(result.success).toBe(false)
  })

  it('accepts valid account selection', () => {
    const result = accountValidation.safeParse({ 5: true })
    expect(result.success).toBe(true)
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/utils/entryValidation.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/utils/entryValidation.test.ts
git commit -m "test: add unit tests for entryValidation schema"
```

---

## Task 12: Tier 1 — `types/account.ts`

**Files:**
- Create: `webui/src/types/account.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect } from 'vitest'
import {
  getAccountTypeLabel,
  getAccountTypeIcon,
  getAllowedOperations,
  isOperationAllowed,
  ACCOUNT_TYPES,
  ENTRY_OPERATIONS,
} from './account'

describe('getAccountTypeLabel', () => {
  it('returns label for known type', () => {
    expect(getAccountTypeLabel(ACCOUNT_TYPES.CASH)).toBeTruthy()
    expect(typeof getAccountTypeLabel(ACCOUNT_TYPES.CASH)).toBe('string')
  })

  it('returns label for each type', () => {
    for (const type of Object.values(ACCOUNT_TYPES)) {
      expect(getAccountTypeLabel(type)).not.toBe('Unknown')
    }
  })

  it('returns Unknown for null', () => {
    expect(getAccountTypeLabel(null)).toBe('Unknown')
  })

  it('returns Unknown for undefined', () => {
    expect(getAccountTypeLabel(undefined)).toBe('Unknown')
  })

  it('returns Unknown for invalid type', () => {
    expect(getAccountTypeLabel('nonexistent')).toBe('Unknown')
  })
})

describe('getAccountTypeIcon', () => {
  it('returns icon for known type', () => {
    const icon = getAccountTypeIcon(ACCOUNT_TYPES.INVESTMENT)
    expect(icon).toBeTruthy()
  })

  it('returns fallback for unknown type', () => {
    expect(getAccountTypeIcon('nonexistent')).toContain('wallet')
  })

  it('returns fallback for null', () => {
    expect(getAccountTypeIcon(null)).toContain('wallet')
  })
})

describe('getAllowedOperations', () => {
  it('returns operations for investment account', () => {
    const ops = getAllowedOperations(ACCOUNT_TYPES.INVESTMENT)
    expect(ops).toContain(ENTRY_OPERATIONS.BUY_STOCK)
    expect(ops).toContain(ENTRY_OPERATIONS.SELL_STOCK)
  })

  it('returns all operations for null account type', () => {
    const ops = getAllowedOperations(null)
    expect(ops.length).toBeGreaterThan(0)
  })

  it('returns all operations for undefined account type', () => {
    const ops = getAllowedOperations(undefined)
    expect(ops.length).toBeGreaterThan(0)
  })
})

describe('isOperationAllowed', () => {
  it('allows expense on cash account', () => {
    expect(isOperationAllowed(ENTRY_OPERATIONS.EXPENSE, ACCOUNT_TYPES.CASH)).toBe(true)
  })

  it('allows buy stock on investment account', () => {
    expect(isOperationAllowed(ENTRY_OPERATIONS.BUY_STOCK, ACCOUNT_TYPES.INVESTMENT)).toBe(true)
  })

  it('allows any operation when account type is null', () => {
    expect(isOperationAllowed(ENTRY_OPERATIONS.EXPENSE, null)).toBe(true)
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/types/account.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/types/account.test.ts
git commit -m "test: add unit tests for account type functions"
```

---

## Task 13: Tier 1 — `composables/useEntryDialogForm.ts` (pure functions only)

**Files:**
- Create: `webui/src/composables/useEntryDialogForm.test.ts`

**Step 1: Write tests**

Note: Only test the pure exported functions: `extractAccountId`, `getFormattedAccountId`, `getDateOnly`, `toDateString`, `getSubmitValues`. Skip the composable itself.

```ts
import { describe, it, expect } from 'vitest'
import { ref } from 'vue'
import {
  extractAccountId,
  getFormattedAccountId,
  getDateOnly,
  toDateString,
  getSubmitValues,
} from './useEntryDialogForm'

describe('extractAccountId', () => {
  it('extracts id from { [id]: true } format', () => {
    expect(extractAccountId({ 5: true })).toBe(5)
  })

  it('returns number directly', () => {
    expect(extractAccountId(7)).toBe(7)
  })

  it('returns null for null', () => {
    expect(extractAccountId(null)).toBeNull()
  })

  it('returns null for undefined', () => {
    expect(extractAccountId(undefined)).toBeNull()
  })
})

describe('getFormattedAccountId', () => {
  it('wraps number into { [id]: true } format', () => {
    expect(getFormattedAccountId(5)).toEqual({ 5: true })
  })

  it('returns null for null', () => {
    expect(getFormattedAccountId(null)).toBeNull()
  })

  it('returns null for undefined', () => {
    expect(getFormattedAccountId(undefined)).toBeNull()
  })
})

describe('getDateOnly', () => {
  it('strips time from Date', () => {
    const input = new Date(2026, 2, 10, 14, 30, 0)
    const result = getDateOnly(input)
    expect(result.getHours()).toBe(0)
    expect(result.getMinutes()).toBe(0)
  })

  it('parses string date', () => {
    const result = getDateOnly('2026-03-10')
    expect(result instanceof Date).toBe(true)
  })

  it('returns current date for null', () => {
    const result = getDateOnly(null)
    expect(result instanceof Date).toBe(true)
  })
})

describe('toDateString', () => {
  it('formats Date to YYYY-MM-DD', () => {
    expect(toDateString(new Date(2026, 2, 10))).toBe('2026-03-10')
  })

  it('handles string input', () => {
    expect(toDateString('2026-03-10')).toBe('2026-03-10')
  })
})

describe('getSubmitValues', () => {
  it('merges event values with form ref', () => {
    const formValues = ref({ name: 'old', amount: 100 })
    const event = { values: { name: 'new' } }
    const result = getSubmitValues(event, formValues)
    expect(result.name).toBe('new')
  })

  it('uses formValues when event has no values', () => {
    const formValues = ref({ name: 'keep' })
    const event = {}
    const result = getSubmitValues(event, formValues)
    expect(result.name).toBe('keep')
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/composables/useEntryDialogForm.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/composables/useEntryDialogForm.test.ts
git commit -m "test: add unit tests for useEntryDialogForm pure functions"
```

---

## Task 14: Tier 1 — `lib/api/helpers.ts`, `lib/api/CurrencyRates.ts` (parsePair), `lib/api/Entry.ts` (formatDate)

**Files:**
- Create: `webui/src/lib/api/helpers.test.ts`
- Create: `webui/src/lib/api/CurrencyRates.test.ts` (parsePair only — remaining API tests in Tier 2)
- Create: `webui/src/lib/api/Entry.test.ts` (formatDate only — remaining API tests in Tier 2)

**Step 1: Write helpers test**

```ts
// webui/src/lib/api/helpers.test.ts
import { describe, it, expect } from 'vitest'
import { with404Null } from './helpers'

describe('with404Null', () => {
  it('returns value on success', async () => {
    const result = await with404Null(() => Promise.resolve('data'))
    expect(result).toBe('data')
  })

  it('returns null on 404', async () => {
    const err = { response: { status: 404 } }
    const result = await with404Null(() => Promise.reject(err))
    expect(result).toBeNull()
  })

  it('re-throws non-404 errors', async () => {
    const err = { response: { status: 500 } }
    await expect(with404Null(() => Promise.reject(err))).rejects.toEqual(err)
  })
})
```

**Step 2: Write parsePair test**

```ts
// webui/src/lib/api/CurrencyRates.test.ts (only parsePair for now)
import { describe, it, expect } from 'vitest'
import { parsePair } from './CurrencyRates'

describe('parsePair', () => {
  it('splits standard pair', () => {
    expect(parsePair('USD/EUR')).toEqual(['USD', 'EUR'])
  })

  it('handles pair with no slash', () => {
    const [main, secondary] = parsePair('USDEUR')
    expect(main).toBe('USDEUR')
    expect(secondary).toBe('')
  })
})
```

**Step 3: Write formatDate test**

```ts
// webui/src/lib/api/Entry.test.ts (only formatDate for now)
import { describe, it, expect } from 'vitest'
import { formatDate } from './Entry'

describe('formatDate', () => {
  it('formats Date to YYYY-MM-DD', () => {
    expect(formatDate(new Date(2026, 2, 10))).toBe('2026-03-10')
  })

  it('returns empty string for non-Date input', () => {
    expect(formatDate('not a date' as any)).toBe('')
  })
})
```

**Step 4: Run all three tests**

Run: `cd webui && npx vitest run src/lib/api/helpers.test.ts src/lib/api/CurrencyRates.test.ts src/lib/api/Entry.test.ts`
Expected: All PASS

**Step 5: Commit**

```bash
git add webui/src/lib/api/helpers.test.ts webui/src/lib/api/CurrencyRates.test.ts webui/src/lib/api/Entry.test.ts
git commit -m "test: add unit tests for API helpers, parsePair, formatDate"
```

---

## Task 15: Tier 2 — `lib/api/Account.ts` (API wrapper tests)

**Files:**
- Create: `webui/src/lib/api/Account.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { apiClient } from './client'
import {
  getProviders,
  createAccountProvider,
  updateAccountProvider,
  deleteAccountProvider,
  createAccount,
  updateAccount,
  deleteAccount,
} from './Account'

vi.mock('./client', () => ({
  apiClient: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}))

beforeEach(() => vi.clearAllMocks())

describe('getProviders', () => {
  it('calls GET /fin/provider and returns items', async () => {
    const items = [{ id: 1, name: 'Bank', accounts: [] }]
    vi.mocked(apiClient.get).mockResolvedValue({ data: { items } })

    const result = await getProviders()
    expect(apiClient.get).toHaveBeenCalledWith('/fin/provider')
    expect(result).toEqual(items)
  })
})

describe('createAccountProvider', () => {
  it('calls POST /fin/provider', async () => {
    const payload = { name: 'New Bank' }
    vi.mocked(apiClient.post).mockResolvedValue({ data: { id: 1, ...payload } })

    await createAccountProvider(payload)
    expect(apiClient.post).toHaveBeenCalledWith('/fin/provider', payload)
  })
})

describe('updateAccountProvider', () => {
  it('calls PUT /fin/provider/:id', async () => {
    vi.mocked(apiClient.put).mockResolvedValue({ data: {} })

    await updateAccountProvider(5, { name: 'Updated' })
    expect(apiClient.put).toHaveBeenCalledWith('/fin/provider/5', { name: 'Updated' })
  })
})

describe('deleteAccountProvider', () => {
  it('calls DELETE /fin/provider/:id', async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({ data: {} })

    await deleteAccountProvider(5)
    expect(apiClient.delete).toHaveBeenCalledWith('/fin/provider/5')
  })
})

describe('createAccount', () => {
  it('calls POST /fin/account', async () => {
    const payload = { name: 'Checking', currency: 'CHF', type: 'checkin' }
    vi.mocked(apiClient.post).mockResolvedValue({ data: { id: 1, ...payload } })

    await createAccount(payload)
    expect(apiClient.post).toHaveBeenCalledWith('/fin/account', payload)
  })
})

describe('updateAccount', () => {
  it('calls PUT /fin/account/:id', async () => {
    vi.mocked(apiClient.put).mockResolvedValue({ data: {} })

    await updateAccount(10, { name: 'Renamed' })
    expect(apiClient.put).toHaveBeenCalledWith('/fin/account/10', { name: 'Renamed' })
  })
})

describe('deleteAccount', () => {
  it('calls DELETE /fin/account/:id', async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({ data: {} })

    await deleteAccount(10)
    expect(apiClient.delete).toHaveBeenCalledWith('/fin/account/10')
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/lib/api/Account.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/lib/api/Account.test.ts
git commit -m "test: add unit tests for Account API wrapper"
```

---

## Task 16: Tier 2 — `lib/api/Entry.ts` (remaining API tests)

**Files:**
- Modify: `webui/src/lib/api/Entry.test.ts` (add API wrapper tests below existing formatDate tests)

**Step 1: Add API wrapper tests**

Append to existing `Entry.test.ts`:

```ts
// Add below existing formatDate tests
import { apiClient } from './client'
import { getEntries, createEntry, updateEntry, deleteEntry } from './Entry'

vi.mock('./client', () => ({
  apiClient: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}))

beforeEach(() => vi.clearAllMocks())

describe('getEntries', () => {
  it('builds query string with date range and pagination', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({
      data: { items: [], total: 0, page: 1, limit: 25, priorBalance: 0 },
    })

    await getEntries({
      startDate: new Date(2026, 0, 1),
      endDate: new Date(2026, 0, 31),
      page: 2,
      limit: 25,
      accountIds: ['1', '3'],
    })

    const url = vi.mocked(apiClient.get).mock.calls[0][0]
    expect(url).toContain('startDate=2026-01-01')
    expect(url).toContain('endDate=2026-01-31')
    expect(url).toContain('page=2')
    expect(url).toContain('limit=25')
  })

  it('includes accountIds as repeated params', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({
      data: { items: [], total: 0 },
    })

    await getEntries({
      startDate: new Date(2026, 0, 1),
      endDate: new Date(2026, 0, 31),
      accountIds: ['1', '3'],
    })

    const url = vi.mocked(apiClient.get).mock.calls[0][0]
    expect(url).toContain('accountIds')
  })
})

describe('createEntry', () => {
  it('calls POST /fin/entries', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ data: { id: 1 } })

    await createEntry({ type: 'expense', amount: 50 } as any)
    expect(apiClient.post).toHaveBeenCalledWith('/fin/entries', expect.any(Object))
  })
})

describe('updateEntry', () => {
  it('calls PUT /fin/entries/:id', async () => {
    vi.mocked(apiClient.put).mockResolvedValue({ data: {} })

    await updateEntry({ id: 5, amount: 100 } as any)
    expect(apiClient.put).toHaveBeenCalledWith('/fin/entries/5', expect.any(Object))
  })
})

describe('deleteEntry', () => {
  it('calls DELETE /fin/entries/:id', async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({ data: {} })

    await deleteEntry('5')
    expect(apiClient.delete).toHaveBeenCalledWith('/fin/entries/5')
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/lib/api/Entry.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/lib/api/Entry.test.ts
git commit -m "test: add API wrapper tests for Entry"
```

---

## Task 17: Tier 2 — `lib/api/Category.ts`

**Files:**
- Create: `webui/src/lib/api/Category.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { apiClient } from './client'
import {
  GetIncomeCategories,
  CreateIncomeCategory,
  UpdateIncomeCategory,
  deleteIncomeCategory,
  GetExpenseCategories,
  CreateExpenseCategory,
  UpdateExpenseCategory,
  DeleteExpenseCategory,
} from './Category'

vi.mock('./client', () => ({
  apiClient: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}))

beforeEach(() => vi.clearAllMocks())

describe('Income categories', () => {
  it('GetIncomeCategories calls GET /fin/category/income', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { items: [] } })
    await GetIncomeCategories()
    expect(apiClient.get).toHaveBeenCalledWith('/fin/category/income')
  })

  it('CreateIncomeCategory calls POST /fin/category/income', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ data: {} })
    await CreateIncomeCategory({ name: 'Salary' } as any)
    expect(apiClient.post).toHaveBeenCalledWith('/fin/category/income', expect.any(Object))
  })

  it('UpdateIncomeCategory calls PUT /fin/category/income/:id', async () => {
    vi.mocked(apiClient.put).mockResolvedValue({ data: {} })
    await UpdateIncomeCategory({ id: 1, payload: { name: 'Updated' } } as any)
    expect(apiClient.put).toHaveBeenCalledWith('/fin/category/income/1', expect.any(Object))
  })

  it('deleteIncomeCategory calls DELETE /fin/category/income/:id', async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({ data: {} })
    await deleteIncomeCategory(1)
    expect(apiClient.delete).toHaveBeenCalledWith('/fin/category/income/1')
  })
})

describe('Expense categories', () => {
  it('GetExpenseCategories calls GET /fin/category/expense', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { items: [] } })
    await GetExpenseCategories()
    expect(apiClient.get).toHaveBeenCalledWith('/fin/category/expense')
  })

  it('CreateExpenseCategory calls POST /fin/category/expense', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ data: {} })
    await CreateExpenseCategory({ name: 'Food' } as any)
    expect(apiClient.post).toHaveBeenCalledWith('/fin/category/expense', expect.any(Object))
  })

  it('UpdateExpenseCategory calls PUT /fin/category/expense/:id', async () => {
    vi.mocked(apiClient.put).mockResolvedValue({ data: {} })
    await UpdateExpenseCategory({ id: 2, payload: { name: 'Updated' } } as any)
    expect(apiClient.put).toHaveBeenCalledWith('/fin/category/expense/2', expect.any(Object))
  })

  it('DeleteExpenseCategory calls DELETE /fin/category/expense/:id', async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({ data: {} })
    await DeleteExpenseCategory(2)
    expect(apiClient.delete).toHaveBeenCalledWith('/fin/category/expense/2')
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/lib/api/Category.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/lib/api/Category.test.ts
git commit -m "test: add unit tests for Category API wrapper"
```

---

## Task 18: Tier 2 — `lib/api/report.ts`

**Files:**
- Create: `webui/src/lib/api/report.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { apiClient } from './client'
import { getBalanceReport, getAccountBalance, getIncomeExpenseReport } from './report'

vi.mock('./client', () => ({
  apiClient: { get: vi.fn() },
}))

beforeEach(() => vi.clearAllMocks())

describe('getBalanceReport', () => {
  it('builds query with comma-joined accountIds', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: {} })

    await getBalanceReport([1, 2, 3], 12, '2026-01-01')
    const url = vi.mocked(apiClient.get).mock.calls[0][0]
    expect(url).toContain('accountIds=1%2C2%2C3')
    expect(url).toContain('steps=12')
    expect(url).toContain('startDate=2026-01-01')
  })
})

describe('getAccountBalance', () => {
  it('extracts nested Sum value', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({
      data: { accounts: { 5: [{ Sum: 1234.56 }] } },
    })

    const result = await getAccountBalance(5, '2026-01-01')
    expect(result).toBe(1234.56)
  })

  it('returns 0 when account data missing', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({
      data: { accounts: {} },
    })

    const result = await getAccountBalance(5, '2026-01-01')
    expect(result).toBe(0)
  })
})

describe('getIncomeExpenseReport', () => {
  it('passes date range as query params', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: [] })

    await getIncomeExpenseReport('2026-01-01', '2026-01-31')
    const url = vi.mocked(apiClient.get).mock.calls[0][0]
    expect(url).toContain('startDate=2026-01-01')
    expect(url).toContain('endDate=2026-01-31')
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/lib/api/report.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/lib/api/report.test.ts
git commit -m "test: add unit tests for report API wrapper"
```

---

## Task 19: Tier 2 — `lib/api/CurrencyRates.ts` (remaining API tests)

**Files:**
- Modify: `webui/src/lib/api/CurrencyRates.test.ts` (add API tests below existing parsePair tests)

**Step 1: Add API wrapper tests**

Append to existing file:

```ts
import { apiClient } from './client'
import { getFXPairs, getRateHistory, getLatestRate, createRate, deleteRate } from './CurrencyRates'

vi.mock('./client', () => ({
  apiClient: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}))

beforeEach(() => vi.clearAllMocks())

describe('getFXPairs', () => {
  it('calls GET /fin/fx/pairs', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { pairs: ['USD/EUR'] } })
    const result = await getFXPairs()
    expect(apiClient.get).toHaveBeenCalledWith('/fin/fx/pairs')
    expect(result).toEqual(['USD/EUR'])
  })
})

describe('getRateHistory', () => {
  it('URL-encodes currency names', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { items: [] } })
    await getRateHistory('USD', 'EUR', '2026-01-01', '2026-01-31')
    const url = vi.mocked(apiClient.get).mock.calls[0][0]
    expect(url).toContain('/fin/fx/')
    expect(url).toContain('start=2026-01-01')
  })
})

describe('getLatestRate', () => {
  it('returns null on 404', async () => {
    vi.mocked(apiClient.get).mockRejectedValue({ response: { status: 404 } })
    const result = await getLatestRate('USD', 'EUR')
    expect(result).toBeNull()
  })
})

describe('deleteRate', () => {
  it('calls DELETE /fin/fx/rates/:id', async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({ data: {} })
    await deleteRate(5)
    expect(apiClient.delete).toHaveBeenCalledWith('/fin/fx/rates/5')
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/lib/api/CurrencyRates.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/lib/api/CurrencyRates.test.ts
git commit -m "test: add API wrapper tests for CurrencyRates"
```

---

## Task 20: Tier 2 — `lib/api/MarketData.ts`

**Files:**
- Create: `webui/src/lib/api/MarketData.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { apiClient } from './client'
import { getMarketDataSymbols, getPriceHistory, getLatestPrice, createPrice, deletePrice } from './MarketData'

vi.mock('./client', () => ({
  apiClient: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}))

beforeEach(() => vi.clearAllMocks())

describe('getMarketDataSymbols', () => {
  it('calls GET /fin/marketdata/symbols', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { symbols: ['AAPL', 'GOOG'] } })
    const result = await getMarketDataSymbols()
    expect(result).toEqual(['AAPL', 'GOOG'])
  })
})

describe('getPriceHistory', () => {
  it('URL-encodes symbol and passes date params', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { items: [] } })
    await getPriceHistory('AAPL', '2026-01-01', '2026-01-31')
    const url = vi.mocked(apiClient.get).mock.calls[0][0]
    expect(url).toContain(encodeURIComponent('AAPL'))
    expect(url).toContain('start=2026-01-01')
  })
})

describe('getLatestPrice', () => {
  it('returns null on 404', async () => {
    vi.mocked(apiClient.get).mockRejectedValue({ response: { status: 404 } })
    const result = await getLatestPrice('AAPL')
    expect(result).toBeNull()
  })

  it('returns price data on success', async () => {
    const priceData = { date: '2026-01-01', close: 150 }
    vi.mocked(apiClient.get).mockResolvedValue({ data: priceData })
    const result = await getLatestPrice('AAPL')
    expect(result).toEqual(priceData)
  })
})

describe('deletePrice', () => {
  it('calls DELETE /fin/marketdata/prices/:id', async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({ data: {} })
    await deletePrice(5)
    expect(apiClient.delete).toHaveBeenCalledWith('/fin/marketdata/prices/5')
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/lib/api/MarketData.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/lib/api/MarketData.test.ts
git commit -m "test: add unit tests for MarketData API wrapper"
```

---

## Task 21: Tier 2 — `lib/api/CsvImport.ts`

**Files:**
- Create: `webui/src/lib/api/CsvImport.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { apiClient } from './client'
import { getProfiles, createProfile, deleteProfile, parseCSV, previewCSV } from './CsvImport'

vi.mock('./client', () => ({
  apiClient: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}))

beforeEach(() => vi.clearAllMocks())

describe('getProfiles', () => {
  it('calls GET /import/profiles', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: [] })
    await getProfiles()
    expect(apiClient.get).toHaveBeenCalledWith('/import/profiles')
  })
})

describe('createProfile', () => {
  it('calls POST /import/profiles', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ data: { id: 1 } })
    await createProfile({ name: 'Profile 1' } as any)
    expect(apiClient.post).toHaveBeenCalledWith('/import/profiles', expect.any(Object))
  })
})

describe('deleteProfile', () => {
  it('calls DELETE /import/profiles/:id', async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({ data: {} })
    await deleteProfile(3)
    expect(apiClient.delete).toHaveBeenCalledWith('/import/profiles/3')
  })
})

describe('parseCSV', () => {
  it('sends FormData with file and accountId', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ data: { rows: [] } })
    const file = new File(['csv data'], 'test.csv', { type: 'text/csv' })
    await parseCSV(1, file)

    const [url, formData, config] = vi.mocked(apiClient.post).mock.calls[0]
    expect(url).toBe('/import/parse')
    expect(formData).toBeInstanceOf(FormData)
    expect(config?.headers?.['Content-Type']).toBe('multipart/form-data')
  })
})

describe('previewCSV', () => {
  it('sends FormData with file and config fields', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ data: {} })
    const file = new File(['csv data'], 'test.csv')
    await previewCSV(file, { csvSeparator: ',', dateColumn: 0, dateFormat: 'YYYY-MM-DD' } as any)

    const [url, formData] = vi.mocked(apiClient.post).mock.calls[0]
    expect(url).toBe('/import/preview')
    expect(formData).toBeInstanceOf(FormData)
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/lib/api/CsvImport.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/lib/api/CsvImport.test.ts
git commit -m "test: add unit tests for CsvImport API wrapper"
```

---

## Task 22: Tier 2 — `lib/api/Attachment.ts`

**Files:**
- Create: `webui/src/lib/api/Attachment.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { apiClient } from './client'
import { uploadAttachment, getAttachmentUrl, deleteAttachment } from './Attachment'

vi.mock('./client', () => ({
  apiClient: { post: vi.fn(), delete: vi.fn() },
}))

beforeEach(() => vi.clearAllMocks())

describe('uploadAttachment', () => {
  it('sends FormData to POST /fin/entries/:txId/attachment', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ data: { id: 1 } })
    const file = new File(['data'], 'receipt.pdf')
    await uploadAttachment(42, file)

    const [url, formData] = vi.mocked(apiClient.post).mock.calls[0]
    expect(url).toBe('/fin/entries/42/attachment')
    expect(formData).toBeInstanceOf(FormData)
  })
})

describe('getAttachmentUrl', () => {
  it('builds URL with txId', () => {
    const url = getAttachmentUrl(42)
    expect(url).toContain('/fin/entries/42/attachment')
  })
})

describe('deleteAttachment', () => {
  it('calls DELETE /fin/entries/:txId/attachment', async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({ data: {} })
    await deleteAttachment(42)
    expect(apiClient.delete).toHaveBeenCalledWith('/fin/entries/42/attachment')
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/lib/api/Attachment.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/lib/api/Attachment.test.ts
git commit -m "test: add unit tests for Attachment API wrapper"
```

---

## Task 23: Tier 2 — `lib/api/Portfolio.ts`

**Files:**
- Create: `webui/src/lib/api/Portfolio.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { apiClient } from './client'
import { getPositions, getPositionDetail, getLots, getTrades } from './Portfolio'

vi.mock('./client', () => ({
  apiClient: { get: vi.fn() },
}))

beforeEach(() => vi.clearAllMocks())

describe('getPositions', () => {
  it('calls GET /fin/portfolio/positions without params', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { items: [] } })
    await getPositions()
    const url = vi.mocked(apiClient.get).mock.calls[0][0]
    expect(url).toContain('/fin/portfolio/positions')
  })

  it('includes accountId when provided', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { items: [] } })
    await getPositions(5)
    const url = vi.mocked(apiClient.get).mock.calls[0][0]
    expect(url).toContain('accountId=5')
  })
})

describe('getPositionDetail', () => {
  it('calls GET /fin/portfolio/positions/:accountId/:instrumentId', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: {} })
    await getPositionDetail(1, 2)
    expect(apiClient.get).toHaveBeenCalledWith('/fin/portfolio/positions/1/2')
  })
})

describe('getLots', () => {
  it('calls GET /fin/portfolio/lots with optional params', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { items: [] } })
    await getLots(1, 2)
    const url = vi.mocked(apiClient.get).mock.calls[0][0]
    expect(url).toContain('accountId=1')
    expect(url).toContain('instrumentId=2')
  })
})

describe('getTrades', () => {
  it('calls GET /fin/portfolio/trades', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { items: [] } })
    await getTrades()
    const url = vi.mocked(apiClient.get).mock.calls[0][0]
    expect(url).toContain('/fin/portfolio/trades')
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/lib/api/Portfolio.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/lib/api/Portfolio.test.ts
git commit -m "test: add unit tests for Portfolio API wrapper"
```

---

## Task 24: Tier 2 — `lib/api/Tasks.ts`

**Files:**
- Create: `webui/src/lib/api/Tasks.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { apiClient } from './client'
import { listTasks, getTask, listExecutions, triggerTask, cancelExecution, getExecutionLog, upsertTask, deleteTaskSchedule } from './Tasks'

vi.mock('./client', () => ({
  apiClient: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
}))

beforeEach(() => vi.clearAllMocks())

describe('listTasks', () => {
  it('calls GET /tasks', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { tasks: [] } })
    await listTasks()
    expect(apiClient.get).toHaveBeenCalledWith('/tasks')
  })
})

describe('getTask', () => {
  it('URL-encodes task name', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: {} })
    await getTask('my task')
    const url = vi.mocked(apiClient.get).mock.calls[0][0]
    expect(url).toContain(encodeURIComponent('my task'))
  })
})

describe('triggerTask', () => {
  it('calls POST /tasks/:name/trigger', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ data: { execution_id: 'abc' } })
    const result = await triggerTask('sync')
    expect(result).toBe('abc')
  })
})

describe('cancelExecution', () => {
  it('calls POST /tasks/executions/:id/cancel', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ data: {} })
    await cancelExecution('exec-1')
    const url = vi.mocked(apiClient.post).mock.calls[0][0]
    expect(url).toContain('exec-1')
    expect(url).toContain('cancel')
  })
})

describe('getExecutionLog', () => {
  it('requests text response type', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: 'log output' })
    const result = await getExecutionLog('exec-1')
    const config = vi.mocked(apiClient.get).mock.calls[0][1]
    expect(config?.responseType).toBe('text')
    expect(result).toBe('log output')
  })
})

describe('deleteTaskSchedule', () => {
  it('calls DELETE /tasks/:name', async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({ data: {} })
    await deleteTaskSchedule('sync')
    const url = vi.mocked(apiClient.delete).mock.calls[0][0]
    expect(url).toContain('sync')
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/lib/api/Tasks.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/lib/api/Tasks.test.ts
git commit -m "test: add unit tests for Tasks API wrapper"
```

---

## Task 25: Tier 2 — `lib/api/Backup.ts`

**Files:**
- Create: `webui/src/lib/api/Backup.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { apiClient } from './client'
import { GetBackupFiles, DeleteBackupFile, CreateBackup, RestoreBackup } from './Backup'

vi.mock('./client', () => ({
  apiClient: { get: vi.fn(), post: vi.fn(), delete: vi.fn() },
}))

beforeEach(() => vi.clearAllMocks())

describe('GetBackupFiles', () => {
  it('calls GET /backup', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { files: [] } })
    await GetBackupFiles()
    expect(apiClient.get).toHaveBeenCalledWith('/backup')
  })
})

describe('DeleteBackupFile', () => {
  it('calls DELETE /backup/:id', async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({ data: {} })
    await DeleteBackupFile('backup-1')
    expect(apiClient.delete).toHaveBeenCalledWith('/backup/backup-1')
  })
})

describe('CreateBackup', () => {
  it('calls POST /backup', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ data: {} })
    await CreateBackup()
    expect(apiClient.post).toHaveBeenCalledWith('/backup')
  })
})

describe('RestoreBackup', () => {
  it('sends FormData with file', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ data: {} })
    const file = new File(['data'], 'backup.zip')
    await RestoreBackup(file)

    const [url, formData] = vi.mocked(apiClient.post).mock.calls[0]
    expect(url).toBe('/restore')
    expect(formData).toBeInstanceOf(FormData)
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/lib/api/Backup.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/lib/api/Backup.test.ts
git commit -m "test: add unit tests for Backup API wrapper"
```

---

## Task 26: Tier 3 — `store/settingsStore.js`

**Files:**
- Create: `webui/src/store/settingsStore.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import axios from 'axios'
import { useSettingsStore } from './settingsStore'

vi.mock('axios')

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

describe('settingsStore', () => {
  it('starts with isLoaded = false', () => {
    const store = useSettingsStore()
    expect(store.isLoaded).toBe(false)
  })

  it('sets isLoaded after successful fetch', async () => {
    vi.mocked(axios.get).mockResolvedValue({
      data: {
        dateFormat: 'YYYY-MM-DD',
        mainCurrency: 'CHF',
        currencies: ['CHF', 'EUR'],
        instruments: true,
        version: '1.0.0',
      },
    })

    const store = useSettingsStore()
    await store.fetchSettings()

    expect(store.isLoaded).toBe(true)
    expect(store.mainCurrency).toBe('CHF')
    expect(store.dateFormat).toBe('YYYY-MM-DD')
  })

  it('hasMultipleCurrencies is true when 2+ currencies', async () => {
    vi.mocked(axios.get).mockResolvedValue({
      data: { currencies: ['CHF', 'EUR'] },
    })

    const store = useSettingsStore()
    await store.fetchSettings()
    expect(store.hasMultipleCurrencies).toBe(true)
  })

  it('hasMultipleCurrencies is false when 1 currency', async () => {
    vi.mocked(axios.get).mockResolvedValue({
      data: { currencies: ['CHF'] },
    })

    const store = useSettingsStore()
    await store.fetchSettings()
    expect(store.hasMultipleCurrencies).toBe(false)
  })

  it('$reset restores initial state', async () => {
    vi.mocked(axios.get).mockResolvedValue({
      data: { mainCurrency: 'CHF', currencies: ['CHF'] },
    })

    const store = useSettingsStore()
    await store.fetchSettings()
    store.$reset()

    expect(store.isLoaded).toBe(false)
    expect(store.mainCurrency).toBe('')
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/store/settingsStore.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/store/settingsStore.test.ts
git commit -m "test: add unit tests for settingsStore"
```

---

## Task 27: Tier 3 — `store/uiStore.js`

**Files:**
- Create: `webui/src/store/uiStore.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useUiStore } from './uiStore'

beforeEach(() => {
  setActivePinia(createPinia())
})

describe('uiStore', () => {
  it('starts with drawer hidden', () => {
    const store = useUiStore()
    expect(store.isDrawerVisible).toBe(false)
  })

  it('openDrawer sets visible', () => {
    const store = useUiStore()
    store.openDrawer()
    expect(store.isDrawerVisible).toBe(true)
  })

  it('closeDrawer sets hidden', () => {
    const store = useUiStore()
    store.openDrawer()
    store.closeDrawer()
    expect(store.isDrawerVisible).toBe(false)
  })

  it('toggleDrawer flips state', () => {
    const store = useUiStore()
    store.toggleDrawer()
    expect(store.isDrawerVisible).toBe(true)
    store.toggleDrawer()
    expect(store.isDrawerVisible).toBe(false)
  })

  it('secondary drawer works independently', () => {
    const store = useUiStore()
    store.openSecondaryDrawer()
    expect(store.isSecondaryDrawerVisible).toBe(true)
    expect(store.isDrawerVisible).toBe(false)
  })

  it('checkScreenWidth opens drawer on wide screen', () => {
    const store = useUiStore()
    Object.defineProperty(window, 'innerWidth', { value: 1200, writable: true })
    store.checkScreenWidth()
    expect(store.isDrawerVisible).toBe(true)
  })

  it('checkScreenWidth closes drawer on narrow screen', () => {
    const store = useUiStore()
    store.openDrawer()
    Object.defineProperty(window, 'innerWidth', { value: 800, writable: true })
    store.checkScreenWidth()
    expect(store.isDrawerVisible).toBe(false)
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/store/uiStore.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/store/uiStore.test.ts
git commit -m "test: add unit tests for uiStore"
```

---

## Task 28: Tier 3 — `lib/user/userstore.js`

**Files:**
- Create: `webui/src/lib/user/userstore.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import axios from 'axios'
import { useUserStore } from './userstore'

vi.mock('axios')

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

describe('userstore', () => {
  it('starts logged out', () => {
    const store = useUserStore()
    expect(store.isLoggedIn).toBe(false)
  })

  it('checkState sets isLoggedIn on success', async () => {
    vi.mocked(axios.get).mockResolvedValue({
      data: { loggedIn: true, user: 'admin' },
    })

    const store = useUserStore()
    await store.checkState()
    expect(store.isLoggedIn).toBe(true)
    expect(store.loggedInUser).toBe('admin')
  })

  it('registerLogoutAction stores callback', () => {
    const store = useUserStore()
    const callback = vi.fn()
    store.registerLogoutAction(callback)

    // Trigger logout
    vi.mocked(axios.post).mockResolvedValue({ data: {} })
    store.logout()

    expect(callback).toHaveBeenCalled()
  })

  it('login sets wrongPwErr on 401', async () => {
    vi.mocked(axios.post).mockRejectedValue({
      response: { status: 401 },
    })

    const store = useUserStore()
    await store.login('user', 'wrong')
    expect(store.wrongPwErr).toBe(true)
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/lib/user/userstore.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/lib/user/userstore.test.ts
git commit -m "test: add unit tests for userStore"
```

---

## Task 29: Tier 4 — `composables/queryUtils.ts`

**Files:**
- Create: `webui/src/composables/queryUtils.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect, vi } from 'vitest'
import { invalidateAndRefetch } from './queryUtils'

describe('invalidateAndRefetch', () => {
  it('calls invalidateQueries and refetchQueries', () => {
    const queryClient = {
      invalidateQueries: vi.fn(),
      refetchQueries: vi.fn(),
    }

    invalidateAndRefetch(queryClient as any, ['accounts'])

    expect(queryClient.invalidateQueries).toHaveBeenCalledWith({ queryKey: ['accounts'] })
    expect(queryClient.refetchQueries).toHaveBeenCalledWith({ queryKey: ['accounts'] })
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/composables/queryUtils.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/composables/queryUtils.test.ts
git commit -m "test: add unit tests for queryUtils"
```

---

## Task 30: Tier 4 — `composables/useAccounts.ts`

**Files:**
- Create: `webui/src/composables/useAccounts.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect, vi, afterEach } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { renderComposable } from '../test/helpers'
import { getProviders } from '../lib/api/Account'
import { useAccounts } from './useAccounts'

vi.mock('../lib/api/Account', () => ({
  getProviders: vi.fn(),
  createAccountProvider: vi.fn(),
  updateAccountProvider: vi.fn(),
  deleteAccountProvider: vi.fn(),
  createAccount: vi.fn(),
  updateAccount: vi.fn(),
  deleteAccount: vi.fn(),
}))

describe('useAccounts', () => {
  let unmount: () => void

  afterEach(() => unmount?.())

  it('returns providers from API', async () => {
    vi.mocked(getProviders).mockResolvedValue([
      { id: 1, name: 'Bank', accounts: [{ id: 10, name: 'Checking', currency: 'CHF', type: 'checkin' }] },
    ])

    const { result, unmount: u } = renderComposable(() => useAccounts())
    unmount = u
    await flushPromises()

    expect(result.accounts.value).toHaveLength(1)
    expect(result.accounts.value[0].name).toBe('Bank')
  })

  it('normalizes providers with null accounts', async () => {
    vi.mocked(getProviders).mockResolvedValue([
      { id: 1, name: 'Bank', accounts: null },
    ])

    const { result, unmount: u } = renderComposable(() => useAccounts())
    unmount = u
    await flushPromises()

    expect(result.accounts.value[0].accounts).toEqual([])
  })

  it('returns empty array when no providers', async () => {
    vi.mocked(getProviders).mockResolvedValue([])

    const { result, unmount: u } = renderComposable(() => useAccounts())
    unmount = u
    await flushPromises()

    expect(result.accounts.value).toEqual([])
  })

  it('exposes loading state', () => {
    vi.mocked(getProviders).mockReturnValue(new Promise(() => {})) // never resolves

    const { result, unmount: u } = renderComposable(() => useAccounts())
    unmount = u

    expect(result.isLoading.value).toBe(true)
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/composables/useAccounts.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/composables/useAccounts.test.ts
git commit -m "test: add unit tests for useAccounts composable"
```

---

## Task 31: Tier 4 — `composables/useCategories.ts`

**Files:**
- Create: `webui/src/composables/useCategories.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect, vi, afterEach } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { renderComposable } from '../test/helpers'
import { GetIncomeCategories, GetExpenseCategories } from '../lib/api/Category'
import { useCategories } from './useCategories'

vi.mock('../lib/api/Category', () => ({
  GetIncomeCategories: vi.fn(),
  CreateIncomeCategory: vi.fn(),
  UpdateIncomeCategory: vi.fn(),
  deleteIncomeCategory: vi.fn(),
  GetExpenseCategories: vi.fn(),
  CreateExpenseCategory: vi.fn(),
  UpdateExpenseCategory: vi.fn(),
  DeleteExpenseCategory: vi.fn(),
}))

describe('useCategories', () => {
  let unmount: () => void

  afterEach(() => unmount?.())

  it('fetches income categories', async () => {
    vi.mocked(GetIncomeCategories).mockResolvedValue([{ id: 1, name: 'Salary' }])
    vi.mocked(GetExpenseCategories).mockResolvedValue([])

    const { result, unmount: u } = renderComposable(() => useCategories())
    unmount = u
    await flushPromises()

    expect(result.incomeCategories.data.value).toHaveLength(1)
  })

  it('fetches expense categories', async () => {
    vi.mocked(GetIncomeCategories).mockResolvedValue([])
    vi.mocked(GetExpenseCategories).mockResolvedValue([{ id: 1, name: 'Food' }])

    const { result, unmount: u } = renderComposable(() => useCategories())
    unmount = u
    await flushPromises()

    expect(result.expenseCategories.data.value).toHaveLength(1)
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/composables/useCategories.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/composables/useCategories.test.ts
git commit -m "test: add unit tests for useCategories composable"
```

---

## Task 32: Tier 4 — `composables/useCategoryTree.ts`

**Files:**
- Create: `webui/src/composables/useCategoryTree.test.ts`

**Step 1: Write tests**

```ts
import { describe, it, expect, vi, afterEach } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { renderComposable } from '../test/helpers'
import { GetIncomeCategories, GetExpenseCategories } from '../lib/api/Category'
import { useCategoryTree } from './useCategoryTree'

vi.mock('../lib/api/Category', () => ({
  GetIncomeCategories: vi.fn(),
  CreateIncomeCategory: vi.fn(),
  UpdateIncomeCategory: vi.fn(),
  deleteIncomeCategory: vi.fn(),
  GetExpenseCategories: vi.fn(),
  CreateExpenseCategory: vi.fn(),
  UpdateExpenseCategory: vi.fn(),
  DeleteExpenseCategory: vi.fn(),
}))

describe('useCategoryTree', () => {
  let unmount: () => void

  afterEach(() => unmount?.())

  it('transforms flat categories into tree', async () => {
    vi.mocked(GetExpenseCategories).mockResolvedValue([
      { id: 1, parentId: null, name: 'Food' },
      { id: 2, parentId: 1, name: 'Groceries' },
    ])
    vi.mocked(GetIncomeCategories).mockResolvedValue([])

    const { result, unmount: u } = renderComposable(() => useCategoryTree())
    unmount = u
    await flushPromises()

    expect(result.ExpenseTreeData.value).toHaveLength(1)
    expect(result.ExpenseTreeData.value[0].children).toHaveLength(1)
  })

  it('returns empty tree when no categories', async () => {
    vi.mocked(GetExpenseCategories).mockResolvedValue([])
    vi.mocked(GetIncomeCategories).mockResolvedValue([])

    const { result, unmount: u } = renderComposable(() => useCategoryTree())
    unmount = u
    await flushPromises()

    expect(result.ExpenseTreeData.value).toEqual([])
    expect(result.IncomeTreeData.value).toEqual([])
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/composables/useCategoryTree.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/composables/useCategoryTree.test.ts
git commit -m "test: add unit tests for useCategoryTree composable"
```

---

## Task 33: Tier 4 — `composables/useTasks.ts`

**Files:**
- Create: `webui/src/composables/useTasks.test.ts`

**Step 1: Write tests**

Note: Focus on `useTaskExecutions` (pure filtering) and status mapping functions. The full `useTasks` composable involves toasts and complex polling — test the derivation logic.

```ts
import { describe, it, expect, vi, afterEach } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { renderComposable } from '../test/helpers'
import { listTasks, listExecutions } from '../lib/api/Tasks'
import { useTasks } from './useTasks'

vi.mock('../lib/api/Tasks', () => ({
  listTasks: vi.fn(),
  listExecutions: vi.fn(),
  triggerTask: vi.fn(),
  cancelExecution: vi.fn(),
  getExecutionLog: vi.fn(),
  upsertTask: vi.fn(),
  patchTask: vi.fn(),
  deleteTaskSchedule: vi.fn(),
}))

// Mock PrimeVue toast
vi.mock('primevue/usetoast', () => ({
  useToast: () => ({ add: vi.fn() }),
}))

describe('useTasks', () => {
  let unmount: () => void

  afterEach(() => unmount?.())

  it('derives tasks with last execution info', async () => {
    vi.mocked(listTasks).mockResolvedValue([
      { name: 'sync', cron_expression: '0 * * * *', enabled: true },
    ])
    vi.mocked(listExecutions).mockResolvedValue([
      { id: 'exec-1', task_name: 'sync', status: 'complete', queued_at: '2026-03-10T00:00:00Z', started_at: '2026-03-10T00:00:01Z', finished_at: '2026-03-10T00:00:05Z' },
    ])

    const { result, unmount: u } = renderComposable(() => useTasks())
    unmount = u
    await flushPromises()

    expect(result.tasks.value).toHaveLength(1)
    expect(result.tasks.value[0].name).toBe('sync')
  })

  it('getStatusSeverity returns correct severity', async () => {
    vi.mocked(listTasks).mockResolvedValue([])
    vi.mocked(listExecutions).mockResolvedValue([])

    const { result, unmount: u } = renderComposable(() => useTasks())
    unmount = u

    expect(result.getStatusSeverity('complete')).toBe('success')
    expect(result.getStatusSeverity('failed')).toBe('danger')
    expect(result.getStatusSeverity('running')).toBe('warn')
  })

  it('getStatusLabel maps waiting to queued', async () => {
    vi.mocked(listTasks).mockResolvedValue([])
    vi.mocked(listExecutions).mockResolvedValue([])

    const { result, unmount: u } = renderComposable(() => useTasks())
    unmount = u

    expect(result.getStatusLabel('waiting')).toBe('queued')
  })
})
```

**Step 2: Run tests**

Run: `cd webui && npx vitest run src/composables/useTasks.test.ts`
Expected: All PASS

**Step 3: Commit**

```bash
git add webui/src/composables/useTasks.test.ts
git commit -m "test: add unit tests for useTasks composable"
```

---

## Task 34: Run Full Test Suite

**Step 1: Run all unit tests**

Run: `cd webui && npx vitest run`
Expected: All tests PASS

**Step 2: Check coverage**

Run: `cd webui && npx vitest run --coverage`
Expected: Coverage report generated — review for gaps

**Step 3: Final commit**

No code changes — just verify everything works together.

---

## Summary

| Task | Tier | File | Test Count (est.) |
|------|------|------|-------------------|
| 1 | Infra | test/helpers.ts | 0 (helper only) |
| 2 | 1 | utils/currency.test.ts | 7 |
| 3 | 1 | utils/format.test.ts | 10 |
| 4 | 1 | utils/date.test.ts | 5 |
| 5 | 1 | utils/dateRange.test.ts | 4 |
| 6 | 1 | utils/apiError.test.ts | 10 |
| 7 | 1 | utils/convertToTree.test.ts | 7 |
| 8 | 1 | utils/categoryUtils.test.ts | 8 |
| 9 | 1 | utils/accountUtils.test.ts | 5 |
| 10 | 1 | utils/entryDisplay.test.ts | 8 |
| 11 | 1 | utils/entryValidation.test.ts | 2 |
| 12 | 1 | types/account.test.ts | 10 |
| 13 | 1 | composables/useEntryDialogForm.test.ts | 10 |
| 14 | 1 | lib/api/{helpers,CurrencyRates,Entry}.test.ts | 7 |
| 15 | 2 | lib/api/Account.test.ts | 7 |
| 16 | 2 | lib/api/Entry.test.ts (API part) | 5 |
| 17 | 2 | lib/api/Category.test.ts | 8 |
| 18 | 2 | lib/api/report.test.ts | 4 |
| 19 | 2 | lib/api/CurrencyRates.test.ts (API part) | 4 |
| 20 | 2 | lib/api/MarketData.test.ts | 4 |
| 21 | 2 | lib/api/CsvImport.test.ts | 5 |
| 22 | 2 | lib/api/Attachment.test.ts | 3 |
| 23 | 2 | lib/api/Portfolio.test.ts | 4 |
| 24 | 2 | lib/api/Tasks.test.ts | 6 |
| 25 | 2 | lib/api/Backup.test.ts | 4 |
| 26 | 3 | store/settingsStore.test.ts | 5 |
| 27 | 3 | store/uiStore.test.ts | 7 |
| 28 | 3 | lib/user/userstore.test.ts | 4 |
| 29 | 4 | composables/queryUtils.test.ts | 1 |
| 30 | 4 | composables/useAccounts.test.ts | 4 |
| 31 | 4 | composables/useCategories.test.ts | 2 |
| 32 | 4 | composables/useCategoryTree.test.ts | 2 |
| 33 | 4 | composables/useTasks.test.ts | 3 |
| **Total** | | **33 tasks** | **~170 tests** |
