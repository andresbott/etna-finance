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
import ConfirmDialog from '@/components/common/confirmDialog.vue'
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
    toLocalDateString,
    formatPrice,
    formatPct,
    formatChange,
    getChangeSeverity,
    type MarketInstrument,
    type PriceHistoryRange
} from '@/composables/useMarketData'
import type { PriceRecord } from '@/lib/api/MarketData'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, LegendComponent, MarkLineComponent])

const route = useRoute()
const router = useRouter()
const { instruments, isLoading } = useMarketInstruments()
const { formatDate, pickerDateFormat } = useDateFormat()
const TAB_NAMES = ['overview', 'raw-data'] as const

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
    { value: '6m', label: '6M' },
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

const rawDataRows = computed(() => {
    const { records } = priceHistoryData.value
    const rows = (records ?? []).map((r: PriceRecord) => ({ id: r.id, date: r.time, price: r.price }))
    return rows.sort((a, b) => b.date.localeCompare(a.date))
})

const dataDialogVisible = ref(false)
const dataDialogMode = ref<'add' | 'edit'>('add')
const dataDialogForm = ref<{ id?: number; date: string; price: number }>({ date: '', price: 0 })
const dataDialogEditRecord = ref<{ id: number; date: string; price: number } | null>(null)

function openAddDataDialog() {
    dataDialogMode.value = 'add'
    dataDialogForm.value = { date: '', price: 0 }
    dataDialogVisible.value = true
}

function openEditDataDialog(record: { id: number; date: string; price: number }) {
    dataDialogMode.value = 'edit'
    dataDialogForm.value = { id: record.id, date: record.date, price: record.price }
    dataDialogEditRecord.value = record
    dataDialogVisible.value = true
}

async function saveDataDialog() {
    const sym = instrument.value?.symbol
    if (!sym) return
    const { date, price } = dataDialogForm.value
    if (!date) {
        dataDialogVisible.value = false
        return
    }
    // Use date as-is (already local YYYY-MM-DD from picker); avoid UTC shift
    const time = date.includes('T') ? toLocalDateString(new Date(date)) : date
    try {
        if (dataDialogMode.value === 'add') {
            await createPriceMutation({ time, price })
        } else {
            const id = dataDialogForm.value.id ?? dataDialogEditRecord.value?.id
            if (id != null) await updatePriceMutation({ id, payload: { time, price } })
        }
        dataDialogVisible.value = false
        await refetchPriceHistory()
    } catch (_) {
        // Error surfaced by mutation / could add toast
    }
}

const deleteDialogVisible = ref(false)
const deleteTargetRecord = ref<{ id: number; date: string; price: number } | null>(null)

function openDeleteDataDialog(record: { id: number; date: string; price: number }) {
    deleteTargetRecord.value = record
    deleteDialogVisible.value = true
}

async function confirmDeleteData() {
    const rec = deleteTargetRecord.value
    if (rec) {
        try {
            await deletePriceMutation(rec.id)
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

const chartOption = computed(() => {
    const { dates, prices } = priceHistory.value
    if (!dates.length || !instrument.value) return {}

    const inst = instrument.value
    const isPositive = (inst.change ?? 0) >= 0
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
            formatter: (params) => {
                const p = params[0]
                return `<strong>${p.axisValueLabel}</strong><br/>
                    Price: <strong>${p.value.toLocaleString('en-US', { minimumFractionDigits: 2 })}</strong> ${inst.currency || ''}`
            }
        },
        xAxis: {
            type: 'category',
            data: dates,
            axisLabel: {
                rotate: 45,
                fontSize: 11
            },
            boundaryGap: false
        },
        yAxis: {
            type: 'value',
            scale: true,
            axisLabel: {
                formatter: (v) => v.toFixed(0)
            },
            splitLine: {
                lineStyle: { type: 'dashed', opacity: 0.4 }
            }
        },
        series: [{
            type: 'line',
            data: prices,
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
                        icon="pi pi-arrow-left"
                        label="Back to Market"
                        text
                        severity="secondary"
                        @click="goBack"
                    />
                </div>

                <div v-if="isLoading" class="loading-state">
            <i class="pi pi-spinner pi-spin" style="font-size: 2rem"></i>
        </div>

        <div v-else-if="!instrument" class="empty-state">
            <Card>
                <template #content>
                    <div class="text-center p-4">
                        <i class="pi pi-exclamation-triangle" style="font-size: 2rem; color: var(--p-text-muted-color)"></i>
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
                        <TabPanel header="Overview">
                            <div class="instrument-content">
                                <div class="price-row">
                                    <span class="current-price">{{ formatPrice(instrument.lastPrice) }}</span>
                                    <span class="currency-label">{{ instrument.currency }}</span>
                                    <Tag
                                        :value="formatPct(instrument.changePct)"
                                        :severity="getChangeSeverity(instrument.changePct)"
                                        class="ml-2"
                                    />
                                    <span
                                        :class="(instrument.change ?? 0) >= 0 ? 'text-green-600' : 'text-red-600'"
                                        class="change-value ml-2"
                                    >
                                        {{ formatChange(instrument.change) }}
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

                        <TabPanel header="Raw data">
                            <div class="raw-data-toolbar mb-3">
                                <Button label="Add" icon="pi pi-plus" :loading="isCreating" @click="openAddDataDialog" />
                            </div>
                            <DataTable
                                :value="rawDataRows"
                                stripedRows
                                :paginator="rawDataRows.length > 10"
                                :rows="10"
                                dataKey="id"
                                class="p-datatable-sm"
                            >
                                <Column field="date" header="Date">
                                    <template #body="{ data }">
                                        {{ formatTableDate(data.date) }}
                                    </template>
                                </Column>
                                <Column field="price" header="Price">
                                    <template #body="{ data }">
                                        {{ formatPrice(data.price) }}
                                    </template>
                                </Column>
                                <Column header="Actions" style="width: 120px">
                                    <template #body="{ data }">
                                        <Button
                                            icon="pi pi-pencil"
                                            text
                                            size="small"
                                            severity="secondary"
                                            :loading="isUpdating"
                                            @click="openEditDataDialog(data)"
                                        />
                                        <Button
                                            icon="pi pi-trash"
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
                                    @update:modelValue="(d: Date | null) => { dataDialogForm.date = d ? toLocalDateString(d) : '' }"
                                    :dateFormat="pickerDateFormat"
                                    showIcon
                                    class="w-full"
                                />
                            </div>
                            <div class="field">
                                <label for="data-price">Price</label>
                                <InputNumber
                                    id="data-price"
                                    v-model="dataDialogForm.price"
                                    mode="decimal"
                                    :minFractionDigits="2"
                                    :maxFractionDigits="2"
                                    :min="0"
                                    class="w-full"
                                />
                            </div>
                        </div>
                        <template #footer>
                            <Button label="Cancel" text severity="secondary" @click="dataDialogVisible = false" />
                            <Button label="Save" icon="pi pi-check" :loading="isCreating || isUpdating" @click="saveDataDialog" />
                        </template>
                    </Dialog>

                    <ConfirmDialog
                        v-model:visible="deleteDialogVisible"
                        title="Delete market data"
                        message="Remove this data point?"
                        @confirm="confirmDeleteData"
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
