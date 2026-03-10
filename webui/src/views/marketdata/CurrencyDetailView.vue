<script setup lang="ts">
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Button from 'primevue/button'
import Card from 'primevue/card'
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
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import {
    GridComponent,
    TooltipComponent,
    LegendComponent
} from 'echarts/components'
import ConfirmDialog from '@/components/common/confirmDialog.vue'
import { useSettingsStore } from '@/store/settingsStore'
import { useDateFormat } from '@/composables/useDateFormat'
import { toLocalDateString } from '@/composables/useMarketData'
import {
    useRateHistory,
    useFXMutations,
    formatPct,
    getChangeSeverity
} from '@/composables/useCurrencyRates'
import type { RateRecord } from '@/lib/api/CurrencyRates'
import type { RateHistoryRange } from '@/composables/useCurrencyRates'


use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, LegendComponent])

const route = useRoute()
const router = useRouter()
const { formatDate, pickerDateFormat } = useDateFormat()
const settingsStore = useSettingsStore()
const mainCurrency = computed(() => settingsStore.mainCurrency || 'CHF')

const TAB_NAMES = ['overview', 'raw-data']
const leftSidebarCollapsed = ref(true)
const activeTabIndex = ref(0)

const currency = computed(() => (String(route.params.currency || '')).toUpperCase())

function tabFromRoute() {
    const tab = String(route.params.tab || '')
    return TAB_NAMES.indexOf(tab) >= 0 ? TAB_NAMES.indexOf(tab) : 0
}

watch(
    () => route.params.tab,
    () => { activeTabIndex.value = tabFromRoute() },
    { immediate: true }
)

watch(activeTabIndex, (index) => {
    const tab = TAB_NAMES[index] ?? 'overview'
    if (route.params.tab !== tab) {
        router.replace({
            name: 'currency-detail',
            params: { currency: currency.value, tab }
        })
    }
})

const dateRangeOptions = [
    { value: '6m', label: '6M' },
    { value: 'max', label: 'Max' }
]
const selectedRange = ref<RateHistoryRange>('6m')

const { data: rateHistoryData, refetch: refetchRateHistory } = useRateHistory(
    mainCurrency,
    currency,
    selectedRange
)
const {
    createRate: createRateMutation,
    updateRate: updateRateMutation,
    deleteRate: deleteRateMutation,
    isCreating,
    isUpdating,
    isDeleting
} = useFXMutations(mainCurrency, currency)

const pairLabel = computed(() => `${mainCurrency.value}/${currency.value}`)

const priceHistory = computed(() => rateHistoryData.value)

const currentRate = computed(() => {
    const p = priceHistory.value?.prices
    return p?.length ? p[p.length - 1] : 0
})
const currentChange = computed(() => {
    const p = priceHistory.value?.prices ?? []
    if (p.length < 2) return 0
    const prev = p[p.length - 2]
    const curr = p[p.length - 1]
    return prev !== 0 ? ((curr - prev) / prev) * 100 : 0
})

const getTextColor = () => getComputedStyle(document.documentElement).getPropertyValue('--text-color') || '#495057'
const getSurfaceColor = () => getComputedStyle(document.documentElement).getPropertyValue('--surface-border') || '#dfe7ef'

const chartOption = computed(() => {
    const d = priceHistory.value?.dates ?? []
    const p = priceHistory.value?.prices ?? []
    if (!d.length || !p.length) return {}
    const isPositive = currentChange.value >= 0
    const lineColor = isPositive ? '#22c55e' : '#ef4444'
    const textColor = getTextColor()
    const surfaceColor = getSurfaceColor()
    return {
        animation: true,
        animationDuration: 400,
        grid: {
            left: '3%',
            right: '4%',
            bottom: '12%',
            top: '3%',
            containLabel: true
        },
        tooltip: {
            trigger: 'axis',
            formatter: (params: any) => {
                const x = params[0]
                return `<strong>${x.axisValueLabel}</strong><br/>${pairLabel.value}: <strong>${Number(x.value).toFixed(4)}</strong>`
            }
        },
        xAxis: {
            type: 'category',
            data: d,
            axisLabel: { color: textColor, rotate: 45 },
            axisLine: { lineStyle: { color: surfaceColor } },
            splitLine: { lineStyle: { color: surfaceColor } }
        },
        yAxis: {
            type: 'value',
            axisLabel: { color: textColor },
            axisLine: { lineStyle: { color: surfaceColor } },
            splitLine: { lineStyle: { color: surfaceColor } }
        },
        series: [{
            type: 'line',
            data: p,
            smooth: 0.3,
            showSymbol: false,
            lineStyle: { color: lineColor, width: 2 },
            itemStyle: { color: lineColor }
        }]
    }
})

const rawDataRows = computed(() => {
    const recs = priceHistory.value?.records ?? []
    return [...recs].sort((a, b) => b.time.localeCompare(a.time))
})

const dataDialogVisible = ref(false)
const dataDialogMode = ref<'add' | 'edit'>('add')
const dataDialogForm = ref({ date: '', rate: 0 })
const dataDialogEditRecord = ref<RateRecord | null>(null)

function openAddDataDialog() {
    dataDialogMode.value = 'add'
    dataDialogForm.value = { date: toLocalDateString(new Date()), rate: 0 }
    dataDialogVisible.value = true
}

