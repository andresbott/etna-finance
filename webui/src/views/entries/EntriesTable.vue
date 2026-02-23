<script setup>
import { computed } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Card from 'primevue/card'
import { useCategoryUtils } from '@/utils/categoryUtils'
import { useAccountUtils } from '@/utils/accountUtils'
import { useInstruments } from '@/composables/useInstruments'
import { useDateFormat } from '@/composables/useDateFormat'
import { getEntryTypeIcon } from '@/utils/entryDisplay'

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
const { instruments: instrumentsData } = useInstruments()
const { formatDate } = useDateFormat()

const instrumentsMap = computed(() => {
    const list = instrumentsData.value ?? []
    return Object.fromEntries(list.map((i) => [i.id, i]))
})

const getInstrument = (instrumentId) => instrumentsMap.value[instrumentId]
const getInstrumentSymbol = (instrumentId) => getInstrument(instrumentId)?.symbol ?? '—'
const getInstrumentCurrency = (instrumentId) => getInstrument(instrumentId)?.currency ?? ''

/** True when this row is a stock sell (in the stock-trade block we only have buy or sell, so not-buy => sell) */
const isStockSell = (data) => {
    const t = data.type ?? data.Type
    if (t == null) return false
    return String(t).toLowerCase() !== 'stockbuy'
}

/** Amount to show for stock trade total: negative for buy (cash out), positive for sell (cash in). */
const stockTradeTotalAmount = (data) => (isStockSell(data) ? (data.totalAmount || 0) : -(data.totalAmount || 0))

/* --- Helpers --- */
const getRowClass = (data) => ({
    'expense-row': data.type === 'expense',
    'income-row': data.type === 'income',
    'transfer-row': data.type === 'transfer',
    'stockbuy-row': data.type === 'stockbuy',
    'stocksell-row': data.type === 'stocksell',
    'stockgrant-row': data.type === 'stockgrant',
    'stocktransfer-row': data.type === 'stocktransfer'
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
                class="datatable-compact"
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
                        <span v-else-if="data.type === 'stockbuy' || data.type === 'stocksell'">
                            {{ getAccountName(data.cashAccountId)
                            }}<i
                                class="pi pi-arrow-right"
                                style="font-size: 0.9rem; margin: 0 8px"
                            />{{ getAccountName(data.investmentAccountId) }}
                        </span>
                        <span v-else-if="data.type === 'stocktransfer'">
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
                        {{ formatDate(data.date) }}
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
                            +{{
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
                        <div v-else-if="data.type === 'stockbuy' || data.type === 'stocksell'" class="amount stock-trade">
                            <template v-if="data.quantity && data.StockAmount != null">
                                ({{ getInstrumentSymbol(data.instrumentId) }}) {{ data.quantity }}
                                @ {{ (data.StockAmount / data.quantity).toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) }}
                                {{ getInstrumentCurrency(data.instrumentId) }}
                                <i
                                    class="pi pi-arrow-right"
                                    style="font-size: 0.9rem; margin: 0 8px"
                                />
                                <span :class="isStockSell(data) ? 'stock-trade-total sell' : 'stock-trade-total buy'">{{ isStockSell(data) ? '+' : '' }}{{ stockTradeTotalAmount(data).toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) }} {{ getAccountCurrency(data.cashAccountId) }}</span>
                            </template>
                            <template v-else>—</template>
                        </div>
                        <div v-else-if="data.type === 'stockgrant'" class="amount stock-trade">
                            <template v-if="data.quantity != null && data.instrumentId != null">
                                ({{ getInstrumentSymbol(data.instrumentId) }}) {{ data.quantity }}
                                @ {{ (0).toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) }} {{ getInstrumentCurrency(data.instrumentId) }}
                            </template>
                            <template v-else>—</template>
                        </div>
                        <div v-else-if="data.type === 'stocktransfer'" class="amount">
                            {{ data.quantity }} (transfer)
                        </div>
                        <div v-else class="amount">
                            {{
                                (data.totalAmount || data.targetAmount || 0).toLocaleString('es-ES', {
                                    minimumFractionDigits: 2,
                                    maximumFractionDigits: 2
                                })
                            }}
                        </div>
                    </template>
                </Column>

                <Column header="Actions" style="width: 150px">
                    <template #body="{ data }">
                        <div class="flex gap-2 justify-content-start">
                            <Button
                                icon="pi pi-pencil"
                                text
                                rounded
                                class="p-1"
                                @click="handleEdit(data)"
                                v-tooltip.bottom="'Edit'"
                            />
                            <Button
                                icon="pi pi-copy"
                                text
                                rounded
                                class="p-1"
                                @click="handleDuplicate(data)"
                                v-tooltip.bottom="'Duplicate'"
                            />
                            <Button
                                icon="pi pi-trash"
                                text
                                rounded
                                severity="danger"
                                class="p-1"
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

