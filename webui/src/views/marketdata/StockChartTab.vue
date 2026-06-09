<!-- webui/src/views/marketdata/StockChartTab.vue -->
<script setup lang="ts">
import { computed, ref, watch, onMounted, onUnmounted } from 'vue'
import { storeToRefs } from 'pinia'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { CandlestickChart, LineChart, BarChart } from 'echarts/charts'
import {
    GridComponent,
    TooltipComponent,
    DataZoomComponent,
    MarkLineComponent,
    LegendComponent
} from 'echarts/components'
import ChartToolbar from '@/components/marketdata/ChartToolbar.vue'
import { useTsdbPrices, type OhlcvData } from '@/composables/useTsdbPrices'
import { useIndicators, type IndicatorParams } from '@/composables/useIndicators'
import { useCandlestickChart } from '@/composables/useCandlestickChart'
import { useChartControls } from '@/composables/useChartControls'

use([
    CanvasRenderer, CandlestickChart, LineChart, BarChart,
    GridComponent, TooltipComponent, DataZoomComponent, MarkLineComponent, LegendComponent
])

const props = defineProps<{
    symbol: string
}>()

const store = useChartControls()
const { selectedRange, indicatorParams, pe } = storeToRefs(store)

/** Largest indicator lookback period (in trading days). */
function maxWarmupPeriod(p: IndicatorParams): number {
    let max = 0
    if (p.sma.enabled) {
        max = Math.max(max, p.sma.period1)
        if (p.sma.showSecond) max = Math.max(max, p.sma.period2)
    }
    if (p.ema.enabled) {
        max = Math.max(max, p.ema.period1)
        if (p.ema.showSecond) max = Math.max(max, p.ema.period2)
    }
    if (p.bollinger.enabled) max = Math.max(max, p.bollinger.period)
    if (p.rsi.enabled) max = Math.max(max, p.rsi.period + 1)
    if (p.macd.enabled) max = Math.max(max, p.macd.slow + p.macd.signal)
    return max
}

// Convert trading days to calendar days (×2 covers weekends/holidays generously)
const warmupCalendarDays = computed(() => maxWarmupPeriod(indicatorParams.value) * 2)

const symbolRef = computed(() => props.symbol)
const { data: fullOhlcv, visibleStartDate, isLoading, isError } = useTsdbPrices(symbolRef, selectedRange, warmupCalendarDays)

// Compute indicators on full dataset (including warmup prefix)
const fullCloses = computed(() => fullOhlcv.value.closes)
const indicatorData = useIndicators(fullCloses, indicatorParams)

// Find trim index: first date >= visibleStartDate
const trimIndex = computed(() => {
    const vsd = visibleStartDate.value
    if (!vsd) return 0
    const idx = fullOhlcv.value.dates.findIndex(d => d >= vsd)
    return idx < 0 ? 0 : idx
})

function sliceArray<T>(arr: T[], from: number): T[] {
    return from > 0 ? arr.slice(from) : arr
}

// Trim warmup prefix from OHLCV data
const ohlcv = computed<OhlcvData>(() => {
    const f = fullOhlcv.value
    const i = trimIndex.value
    return {
        dates: sliceArray(f.dates, i),
        opens: sliceArray(f.opens, i),
        highs: sliceArray(f.highs, i),
        lows: sliceArray(f.lows, i),
        closes: sliceArray(f.closes, i),
        volumes: sliceArray(f.volumes, i),
        points: sliceArray(f.points, i)
    }
})

// P/E ratio — not available in etna (no fundamentals endpoint); placeholders keep the
// chart composable's optional P/E grid dormant.
const peRatio = computed<(number | null)[]>(() => [])
const peEnabled = computed(() => false)

// Trim warmup prefix from indicator data
const indicators = computed(() => {
    const i = trimIndex.value
    const bb = indicatorData.bollinger.value
    const m = indicatorData.macd.value
    return {
        sma1: sliceArray(indicatorData.sma1.value, i),
        sma2: sliceArray(indicatorData.sma2.value, i),
        ema1: sliceArray(indicatorData.ema1.value, i),
        ema2: sliceArray(indicatorData.ema2.value, i),
        bollinger: {
            middle: sliceArray(bb.middle, i),
            upper: sliceArray(bb.upper, i),
            lower: sliceArray(bb.lower, i)
        },
        rsi: sliceArray(indicatorData.rsi.value, i),
        macd: {
            macd: sliceArray(m.macd, i),
            signal: sliceArray(m.signal, i),
            histogram: sliceArray(m.histogram, i)
        }
    }
})

// Read the wrapper's actual rendered height (set by flex layout) for internal grid calculations
const rootRef = ref<HTMLElement | null>(null)
const chartHeight = ref(500)

function measure() {
    const el = rootRef.value
    if (!el) return
    const wrapper = el.querySelector('.chart-wrapper') as HTMLElement | null
    if (!wrapper) return
    const h = wrapper.clientHeight
    if (h > 0) chartHeight.value = h
}

let rafId = 0
function onResize() {
    cancelAnimationFrame(rafId)
    rafId = requestAnimationFrame(measure)
}

onMounted(() => {
    window.addEventListener('resize', onResize)
})
onUnmounted(() => {
    window.removeEventListener('resize', onResize)
    cancelAnimationFrame(rafId)
})

// Measure whenever the chart wrapper appears (data loaded)
watch(() => ohlcv.value.dates.length, (len) => {
    if (len > 0) requestAnimationFrame(measure)
})

const { chartOption } = useCandlestickChart(ohlcv, indicators, indicatorParams, chartHeight, peRatio, peEnabled)
</script>

<template>
    <div ref="rootRef" class="stock-chart-tab">
        <ChartToolbar />

        <div v-if="isLoading" class="chart-loading">
            <i class="ti ti-loader-2 spin-icon" style="font-size: 2rem"></i>
        </div>

        <div v-else-if="isError" class="chart-error">
            <p>Failed to load chart data.</p>
        </div>

        <div v-else-if="ohlcv.dates.length === 0" class="chart-empty">
            <p>No OHLCV data available for this symbol and date range.</p>
        </div>

        <div v-else class="chart-wrapper">
            <VChart
                :option="chartOption"
                autoresize
                class="candlestick-chart"
            />
        </div>
    </div>
</template>

<style scoped>
.stock-chart-tab {
    padding-top: 0.5rem;
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
}

.chart-wrapper {
    overflow: hidden;
    flex: 1;
    min-height: 0;
}

.candlestick-chart {
    width: 100%;
    height: 100%;
}

.chart-loading,
.chart-error,
.chart-empty {
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 300px;
    color: var(--p-text-muted-color);
}
</style>
