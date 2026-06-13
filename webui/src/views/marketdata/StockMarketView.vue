<script setup>
import { ResponsiveHorizontal } from '@/components/layout'
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import Button from 'primevue/button'
import Card from 'primevue/card'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Tag from 'primevue/tag'
import Message from 'primevue/message'
import Dialog from 'primevue/dialog'
import Checkbox from 'primevue/checkbox'
import DatePicker from 'primevue/datepicker'
import InputNumber from 'primevue/inputnumber'
import InstrumentFilters from '@/components/marketdata/InstrumentFilters.vue'
import InstrumentDialog from '@/views/instruments/dialogs/InstrumentDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { useDateFormat } from '@/composables/useDateFormat'
import { useInstruments } from '@/composables/useInstruments'
import { useSettingsStore } from '@/store/settingsStore'
import { useCompareSelection } from '@/store/compareSelection'
import { useToast } from 'primevue/usetoast'
import { useTaskRunner } from '@/composables/useTaskRunner'
import { getApiErrorMessage } from '@/utils/apiError'
import {
    useMarketInstruments,
    useMarketDataMutations,
    filterMarketInstruments,
    toLocalDateString,
    formatPrice,
    formatPct,
    getChangeSeverity
} from '@/composables/useMarketData'
const { formatDate, pickerDateFormat } = useDateFormat()

const router = useRouter()
const toast = useToast()
const settingsStore = useSettingsStore()
const defaultCurrency = computed(() => settingsStore.mainCurrency || 'CHF')

const compare = useCompareSelection()

function onToggleCompare(id) {
    if (!compare.toggle(id)) {
        toast.add({
            severity: 'warn',
            summary: 'Compare limit reached',
            detail: 'You can compare up to 10 instruments at once.',
            life: 4000
        })
    }
}

// Compare mode reveals the selection checkboxes. The first click on the Compare
// button enters this mode; a subsequent click (with >= 2 selected) opens the
// comparison. "Clear" leaves the mode and drops the selection. Initialised from
// the store so returning from the compare view keeps the checkboxes visible.
const compareMode = ref(compare.count > 0)

function onCompareClick() {
    if (!compareMode.value) {
        compareMode.value = true
        return
    }
    // Carry the selection in the URL so the comparison link is shareable.
    router.push({ name: 'stock-compare', query: { ids: compare.selectedIds.join(',') } })
}

function exitCompareMode() {
    compare.clear()
    compareMode.value = false
}

function clearSelection() {
    compare.clear()
}

const compareTooltip = computed(() => {
    if (!compareMode.value) return 'Select instruments to compare'
    return compare.canCompare ? 'Compare selected instruments' : 'Select at least 2 instruments'
})

const { instruments, isLoading, isError, error, refetch } = useMarketInstruments()

// "Update" button: trigger the financial-import task and refresh the table when it finishes.
const {
    run: runImport,
    isRunning: isImporting,
    isTriggering: isTriggeringImport
} = useTaskRunner('financial-import', {
    onComplete: (status) => {
        if (status === 'complete') {
            toast.add({
                severity: 'success',
                summary: 'Update complete',
                detail: 'Market data was refreshed.',
                life: 4000
            })
            refetch()
        } else {
            toast.add({
                severity: 'error',
                summary: 'Update failed',
                detail: 'Check the Tasks page for details.',
                life: 6000
            })
        }
    }
})

async function handleImport() {
    try {
        await runImport()
    } catch (e) {
        toast.add({
            severity: 'error',
            summary: 'Failed to start update',
            detail: getApiErrorMessage(e),
            life: 6000
        })
    }
}
const {
    createInstrument,
    updateInstrument,
    deleteInstrument,
    isCreatingInstrument,
    isUpdatingInstrument
} = useInstruments()
const isSaving = computed(() => isCreatingInstrument.value || isUpdatingInstrument.value)

const leftSidebarCollapsed = ref(true)

// --- Filtering (client-side) ---
const filtersExpanded = ref(false)
const search = ref('')
const types = ref([])
const exchanges = ref([])

const uniqueSorted = (values) =>
    [...new Set(values.filter((v) => v && v.length > 0))].sort((a, b) => a.localeCompare(b))

const typeOptions = computed(() => uniqueSorted(instruments.value.map((i) => i.type)))
const exchangeOptions = computed(() => uniqueSorted(instruments.value.map((i) => i.exchange)))

const filteredInstruments = computed(() =>
    filterMarketInstruments(instruments.value, {
        search: search.value,
        types: types.value,
        exchanges: exchanges.value
    })
)

