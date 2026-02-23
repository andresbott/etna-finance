import { computed, unref, type MaybeRefOrGetter } from 'vue'
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { useSettingsStore } from '@/store/settingsStore'
import {
    getFXPairs,
    getRateHistory,
    getLatestRate,
    createRate as createRateApi,
    createRatesBulk,
    updateRate as updateRateApi,
    deleteRate as deleteRateApi,
    parsePair
} from '@/lib/api/CurrencyRates'
import type { CreateRateDTO, UpdateRateDTO, RateRecord } from '@/lib/api/CurrencyRates'
import { toLocalDateString } from '@/composables/useMarketData'

export interface FXOverviewRow {
    currency: string
    pair: string
    rate: number
    change: number
    lastUpdate: string | null
}

const FX_OVERVIEW_QUERY_KEY = ['fxOverview']
const FX_RATE_HISTORY_QUERY_KEY = 'fxRateHistory'

/** Date range for overview (last 30 days to compute change). Use local date so "today" is correct in all timezones. */
function lastDaysRange(days: number): { start: string; end: string } {
    const end = new Date()
    end.setHours(0, 0, 0, 0)
    const start = new Date(end)
    start.setDate(start.getDate() - days)
    return { start: toLocalDateString(start), end: toLocalDateString(end) }
}

function rangeToStartEnd(range: '6m' | 'max'): { start: string; end: string } {
    const end = new Date()
    end.setHours(0, 0, 0, 0)
    const start = new Date(end)
    if (range === '6m') start.setMonth(start.getMonth() - 6)
    else start.setFullYear(start.getFullYear() - 10)
    return { start: toLocalDateString(start), end: toLocalDateString(end) }
}

export function useFXOverview() {
    const settingsStore = useSettingsStore()
    const mainCurrency = computed(() => settingsStore.mainCurrency || 'CHF')
    const { start, end } = lastDaysRange(30)

    const pairsQuery = useQuery({
        queryKey: [FX_OVERVIEW_QUERY_KEY],
        queryFn: getFXPairs,
        enabled: true
    })

    const overviewQuery = useQuery({
        queryKey: computed(() => [...FX_OVERVIEW_QUERY_KEY, 'rows', pairsQuery.data.value?.join(',')]),
        queryFn: async (): Promise<FXOverviewRow[]> => {
            const pairs = pairsQuery.data.value ?? []
            if (pairs.length === 0) return []
            const rows = await Promise.all(
                pairs.map(async (pairStr) => {
                    const [main, secondary] = parsePair(pairStr)
                    const items = await getRateHistory(main, secondary, start, end)
                    const sorted = [...items].sort((a, b) => a.time.localeCompare(b.time))
                    const n = sorted.length
                    const latest = n > 0 ? sorted[n - 1] : null
                    const prev = n >= 2 ? sorted[n - 2] : null
                    let change = 0
                    if (latest && prev && prev.rate !== 0) {
                        change = ((latest.rate - prev.rate) / prev.rate) * 100
                    }
                    return {
                        currency: secondary,
                        pair: pairStr,
                        rate: latest?.rate ?? 0,
                        change,
                        lastUpdate: latest?.time ?? null
                    }
                })
            )
            return rows
        },
        enabled: computed(() => (pairsQuery.data.value?.length ?? 0) > 0)
    })

    const currencyRows = computed<FXOverviewRow[]>(() => overviewQuery.data.value ?? [])
    const isLoading = computed(() => pairsQuery.isLoading.value || overviewQuery.isLoading.value)

    return {
        mainCurrency,
        currencyRows,
        isLoading,
        isError: pairsQuery.isError,
        error: pairsQuery.error,
        refetch: () => {
            pairsQuery.refetch()
            overviewQuery.refetch()
        }
    }
}

export type RateHistoryRange = '6m' | 'max'

export function useRateHistory(
    main: MaybeRefOrGetter<string>,
    secondary: MaybeRefOrGetter<string>,
    range: MaybeRefOrGetter<RateHistoryRange>
) {
    const mainVal = computed(() => (typeof main === 'function' ? main() : unref(main)))
    const secondaryVal = computed(() => (typeof secondary === 'function' ? secondary() : unref(secondary)))
    const rangeVal = computed(() => (typeof range === 'function' ? range() : unref(range)))

    const historyQuery = useQuery({
        queryKey: computed(() => [FX_RATE_HISTORY_QUERY_KEY, mainVal.value, secondaryVal.value, rangeVal.value]),
        queryFn: async () => {
            const m = mainVal.value
            const s = secondaryVal.value
            if (!m || !s) return { dates: [] as string[], prices: [] as number[], records: [] as RateRecord[] }
            const { start, end } = rangeToStartEnd(rangeVal.value)
            const items = await getRateHistory(m, s, start, end)
            const dates = items.map((r) => r.time)
            const prices = items.map((r) => r.rate)
            return { dates, prices, records: items }
        },
        enabled: computed(() => !!mainVal.value && !!secondaryVal.value)
    })

    return {
        data: computed(
            () =>
                historyQuery.data.value ?? {
                    dates: [] as string[],
                    prices: [] as number[],
                    records: [] as RateRecord[]
                }
        ),
        isLoading: historyQuery.isLoading,
        refetch: historyQuery.refetch
    }
}

export function useFXMutations(main: MaybeRefOrGetter<string>, secondary: MaybeRefOrGetter<string>) {
    const queryClient = useQueryClient()
    const getMain = () => (typeof main === 'function' ? main() : unref(main))
    const getSecondary = () => (typeof secondary === 'function' ? secondary() : unref(secondary))

    const invalidate = () => {
        queryClient.invalidateQueries({ queryKey: [FX_OVERVIEW_QUERY_KEY] })
        queryClient.invalidateQueries({
            queryKey: [FX_RATE_HISTORY_QUERY_KEY, getMain(), getSecondary()]
        })
    }

    const createMutation = useMutation({
        mutationFn: (payload: CreateRateDTO) => createRateApi(getMain(), getSecondary(), payload),
        onSuccess: invalidate
    })
    const updateMutation = useMutation({
        mutationFn: ({ id, payload }: { id: number; payload: UpdateRateDTO }) =>
            updateRateApi(id, payload),
        onSuccess: invalidate
    })
    const deleteMutation = useMutation({
        mutationFn: (id: number) => deleteRateApi(id),
        onSuccess: invalidate
    })

    return {
        createRate: createMutation.mutateAsync,
        updateRate: updateMutation.mutateAsync,
        deleteRate: deleteMutation.mutateAsync,
        isCreating: createMutation.isPending,
        isUpdating: updateMutation.isPending,
        isDeleting: deleteMutation.isPending
    }
}

export function formatRate(value: number | null | undefined): string {
    if (value == null) return '-'
    return value.toFixed(4)
}

export function formatPct(value: number | null | undefined): string {
    if (value == null) return '-'
    const sign = value >= 0 ? '+' : ''
    return sign + value.toFixed(2) + '%'
}

export function getChangeSeverity(value: number | null | undefined): string {
    if (value == null) return 'secondary'
    if (value > 0) return 'success'
    if (value < 0) return 'danger'
    return 'secondary'
}
