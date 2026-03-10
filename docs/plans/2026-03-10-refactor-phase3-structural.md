# Phase 3: Structural Refactors — Deduplication & Naming

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Deduplicate the entries views by extracting shared dialog logic, convert balance fetching from mutation to query, and standardize naming conventions across the codebase.

**Architecture:** Item 1 is the largest refactor — extracting a `useEntryDialogs` composable and `EntryDialogs.vue` component from two views with ~110 lines of duplicated code. Item 8 converts a mutation-based data fetch to a proper query. Item 3 is a codebase-wide rename sweep done last to avoid merge conflicts.

**Tech Stack:** TypeScript, Vue 3, TanStack Query, PrimeVue

**Prerequisite:** Phase 2 must be completed first — clean mutations (Item 5), consolidated account lookups (Item 2), and fixed watchers (Item 6) mean the extracted code is already clean.

---

## Task 1: Create `useEntryDialogs` composable (Item 1, part 1)

**Files:**
- Create: `webui/src/composables/useEntryDialogs.ts`

**Context:** Both `EntriesView.vue` and `AccountEntriesView.vue` duplicate: dialog visibility state (9 refs), `selectedEntry`/`isEditMode`/`isDuplicateMode` state, `openEditEntryDialog`, `openDuplicateEntryDialog`, `openDeleteDialog`, `handleDeleteEntry`, and delete dialog state.

### Step 1: Create the composable

Create `webui/src/composables/useEntryDialogs.ts`:

```typescript
import { ref } from 'vue'
import { getEntry } from '@/lib/api/Entry'

export function useEntryDialogs(deleteEntryFn: (id: string) => Promise<void>) {
    const selectedEntry = ref<Record<string, unknown> | null>(null)
    const isEditMode = ref(false)
    const isDuplicateMode = ref(false)

    const deleteDialogVisible = ref(false)
    const entryToDelete = ref<Record<string, unknown> | null>(null)

    const dialogs = {
        incomeExpense: ref(false),
        expense: ref(false),
        income: ref(false),
        transfer: ref(false),
        buyStock: ref(false),
        sellStock: ref(false),
        grantStock: ref(false),
        transferInstrument: ref(false),
        balanceStatus: ref(false)
    }

    const ENTRY_TYPE_TO_DIALOG: Record<string, keyof typeof dialogs> = {
        income: 'incomeExpense',
        expense: 'incomeExpense',
        transfer: 'transfer',
        stockbuy: 'buyStock',
        stocksell: 'sellStock',
        stockgrant: 'grantStock',
        stocktransfer: 'transferInstrument',
        balancestatus: 'balanceStatus'
    }

    function openDialogForType(type: string) {
        const dialogKey = ENTRY_TYPE_TO_DIALOG[type]
        if (dialogKey) {
            dialogs[dialogKey].value = true
        }
    }

    const openEditEntryDialog = async (entry: Record<string, unknown>) => {
        isEditMode.value = true
        isDuplicateMode.value = false

        if (entry.type === 'stocksell') {
            try {
                const full = await getEntry(entry.id as string)
                selectedEntry.value = full
            } catch (e) {
                console.error('Failed to load sell entry for edit', e)
                selectedEntry.value = entry
            }
        } else {
            selectedEntry.value = entry
        }

        openDialogForType(entry.type as string)
    }

    const openDuplicateEntryDialog = (entry: Record<string, unknown>) => {
        isEditMode.value = false
        isDuplicateMode.value = true
        selectedEntry.value = entry
        openDialogForType(entry.type as string)
    }

    const openDeleteDialog = (entry: Record<string, unknown>) => {
        entryToDelete.value = entry
        deleteDialogVisible.value = true
    }

    const handleDeleteEntry = async () => {
        try {
            await deleteEntryFn(entryToDelete.value?.id as string)
            deleteDialogVisible.value = false
        } catch (error) {
            console.error('Failed to delete entry:', error)
        }
    }

    return {
        selectedEntry,
        isEditMode,
        isDuplicateMode,
        dialogs,
        deleteDialogVisible,
        entryToDelete,
        openEditEntryDialog,
        openDuplicateEntryDialog,
        openDeleteDialog,
        handleDeleteEntry
    }
}
```

