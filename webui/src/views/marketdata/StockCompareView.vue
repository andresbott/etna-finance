<script setup lang="ts">
import { ResponsiveHorizontal } from '@/components/layout'
import { ref, computed, watch, watchEffect } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import Card from 'primevue/card'
import Button from 'primevue/button'
import Message from 'primevue/message'
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
import CompareToolbar from '@/components/marketdata/CompareToolbar.vue'
import { useCompareSelection } from '@/store/compareSelection'
import { useMarketInstruments } from '@/composables/useMarketData'
import { useCompareSeries } from '@/composables/useCompareSeries'
import { useCompareChart, type CompareView, type CompareChartSeries } from '@/composables/useCompareChart'
import { useCompareStats } from '@/composables/useCompareStats'
import CompareStatsTable from '@/components/marketdata/CompareStatsTable.vue'
import type { PriceHistoryRange } from '@/utils/dateRange'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, LegendComponent, MarkLineComponent])

const router = useRouter()
const route = useRoute()
const compare = useCompareSelection()
const { instruments } = useMarketInstruments()

// The selection lives in the URL (`?ids=15,16,17`) so the comparison is shareable.
function parseIds(raw: unknown): number[] {
    const str = Array.isArray(raw) ? raw.join(',') : (raw ?? '')
    return [
        ...new Set(
            String(str)
                .split(',')
                .map((s) => Number(s.trim()))
                .filter((n) => Number.isInteger(n) && n > 0)
        )
    ]
}
const selectedIds = computed(() => parseIds(route.query.ids))
const canCompare = computed(() => selectedIds.value.length >= 2)

// Mirror the URL into the store so "Back to Market" keeps the checkboxes selected,
// including when arriving via a shared link with an empty store.
watchEffect(() => {
    compare.setSelection(selectedIds.value)
})

const leftSidebarCollapsed = ref(true)

const view = ref<CompareView>('price')
const period = ref(50)
const selectedRange = ref<PriceHistoryRange>('6m')

// Each indicator view has its own sensible default period; switching view resets it.
const VIEW_DEFAULT_PERIOD: Record<CompareView, number> = { price: 0, sma: 50, ema: 20, rsi: 14 }
watch(view, (v) => {
    if (v !== 'price') period.value = VIEW_DEFAULT_PERIOD[v]
})

const selectedInstruments = computed(() =>
    instruments.value.filter((i) => selectedIds.value.includes(i.id))
)
const symbols = computed(() => selectedInstruments.value.map((i) => i.symbol))
const warmupDays = computed(() => (view.value === 'price' ? 0 : period.value * 2))

const { data: series, visibleStartDate, isLoading, isError } = useCompareSeries(
    symbols,
    selectedRange,
    warmupDays
)

const currencyBySymbol = computed<Record<string, string>>(() => {
    const m: Record<string, string> = {}
    for (const i of selectedInstruments.value) m[i.symbol] = i.currency
    return m
})

const chartInput = computed<CompareChartSeries[]>(() =>
    series.value.map((s) => ({
        symbol: s.symbol,
        currency: currencyBySymbol.value[s.symbol] ?? '',
        dates: s.ohlcv.dates,
        closes: s.ohlcv.closes
    }))
)

const { chartOption } = useCompareChart(chartInput, view, period, visibleStartDate)

const { rows: statsRows } = useCompareStats(series, visibleStartDate, currencyBySymbol)

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

                <Card v-if="!canCompare">
                    <template #content>
                        <div class="empty-state">
                            <i class="ti ti-git-compare" style="font-size: 2rem; color: var(--p-text-muted-color)"></i>
                            <p class="mt-3">Select at least two instruments to compare.</p>
                            <Button label="Back to Market" @click="goBack" class="mt-2" />
                        </div>
                    </template>
                </Card>

                <Card v-else>
                    <template #title>
                        <div class="flex align-items-center gap-2">
                            <i class="ti ti-git-compare text-primary"></i>
                            <span class="font-bold">Compare</span>
                            <span class="text-color-secondary font-normal">{{ symbols.join(', ') }}</span>
                        </div>
                    </template>
                    <template #content>
                        <CompareToolbar
                            v-model:view="view"
                            v-model:period="period"
                            v-model:range="selectedRange"
                        />

                        <div v-if="isLoading" class="chart-state">
                            <i class="ti ti-loader-2 spin-icon" style="font-size: 2rem"></i>
                        </div>
                        <Message v-else-if="isError" severity="error" :closable="false">
                            Failed to load comparison data.
                        </Message>
                        <div v-else>
                            <!-- notMerge so switching view (e.g. RSI->Price) fully replaces the
                                 option; otherwise ECharts keeps the old yAxis min/max and markLine. -->
                            <div class="chart-wrapper">
                                <VChart
                                    :option="chartOption"
                                    :update-options="{ notMerge: true }"
                                    autoresize
                                    class="compare-chart"
                                />
                            </div>
                            <CompareStatsTable :rows="statsRows" class="mt-4" />
                        </div>
                    </template>
                </Card>
            </div>
        </template>
    </ResponsiveHorizontal>
</template>

<style scoped>
.header-nav {
    margin-bottom: 1rem;
}

.empty-state {
    text-align: center;
    padding: 2rem;
    color: var(--p-text-muted-color);
}

.chart-state {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 600px;
}

.chart-wrapper {
    height: 600px;
    width: 100%;
}

.compare-chart {
    height: 100%;
    width: 100%;
}
</style>
