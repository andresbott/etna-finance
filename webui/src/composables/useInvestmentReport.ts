import { computed } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { useHoldings } from '@/composables/useHoldings'
import { useInstruments } from '@/composables/useInstruments'
import { useSettingsStore } from '@/store/settingsStore'
import { getLatestRate } from '@/lib/api/CurrencyRates'

export interface ProductPosition {
    instrumentId: number
    symbol: string
    name: string
    currency: string
    totalQuantity: number
    lastPrice: number | null
    totalValue: number
    investedAmount: number
    winLoss: number // totalValue - investedAmount (unrealized gain when positive, loss when negative)
}

/**
 * Aggregates holdings by investment product for the investment report.
 */
export function useInvestmentReport() {
    const settingsStore = useSettingsStore()
    const mainCurrency = computed(() => settingsStore.mainCurrency || 'CHF')
    const { providersWithHoldings, isLoading } = useHoldings()
    const { instruments } = useInstruments()

    const instrumentsMap = computed(() => {
        const list = instruments.value ?? []
        return Object.fromEntries(list.map((i) => [i.id, i]))
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
            .map((row) => ({
                ...row,
                winLoss: row.totalValue - row.investedAmount
            }))
            .sort((a, b) =>
                a.symbol.localeCompare(b.symbol, undefined, { sensitivity: 'base' })
            )
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
        totalByCurrency,
        totalInMainCurrency,
        mainCurrency,
        isLoading
    }
}