### Step 2: Verify build

Run: `cd webui && npx vue-tsc --noEmit 2>&1 | head -20`
Expected: No errors (composable is not imported yet).

### Step 3: Commit

```bash
git add webui/src/composables/useEntryDialogs.ts
git commit -m "refactor: create useEntryDialogs composable for shared dialog logic"
```

---

## Task 2: Create `EntryDialogs.vue` component (Item 1, part 2)

**Files:**
- Create: `webui/src/views/entries/EntryDialogs.vue`

**Context:** Both views render the same 8 dialog instances (IncomeExpenseDialog, TransferDialog, 2x BuySellInstrumentDialog, GrantDialog, TransferInstrumentDialog, BalanceStatusDialog, DeleteDialog) with identical prop bindings derived from `selectedEntry`, `isEditMode`, `isDuplicateMode`, and `dialogs`.

### Step 1: Create the component

Create `webui/src/views/entries/EntryDialogs.vue`. This component receives the composable's return values as props and renders all dialog instances:

```vue
<script setup lang="ts">
import { type Ref } from 'vue'
import IncomeExpenseDialog from '@/views/entries/dialogs/IncomeExpenseDialog.vue'
import TransferDialog from './dialogs/TransferDialog.vue'
import BuySellInstrumentDialog from './dialogs/BuySellInstrumentDialog.vue'
import GrantDialog from './dialogs/GrantDialog.vue'
import TransferInstrumentDialog from './dialogs/TransferInstrumentDialog.vue'
import BalanceStatusDialog from '@/views/entries/dialogs/BalanceStatusDialog.vue'
import DeleteDialog from '@/components/common/confirmDialog.vue'

defineProps<{
    selectedEntry: Record<string, unknown> | null
    isEditMode: boolean
    isDuplicateMode: boolean
    dialogs: Record<string, Ref<boolean>>
    deleteDialogVisible: boolean
    entryToDelete: Record<string, unknown> | null
}>()

const emit = defineEmits<{
    (e: 'update:deleteDialogVisible', value: boolean): void
    (e: 'confirmDelete'): void
}>()
</script>

<template>
    <IncomeExpenseDialog
        v-model:visible="dialogs.incomeExpense.value"
        :is-edit="isEditMode"
        :entry-type="selectedEntry?.type"
        :description="selectedEntry?.description"
        :amount="selectedEntry?.Amount"
        :account-id="selectedEntry?.accountId"
        :stock-amount="selectedEntry?.targetStockAmount"
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date as string) : new Date())"
        :entry-id="selectedEntry?.id"
        :category-id="selectedEntry?.categoryId"
        :autofocus-amount="isDuplicateMode"
        :attachment-id="selectedEntry?.attachmentId"
    />

    <TransferDialog
        v-model:visible="dialogs.transfer.value"
        :is-edit="isEditMode"
        :entry-id="selectedEntry?.id"
        :description="selectedEntry?.description"
        :target-amount="selectedEntry?.targetAmount"
        :origin-amount="selectedEntry?.originAmount"
        :target-stock-amount="selectedEntry?.targetStockAmount"
        :origin-stock-amount="selectedEntry?.originStockAmount"
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date as string) : new Date())"
        :target-account-id="selectedEntry?.targetAccountId"
        :origin-account-id="selectedEntry?.originAccountId"
        :autofocus-amount="isDuplicateMode"
        :attachment-id="selectedEntry?.attachmentId"
    />

    <BuySellInstrumentDialog
        v-model:visible="dialogs.buyStock.value"
        :is-edit="isEditMode"
        :entry-id="selectedEntry?.id"
        operation-type="buy"
        :instrument-id="selectedEntry?.instrumentId"
        :description="selectedEntry?.description"
        :quantity="selectedEntry?.quantity"
        :price-per-share="(selectedEntry?.StockAmount as number) && (selectedEntry?.quantity as number) ? (selectedEntry?.StockAmount as number) / (selectedEntry?.quantity as number) : undefined"
        :cash-amount="selectedEntry?.totalAmount"
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date as string) : new Date())"
        :investment-account-id="selectedEntry?.investmentAccountId"
        :cash-account-id="selectedEntry?.cashAccountId"
        @update:visible="dialogs.buyStock.value = $event"
    />
    <BuySellInstrumentDialog
        v-model:visible="dialogs.sellStock.value"
        :is-edit="isEditMode"
        :entry-id="selectedEntry?.id"
        operation-type="sell"
        :instrument-id="selectedEntry?.instrumentId"
        :description="selectedEntry?.description"
        :quantity="selectedEntry?.quantity"
        :price-per-share="((selectedEntry?.quantity as number) && ((selectedEntry?.costBasis as number | undefined) != null || (selectedEntry?.StockAmount as number | undefined) != null)) ? (((selectedEntry?.costBasis as number | undefined) ?? (selectedEntry?.StockAmount as number)) / (selectedEntry?.quantity as number)) : undefined"
        :cash-amount="((selectedEntry?.totalAmount as number) ?? 0) - ((selectedEntry?.fees as number) ?? 0)"
        :fees="(selectedEntry?.fees as number) ?? 0"
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date as string) : new Date())"
        :investment-account-id="selectedEntry?.investmentAccountId"
        :cash-account-id="selectedEntry?.cashAccountId"
        @update:visible="dialogs.sellStock.value = $event"
    />

    <GrantDialog
        v-model:visible="dialogs.grantStock.value"
        :is-edit="isEditMode"
        :entry-id="selectedEntry?.id"
        :instrument-id="selectedEntry?.instrumentId"
        :description="selectedEntry?.description"
        :quantity="selectedEntry?.quantity"
        :fair-market-value="(selectedEntry?.fairMarketValue as number) ?? 0"
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date as string) : new Date())"
        :account-id="selectedEntry?.accountId"
        @update:visible="dialogs.grantStock.value = $event"
    />
    <TransferInstrumentDialog
        v-model:visible="dialogs.transferInstrument.value"
        :is-edit="isEditMode"
        :entry-id="selectedEntry?.id"
        :instrument-id="selectedEntry?.instrumentId"
        :description="selectedEntry?.description"
        :quantity="selectedEntry?.quantity"
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date as string) : new Date())"
        :origin-account-id="selectedEntry?.originAccountId"
        :target-account-id="selectedEntry?.targetAccountId"
        @update:visible="dialogs.transferInstrument.value = $event"
    />

    <BalanceStatusDialog
        v-model:visible="dialogs.balanceStatus.value"
        :is-edit="isEditMode"
        :entry-id="selectedEntry?.id"
        :description="selectedEntry?.description"
        :amount="selectedEntry?.Amount"
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date as string) : new Date())"
        :account-id="selectedEntry?.accountId"
        :attachment-id="selectedEntry?.attachmentId"
    />

    <DeleteDialog
        :visible="deleteDialogVisible"
        :name="(entryToDelete?.description as string)"
        message="Are you sure you want to delete this entry?"
        @update:visible="emit('update:deleteDialogVisible', $event)"
        @confirm="emit('confirmDelete')"
    />
</template>
```

