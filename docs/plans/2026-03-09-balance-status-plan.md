# Balance Status Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a "Balance Status" movement type that records the real bank statement balance for cash accounts, computes and displays the discrepancy against the calculated balance, and does not affect the running balance.

**Architecture:** New `BalanceStatusTransaction` TxType (value 9) with `balanceStatusEntry` entry type. One `dbTransaction` + one `dbEntry` per record. Balance status entries are excluded from `balanceEntryTypes` so they don't affect calculated balances. The API computes the discrepancy on read using the existing `AccountBalanceSingle` function.

**Tech Stack:** Go (backend store + API handler), Vue 3 / PrimeVue (frontend dialog + table display), TypeScript

---

### Task 1: Add BalanceStatus type and struct to the accounting package

**Files:**
- Modify: `internal/accounting/transaction.go:16-28` (TxType enum)
- Modify: `internal/accounting/transaction.go:42-48` (Transaction interface area — add BalanceStatus struct)
- Modify: `internal/accounting/entry.go:9-24` (entryType enum)

**Step 1: Add `BalanceStatusTransaction` to the TxType enum**

In `internal/accounting/transaction.go`, after `LoanTransaction` (line 27), add:

```go
BalanceStatusTransaction
```

**Step 2: Add `balanceStatusEntry` to the entryType enum**

In `internal/accounting/entry.go`, after `stockTransferInEntry` (line 23), add:

```go
balanceStatusEntry // balance checkpoint — does NOT affect cash balance
```

**Step 3: Add the BalanceStatus struct**

In `internal/accounting/transaction.go`, after the `StockTransfer` struct, add:

```go
type BalanceStatus struct {
	Id          uint
	Description string
	Amount      float64 // the stated balance from the bank statement
	AccountID   uint
	Date        time.Time

	baseTx
}
```

**Step 4: Add the BalanceStatusUpdate struct**

Near the other update structs (e.g. `IncomeUpdate`, `ExpenseUpdate`), add:

```go
type BalanceStatusUpdate struct {
	Description *string
	Date        *time.Time
	Amount      *float64
	AccountID   *uint

	baseTxUpdate
}
```

**Step 5: Commit**

```bash
git add internal/accounting/transaction.go internal/accounting/entry.go
git commit -m "feat: add BalanceStatus type, struct, and entry type"
```

---

### Task 2: Implement CreateBalanceStatus in the store

**Files:**
- Modify: `internal/accounting/transaction.go` (CreateTransaction dispatcher + new CreateBalanceStatus method)

**Step 1: Write the test for CreateBalanceStatus**

In `internal/accounting/transaction_test.go`, add test cases to the existing `TestStore_CreateTransaction` table (after the stock transfer test cases):

```go
{
	name:   "create valid balance status",
	tenant: tenant1,
	input: BalanceStatus{
		Description: "March bank statement",
		Amount:      1250.00,
		AccountID:   1,
		Date:        time.Now(),
	},
},
{
	name:   "balance status on investment account should fail",
	tenant: tenant1,
	input: BalanceStatus{
		Description: "bad balance status",
		Amount:      1000.00,
		AccountID:   5, // investment account from sample data
		Date:        time.Now(),
	},
	wantErr: "incompatible account type",
},
{
	name:   "balance status with zero account should fail",
	tenant: tenant1,
	input: BalanceStatus{
		Description: "bad balance status",
		Amount:      1000.00,
		AccountID:   0,
		Date:        time.Now(),
	},
	wantErr: "account id is required",
},
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/accounting/ -run TestStore_CreateTransaction -v -count=1`
Expected: FAIL — BalanceStatus not handled in CreateTransaction

**Step 3: Add BalanceStatus case to CreateTransaction dispatcher**

In `internal/accounting/transaction.go`, in the `CreateTransaction` switch (around line 165), add before `default`:

```go
case BalanceStatus:
	return store.CreateBalanceStatus(ctx, item)
```

**Step 4: Implement CreateBalanceStatus**

Add the method after `CreateStockTransfer`:

