<script setup>
import { ref, computed } from 'vue'
import {
  VerticalLayout,
  HorizontalLayout,
  Placeholder,
  ResponsiveHorizontal
} from '@go-bumbu/vue-components/layout'
import '@go-bumbu/vue-components/layout.css'

import TopBar from '@/views/topbar.vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Menu from 'primevue/menu'
import Listbox from 'primevue/listbox'
import DatePicker from 'primevue/datepicker'

import ExpenseDialog from './ExpenseDialog.vue'
import IncomeDialog from './IncomeDialog.vue'
import TransferDialog from './TransferDialog.vue'
import StockDialog from './StockDialog.vue'

import { useEntries } from '@/composables/useEntries.js'
import { useAccounts } from '@/composables/useAccounts.js'

/* --- Reactive State --- */
const today = new Date()
const startDate = ref(new Date(today.setDate(today.getDate() - 35)))
const endDate = ref(new Date())

const { entries, isLoading, deleteEntry, isDeleting, refetch } = useEntries(startDate, endDate)
const { accounts } = useAccounts()

const selectedAccount = ref(null)
const accountSearch = ref('')
const leftSidebarCollapsed = ref(true)
const menu = ref(null)

const selectedEntry = ref(null)
const isEditMode = ref(false)

/* --- Dialog Visibility State --- */
const dialogs = {
  expense: ref(false),
  income: ref(false),
  transfer: ref(false),
  stock: ref(false)
}

/* --- Filtering Logic --- */
const filteredAccounts = computed(() =>
    !accountSearch.value
        ? accounts.value
        : accounts.value?.filter(account =>
            account.name.toLowerCase().includes(accountSearch.value.toLowerCase())
        )
)

const filteredEntries = computed(() =>
    !selectedAccount.value
        ? entries.value
        : entries.value?.filter(
            entry =>
                entry.targetAccountId === selectedAccount.value.id ||
                entry.originAccountId === selectedAccount.value.id
        )
)

/* --- Category Data --- */
const incomeCategories = [
  { id: 1, name: 'Salary' },
  { id: 2, name: 'Freelance' },
  { id: 3, name: 'Investments' },
  { id: 4, name: 'Other Income' }
]

const expenseCategories = [
  { id: 1, name: 'Food' },
  { id: 2, name: 'Transport' },
  { id: 3, name: 'Housing' },
  { id: 4, name: 'Entertainment' },
  { id: 5, name: 'Utilities' },
  { id: 6, name: 'Shopping' }
]

/* --- Menu Actions --- */
const openNewEntryDialog = (type) => {
  isEditMode.value = false
  selectedEntry.value = null
  dialogs[type].value = true
}

const menuItems = ref([
  { label: 'Add Expense', icon: 'pi pi-minus', command: () => openNewEntryDialog('expense') },
  { label: 'Add Income', icon: 'pi pi-plus', command: () => openNewEntryDialog('income') },
  { label: 'Add Transfer', icon: 'pi pi-arrow-right-arrow-left', command: () => openNewEntryDialog('transfer') },
  { label: 'CSV import', icon: 'pi pi-bolt', command: () => openNewEntryDialog('transfer') },
  { label: 'Stock Operation', icon: 'pi pi-chart-line', command: () => openNewEntryDialog('stock') }
])

/* --- Entry Actions --- */
const openEditEntryDialog = (entry) => {
  isEditMode.value = true
  selectedEntry.value = entry
  if (entry.type === 'expense' || entry.type === 'income' || entry.type === 'transfer') {
    dialogs[entry.type].value = true
  } else if (entry.type === 'buystock' || entry.type === 'sellstock') {
    dialogs.stock.value = true
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
            <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
                <template #left>
                    <div class="left-sidebar-content">
                        <div class="filter-section">
                            <h3>Filter by Account</h3>
                            <Listbox
                                v-model="selectedAccount"
                                :options="filteredAccounts"
                                optionLabel="name"
                                class="w-full"
                                listStyle="max-height: 200px"
                            />
                        </div>

                        <div class="categories-section">
                            <h3>Income Categories</h3>
                            <div class="category-list">
                                <div
                                    v-for="category in incomeCategories"
                                    :key="category.id"
                                    class="category-item"
                                >
                                    <i
                                        class="pi pi-circle-fill"
                                        style="color: var(--green-500)"
                                    ></i>
                                    <span>{{ category.name }}</span>
                                </div>
                            </div>

                            <h3>Expense Categories</h3>
                            <div class="category-list">
                                <div
                                    v-for="category in expenseCategories"
                                    :key="category.id"
                                    class="category-item"
                                >
                                    <i class="pi pi-circle-fill" style="color: var(--red-500)"></i>
                                    <span>{{ category.name }}</span>
                                </div>
                            </div>
                        </div>
                    </div>
                </template>

                <template #default>
                    <div class="sidebar-controls">
                        <Button
                            icon="pi pi-chevron-left"
                            text
                            rounded
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
                            <Button
                                label=""
                                icon="pi pi-plus"
                                @click="menu.toggle($event)"
                                aria-haspopup="true"
                                aria-controls="overlay_menu"
                            />
                            <Menu ref="menu" :model="menuItems" :popup="true" id="overlay_menu" />
                        </div>
                    </div>

                    <div class="entries-view">
                        <DataTable
                            :value="filteredEntries"
                            :loading="isLoading"
                            stripedRows
                            paginator
                            style="width: 100%"
                            :rows="50"
                            :rowsPerPageOptions="[5, 10, 20, 50]"
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
                            <Column field="targetAmount" header="Amount">
                                <template #body="{ data }">
                                    <div v-if="data.type === 'expense'" class="amount expense">
                                        -{{
                                            data.targetAmount.toLocaleString('es-ES', {
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
                </template>
            </ResponsiveHorizontal>
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

:deep(.p-datatable .p-datatable-tbody > tr:hover) {
    background-color: rgba(0, 0, 0, 0.1) !important;
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

.left-sidebar-content {
    padding: 1rem;
}

.filter-section {
    margin-bottom: 2rem;
}

.filter-section h3 {
    margin-bottom: 0.5rem;
    font-size: 1rem;
    color: var(--text-color-secondary);
}

.categories-section {
    margin-top: 2rem;
}

.categories-section h3 {
    margin-bottom: 1rem;
    font-size: 1rem;
    color: var(--text-color-secondary);
}

.category-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
}

.category-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.25rem 0;
}

.category-item i {
    font-size: 0.75rem;
}

:deep(.p-listbox) {
    border: none;
    background: transparent;
}

:deep(.p-listbox .p-listbox-list) {
    padding: 0;
}

:deep(.p-listbox .p-listbox-item) {
    padding: 0.5rem;
    border-radius: 4px;
}

:deep(.p-listbox .p-listbox-item.p-highlight) {
    background: var(--primary-color);
    color: var(--primary-color-text);
}

.search-box {
    margin-bottom: 0.5rem;
}

:deep(.p-inputtext) {
    width: 100%;
    padding: 0.5rem;
}

:deep(.p-input-icon-left > i:first-of-type) {
    left: 0.5rem;
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
