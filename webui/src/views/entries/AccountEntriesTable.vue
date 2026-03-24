<script setup>
import { computed, ref } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Card from 'primevue/card'
import { useCategoryUtils } from '@/utils/categoryUtils'
import { useAccountUtils } from '@/utils/accountUtils'
import { useInstruments } from '@/composables/useInstruments'
import { useDateFormat } from '@/composables/useDateFormat'
import { ACCOUNT_TYPES } from '@/types/account'
import { getAttachmentUrl } from '@/lib/api/Attachment'
import { formatAmount, formatCurrency } from '@/utils/currency'
import AdHocCategoryRuleDialog from '@/components/common/AdHocCategoryRuleDialog.vue'

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
const { getCategoryName, getCategoryPath, getCategoryIcon } = useCategoryUtils()
const { getAccountCurrency, getAccountName } = useAccountUtils()
const { instruments: instrumentsData } = useInstruments()
const { formatDate } = useDateFormat()

const instrumentsMap = computed(() => {
    const list = instrumentsData.value ?? []
    return Object.fromEntries(list.map((i) => [i.id, i]))
})
const getInstrumentSymbol = (instrumentId) => instrumentsMap.value[instrumentId]?.symbol ?? '—'
const getInstrumentCurrency = (instrumentId) => instrumentsMap.value[instrumentId]?.currency ?? ''
const formatPrice = (n) => (n != null && !Number.isNaN(n) ? formatCurrency(n, 2, 2) : '—')

