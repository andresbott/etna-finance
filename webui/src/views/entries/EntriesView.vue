<script setup>
import { ref, computed, watch } from 'vue'
import {
    VerticalLayout,
    Placeholder,
    SidebarContent,
} from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'

import TopBar from '@/views/topbar.vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Tree from 'primevue/tree'
import DatePicker from 'primevue/datepicker'
import TransferDialog from './dialogs/TransferDialog.vue'
import StockDialog from './dialogs/StockDialog.vue'
import DeleteDialog from '@/components/common/confirmDialog.vue'

import { useEntries } from '@/composables/useEntries.ts'
import { useAccounts } from '@/composables/useAccounts.js'
import IncomeExpenseDialog from '@/views/entries/dialogs/IncomeExpenseDialog.vue'
import AddEntryMenu from '@/views/entries/AddEntryMenu.vue'

/* --- Reactive State --- */
const today = new Date()
const startDate = ref(new Date(today.setDate(today.getDate() - 35)))
const endDate = ref(new Date())

// Account filtering state
const selectedAccountIds = ref([])
const selectedKeys = ref({}) // For Tree component selection
const { entries, isLoading, deleteEntry, isDeleting, refetch } = useEntries(
    startDate,
    endDate,
    selectedAccountIds
)
const { accounts, isLoading: isAccountsLoading } = useAccounts()

const leftSidebarCollapsed = ref(true)
const menu = ref(null)

const selectedEntry = ref(null)
const isEditMode = ref(false)

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

const expenseDialog = ref(false)

/* --- Account Tree Data --- */
const accountsTree = computed(() => {
    if (!accounts.value) return []

    return accounts.value.map((provider) => ({
        key: `provider-${provider.id}`,
        label: provider.name,
        selectable: true,
        children: provider.accounts.map((account) => ({
            key: account.id.toString(),
            label: `${account.name} (${account.currency})`,
            data: {
                id: account.id,
                provider: provider.name,
                account: account
            }
        }))
    }))
})

// Keep all provider nodes expanded by default
const expandedKeys = ref({})

// Initialize expanded keys when accounts are loaded
watch(
    () => accounts.value,
    (newAccounts) => {
        if (newAccounts) {
            const keys = {}
            newAccounts.forEach((provider) => {
                keys[`provider-${provider.id}`] = true
            })
            expandedKeys.value = keys
        }
    },
    { immediate: true }
)

// Handle account selection change - convert selectionKeys to accountIds
watch(
    () => selectedKeys.value,
    (newSelection) => {
        if (!newSelection) {
            selectedAccountIds.value = []
            return
        }

        const ids = []
        const providerIds = []

        // Process selected keys
        Object.keys(newSelection).forEach((key) => {
            if (key.startsWith('provider-')) {
                // Store provider ID for later processing
                providerIds.push(key.replace('provider-', ''))
            } else if (!isNaN(parseInt(key))) {
                // Direct account selection
                ids.push(parseInt(key))
            }
        })

        // Add all accounts from selected providers
        if (providerIds.length > 0 && accounts.value) {
            accounts.value.forEach((provider) => {
                if (providerIds.includes(provider.id.toString())) {
                    provider.accounts.forEach((account) => {
                        if (!ids.includes(account.id)) {
                            ids.push(account.id)
                        }
                    })
                }
            })
        }

        selectedAccountIds.value = ids

        // Trigger refetch when selection changes
        if (ids.length > 0) {
            refetch()
        }
    },
    { deep: true }
)

// Clear selection handler
const clearSelection = () => {
    selectedKeys.value = {}
    selectedAccountIds.value = []
    refetch()
}

/* --- Menu Actions --- */
const openNewEntryDialog = (type) => {
    isEditMode.value = false
    selectedEntry.value = null

    // Handle IncomeExpense dialog for income and expense types
    if (type === 'income' || type === 'expense') {
        selectedEntry.value = { type }
        dialogs.incomeExpense.value = true
    } else {
        dialogs[type].value = true
    }
}

