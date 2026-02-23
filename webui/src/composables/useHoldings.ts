import { computed, watch } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { useAccounts } from '@/composables/useAccounts'
import { useInstruments } from '@/composables/useInstruments'
import { getEntries } from '@/lib/api/Entry'
import { getLatestPrice } from '@/lib/api/MarketData'
import { ACCOUNT_TYPES } from '@/types/account'

export interface Holding {
    instrumentId: number
    symbol: string
    currency: string
    quantity: number
    lastPrice: number | null
    value: number // quantity * lastPrice (or 0 if no price)
    costBasis: number // invested amount (sum of purchase costs, average-cost method)
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

/**
 * Aggregates quantity deltas from stock transactions to compute positions per account.
 * Returns a map: accountId -> instrumentId -> quantity
 */
function aggregatePositionsFromEntries(
    entries: Array<{
        type: string
        investmentAccountId?: number
        cashAccountId?: number
        accountId?: number
        originAccountId?: number
        targetAccountId?: number
        instrumentId?: number
        quantity?: number
    }>
): Map<number, Map<number, number>> {
    const positions = new Map<number, Map<number, number>>()

    function addQty(accountId: number, instrumentId: number, delta: number) {
        if (!accountId || !instrumentId) return
        let accMap = positions.get(accountId)
        if (!accMap) {
            accMap = new Map()
            positions.set(accountId, accMap)
        }
        const prev = accMap.get(instrumentId) ?? 0
        accMap.set(instrumentId, prev + delta)
    }

    for (const e of entries) {
        const instId = e.instrumentId
        const qty = e.quantity ?? 0
        if (!instId || qty <= 0) continue

        switch (e.type) {
            case 'stockbuy':
                if (e.investmentAccountId) {
                    addQty(e.investmentAccountId, instId, qty)
                }
                break
            case 'stocksell':
                if (e.investmentAccountId) {
                    addQty(e.investmentAccountId, instId, -qty)
                }
                break
            case 'stockgrant':
                if (e.accountId) {
                    addQty(e.accountId, instId, qty)
                }
                break
            case 'stocktransfer':
                if (e.originAccountId) addQty(e.originAccountId, instId, -qty)
                if (e.targetAccountId) addQty(e.targetAccountId, instId, qty)
                break
        }
    }

    return positions
}

type EntryLike = {
    date: string
    type: string
    instrumentId?: number
    quantity?: number
    StockAmount?: number
    fairMarketValue?: number
    investmentAccountId?: number
    accountId?: number
    originAccountId?: number
    targetAccountId?: number
}

/**
 * Computes cost basis per (accountId, instrumentId) using average-cost method.
 * Processes entries in date order. Returns Map<accountId, Map<instrumentId, costBasis>>.
 */
function aggregateCostBasisFromEntries(entries: EntryLike[]): Map<number, Map<number, number>> {
    const costMap = new Map<number, Map<number, number>>()
    // running state: accountId -> instrumentId -> { quantity, costBasis }
    const state = new Map<number, Map<number, { quantity: number; costBasis: number }>>()

    function get(accountId: number, instrumentId: number) {
        let acc = state.get(accountId)
        if (!acc) {
            acc = new Map()
            state.set(accountId, acc)
        }
        let row = acc.get(instrumentId)
        if (!row) {
            row = { quantity: 0, costBasis: 0 }
            acc.set(instrumentId, row)
        }
        return row
    }

    const sorted = [...entries].sort(
        (a, b) => (a.date || '').localeCompare(b.date || '', undefined, { numeric: true })
    )

    for (const e of sorted) {
        const instId = e.instrumentId
        const qty = e.quantity ?? 0
        if (!instId || qty <= 0) continue

        switch (e.type) {
            case 'stockbuy': {
                const accId = e.investmentAccountId
                if (!accId) break
                const stockAmt = e.StockAmount ?? 0
                const row = get(accId, instId)
                row.quantity += qty
                row.costBasis += stockAmt
                break
            }
            case 'stocksell': {
                const accId = e.investmentAccountId
                if (!accId) break
                const row = get(accId, instId)
                if (row.quantity > 0) {
                    const ratio = qty / row.quantity
                    row.costBasis *= 1 - ratio
                    row.quantity -= qty
                }
                break
            }
            case 'stockgrant': {
                const accId = e.accountId
                if (!accId) break
                const fmv = e.fairMarketValue ?? 0
                const row = get(accId, instId)
                row.quantity += qty
                row.costBasis += fmv * qty
                break
            }
            case 'stocktransfer': {
                const origId = e.originAccountId
                const tgtId = e.targetAccountId
                if (!origId || !tgtId) break
                const orig = get(origId, instId)
                if (orig.quantity > 0) {
                    const ratio = qty / orig.quantity
                    const movedCost = orig.costBasis * ratio
                    orig.costBasis -= movedCost
                    orig.quantity -= qty
                    const tgt = get(tgtId, instId)
                    tgt.quantity += qty
                    tgt.costBasis += movedCost
                }
                break
            }
        }
    }

    for (const [accId, accMap] of state) {
        const out = new Map<number, number>()
        for (const [instId, row] of accMap) {
            if (row.quantity > 0 && row.costBasis !== 0) {
                out.set(instId, row.costBasis)
            }
        }
        if (out.size > 0) costMap.set(accId, out)
    }
    return costMap
}

export function useHoldings() {
    const { accounts: accountProviders } = useAccounts()
    const { instruments } = useInstruments()

    const investmentAccountIds = computed(() => {
        const ids: string[] = []
        for (const provider of accountProviders.value ?? []) {
            for (const acc of provider.accounts ?? []) {
                if (
                    acc.type === ACCOUNT_TYPES.INVESTMENT ||
                    acc.type === ACCOUNT_TYPES.UNVESTED
                ) {
                    ids.push(String(acc.id))
                }
            }
        }
        return ids
    })

    const instrumentsMap = computed(() => {
        const list = instruments.value ?? []
        return Object.fromEntries(list.map((i) => [i.id, i]))
    })

    const entriesQuery = useQuery({
        queryKey: computed(() => ['holdings-entries', investmentAccountIds.value]),
        queryFn: async () => {
            const ids = investmentAccountIds.value
            if (ids.length === 0) return []

            const startDate = new Date()
            startDate.setFullYear(startDate.getFullYear() - 20)
            const endDate = new Date()

            const all: unknown[] = []
            let page = 1
            const limit = 500

            while (true) {
                const res = await getEntries({
                    startDate,
                    endDate,
                    accountIds: ids,
                    page,
                    limit
                })
                all.push(...(res.items ?? []))
                if (res.items.length < limit || all.length >= res.total) break
                page++
            }
            return all
        },
        enabled: computed(() => investmentAccountIds.value.length > 0)
    })

    const positionsMap = computed(() => {
        const items = entriesQuery.data.value ?? []
        return aggregatePositionsFromEntries(items as Parameters<typeof aggregatePositionsFromEntries>[0])
    })

    const costBasisMap = computed(() => {
        const items = entriesQuery.data.value ?? []
        return aggregateCostBasisFromEntries(items as EntryLike[])
    })

    const symbolSet = computed(() => {
        const symbols = new Set<string>()
        const instMap = instrumentsMap.value
        for (const accMap of positionsMap.value.values()) {
            for (const [instId, qty] of accMap) {
                if (qty > 0 && instMap[instId]?.symbol) {
                    symbols.add(instMap[instId].symbol)
                }
            }
        }
        return Array.from(symbols)
    })

    const pricesQuery = useQuery({
        queryKey: computed(() => ['holdings-prices', symbolSet.value.join(',')]),
        queryFn: async () => {
            const map: Record<string, number> = {}
            for (const sym of symbolSet.value) {
                try {
                    const p = await getLatestPrice(sym)
                    if (p?.price != null) map[sym] = p.price
                } catch {
                    // ignore missing prices
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
        const costBasis = costBasisMap.value
        const prices = pricesQuery.data.value ?? {}

        return provs
            .map((provider) => {
                const accounts: AccountWithHoldings[] = []

                for (const account of provider.accounts ?? []) {
                    if (
                        account.type !== ACCOUNT_TYPES.INVESTMENT &&
                        account.type !== ACCOUNT_TYPES.UNVESTED
                    ) continue

                    const accPos = positions.get(account.id)
                    const holdings: Holding[] = []
                    let totalValue = 0

                    if (accPos) {
                        const accCost = costBasis.get(account.id)
                        for (const [instrumentId, quantity] of accPos) {
                            if (quantity <= 0) continue
                            const inst = instMap[instrumentId]
                            const symbol = inst?.symbol ?? ''
                            const currency = inst?.currency ?? 'CHF'
                            const lastPrice = symbol ? (prices[symbol] ?? null) : null
                            const value = lastPrice != null ? quantity * lastPrice : 0
                            const costBasisVal = accCost?.get(instrumentId) ?? 0
                            totalValue += value
                            holdings.push({
                                instrumentId,
                                symbol,
                                currency,
                                quantity,
                                lastPrice,
                                value,
                                costBasis: costBasisVal
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
                return {
                    id: provider.id,
                    name: provider.name,
                    icon: provider.icon,
                    accounts
                }
            })
            .filter((p): p is ProviderWithHoldings => p != null)
    })

    return {
        providersWithHoldings,
        isLoading: computed(
            () => entriesQuery.isLoading.value || pricesQuery.isLoading.value
        ),
        isError: computed(
            () => entriesQuery.isError.value || pricesQuery.isError.value
        ),
        refetch: () => {
            entriesQuery.refetch()
            pricesQuery.refetch()
        }
    }
}
