// webui/src/composables/useCompareStats.ts
import { computed, type Ref } from 'vue'
import type { CompareSeries } from '@/composables/useCompareSeries'

export interface CompareStatsRow {
    symbol: string
    currency: string
    open: number
    close: number
    change: number // close - open
    changePct: number // (close - open) / open * 100
    high: number
    low: number
    maxDrawdownPct: number // <= 0
    volatilityPct: number // stdev of daily returns * 100
    avgVolume: number
}

// First index whose date is on/after the visible start. Mirrors useCompareChart's
// trimIndex so stats are computed over the visible range only (indicator views add
// a warmup prefix before visibleStartDate). When no date is on/after the visible
// start the symbol has no visible data, so we return dates.length to yield an empty
// slice (which omits the symbol downstream).
function trimIndex(dates: string[], visibleStartDate: string): number {
    if (!visibleStartDate) return 0
    const idx = dates.findIndex((d) => d >= visibleStartDate)
    return idx < 0 ? dates.length : idx
}

function maxDrawdownPct(closes: number[]): number {
    if (closes.length < 2) return 0
    let peak = closes[0]
    let worst = 0
    for (const c of closes) {
        if (c > peak) peak = c
        if (peak > 0) {
            const dd = (c - peak) / peak
            if (dd < worst) worst = dd
        }
    }
    return worst * 100
}

function volatilityPct(closes: number[]): number {
    if (closes.length < 2) return 0
    const returns: number[] = []
    for (let i = 1; i < closes.length; i++) {
        if (closes[i - 1] !== 0) returns.push(closes[i] / closes[i - 1] - 1)
    }
    if (returns.length === 0) return 0
    const mean = returns.reduce((a, b) => a + b, 0) / returns.length
    const variance =
        returns.reduce((a, r) => a + (r - mean) * (r - mean), 0) / returns.length
    return Math.sqrt(variance) * 100
}

export function computeCompareStats(
    series: CompareSeries[],
    visibleStartDate: string,
    currencyBySymbol: Record<string, string>
): CompareStatsRow[] {
    const rows: CompareStatsRow[] = []
    for (const s of series) {
        const { dates, opens, highs, lows, closes, volumes } = s.ohlcv
        const i = trimIndex(dates, visibleStartDate)
        const vOpens = opens.slice(i)
        const vHighs = highs.slice(i)
        const vLows = lows.slice(i)
        const vCloses = closes.slice(i)
        const vVolumes = volumes.slice(i)

        // Omit symbols with no data in the visible range.
        if (vCloses.length === 0) continue

        const open = vOpens[0]
        const close = vCloses[vCloses.length - 1]
        const change = close - open
        const changePct = open !== 0 ? (change / open) * 100 : 0

        rows.push({
            symbol: s.symbol,
            currency: currencyBySymbol[s.symbol] ?? '',
            open,
            close,
            change,
            changePct,
            high: Math.max(...vHighs),
            low: Math.min(...vLows),
            maxDrawdownPct: maxDrawdownPct(vCloses),
            volatilityPct: volatilityPct(vCloses),
            avgVolume:
                vVolumes.length > 0
                    ? vVolumes.reduce((a, b) => a + b, 0) / vVolumes.length
                    : 0
        })
    }
    return rows
}

export function useCompareStats(
    series: Ref<CompareSeries[]>,
    visibleStartDate: Ref<string>,
    currencyBySymbol: Ref<Record<string, string>>
): { rows: Ref<CompareStatsRow[]> } {
    const rows = computed(() =>
        computeCompareStats(series.value, visibleStartDate.value, currencyBySymbol.value)
    )
    return { rows }
}
