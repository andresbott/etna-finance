# Frontend Unit Test Strategy Design

## Goal

Add comprehensive Vitest unit tests across all frontend logic layers (utils, API wrappers, stores, composables) to prevent regressions and improve confidence in refactoring.

## Decisions

- **Test runner:** Vitest (existing `unit` project config, jsdom environment)
- **File layout:** Colocated — `foo.ts` gets `foo.test.ts` in the same directory
- **Mocking strategy (layered):**
  - Utils/types: No mocking (pure functions)
  - API wrappers: Mock `apiClient` from `lib/api/client.ts` via `vi.mock('./client')`
  - Stores: Mock `axios` directly via `vi.mock('axios')` (stores import axios, not apiClient)
  - Composables: Mock the API module (e.g. `vi.mock('../lib/api/Account')`) + test QueryClient with seeded cache
- **No new dependencies:** Use existing `@vue/test-utils`, `@tanstack/vue-query`, `pinia`, `vitest`. Skip `@pinia/testing` and MSW.
- **Coverage:** Use `@vitest/coverage-v8` (already installed). No enforced threshold initially.

## Test Infrastructure

### Shared helper: `webui/src/test/helpers.ts`

```ts
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

Key design decisions for the helper:
- `renderComposable` creates a real Vue app context so `inject()` works (required by useQuery/useMutation)
- Returns `unmount` function -- callers MUST call it in `afterEach` to prevent leaked reactive state
- Sets active Pinia so stores work inside composables
- `gcTime: Infinity` prevents garbage collection during test execution
- `retry: false` makes query failures immediate instead of retrying 3 times

## Test Patterns

### Tier 1 -- Pure utility functions (no mocking)

```ts
// utils/currency.test.ts
import { describe, it, expect } from 'vitest'
import { formatCurrency, formatAmount } from './currency'

describe('formatCurrency', () => {
  it('formats with default locale', () => {
    expect(formatCurrency(1234.5)).toBe('1,234.50')
  })
  it('handles zero', () => {
    expect(formatCurrency(0)).toBe('0.00')
  })
  it('handles negative values', () => {
    expect(formatCurrency(-500.1)).toBe('-500.10')
  })
})
```

### Tier 2 -- API wrappers (mock apiClient)

```ts
// lib/api/Account.test.ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { apiClient } from './client'
import { getProviders, createAccount } from './Account'

vi.mock('./client', () => ({
  apiClient: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }
}))

beforeEach(() => vi.clearAllMocks())