### Step 2: Verify build

Run: `cd webui && npx vue-tsc --noEmit 2>&1 | head -20`
Expected: No errors.

### Step 3: Commit

```bash
git add webui/src/views/entries/EntryDialogs.vue
git commit -m "refactor: create EntryDialogs component to render all entry dialog instances"
```

---

## Task 3: Refactor EntriesView.vue to use shared composable + component (Item 1, part 3)

**Files:**
- Modify: `webui/src/views/entries/EntriesView.vue`

**Context:** Replace dialog state (lines 84-175) and dialog template (lines 213-319) with `useEntryDialogs` composable and `EntryDialogs` component.

### Step 1: Update the script section

Remove these imports (no longer needed directly):
```typescript
// Remove:
import TransferDialog from './dialogs/TransferDialog.vue'
import BuySellInstrumentDialog from './dialogs/BuySellInstrumentDialog.vue'
import GrantDialog from './dialogs/GrantDialog.vue'
import TransferInstrumentDialog from './dialogs/TransferInstrumentDialog.vue'
import DeleteDialog from '@/components/common/confirmDialog.vue'
import IncomeExpenseDialog from '@/views/entries/dialogs/IncomeExpenseDialog.vue'
import BalanceStatusDialog from '@/views/entries/dialogs/BalanceStatusDialog.vue'
import { getEntry } from '@/lib/api/Entry'
```

