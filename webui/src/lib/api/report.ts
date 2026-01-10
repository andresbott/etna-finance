import { apiClient } from './client'

export const getBalanceReport = async (
    accountIds: number[],
    steps: number,
    startDate: string
): Promise<void> => {
    const idsParam = accountIds.join(',')
    const { data } = await apiClient.get(
        `/fin/report/balance?accountIds=${idsParam}&steps=${steps}&startDate=${startDate}`
    )

    return data
}

export const getAccountBalance = async (
    accountId: number,
    date: string
): Promise<number> => {
    const { data } = await apiClient.get(
        `/fin/report/balance?accountIds=${accountId}&steps=1&endDate=${date}`
    )

    // Extract the balance for the specific account at the given date
    const accountData = data?.accounts?.[accountId]
    if (!accountData || accountData.length === 0) {
        return 0
    }
    
    // Return the Sum from the first (and only) entry
    return accountData[0]?.Sum || 0
}
