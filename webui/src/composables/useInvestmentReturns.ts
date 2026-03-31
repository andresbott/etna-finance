import { computed } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { useHoldings } from '@/composables/useHoldings'
import { useInstruments } from '@/composables/useInstruments'
import { useSettingsStore } from '@/store/settingsStore'
import { getLatestRate } from '@/lib/api/CurrencyRates'
import { getInstrumentReturns } from '@/lib/api/Portfolio'

export interface InstrumentReturnRow {
    instrumentId: number
    symbol: string
    name: string
    currency: string
    totalInvested: number
    realizedProceeds: number
    realizedGL: number
    unrealizedGL: number
    totalReturn: number
    currentQuantity: number
    currentValue: number
    currentCostBasis: number
    roi: number | null           // totalReturn / totalInvested * 100
    annualizedReturn: number | null  // annualized ROI
    firstTradeDate: string
    lastTradeDate: string
    holdingDays: number
    status: 'open' | 'closed' | 'mixed'
}

export function useInvestmentReturns() {
    const settingsStore = useSettingsStore()
    const mainCurrency = computed(() => settingsStore.mainCurrency || 'CHF')
    const { providersWithHoldings, isLoading: holdingsLoading } = useHoldings()
    const { instruments } = useInstruments()

    const { data: returnsData, isLoading: returnsLoading } = useQuery({
        queryKey: ['portfolio-returns'],
        queryFn: () => getInstrumentReturns()
    })

    const isLoading = computed(() => holdingsLoading.value || returnsLoading.value)

    const instrumentsMap = computed(() => {
        const list = instruments.value ?? []
        return Object.fromEntries(list.map((i) => [i.id, i]))
    })

    // Build a map of current market value per instrument from holdings
    const marketValueMap = computed(() => {
        const map: Record<number, { value: number; lastPrice: number | null; currency: string }> = {}
        const provs = providersWithHoldings.value ?? []
        for (const provider of provs) {
            for (const account of provider.accounts) {
                for (const h of account.holdings) {
                    if (h.quantity <= 0) continue
                    const existing = map[h.instrumentId]
                    if (existing) {
                        existing.value += h.value
                    } else {
                        map[h.instrumentId] = {
                            value: h.value,
                            lastPrice: h.lastPrice,
                            currency: h.currency || 'CHF'
                        }
                    }
                }
            }
        }
        return map
    })

    const returnRows = computed<InstrumentReturnRow[]>(() => {
        const returns = returnsData.value ?? []
        const instMap = instrumentsMap.value
        const mvMap = marketValueMap.value
        const now = new Date()

        return returns.map((ret) => {
            const inst = instMap[ret.instrumentId]
            const mv = mvMap[ret.instrumentId]
            const currentValue = mv?.value ?? 0
            const currency = inst?.currency || mv?.currency || 'CHF'

            const unrealizedGL = ret.currentQuantity > 0
                ? currentValue - ret.currentCostBasis
                : 0
            const totalReturn = ret.realizedGL + unrealizedGL

            const firstDate = new Date(ret.firstTradeDate)
            const holdingDays = Math.max(1, Math.round((now.getTime() - firstDate.getTime()) / (1000 * 60 * 60 * 24)))

            let roi: number | null = null
            let annualizedReturn: number | null = null
            if (ret.totalInvested > 0) {
                roi = (totalReturn / ret.totalInvested) * 100
                const totalReturnRatio = totalReturn / ret.totalInvested
                if (totalReturnRatio > -1) {
                    annualizedReturn = (Math.pow(1 + totalReturnRatio, 365 / holdingDays) - 1) * 100
                }
            }

            let status: 'open' | 'closed' | 'mixed' = 'open'
            if (ret.currentQuantity <= 0 && ret.realizedProceeds > 0) {
                status = 'closed'
            } else if (ret.currentQuantity > 0 && ret.realizedProceeds > 0) {
                status = 'mixed'
            }

            return {
                instrumentId: ret.instrumentId,
                symbol: inst?.symbol || `#${ret.instrumentId}`,
                name: inst?.name || `Product #${ret.instrumentId}`,
                currency,
                totalInvested: ret.totalInvested,
                realizedProceeds: ret.realizedProceeds,
                realizedGL: ret.realizedGL,
                unrealizedGL,
                totalReturn,
                currentQuantity: ret.currentQuantity,
                currentValue,
                currentCostBasis: ret.currentCostBasis,
                roi,
                annualizedReturn,
                firstTradeDate: ret.firstTradeDate,
                lastTradeDate: ret.lastTradeDate,
                holdingDays,
                status
            }
        }).sort((a, b) => a.symbol.localeCompare(b.symbol, undefined, { sensitivity: 'base' }))
    })

    // Totals in main currency
    const currenciesInReturns = computed(() => {
        const set = new Set<string>()
        for (const r of returnRows.value) set.add(r.currency)
        return Array.from(set).sort()
    })

    const { data: latestRatesMap } = useQuery({
        queryKey: computed(() => ['fxLatestRates', 'investmentReturns', mainCurrency.value, currenciesInReturns.value.join(',')]),
        queryFn: async () => {
            const main = mainCurrency.value
            const map: Record<string, number> = {}
            for (const currency of currenciesInReturns.value) {
                if (currency === main) continue
                const r = await getLatestRate(main, currency)
                if (r?.rate) map[currency] = r.rate
            }
            return map
        },
        enabled: computed(() => mainCurrency.value !== '' && returnRows.value.length > 0)
    })

    const totals = computed(() => {
        const main = mainCurrency.value
        const rates = latestRatesMap.value ?? {}
        let totalInvested = 0
        let totalReturn = 0
        let realizedGL = 0
        let unrealizedGL = 0

        for (const r of returnRows.value) {
            const fx = r.currency === main ? 1 : (rates[r.currency] ? 1 / rates[r.currency] : 0)
            totalInvested += r.totalInvested * fx
            totalReturn += r.totalReturn * fx
            realizedGL += r.realizedGL * fx
            unrealizedGL += r.unrealizedGL * fx
        }

        let annualizedReturn: number | null = null
        if (totalInvested > 0) {
            const rows = returnRows.value
            const now = new Date()
            let earliestDate = now
            for (const r of rows) {
                const d = new Date(r.firstTradeDate)
                if (d < earliestDate) earliestDate = d
            }
            const holdingDays = Math.max(1, Math.round((now.getTime() - earliestDate.getTime()) / (1000 * 60 * 60 * 24)))
            const totalReturnRatio = totalReturn / totalInvested
            if (totalReturnRatio > -1) {
                annualizedReturn = (Math.pow(1 + totalReturnRatio, 365 / holdingDays) - 1) * 100
            }
        }
        return { totalInvested, totalReturn, realizedGL, unrealizedGL, annualizedReturn, currency: main }
    })

    return {
        returnRows,
        totals,
        mainCurrency,
        isLoading
    }
}