Add these imports:
```typescript
import { useEntryDialogs } from '@/composables/useEntryDialogs'
import EntryDialogs from './EntryDialogs.vue'
```

Replace lines 84-175 (all dialog state and handler functions) with:
```typescript
const {
    selectedEntry, isEditMode, isDuplicateMode, dialogs,
    deleteDialogVisible, entryToDelete,
    openEditEntryDialog, openDuplicateEntryDialog, openDeleteDialog, handleDeleteEntry
} = useEntryDialogs(deleteEntry)
```

### Step 2: Update the template section

Replace all dialog component instances (lines 213-319) with:
```vue
<EntryDialogs
    :selected-entry="selectedEntry"
    :is-edit-mode="isEditMode"
    :is-duplicate-mode="isDuplicateMode"
    :dialogs="dialogs"
    :delete-dialog-visible="deleteDialogVisible"
    :entry-to-delete="entryToDelete"
    @update:delete-dialog-visible="deleteDialogVisible = $event"
    @confirm-delete="handleDeleteEntry"
/>
```

### Step 3: Verify build

Run: `cd webui && npx vue-tsc --noEmit 2>&1 | head -20`
Expected: No errors.

### Step 4: Commit

```bash
git add webui/src/views/entries/EntriesView.vue
git commit -m "refactor: EntriesView uses useEntryDialogs composable and EntryDialogs component"
```

---

## Task 4: Refactor AccountEntriesView.vue to use shared composable + component (Item 1, part 4)

**Files:**
- Modify: `webui/src/views/entries/AccountEntriesView.vue`

**Context:** Same as Task 3 but for the account-specific view. Lines 154-245 contain the duplicated dialog state, and lines 293-399 contain the dialog template.

### Step 1: Update the script section

Same import changes as Task 3. Remove individual dialog imports and `getEntry`, add `useEntryDialogs` and `EntryDialogs`.

Replace lines 154-245 with:
```typescript
const {
    selectedEntry, isEditMode, isDuplicateMode, dialogs,
    deleteDialogVisible, entryToDelete,
    openEditEntryDialog, openDuplicateEntryDialog, openDeleteDialog, handleDeleteEntry
} = useEntryDialogs(deleteEntry)
```

### Step 2: Update the template section

Replace all dialog component instances (lines 293-399) with:
```vue
<EntryDialogs
    :selected-entry="selectedEntry"
    :is-edit-mode="isEditMode"
    :is-duplicate-mode="isDuplicateMode"
    :dialogs="dialogs"
    :delete-dialog-visible="deleteDialogVisible"
    :entry-to-delete="entryToDelete"
    @update:delete-dialog-visible="deleteDialogVisible = $event"
    @confirm-delete="handleDeleteEntry"
/>
```

### Step 3: Verify build

Run: `cd webui && npx vue-tsc --noEmit 2>&1 | head -20`
Expected: No errors.

