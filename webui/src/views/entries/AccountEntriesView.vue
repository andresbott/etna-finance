<script setup>
import { ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'

import DateRangePicker from '@/components/common/DateRangePicker.vue'
import AccountEntriesTable from './AccountEntriesTable.vue'
import EntryDialogs from './EntryDialogs.vue'

import { useEntries } from '@/composables/useEntries.ts'
import { useEntryDialogs } from '@/composables/useEntryDialogs'
import { useRouteState } from '@/composables/useRouteState'
import AddEntryMenu from '@/views/entries/AddEntryMenu.vue'
import { useAccounts } from '@/composables/useAccounts'
import { findAccountById } from '@/utils/accountUtils'
import { useBalance } from '@/composables/useGetBalanceReport'

/* --- Route --- */
const route = useRoute()
const accountId = computed(() => route.params.id)

/* --- Reactive State (synced with URL query params) --- */
const today = new Date()
const { startDate, endDate, page, limit } = useRouteState({
    startDate: new Date(today.getFullYear(), today.getMonth(), today.getDate() - 35),
    endDate: new Date(),
    limit: 25
})

// Create accountIds array for the API query - filters entries server-side
const accountIds = computed(() => accountId.value ? [String(accountId.value)] : [])
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

const {
    selectedEntry, isEditMode, isDuplicateMode, dialogs,
    deleteDialogVisible, entryToDelete,
    openEditEntryDialog, openDuplicateEntryDialog, openDeleteDialog, handleDeleteEntry
} = useEntryDialogs(deleteEntry)
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

