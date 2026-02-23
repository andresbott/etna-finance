import { computed, unref, type MaybeRefOrGetter } from 'vue'
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { useInstruments } from '@/composables/useInstruments'
import {
    getLatestPrice,
    getPriceHistory,
    createPrice as createPriceApi,
    createPricesBulk,
    updatePrice as updatePriceApi,
    deletePrice as deletePriceApi
} from '@/lib/api/MarketData'
import type { CreatePriceDTO, UpdatePriceDTO, PriceRecord } from '@/lib/api/MarketData'
import { lastDaysRange, rangeToStartEnd, type PriceHistoryRange } from '@/utils/dateRange'

export type { PriceHistoryRange } from '@/utils/dateRange'
export { toLocalDateString } from '@/utils/date'
export { formatPct, getChangeSeverity } from '@/utils/format'

export interface MarketInstrument {
    id: number
    symbol: string
    name: string
    currency: string
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
    prices: number[]
    volumes: number[]
}

const MARKET_INSTRUMENTS_QUERY_KEY = ['marketInstruments']

export function useMarketInstruments() {
    const queryClient = useQueryClient()
    const { instruments: instrumentsData, isLoading: instrumentsLoading } = useInstruments()
    const { start, end } = lastDaysRange(30)

    const marketInstrumentsQuery = useQuery({
        queryKey: computed(() => [
            ...MARKET_INSTRUMENTS_QUERY_KEY,
            (instrumentsData.value ?? []).map((i) => i.id).join(',')
        ]),
        queryFn: async (): Promise<MarketInstrument[]> => {
            const list = instrumentsData.value ?? []
            if (list.length === 0) return []
            const withLatest = await Promise.all(
                list.map(async (inst) => {
                    const items = await getPriceHistory(inst.symbol, start, end)
                    const n = items.length
                    const lastPrice = n > 0 ? items[n - 1].price : 0
                    const lastUpdate = n > 0 ? items[n - 1].time : ''
                    let change: number | null = null
                    let changePct: number | null = null
                    if (n >= 2) {
                        const prevPrice = items[n - 2].price
                        change = lastPrice - prevPrice
                        changePct = prevPrice !== 0 ? (change / prevPrice) * 100 : null
                    }
                    return {
                        id: inst.id,
                        symbol: inst.symbol,
                        name: inst.name,
                        currency: inst.currency,
                        lastPrice,
                        change,
                        changePct,
                        volume: 0,
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
        enabled: computed(() => (instrumentsData.value?.length ?? 0) > 0)
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
    const queryClient = useQueryClient()
    const rangeValue = computed(() => (typeof range === 'function' ? range() : unref(range)))

    const historyQuery = useQuery({
        queryKey: computed(() => ['priceHistory', typeof symbol === 'function' ? symbol() : unref(symbol), rangeValue.value]),
        queryFn: async () => {
            const sym = typeof symbol === 'function' ? symbol() : unref(symbol)
            const r = rangeValue.value
            const { start, end } = rangeToStartEnd(r)
            const items = await getPriceHistory(sym, start, end)
            const dates = items.map((r) => r.time)
            const prices = items.map((r) => r.price)
            return { dates, prices, volumes: [] as number[], records: items }
        },
        enabled: computed(() => !!(typeof symbol === 'function' ? symbol() : unref(symbol)))
    })

    return {
        data: computed(
            () =>
                historyQuery.data.value ?? {
                    dates: [],
                    prices: [],
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

    // Invalidate this instrument's price history and the market instruments list so the list view
    // and detail overview (lastPrice, lastUpdate) stay in sync after create/update/delete.
    const invalidateMarketData = () => {
        const sym = getSymbol()
        queryClient.invalidateQueries({ queryKey: ['priceHistory', sym] })
        queryClient.invalidateQueries({ queryKey: MARKET_INSTRUMENTS_QUERY_KEY })
    }

    const createPriceMutation = useMutation({
        mutationFn: (payload: CreatePriceDTO) => createPriceApi(getSymbol(), payload),
        onSuccess: invalidateMarketData
    })

    const updatePriceMutation = useMutation({
        mutationFn: ({ id, payload }: { id: number; payload: UpdatePriceDTO }) =>
            updatePriceApi(id, payload),
        onSuccess: invalidateMarketData
    })

    const deletePriceMutation = useMutation({
        mutationFn: (id: number) => deletePriceApi(id),
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