// --- Add price dialog (unchanged) ---
const addDialogInstrument = ref(null)
const addDialogVisible = ref(false)
const addDialogForm = ref({ date: '', price: 0 })
const addDialogSymbol = computed(() => addDialogInstrument.value?.symbol ?? '')
const { createPrice: createPriceMutation, isCreating: isCreatingPrice } = useMarketDataMutations(addDialogSymbol)

function openAddDialog(inst) {
    addDialogInstrument.value = inst
    addDialogForm.value = { date: toLocalDateString(new Date()), price: 0 }
    addDialogVisible.value = true
}

async function saveAddDialog() {
    if (!addDialogInstrument.value) return
    const { date, price } = addDialogForm.value
    if (!date) return
    const time = date.includes('T') ? toLocalDateString(new Date(date)) : date
    try {
        // The quick-add dialog captures a single price; map it to a flat OHLC bar
        // (open=high=low=close) since the backend expects the full price record.
        await createPriceMutation({ time, open: price, high: price, low: price, close: price, volume: 0 })
        addDialogVisible.value = false
    } catch (err) {
        toast.add({ severity: 'error', summary: 'Error', detail: getApiErrorMessage(err), life: 5000 })
        console.error('Failed to add price:', err)
    }
}

// --- Notes dialog (unchanged) ---
const notesDialogVisible = ref(false)
const notesDialogInstrument = ref(null)

function openNotesDialog(inst) {
    notesDialogInstrument.value = inst
    notesDialogVisible.value = true
}

// --- Instrument create / edit / delete ---
const selectedInstrument = ref(null)
const instrumentDialogVisible = ref(false)
const isEditInstrument = ref(false)
const deleteInstrumentDialogVisible = ref(false)
const instrumentToDelete = ref(null)

const openNewInstrumentDialog = () => {
    selectedInstrument.value = {
        symbol: '',
        name: '',
        currency: defaultCurrency.value,
        notes: '',
        type: '',
        exchange: ''
    }
    isEditInstrument.value = false
    instrumentDialogVisible.value = true
}

const editInstrument = (inst) => {
    selectedInstrument.value = {
        id: inst.id,
        symbol: inst.symbol,
        name: inst.name,
        currency: inst.currency,
        notes: inst.notes,
        type: inst.type,
        exchange: inst.exchange
    }
    isEditInstrument.value = true
    instrumentDialogVisible.value = true
}

const showDeleteInstrumentDialog = (inst) => {
    instrumentToDelete.value = inst
    deleteInstrumentDialogVisible.value = true
}

const confirmDeleteInstrument = async () => {
    if (!instrumentToDelete.value) return
    try {
        await deleteInstrument(instrumentToDelete.value.id)
        deleteInstrumentDialogVisible.value = false
        instrumentToDelete.value = null
    } catch (err) {
        toast.add({ severity: 'error', summary: 'Error', detail: getApiErrorMessage(err), life: 5000 })
        console.error('Failed to delete instrument:', err)
    }
}

const saveInstrument = async (payload) => {
    try {
        if (payload.id) {
            await updateInstrument({
                id: payload.id,
                payload: {
                    symbol: payload.symbol,
                    name: payload.name,
                    currency: payload.currency,
                    notes: payload.notes ?? '',
                    type: payload.type,
                    exchange: payload.exchange
                }
            })
        } else {
            await createInstrument({
                symbol: payload.symbol,
                name: payload.name,
                currency: payload.currency,
                notes: payload.notes ?? '',
                type: payload.type,
                exchange: payload.exchange
            })
        }
        instrumentDialogVisible.value = false
    } catch (err) {
        toast.add({ severity: 'error', summary: 'Error', detail: getApiErrorMessage(err), life: 5000 })
        console.error('Failed to save instrument:', err)
    }
}

