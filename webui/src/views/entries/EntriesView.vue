<script setup>
import { ref, computed, watch } from 'vue'

import DateRangePicker from '@/components/common/DateRangePicker.vue'
import TransferDialog from './dialogs/TransferDialog.vue'
import StockDialog from './dialogs/StockDialog.vue'
import DeleteDialog from '@/components/common/confirmDialog.vue'
import EntriesTable from './EntriesTable.vue'

import { useEntries } from '@/composables/useEntries.ts'
import IncomeExpenseDialog from '@/views/entries/dialogs/IncomeExpenseDialog.vue'
import AddEntryMenu from '@/views/entries/AddEntryMenu.vue'

/* --- Reactive State --- */
const today = new Date()
const startDate = ref(new Date(today.setDate(today.getDate() - 35)))
const endDate = ref(new Date())

const { entries, isLoading, deleteEntry, isDeleting, refetch } = useEntries(
    startDate,
    endDate
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
                    :isLoading="isLoading"
                    :isDeleting="isDeleting"
                    @edit="openEditEntryDialog"
                    @duplicate="openDuplicateEntryDialog"
                    @delete="openDeleteDialog"
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
