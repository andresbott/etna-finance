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
    accountId: {
        type: [String, Number],
        required: true
    }
})

/* --- Emits --- */
const emit = defineEmits(['edit', 'duplicate', 'delete'])

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
        sellstock: 'pi pi-chart-line text-orange-500',
        'opening-balance': 'pi pi-calculator text-gray-500'
    }
    return icons[type] || 'pi pi-question-circle'
}

const getRowClass = (data) => ({
    'expense-row': data.type === 'expense',
    'income-row': data.type === 'income',
    'transfer-row': data.type === 'transfer',
    'buystock-row': data.type === 'buystock',
    'sellstock-row': data.type === 'sellstock',
    'opening-balance-row': data.type === 'opening-balance'
})

/* --- Balance Calculation --- */
const entriesWithBalance = computed(() => {
    if (!props.entries || props.entries.length === 0) return []
    
    // Find the opening balance entry to get the starting balance
    const openingBalanceEntry = props.entries.find(entry => entry.type === 'opening-balance')
    let balance = openingBalanceEntry?.Amount || 0
    
    // Create a copy and reverse to process in chronological order (oldest to newest)
    const entriesReversed = [...props.entries].reverse()
    
    // Calculate running balance
    const entriesWithBalanceReversed = entriesReversed.map(entry => {
        let entryAmount = 0
        
        // Calculate the amount that affects this account's balance
        if (entry.type === 'opening-balance') {
            // Opening balance is the starting point, no change needed
            entryAmount = 0
        } else if (entry.type === 'expense') {
            entryAmount = -(entry.Amount || 0)
        } else if (entry.type === 'income') {
            entryAmount = entry.Amount || 0
        } else if (entry.type === 'transfer') {
            // For transfers, check if this account is origin or target
            if (String(entry.originAccountId) === String(props.accountId)) {
                entryAmount = -(entry.originAmount || 0)
            } else if (String(entry.targetAccountId) === String(props.accountId)) {
                entryAmount = entry.targetAmount || 0
            }
        } else if (entry.type === 'buystock') {
            entryAmount = -(entry.targetAmount || 0)
        } else if (entry.type === 'sellstock') {
            entryAmount = entry.targetAmount || 0
        }
        
        // For opening balance, don't add the amount (it's already the starting balance)
        if (entry.type !== 'opening-balance') {
            balance += entryAmount
        }
        
        return {
            ...entry,
            balance: balance
        }
    })
    
    // Reverse back to show newest first
    return entriesWithBalanceReversed.reverse()
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
</script>

<template>
    <Card>
        <template #content>
            <DataTable
                :value="entriesWithBalance"
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

                <Column field="description" header="Description" class="description-column">
                    <template #body="{ data }">
                        <span 
                            v-if="data.type === 'expense' || data.type === 'income'"
                            v-tooltip.bottom="`Category: ${getCategoryPath(data?.categoryId, data.type)}`"
                        >
                            {{ data.description }}
                        </span>
                        <span 
                            v-else-if="data.type === 'transfer'"
                            v-tooltip.bottom="`${getAccountName(data.originAccountId)}: ${data.originAmount?.toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} ${getAccountCurrency(data.originAccountId)} → ${getAccountName(data.targetAccountId)}: ${data.targetAmount.toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} ${getAccountCurrency(data.targetAccountId)}`"
                        >
                            {{ data.description }}
                        </span>
                        <span v-else>{{ data.description }}</span>
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
                        <div v-if="data.type === 'opening-balance'" class="amount opening-balance">
                            <!-- Opening balance amount is blank, since it's shown in Balance column -->
                        </div>
                        <div v-else-if="data.type === 'expense'" class="amount expense">
                            -{{
                                data.Amount.toLocaleString('es-ES', {
                                    minimumFractionDigits: 2,
                                    maximumFractionDigits: 2
                                })
                            }}
                        </div>
                        <div v-else-if="data.type === 'income'" class="amount income">
                            {{
                                data.Amount.toLocaleString('es-ES', {
                                    minimumFractionDigits: 2,
                                    maximumFractionDigits: 2
                                })
                            }}
                        </div>
                        <div v-else-if="data.type === 'transfer'" class="amount transfer">
                            <!-- Incoming transfer (this account is the target) -->
                            <template v-if="String(data.targetAccountId) === String(accountId)">
                                +{{
                                    data.targetAmount.toLocaleString('es-ES', {
                                        minimumFractionDigits: 2,
                                        maximumFractionDigits: 2
                                    })
                                }}
                            </template>
                            <!-- Outgoing transfer (this account is the origin) -->
                            <template v-else>
                                -{{
                                    data.originAmount?.toLocaleString('es-ES', {
                                        minimumFractionDigits: 2,
                                        maximumFractionDigits: 2
                                    }) || '0.00'
                                }}
                            </template>
                        </div>
                        <div v-else class="amount">
                            {{
                                data.targetAmount.toLocaleString('es-ES', {
                                    minimumFractionDigits: 2,
                                    maximumFractionDigits: 2
                                })
                            }}
                        </div>
                    </template>
                </Column>

                <Column field="balance" header="Balance" bodyStyle="text-align: right" class="balance-column">
                    <template #body="{ data }">
                        <div class="balance" :class="{ 'balance-negative': data.balance < 0 }">
                            {{
                                data.balance.toLocaleString('es-ES', {
                                    minimumFractionDigits: 2,
                                    maximumFractionDigits: 2
                                })
                            }}
                        </div>
                    </template>
                </Column>

                <Column header="Actions" style="width: 150px">
                    <template #body="{ data }">
                        <!-- No actions for opening balance entry, but add padding to match height -->
                        <div v-if="data.type === 'opening-balance'" class="actions-placeholder"></div>
                        <div v-else class="actions">
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

.actions-placeholder {
    height: 2.5rem;
    display: flex;
    align-items: center;
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

.balance-negative {
    color: var(--c-red-600);
}

:deep(.amount-column .p-datatable-column-title),
:deep(.balance-column .p-datatable-column-title) {
    margin-left: auto;
}
</style>