function openEditDataDialog(record: RateRecord) {
    dataDialogMode.value = 'edit'
    dataDialogForm.value = { date: record.time, rate: record.rate }
    dataDialogEditRecord.value = record
    dataDialogVisible.value = true
}

async function saveDataDialog() {
    const { date, rate } = dataDialogForm.value
    if (!date) {
        dataDialogVisible.value = false
        return
    }
    const time = date.includes('T') ? toLocalDateString(new Date(date)) : date
    try {
        if (dataDialogMode.value === 'add') {
            await createRateMutation({ time, rate })
        } else {
            const id = dataDialogEditRecord.value?.id
            if (id != null) await updateRateMutation({ id, payload: { time, rate } })
        }
        dataDialogVisible.value = false
        await refetchRateHistory()
    } catch (_) {}
}

const deleteDialogVisible = ref(false)
const deleteTargetRecord = ref<RateRecord | null>(null)

function openDeleteDataDialog(record: RateRecord) {
    deleteTargetRecord.value = record
    deleteDialogVisible.value = true
}

async function confirmDeleteData() {
    const rec = deleteTargetRecord.value
    if (rec) {
        try {
            await deleteRateMutation(rec.id)
            deleteTargetRecord.value = null
            deleteDialogVisible.value = false
            await refetchRateHistory()
        } catch (_) {}
    }
}

function formatTableDate(isoDate: string) {
    return isoDate ? formatDate(isoDate) : '-'
}

function goBack() {
    router.push({ name: 'currency-exchange' })
}
</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="p-3">
                <div class="header-nav">
                    <Button
                        icon="pi pi-arrow-left"
                        label="Back to Currency Exchange"
                        text
                        severity="secondary"
                        @click="goBack"
                    />
                </div>

                <div v-if="!currency" class="empty-state">
                    <Card>
                        <template #content>
                            <div class="text-center p-4">
                                <i class="pi pi-exclamation-triangle" style="font-size: 2rem; color: var(--p-text-muted-color)"></i>
                                <p class="mt-3">Currency not specified.</p>
                                <Button label="Back to Currency Exchange" @click="goBack" class="mt-2" />
                            </div>
                        </template>
                    </Card>
                </div>

                <template v-else>
                    <Card>
                        <template #title>
                            <div class="flex align-items-center gap-2">
                                <span class="font-bold">{{ currency }}</span>
                                <span class="text-color-secondary font-normal">{{ pairLabel }}</span>
                            </div>
                        </template>
                        <template #content>
                            <TabView v-model:activeIndex="activeTabIndex">
                                <TabPanel header="Overview" value="overview">
                                    <div class="instrument-content">
                                        <div class="price-row">
                                            <span class="current-price">{{ currentRate.toFixed(4) }}</span>
                                            <span class="currency-label">{{ pairLabel }}</span>
                                            <Tag
                                                :value="formatPct(currentChange)"
                                                :severity="getChangeSeverity(currentChange)"
                                                class="ml-2"
                                            />
                                        </div>
                                    </div>

                                    <Divider />

                                    <div class="chart-section">
                                        <div class="chart-header">
                                            <h3 class="chart-title">Exchange rate history</h3>
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

                                <TabPanel header="Raw data" value="raw-data">
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
                                        <Column field="time" header="Date">
                                            <template #body="{ data }">
                                                {{ formatTableDate(data.time) }}
                                            </template>
                                        </Column>
                                        <Column field="rate" :header="`Rate (${pairLabel})`">
                                            <template #body="{ data }">
                                                {{ data.rate.toFixed(4) }}
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
                                    <p class="text-color-secondary text-sm mt-2 mb-0">
                                        <i class="pi pi-info-circle"></i> Use + to add a rate. Edit or delete from the row actions.
                                    </p>
                                </TabPanel>
                            </TabView>
                        </template>
                    </Card>

                    <Dialog
                        v-model:visible="dataDialogVisible"
                        :header="dataDialogMode === 'add' ? 'Add exchange rate' : 'Edit exchange rate'"
                        modal
                        class="entry-dialog"
                        @hide="dataDialogVisible = false"
                    >
                        <div class="flex flex-column gap-3 py-2">
                            <div class="field">
                                <label for="fx-data-date">Date</label>
                                <DatePicker
                                    id="fx-data-date"
                                    :modelValue="dataDialogForm.date ? new Date(dataDialogForm.date + 'T12:00:00') : null"
                                    @update:modelValue="(d: any) => { dataDialogForm.date = d ? toLocalDateString(d) : '' }"
                                    :dateFormat="pickerDateFormat"
                                    showIcon
                                    class="w-full"
                                />
                            </div>
                            <div class="field">
                                <label for="fx-data-rate">Rate</label>
                                <InputNumber
                                    id="fx-data-rate"
                                    v-model="dataDialogForm.rate"
                                    mode="decimal"
                                    :minFractionDigits="4"
                                    :maxFractionDigits="6"
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
                        title="Delete exchange rate"
                        message="Remove this rate?"
                        @confirm="confirmDeleteData"
                    />
                </template>
            </div>
        </template>
    </ResponsiveHorizontal>
</template>

<style scoped>
.header-nav {
    margin-bottom: 1rem;
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
