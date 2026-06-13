// webui/src/composables/useCompareChart.ts
import { computed, type Ref } from 'vue'
import { computeSMA, computeEMA, computeRSI } from '@/composables/useIndicators'
import { useDateFormat } from '@/composables/useDateFormat'

export type CompareView = 'price' | 'sma' | 'ema' | 'rsi'

export interface CompareChartSeries {
    symbol: string
    currency: string
    dates: string[]   // full, warmup-inclusive
    closes: number[]  // full, warmup-inclusive
}

export const COMPARE_PALETTE = [
    '#3b82f6', '#ef4444', '#22c55e', '#f59e0b', '#a855f7',
    '#06b6d4', '#ec4899', '#84cc16', '#f97316', '#6366f1'
]

function seriesValues(closes: number[], view: CompareView, period: number): (number | null)[] {
    switch (view) {
        case 'price': return closes
        case 'sma': return computeSMA(closes, period)
        case 'ema': return computeEMA(closes, period)
        case 'rsi': return computeRSI(closes, period)
    }
}

function trimIndex(dates: string[], visibleStartDate: string): number {
    if (!visibleStartDate) return 0
    const idx = dates.findIndex((d) => d >= visibleStartDate)
    return idx < 0 ? 0 : idx
}

export function buildCompareChartOption(
    series: CompareChartSeries[],
    view: CompareView,
    period: number,
    visibleStartDate: string,
    formatDate: (date: string) => string = (d) => d
) {
    const isRsi = view === 'rsi'

    // Compute the active metric on the full (warmup-inclusive) closes, then trim the
    // warmup prefix so indicators are valid at the visible left edge.
    const trimmed = series.map((s) => {
        const values = seriesValues(s.closes, view, period)
        const i = trimIndex(s.dates, visibleStartDate)
        return {
            symbol: s.symbol,
            currency: s.currency,
            dates: i > 0 ? s.dates.slice(i) : s.dates,
            values: i > 0 ? values.slice(i) : values
        }
    })

    // X axis = sorted union of every series' dates.
    const dateSet = new Set<string>()
    for (const s of trimmed) for (const d of s.dates) dateSet.add(d)
    const axisDates = [...dateSet].sort()

    const chartSeries = trimmed.map((s, idx) => {
        const byDate = new Map<string, number | null>()
        for (let k = 0; k < s.dates.length; k++) byDate.set(s.dates[k], s.values[k])
        const color = COMPARE_PALETTE[idx % COMPARE_PALETTE.length]
        const out: any = {
            name: s.symbol,
            type: 'line',
            data: axisDates.map((d) => (byDate.has(d) ? byDate.get(d)! : null)),
            showSymbol: false,
            smooth: 0.2,
            connectNulls: false,
            lineStyle: { width: 2, color },
            itemStyle: { color }
        }
        return out
    })

    if (isRsi && chartSeries.length > 0) {
        chartSeries[0].markLine = {
            silent: true,
            symbol: 'none',
            lineStyle: { type: 'dashed', color: '#9ca3af', width: 1 },
            data: [{ yAxis: 30 }, { yAxis: 70 }]
        }
    }

    return {
        animation: true,
        animationDuration: 400,
        grid: { left: '1%', right: '3%', bottom: '12%', top: '10%', containLabel: true },
        legend: { type: 'scroll', top: 0 },
        tooltip: {
            trigger: 'axis',
            formatter: (params: any[]) => {
                const header = params.length
                    ? `<strong>${formatDate(params[0].axisValue ?? params[0].name ?? '')}</strong>`
                    : ''
                const lines = params
                    .filter((p) => p.value != null)
                    .map((p) => {
                        const cur = trimmed.find((t) => t.symbol === p.seriesName)?.currency ?? ''
                        const v = Number(p.value).toLocaleString('en-US', {
                            minimumFractionDigits: 2,
                            maximumFractionDigits: 2
                        })
                        const suffix = view === 'price' ? ` ${cur}` : ''
                        return `${p.marker}${p.seriesName}: <strong>${v}</strong>${suffix}`
                    })
                return [header, ...lines].join('<br/>')
            }
        },
        xAxis: {
            type: 'category',
            data: axisDates,
            boundaryGap: false,
            axisLabel: { rotate: 45, fontSize: 11, formatter: (v: string) => formatDate(v) }
        },
        yAxis: {
            type: 'value',
            scale: !isRsi,
            ...(isRsi ? { min: 0, max: 100 } : {}),
            splitLine: { lineStyle: { type: 'dashed', opacity: 0.4 } }
        },
        series: chartSeries
    }
}

export function useCompareChart(
    series: Ref<CompareChartSeries[]>,
    view: Ref<CompareView>,
    period: Ref<number>,
    visibleStartDate: Ref<string>
) {
    const { formatDate } = useDateFormat()
    const chartOption = computed(() =>
        buildCompareChartOption(series.value, view.value, period.value, visibleStartDate.value, formatDate)
    )
    return { chartOption }
}
