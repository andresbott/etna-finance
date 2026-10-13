import { apiClient } from '@/lib/api/client'
import type { BalanceReport } from '@/types/report'

export const getBalanceReport = async (
    accountIds: number[],
    steps: number,
    startDate: string
): Promise<BalanceReport> => {
    const idsParam = accountIds.join(',')
    const { data } = await apiClient.get<BalanceReport>(
        `/fin/report/balance?accountIds=${idsParam}&steps=${steps}&startDate=${startDate}`
    )

    return data
}

/**
 * Current balance for many accounts in a single request, each in the account's own currency.
 * Returns a map of accountId -> balance (0 when an account has no data).
 * @param accountIds - Account IDs to fetch balances for
 * @param date - YYYY-MM-DD; balance is computed up to and including this date
 */
export const getAccountBalances = async (
    accountIds: number[],
    date: string
): Promise<Record<number, number>> => {
    const result: Record<number, number> = {}
    if (accountIds.length === 0) return result

    const idsParam = accountIds.join(',')
    const { data } = await apiClient.get<BalanceReport>(
        `/fin/report/balance?accountIds=${idsParam}&steps=1&endDate=${date}`
    )

    for (const id of accountIds) {
        result[id] = data?.accounts?.[id]?.[0]?.nativeBalance ?? 0
    }
    return result
}

/**
 * Balance of one account in its own currency, up to and including the given date (YYYY-MM-DD).
 */
export const getAccountBalance = async (
    accountId: number,
    date: string
): Promise<number> => {
    const { data } = await apiClient.get<BalanceReport>(
        `/fin/report/balance?accountIds=${accountId}&steps=1&endDate=${date}`
    )

    // Extract the balance for the specific account at the given date
    const accountData = data?.accounts?.[accountId]
    if (!accountData || accountData.length === 0) {
        return 0
    }

    // Return the native balance from the first (and only) entry
    return accountData[0]?.nativeBalance ?? 0
}

/**
 * Income/expense report for the given date range.
 * @param startDate - YYYY-MM-DD
 * @param endDate - YYYY-MM-DD
 */
export const getIncomeExpenseReport = async (
    startDate: string,
    endDate: string
): Promise<Record<string, unknown>[]> => {
    const params = new URLSearchParams({ startDate, endDate })
    const { data } = await apiClient.get(`/fin/report/income-expense?${params}`)
    return data ?? []
}
