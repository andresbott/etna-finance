<script setup>
import { ResponsiveHorizontal } from '@go-bumbu/vue-layouts'
import '@go-bumbu/vue-layouts/dist/vue-layouts.css'
import { ref, computed } from 'vue'
import Card from 'primevue/card'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import {
    GridComponent,
    TooltipComponent,
    LegendComponent
} from 'echarts/components'
import TabView from 'primevue/tabview'
import TabPanel from 'primevue/tabpanel'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Message from 'primevue/message'
import { useSettingsStore } from '@/store/settingsStore'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, LegendComponent])

const settingsStore = useSettingsStore()
const mainCurrency = computed(() => settingsStore.mainCurrency || 'CHF')
const otherCurrencies = computed(() => {
    const all = settingsStore.currencies.length > 0 ? settingsStore.currencies : ['CHF']
    return all.filter(c => c !== mainCurrency.value)
})

const leftSidebarCollapsed = ref(true)

// Date range
const startDate = ref(new Date(new Date().setMonth(new Date().getMonth() - 3)))
const endDate = ref(new Date())

// Mock exchange rate data
const defaultBaseRates = { USD: 0.92, EUR: 1.08, GBP: 1.26, JPY: 0.0061, CHF: 1.0 }

const generateMockData = () => {
    const data = []
    const currencies = otherCurrencies.value
    
    const start = new Date(startDate.value)
    const end = new Date(endDate.value)
    
    for (let d = new Date(start); d <= end; d.setDate(d.getDate() + 1)) {
        currencies.forEach(currency => {
            const baseRate = defaultBaseRates[currency] ?? 1.0
            const variance = (Math.random() - 0.5) * 0.02
            const rate = baseRate * (1 + variance)
            data.push({
                date: new Date(d).toISOString().split('T')[0],
                currency,
                rate: rate.toFixed(4),
                change: (variance * 100).toFixed(2)
            })
        })
    }
    
    return data
}

const mockData = ref(generateMockData())

const getTextColor = () => {
    return getComputedStyle(document.documentElement).getPropertyValue('--text-color') || '#495057'
}

const getSurfaceColor = () => {
    return getComputedStyle(document.documentElement).getPropertyValue('--surface-border') || '#dfe7ef'
}

const chartColors = ['#22c55e', '#3b82f6', '#eab308', '#ef4444', '#8b5cf6', '#ec4899', '#14b8a6', '#f97316']

// ECharts option
const chartOption = computed(() => {
    const currencies = otherCurrencies.value

    const labels = [...new Set(mockData.value.map(d => d.date))].sort()

    const series = currencies.map((currency, idx) => {
        const color = chartColors[idx % chartColors.length]
        const currencyData = mockData.value
            .filter(d => d.currency === currency)
            .sort((a, b) => new Date(a.date) - new Date(b.date))

        return {
            name: `${mainCurrency.value}/${currency}`,
            type: 'line',
            smooth: 0.4,
            showSymbol: false,
            data: currencyData.map(d => parseFloat(d.rate)),
            lineStyle: { color },
            itemStyle: { color }
        }
    })

    const textColor = getTextColor()
    const surfaceColor = getSurfaceColor()

    return {
        animation: true,
        animationDuration: 500,
        grid: {
            left: '3%',
            right: '4%',
            bottom: '12%',
            top: '3%',
            containLabel: true
        },
        tooltip: {
            trigger: 'axis',
            formatter: (params) => {
                let result = `<strong>${params[0].axisValueLabel}</strong><br/>`
                for (const p of params) {
                    result += `${p.marker} ${p.seriesName}: ${p.value.toFixed(4)}<br/>`
                }
                return result
            }
        },
        legend: {
            bottom: 0,
            textStyle: { color: textColor }
        },
        xAxis: {
            type: 'category',
            data: labels,
            axisLabel: {
                color: textColor,
                rotate: 45
            },
            axisLine: { lineStyle: { color: surfaceColor } },
            splitLine: { lineStyle: { color: surfaceColor } }
        },
        yAxis: {
            type: 'value',
            axisLabel: { color: textColor },
            axisLine: { lineStyle: { color: surfaceColor } },
            splitLine: { lineStyle: { color: surfaceColor } }
        },
        series
    }
})

// Data by currency for tables (dynamic)
const currencyTableData = computed(() => {
    const result = {}
    for (const currency of otherCurrencies.value) {
        result[currency] = mockData.value
            .filter(d => d.currency === currency)
            .sort((a, b) => new Date(b.date) - new Date(a.date))
    }
    return result
})

const formatChange = (value) => {
    const num = parseFloat(value)
    return num >= 0 ? `+${value}%` : `${value}%`
}

const getChangeClass = (value) => {
    return parseFloat(value) >= 0 ? 'text-green-600' : 'text-red-600'
}
</script>

<template>
    <ResponsiveHorizontal :leftSidebarCollapsed="leftSidebarCollapsed">
        <template #default>
            <div class="p-3">
                <!-- Mock UI Warning -->
                <Message severity="error" :closable="false" class="warning-message">
                    <div class="warning-content">
                        <i class="pi pi-exclamation-triangle"></i>
                        <strong>This is a mock UI only.</strong> Exchange rate data is simulated and not real market data.
                    </div>
                </Message>

                <!-- Date Range Picker -->
                <div class="mb-4 flex justify-content-center">
                    <DateRangePicker
                        v-model:startDate="startDate"
                        v-model:endDate="endDate"
                        startLabel="From:"
                        endLabel="To:"
                    />
                </div>

                <div class="grid">
                    <!-- Exchange Rate Chart -->
                    <div class="col-12">
                        <Card>
                            <template #title>
                                <div class="flex align-items-center gap-2">
                                    <i class="pi pi-chart-line"></i>
                                    <span>Exchange Rate Trends</span>
                                </div>
                            </template>
                            <template #content>
                                <VChart
                                    :option="chartOption"
                                    autoresize
                                    style="height: 400px"
                                />
                            </template>
                        </Card>
                    </div>

                    <!-- Exchange Rate Tables -->
                    <div class="col-12">
                        <Card>
                            <template #title>
                                <div class="flex align-items-center gap-2">
                                    <i class="pi pi-table"></i>
                                    <span>Exchange Rate Details</span>
                                </div>
                            </template>
                            <template #content>
                                <TabView>
                                    <TabPanel v-for="currency in otherCurrencies" :key="currency" :header="currency">
                                        <DataTable :value="currencyTableData[currency] || []" stripedRows :paginator="true" :rows="10">
                                            <Column field="date" header="Date" sortable style="min-width: 150px"></Column>
                                            <Column field="rate" :header="`${mainCurrency}/${currency} Rate`" sortable style="min-width: 150px"></Column>
                                            <Column field="change" header="Change %" sortable style="min-width: 120px">
                                                <template #body="slotProps">
                                                    <span :class="getChangeClass(slotProps.data.change)">
                                                        {{ formatChange(slotProps.data.change) }}
                                                    </span>
                                                </template>
                                            </Column>
                                        </DataTable>
                                    </TabPanel>
                                </TabView>
                            </template>
                        </Card>
                    </div>
                </div>
            </div>
        </template>
    </ResponsiveHorizontal>
</template>

<style scoped>
.card {
    height: 100%;
}

.warning-message {
    margin-bottom: 2rem;
}

.warning-message :deep(.p-message-wrapper) {
    padding: 1rem 1.25rem;
}

.warning-content {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    font-size: 1rem;
}

.warning-content i {
    font-size: 1.25rem;
}

.warning-content strong {
    margin-right: 0.25rem;
}
</style>
