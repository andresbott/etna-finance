import { computed } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { useAccounts } from '@/composables/useAccounts'
import { useInstruments } from '@/composables/useInstruments'
import { getPositions } from '@/lib/api/Portfolio'
import { getLatestPrice } from '@/lib/api/MarketData'
import { ACCOUNT_TYPES } from '@/types/account'

export interface Holding {
    instrumentId: number
    symbol: string
    currency: string
    quantity: number
    lastPrice: number | null
    value: number // quantity * lastPrice (or 0 if no price)
    costBasis: number
}

export interface AccountWithHoldings {
    id: number
    name: string
    type: string
    currency: string
    icon?: string
    holdings: Holding[]
    totalValue: number
}

export interface ProviderWithHoldings {
    id: number
    name: string
    icon?: string
    accounts: AccountWithHoldings[]
}

export function useHoldings() {
    const { accounts: accountProviders } = useAccounts()
    const { instruments } = useInstruments()

    const instrumentsMap = computed(() => {
        const list = instruments.value ?? []
        return Object.fromEntries(list.map((i) => [i.id, i]))
    })

    // Fetch positions from the backend API (replaces entry replay logic)
    const positionsQuery = useQuery({
        queryKey: ['portfolio-positions'],
        queryFn: () => getPositions()
    })

    // Build a map: accountId -> instrumentId -> { quantity, costBasis }
    const positionsMap = computed(() => {
        const map = new Map<number, Map<number, { quantity: number; costBasis: number }>>()
        for (const pos of positionsQuery.data.value ?? []) {
            let accMap = map.get(pos.accountId)
            if (!accMap) {
                accMap = new Map()
                map.set(pos.accountId, accMap)
            }
            accMap.set(pos.instrumentId, {
                quantity: pos.quantity,
                costBasis: pos.costBasis
            })
        }
        return map
    })

    const symbolSet = computed(() => {
        const symbols = new Set<string>()
        const instMap = instrumentsMap.value
        for (const accMap of positionsMap.value.values()) {
            for (const [instId, data] of accMap) {
                if (data.quantity > 0 && instMap[instId]?.symbol) {
                    symbols.add(instMap[instId].symbol)
                }
            }
        }
        return Array.from(symbols)
    })

    const pricesQuery = useQuery({
        queryKey: computed(() => ['holdings-prices', symbolSet.value.join(',')]),
        queryFn: async () => {
            const results = await Promise.allSettled(
                symbolSet.value.map(async (sym) => {
                    const p = await getLatestPrice(sym)
                    return { sym, price: p?.price ?? null }
                })
            )
            const map: Record<string, number> = {}
            for (const result of results) {
                if (result.status === 'fulfilled' && result.value.price != null) {
                    map[result.value.sym] = result.value.price
                }
            }
            return map
        },
        enabled: computed(() => symbolSet.value.length > 0)
    })

    const providersWithHoldings = computed<ProviderWithHoldings[]>(() => {
        const provs = accountProviders.value ?? []
        const instMap = instrumentsMap.value
        const positions = positionsMap.value
        const prices = pricesQuery.data.value ?? {}

        return provs
            .map((provider) => {
                const accounts: AccountWithHoldings[] = []

                for (const account of provider.accounts ?? []) {
                    if (
                        account.type !== ACCOUNT_TYPES.INVESTMENT &&
                        account.type !== ACCOUNT_TYPES.RESTRICTED_STOCK
                    ) continue

                    const accPos = positions.get(account.id)
                    const holdings: Holding[] = []
                    let totalValue = 0

                    if (accPos) {
                        for (const [instrumentId, data] of accPos) {
                            if (data.quantity <= 0) continue
                            const inst = instMap[instrumentId]
                            const symbol = inst?.symbol ?? ''
                            const currency = inst?.currency ?? 'CHF'
                            const lastPrice = symbol ? (prices[symbol] ?? null) : null
                            const value = lastPrice != null ? data.quantity * lastPrice : 0
                            totalValue += value
                            holdings.push({
                                instrumentId,
                                symbol,
                                currency,
                                quantity: data.quantity,
                                lastPrice,
                                value,
                                costBasis: data.costBasis
                            })
                        }
                    }

                    accounts.push({
                        id: account.id,
                        name: account.name,
                        type: account.type,
                        currency: account.currency ?? 'CHF',
                        icon: account.icon,
                        holdings,
                        totalValue
                    })
                }

                if (accounts.length === 0) return null
                const result: ProviderWithHoldings = {
                    id: provider.id,
                    name: provider.name,
                    accounts
                }
                if (provider.icon) result.icon = provider.icon
                return result
            })
            .filter((p): p is ProviderWithHoldings => p != null)
    })

    return {
        providersWithHoldings,
        isLoading: computed(
            () => positionsQuery.isLoading.value || pricesQuery.isLoading.value
        ),
        isError: computed(
            () => positionsQuery.isError.value || pricesQuery.isError.value
        ),
        refetch: () => {
            positionsQuery.refetch()
            pricesQuery.refetch()
        }
    }
}
