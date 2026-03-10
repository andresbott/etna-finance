import { computed } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { useAccounts } from '@/composables/useAccounts'
import { useSettingsStore } from '@/store/settingsStore'
import { getLatestRate } from '@/lib/api/CurrencyRates'
import { getBalanceReport } from '@/lib/api/report'

export interface AccountTypeRow {
    type: string
    currencies: Record<string, number>
}

export function useAccountTypesData() {
    const { accounts: accountProviders } = useAccounts()
    const settingsStore = useSettingsStore()
    const mainCurrency = computed(() => settingsStore.mainCurrency || 'CHF')

    const allAccounts = computed(() => {
        if (!accountProviders.value) return []
        const accounts: Array<{ id: number; type?: string; currency?: string; reportData?: Array<{ Sum: number }> }> = []
        for (const provider of accountProviders.value) {
            if (provider.accounts && Array.isArray(provider.accounts)) {
                accounts.push(...provider.accounts)
            }
        }
        return accounts
    })

    const balanceReportQuery = useQuery({
        queryKey: computed(() => {
            const ids = allAccounts.value.map((a) => a.id).filter(Boolean)
            return ['balanceReport', ...ids]
        }),
        queryFn: () => {
            const accountIds = allAccounts.value.map((a) => a.id).filter(Boolean)
            const oneYearAgo = new Date()
            oneYearAgo.setFullYear(oneYearAgo.getFullYear() - 1)
            return getBalanceReport(accountIds, 30, oneYearAgo.toISOString().split('T')[0])
        },
        enabled: computed(() => allAccounts.value.length > 0)
    })

    const balanceReport = computed(() => balanceReportQuery.data.value)

    function getLatestBalance(account: { reportData?: Array<{ Sum: number }> }): number {
        if (!account.reportData || account.reportData.length === 0) return 0
        const latestEntry = account.reportData[account.reportData.length - 1]
        return latestEntry?.Sum ?? 0
    }

    const accountsByType = computed<AccountTypeRow[]>(() => {
        if (!allAccounts.value || !balanceReport.value) return []
        const accountsWithBalances = allAccounts.value
            .map((account) => {
                const reportData = balanceReport.value?.accounts?.[account.id]
                if (!reportData) return null
                return { ...account, reportData }
            })
            .filter(Boolean) as Array<{ id: number; type?: string; currency?: string; reportData: Array<{ Sum: number }> }>

        const grouped: Record<string, AccountTypeRow> = {}
        for (const account of accountsWithBalances) {
            const type = account.type || 'Other'
            const currency = account.currency || 'CHF'
            if (!grouped[type]) {
                grouped[type] = { type, currencies: {} }
            }
            if (!grouped[type].currencies[currency]) {
                grouped[type].currencies[currency] = 0
            }
            grouped[type].currencies[currency] += getLatestBalance(account)
        }
        return Object.values(grouped)
    })

    const allCurrencies = computed(() => {
        const currencies = new Set<string>()
        accountsByType.value.forEach((typeGroup) => {
            Object.keys(typeGroup.currencies).forEach((c) => currencies.add(c))
        })
        return Array.from(currencies).sort()
    })

    const currencyListKey = computed(() => allCurrencies.value.join(','))
    const { data: latestRatesMap } = useQuery({
        queryKey: computed(() => ['fxLatestRates', mainCurrency.value, currencyListKey.value]),
        queryFn: async () => {
            const main = mainCurrency.value
            const map: Record<string, number> = {}
            for (const currency of allCurrencies.value) {
                if (currency === main) continue
                const r = await getLatestRate(main, currency)
                if (r?.rate) map[currency] = r.rate
            }
            return map
        },
        enabled: computed(() => mainCurrency.value !== '' && allCurrencies.value.length > 0)
    })

    function totalInMainCurrency(row: AccountTypeRow): number {
        const main = mainCurrency.value
        const rates = latestRatesMap.value ?? {}
        let total = 0
        for (const [currency, amount] of Object.entries(row.currencies)) {
            if (currency === main) {
                total += amount
            } else if (rates[currency]) {
                total += amount / rates[currency]
            }
        }
        return total
    }

    return {
        accountsByType,
        totalInMainCurrency,
        mainCurrency
    }
}