const menuItems = ref([
    { label: 'Add Expense', icon: 'pi pi-minus', command: () => openNewEntryDialog('expense') },
    { label: 'Add Income', icon: 'pi pi-plus', command: () => openNewEntryDialog('income') },
    {
        label: 'Add Transfer',
        icon: 'pi pi-arrow-right-arrow-left',
        command: () => openNewEntryDialog('transfer')
    },
    { label: 'CSV import', icon: 'pi pi-bolt', command: () => openNewEntryDialog('transfer') },
    {
        label: 'Stock Operation',
        icon: 'pi pi-chart-line',
        command: () => openNewEntryDialog('stock')
    }
])

/* --- Entry Actions --- */
const openEditEntryDialog = (entry) => {
    isEditMode.value = true
    selectedEntry.value = entry

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
    <VerticalLayout :center-content="false" :fullHeight="true">
        <template #header>
            <TopBar />
        </template>
        <template #default>
            <div class="main-app-content">
                <SidebarContent  :leftSidebarCollapsed="leftSidebarCollapsed" :rightSidebarCollapsed="true">
                    <template #left>
                        <div class="left-sidebar-content">
                            <div class="filter-section">
                                <div class="filter-header">
                                    <h3>Filter by Accounts</h3>
                                    <Button
                                        icon="pi pi-times"
                                        text
                                        size="small"
                                        @click="clearSelection"
                                        v-if="Object.keys(selectedKeys).length > 0"
                                        class="clear-button"
                                    />
                                </div>

                                <div class="tree-container">
                                    <Tree
                                        :value="accountsTree"
                                        v-model:selectionKeys="selectedKeys"
                                        :expandedKeys="expandedKeys"
                                        v-model:expandedKeys="expandedKeys"
                                        selectionMode="multiple"
                                        :loading="isAccountsLoading"
                                        class="w-full mb-3 account-tree"
                                    >
                                        <template #empty>
                                            <div class="p-2" v-if="isAccountsLoading">
                                                Loading accounts...
                                            </div>
                                            <div class="p-2" v-else>No accounts found</div>
                                        </template>
                                    </Tree>
                                </div>

                                <div class="filter-actions">
                                <span class="selected-count" v-if="selectedAccountIds.length > 0">
                                    {{ selectedAccountIds.length }} account(s) selected
                                </span>
                                </div>
                            </div>
                        </div>
                    </template>

                    <template #default>
                        <div class="sidebar-controls">
                            <Button
                                icon="pi pi-chevron-left"
                                @click="leftSidebarCollapsed = !leftSidebarCollapsed"
                                :class="{ 'rotate-180': leftSidebarCollapsed }"
                            />
                            <div class="date-filters">
                                <div class="date-field">
                                    <label>From:</label>
                                    <DatePicker
                                        v-model="startDate"
                                        :showIcon="true"
                                        :showButtonBar="true"
                                        dateFormat="dd/mm/y"
                                        placeholder="Start date"
                                        @date-select="refetch"
                                    />
                                </div>
                                <div class="date-field">
                                    <label>To:</label>
                                    <DatePicker
                                        v-model="endDate"
                                        :showIcon="true"
                                        :showButtonBar="true"
                                        dateFormat="dd/mm/y"
                                        placeholder="End date"
                                        @date-select="refetch"
                                    />
                                </div>
                            </div>
                            <div class="add-entry-menu">
                                <AddEntryMenu />
                            </div>
                        </div>

                        <div class="entries-view">
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
                                        <i
                                            :class="getEntryTypeIcon(data.type)"
                                            style="font-size: 0.8rem"
                                        />
                                    </template>
                                </Column>

                                <Column field="description" header="Description" />

                                <Column header="Account">
                                    <template #body="{ data }">
                                    <span v-if="data.type === 'transfer'">
                                        {{ data.originAccountName
                                        }}<i
                                        class="pi pi-arrow-right"
                                        style="font-size: 0.9rem; margin: 0 8px"
                                    />{{ data.targetAccountName }}
                                    </span>
                                        <span v-else>
                                        {{ data.targetAccountName }}
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
                                            {{ data.targetAccountCurrency || '' }}
                                        </div>
                                        <div v-else-if="data.type === 'income'" class="amount income">
                                            {{
                                                data.targetAmount.toLocaleString('es-ES', {
                                                    minimumFractionDigits: 2,
                                                    maximumFractionDigits: 2
                                                })
                                            }}
                                            {{ data.targetAccountCurrency || '' }}
                                        </div>
                                        <div
                                            v-else-if="data.type === 'transfer'"
                                            class="amount transfer"
                                        >
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
                                                @click="openDeleteDialog(data)"
                                            />
                                        </div>
                                    </template>
                                </Column>
                            </DataTable>
                        </div>
                    </template>
                </SidebarContent>
            </div>

        </template>
        <template #footer>
            <Placeholder :width="'100%'" :height="30" :color="12">Footer</Placeholder>
        </template>
    </VerticalLayout>

    <!-- Dialog Components -->
    <IncomeExpenseDialog
        v-model:visible="dialogs.incomeExpense.value"
        :is-edit="isEditMode"
        :entry-type="selectedEntry?.type"
        :description="selectedEntry?.description"
        :target-amount="selectedEntry?.targetAmount"
        :target-account-id="selectedEntry?.targetAccountId"
        :stock-amount="selectedEntry?.targetStockAmount"
        :date="selectedEntry?.date ? new Date(selectedEntry.date) : new Date()"
        :entry-id="selectedEntry?.id"
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
        :date="selectedEntry?.date ? new Date(selectedEntry.date) : new Date()"
        :target-account-id="selectedEntry?.targetAccountId"
        :origin-account-id="selectedEntry?.originAccountId"
    />

    <StockDialog
        v-model:visible="dialogs.stock.value"
        :is-edit="isEditMode"
        :entry-id="selectedEntry?.id"
        :description="selectedEntry?.description"
        :amount="selectedEntry?.targetAmount"
        :stock-amount="selectedEntry?.targetStockAmount"
        :date="selectedEntry?.date ? new Date(selectedEntry.date) : new Date()"
        :type="selectedEntry?.type"
        :target-account-id="selectedEntry?.targetAccountId"
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


.header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 2rem;
}

