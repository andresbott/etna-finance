<script setup>
import { ref, computed, watch } from 'vue'

import DateRangePicker from '@/components/common/DateRangePicker.vue'
import TransferDialog from './dialogs/TransferDialog.vue'
import BuySellInstrumentDialog from './dialogs/BuySellInstrumentDialog.vue'
import GrantDialog from './dialogs/GrantDialog.vue'
import TransferInstrumentDialog from './dialogs/TransferInstrumentDialog.vue'
import DeleteDialog from '@/components/common/confirmDialog.vue'
import EntriesTable from './EntriesTable.vue'

import { useEntries } from '@/composables/useEntries.ts'
import IncomeExpenseDialog from '@/views/entries/dialogs/IncomeExpenseDialog.vue'
import AddEntryMenu from '@/views/entries/AddEntryMenu.vue'

/* --- Reactive State --- */
const today = new Date()
const startDate = ref(new Date(today.setDate(today.getDate() - 35)))
const endDate = ref(new Date())

/* --- Pagination State --- */
const page = ref(1)
const limit = ref(25)
const first = ref(0) // First row index for DataTable

const { entries, totalRecords, isLoading, isFetching, deleteEntry, isDeleting, refetch } = useEntries({
    startDate,
    endDate,
    page,
    limit
})

/* --- Computed pagination values for template --- */
const paginationRows = computed(() => limit.value)
const paginationFirst = computed(() => first.value)
const paginationTotal = computed(() => totalRecords.value || 0)

/* --- Pagination Handler --- */
const handlePage = (event) => {
    page.value = event.page + 1 // PrimeVue uses 0-based page, API uses 1-based
    limit.value = event.rows
    first.value = event.first
}

/* --- Reset pagination when date range changes --- */
watch([startDate, endDate], () => {
    page.value = 1
    first.value = 0
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
    buyStock: ref(false),
    sellStock: ref(false),
    grantStock: ref(false),
    transferInstrument: ref(false)
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
        dialogs.transfer.value = true
    } else if (entry.type === 'stockbuy') {
        dialogs.buyStock.value = true
    } else if (entry.type === 'stocksell') {
        dialogs.sellStock.value = true
    } else if (entry.type === 'stockgrant') {
        dialogs.grantStock.value = true
    } else if (entry.type === 'stocktransfer') {
        dialogs.transferInstrument.value = true
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
        dialogs.transfer.value = true
    } else if (entry.type === 'stockbuy') {
        dialogs.buyStock.value = true
    } else if (entry.type === 'stocksell') {
        dialogs.sellStock.value = true
    } else if (entry.type === 'stockgrant') {
        dialogs.grantStock.value = true
    } else if (entry.type === 'stocktransfer') {
        dialogs.transferInstrument.value = true
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
                <div class="toolbar-spacer"></div>
                <div class="date-filters">
                    <DateRangePicker
                        v-model:startDate="startDate"
                        v-model:endDate="endDate"
                        @change="refetch"
                    />
                </div>
                <div class="add-entry-menu">
                    <AddEntryMenu />
                </div>
            </div>

            <div class="entries-view">
                <EntriesTable
                    :entries="entries"
                    :isLoading="isLoading || isFetching"
                    :isDeleting="isDeleting"
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
    <!-- TODO: Pass the `category-id`  -->
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

    <BuySellInstrumentDialog
        v-model:visible="dialogs.buyStock.value"
        :is-edit="isEditMode"
        :entry-id="selectedEntry?.id"
        operation-type="buy"
        :instrument-id="selectedEntry?.instrumentId"
        :description="selectedEntry?.description"
        :quantity="selectedEntry?.quantity"
        :price-per-share="selectedEntry?.StockAmount && selectedEntry?.quantity ? selectedEntry.StockAmount / selectedEntry.quantity : undefined"
        :cash-amount="isDuplicateMode ? undefined : selectedEntry?.totalAmount"
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
        :price-per-share="selectedEntry?.StockAmount && selectedEntry?.quantity ? selectedEntry.StockAmount / selectedEntry.quantity : undefined"
        :cash-amount="isDuplicateMode ? undefined : selectedEntry?.totalAmount"
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
