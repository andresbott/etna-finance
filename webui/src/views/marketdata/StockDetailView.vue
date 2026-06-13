<script setup lang="ts">
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Card from 'primevue/card'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import Divider from 'primevue/divider'
import SelectButton from 'primevue/selectbutton'
import TabView from 'primevue/tabview'
import TabPanel from 'primevue/tabpanel'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Dialog from 'primevue/dialog'
import DatePicker from 'primevue/datepicker'
import InputNumber from 'primevue/inputnumber'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import StockChartTab from '@/views/marketdata/StockChartTab.vue'
import { useDateFormat } from '@/composables/useDateFormat'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import {
    GridComponent,
    TooltipComponent,
    LegendComponent,
    MarkLineComponent
} from 'echarts/components'
import {
    useMarketInstruments,
    usePriceHistory,
    useMarketDataMutations,
    useEpsHistory,
    useEpsMutations,
    toLocalDateString,
    formatPrice,
    formatVolume,
    formatPct,
    formatChange,
    getChangeSeverity,
    type MarketInstrument,
    type PriceHistoryRange
} from '@/composables/useMarketData'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, LegendComponent, MarkLineComponent])

const route = useRoute()
const router = useRouter()
const { instruments, isLoading } = useMarketInstruments()
const { formatDate, pickerDateFormat } = useDateFormat()
const TAB_NAMES = ['overview', 'chart', 'raw-data', 'eps'] as const

const leftSidebarCollapsed = ref(true)
const activeTabIndex = ref(0)

// Sync tab from route (e.g. on load or browser back/forward)
function tabFromRoute(): number {
    const tab = route.params.tab as string | undefined
    const idx = TAB_NAMES.indexOf(tab as (typeof TAB_NAMES)[number])
    return idx >= 0 ? idx : 0
}
watch(
    () => route.params.tab,
    () => {
        activeTabIndex.value = tabFromRoute()
    },
    { immediate: true }
)
// Update URL when user switches tab (replace to avoid stacking history)
watch(activeTabIndex, (index) => {
    const tab = TAB_NAMES[index] ?? 'overview'
    if (route.params.tab !== tab) {
        router.replace({
            name: 'stock-detail',
            params: { id: route.params.id, tab }
        })
    }
})

const instrument = computed<MarketInstrument | undefined>(() => {
    const id = Number(route.params.id)
    return instruments.value.find((inst) => inst.id === id)
})

const dateRangeOptions: { value: PriceHistoryRange; label: string }[] = [
    { value: '7d', label: '7D' },
    { value: '6m', label: '6M' },
    { value: '1y', label: '1Y' },
    { value: 'max', label: 'Max' }
]

const selectedRange = ref<PriceHistoryRange>('6m')
const symbol = computed(() => instrument.value?.symbol ?? '')

const { data: priceHistoryData, refetch: refetchPriceHistory } = usePriceHistory(symbol, selectedRange)
const {
    createPrice: createPriceMutation,
    updatePrice: updatePriceMutation,
    deletePrice: deletePriceMutation,
    isCreating,
    isUpdating,
    isDeleting
} = useMarketDataMutations(symbol)

const priceHistory = computed(() => priceHistoryData.value)

const rangeChange = computed(() => {
    const closes = priceHistoryData.value.closes
    if (closes.length < 2) return null
    return closes[closes.length - 1] - closes[0]
})

const rangeChangePct = computed(() => {
    const closes = priceHistoryData.value.closes
    if (closes.length < 2) return null
    const first = closes[0]
    if (first === 0) return null
    return ((closes[closes.length - 1] - first) / first) * 100
})

interface RawDataRow {
    date: string
    open: number
    high: number
    low: number
    close: number
    volume: number
}

const rawDataRows = computed<RawDataRow[]>(() => {
    const { records } = priceHistoryData.value
    const rows = (records ?? []).map((r) => ({
        date: r.time,
        open: r.open,
        high: r.high,
        low: r.low,
        close: r.close,
        volume: r.volume
    }))
    return rows.sort((a, b) => b.date.localeCompare(a.date))
})

