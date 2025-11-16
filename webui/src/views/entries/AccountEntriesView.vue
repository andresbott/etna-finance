<script setup>
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'

import DateRangePicker from '@/components/common/DateRangePicker.vue'
import TransferDialog from './dialogs/TransferDialog.vue'
import StockDialog from './dialogs/StockDialog.vue'
import DeleteDialog from '@/components/common/confirmDialog.vue'
import AccountEntriesTable from './AccountEntriesTable.vue'

import { useEntries } from '@/composables/useEntries.ts'
import IncomeExpenseDialog from '@/views/entries/dialogs/IncomeExpenseDialog.vue'
import AddEntryMenu from '@/views/entries/AddEntryMenu.vue'
import { useAccountUtils } from '@/utils/accountUtils'
import { useAccounts } from '@/composables/useAccounts'

/* --- Route --- */
const route = useRoute()
const accountId = computed(() => route.params.id)

/* --- Reactive State --- */
const today = new Date()
const startDate = ref(new Date(today.setDate(today.getDate() - 35)))
const endDate = ref(new Date())

const { entries: allEntries, isLoading, deleteEntry, isDeleting, refetch } = useEntries(
    startDate,
    endDate
)

/* --- Accounts --- */
const { accounts } = useAccounts()

/* --- Account Name and Currency --- */
const accountName = computed(() => {
    if (!accountId.value) return 'Unknown Account'
    if (!accounts?.value) return 'Loading...'
    
    // Find the account directly
    for (const provider of accounts.value) {
        if (provider.accounts) {
            for (const account of provider.accounts) {
                if (String(account.id) === String(accountId.value)) {
                    return account.name
                }
            }
        }
    }
    
    return 'Unknown Account'
})

const accountCurrency = computed(() => {
    if (!accountId.value) return ''
    if (!accounts?.value) return ''
    
    // Find the account currency
    for (const provider of accounts.value) {
        if (provider.accounts) {
            for (const account of provider.accounts) {
                if (String(account.id) === String(accountId.value)) {
                    return account.currency
                }
            }
        }
    }
    
    return ''
})

const accountTitle = computed(() => {
    if (accountCurrency.value) {
        return `${accountName.value} (${accountCurrency.value})`
    }
    return accountName.value
})

/* --- Filtered Entries --- */
const entries = computed(() => {
    if (!accountId.value || !allEntries.value) return []
    
    // Filter entries that belong to this account
    const filtered = allEntries.value.filter(entry => {
        // For income/expense entries, check accountId
        if (entry.type === 'income' || entry.type === 'expense') {
            return String(entry.accountId) === String(accountId.value)
        }
        // For transfers, check both origin and target accounts
        if (entry.type === 'transfer') {
            return String(entry.originAccountId) === String(accountId.value) || 
                   String(entry.targetAccountId) === String(accountId.value)
        }
        // For stock operations, check targetAccountId
        if (entry.type === 'buystock' || entry.type === 'sellstock') {
            return String(entry.targetAccountId) === String(accountId.value)
        }
        return false
    })
    
    // Calculate opening balance (balance at start of period)
    // Get all entries before startDate to calculate this
    const openingBalance = allEntries.value
        .filter(entry => {
            // Only include entries before startDate that belong to this account
            const entryDate = new Date(entry.date)
            const start = new Date(startDate.value)
            
            if (entryDate >= start) return false
            
            // Check if entry belongs to this account
            if (entry.type === 'income' || entry.type === 'expense') {
                return String(entry.accountId) === String(accountId.value)
            }
            if (entry.type === 'transfer') {
                return String(entry.originAccountId) === String(accountId.value) || 
                       String(entry.targetAccountId) === String(accountId.value)
            }
            if (entry.type === 'buystock' || entry.type === 'sellstock') {
                return String(entry.targetAccountId) === String(accountId.value)
            }
            return false
        })
        .reduce((balance, entry) => {
            let entryAmount = 0
            
            if (entry.type === 'expense') {
                entryAmount = -(entry.Amount || 0)
            } else if (entry.type === 'income') {
                entryAmount = entry.Amount || 0
            } else if (entry.type === 'transfer') {
                if (String(entry.originAccountId) === String(accountId.value)) {
                    entryAmount = -(entry.originAmount || 0)
                } else if (String(entry.targetAccountId) === String(accountId.value)) {
                    entryAmount = entry.targetAmount || 0
                }
            } else if (entry.type === 'buystock') {
                entryAmount = -(entry.targetAmount || 0)
            } else if (entry.type === 'sellstock') {
                entryAmount = entry.targetAmount || 0
            }
            
            return balance + entryAmount
        }, 0)
    
    // Create opening balance entry
    const openingBalanceEntry = {
        id: 'opening-balance',
        type: 'opening-balance',
        description: 'Balance at beginning of period',
        date: startDate.value,
        Amount: openingBalance,
        accountId: accountId.value,
        isOpeningBalance: true
    }
    
    // Return filtered entries followed by opening balance at the end (bottom in descending order)
    return [...filtered, openingBalanceEntry]
})

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
    stock: ref(false)
}