.left-sidebar-content {
    padding: 1rem;
}

.filter-section {
    margin-bottom: 2rem;
}

.filter-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.5rem;
}

.filter-header h3 {
    margin: 0;
    font-size: 1rem;
    color: var(--text-color-secondary);
}

.tree-container {
    border: 1px solid var(--surface-border);
    border-radius: 6px;
    background-color: var(--surface-card);
    overflow-y: auto;
}

.filter-actions {
    margin-top: 0.5rem;
}

.selected-count {
    display: block;
    font-size: 0.85rem;
    color: var(--text-color-secondary);
    margin-top: 0.5rem;
    text-align: center;
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

:deep(.p-datatable .p-datatable-tbody > tr:hover) {
    background-color: rgba(0, 0, 0, 0.1) !important;
}

:deep(.account-tree .p-tree-container) {
    padding: 0.5rem;
}

:deep(.account-tree .p-treenode-content) {
    padding: 0.3rem;
}

:deep(.account-tree .p-treenode-content:hover) {
    background-color: var(--surface-hover);
}

.sidebar-controls {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.5rem;
    position: sticky;
    top: 0;
    z-index: 1;
    background-color: var(--surface-ground);
}

.date-filters {
    display: flex;
    gap: 1rem;
    align-items: center;
}

.date-field {
    display: flex;
    align-items: center;
    gap: 0.5rem;
}

.date-field label {
    white-space: nowrap;
}

.rotate-180 {
    transform: rotate(180deg);
}

.add-entry-menu {
    display: flex;
    justify-content: center;
    padding: 1rem;
}

.amount.expense {
    color: #dc2626;
}

.amount.income {
    color: #16a34a;
}

.amount.transfer {
    color: #2563eb;
}
</style>