const dataDialogVisible = ref(false)
const dataDialogMode = ref<'add' | 'edit'>('add')
const dataDialogForm = ref<{
    origDate?: string
    date: string
    open: number
    high: number
    low: number
    close: number
    volume: number
}>({ date: '', open: 0, high: 0, low: 0, close: 0, volume: 0 })

function openAddDataDialog() {
    dataDialogMode.value = 'add'
    dataDialogForm.value = { date: '', open: 0, high: 0, low: 0, close: 0, volume: 0 }
    dataDialogVisible.value = true
}

function openEditDataDialog(record: RawDataRow) {
    dataDialogMode.value = 'edit'
    dataDialogForm.value = {
        origDate: record.date,
        date: record.date,
        open: record.open,
        high: record.high,
        low: record.low,
        close: record.close,
        volume: record.volume
    }
    dataDialogVisible.value = true
}

async function saveDataDialog() {
    const sym = instrument.value?.symbol
    if (!sym) return
    const f = dataDialogForm.value
    if (!f.date) {
        dataDialogVisible.value = false
        return
    }
    // Use date as-is (already local YYYY-MM-DD from picker); avoid UTC shift
    const time = f.date.includes('T') ? toLocalDateString(new Date(f.date)) : f.date
    const payload = {
        time,
        open: f.open,
        high: f.high,
        low: f.low,
        close: f.close,
        volume: f.volume
    }
    try {
        if (dataDialogMode.value === 'add') {
            await createPriceMutation(payload)
        } else {
            await updatePriceMutation({ origDate: f.origDate!, payload })
        }
        dataDialogVisible.value = false
        await refetchPriceHistory()
    } catch (_) {
        // Error surfaced by mutation / could add toast
    }
}

const deleteDialogVisible = ref(false)
const deleteTargetRecord = ref<RawDataRow | null>(null)

function openDeleteDataDialog(record: RawDataRow) {
    deleteTargetRecord.value = record
    deleteDialogVisible.value = true
}

async function confirmDeleteData() {
    const rec = deleteTargetRecord.value
    if (rec) {
        try {
            await deletePriceMutation(rec.date)
            deleteTargetRecord.value = null
            deleteDialogVisible.value = false
            await refetchPriceHistory()
        } catch (_) {
            // Error surfaced by mutation
        }
    }
}

function formatTableDate(isoDate: string) {
    return isoDate ? formatDate(isoDate) : '-'
}

// --- EPS (earnings per share) ----------------------------------------------
// EPS is quarterly, so we fetch the full history (no date range) and edit it directly.
const { data: epsHistoryData } = useEpsHistory(symbol)
const {
    createEps: createEpsMutation,
    updateEps: updateEpsMutation,
    deleteEps: deleteEpsMutation,
    isCreating: isEpsCreating,
    isUpdating: isEpsUpdating,
    isDeleting: isEpsDeleting
} = useEpsMutations(symbol)

interface EpsRow {
    date: string
    basic: number
    diluted: number
}

const epsRows = computed<EpsRow[]>(() => {
    const rows = epsHistoryData.value.map((r) => ({
        date: r.time,
        basic: r.eps_basic,
        diluted: r.eps_diluted
    }))
    return rows.sort((a, b) => b.date.localeCompare(a.date))
})

const epsDialogVisible = ref(false)
const epsDialogMode = ref<'add' | 'edit'>('add')
const epsDialogForm = ref<{ origDate?: string; date: string; basic: number; diluted: number }>({
    date: '',
    basic: 0,
    diluted: 0
})

function openAddEpsDialog() {
    epsDialogMode.value = 'add'
    epsDialogForm.value = { date: '', basic: 0, diluted: 0 }
    epsDialogVisible.value = true
}

function openEditEpsDialog(record: EpsRow) {
    epsDialogMode.value = 'edit'
    epsDialogForm.value = {
        origDate: record.date,
        date: record.date,
        basic: record.basic,
        diluted: record.diluted
    }
    epsDialogVisible.value = true
}

