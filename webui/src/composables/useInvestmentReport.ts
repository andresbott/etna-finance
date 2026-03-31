import { computed } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { useHoldings } from '@/composables/useHoldings'
import { useInstruments } from '@/composables/useInstruments'
import { useAccounts } from '@/composables/useAccounts'
import { useSettingsStore } from '@/store/settingsStore'
import { getLatestRate } from '@/lib/api/CurrencyRates'
import { getLots } from '@/lib/api/Portfolio'
import type { TreeNode } from 'primevue/treenode'

export interface ProductPosition {
    instrumentId: number
    symbol: string
    name: string
    currency: string
    totalQuantity: number
    avgCostPerShare: number | null // investedAmount / totalQuantity, null when totalQuantity is 0
    lastPrice: number | null
    totalValue: number
    investedAmount: number
    winLoss: number // totalValue - investedAmount (unrealized gain when positive, loss when negative)
    winLossPercent: number | null // winLoss / investedAmount * 100, null when investedAmount is 0
}

/**
 * Aggregates holdings by investment product for the investment report.
 */
export function useInvestmentReport() {
    const settingsStore = useSettingsStore()
    const mainCurrency = computed(() => settingsStore.mainCurrency || 'CHF')
    const { providersWithHoldings, isLoading: holdingsLoading } = useHoldings()
    const { instruments } = useInstruments()
    const { accounts } = useAccounts()

    const { data: lots, isLoading: lotsLoading } = useQuery({
        queryKey: ['portfolio-lots'],
        queryFn: () => getLots()
    })

    const isLoading = computed(() => holdingsLoading.value || lotsLoading.value)

    const instrumentsMap = computed(() => {
        const list = instruments.value ?? []
        return Object.fromEntries(list.map((i) => [i.id, i]))
    })

    const accountMap = computed(() => {
        const map: Record<number, { name: string }> = {}
        for (const provider of accounts.value ?? []) {
            for (const account of provider.accounts) {
                map[account.id] = { name: account.name }
            }
        }
        return map
    })

    const productPositions = computed<ProductPosition[]>(() => {
        const instMap = instrumentsMap.value
        const provs = providersWithHoldings.value ?? []
        const byInstrument = new Map<
            number,
            {
                instrumentId: number
                symbol: string
                name: string
                currency: string
                totalQuantity: number
                totalValue: number
                investedAmount: number
                lastPrice: number | null
            }
        >()

        for (const provider of provs) {
            for (const account of provider.accounts) {
                for (const h of account.holdings) {
                    if (h.quantity <= 0) continue
                    const inst = instMap[h.instrumentId]
                    let row = byInstrument.get(h.instrumentId)
                    if (!row) {
                        row = {
                            instrumentId: h.instrumentId,
                            symbol: h.symbol || inst?.symbol || `#${h.instrumentId}`,
                            name: inst?.name || h.symbol || `Product #${h.instrumentId}`,
                            currency: h.currency || inst?.currency || 'CHF',
                            totalQuantity: 0,
                            totalValue: 0,
                            investedAmount: 0,
                            lastPrice: h.lastPrice
                        }
                        byInstrument.set(h.instrumentId, row)
                    }
                    row.totalQuantity += h.quantity
                    row.totalValue += h.value
                    row.investedAmount += h.costBasis ?? 0
                    if (row.lastPrice == null && h.lastPrice != null) row.lastPrice = h.lastPrice
                }
            }
        }

        return Array.from(byInstrument.values())
            .map((row) => {
                const winLoss = row.totalValue - row.investedAmount
                return {
                    ...row,
                    avgCostPerShare: row.totalQuantity > 0 ? row.investedAmount / row.totalQuantity : null,
                    winLoss,
                    winLossPercent: row.investedAmount !== 0 ? (winLoss / row.investedAmount) * 100 : null
                }
            })
            .sort((a, b) =>
                a.symbol.localeCompare(b.symbol, undefined, { sensitivity: 'base' })
            )
    })

    const treeNodes = computed<TreeNode[]>(() => {
        const allLots = lots.value ?? []
        const accMap = accountMap.value

        return productPositions.value.map((pos) => {
            const openLots = allLots.filter(
                (l) => l.instrumentId === pos.instrumentId && (l.status === 1 || l.status === 2)
            )

            const now = new Date()

            const children: TreeNode[] = openLots.map((lot) => {
                const marketValue = lot.quantity * (pos.lastPrice ?? 0)
                const lotGainLoss = marketValue - lot.costBasis
                const lotGainLossPct = lot.costBasis !== 0 ? (lotGainLoss / lot.costBasis) * 100 : null

                let lotAnnualized: number | null = null
                if (lot.costBasis > 0) {
                    const days = Math.max(1, Math.round((now.getTime() - new Date(lot.openDate).getTime()) / (1000 * 60 * 60 * 24)))
                    const ratio = lotGainLoss / lot.costBasis
                    if (ratio > -1) {
                        lotAnnualized = (Math.pow(1 + ratio, 365 / days) - 1) * 100
                    }
                }

                return {
                    key: `lot-${lot.id}`,
                    data: {
                        rowType: 'lot',
                        instrumentId: pos.instrumentId,
                        currency: pos.currency,
                        lastPrice: pos.lastPrice,
                        accountName: accMap[lot.accountId]?.name,
                        openDate: lot.openDate,
                        quantity: lot.quantity,
                        originalQty: lot.originalQty,
                        isPartial: lot.status === 2,
                        costPerShare: lot.costPerShare,
                        costBasis: lot.costBasis,
                        marketValue,
                        lotGainLoss,
                        lotGainLossPct,
                        lotAnnualized
                    },
                    leaf: true
                }
            })

            // Instrument-level annualized: use earliest lot open date
            let annualized: number | null = null
            if (pos.investedAmount > 0 && openLots.length > 0) {
                const earliest = openLots.reduce((min, l) => l.openDate < min ? l.openDate : min, openLots[0].openDate)
                const days = Math.max(1, Math.round((now.getTime() - new Date(earliest).getTime()) / (1000 * 60 * 60 * 24)))
                const ratio = pos.winLoss / pos.investedAmount
                if (ratio > -1) {
                    annualized = (Math.pow(1 + ratio, 365 / days) - 1) * 100
                }
            }

            return {
                key: `instrument-${pos.instrumentId}`,
                data: { rowType: 'instrument', ...pos, annualized },
                children,
                leaf: children.length === 0
            }
        })
    })

    const totalByCurrency = computed(() => {
        const byCur: Record<string, number> = {}
        for (const p of productPositions.value) {
            byCur[p.currency] = (byCur[p.currency] ?? 0) + p.totalValue
        }
        return Object.entries(byCur)
            .map(([currency, value]) => ({ currency, value }))
            .sort((a, b) => a.currency.localeCompare(b.currency))
    })

    const currenciesInPositions = computed(() => {
        const set = new Set<string>()
        for (const p of productPositions.value) set.add(p.currency)
        return Array.from(set).sort()
    })

    const { data: latestRatesMap } = useQuery({
        queryKey: computed(() => ['fxLatestRates', 'investmentReport', mainCurrency.value, currenciesInPositions.value.join(',')]),
        queryFn: async () => {
            const main = mainCurrency.value
            const map: Record<string, number> = {}
            for (const currency of currenciesInPositions.value) {
                if (currency === main) continue
                const r = await getLatestRate(main, currency)
                if (r?.rate) map[currency] = r.rate
            }
            return map
        },
        enabled: computed(() => mainCurrency.value !== '' && productPositions.value.length > 0)
    })

    const totalInMainCurrency = computed(() => {
        const main = mainCurrency.value
        const rates = latestRatesMap.value ?? {}
        let total = 0
        for (const p of productPositions.value) {
            if (p.currency === main) {
                total += p.totalValue
            } else if (rates[p.currency]) {
                total += p.totalValue / rates[p.currency]
            }
        }
        return total
    })

    return {
        productPositions,
        treeNodes,
        totalByCurrency,
        totalInMainCurrency,
        mainCurrency,
        isLoading
    }
}