```go
func (store *Store) CreateBalanceStatus(ctx context.Context, item BalanceStatus) (uint, error) {
	if item.AccountID == 0 {
		return 0, ErrValidation("account id is required")
	}

	acc, err := store.GetAccount(ctx, item.AccountID)
	if err != nil {
		return 0, fmt.Errorf("error creating balance status: %w", err)
	}
	allowedAccountTypes := []AccountType{
		CashAccountType, CheckinAccountType, SavingsAccountType,
	}
	if !slices.Contains(allowedAccountTypes, acc.Type) {
		return 0, NewValidationErr(fmt.Sprintf("incompatible account type %s for balance status transaction", acc.Type.String()))
	}

	tx := dbTransaction{
		Description: item.Description,
		Date:        item.Date,
		Type:        BalanceStatusTransaction,
		Entries: []dbEntry{
			{
				AccountID: item.AccountID,
				Amount:    item.Amount,
				EntryType: balanceStatusEntry,
			},
		},
	}

	if err := validateTransaction(tx); err != nil {
		return 0, err
	}

	if err := store.db.WithContext(ctx).Create(&tx).Error; err != nil {
		return 0, err
	}
	return tx.Id, nil
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/accounting/ -run TestStore_CreateTransaction -v -count=1`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/accounting/transaction.go internal/accounting/transaction_test.go
git commit -m "feat: implement CreateBalanceStatus store method"
```

---

### Task 3: Implement GetTransaction and publicTransactions for BalanceStatus

**Files:**
- Modify: `internal/accounting/transaction.go` (publicTransactions switch + balanceStatusFromDb)

**Step 1: Write the test**

Add a test case to an existing Get test, or write a simple round-trip test in `transaction_test.go`:

```go
func TestStore_BalanceStatus_RoundTrip(t *testing.T) {
	testdbs.RunTest(t, func(t *testing.T, db testdbs.DBGetter) {
		dbCon := db.ConnDbName("TestBalanceStatusRoundTrip")
		store, _ := newAccountingStoreWithMarketData(t, dbCon)
		accountSampleData(t, store)

		ctx := t.Context()
		date := getDate("2025-03-01")

		id, err := store.CreateBalanceStatus(ctx, BalanceStatus{
			Description: "March statement",
			Amount:      1250.50,
			AccountID:   1,
			Date:        date,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := store.GetTransaction(ctx, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := BalanceStatus{
			Id:          id,
			Description: "March statement",
			Amount:      1250.50,
			AccountID:   1,
			Date:        date,
		}

		if diff := cmp.Diff(want, got, cmpopts.IgnoreUnexported(BalanceStatus{})); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/accounting/ -run TestStore_BalanceStatus_RoundTrip -v -count=1`
Expected: FAIL — BalanceStatusTransaction not handled in publicTransactions

**Step 3: Add balanceStatusFromDb and update publicTransactions**

In `publicTransactions` switch (around line 950), add before `default`:

```go
case BalanceStatusTransaction:
	return balanceStatusFromDb(in)
```

Add the conversion function:

```go
func balanceStatusFromDb(in dbTransaction) (Transaction, error) {
	return BalanceStatus{
		Id:          in.Id,
		Description: in.Description,
		Date:        in.Date,
		Amount:      in.Entries[0].Amount,
		AccountID:   in.Entries[0].AccountID,
	}, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/accounting/ -run TestStore_BalanceStatus_RoundTrip -v -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/accounting/transaction.go internal/accounting/transaction_test.go
git commit -m "feat: implement GetTransaction support for BalanceStatus"
```

---

### Task 4: Implement UpdateBalanceStatus and add to UpdateTransaction dispatcher

**Files:**
- Modify: `internal/accounting/transaction.go`

**Step 1: Write the test**

```go
func TestStore_UpdateBalanceStatus(t *testing.T) {
	testdbs.RunTest(t, func(t *testing.T, db testdbs.DBGetter) {
		dbCon := db.ConnDbName("TestUpdateBalanceStatus")
		store, _ := newAccountingStoreWithMarketData(t, dbCon)
		accountSampleData(t, store)

		ctx := t.Context()
		id, err := store.CreateBalanceStatus(ctx, BalanceStatus{
			Description: "March statement",
			Amount:      1250.50,
			AccountID:   1,
			Date:        getDate("2025-03-01"),
		})
		if err != nil {
			t.Fatalf("create error: %v", err)
		}

		newDesc := "Updated March statement"
		newAmount := 1300.00
		err = store.UpdateTransaction(ctx, BalanceStatusUpdate{
			Description: &newDesc,
			Amount:      &newAmount,
		}, id)
		if err != nil {
			t.Fatalf("update error: %v", err)
		}

		got, err := store.GetTransaction(ctx, id)
		if err != nil {
			t.Fatalf("get error: %v", err)
		}

		bs := got.(BalanceStatus)
		if bs.Description != newDesc {
			t.Errorf("description = %q, want %q", bs.Description, newDesc)
		}
		if bs.Amount != newAmount {
			t.Errorf("amount = %v, want %v", bs.Amount, newAmount)
		}
	})
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/accounting/ -run TestStore_UpdateBalanceStatus -v -count=1`
Expected: FAIL

**Step 3: Implement UpdateBalanceStatus**

Add the case to `UpdateTransaction` switch:

```go
case BalanceStatusUpdate:
	return store.UpdateBalanceStatus(ctx, item, Id)
```

Implement using the same pattern as `updateIncomeExpense` but simpler (no category, no multiplier):

```go
func (store *Store) UpdateBalanceStatus(ctx context.Context, input BalanceStatusUpdate, id uint) error {
	params := updateIncomeExpenseParams{
		description:      input.Description,
		date:             input.Date,
		amount:           input.Amount,
		accountID:        input.AccountID,
		amountMultiplier: 1,
		txType:           BalanceStatusTransaction,
		entryType:        balanceStatusEntry,
	}
	return store.updateIncomeExpense(ctx, params, id)
}
```

Note: `updateIncomeExpense` handles the category check only when `expectedCategoryType` is set and `categoryID` is non-nil, which won't be the case here since we don't set them. Verify this works with the existing code — if `updateIncomeExpense` validates category unconditionally, write a standalone update instead. The zero value for `CategoryType` should cause it to skip category validation.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/accounting/ -run TestStore_UpdateBalanceStatus -v -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/accounting/transaction.go internal/accounting/transaction_test.go
git commit -m "feat: implement UpdateBalanceStatus"
```

---

### Task 5: Verify BalanceStatus does NOT affect balance calculation

**Files:**
- Modify: `internal/accounting/report_test.go` (or `transaction_test.go`)

**Step 1: Write the test**

```go
func TestStore_BalanceStatus_DoesNotAffectBalance(t *testing.T) {
	testdbs.RunTest(t, func(t *testing.T, db testdbs.DBGetter) {
		dbCon := db.ConnDbName("TestBalanceStatusNoBalance")
		store, _ := newAccountingStoreWithMarketData(t, dbCon)
		accountSampleData(t, store)

		ctx := t.Context()
		// Create an income entry
		_, err := store.CreateIncome(ctx, Income{
			Description: "salary",
			Amount:      1000,
			AccountID:   1,
			Date:        getDate("2025-01-15"),
		})
		if err != nil {
			t.Fatalf("create income: %v", err)
		}

		// Get balance before adding balance status
		balBefore, err := store.AccountBalanceSingle(ctx, 1, getDate("2025-12-31"))
		if err != nil {
			t.Fatalf("balance before: %v", err)
		}

		// Create a balance status entry
		_, err = store.CreateBalanceStatus(ctx, BalanceStatus{
			Description: "bank statement",
			Amount:      5000, // intentionally different from actual balance
			AccountID:   1,
			Date:        getDate("2025-06-15"),
		})
		if err != nil {
			t.Fatalf("create balance status: %v", err)
		}

		// Get balance after — should be identical
		balAfter, err := store.AccountBalanceSingle(ctx, 1, getDate("2025-12-31"))
		if err != nil {
			t.Fatalf("balance after: %v", err)
		}

		if balBefore.Sum != balAfter.Sum {
			t.Errorf("balance changed after adding balance status: before=%v, after=%v", balBefore.Sum, balAfter.Sum)
		}
	})
}
```

**Step 2: Run the test**

Run: `go test ./internal/accounting/ -run TestStore_BalanceStatus_DoesNotAffectBalance -v -count=1`
Expected: PASS (since `balanceStatusEntry` is not in `balanceEntryTypes`)

**Step 3: Commit**

```bash
git add internal/accounting/transaction_test.go
git commit -m "test: verify balance status does not affect balance calculation"
```

---

### Task 6: Add BalanceStatus to the API handler (create, get, list, update)

**Files:**
- Modify: `app/router/handlers/finance/transaction.go`

**Step 1: Add the `balanceStatusTxStr` constant and parser**

In `app/router/handlers/finance/transaction.go`, add to constants (after `stockTransferTxStr`):

```go
balanceStatusTxStr = "balancestatus"
```

Add to `parseTxType` switch:

```go
case balanceStatusTxStr:
	return accounting.BalanceStatusTransaction
```

**Step 2: Add BalanceStatus case to CreateTx handler**

In the `CreateTx` switch, add before `default`:

```go
case accounting.BalanceStatusTransaction:
	entry = accounting.BalanceStatus{
		Description: payload.Description,
		Date:        payload.Date.Time,
		Amount:      payload.Amount,
		AccountID:   payload.AccountId,
	}
```

**Step 3: Add BalanceStatus case to transactionToPayload**

```go
case accounting.BalanceStatus:
	return transactionPayload{
		Id:          entry.Id,
		Description: entry.Description,
		Date:        dateOnlyTime{Time: entry.Date},
		Type:        balanceStatusTxStr,
		Amount:      entry.Amount,
		AccountId:   entry.AccountID,
	}
```

**Step 4: Add BalanceStatus case to UpdateTx handler**

In the `UpdateTx` switch, add before `default`:

```go
case accounting.BalanceStatus:
	entry = accounting.BalanceStatusUpdate{
		Description: payload.Description,
		Date:        datePtr,
		Amount:      payload.Amount,
		AccountID:   payload.AccountId,
	}
```

**Step 5: Add `calculatedBalance` field to transactionPayload**

Add a new field to `transactionPayload`:

```go
CalculatedBalance *float64 `json:"calculatedBalance,omitempty"`
```

**Step 6: Compute calculatedBalance in ListTx**

In the `ListTx` handler, after the loop that builds `response.Items` (around line 582-583), add:

```go
for i, item := range response.Items {
	if item.Type == balanceStatusTxStr && item.AccountId != 0 {
		bal, err := h.Store.AccountBalanceSingle(r.Context(), item.AccountId, item.Date.Time)
		if err == nil {
			cb := bal.Sum
			response.Items[i].CalculatedBalance = &cb
		}
	}
}
```

Also compute it in `GetTx` — after `transactionToPayload`, if the type is `balanceStatusTxStr`:

```go
if payload.Type == balanceStatusTxStr && payload.AccountId != 0 {
	bal, err := h.Store.AccountBalanceSingle(r.Context(), payload.AccountId, payload.Date.Time)
	if err == nil {
		cb := bal.Sum
		payload.CalculatedBalance = &cb
	}
}
```

**Step 7: Run all backend tests**

Run: `go test ./... -count=1`
Expected: PASS

**Step 8: Commit**

```bash
git add app/router/handlers/finance/transaction.go
git commit -m "feat: add BalanceStatus to API handlers (create, get, list, update)"
```

---

### Task 7: Add BalanceStatus to the ignoreUnexported test helpers

**Files:**
- Modify: `internal/accounting/transaction_test.go`

**Step 1: Update ignoreUnexportedTxFields**

Add `cmpopts.IgnoreUnexported(BalanceStatus{})` to the `ignoreUnexportedTxFields` slice (around line 1180).

**Step 2: Run all accounting tests to make sure nothing is broken**

Run: `go test ./internal/accounting/ -v -count=1`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/accounting/transaction_test.go
git commit -m "test: add BalanceStatus to ignoreUnexported helpers"
```

---

### Task 8: Add BalanceStatus to ListTransactions (if filtered by type)

**Files:**
- Modify: `internal/accounting/transaction.go` (check if ListTransactions/PriorPageBalance need changes)

**Step 1: Verify BalanceStatus entries appear in ListTransactions**

Check that `ListTransactions` does not filter by type by default — it queries all `dbTransaction` records. Since `publicTransactions` now handles `BalanceStatusTransaction`, no changes should be needed for listing.

**Step 2: Verify PriorPageBalance excludes BalanceStatus**

`PriorPageBalance` sums `balanceEntryTypes` entries, which does NOT include `balanceStatusEntry`. This is correct — balance status should not affect the running balance. Verify with a test:

```go
func TestStore_PriorPageBalance_ExcludesBalanceStatus(t *testing.T) {
	testdbs.RunTest(t, func(t *testing.T, db testdbs.DBGetter) {
		dbCon := db.ConnDbName("TestPriorBalExcludesBS")
		store, _ := newAccountingStoreWithMarketData(t, dbCon)
		accountSampleData(t, store)

		ctx := t.Context()
		// Create some income entries
		for i := 1; i <= 5; i++ {
			_, _ = store.CreateIncome(ctx, Income{
				Description: "income",
				Amount:      100,
				AccountID:   1,
				Date:        getDate("2025-01-" + fmt.Sprintf("%02d", i)),
			})
		}

		// Create a balance status in the middle
		_, _ = store.CreateBalanceStatus(ctx, BalanceStatus{
			Description: "checkpoint",
			Amount:      9999,
			AccountID:   1,
			Date:        getDate("2025-01-03"),
		})

		opts := ListOpts{
			StartDate: getDate("2025-01-01"),
			EndDate:   getDate("2025-12-31"),
			AccountId: []int{1},
			Limit:     2, Page: 1,
		}
		got, err := store.PriorPageBalance(ctx, opts, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Prior balance should reflect only income entries, not the balance status amount
		// With 6 total transactions (5 income + 1 balance status), page 1 shows 2 newest,
		// prior balance = sum of the 3 oldest income entries = 300
		// (balance status contributes 0 to balance)
		if got != 300 {
			t.Errorf("PriorPageBalance = %v, want 300", got)
		}
	})
}
```

Note: The exact expected value depends on ordering. Adjust the test based on how transactions sort (DESC by date, then by id). The key assertion is that the balance status amount (9999) does NOT contribute.

**Step 3: Run the test**

Run: `go test ./internal/accounting/ -run TestStore_PriorPageBalance_ExcludesBalanceStatus -v -count=1`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/accounting/transaction_test.go
git commit -m "test: verify PriorPageBalance excludes balance status entries"
```

---

### Task 9: Frontend — update entry display, types, and API

**Files:**
- Modify: `webui/src/utils/entryDisplay.ts`
- Modify: `webui/src/types/entry.ts`
- Modify: `webui/src/types/account.ts`

**Step 1: Update entryDisplay.ts**

Replace `'opening-balance'` with `'balancestatus'` in `ENTRY_TYPE_ICONS`:

```typescript
const ENTRY_TYPE_ICONS: Record<string, string> = {
    expense: 'pi pi-minus text-red-500',
    income: 'pi pi-plus text-green-500',
    transfer: 'pi pi-arrow-right-arrow-left text-blue-500',
    stockbuy: 'pi pi-chart-line text-yellow-500',
    stocksell: 'pi pi-chart-line text-orange-500',
    stockgrant: 'pi pi-gift text-purple-500',
    stocktransfer: 'pi pi-arrow-right-arrow-left text-indigo-500',
    balancestatus: 'pi pi-calculator text-gray-500'
}
```

**Step 2: Update Entry type in `webui/src/types/entry.ts`**

Add `calculatedBalance` to the `Entry` interface:

```typescript
export interface Entry {
    id: string
    date: string
    description?: string
    amount: number
    accountId: string
    categoryId?: number
    notes?: string
    calculatedBalance?: number // only present for balancestatus entries
}
```

**Step 3: Add BALANCE_STATUS to ENTRY_OPERATIONS in `webui/src/types/account.ts`**

Add to `ENTRY_OPERATIONS`:

```typescript
BALANCE_STATUS: 'balanceStatus',
```

Add to `ALLOWED_OPERATIONS_BY_ACCOUNT_TYPE` for cash, checking, savings:

```typescript
[ACCOUNT_TYPES.CASH]: [
    ENTRY_OPERATIONS.INCOME,
    ENTRY_OPERATIONS.EXPENSE,
    ENTRY_OPERATIONS.TRANSFER,
    ENTRY_OPERATIONS.BALANCE_STATUS,
    ENTRY_OPERATIONS.IMPORT_CSV,
],
[ACCOUNT_TYPES.CHECKING]: [
    ENTRY_OPERATIONS.INCOME,
    ENTRY_OPERATIONS.EXPENSE,
    ENTRY_OPERATIONS.TRANSFER,
    ENTRY_OPERATIONS.BALANCE_STATUS,
    ENTRY_OPERATIONS.IMPORT_CSV,
],
[ACCOUNT_TYPES.SAVINGS]: [
    ENTRY_OPERATIONS.INCOME,
    ENTRY_OPERATIONS.EXPENSE,
    ENTRY_OPERATIONS.TRANSFER,
    ENTRY_OPERATIONS.BALANCE_STATUS,
    ENTRY_OPERATIONS.IMPORT_CSV,
],
```

**Step 4: Commit**

```bash
git add webui/src/utils/entryDisplay.ts webui/src/types/entry.ts webui/src/types/account.ts
git commit -m "feat: add balance status to frontend types and display config"
```

---

### Task 10: Frontend — create BalanceStatusDialog.vue

**Files:**
- Create: `webui/src/views/entries/dialogs/BalanceStatusDialog.vue`

**Step 1: Create the dialog component**

Follow the same pattern as `IncomeExpenseDialog.vue` but simpler — only fields: Description (optional), Amount (required), Date (required), Account (cash-only, required).

```vue
<script setup>
import { ref, watch, computed } from 'vue'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import { Form } from '@primevue/forms'
import { zodResolver } from '@primevue/forms/resolvers/zod'
import { z } from 'zod'
import { useEntries } from '@/composables/useEntries.ts'
import {
    getFormattedAccountId,
    getDateOnly,
    extractAccountId,
    toDateString,
    getSubmitValues
} from '@/composables/useEntryDialogForm'

import AccountSelector from '@/components/AccountSelector.vue'
import Message from 'primevue/message'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import DatePicker from 'primevue/datepicker'
import { useDateFormat } from '@/composables/useDateFormat'
import { accountValidation } from '@/utils/entryValidation'
import { getApiErrorMessage } from '@/utils/apiError'

const { createEntry, updateEntry, isCreating, isUpdating } = useEntries({})
const backendError = ref('')
const { pickerDateFormat, dateValidation } = useDateFormat()

const props = defineProps({
    isEdit: { type: Boolean, default: false },
    entryId: { type: Number, default: null },
    description: { type: String, default: '' },
    amount: { type: Number, default: 0 },
    date: { type: Date, default: () => new Date() },
    accountId: { type: Number, default: null },
    visible: { type: Boolean, default: false }
})

watch(() => props.visible, (v) => { if (!v) backendError.value = '' })

const formValues = ref({
    description: props.description,
    amount: props.amount,
    AccountId: getFormattedAccountId(props.accountId),
    date: getDateOnly(props.date)
})

const formKey = ref(0)
watch(props, (newProps) => {
    formValues.value = {
        description: newProps.description,
        amount: newProps.amount,
        AccountId: getFormattedAccountId(newProps.accountId),
        date: getDateOnly(newProps.date)
    }
    formKey.value++
})

const resolver = computed(() =>
    zodResolver(
        z.object({
            date: dateValidation.value,
            amount: z.number({ message: 'Amount is required' }),
            AccountId: accountValidation
        })
    )
)

const dialogTitle = computed(() => (props.isEdit ? 'Edit Balance Status' : 'Add Balance Status'))

const handleSubmit = async (e) => {
    e.preventDefault?.()
    if (e.valid === false) return
    const values = getSubmitValues(e, formValues)
    const accountId = extractAccountId(values.AccountId)
    const description = (values.description ?? formValues.value.description ?? '').toString().trim()
    const amount = Number(values.amount ?? formValues.value.amount ?? 0)
    const date = values.date ?? formValues.value.date

    if (accountId == null) return

    const entryData = {
        description: description || 'Balance status',
        Amount: amount,
        date: toDateString(date),
        accountId,
        type: 'balancestatus'
    }

    backendError.value = ''
    try {
        if (props.isEdit) {
            await updateEntry({ id: props.entryId, ...entryData })
        } else {
            await createEntry(entryData)
        }
        emit('update:visible', false)
    } catch (error) {
        backendError.value = getApiErrorMessage(error)
        console.error(`Failed to ${props.isEdit ? 'update' : 'create'} balance status:`, error)
    }
}

const emit = defineEmits(['update:visible'])
</script>

<template>
    <Dialog
        :visible="visible"
        @update:visible="$emit('update:visible', $event)"
        :draggable="false"
        modal
        :header="dialogTitle"
        class="entry-dialog"
    >
        <Form
            :key="formKey"
            v-slot="$form"
            :resolver="resolver"
            :initialValues="formValues"
            :validateOnValueUpdate="false"
            :validateOnBlur="false"
            @submit="handleSubmit"
        >
            <Message v-if="backendError" severity="error" :closable="false" class="mb-2">{{ backendError }}</Message>
            <div class="flex flex-column gap-3">
                <div>
                    <label for="description" class="form-label">Description</label>
                    <InputText
                        id="description"
                        v-model="formValues.description"
                        name="description"
                        placeholder="e.g. March bank statement"
                    />
                </div>

                <div>
                    <label for="amount" class="form-label">Stated Balance</label>
                    <InputNumber
                        id="amount"
                        v-model="formValues.amount"
                        name="amount"
                        :minFractionDigits="2"
                        :maxFractionDigits="2"
                        v-focus
                    />
                    <Message v-if="$form.amount?.invalid" severity="error" size="small">
                        {{ $form.amount.error?.message }}
                    </Message>
                </div>

                <div>
                    <label for="date" class="form-label">Date</label>
                    <DatePicker
                        id="date"
                        name="date"
                        v-model="formValues.date"
                        :showIcon="true"
                        iconDisplay="input"
                        :dateFormat="pickerDateFormat"
                        :showButtonBar="true"
                    />
                    <Message v-if="$form.date?.invalid" severity="error" size="small">
                        {{ $form.date.error?.message }}
                    </Message>
                </div>

                <div>
                    <label for="AccountId" class="form-label">Account</label>
                    <AccountSelector
                        v-model="formValues.AccountId"
                        name="AccountId"
                        :accountTypes="['cash', 'checkin', 'savings']"
                    />
                    <Message v-if="$form.AccountId?.invalid" severity="error" size="small">
                        {{ $form.AccountId.error?.message }}
                    </Message>
                </div>

                <div class="flex justify-content-end gap-3">
                    <Button
                        type="submit"
                        label="Save"
                        icon="pi pi-check"
                        :loading="isCreating || isUpdating"
                    />
                    <Button
                        type="button"
                        label="Cancel"
                        icon="pi pi-times"
                        severity="secondary"
                        @click="$emit('update:visible', false)"
                    />
                </div>
            </div>
        </Form>
    </Dialog>
</template>
```

**Step 2: Commit**

```bash
git add webui/src/views/entries/dialogs/BalanceStatusDialog.vue
git commit -m "feat: create BalanceStatusDialog component"
```

---

### Task 11: Frontend — wire BalanceStatusDialog into AddEntryMenu and view components

**Files:**
- Modify: `webui/src/views/entries/AddEntryMenu.vue`
- Modify: `webui/src/views/entries/EntriesView.vue`
- Modify: `webui/src/views/entries/AccountEntriesView.vue`

**Step 1: Update AddEntryMenu.vue**

Import `BalanceStatusDialog`:

```javascript
import BalanceStatusDialog from '@/views/entries/dialogs/BalanceStatusDialog.vue'
```

Add to `dialogs` ref:

```javascript
balanceStatus: false
```

Add to `allDropdownOptions` array (before Import CSV):

```javascript
{
    label: 'Balance Status',
    value: 'balanceStatus',
    icon: 'pi pi-calculator'
},
```

Add dialog template at the end of the template:

```vue
<BalanceStatusDialog
    v-model:visible="dialogs.balanceStatus"
    :isEdit="false"
    :account-id="defaultAccountId"
    @update:visible="dialogs.balanceStatus = $event"
/>
```

**Step 2: Update EntriesView.vue**

Import `BalanceStatusDialog`:

```javascript
import BalanceStatusDialog from '@/views/entries/dialogs/BalanceStatusDialog.vue'
```

Add to `dialogs`:

```javascript
balanceStatus: ref(false)
```

Add to `openEditEntryDialog` and `openDuplicateEntryDialog`:

```javascript
} else if (entry.type === 'balancestatus') {
    dialogs.balanceStatus.value = true
}
```

Add dialog template:

```vue
<BalanceStatusDialog
    v-model:visible="dialogs.balanceStatus.value"
    :is-edit="isEditMode"
    :entry-id="selectedEntry?.id"
    :description="selectedEntry?.description"
    :amount="selectedEntry?.Amount"
    :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date) : new Date())"
    :account-id="selectedEntry?.accountId"
/>
```

**Step 3: Update AccountEntriesView.vue**

Same changes as EntriesView.vue — import, dialog state, edit/duplicate handlers, template.

**Step 4: Commit**

```bash
git add webui/src/views/entries/AddEntryMenu.vue webui/src/views/entries/EntriesView.vue webui/src/views/entries/AccountEntriesView.vue
git commit -m "feat: wire BalanceStatusDialog into entry menu and views"
```

---

### Task 12: Frontend — display balance status in AccountEntriesTable with discrepancy

**Files:**
- Modify: `webui/src/views/entries/AccountEntriesTable.vue`
- Modify: `webui/src/views/entries/EntriesTable.vue`

**Step 1: Update AccountEntriesTable.vue**

Add `'balancestatus-row': data.type === 'balancestatus'` to `getRowClass`.

In the `entriesWithBalance` computed, handle `balancestatus` like `opening-balance` (entryAmount = 0, does not affect running balance):

```javascript
else if (entry.type === 'balancestatus') entryAmount = 0
```

In the Amount column template (cash layout), add a block for `balancestatus`:

```vue
<div v-else-if="data.type === 'balancestatus'" class="amount balance-status">
    {{ formatAmount(data.Amount) }}
    {{ getAccountCurrency(data.accountId) }}
    <span v-if="data.calculatedBalance != null" class="ml-2">
        <span v-if="Math.abs(data.Amount - data.calculatedBalance) < 0.01" class="text-green-500">
            <i class="pi pi-check-circle" />
        </span>
        <span v-else class="text-orange-500">
            Diff: {{ (data.Amount - data.calculatedBalance) >= 0 ? '+' : '' }}{{ formatAmount(data.Amount - data.calculatedBalance) }}
        </span>
    </span>
</div>
```

**Step 2: Update EntriesTable.vue**

Add `'balancestatus-row': data.type === 'balancestatus'` to `getRowClass`.

In the "All Transactions" Amount column, add:

```vue
<div v-else-if="data.type === 'balancestatus'" class="amount balance-status">
    {{ (data.Amount ?? 0).toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) }}
    {{ getAccountCurrency(data.accountId) }}
    <span v-if="data.calculatedBalance != null" class="ml-2">
        <span v-if="Math.abs((data.Amount ?? 0) - data.calculatedBalance) < 0.01" class="text-green-500">
            <i class="pi pi-check-circle" />
        </span>
        <span v-else class="text-orange-500">
            Diff: {{ ((data.Amount ?? 0) - data.calculatedBalance) >= 0 ? '+' : '' }}{{ ((data.Amount ?? 0) - data.calculatedBalance).toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) }}
        </span>
    </span>
</div>
```

In the Account column, add:

```vue
<span v-else-if="data.type === 'balancestatus'">
    {{ getAccountName(data.accountId) }}
</span>
```

**Step 3: Commit**

```bash
git add webui/src/views/entries/AccountEntriesTable.vue webui/src/views/entries/EntriesTable.vue
git commit -m "feat: display balance status with discrepancy in entry tables"
```

---

### Task 13: End-to-end verification

**Step 1: Run all Go tests**

Run: `go test ./... -count=1`
Expected: PASS

**Step 2: Run frontend build**

Run: `cd webui && npm run build`
Expected: No errors

**Step 3: Manual smoke test (if applicable)**

- Create a balance status entry for a cash account
- Verify it appears in the entries table with the calculator icon
- Verify it shows a discrepancy (or checkmark) in the Amount column
- Verify it does NOT change the running balance in the Balance column
- Edit the balance status entry, verify changes save
- Delete the balance status entry, verify it disappears

**Step 4: Commit any final fixes**

```bash
git add -A
git commit -m "feat: balance status implementation complete"
```