async function saveEpsDialog() {
    const sym = instrument.value?.symbol
    if (!sym) return
    const f = epsDialogForm.value
    if (!f.date) {
        epsDialogVisible.value = false
        return
    }
    const time = f.date.includes('T') ? toLocalDateString(new Date(f.date)) : f.date
    const payload = { time, eps_basic: f.basic, eps_diluted: f.diluted }
    try {
        if (epsDialogMode.value === 'add') {
            await createEpsMutation(payload)
        } else {
            await updateEpsMutation({ origDate: f.origDate!, payload })
        }
        epsDialogVisible.value = false
    } catch (_) {
        // Error surfaced by mutation
    }
}

const epsDeleteDialogVisible = ref(false)
const epsDeleteTarget = ref<EpsRow | null>(null)

function openDeleteEpsDialog(record: EpsRow) {
    epsDeleteTarget.value = record
    epsDeleteDialogVisible.value = true
}

async function confirmDeleteEps() {
    const rec = epsDeleteTarget.value
    if (rec) {
        try {
            await deleteEpsMutation(rec.date)
            epsDeleteTarget.value = null
            epsDeleteDialogVisible.value = false
        } catch (_) {
            // Error surfaced by mutation
        }
    }
}

function formatEps(value: number | null | undefined): string {
    if (value == null) return '-'
    return value.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

const chartOption = computed(() => {
    const { dates, closes } = priceHistory.value
    if (!dates.length || !instrument.value) return {}

    const inst = instrument.value
    const isPositive = (rangeChange.value ?? 0) >= 0
    const lineColor = isPositive ? '#22c55e' : '#ef4444'
    const areaColorTop = isPositive ? 'rgba(34,197,94,0.15)' : 'rgba(239,68,68,0.15)'
    const areaColorBottom = 'rgba(255,255,255,0)'

    return {
        animation: true,
        animationDuration: 400,
        grid: {
            left: '1%',
            right: '3%',
            bottom: '12%',
            top: '4%',
            containLabel: true
        },
        tooltip: {
            trigger: 'axis',
            formatter: (params: any) => {
                const p = params[0]
                return `<strong>${formatDate(p.axisValue ?? p.name ?? '')}</strong><br/>
                    Price: <strong>${p.value.toLocaleString('en-US', { minimumFractionDigits: 2 })}</strong> ${inst.currency || ''}`
            }
        },
        xAxis: {
            type: 'category',
            data: dates,
            axisLabel: {
                rotate: 45,
                fontSize: 11,
                formatter: (v: string) => formatDate(v)
            },
            boundaryGap: false
        },
        yAxis: {
            type: 'value',
            scale: true,
            axisLabel: {
                formatter: (v: number) => v.toFixed(0)
            },
            splitLine: {
                lineStyle: { type: 'dashed', opacity: 0.4 }
            }
        },
        series: [{
            type: 'line',
            data: closes,
            smooth: 0.3,
            showSymbol: false,
            lineStyle: { color: lineColor, width: 2 },
            itemStyle: { color: lineColor },
            areaStyle: {
                color: {
                    type: 'linear',
                    x: 0, y: 0, x2: 0, y2: 1,
                    colorStops: [
                        { offset: 0, color: areaColorTop },
                        { offset: 1, color: areaColorBottom }
                    ]
                }
            },
            markLine: {
                silent: true,
                symbol: 'none',
                lineStyle: { type: 'dashed', color: '#9ca3af', width: 1 },
                data: [
                    { yAxis: inst.lastPrice, label: { formatter: 'Current', position: 'end', fontSize: 10 } }
                ]
            }
        }]
    }
})

function goBack() {
    router.push({ name: 'stock-market' })
}
</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="p-3">
                <div class="header-nav">
                    <Button
                        icon="ti ti-arrow-left"
                        label="Back to Market"
                        text
                        severity="secondary"
                        @click="goBack"
                    />
                </div>

                <div v-if="isLoading" class="loading-state">
            <i class="ti ti-loader-2 spin-icon" style="font-size: 2rem"></i>
        </div>

        <div v-else-if="!instrument" class="empty-state">
            <Card>
                <template #content>
                    <div class="text-center p-4">
                        <i class="ti ti-alert-triangle" style="font-size: 2rem; color: var(--p-text-muted-color)"></i>
                        <p class="mt-3">Instrument not found.</p>
                        <Button label="Back to Market" @click="goBack" class="mt-2" />
                    </div>
                </template>
            </Card>
        </div>

        <template v-else>
            <Card>
                <template #title>
                    <div class="flex align-items-center gap-2">
                        <span class="font-bold">{{ instrument.symbol }}</span>
                        <span class="text-color-secondary font-normal">{{ instrument.name }}</span>
                    </div>
                </template>
                <template #content>
                    <TabView v-model:activeIndex="activeTabIndex">
                        <TabPanel header="Overview" value="overview">
                            <div class="instrument-content">
                                <div class="price-row">
                                    <span class="current-price">{{ formatPrice(instrument.lastPrice) }}</span>
                                    <span class="currency-label">{{ instrument.currency }}</span>
                                    <Tag
                                        :value="formatPct(rangeChangePct)"
                                        :severity="getChangeSeverity(rangeChangePct)"
                                        class="ml-2"
                                    />
                                    <span
                                        :class="(rangeChange ?? 0) >= 0 ? 'text-green-600' : 'text-red-600'"
                                        class="change-value ml-2"
                                    >
                                        {{ formatChange(rangeChange) }}
                                    </span>
                                </div>
                            </div>

                            <Divider />

                            <div class="chart-section">
                                <div class="chart-header">
                                    <h3 class="chart-title">Price History</h3>
                                    <SelectButton
                                        v-model="selectedRange"
                                        :options="dateRangeOptions"
                                        optionLabel="label"
                                        optionValue="value"
                                        class="range-selector"
                                    />
                                </div>
                                <VChart
                                    :option="chartOption"
                                    autoresize
                                    class="price-chart"
                                />
                            </div>
                        </TabPanel>

                        <TabPanel header="Chart" value="chart">
                            <StockChartTab v-if="symbol" :symbol="symbol" class="chart-tab-host" />
                        </TabPanel>

                        <TabPanel header="Raw data" value="raw-data">
                            <div class="raw-data-toolbar mb-3">
                                <Button label="Add" icon="ti ti-plus" :loading="isCreating" @click="openAddDataDialog" />
                            </div>
                            <DataTable
                                :value="rawDataRows"
                                stripedRows
                                :paginator="rawDataRows.length > 10"
                                :rows="10"
                                dataKey="date"
                                class="p-datatable-sm"
                            >
                                <Column field="date" header="Date">
                                    <template #body="{ data }">
                                        {{ formatTableDate(data.date) }}
                                    </template>
                                </Column>
                                <Column field="open" header="Open">
                                    <template #body="{ data }">
                                        {{ formatPrice(data.open) }}
                                    </template>
                                </Column>
                                <Column field="high" header="High">
                                    <template #body="{ data }">
                                        {{ formatPrice(data.high) }}
                                    </template>
                                </Column>
                                <Column field="low" header="Low">
                                    <template #body="{ data }">
                                        {{ formatPrice(data.low) }}
                                    </template>
                                </Column>
                                <Column field="close" header="Close">
                                    <template #body="{ data }">
                                        {{ formatPrice(data.close) }}
                                    </template>
                                </Column>
                                <Column field="volume" header="Volume">
                                    <template #body="{ data }">
                                        {{ formatVolume(data.volume) }}
                                    </template>
                                </Column>
                                <Column header="Actions" style="width: 120px">
                                    <template #body="{ data }">
                                        <Button
                                            icon="ti ti-pencil"
                                            text
                                            size="small"
                                            severity="secondary"
                                            :loading="isUpdating"
                                            @click="openEditDataDialog(data)"
                                        />
                                        <Button
                                            icon="ti ti-trash"
                                            text
                                            size="small"
                                            severity="danger"
                                            :loading="isDeleting"
                                            @click="openDeleteDataDialog(data)"
                                        />
                                    </template>
                                </Column>
                            </DataTable>
                        </TabPanel>

                        <TabPanel header="EPS" value="eps">
                            <div class="raw-data-toolbar mb-3">
                                <Button label="Add" icon="ti ti-plus" :loading="isEpsCreating" @click="openAddEpsDialog" />
                            </div>
                            <DataTable
                                :value="epsRows"
                                stripedRows
                                :paginator="epsRows.length > 10"
                                :rows="10"
                                dataKey="date"
                                class="p-datatable-sm"
                            >
                                <template #empty>
                                    <div class="text-center p-3 text-color-secondary">
                                        No EPS data. Add a filing or run the <code>eps-import</code> task.
                                    </div>
                                </template>
                                <Column field="date" header="Filing date">
                                    <template #body="{ data }">
                                        {{ formatTableDate(data.date) }}
                                    </template>
                                </Column>
                                <Column field="basic" header="Basic EPS">
                                    <template #body="{ data }">
                                        {{ formatEps(data.basic) }}
                                    </template>
                                </Column>
                                <Column field="diluted" header="Diluted EPS">
                                    <template #body="{ data }">
                                        {{ formatEps(data.diluted) }}
                                    </template>
                                </Column>
                                <Column header="Actions" style="width: 120px">
                                    <template #body="{ data }">
                                        <Button
                                            icon="ti ti-pencil"
                                            text
                                            size="small"
                                            severity="secondary"
                                            :loading="isEpsUpdating"
                                            @click="openEditEpsDialog(data)"
                                        />
                                        <Button
                                            icon="ti ti-trash"
                                            text
                                            size="small"
                                            severity="danger"
                                            :loading="isEpsDeleting"
                                            @click="openDeleteEpsDialog(data)"
                                        />
                                    </template>
                                </Column>
                            </DataTable>
                        </TabPanel>
                    </TabView>

                    <Dialog
                        v-model:visible="dataDialogVisible"
                        :header="dataDialogMode === 'add' ? 'Add market data' : 'Edit market data'"
                        modal
                        class="entry-dialog"
                        @hide="dataDialogVisible = false"
                    >
                        <div class="flex flex-column gap-3 py-2">
                            <div class="field">
                                <label for="data-date">Date</label>
                                <DatePicker
                                    :id="'data-date'"
                                    :modelValue="dataDialogForm.date ? new Date(dataDialogForm.date + 'T12:00:00') : null"
                                    @update:modelValue="(d: any) => { dataDialogForm.date = d ? toLocalDateString(d) : '' }"
                                    :dateFormat="pickerDateFormat"
                                    :disabled="dataDialogMode === 'edit'"
                                    showIcon
                                    class="w-full"
                                />
                                <small v-if="dataDialogMode === 'edit'" class="text-color-secondary">
                                    To change the date, delete this record and add a new one.
                                </small>
                            </div>
                            <div class="field">
                                <label for="data-open">Open</label>
                                <InputNumber
                                    id="data-open"
                                    v-model="dataDialogForm.open"
                                    mode="decimal"
                                    :minFractionDigits="2"
                                    :maxFractionDigits="2"
                                    :min="0"
                                    class="w-full"
                                />
                            </div>
                            <div class="field">
                                <label for="data-high">High</label>
                                <InputNumber
                                    id="data-high"
                                    v-model="dataDialogForm.high"
                                    mode="decimal"
                                    :minFractionDigits="2"
                                    :maxFractionDigits="2"
                                    :min="0"
                                    class="w-full"
                                />
                            </div>
                            <div class="field">
                                <label for="data-low">Low</label>
                                <InputNumber
                                    id="data-low"
                                    v-model="dataDialogForm.low"
                                    mode="decimal"
                                    :minFractionDigits="2"
                                    :maxFractionDigits="2"
                                    :min="0"
                                    class="w-full"
                                />
                            </div>
                            <div class="field">
                                <label for="data-close">Close</label>
                                <InputNumber
                                    id="data-close"
                                    v-model="dataDialogForm.close"
                                    mode="decimal"
                                    :minFractionDigits="2"
                                    :maxFractionDigits="2"
                                    :min="0"
                                    class="w-full"
                                />
                            </div>
                            <div class="field">
                                <label for="data-volume">Volume</label>
                                <InputNumber
                                    id="data-volume"
                                    v-model="dataDialogForm.volume"
                                    :minFractionDigits="0"
                                    :maxFractionDigits="0"
                                    :min="0"
                                    class="w-full"
                                />
                            </div>
                        </div>
                        <template #footer>
                            <Button label="Cancel" text severity="secondary" @click="dataDialogVisible = false" />
                            <Button label="Save" icon="ti ti-check" :loading="isCreating || isUpdating" @click="saveDataDialog" />
                        </template>
                    </Dialog>

                    <ConfirmDialog
                        v-model:visible="deleteDialogVisible"
                        title="Delete market data"
                        message="Remove this data point?"
                        @confirm="confirmDeleteData"
                    />

                    <Dialog
                        v-model:visible="epsDialogVisible"
                        :header="epsDialogMode === 'add' ? 'Add EPS filing' : 'Edit EPS filing'"
                        modal
                        class="entry-dialog"
                        @hide="epsDialogVisible = false"
                    >
                        <div class="flex flex-column gap-3 py-2">
                            <div class="field">
                                <label for="eps-date">Filing date</label>
                                <DatePicker
                                    :id="'eps-date'"
                                    :modelValue="epsDialogForm.date ? new Date(epsDialogForm.date + 'T12:00:00') : null"
                                    @update:modelValue="(d: any) => { epsDialogForm.date = d ? toLocalDateString(d) : '' }"
                                    :dateFormat="pickerDateFormat"
                                    :disabled="epsDialogMode === 'edit'"
                                    showIcon
                                    class="w-full"
                                />
                                <small v-if="epsDialogMode === 'edit'" class="text-color-secondary">
                                    To change the date, delete this filing and add a new one.
                                </small>
                            </div>
                            <div class="field">
                                <label for="eps-basic">Basic EPS</label>
                                <InputNumber
                                    id="eps-basic"
                                    v-model="epsDialogForm.basic"
                                    mode="decimal"
                                    :minFractionDigits="2"
                                    :maxFractionDigits="2"
                                    class="w-full"
                                />
                            </div>
                            <div class="field">
                                <label for="eps-diluted">Diluted EPS</label>
                                <InputNumber
                                    id="eps-diluted"
                                    v-model="epsDialogForm.diluted"
                                    mode="decimal"
                                    :minFractionDigits="2"
                                    :maxFractionDigits="2"
                                    class="w-full"
                                />
                            </div>
                        </div>
                        <template #footer>
                            <Button label="Cancel" text severity="secondary" @click="epsDialogVisible = false" />
                            <Button label="Save" icon="ti ti-check" :loading="isEpsCreating || isEpsUpdating" @click="saveEpsDialog" />
                        </template>
                    </Dialog>

                    <ConfirmDialog
                        v-model:visible="epsDeleteDialogVisible"
                        title="Delete EPS filing"
                        message="Remove this EPS filing?"
                        @confirm="confirmDeleteEps"
                    />
                </template>
            </Card>
        </template>
            </div>
        </template>
    </ResponsiveHorizontal>
</template>

<style scoped>
.header-nav {
    margin-bottom: 1rem;
}

.loading-state {
    display: flex;
    justify-content: center;
    padding: 4rem;
}

.instrument-content {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    flex-wrap: wrap;
}

.price-row {
    display: flex;
    align-items: center;
    gap: 0.25rem;
}

.current-price {
    font-size: 1.6rem;
    font-weight: 700;
}

.currency-label {
    font-size: 0.9rem;
    color: var(--p-text-muted-color);
    margin-left: 0.25rem;
}

.change-value {
    font-weight: 600;
    font-size: 0.95rem;
}

.chart-section {
    margin-top: 1rem;
}

.chart-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    flex-wrap: wrap;
    margin-bottom: 1rem;
}

.chart-title {
    font-size: 1rem;
    font-weight: 600;
    margin: 0;
    color: var(--p-text-color);
}

.range-selector {
    flex-shrink: 0;
}

.price-chart {
    height: 400px;
    width: 100%;
}

.chart-tab-host {
    display: flex;
    flex-direction: column;
    /* Definite height (not min-height) so the inner flex:1 chart-wrapper and the
       VChart's height:100% resolve; min-height is indefinite and collapses the chart. */
    height: 600px;
}

.raw-data-toolbar {
    display: flex;
    align-items: center;
}

.field label {
    display: block;
    font-weight: 600;
    margin-bottom: 0.35rem;
    font-size: 0.9rem;
}
</style>
