// webui/src/composables/useTsdbPrices.ts
import { computed, unref, type MaybeRefOrGetter } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { getPriceHistory, type PriceRecord } from '@/lib/api/MarketData'
import { rangeToStartEnd, toLocalDateString, type PriceHistoryRange } from '@/utils/dateRange'

export interface OhlcvData {
    dates: string[]
    opens: number[]
    highs: number[]
    lows: number[]
    closes: number[]
    volumes: number[]
    points: PriceRecord[]
}

const EMPTY: OhlcvData = { dates: [], opens: [], highs: [], lows: [], closes: [], volumes: [], points: [] }

interface TsdbResult {
    ohlcv: OhlcvData
    visibleStartDate: string
}

export function useTsdbPrices(
    symbol: MaybeRefOrGetter<string>,
    range: MaybeRefOrGetter<PriceHistoryRange>,
    warmupDays?: MaybeRefOrGetter<number>
) {
    const getSymbol = () => (typeof symbol === 'function' ? symbol() : unref(symbol))
    const getRange = () => (typeof range === 'function' ? range() : unref(range))
    const getWarmup = () => (warmupDays ? (typeof warmupDays === 'function' ? warmupDays() : unref(warmupDays)) : 0)

    const query = useQuery({
        queryKey: computed(() => ['tsdbPrices', getSymbol(), getRange(), getWarmup()]),
        queryFn: async (): Promise<TsdbResult> => {
            const sym = getSymbol()
            const { start, end } = rangeToStartEnd(getRange())
            const warmup = getWarmup()

            let fetchStart = start
            if (warmup > 0) {
                const d = new Date(start)
                d.setDate(d.getDate() - warmup)
                fetchStart = toLocalDateString(d)
            }

            const items = await getPriceHistory(sym, fetchStart, end)
            return {
                ohlcv: {
                    dates: items.map(p => p.time),
                    opens: items.map(p => p.open),
                    highs: items.map(p => p.high),
                    lows: items.map(p => p.low),
                    closes: items.map(p => p.close),
                    volumes: items.map(p => p.volume),
                    points: items
                },
                visibleStartDate: start
            }
        },
        enabled: computed(() => !!getSymbol())
    })

    return {
        data: computed(() => query.data.value?.ohlcv ?? EMPTY),
        visibleStartDate: computed(() => query.data.value?.visibleStartDate ?? ''),
        isLoading: query.isLoading,
        isError: query.isError,
        error: query.error
    }
}
