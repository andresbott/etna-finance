<script setup>
import { ref } from 'vue'
import { VerticalLayout, HorizontalLayout, Placeholder } from '@go-bumbu/vue-components/layout'
import '@go-bumbu/vue-components/layout.css'
import TopBar from '@/views/topbar.vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import { useEntries } from '@/composables/useEntries.js'
import { useAccounts } from '@/composables/useAccounts.js'
import ExpenseDialog from './ExpenseDialog.vue'
import IncomeDialog from './IncomeDialog.vue'
import TransferDialog from './TransferDialog.vue'
import StockDialog from './StockDialog.vue'

const { entries, isLoading, deleteEntry, isDeleting } = useEntries()
const { accounts } = useAccounts()

const expenseDialogVisible = ref(false)
const incomeDialogVisible = ref(false)
const transferDialogVisible = ref(false)
const stockDialogVisible = ref(false)
const isEditMode = ref(false)
const selectedEntry = ref(null)

const openNewEntryDialog = (type) => {
    isEditMode.value = false
    selectedEntry.value = null
    switch (type) {
        case 'expense':
            expenseDialogVisible.value = true
            break
        case 'income':
            incomeDialogVisible.value = true
            break
        case 'transfer':
            transferDialogVisible.value = true
            break
        case 'stock':
            stockDialogVisible.value = true
            break
    }
}

const openEditEntryDialog = (entry) => {
    isEditMode.value = true
    selectedEntry.value = entry
    switch (entry.type) {
        case 'expense':
            expenseDialogVisible.value = true
            break
        case 'income':
            incomeDialogVisible.value = true
            break
        case 'transfer':
            transferDialogVisible.value = true
            break
        case 'buystock':
        case 'sellstock':
            stockDialogVisible.value = true
            break
    }
}

const getEntryTypeIcon = (type) => {
    switch (type) {
        case 'expense':
            return 'pi pi-minus text-red-500'
        case 'income':
            return 'pi pi-plus text-green-500'
        case 'transfer':
            return 'pi pi-arrow-right-arrow-left text-blue-500'
        case 'buystock':
            return 'pi pi-chart-line text-yellow-500'
        case 'sellstock':
            return 'pi pi-chart-line text-orange-500'
        default:
            return 'pi pi-question-circle'
    }
}

const getAccountName = (accountId) => {
    const account = accounts.value?.find(acc => acc.id === accountId)
    return account ? account.name : 'Unknown Account'
}

const getRowClass = (data) => {
    return {
        'expense-row': data.type === 'expense',
        'income-row': data.type === 'income',
        'transfer-row': data.type === 'transfer',
        'buystock-row': data.type === 'buystock',
        'sellstock-row': data.type === 'sellstock'
    }
}

const handleDeleteEntry = async (entry) => {
    if (confirm('Are you sure you want to delete this entry?')) {
        try {
            await deleteEntry(entry.id)
        } catch (error) {
            console.error('Failed to delete entry:', error)
        }
    }
}
</script>

