import { computed, unref, type MaybeRefOrGetter } from 'vue'
import { useQuery, useMutation, useQueryClient, keepPreviousData } from '@tanstack/vue-query'
import { useInstruments } from '@/composables/useInstruments'
import {
    getPriceHistory,
    createPrice as createPriceApi,
    updatePrice as updatePriceApi,
    deletePrice as deletePriceApi
} from '@/lib/api/MarketData'
import type { CreatePriceDTO, PriceRecord } from '@/lib/api/MarketData'
import { lastDaysRange, rangeToStartEnd, type PriceHistoryRange } from '@/utils/dateRange'

export type { PriceHistoryRange } from '@/utils/dateRange'
export { toLocalDateString } from '@/utils/date'
export { formatPct, getChangeSeverity } from '@/utils/format'

export interface MarketInstrument {
    id: number
    symbol: string
    name: string
    notes: string
    currency: string
    type: string
    exchange: string
    lastPrice: number
    change: number | null
    changePct: number | null
    volume: number
    peRatio: number | null
    dividendYield: number
    week52High: number
    week52Low: number
    lastUpdate: string
}

export interface PriceHistory {
    dates: string[]
    opens: number[]
    highs: number[]
    lows: number[]
    closes: number[]
    volumes: number[]
    records: PriceRecord[]
}

const MARKET_INSTRUMENTS_QUERY_KEY = ['marketInstruments']

export function useMarketInstruments() {
    const { instruments: instrumentsData, isLoading: instrumentsLoading } = useInstruments()
    const { start, end } = lastDaysRange(30)

    const marketInstrumentsQuery = useQuery({
        // Key on instrument *content* (not just ids) so that editing a name/type/exchange/
        // currency/symbol reactively busts this cache once the instruments query refetches.
        // Keying on ids alone would only refetch on add/remove, leaving edits stale.
        queryKey: computed(() => [
            ...MARKET_INSTRUMENTS_QUERY_KEY,
            (instrumentsData.value ?? [])
                .map((i) => `${i.id}:${i.symbol}:${i.name}:${i.type}:${i.exchange}:${i.currency}`)
                .join('|')
        ]),
        queryFn: async (): Promise<MarketInstrument[]> => {
            const list = instrumentsData.value ?? []
            if (list.length === 0) return []
            const withLatest = await Promise.all(
                list.map(async (inst) => {
                    const items = await getPriceHistory(inst.symbol, start, end)
                    const n = items.length
                    const lastPrice = n > 0 ? items[n - 1].close : 0
                    const lastUpdate = n > 0 ? items[n - 1].time : ''
                    let change: number | null = null
                    let changePct: number | null = null
                    if (n >= 2) {
                        const prevPrice = items[n - 2].close
                        change = lastPrice - prevPrice
                        changePct = prevPrice !== 0 ? (change / prevPrice) * 100 : null
                    }
                    return {
                        id: inst.id,
                        symbol: inst.symbol,
                        name: inst.name,
                        notes: inst.notes ?? '',
                        currency: inst.currency,
                        type: inst.type,
                        exchange: inst.exchange,
                        lastPrice,
                        change,
                        changePct,
                        volume: n > 0 ? items[n - 1].volume : 0,
                        peRatio: null as number | null,
                        dividendYield: 0,
                        week52High: 0,
                        week52Low: 0,
                        lastUpdate
                    }
                })
            )
            return withLatest
        },
        enabled: computed(() => (instrumentsData.value?.length ?? 0) > 0),
        // Keep showing the previous rows while a content-key change (edit/add/delete) or a
        // price update triggers a refetch, so the table does not flash empty mid-refetch.
        placeholderData: keepPreviousData
    })

    const instruments = computed<MarketInstrument[]>(() => {
        return marketInstrumentsQuery.data.value ?? []
    })

    const isLoading = computed(
        () => instrumentsLoading.value || marketInstrumentsQuery.isLoading.value
    )

    return {
        instruments,
        isLoading,
        isError: marketInstrumentsQuery.isError,
        error: marketInstrumentsQuery.error,
        refetch: marketInstrumentsQuery.refetch
    }
}