describe('getProviders', () => {
  it('calls correct endpoint and returns data', async () => {
    const mockData = [{ id: 1, name: 'Bank Alpha', accounts: [] }]
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData })
    const result = await getProviders()
    expect(apiClient.get).toHaveBeenCalledWith('/fin/provider')
    expect(result).toEqual(mockData)
  })
})
```

For complex param construction (the real value of API tests):

```ts
// lib/api/Entry.test.ts
describe('getEntries', () => {
  it('builds query string with date range and pagination', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { entries: [], total: 0 } })
    await getEntries({
      startDate: '2026-01-01',
      endDate: '2026-01-31',
      page: 2,
      limit: 25,
      accountIds: [1, 3],
    })
    expect(apiClient.get).toHaveBeenCalledWith(
      expect.stringContaining('startDate=2026-01-01')
    )
    expect(apiClient.get).toHaveBeenCalledWith(
      expect.stringContaining('page=2')
    )
  })
})
```

### Tier 3 -- Stores (mock axios, plain createPinia)

```ts
// store/settingsStore.test.ts
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
  it('sets isLoaded after successful fetch', async () => {
    vi.mocked(axios.get).mockResolvedValue({
      data: { dateFormat: 'YYYY-MM-DD', mainCurrency: 'CHF', currencies: ['CHF', 'EUR'] }
    })
    const store = useSettingsStore()
    await store.fetchSettings()
    expect(store.isLoaded).toBe(true)
    expect(store.mainCurrency).toBe('CHF')
  })

  it('hasMultipleCurrencies is true when 2+ currencies', async () => {
    vi.mocked(axios.get).mockResolvedValue({
      data: { currencies: ['CHF', 'EUR'] }
    })
    const store = useSettingsStore()
    await store.fetchSettings()
    expect(store.hasMultipleCurrencies).toBe(true)
  })
})
```

### Tier 4 -- Composables (mock API module + renderComposable)

```ts
// composables/useAccounts.test.ts
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

  it('normalizes providers with missing accounts array', async () => {
    vi.mocked(getProviders).mockResolvedValue([
      { id: 1, name: 'Bank Alpha', accounts: null },
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
})
```

## Test Scope

### Tier 1 -- Pure functions (~80 test cases, 17 files)

| File | Functions |
|------|-----------|
| `utils/currency.ts` | `formatCurrency`, `formatAmount` |
| `utils/format.ts` | `formatPct`, `getChangeSeverity` |
| `utils/date.ts` | `toLocalDateString` |
| `utils/dateRange.ts` | `lastDaysRange`, `rangeToStartEnd` |
| `utils/apiError.ts` | `getApiErrorMessage` |
| `utils/convertToTree.ts` | `buildTree`, `buildTreeForTable` |
| `utils/categoryUtils.ts` | `findNodeById`, `buildCategoryPath` |
| `utils/accountUtils.ts` | `findAccountById` |
| `utils/entryDisplay.ts` | `getEntryTypeIcon` |
| `utils/entryValidation.ts` | `accountValidation` (zod schema) |
| `types/account.ts` | `getAccountTypeLabel`, `getAccountTypeIcon`, `getAllowedOperations`, `isOperationAllowed` |
| `composables/useDateFormat.js` | `toPrimeVueDateFormat`, `formatDisplayDate`, `formatTime`, `parseDateString` |
| `composables/useEntryDialogForm.ts` | `extractAccountId`, `getFormattedAccountId`, `getDateOnly`, `toDateString`, `getSubmitValues` |
| `lib/api/CurrencyRates.ts` | `parsePair` |
| `lib/api/Entry.ts` | `formatDate` |
| `lib/api/helpers.ts` | `with404Null` |

### Tier 2 -- API wrappers (~60 test cases, 11 files)

| File | Key test scenarios |
|------|-------------------|
| `lib/api/Account.ts` | CRUD operations, response shape |
| `lib/api/Entry.ts` | Pagination params, date filtering, stock transaction types |
| `lib/api/Category.ts` | Income vs expense dual endpoints |
| `lib/api/report.ts` | Query string construction, `getAccountBalance` nested extraction |
| `lib/api/CurrencyRates.ts` | URL encoding, optional date params, 404-to-null |
| `lib/api/MarketData.ts` | URL encoding, 404-to-null |
| `lib/api/CsvImport.ts` | FormData construction, conditional fields |
| `lib/api/Attachment.ts` | FormData upload, URL construction |
| `lib/api/Portfolio.ts` | Optional param handling |
| `lib/api/Tasks.ts` | URL encoding, response types |
| `lib/api/Backup.ts` | FormData upload, blob download |

### Tier 3 -- Stores (~30 test cases, 3 files)

| File | Key test scenarios |
|------|-------------------|
| `store/settingsStore.js` | Fetch success/error, state reset, `hasMultipleCurrencies` |
| `store/uiStore.js` | Drawer toggling, resize listener lifecycle |
| `lib/user/userstore.js` | Login/logout flows, error mapping, callback registration |

### Tier 4 -- Composables (~80 test cases, 12 files)

| File | Key test scenarios |
|------|-------------------|
| `composables/useAccounts.ts` | `normalizeProviders`, mutation-to-invalidation chain |
| `composables/useEntries.ts` | Dynamic query key computation, `enabled` logic, pagination |
| `composables/useCategories.ts` | Dual query sets, mutation invalidation |
| `composables/useCategoryTree.ts` | Tree transformation from flat data |
| `composables/useInvestmentReport.ts` | Gain/loss calculations, currency conversion, tree building |
| `composables/useAccountTypesData.ts` | Account grouping, balance aggregation |
| `composables/useMarketData.ts` | Parallel aggregation, change calculation |
| `composables/useCurrencyRates.ts` | FX aggregation, rate change calculation |
| `composables/useTasks.ts` | `deriveTasksWithLastExecution`, status mapping |
| `composables/useTaskExecutions.ts` | Filtering, sorting |
| `composables/useHoldings.ts` | Position mapping, price lookup, account filtering |
| `composables/queryUtils.ts` | Invalidation helper |

## Implementation order

1. Test helper (`src/test/helpers.ts`)
2. Tier 1: Pure utils and type functions
3. Tier 2: API wrappers
4. Tier 3: Stores
5. Tier 4: Composables (simplest first, complex aggregation last)

Each tier is independently useful -- if we stop after Tier 1 we still have value.
