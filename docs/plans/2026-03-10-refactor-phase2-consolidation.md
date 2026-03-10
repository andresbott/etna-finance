# Phase 2: Consolidation — Clean Up Before Extraction

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Eliminate duplicated patterns in dialog components: consolidate account lookups, extract entry mutations into a standalone composable, and fix overly broad prop watchers. These clean-ups reduce the code that Phase 3 will extract.

**Architecture:** Three focused refactors targeting dialog components. Item 2 replaces inline account lookups with the existing `findAccountById` utility. Item 5 splits mutations out of `useEntries` so dialogs don't create unused query subscriptions. Item 6 narrows `watch(props, ...)` to only watch `visible`.

**Tech Stack:** TypeScript, Vue 3, TanStack Query, Vitest

**Prerequisite:** Phase 1 should be completed first (Entry types needed for proper typing).

---

## Task 1: Replace inline account lookups with `findAccountById` (Item 2)

**Files:**
- Modify: `webui/src/views/entries/dialogs/IncomeExpenseDialog.vue` (lines ~109-132)
- Modify: `webui/src/views/entries/dialogs/TransferDialog.vue` (lines ~103-156)
- Modify: `webui/src/views/entries/AccountEntriesView.vue` (lines ~82-96)

**Context:** `accountUtils.ts` already exports `findAccountById(providers, id)` and `useAccountUtils().getAccount(id)`. Three dialog/view files reimplement the same nested-loop pattern: `for provider of accounts.value → provider.accounts.find(acc => acc.id === accountId)`.

### Step 1: Fix IncomeExpenseDialog.vue

In `webui/src/views/entries/dialogs/IncomeExpenseDialog.vue`, replace the `updateSelectedAccount` function's inline loop with `findAccountById`.

Add import:
```typescript
import { findAccountById } from '@/utils/accountUtils'
```

Replace the inline search (the `for (const provider of accounts.value)` loop inside `updateSelectedAccount`) with:
```typescript
const updateSelectedAccount = (accountObject: unknown) => {
    if (!accountObject || !accounts.value) {
        selectedAccount.value = null
        return
    }
    const accountId = extractAccountId(accountObject)
    if (isNaN(accountId) || accountId === null) {
        selectedAccount.value = null
        return
    }
    selectedAccount.value = findAccountById(accounts.value, accountId)
}
```

### Step 2: Fix TransferDialog.vue

In `webui/src/views/entries/dialogs/TransferDialog.vue`, there are two similar functions: `updateSelectedTargetAccount` and `updateSelectedOriginAccount`. Both contain the same nested loop. Replace both with `findAccountById`.

Add import:
```typescript
import { findAccountById } from '@/utils/accountUtils'
```

Replace the inline loops in both functions with `findAccountById(accounts.value, accountId)`.

### Step 3: Fix AccountEntriesView.vue

In `webui/src/views/entries/AccountEntriesView.vue` (lines 81-96), the `currentAccount` computed property reimplements the nested-loop search. Replace with:

Add import:
```typescript
import { findAccountById } from '@/utils/accountUtils'
```

```typescript
const currentAccount = computed(() => {
    if (!accountId.value || !accounts?.value) return null
    return findAccountById(accounts.value, accountId.value)
})
```

### Step 4: Verify build

Run: `cd webui && npx vue-tsc --noEmit 2>&1 | head -20`
Expected: No errors.

### Step 5: Run existing tests

Run: `cd webui && npx vitest run --reporter=verbose 2>&1 | tail -20`
Expected: All tests pass.

### Step 6: Commit

```bash
git add webui/src/views/entries/dialogs/IncomeExpenseDialog.vue webui/src/views/entries/dialogs/TransferDialog.vue webui/src/views/entries/AccountEntriesView.vue
git commit -m "refactor: replace inline account lookups with findAccountById utility"
```

---

## Task 2: Extract `useEntryMutations()` composable (Item 5)