### Step 4: Manual test

Test both views:
1. `/entries` — open edit, duplicate, delete for income, expense, transfer, stock entries
2. `/entries/:id` — same operations from the account-specific view
3. Verify dialogs populate correctly and forms submit

### Step 5: Commit

```bash
git add webui/src/views/entries/AccountEntriesView.vue
git commit -m "refactor: AccountEntriesView uses useEntryDialogs composable and EntryDialogs component"
```

---

## Task 5: Refactor balance data fetching from mutation to query (Item 8)

**Files:**
- Modify: `webui/src/composables/useAccountTypesData.ts` (lines 100-117)

**Context:** `useAccountTypesData` calls `balanceReportMutation.mutate(...)` inside a `watch` with `{ immediate: true }`. This should be a `useQuery` — it's a read operation that would benefit from caching, deduplication, and standard loading states.

### Step 1: Replace the mutation+watch with useQuery

In `webui/src/composables/useAccountTypesData.ts`, replace the balance report mutation pattern (lines 14-16 and 100-117):

```typescript
// Remove:
const { balanceReport: balanceReportMutation } = useBalance()
const { mutate, data: balanceReport } = balanceReportMutation

// Remove the watch at lines 100-117

// Add:
import { getBalanceReport } from '@/lib/api/report'

const balanceReportQuery = useQuery({
    queryKey: computed(() => {
        const ids = allAccounts.value.map((a) => a.id).filter(Boolean)
        return ['balanceReport', ...ids]
    }),
    queryFn: () => {
        const accountIds = allAccounts.value.map((a) => a.id).filter(Boolean)
        const oneYearAgo = new Date()
        oneYearAgo.setFullYear(oneYearAgo.getFullYear() - 1)
        return getBalanceReport({
            accountIds,
            steps: 30,
            startDate: oneYearAgo.toISOString().split('T')[0]
        })
    },
    enabled: computed(() => allAccounts.value.length > 0)
})

const balanceReport = computed(() => balanceReportQuery.data.value)
```

### Step 2: Check if `useBalance` import can be removed

If `useBalance` is no longer used anywhere in this file, remove the import:
```typescript
// Remove if unused:
import { useBalance } from '@/composables/useGetBalanceReport'
```

### Step 3: Verify the `getBalanceReport` API function signature

Check `webui/src/lib/api/report.ts` to confirm the function signature matches the parameters we're passing. Adjust if needed.

### Step 4: Verify build

Run: `cd webui && npx vue-tsc --noEmit 2>&1 | head -20`
Expected: No errors.

### Step 5: Commit

```bash
git add webui/src/composables/useAccountTypesData.ts
git commit -m "refactor: use useQuery for balance report fetching instead of mutation+watch pattern"
```

---

## Task 6: Standardize API function naming to camelCase (Item 3, part 1)

**Files:**
- Modify: `webui/src/lib/api/Category.ts`
- Modify: `webui/src/lib/api/Backup.ts`
- Modify: all files that import from these two API modules

**Context:** Inventory of current naming:
- **camelCase (majority):** Account.ts, Entry.ts, Instrument.ts, InstrumentProvider.ts, CurrencyRates.ts, MarketData.ts, report.ts, CsvImport.ts, Attachment.ts, Portfolio.ts
- **PascalCase:** Category.ts (7 of 8 functions), Backup.ts (all 5 functions)
- **Mixed:** Category.ts has `deleteIncomeCategory` (camelCase) among PascalCase siblings

Standard: **camelCase for all API functions** (matches the majority).

### Step 1: Rename Category.ts exports

In `webui/src/lib/api/Category.ts`, rename:

| Before | After |
|--------|-------|
| `GetIncomeCategories` | `getIncomeCategories` |
| `CreateIncomeCategory` | `createIncomeCategory` |
| `UpdateIncomeCategory` | `updateIncomeCategory` |
| `GetExpenseCategories` | `getExpenseCategories` |
| `CreateExpenseCategory` | `createExpenseCategory` |
| `UpdateExpenseCategory` | `updateExpenseCategory` |
| `DeleteExpenseCategory` | `deleteExpenseCategory` |

