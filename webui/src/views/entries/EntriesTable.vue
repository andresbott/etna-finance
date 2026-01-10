<script setup>
import { computed } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Card from 'primevue/card'
import { useCategoryUtils } from '@/utils/categoryUtils'
import { useAccountUtils } from '@/utils/accountUtils'

/* --- Props --- */
const props = defineProps({
    entries: {
        type: Array,
        required: true
    },
    isLoading: {
        type: Boolean,
        default: false
    },
    isDeleting: {
        type: Boolean,
        default: false
    },
    totalRecords: {
        type: Number,
        default: 0
    },
    rows: {
        type: Number,
        default: 25
    },
    first: {
        type: Number,
        default: 0
    }
})

/* --- Emits --- */
const emit = defineEmits(['edit', 'duplicate', 'delete', 'page'])

/* --- Utils --- */
const { getCategoryName, getCategoryPath } = useCategoryUtils()
const { getAccountCurrency, getAccountName } = useAccountUtils()

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

/* --- Event Handlers --- */
const handleEdit = (entry) => {
    emit('edit', entry)
}

const handleDuplicate = (entry) => {
    emit('duplicate', entry)
}

const handleDelete = (entry) => {
    emit('delete', entry)
}

const handlePage = (event) => {
    emit('page', event)
}
</script>

<template>
    <Card>
        <template #content>
            <DataTable
                :value="entries"
                :loading="isLoading"
                stripedRows
                paginator
                lazy
                style="width: 100%"
                :rows="rows"
                :first="first"
                :totalRecords="totalRecords"
                :rowsPerPageOptions="[25, 50, 100]"
                :rowClass="getRowClass"
                @page="handlePage"
            >
                <Column header="" style="width: 40px">
                    <template #body="{ data }">
                        <i :class="getEntryTypeIcon(data.type)" style="font-size: 0.8rem" />
                    </template>
                </Column>

                <Column field="description" header="Description" class="description-column">
                    <template #body="{ data }">
                        <span 
                            v-if="data.type === 'expense' || data.type === 'income'"
                            v-tooltip.bottom="`Category: ${getCategoryPath(data?.categoryId, data.type)}`"
                        >
                            {{ data.description }}
                        </span>
                        <span v-else>{{ data.description }}</span>
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

                <Column field="Amount" header="Amount" bodyStyle="text-align: right" class="amount-column">
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
                            {{ getAccountCurrency(data.originAccountId) }}
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
                            {{ getAccountCurrency(data.targetAccountId) }}
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
                                @click="handleEdit(data)"
                                v-tooltip.bottom="'Edit'"
                            />
                            <Button
                                icon="pi pi-copy"
                                text
                                rounded
                                class="action-button"
                                @click="handleDuplicate(data)"
                                v-tooltip.bottom="'Duplicate'"
                            />
                            <Button
                                icon="pi pi-trash"
                                text
                                rounded
                                severity="danger"
                                class="action-button"
                                :loading="isDeleting"
                                @click="handleDelete(data)"
                                v-tooltip.bottom="'Delete'"
                            />
                        </div>
                    </template>
                </Column>
            </DataTable>
        </template>
    </Card>
</template>

<style scoped>
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

:deep(.amount-column .p-datatable-column-title) {
    margin-left: auto;
}
</style>