**Files:**
- Create: `webui/src/composables/useEntryMutations.ts`
- Modify: `webui/src/composables/useEntries.ts`
- Modify: `webui/src/views/entries/dialogs/IncomeExpenseDialog.vue`
- Modify: `webui/src/views/entries/dialogs/TransferDialog.vue`
- Modify: `webui/src/views/entries/dialogs/BalanceStatusDialog.vue`

**Context:** Dialog components call `useEntries({})` with no date parameters just to access `createEntry` and `updateEntry`. This creates a disabled `useQuery` subscription that's never used. The mutations should be available independently.

### Step 1: Create `useEntryMutations.ts`

Create `webui/src/composables/useEntryMutations.ts`:

```typescript
import { useMutation, useQueryClient } from '@tanstack/vue-query'
import { createEntry, updateEntry, deleteEntry } from '@/lib/api/Entry'
import type { CreateEntryDTO, UpdateEntryDTO } from '@/types/entry'

export function useEntryMutations() {
    const queryClient = useQueryClient()

    const createEntryMutation = useMutation({
        mutationFn: (payload: CreateEntryDTO) => createEntry(payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['entries'] })
        }
    })

    const updateEntryMutation = useMutation({
        mutationFn: (payload: UpdateEntryDTO) => updateEntry(payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['entries'] })
        }
    })

    const deleteEntryMutation = useMutation({
        mutationFn: (id: string) => deleteEntry(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['entries'] })
        }
    })

    return {
        createEntry: createEntryMutation.mutateAsync,
        updateEntry: updateEntryMutation.mutateAsync,
        deleteEntry: deleteEntryMutation.mutateAsync,
        isCreating: createEntryMutation.isPending,
        isUpdating: updateEntryMutation.isPending,
        isDeleting: deleteEntryMutation.isPending
    }
}
```

### Step 2: Refactor `useEntries.ts` to use `useEntryMutations`

In `webui/src/composables/useEntries.ts`, replace the inline mutation definitions (lines 86-110) with a call to `useEntryMutations()`:

```typescript
import { useEntryMutations } from './useEntryMutations'

export function useEntries(options: UseEntriesOptions = {}) {
    // ... existing query setup ...

    const { createEntry: createEntryFn, updateEntry: updateEntryFn, deleteEntry: deleteEntryFn,
            isCreating, isUpdating, isDeleting } = useEntryMutations()

    // ... existing computed values ...

    return {
        entries, totalRecords, currentPage, pageSize, priorBalance,
        isLoading: entriesQuery.isLoading,
        isFetching: entriesQuery.isFetching,
        isError: entriesQuery.isError,
        error: entriesQuery.error,
        refetch: entriesQuery.refetch,
        createEntry: createEntryFn,
        updateEntry: updateEntryFn,
        deleteEntry: deleteEntryFn,
        isCreating, isUpdating, isDeleting
    }
}
```

### Step 3: Update dialog components

In each dialog that imports `useEntries`, replace with `useEntryMutations`:

**IncomeExpenseDialog.vue:**
```typescript
// Before:
import { useEntries } from '@/composables/useEntries'
const { createEntry, updateEntry } = useEntries({})

// After:
import { useEntryMutations } from '@/composables/useEntryMutations'
const { createEntry, updateEntry } = useEntryMutations()
```

**TransferDialog.vue:** Same change.

**BalanceStatusDialog.vue:** Same change.

Check if any other dialog imports `useEntries` just for mutations — search for `useEntries({})` or `useEntries({ })`.

### Step 4: Verify build

Run: `cd webui && npx vue-tsc --noEmit 2>&1 | head -20`
Expected: No errors.

### Step 5: Run tests

Run: `cd webui && npx vitest run --reporter=verbose 2>&1 | tail -20`
Expected: All pass.

### Step 6: Commit

