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