/* --- Helpers --- */
const getRowClass = (data) => ({
    'expense-row': data.type === 'expense',
    'income-row': data.type === 'income',
    'transfer-row': data.type === 'transfer',
    'stockbuy-row': data.type === 'stockbuy',
    'stocksell-row': data.type === 'stocksell',
    'stockgrant-row': data.type === 'stockgrant',
    'stocktransfer-row': data.type === 'stocktransfer',
    'stockvest-row': data.type === 'stockvest',
    'stockforfeit-row': data.type === 'stockforfeit',
    'opening-balance-row': data.type === 'opening-balance',
    'balancestatus-row': data.type === 'balancestatus',
    'revaluation-row': data.type === 'revaluation'
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

const openAttachment = (data) => {
    window.open(getAttachmentUrl(data.id), '_blank')
}

const adhocDialogRef = ref(null)

/* --- Balance (cash accounts only) --- */
const entriesWithBalance = computed(() => {
    if (!props.entries || props.entries.length === 0 || isInstrumentAccount.value) return props.entries ?? []
    const openingBalanceEntry = props.entries.find((e) => e.type === 'opening-balance')
    let balance = openingBalanceEntry?.Amount || 0
    const entriesReversed = [...props.entries].reverse()
    const result = entriesReversed.map((entry) => {
        let entryAmount = 0
        if (entry.type === 'opening-balance') entryAmount = 0
        else if (entry.type === 'balancestatus') entryAmount = 0
        else if (entry.type === 'revaluation') entryAmount = entry.Amount || 0
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
        return { ...entry, runningBalance: balance }
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
                :pageLinkSize="11"
                :rowClass="getRowClass"
                @page="handlePage"
            >
                <Column field="description" header="Description" class="description-column">
                    <template #body="{ data }">
                        <span
                            v-if="data.type === 'transfer'"
                            v-tooltip.bottom="`${getAccountName(data.originAccountId)}: ${data.originAmount != null ? formatAmount(data.originAmount) : '0.00'} ${getAccountCurrency(data.originAccountId)} → ${getAccountName(data.targetAccountId)}: ${formatAmount(data.targetAmount)} ${getAccountCurrency(data.targetAccountId)}`"
                        >
                            {{ data.description }}
                        </span>
                        <span v-else>{{ data.description }}</span>
                    </template>
                </Column>

                <Column style="width: 2rem; padding: 0" bodyStyle="text-align: center">
                    <template #body="{ data }">
                        <Button
                            v-if="data.attachmentId"
                            icon="ti ti-paperclip"
                            text
                            rounded
                            size="small"
                            @click="openAttachment(data)"
                            v-tooltip.bottom="'View Attachment'"
                        />
                    </template>
                </Column>

                <Column field="categoryId">
                    <template #header>
                        <span class="font-semibold">Category</span>
                        <Button
                            icon="ti ti-bolt"
                            text
                            size="small"
                            class="ml-1 p-0 no-hover"
                            @click="adhocDialogRef?.open()"
                        />
                    </template>
                    <template #body="{ data }">
                        <span v-if="data.type === 'expense' || data.type === 'income' || data.type === 'stockvest'" :class="{ 'unclassified': !data.categoryId }" class="category-cell">
                            <i :class="['ti', `ti-${getCategoryIcon(data.categoryId, data.type === 'stockvest' ? 'income' : data.type)}`]"></i>
                            {{ getCategoryName(data.categoryId, data.type === 'stockvest' ? 'income' : data.type) }}
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
                                -{{ formatAmount(data.Amount) }}
                            </div>
                            <div v-else-if="data.type === 'income'" class="amount income">
                                +{{ formatAmount(data.Amount) }}
                            </div>
                            <div v-else-if="data.type === 'transfer'" class="amount transfer">
                                <template v-if="String(data.targetAccountId) === String(accountId)">
                                    +{{ formatAmount(data.targetAmount) }}
                                </template>
                                <template v-else>
                                    -{{ formatAmount(data.originAmount ?? 0) }}
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
                            <div v-else-if="data.type === 'stockvest'" class="amount">
                                <template v-if="data.instrumentId != null">
                                    ({{ getInstrumentSymbol(data.instrumentId) }})
                                    <template v-if="String(data.originAccountId) === String(accountId)">−{{ data.quantity }}</template>
                                    <template v-else-if="String(data.targetAccountId) === String(accountId)">+{{ data.quantity }}</template>
                                    <template v-else>—</template>
                                    <template v-if="data.vestingPrice != null && data.quantity != null">
                                        = {{ formatAmount(data.vestingPrice * data.quantity) }} {{ getInstrumentCurrency(data.instrumentId) }}
                                    </template>
                                </template>
                                <template v-else>
                                    <template v-if="String(data.originAccountId) === String(accountId)">−{{ data.quantity }}</template>
                                    <template v-else-if="String(data.targetAccountId) === String(accountId)">+{{ data.quantity }}</template>
                                    <template v-else>—</template>
                                </template>
                            </div>
                            <div v-else-if="data.type === 'stockforfeit'" class="amount amount-negative">
                                <template v-if="data.instrumentId != null">
                                    ({{ getInstrumentSymbol(data.instrumentId) }}) −{{ data.quantity }}
                                </template>
                                <template v-else>−{{ data.quantity }}</template>
                            </div>
                            <div v-else-if="data.type === 'balancestatus'" class="amount balance-status">
                                {{ formatAmount(data.Amount) }}
                                {{ getAccountCurrency(data.accountId) }}
                            </div>
                            <div v-else-if="data.type === 'revaluation'" :class="['amount', (data.Amount ?? 0) >= 0 ? 'amount-positive' : 'amount-negative']">
                                {{ (data.Amount ?? 0) >= 0 ? '+' : '' }}{{ formatAmount(data.Amount ?? 0) }}
                                {{ getAccountCurrency(data.accountId) }}
                            </div>
                            <div v-else class="amount">
                                {{ formatAmount(data.totalAmount ?? data.targetAmount ?? 0) }}
                            </div>
                        </template>
                        <!-- Instrument layout: quantity + ticker -->
                        <template v-else>
                            <div v-if="data.type === 'opening-balance'" class="amount opening-balance">—</div>
                            <div v-else-if="data.type === 'expense'" class="amount expense">
                                -{{ formatAmount(data.Amount) }}
                            </div>
                            <div v-else-if="data.type === 'income'" class="amount income">
                                +{{ formatAmount(data.Amount) }}
                            </div>
                            <div v-else-if="data.type === 'transfer'" class="amount transfer">
                                <template v-if="String(data.targetAccountId) === String(accountId)">
                                    +{{ formatAmount(data.targetAmount) }}
                                </template>
                                <template v-else>-{{ formatAmount(data.originAmount ?? 0) }}</template>
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
                            <div v-else-if="data.type === 'stockvest'" class="amount">
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
                            <div v-else-if="data.type === 'stockforfeit'" class="amount amount-negative">
                                <template v-if="data.instrumentId != null">−{{ data.quantity }} {{ getInstrumentSymbol(data.instrumentId) }}</template>
                                <template v-else>−{{ data.quantity }}</template>
                            </div>
                            <div v-else class="amount">
                                {{ formatAmount(data.totalAmount ?? data.targetAmount ?? 0) }}
                            </div>
                        </template>
                    </template>
                </Column>

                <Column v-if="!isInstrumentAccount" field="runningBalance" header="Balance" bodyStyle="text-align: right" class="balance-column">
                    <template #body="{ data }">
                        <div class="balance" :class="{ 'balance-negative': data.runningBalance < 0 && !Object.is(data.runningBalance, -0) }">
                            {{ formatAmount(data.runningBalance) }}
                        </div>
                    </template>
                </Column>

                <Column v-if="isInstrumentAccount" field="price" header="Price" bodyStyle="text-align: right" class="price-column">
                    <template #body="{ data }">
                        <template v-if="data.type === 'stockbuy'">
                            <span v-if="data.quantity != null && data.StockAmount != null && data.quantity > 0">
                                {{ formatPrice(data.StockAmount / data.quantity) }} {{ getInstrumentCurrency(data.instrumentId) }}
                            </span>
                            <span v-else>—</span>
                        </template>
                        <template v-else-if="data.type === 'stocksell'">
                            <span v-if="data.quantity != null && data.totalAmount != null && data.quantity !== 0">
                                {{ formatPrice(Math.abs(data.totalAmount / data.quantity)) }} {{ getInstrumentCurrency(data.instrumentId) }}
                            </span>
                            <span v-else>—</span>
                        </template>
                        <template v-else-if="data.type === 'stockvest'">
                            <span v-if="data.vestingPrice != null">
                                {{ formatPrice(data.vestingPrice) }} {{ getInstrumentCurrency(data.instrumentId) }}
                            </span>
                            <span v-else>—</span>
                        </template>
                        <template v-else-if="data.type === 'stockgrant'">—</template>
                        <template v-else-if="data.type === 'stocktransfer'">—</template>
                        <template v-else-if="data.type === 'stockforfeit'">—</template>
                        <template v-else>—</template>
                    </template>
                </Column>

                <Column header="Actions" style="width: 120px">
                    <template #body="{ data }">
                        <!-- No actions for opening balance entry; placeholder keeps row height -->
                        <div v-if="data.type === 'opening-balance'" class="flex align-items-center" style="height: 2.5rem"></div>
                        <div v-else class="flex gap-1 justify-content-start">
                            <Button
                                icon="ti ti-pencil"
                                text
                                rounded
                                class="p-1"
                                @click="handleEdit(data)"
                                v-tooltip.bottom="'Edit'"
                            />
                            <Button
                                icon="ti ti-copy"
                                text
                                rounded
                                class="p-1"
                                @click="handleDuplicate(data)"
                                v-tooltip.bottom="'Duplicate'"
                            />
                            <Button
                                icon="ti ti-trash"
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
            <AdHocCategoryRuleDialog ref="adhocDialogRef" />
        </template>
    </Card>
</template>

<style scoped>
.attachment-icon {
    font-size: 0.85rem;
    margin-left: 0.4rem;
    cursor: pointer;
    opacity: 0.6;
    font-weight: bold;
}
.attachment-icon:hover {
    opacity: 1;
}
.unclassified {
    font-style: italic;
    opacity: 0.6;
}
.category-cell {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
}
:deep(.no-hover:hover) {
    background: transparent !important;
}
</style>