(`deleteIncomeCategory` is already camelCase — no change needed.)

### Step 2: Update all imports of Category.ts

Search for imports:
```bash
cd webui && grep -rn "GetIncomeCategories\|CreateIncomeCategory\|UpdateIncomeCategory\|GetExpenseCategories\|CreateExpenseCategory\|UpdateExpenseCategory\|DeleteExpenseCategory" src/
```

Update each import and usage to use the new camelCase names.

### Step 3: Rename Backup.ts exports

In `webui/src/lib/api/Backup.ts`, rename:

| Before | After |
|--------|-------|
| `GetBackupFiles` | `getBackupFiles` |
| `DeleteBackupFile` | `deleteBackupFile` |
| `DownloadBackupFile` | `downloadBackupFile` |
| `CreateBackup` | `createBackup` |
| `RestoreBackup` | `restoreBackup` |
| `RestoreBackupFromExisting` | `restoreBackupFromExisting` |

### Step 4: Update all imports of Backup.ts

Search and update all imports and usages.

### Step 5: Verify build

Run: `cd webui && npx vue-tsc --noEmit 2>&1 | head -20`
Expected: No errors.

### Step 6: Commit

```bash
git add webui/src/lib/api/Category.ts webui/src/lib/api/Backup.ts
git add -u  # stage updated import files
git commit -m "refactor: standardize API function names to camelCase (Category.ts, Backup.ts)"
```

---

## Task 7: Rename camelCase component files to PascalCase (Item 3, part 2)

**Files:**
- Rename: `webui/src/components/common/categorySelect.vue` → `CategorySelect.vue`
- Rename: `webui/src/components/common/confirmDialog.vue` → `ConfirmDialog.vue`
- Rename: `webui/src/components/common/loadingScreen.vue` → `LoadingScreen.vue`
- Update: 11 import statements across the codebase

**Context:** Vue style guide recommends PascalCase for SFC file names. Only 3 files violate this.

### Step 1: Rename the files

```bash
cd webui/src/components/common
mv categorySelect.vue CategorySelect.vue
mv confirmDialog.vue ConfirmDialog.vue
mv loadingScreen.vue LoadingScreen.vue
```

### Step 2: Update all imports

Files that import these components (found via grep):

**confirmDialog.vue → ConfirmDialog.vue** (7 files):
- `src/views/marketdata/StockDetailView.vue`
- `src/views/instruments/InstrumentsView.vue`
- `src/views/accounts/accounts.vue`
- `src/views/backup/BackupRestoreView.vue`
- `src/views/marketdata/CurrencyDetailView.vue`
- `src/views/categories/CategoriesView.vue`
- `src/views/entries/EntryDialogs.vue` (if created in Task 2; otherwise `EntriesView.vue` and `AccountEntriesView.vue`)

Update import path from `confirmDialog.vue` to `ConfirmDialog.vue`.

**categorySelect.vue → CategorySelect.vue** (2 files):
- `src/views/categories/dialogs/CategoryDialog.vue`
- `src/views/entries/dialogs/IncomeExpenseDialog.vue`

**loadingScreen.vue → LoadingScreen.vue** (1 file):
- `src/lib/user/UserLogin.vue`

### Step 3: Verify build

Run: `cd webui && npx vue-tsc --noEmit 2>&1 | head -20`
Expected: No errors.

### Step 4: Verify no remaining camelCase component files

Run: `cd webui && find src/components -name '*.vue' | grep -E '/[a-z]'`
Expected: No matches (all component files are PascalCase).

### Step 5: Commit

```bash
git add -A webui/src/components/common/
git add -u  # stage updated imports
git commit -m "refactor: rename component files to PascalCase (Vue style guide convention)"
```