```bash
git add webui/src/composables/useEntryMutations.ts webui/src/composables/useEntries.ts webui/src/views/entries/dialogs/IncomeExpenseDialog.vue webui/src/views/entries/dialogs/TransferDialog.vue webui/src/views/entries/dialogs/BalanceStatusDialog.vue
git commit -m "refactor: extract useEntryMutations composable to avoid unused query subscriptions in dialogs"
```

---

## Task 3: Fix `watch(props, ...)` to only watch `visible` (Item 6)

**Files:**
- Modify: `webui/src/views/entries/dialogs/IncomeExpenseDialog.vue` (line ~85)
- Modify: `webui/src/views/entries/dialogs/TransferDialog.vue` (line ~83)
- Modify: `webui/src/views/entries/dialogs/BalanceStatusDialog.vue` (line ~58)

**Context:** All three dialogs use `watch(props, (newProps) => { ... formKey.value++ })` which watches *every* prop. When `visible` toggles, the form is reset and `formKey` is incremented, forcing PrimeVue Form to fully re-render. The fix: only watch `() => props.visible` and only reset when the dialog opens.

### Step 1: Fix IncomeExpenseDialog.vue

Replace the current watch (line ~85):

```typescript
// Before:
watch(props, (newProps) => {
    formValues.value = {
        description: newProps.description,
        amount: newProps.amount,
        AccountId: getFormattedAccountId(newProps.accountId),
        stockAmount: newProps.stockAmount,
        date: getDateOnly(newProps.date)
    }
    categoryId.value = newProps.categoryId
    existingAttachmentId.value = newProps.attachmentId || null
    selectedFile.value = null
    attachmentPendingDelete.value = false
    formKey.value++
})

// After:
watch(() => props.visible, (visible) => {
    if (!visible) return
    formValues.value = {
        description: props.description,
        amount: props.amount,
        AccountId: getFormattedAccountId(props.accountId),
        stockAmount: props.stockAmount,
        date: getDateOnly(props.date)
    }
    categoryId.value = props.categoryId
    existingAttachmentId.value = props.attachmentId || null
    selectedFile.value = null
    attachmentPendingDelete.value = false
    formKey.value++
})
```

### Step 2: Fix TransferDialog.vue

Same pattern. Replace `watch(props, ...)` (line ~83):

```typescript
watch(() => props.visible, (visible) => {
    if (!visible) return
    formValues.value = {
        description: props.description,
        date: getDateOnly(props.date),
        targetAmount: props.targetAmount,
        originAmount: props.originAmount,
        targetAccountId: getFormattedAccountId(props.targetAccountId),
        originAccountId: getFormattedAccountId(props.originAccountId)
    }
    existingAttachmentId.value = props.attachmentId || null
    selectedFile.value = null
    attachmentPendingDelete.value = false
    formKey.value++
})
```

### Step 3: Fix BalanceStatusDialog.vue

Same pattern. Replace `watch(props, ...)` (line ~58):

```typescript
watch(() => props.visible, (visible) => {
    if (!visible) return
    formValues.value = {
        description: props.description,
        amount: props.amount,
        AccountId: getFormattedAccountId(props.accountId),
        date: getDateOnly(props.date)
    }
    existingAttachmentId.value = props.attachmentId || null
    selectedFile.value = null
    attachmentPendingDelete.value = false
    formKey.value++
})
```

### Step 4: Verify build

Run: `cd webui && npx vue-tsc --noEmit 2>&1 | head -20`
Expected: No errors.

### Step 5: Manual test

Open any entry dialog (income/expense, transfer, balance status), verify:
1. Form populates correctly when editing an entry
2. Form resets when closing and reopening with a different entry
3. Duplicate mode works (new date, autofocus amount)

### Step 6: Commit

```bash
git add webui/src/views/entries/dialogs/IncomeExpenseDialog.vue webui/src/views/entries/dialogs/TransferDialog.vue webui/src/views/entries/dialogs/BalanceStatusDialog.vue
git commit -m "fix: only reset dialog forms when visible changes, not on every prop change"
```
