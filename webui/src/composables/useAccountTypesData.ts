import { computed } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { useAccounts } from '@/composables/useAccounts'
import { useSettingsStore } from '@/store/settingsStore'
import { getBalanceReport } from '@/lib/api/report'
import { toLocalDateString } from '@/utils/date'
import { ACCOUNT_TYPES } from '@/types/account'

export interface AccountTypeRow {
    type: string
    total: number
}

export function useAccountTypesData() {
    const { accounts: accountProviders } = useAccounts()
    const settingsStore = useSettingsStore()
    const mainCurrency = computed(() => settingsStore.mainCurrency || 'CHF')

    const allAccounts = computed(() => {
        if (!accountProviders.value) return []
        const accounts: Array<{ id: number; type?: string; currency?: string }> = []
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
            return getBalanceReport(accountIds, 30, toLocalDateString(oneYearAgo))
        },
        enabled: computed(() => allAccounts.value.length > 0)
    })

    const balanceReport = computed(() => balanceReportQuery.data.value)

    function getLatestBalance(reportData: Array<{ sum: number }>): number {
        if (reportData.length === 0) return 0
        return reportData[reportData.length - 1]?.sum ?? 0
    }

    const ACCOUNT_TYPE_ORDER: string[] = [
        ACCOUNT_TYPES.CASH,
        ACCOUNT_TYPES.CHECKING,
        ACCOUNT_TYPES.SAVINGS,
        ACCOUNT_TYPES.LENT,
        ACCOUNT_TYPES.PREPAID_EXPENSE,
        ACCOUNT_TYPES.PENSION,
        ACCOUNT_TYPES.INVESTMENT,
        ACCOUNT_TYPES.RESTRICTED_STOCK,
    ]

    // Balance report values are already converted to main currency by the backend,
    // so we sum them directly per account type without additional FX conversion.
    const accountsByType = computed<AccountTypeRow[]>(() => {
        if (!allAccounts.value) return []
        const grouped: Record<string, AccountTypeRow> = {}
        for (const account of allAccounts.value) {
            const type = account.type || 'Other'

            const reportData = balanceReport.value?.accounts?.[account.id]
            if (!reportData) continue
            const balance = getLatestBalance(reportData)

            if (!grouped[type]) {
                grouped[type] = { type, total: 0 }
            }
            grouped[type].total += balance
        }
        return Object.values(grouped).sort((a, b) => {
            const ai = ACCOUNT_TYPE_ORDER.indexOf(a.type)
            const bi = ACCOUNT_TYPE_ORDER.indexOf(b.type)
            return (ai === -1 ? 999 : ai) - (bi === -1 ? 999 : bi)
        })
    })

    function totalInMainCurrency(row: AccountTypeRow): number {
        return row.total
    }

    return {
        accountsByType,
        totalInMainCurrency,
        mainCurrency
    }
}