const onRowClick = (event) => {
    router.push({ name: 'stock-detail', params: { id: event.data.id, tab: 'overview' } })
}
</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="p-3">
                <div class="mb-2 flex align-items-start justify-content-between gap-2">
                    <div>
                        <h1 class="flex align-items-center gap-3 m-0 mb-2">
                            <i class="ti ti-chart-line text-primary"></i>
                            Stock Market
                        </h1>
                    </div>
                    <div class="flex align-items-center gap-2">
                        <Button
                            v-if="compareMode && compare.count > 0"
                            label="Clear"
                            text
                            size="small"
                            severity="secondary"
                            @click="clearSelection"
                        />
                        <Button
                            v-if="compareMode"
                            label="Exit"
                            icon="ti ti-x"
                            text
                            size="small"
                            severity="secondary"
                            @click="exitCompareMode"
                        />
                        <Button
                            icon="ti ti-git-compare"
                            :label="compareMode ? `Compare (${compare.count})` : 'Compare'"
                            severity="secondary"
                            outlined
                            size="small"
                            :disabled="compareMode && !compare.canCompare"
                            v-tooltip.bottom="compareTooltip"
                            @click="onCompareClick"
                        />
                        <Button
                            :icon="filtersExpanded ? 'ti ti-filter-off' : 'ti ti-filter'"
                            label="Filter"
                            severity="secondary"
                            outlined
                            size="small"
                            @click="filtersExpanded = !filtersExpanded"
                        />
                        <Button
                            icon="ti ti-refresh"
                            label="Update"
                            severity="secondary"
                            outlined
                            size="small"
                            :loading="isTriggeringImport || isImporting"
                            :disabled="isTriggeringImport || isImporting"
                            v-tooltip.bottom="'Import recent prices for all instruments'"
                            @click="handleImport"
                        />
                        <Button
                            icon="ti ti-plus"
                            label="New Instrument"
                            size="small"
                            @click="openNewInstrumentDialog"
                        />
                    </div>
                </div>

                <InstrumentFilters
                    v-model:search="search"
                    v-model:types="types"
                    v-model:exchanges="exchanges"
                    v-model:expanded="filtersExpanded"
                    :type-options="typeOptions"
                    :exchange-options="exchangeOptions"
                />

                <Message v-if="isError" severity="error" :closable="false" class="mb-3">
                    <div class="flex align-items-center gap-2 flex-wrap">
                        <i class="ti ti-alert-triangle"></i>
                        <span>{{ error?.message ?? 'Failed to load market data.' }}</span>
                        <Button label="Retry" icon="ti ti-refresh" text size="small" @click="refetch" />
                    </div>
                </Message>

                <Card v-if="!instruments.length && !isLoading">
                    <template #content>
                        <div class="empty-message">
                            No instruments configured. Use the <strong>New Instrument</strong>
                            button above to add one and start tracking market data.
                        </div>
                    </template>
                </Card>

                <Card v-else>
                    <template #content>
                        <DataTable
                            :value="filteredInstruments"
                            :loading="isLoading"
                            dataKey="id"
                            stripedRows
                            sortField="symbol"
                            :sortOrder="1"
                            :paginator="filteredInstruments.length > 15"
                            :rows="15"
                            class="p-datatable-sm clickable-rows stock-table"
                            selectionMode="single"
                            @rowClick="onRowClick"
                        >
                            <Column v-if="compareMode" header="" :exportable="false" style="width: 3rem; min-width: 3rem">
                                <template #body="{ data }">
                                    <Checkbox
                                        :modelValue="compare.isSelected(data.id)"
                                        binary
                                        @click.stop
                                        @update:modelValue="() => onToggleCompare(data.id)"
                                    />
                                </template>
                            </Column>
                            <Column field="symbol" header="Symbol" sortable>
                                <template #body="{ data }">
                                    <span class="font-bold">{{ data.symbol }}</span>
                                </template>
                            </Column>
                            <Column field="name" header="Name" sortable>
                                <template #body="{ data }">
                                    <span>{{ data.name }}</span>
                                    <Button
                                        v-if="data.notes"
                                        icon="ti ti-help-circle"
                                        text
                                        rounded
                                        size="small"
                                        class="p-1 ml-1 note-btn"
                                        v-tooltip.bottom="'View note'"
                                        @click.stop="openNotesDialog(data)"
                                    />
                                </template>
                            </Column>
                            <Column field="type" header="Type" sortable>
                                <template #body="{ data }">
                                    <span>{{ data.type || '-' }}</span>
                                </template>
                            </Column>
                            <Column field="lastPrice" header="Price" sortable>
                                <template #body="{ data }">
                                    <span class="font-semibold">{{ formatPrice(data.lastPrice) }}</span>
                                    <span class="text-color-secondary text-sm ml-1">{{ data.currency }}</span>
                                </template>
                            </Column>
                            <Column field="changePct" header="Change" sortable>
                                <template #body="{ data }">
                                    <Tag
                                        :value="formatPct(data.changePct)"
                                        :severity="getChangeSeverity(data.changePct)"
                                    />
                                </template>
                            </Column>
                            <Column field="lastUpdate" header="Last update" sortable bodyClass="last-update-cell" headerClass="last-update-cell">
                                <template #body="{ data }">
                                    {{ data.lastUpdate ? formatDate(data.lastUpdate) : '-' }}
                                </template>
                            </Column>
                            <Column header="Actions" class="actions-column" style="width: 7rem; min-width: 7rem">
                                <template #body="{ data }">
                                    <div class="flex gap-2 justify-content-end">
                                        <Button
                                            icon="ti ti-plus"
                                            text
                                            rounded
                                            class="p-1"
                                            v-tooltip.bottom="'Add price'"
                                            :loading="isCreatingPrice && addDialogInstrument?.id === data.id"
                                            @click.stop="openAddDialog(data)"
                                        />
                                        <Button
                                            icon="ti ti-pencil"
                                            text
                                            rounded
                                            class="p-1"
                                            v-tooltip.bottom="'Edit'"
                                            @click.stop="editInstrument(data)"
                                        />
                                        <Button
                                            icon="ti ti-trash"
                                            text
                                            rounded
                                            severity="danger"
                                            class="p-1"
                                            v-tooltip.bottom="'Delete'"
                                            @click.stop="showDeleteInstrumentDialog(data)"
                                        />
                                    </div>
                                </template>
                            </Column>
                        </DataTable>
                        <p class="text-color-secondary text-sm mt-2 mb-0">
                            Click a row to open details, chart and edit data. Use + to add a price.
                        </p>
                    </template>
                </Card>

                <Dialog
                    v-model:visible="addDialogVisible"
                    header="Add market data"
                    modal
                    class="entry-dialog"
                    @hide="addDialogVisible = false"
                >
                    <div v-if="addDialogInstrument" class="flex flex-column gap-3 py-2">
                        <p class="text-color-secondary mt-0 mb-0">
                            {{ addDialogInstrument.symbol }} – {{ addDialogInstrument.name }}
                        </p>
                        <div class="field">
                            <label for="add-data-date">Date</label>
                            <DatePicker
                                id="add-data-date"
                                :modelValue="addDialogForm.date ? new Date(addDialogForm.date + 'T12:00:00') : null"
                                @update:modelValue="(d) => { addDialogForm.date = d ? toLocalDateString(d) : '' }"
                                :dateFormat="pickerDateFormat"
                                showIcon
                                class="w-full"
                            />
                        </div>
                        <div class="field">
                            <label for="add-data-price">Price</label>
                            <InputNumber
                                id="add-data-price"
                                v-model="addDialogForm.price"
                                mode="decimal"
                                :minFractionDigits="2"
                                :maxFractionDigits="2"
                                :min="0"
                                class="w-full"
                            />
                        </div>
                    </div>
                    <template #footer>
                        <Button label="Save" icon="ti ti-check" :loading="isCreatingPrice" @click="saveAddDialog" />
                        <Button label="Cancel" text severity="secondary" @click="addDialogVisible = false" />
                    </template>
                </Dialog>

                <Dialog
                    v-model:visible="notesDialogVisible"
                    :header="notesDialogInstrument ? `Note – ${notesDialogInstrument.symbol}` : 'Note'"
                    modal
                    class="entry-dialog"
                >
                    <p v-if="notesDialogInstrument" class="notes-content">{{ notesDialogInstrument.notes }}</p>
                    <template #footer>
                        <Button label="Close" text severity="secondary" @click="notesDialogVisible = false" />
                    </template>
                </Dialog>

                <InstrumentDialog
                    v-if="selectedInstrument"
                    v-model:visible="instrumentDialogVisible"
                    :is-edit="isEditInstrument"
                    :instrument="selectedInstrument"
                    :loading="isSaving"
                    @save="saveInstrument"
                />

                <ConfirmDialog
                    v-model:visible="deleteInstrumentDialogVisible"
                    :name="instrumentToDelete?.name"
                    title="Delete investment instrument"
                    message="Are you sure you want to delete this investment instrument?"
                    @confirm="confirmDeleteInstrument"
                />
            </div>
        </template>
    </ResponsiveHorizontal>
</template>

<style scoped>
.empty-message {
    padding: 1rem;
    text-align: center;
    color: var(--p-text-muted-color);
}

.empty-message a {
    color: var(--p-primary-color);
    text-decoration: none;
    font-weight: 600;
}

.empty-message a:hover {
    text-decoration: underline;
}

:deep(.clickable-rows .p-datatable-tbody > tr) {
    cursor: pointer;
}

:deep(.last-update-cell) {
    text-align: right;
}

:deep(.stock-table .p-datatable-thead th.last-update-cell .p-datatable-column-header-content) {
    justify-content: flex-end;
}

.field label {
    display: block;
    font-weight: 600;
    margin-bottom: 0.35rem;
    font-size: 0.9rem;
}

.notes-content {
    margin: 0;
    white-space: pre-wrap;
    word-break: break-word;
    max-width: 32rem;
}
</style>
