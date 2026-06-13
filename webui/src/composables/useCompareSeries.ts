// webui/src/composables/useCompareSeries.ts
import { computed, unref, type MaybeRefOrGetter } from 'vue'
import { useQuery, keepPreviousData } from '@tanstack/vue-query'
import { getPriceHistory } from '@/lib/api/MarketData'
import { rangeToStartEnd, toLocalDateString, type PriceHistoryRange } from '@/utils/dateRange'
import { parseLocalDate } from '@/utils/date'
import type { OhlcvData } from '@/composables/useTsdbPrices'

export interface CompareSeries {
    symbol: string
    ohlcv: OhlcvData
}

interface CompareResult {
    series: CompareSeries[]
    visibleStartDate: string
}

export function useCompareSeries(
    symbols: MaybeRefOrGetter<string[]>,
    range: MaybeRefOrGetter<PriceHistoryRange>,
    warmupDays?: MaybeRefOrGetter<number>
) {
    const getSymbols = () => (typeof symbols === 'function' ? symbols() : unref(symbols)) ?? []
    const getRange = () => (typeof range === 'function' ? range() : unref(range))
    const getWarmup = () =>
        warmupDays ? (typeof warmupDays === 'function' ? warmupDays() : unref(warmupDays)) : 0

    const query = useQuery({
        queryKey: computed(() => [
            'compareSeries',
            [...getSymbols()].sort().join(','),
            getRange(),
            getWarmup()
        ]),
        queryFn: async (): Promise<CompareResult> => {
            const syms = getSymbols()
            const { start, end } = rangeToStartEnd(getRange())
            const warmup = getWarmup()

            let fetchStart = start
            if (warmup > 0) {
                const d = parseLocalDate(start)
                d.setDate(d.getDate() - warmup)
                fetchStart = toLocalDateString(d)
            }

            const series = await Promise.all(
                syms.map(async (symbol): Promise<CompareSeries> => {
                    const items = await getPriceHistory(symbol, fetchStart, end)
                    return {
                        symbol,
                        ohlcv: {
                            dates: items.map((p) => p.time),
                            opens: items.map((p) => p.open),
                            highs: items.map((p) => p.high),
                            lows: items.map((p) => p.low),
                            closes: items.map((p) => p.close),
                            volumes: items.map((p) => p.volume),
                            points: items
                        }
                    }
                })
            )
            return { series, visibleStartDate: start }
        },
        enabled: computed(() => getSymbols().length > 0),
        // Keep the previous chart visible while a range/period/selection change refetches,
        // so switching the toolbar does not flash the chart back to a loading spinner.
        placeholderData: keepPreviousData
    })

    return {
        data: computed(() => query.data.value?.series ?? []),
        visibleStartDate: computed(() => query.data.value?.visibleStartDate ?? ''),
        isLoading: query.isLoading,
        isError: query.isError
    }
}
