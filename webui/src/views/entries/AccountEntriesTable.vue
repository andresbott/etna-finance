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
import { ACCOUNT_TYPES } from '@/types/account'

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
    },
    accountType: {
        type: String,
        default: null
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

const isInstrumentAccount = computed(
    () =>
        props.accountType === ACCOUNT_TYPES.INVESTMENT ||
        props.accountType === ACCOUNT_TYPES.UNVESTED
)

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
const getInstrumentSymbol = (instrumentId) => instrumentsMap.value[instrumentId]?.symbol ?? '—'
const getInstrumentCurrency = (instrumentId) => instrumentsMap.value[instrumentId]?.currency ?? ''
const formatPrice = (n) => (n != null && !Number.isNaN(n) ? n.toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) : '—')
const formatAmount = (n) => (n != null && !Number.isNaN(n) ? n.toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) : '0.00')

/* --- Helpers --- */
const getRowClass = (data) => ({
    'expense-row': data.type === 'expense',
    'income-row': data.type === 'income',
    'transfer-row': data.type === 'transfer',
    'stockbuy-row': data.type === 'stockbuy',
    'stocksell-row': data.type === 'stocksell',
    'stockgrant-row': data.type === 'stockgrant',
    'stocktransfer-row': data.type === 'stocktransfer',
    'opening-balance-row': data.type === 'opening-balance'
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

/* --- Balance (cash accounts only) --- */
const entriesWithBalance = computed(() => {
    if (!props.entries || props.entries.length === 0 || isInstrumentAccount.value) return props.entries ?? []
    const openingBalanceEntry = props.entries.find((e) => e.type === 'opening-balance')
    let balance = openingBalanceEntry?.Amount || 0
    const entriesReversed = [...props.entries].reverse()
    const result = entriesReversed.map((entry) => {
        let entryAmount = 0
        if (entry.type === 'opening-balance') entryAmount = 0
        else if (entry.type === 'expense') entryAmount = -(entry.Amount || 0)
        else if (entry.type === 'income') entryAmount = entry.Amount || 0
        else if (entry.type === 'transfer') {
            if (String(entry.originAccountId) === String(props.accountId)) entryAmount = -(entry.originAmount || 0)
            else if (String(entry.targetAccountId) === String(props.accountId)) entryAmount = entry.targetAmount || 0
        } else if (entry.type === 'stockbuy') {
            if (String(entry.cashAccountId) === String(props.accountId)) entryAmount = -(entry.totalAmount || 0)
            else if (String(entry.investmentAccountId) === String(props.accountId)) entryAmount = entry.StockAmount || 0
        } else if (entry.type === 'stocksell') {
            if (String(entry.cashAccountId) === String(props.accountId)) entryAmount = (entry.totalAmount || 0) - (entry.fees || 0)
            else if (String(entry.investmentAccountId) === String(props.accountId)) entryAmount = -(entry.costBasis || entry.StockAmount || 0)
        }
        if (entry.type !== 'opening-balance') balance += entryAmount
        return { ...entry, balance }
    })
    return result.reverse()
})

const tableEntries = computed(() =>
    isInstrumentAccount.value ? props.entries : entriesWithBalance.value
)
</script>

<template>
    <Card>
        <template #content>
            <DataTable
                class="datatable-compact"
                :value="tableEntries"
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
                <Column field="description" header="Description" class="description-column">
                    <template #body="{ data }">
                        <span
                            v-if="data.type === 'transfer'"
                            v-tooltip.bottom="`${getAccountName(data.originAccountId)}: ${data.originAmount?.toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} ${getAccountCurrency(data.originAccountId)} → ${getAccountName(data.targetAccountId)}: ${data.targetAmount.toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} ${getAccountCurrency(data.targetAccountId)}`"
                        >
                            {{ data.description }}
                        </span>
                        <span v-else>{{ data.description }}</span>
                    </template>
                </Column>

                <Column field="categoryId" header="Category">
                    <template #body="{ data }">
                        <span v-if="data.type === 'expense' || data.type === 'income'">
                            {{ getCategoryName(data.categoryId, data.type) }}
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
                        <!-- Cash layout: original format -->
                        <template v-if="!isInstrumentAccount">
                            <div v-if="data.type === 'opening-balance'" class="amount opening-balance"></div>
                            <div v-else-if="data.type === 'expense'" class="amount expense">
                                -{{ data.Amount.toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) }}
                            </div>
                            <div v-else-if="data.type === 'income'" class="amount income">
                                +{{ data.Amount.toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) }}
                            </div>
                            <div v-else-if="data.type === 'transfer'" class="amount transfer">
                                <template v-if="String(data.targetAccountId) === String(accountId)">
                                    +{{ data.targetAmount.toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) }}
                                </template>
                                <template v-else>
                                    -{{ (data.originAmount ?? 0).toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) }}
                                </template>
                            </div>
                            <div v-else-if="data.type === 'stockbuy'" class="amount" :class="String(data.cashAccountId) === String(accountId) ? 'amount-negative' : 'amount-positive'">
                                <template v-if="data.instrumentId != null && data.quantity != null && data.StockAmount != null">
                                    ({{ getInstrumentSymbol(data.instrumentId) }}) {{ data.quantity }} @ {{ formatPrice(data.StockAmount / data.quantity) }} {{ getInstrumentCurrency(data.instrumentId) }}
                                    <template v-if="String(data.cashAccountId) === String(accountId)">−{{ formatAmount(data.totalAmount) }}</template>
                                    <template v-else>+{{ formatAmount(data.StockAmount) }}</template>
                                </template>
                                <template v-else>
                                    <template v-if="String(data.cashAccountId) === String(accountId)">−{{ formatAmount(data.totalAmount) }}</template>
                                    <template v-else>+{{ formatAmount(data.StockAmount) }}</template>
                                </template>
                            </div>
                            <div v-else-if="data.type === 'stocksell'" class="amount" :class="String(data.cashAccountId) === String(accountId) ? 'amount-positive' : 'amount-negative'">
                                <template v-if="data.instrumentId != null && data.quantity != null">
                                    <template v-if="(data.costBasis != null || data.StockAmount != null) && data.quantity > 0">
                                        ({{ getInstrumentSymbol(data.instrumentId) }}) {{ data.quantity }} @ {{ formatPrice((data.costBasis ?? data.StockAmount) / data.quantity) }} {{ getInstrumentCurrency(data.instrumentId) }}
                                        <template v-if="String(data.cashAccountId) === String(accountId)">
                                            +{{ formatAmount(data.totalAmount) }}{{ data.fees ? ` (−${formatAmount(data.fees)} fee)` : '' }}
                                        </template>
                                        <template v-else>−{{ formatAmount(data.costBasis ?? data.StockAmount) }}</template>
                                    </template>
                                    <template v-else>
                                        <template v-if="String(data.cashAccountId) === String(accountId)">+{{ formatAmount(data.totalAmount) }}</template>
                                        <template v-else>−{{ formatAmount(data.costBasis ?? data.StockAmount) }}</template>
                                    </template>
                                </template>
                                <template v-else>
                                    <template v-if="String(data.cashAccountId) === String(accountId)">+{{ formatAmount(data.totalAmount) }}</template>
                                    <template v-else>−{{ formatAmount(data.costBasis ?? data.StockAmount) }}</template>
                                </template>
                            </div>
                            <div v-else-if="data.type === 'stockgrant'" class="amount">
                                <template v-if="data.instrumentId != null">
                                    ({{ getInstrumentSymbol(data.instrumentId) }}) {{ data.quantity ?? '—' }} (grant)
                                </template>
                                <template v-else>{{ data.quantity }} (grant)</template>
                            </div>
                            <div v-else-if="data.type === 'stocktransfer'" class="amount">
                                <template v-if="data.instrumentId != null">
                                    ({{ getInstrumentSymbol(data.instrumentId) }})
                                    <template v-if="String(data.originAccountId) === String(accountId)">−{{ data.quantity }}</template>
                                    <template v-else-if="String(data.targetAccountId) === String(accountId)">+{{ data.quantity }}</template>
                                    <template v-else>—</template>
                                </template>
                                <template v-else>
                                    <template v-if="String(data.originAccountId) === String(accountId)">−{{ data.quantity }}</template>
                                    <template v-else-if="String(data.targetAccountId) === String(accountId)">+{{ data.quantity }}</template>
                                    <template v-else>—</template>
                                </template>
                            </div>
                            <div v-else class="amount">
                                {{ (data.totalAmount ?? data.targetAmount ?? 0).toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) }}
                            </div>
                        </template>
                        <!-- Instrument layout: quantity + ticker -->
                        <template v-else>
                            <div v-if="data.type === 'opening-balance'" class="amount opening-balance">—</div>
                            <div v-else-if="data.type === 'expense'" class="amount expense">
                                -{{ data.Amount.toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) }}
                            </div>
                            <div v-else-if="data.type === 'income'" class="amount income">
                                +{{ data.Amount.toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) }}
                            </div>
                            <div v-else-if="data.type === 'transfer'" class="amount transfer">
                                <template v-if="String(data.targetAccountId) === String(accountId)">
                                    +{{ data.targetAmount.toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) }}
                                </template>
                                <template v-else>-{{ (data.originAmount ?? 0).toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) }}</template>
                            </div>
                            <div v-else-if="data.type === 'stockbuy'" class="amount" :class="String(data.cashAccountId) === String(accountId) ? 'amount-negative' : 'amount-positive'">
                                <template v-if="data.instrumentId != null && data.quantity != null">
                                    <template v-if="String(data.cashAccountId) === String(accountId)">−{{ formatAmount(data.totalAmount) }} {{ getAccountCurrency(data.cashAccountId) }}</template>
                                    <template v-else>+{{ data.quantity }} {{ getInstrumentSymbol(data.instrumentId) }}</template>
                                </template>
                                <template v-else>
                                    <template v-if="String(data.cashAccountId) === String(accountId)">−{{ formatAmount(data.totalAmount) }}</template>
                                    <template v-else>+{{ formatAmount(data.StockAmount) }}</template>
                                </template>
                            </div>
                            <div v-else-if="data.type === 'stocksell'" class="amount" :class="String(data.cashAccountId) === String(accountId) ? 'amount-positive' : 'amount-negative'">
                                <template v-if="data.instrumentId != null && data.quantity != null">
                                    <template v-if="String(data.cashAccountId) === String(accountId)">+{{ formatAmount((data.totalAmount ?? 0) - (data.fees ?? 0)) }} {{ getAccountCurrency(data.cashAccountId) }}{{ data.fees ? ` (net, −${formatAmount(data.fees)} fee)` : '' }}</template>
                                    <template v-else>−{{ data.quantity }} {{ getInstrumentSymbol(data.instrumentId) }}</template>
                                </template>
                                <template v-else>
                                    <template v-if="String(data.cashAccountId) === String(accountId)">+{{ formatAmount(data.totalAmount) }}</template>
                                    <template v-else>−{{ formatAmount(data.costBasis ?? data.StockAmount) }}</template>
                                </template>
                            </div>
                            <div v-else-if="data.type === 'stockgrant'" class="amount">
                                <template v-if="data.instrumentId != null">+{{ data.quantity ?? '—' }} {{ getInstrumentSymbol(data.instrumentId) }}</template>
                                <template v-else>{{ data.quantity }} (grant)</template>
                            </div>
                            <div v-else-if="data.type === 'stocktransfer'" class="amount">
                                <template v-if="data.instrumentId != null">
                                    <template v-if="String(data.originAccountId) === String(accountId)">−{{ data.quantity }} {{ getInstrumentSymbol(data.instrumentId) }}</template>
                                    <template v-else-if="String(data.targetAccountId) === String(accountId)">+{{ data.quantity }} {{ getInstrumentSymbol(data.instrumentId) }}</template>
                                    <template v-else>—</template>
                                </template>
                                <template v-else>
                                    <template v-if="String(data.originAccountId) === String(accountId)">−{{ data.quantity }}</template>
                                    <template v-else-if="String(data.targetAccountId) === String(accountId)">+{{ data.quantity }}</template>
                                    <template v-else>—</template>
                                </template>
                            </div>
                            <div v-else class="amount">
                                {{ (data.totalAmount ?? data.targetAmount ?? 0).toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) }}
                            </div>
                        </template>
                    </template>
                </Column>

                <Column v-if="!isInstrumentAccount" field="balance" header="Balance" bodyStyle="text-align: right" class="balance-column">
                    <template #body="{ data }">
                        <div class="balance" :class="{ 'balance-negative': data.balance < 0 }">
                            {{ data.balance.toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) }}
                        </div>
                    </template>
                </Column>

                <Column v-if="isInstrumentAccount" field="price" header="Price" bodyStyle="text-align: right" class="price-column">
                    <template #body="{ data }">
                        <template v-if="data.type === 'stockbuy' || data.type === 'stocksell'">
                            <span v-if="data.quantity != null && data.StockAmount != null && data.quantity > 0">
                                {{ formatPrice(data.StockAmount / data.quantity) }} {{ getInstrumentCurrency(data.instrumentId) }}
                            </span>
                            <span v-else>—</span>
                        </template>
                        <template v-else-if="data.type === 'stockgrant'">—</template>
                        <template v-else-if="data.type === 'stocktransfer'">—</template>
                        <template v-else>—</template>
                    </template>
                </Column>

                <Column header="Actions" style="width: 120px">
                    <template #body="{ data }">
                        <!-- No actions for opening balance entry; placeholder keeps row height -->
                        <div v-if="data.type === 'opening-balance'" class="flex align-items-center" style="height: 2.5rem"></div>
                        <div v-else class="flex gap-1 justify-content-start">
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