export function usePriceHistory(
    symbol: MaybeRefOrGetter<string>,
    range: MaybeRefOrGetter<PriceHistoryRange>
) {
    const rangeValue = computed(() => (typeof range === 'function' ? range() : unref(range)))

    const historyQuery = useQuery({
        queryKey: computed(() => ['priceHistory', typeof symbol === 'function' ? symbol() : unref(symbol), rangeValue.value]),
        queryFn: async () => {
            const sym = typeof symbol === 'function' ? symbol() : unref(symbol)
            const r = rangeValue.value
            const { start, end } = rangeToStartEnd(r)
            const items = await getPriceHistory(sym, start, end)
            const dates = items.map((r) => r.time)
            return {
                dates,
                opens: items.map(r => r.open),
                highs: items.map(r => r.high),
                lows: items.map(r => r.low),
                closes: items.map(r => r.close),
                volumes: items.map(r => r.volume),
                records: items
            }
        },
        enabled: computed(() => !!(typeof symbol === 'function' ? symbol() : unref(symbol)))
    })

    return {
        data: computed(
            () =>
                historyQuery.data.value ?? {
                    dates: [],
                    opens: [],
                    highs: [],
                    lows: [],
                    closes: [],
                    volumes: [],
                    records: [] as PriceRecord[]
                }
        ),
        isLoading: historyQuery.isLoading,
        refetch: historyQuery.refetch
    }
}

export function useMarketDataMutations(symbol: MaybeRefOrGetter<string>) {
    const queryClient = useQueryClient()

    const getSymbol = () => (typeof symbol === 'function' ? symbol() : unref(symbol))

    // Invalidate this instrument's price history, the candlestick chart series, and the market
    // instruments list so the list view, detail overview (lastPrice, lastUpdate), and Chart tab
    // all stay in sync after create/update/delete.
    const invalidateMarketData = () => {
        const sym = getSymbol()
        queryClient.invalidateQueries({ queryKey: ['priceHistory', sym] })
        queryClient.invalidateQueries({ queryKey: ['tsdbPrices', sym] })
        queryClient.invalidateQueries({ queryKey: MARKET_INSTRUMENTS_QUERY_KEY })
    }

    const createPriceMutation = useMutation({
        mutationFn: (payload: CreatePriceDTO) => createPriceApi(getSymbol(), payload),
        onSuccess: invalidateMarketData
    })

    const updatePriceMutation = useMutation({
        mutationFn: ({ origDate, payload }: { origDate: string; payload: CreatePriceDTO }) =>
            updatePriceApi(getSymbol(), origDate, payload),
        onSuccess: invalidateMarketData
    })

    const deletePriceMutation = useMutation({
        mutationFn: (date: string) => deletePriceApi(getSymbol(), date),
        onSuccess: invalidateMarketData
    })

    return {
        createPrice: createPriceMutation.mutateAsync,
        updatePrice: updatePriceMutation.mutateAsync,
        deletePrice: deletePriceMutation.mutateAsync,
        isCreating: createPriceMutation.isPending,
        isUpdating: updatePriceMutation.isPending,
        isDeleting: deletePriceMutation.isPending
    }
}

export function formatPrice(value: number | null | undefined): string {
    if (value == null) return '-'
    return value.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

export function formatVolume(value: number | null | undefined): string {
    if (value == null) return '-'
    if (value >= 1_000_000) return (value / 1_000_000).toFixed(1) + 'M'
    if (value >= 1_000) return (value / 1_000).toFixed(0) + 'K'
    return value.toString()
}

export function formatChange(value: number | null | undefined): string {
    if (value == null) return '-'
    const sign = value >= 0 ? '+' : ''
    return sign + value.toFixed(2)
}

export interface MarketInstrumentFilters {
    search: string
    types: string[]
    exchanges: string[]
}

export function filterMarketInstruments(
    instruments: MarketInstrument[],
    filters: MarketInstrumentFilters
): MarketInstrument[] {
    const search = (filters.search ?? '').trim().toLowerCase()
    return instruments.filter((inst) => {
        const matchesSearch =
            search === '' ||
            inst.symbol.toLowerCase().includes(search) ||
            inst.name.toLowerCase().includes(search)
        const matchesType = filters.types.length === 0 || filters.types.includes(inst.type)
        const matchesExchange =
            filters.exchanges.length === 0 || filters.exchanges.includes(inst.exchange)
        return matchesSearch && matchesType && matchesExchange
    })
}
