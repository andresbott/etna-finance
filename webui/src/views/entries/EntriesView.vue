<script setup>
import { ref, computed, watch } from 'vue'

import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Card from 'primevue/card'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import TransferDialog from './dialogs/TransferDialog.vue'
import StockDialog from './dialogs/StockDialog.vue'
import DeleteDialog from '@/components/common/confirmDialog.vue'

import { useEntries } from '@/composables/useEntries.ts'
import IncomeExpenseDialog from '@/views/entries/dialogs/IncomeExpenseDialog.vue'
import AddEntryMenu from '@/views/entries/AddEntryMenu.vue'
import { useCategoryUtils } from '@/utils/categoryUtils'
import { useAccountUtils } from '@/utils/accountUtils'

/* --- Reactive State --- */
const today = new Date()
const startDate = ref(new Date(today.setDate(today.getDate() - 35)))
const endDate = ref(new Date())

const { entries, isLoading, deleteEntry, isDeleting, refetch } = useEntries(
    startDate,
    endDate
)
const { getCategoryName } = useCategoryUtils()
const { getAccountCurrency, getAccountName } = useAccountUtils()

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
    } catch (error) {
        console.error('Failed to delete entry:', error)
    }
}

/* --- Helpers --- */
const getEntryTypeIcon = (type) => {
    const icons = {
        expense: 'pi pi-minus text-red-500',
        income: 'pi pi-plus text-green-500',
        transfer: 'pi pi-arrow-right-arrow-left text-blue-500',
        buystock: 'pi pi-chart-line text-yellow-500',
        sellstock: 'pi pi-chart-line text-orange-500'
    }
    return icons[type] || 'pi pi-question-circle'
}

const getRowClass = (data) => ({
    'expense-row': data.type === 'expense',
    'income-row': data.type === 'income',
    'transfer-row': data.type === 'transfer',
    'buystock-row': data.type === 'buystock',
    'sellstock-row': data.type === 'sellstock'
})
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
                <Card>
                    <template #content>
                        <DataTable
                            :value="entries"
                            :loading="isLoading"
                            stripedRows
                            paginator
                            style="width: 100%"
                            :rows="50"
                            :rowsPerPageOptions="[50, 100, 200]"
                            :rowClass="getRowClass"
                        >
                        <Column header="" style="width: 40px">
                            <template #body="{ data }">
                                <i :class="getEntryTypeIcon(data.type)" style="font-size: 0.8rem" />
                            </template>
                        </Column>

                        <Column field="description" header="Description" />
                        <Column field="category" header="Category">
                            <template #body="{ data }">
                                <div v-if="data.type === 'expense' || data.type === 'income'">
                                    {{ getCategoryName(data?.categoryId, data.type) }}
                                </div>
                                <div v-else>-</div>
                            </template>
                        </Column>

                        <Column header="Account">
                            <template #body="{ data }">
                                <span v-if="data.type === 'transfer'">
                                    {{ getAccountName(data.originAccountId)
                                    }}<i
                                        class="pi pi-arrow-right"
                                        style="font-size: 0.9rem; margin: 0 8px"
                                    />{{ getAccountName(data.targetAccountId) }}
                                </span>
                                <span v-else>
                                    {{ getAccountName(data.accountId) }}
                                </span>
                            </template>
                        </Column>
                        <Column field="date" header="Date">
                            <template #body="{ data }">
                                {{
                                    new Date(data.date).toLocaleDateString('es-ES', {
                                        day: '2-digit',
                                        month: '2-digit',
                                        year: '2-digit'
                                    })
                                }}
                            </template>
                        </Column>
                        <Column field="Amount" header="Amount">
                            <template #body="{ data }">
                                <div v-if="data.type === 'expense'" class="amount expense">
                                    -{{
                                        data.Amount.toLocaleString('es-ES', {
                                            minimumFractionDigits: 2,
                                            maximumFractionDigits: 2
                                        })
                                    }}
                                    {{ getAccountCurrency(data.accountId) }}
                                </div>
                                <div v-else-if="data.type === 'income'" class="amount income">
                                    {{
                                        data.Amount.toLocaleString('es-ES', {
                                            minimumFractionDigits: 2,
                                            maximumFractionDigits: 2
                                        })
                                    }}
                                    {{ getAccountCurrency(data.accountId) }}
                                </div>
                                <div v-else-if="data.type === 'transfer'" class="amount transfer">
                                    {{
                                        data.originAmount?.toLocaleString('es-ES', {
                                            minimumFractionDigits: 2,
                                            maximumFractionDigits: 2
                                        }) || '0.00'
                                    }}
                                    {{ data.originAccountCurrency || '' }}
                                    <i
                                        class="pi pi-arrow-right"
                                        style="font-size: 0.9rem; margin: 0 8px"
                                    />
                                    {{
                                        data.targetAmount.toLocaleString('es-ES', {
                                            minimumFractionDigits: 2,
                                            maximumFractionDigits: 2
                                        })
                                    }}
                                    {{ data.targetAccountCurrency || '' }}
                                </div>
                                <div v-else class="amount">
                                    {{
                                        data.targetAmount.toLocaleString('es-ES', {
                                            minimumFractionDigits: 2,
                                            maximumFractionDigits: 2
                                        })
                                    }}
                                    {{ data.targetAccountCurrency || '' }}
                                </div>
                            </template>
                        </Column>

                        <Column header="Actions" style="width: 150px">
                            <template #body="{ data }">
                                <div class="actions">
                                    <Button
                                        icon="pi pi-pencil"
                                        text
                                        rounded
                                        class="action-button"
                                        @click="openEditEntryDialog(data)"
                                        v-tooltip.bottom="'Edit'"
                                    />
                                    <Button
                                        icon="pi pi-copy"
                                        text
                                        rounded
                                        class="action-button"
                                        @click="openDuplicateEntryDialog(data)"
                                        v-tooltip.bottom="'Duplicate'"
                                    />
                                    <Button
                                        icon="pi pi-trash"
                                        text
                                        rounded
                                        severity="danger"
                                        class="action-button"
                                        :loading="isDeleting"
                                        @click="openDeleteDialog(data)"
                                        v-tooltip.bottom="'Delete'"
                                    />
                                </div>
                            </template>
                        </Column>
                    </DataTable>
                    </template>
                </Card>
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

.actions {
    display: flex;
    gap: 0.5rem;
    justify-content: flex-start;
}

.action-button {
    padding: 0.25rem;
}

:deep(.p-datatable-tbody > tr > td) {
    padding-top: 0;
    padding-bottom: 0;
}

:deep(.p-datatable .p-datatable-tbody > tr:hover) {
    background-color: rgba(0, 0, 0, 0.1) !important;
}

.amount.expense {
    color: var(--c-red-600);
}

.amount.income {
    color: var(--c-green-600);
}

.amount.transfer {
    color: var(--c-blue-600);
}
</style>
