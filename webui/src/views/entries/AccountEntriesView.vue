<script setup>
import { ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'

import DateRangePicker from '@/components/common/DateRangePicker.vue'
import TransferDialog from './dialogs/TransferDialog.vue'
import BuySellInstrumentDialog from './dialogs/BuySellInstrumentDialog.vue'
import GrantDialog from './dialogs/GrantDialog.vue'
import TransferInstrumentDialog from './dialogs/TransferInstrumentDialog.vue'
import DeleteDialog from '@/components/common/confirmDialog.vue'
import AccountEntriesTable from './AccountEntriesTable.vue'

import { useEntries } from '@/composables/useEntries.ts'
import { getEntry } from '@/lib/api/Entry'
import { useRouteState } from '@/composables/useRouteState'
import IncomeExpenseDialog from '@/views/entries/dialogs/IncomeExpenseDialog.vue'
import BalanceStatusDialog from '@/views/entries/dialogs/BalanceStatusDialog.vue'
import AddEntryMenu from '@/views/entries/AddEntryMenu.vue'
import { useAccounts } from '@/composables/useAccounts'
import { findAccountById } from '@/utils/accountUtils'
import { useBalance } from '@/composables/useGetBalanceReport'

/* --- Route --- */
const route = useRoute()
const accountId = computed(() => route.params.id)

/* --- Reactive State (synced with URL query params) --- */
const today = new Date()
const { startDate, endDate, page } = useRouteState({
    startDate: new Date(today.getFullYear(), today.getMonth(), today.getDate() - 35),
    endDate: new Date()
})

// Create accountIds array for the API query - filters entries server-side
const accountIds = computed(() => accountId.value ? [String(accountId.value)] : [])

/* --- Pagination State --- */
const limit = ref(25)
const first = computed(() => (page.value - 1) * limit.value)

const { entries: fetchedEntries, totalRecords, priorBalance, isLoading, isFetching, deleteEntry, isDeleting, refetch } = useEntries({
    startDate,
    endDate,
    accountIds,
    page,
    limit
})

/* --- Computed pagination values for template --- */
const paginationRows = computed(() => limit.value)
const paginationFirst = computed(() => first.value)
const paginationTotal = computed(() => (totalRecords.value || 0) + 1) // +1 for opening balance entry

/* --- Pagination Handler --- */
const handlePage = (event) => {
    page.value = event.page + 1 // PrimeVue uses 0-based page, API uses 1-based
    limit.value = event.rows
}

/* --- Reset pagination when date range or account changes --- */
watch([startDate, endDate, accountId], () => {
    page.value = 1
})

/* --- Accounts --- */
const { accounts } = useAccounts()

/* --- Balance API --- */
const { accountBalance } = useBalance()
const openingBalance = ref(0)
const isLoadingBalance = ref(false)

/* --- Account Name and Currency --- */
const accountName = computed(() => {
    if (!accountId.value) return 'Unknown Account'
    if (!accounts?.value) return 'Loading...'
    return currentAccount.value?.name ?? 'Unknown Account'
})

const accountCurrency = computed(() => currentAccount.value?.currency ?? '')

const currentAccount = computed(() => {
    if (!accountId.value || !accounts?.value) return null
    return findAccountById(accounts.value, accountId.value)
})

const accountType = computed(() => currentAccount.value?.type ?? null)

const accountTitle = computed(() => {
    if (accountCurrency.value) {
        return `${accountName.value} (${accountCurrency.value})`
    }
    return accountName.value
})

/* --- Entries with Opening Balance --- */
const entries = computed(() => {
    if (!accountId.value || !fetchedEntries.value) return []
    
    // Create opening balance entry using the API-fetched balance + prior page balance
    // priorBalance accounts for entries on older pages not visible on the current page
    const openingBalanceEntry = {
        id: 'opening-balance',
        type: 'opening-balance',
        description: 'Balance at beginning of period',
        date: startDate.value,
        Amount: openingBalance.value + priorBalance.value,
        accountId: accountId.value,
        isOpeningBalance: true
    }
    
    // Return fetched entries (already filtered by backend) followed by opening balance at the end
    return [...fetchedEntries.value, openingBalanceEntry]
})

// Watch for changes in accountId or startDate to fetch the opening balance
watch(
    [accountId, startDate],
    async ([newAccountId, newStartDate]) => {
        if (!newAccountId || !newStartDate) {
            openingBalance.value = 0
            return
        }
        
        try {
            isLoadingBalance.value = true
            const dateStr = new Date(newStartDate).toISOString().split('T')[0]
            const balance = await accountBalance.mutateAsync({
                accountId: Number(newAccountId),
                date: dateStr
            })
            openingBalance.value = balance || 0
        } catch (error) {
            console.error('Failed to fetch opening balance:', error)
            openingBalance.value = 0
        } finally {
            isLoadingBalance.value = false
        }
    },
    { immediate: true }
)

const selectedEntry = ref(null)
const isEditMode = ref(false)
const isDuplicateMode = ref(false)

/* --- Delete Dialog State --- */
const deleteDialogVisible = ref(false)
const entryToDelete = ref(null)

/* --- Dialog Visibility State --- */
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

/* --- Entry Actions --- */
const openEditEntryDialog = async (entry) => {
    isEditMode.value = true
    isDuplicateMode.value = false
    // For sells, list API does not include fees; fetch full entry so dialog shows correct net + fees
    if (entry.type === 'stocksell') {
        try {
            const full = await getEntry(entry.id)
            selectedEntry.value = full
        } catch (e) {
            console.error('Failed to load sell entry for edit', e)
            selectedEntry.value = entry
        }
    } else {
        selectedEntry.value = entry
    }

    if (entry.type === 'expense' || entry.type === 'income') {
        dialogs.incomeExpense.value = true
    } else if (entry.type === 'transfer') {
        dialogs.transfer.value = true
    } else if (entry.type === 'stockbuy') {
        dialogs.buyStock.value = true
    } else if (entry.type === 'stocksell') {
        dialogs.sellStock.value = true
    } else if (entry.type === 'stockgrant') {
        dialogs.grantStock.value = true
    } else if (entry.type === 'stocktransfer') {
        dialogs.transferInstrument.value = true
    } else if (entry.type === 'balancestatus') {
        dialogs.balanceStatus.value = true
    }
}

const openDuplicateEntryDialog = (entry) => {
    isEditMode.value = false // Not in edit mode, creating a new entry
    isDuplicateMode.value = true
    selectedEntry.value = entry

    if (entry.type === 'expense' || entry.type === 'income') {
        // Use IncomeExpenseDialog for income and expense entries
        dialogs.incomeExpense.value = true
    } else if (entry.type === 'transfer') {
        dialogs.transfer.value = true
    } else if (entry.type === 'stockbuy') {
        dialogs.buyStock.value = true
    } else if (entry.type === 'stocksell') {
        dialogs.sellStock.value = true
    } else if (entry.type === 'stockgrant') {
        dialogs.grantStock.value = true
    } else if (entry.type === 'stocktransfer') {
        dialogs.transferInstrument.value = true
    } else if (entry.type === 'balancestatus') {
        dialogs.balanceStatus.value = true
    }
}

const openDeleteDialog = (entry) => {
    entryToDelete.value = entry
    deleteDialogVisible.value = true
}

const handleDeleteEntry = async () => {
    try {
        await deleteEntry(entryToDelete.value.id)
        deleteDialogVisible.value = false
    } catch (error) {
        console.error('Failed to delete entry:', error)
        // Keep dialog open on error so user knows something went wrong
    }
}
</script>

<template>
    <div class="main-app-content">
        <div class="entries-content">
            <div class="toolbar">
                <div class="toolbar-left">
                    <h2 class="account-title">
                        {{ accountTitle }}
                    </h2>
                </div>
                <div class="date-filters">
                    <DateRangePicker
                        v-model:startDate="startDate"
                        v-model:endDate="endDate"
                        @change="refetch"
                    />
                </div>
                <div class="add-entry-menu">
                    <AddEntryMenu
                        :default-account-id="Number(accountId)"
                        :default-origin-account-id="Number(accountId)"
                        :account-type="accountType"
                        :has-import-profile="!!currentAccount?.importProfileId"
                    />
                </div>
            </div>

            <div class="entries-view">
                <AccountEntriesTable
                    :entries="entries"
                    :isLoading="isLoading || isFetching"
                    :isDeleting="isDeleting"
                    :accountId="accountId"
                    :accountType="accountType"
                    :totalRecords="paginationTotal"
                    :rows="paginationRows"
                    :first="paginationFirst"
                    @edit="openEditEntryDialog"
                    @duplicate="openDuplicateEntryDialog"
                    @delete="openDeleteDialog"
                    @page="handlePage"
                />
            </div>
        </div>
    </div>

    <!-- Dialog Components -->
    <IncomeExpenseDialog
        v-model:visible="dialogs.incomeExpense.value"
        :is-edit="isEditMode"
        :entry-type="selectedEntry?.type"
        :description="selectedEntry?.description"
        :amount="selectedEntry?.Amount"
        :account-id="selectedEntry?.accountId"
        :stock-amount="selectedEntry?.targetStockAmount"
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date) : new Date())"
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
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date) : new Date())"
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
        :price-per-share="selectedEntry?.StockAmount && selectedEntry?.quantity ? selectedEntry.StockAmount / selectedEntry.quantity : undefined"
        :cash-amount="selectedEntry?.totalAmount"
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date) : new Date())"
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
        :price-per-share="(selectedEntry?.quantity && (selectedEntry?.costBasis != null || selectedEntry?.StockAmount != null)) ? ((selectedEntry?.costBasis ?? selectedEntry?.StockAmount) / selectedEntry.quantity) : undefined"
        :cash-amount="(selectedEntry?.totalAmount ?? 0) - (selectedEntry?.fees ?? 0)"
        :fees="selectedEntry?.fees ?? 0"
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date) : new Date())"
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
        :fair-market-value="selectedEntry?.fairMarketValue ?? 0"
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date) : new Date())"
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
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date) : new Date())"
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
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date) : new Date())"
        :account-id="selectedEntry?.accountId"
        :attachment-id="selectedEntry?.attachmentId"
    />

    <!-- Delete Confirmation Dialog -->
    <DeleteDialog
        v-model:visible="deleteDialogVisible"
        :name="entryToDelete?.description"
        message="Are you sure you want to delete this entry?"
        @confirm="handleDeleteEntry"
    />
</template>

<style scoped>
.main-app-content {
    display: flex;
    flex-direction: column;
    height: 100%;
}

.entries-content {
    display: flex;
    flex-direction: column;
    flex: 1;
    overflow: hidden;
}

.toolbar {
    display: grid;
    grid-template-columns: 1fr auto 1fr;
    align-items: center;
    padding: 1rem;
    background-color: var(--surface-ground);
    border-bottom: 1px solid var(--surface-border);
}

.toolbar-left {
    display: flex;
    align-items: center;
}

.account-title {
    margin: 0;
    font-size: 1.5rem;
    font-weight: 600;
    color: var(--c-primary-700);
}

.date-filters {
    display: flex;
    gap: 1rem;
    align-items: center;
    justify-content: center;
}

.add-entry-menu {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 0.5rem;
}

.entries-view {
    flex: 1;
    overflow: auto;
    padding: 1rem;
}
</style>