/* --- Entry Actions --- */
const openEditEntryDialog = (entry) => {
    isEditMode.value = true
    isDuplicateMode.value = false
    selectedEntry.value = entry
    console.log(entry)

    if (entry.type === 'expense' || entry.type === 'income') {
        // Use IncomeExpenseDialog for income and expense entries
        dialogs.incomeExpense.value = true
    } else if (entry.type === 'transfer') {
        // Use TransferDialog for transfer entries
        dialogs.transfer.value = true
    } else if (entry.type === 'buystock' || entry.type === 'sellstock') {
        // Use StockDialog for stock entries
        dialogs.stock.value = true
    }
}

const openDuplicateEntryDialog = (entry) => {
    isEditMode.value = false // Not in edit mode, creating a new entry
    isDuplicateMode.value = true
    selectedEntry.value = entry
    console.log('Duplicating entry:', entry)

    if (entry.type === 'expense' || entry.type === 'income') {
        // Use IncomeExpenseDialog for income and expense entries
        dialogs.incomeExpense.value = true
    } else if (entry.type === 'transfer') {
        // Use TransferDialog for transfer entries
        dialogs.transfer.value = true
    } else if (entry.type === 'buystock' || entry.type === 'sellstock') {
        // Use StockDialog for stock entries
        dialogs.stock.value = true
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
                    />
                </div>
            </div>

            <div class="entries-view">
                <AccountEntriesTable
                    :entries="entries"
                    :isLoading="isLoading"
                    :isDeleting="isDeleting"
                    :accountId="accountId"
                    @edit="openEditEntryDialog"
                    @duplicate="openDuplicateEntryDialog"
                    @delete="openDeleteDialog"
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
        :amount="isDuplicateMode ? undefined : selectedEntry?.Amount"
        :account-id="selectedEntry?.accountId"
        :stock-amount="isDuplicateMode ? undefined : selectedEntry?.targetStockAmount"
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date) : new Date())"
        :entry-id="selectedEntry?.id"
        :category-id="selectedEntry?.category?.id"
        :autofocus-amount="isDuplicateMode"
    />

    <TransferDialog
        v-model:visible="dialogs.transfer.value"
        :is-edit="isEditMode"
        :entry-id="selectedEntry?.id"
        :description="selectedEntry?.description"
        :target-amount="isDuplicateMode ? undefined : selectedEntry?.targetAmount"
        :origin-amount="isDuplicateMode ? undefined : selectedEntry?.originAmount"
        :target-stock-amount="isDuplicateMode ? undefined : selectedEntry?.targetStockAmount"
        :origin-stock-amount="isDuplicateMode ? undefined : selectedEntry?.originStockAmount"
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date) : new Date())"
        :target-account-id="selectedEntry?.targetAccountId"
        :origin-account-id="selectedEntry?.originAccountId"
        :autofocus-amount="isDuplicateMode"
    />

    <StockDialog
        v-model:visible="dialogs.stock.value"
        :is-edit="isEditMode"
        :entry-id="selectedEntry?.id"
        :description="selectedEntry?.description"
        :amount="isDuplicateMode ? undefined : selectedEntry?.targetAmount"
        :stock-amount="isDuplicateMode ? undefined : selectedEntry?.targetStockAmount"
        :date="isDuplicateMode ? new Date() : (selectedEntry?.date ? new Date(selectedEntry.date) : new Date())"
        :type="selectedEntry?.type"
        :target-account-id="selectedEntry?.targetAccountId"
        :autofocus-amount="isDuplicateMode"
    />

    <!-- Delete Confirmation Dialog -->
    <DeleteDialog
        v-model:visible="deleteDialogVisible"
        :name="entryToDelete?.description"
        message="Are you sure you want to delete this entry?"
        :onConfirm="handleDeleteEntry"
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
}

.entries-view {
    flex: 1;
    overflow: auto;
    padding: 1rem;
}
</style>