<template>
    <VerticalLayout :center-content="false" :fullHeight="true">
        <template #header>
            <TopBar />
        </template>
        <template #default>
            <HorizontalLayout
                :fullHeight="true"
                :centerContent="true"
                :verticalCenterContent="false"
            >
                <template #default>
                    <Placeholder :width="'960px'" :height="'auto'">
                        <div class="entries-view">
                            <div class="header">
                                <h1>Entries</h1>
                                <div class="action-buttons">
                                    <Button
                                        label="Add Expense"
                                        icon="pi pi-minus"
                                        severity="danger"
                                        @click="openNewEntryDialog('expense')"
                                    />
                                    <Button
                                        label="Add Income"
                                        icon="pi pi-plus"
                                        severity="success"
                                        @click="openNewEntryDialog('income')"
                                    />
                                    <Button
                                        label="Add Transfer"
                                        icon="pi pi-exchange"
                                        severity="info"
                                        @click="openNewEntryDialog('transfer')"
                                    />
                                    <Button
                                        label="Stock Operation"
                                        icon="pi pi-chart-line"
                                        severity="warning"
                                        @click="openNewEntryDialog('stock')"
                                    />
                                </div>
                            </div>

                            <DataTable
                                :value="entries"
                                :loading="isLoading"
                                stripedRows
                                paginator
                                :rows="50"
                                :rowsPerPageOptions="[5, 10, 20, 50]"
                                tableStyle="min-width: 50rem"
                                :rowClass="getRowClass"
                            >
                                <Column header="" style="width: 50px">
                                    <template #body="{ data }">
                                        <i :class="getEntryTypeIcon(data.type)" style="font-size: 1.2rem" />
                                    </template>
                                </Column>
                                <Column field="description" header="Description" />
                                <Column header="Account">
                                    <template #body="{ data }">
                                        <span v-if="data.type === 'transfer'">
                                            {{ getAccountName(data.originAccountId) }}<i class="pi pi-arrow-right" style="font-size: 0.9rem; margin: 0 8px;" />{{ getAccountName(data.targetAccountId) }}
                                        </span>
                                        <span v-else>
                                            {{ getAccountName(data.targetAccountId) }}
                                        </span>
                                    </template>
                                </Column>
                                <Column field="amount" header="Amount">
                                    <template #body="{ data }">
                                      <div v-if="data.type === 'expense'">
                                        -{{ data.amount.toFixed(2) }}
                                      </div>
                                      <div v-else>
                                        {{ data.amount.toFixed(2) }}
                                      </div>

                                    </template>
                                </Column>
                                <Column field="date" header="Date">
                                    <template #body="{ data }">
                                        {{ new Date(data.date).toLocaleDateString() }}
                                    </template>
                                </Column>
                                <Column header="Actions" style="width: 100px">
                                    <template #body="{ data }">
                                        <div class="actions">
                                            <Button
                                                icon="pi pi-pencil"
                                                text
                                                rounded
                                                class="action-button"
                                                @click="openEditEntryDialog(data)"
                                            />
                                            <Button
                                                icon="pi pi-trash"
                                                text
                                                rounded
                                                severity="danger"
                                                class="action-button"
                                                :loading="isDeleting"
                                                @click="handleDeleteEntry(data)"
                                            />
                                        </div>
                                    </template>
                                </Column>
                            </DataTable>

                            <ExpenseDialog
                                v-model:visible="expenseDialogVisible"
                                :isEdit="isEditMode"
                                :entryId="selectedEntry?.id"
                                :description="selectedEntry?.description"
                                :amount="selectedEntry?.amount"
                                :date="selectedEntry?.date"
                                :targetAccountId="selectedEntry?.targetAccountId"
                                :categoryId="selectedEntry?.categoryId"
                            />

                            <IncomeDialog
                                v-model:visible="incomeDialogVisible"
                                :isEdit="isEditMode"
                                :entryId="selectedEntry?.id"
                                :description="selectedEntry?.description"
                                :amount="selectedEntry?.amount"
                                :date="selectedEntry?.date"
                                :targetAccountId="selectedEntry?.targetAccountId"
                                :categoryId="selectedEntry?.categoryId"
                            />

                            <TransferDialog
                                v-model:visible="transferDialogVisible"
                                :isEdit="isEditMode"
                                :entryId="selectedEntry?.id"
                                :description="selectedEntry?.description"
                                :amount="selectedEntry?.amount"
                                :date="selectedEntry?.date"
                                :targetAccountId="selectedEntry?.targetAccountId"
                                :originAccountId="selectedEntry?.originAccountId"
                                :categoryId="selectedEntry?.categoryId"
                            />

                            <StockDialog
                                v-model:visible="stockDialogVisible"
                                :isEdit="isEditMode"
                                :entryId="selectedEntry?.id"
                                :description="selectedEntry?.description"
                                :amount="selectedEntry?.amount"
                                :stockAmount="selectedEntry?.stockAmount"
                                :date="selectedEntry?.date"
                                :type="selectedEntry?.type"
                                :targetAccountId="selectedEntry?.targetAccountId"
                                :originAccountId="selectedEntry?.originAccountId"
                                :categoryId="selectedEntry?.categoryId"
                            />
                        </div>
                    </Placeholder>
                </template>
            </HorizontalLayout>
        </template>
        <template #footer>
            <Placeholder :width="'100%'" :height="30" :color="12">Footer</Placeholder>
        </template>
    </VerticalLayout>
</template>

<style scoped>

.header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 2rem;
}

.action-buttons {
    display: flex;
    gap: 0.5rem;
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

:deep(.p-datatable .p-datatable-tbody > tr.expense-row) {
    background-color: rgba(190, 45, 45, 0.05);
}

:deep(.p-datatable .p-datatable-tbody > tr.income-row) {
    background-color: rgba(34, 197, 94, 0.05);
}

:deep(.p-datatable .p-datatable-tbody > tr.transfer-row) {
    background-color: rgba(59, 130, 246, 0.05);
}

:deep(.p-datatable .p-datatable-tbody > tr.buystock-row) {
    background-color: rgba(234, 179, 8, 0.05);
}

:deep(.p-datatable .p-datatable-tbody > tr.sellstock-row) {
    background-color: rgba(249, 115, 22, 0.05);
}

:deep(.p-datatable .p-datatable-tbody > tr:hover) {
    background-color: rgba(0, 0, 0, 0.05) !important;
}
</style> 