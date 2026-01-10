import { useMutation } from '@tanstack/vue-query'
import { getBalanceReport, getAccountBalance } from '@/lib/api/report'

export const useBalance = () => {
    const balanceReport = useMutation({
        mutationFn: ({
            accountIds,
            steps,
            startDate
        }: {
            accountIds: number[]
            steps: number
            startDate: string
        }) => getBalanceReport(accountIds, steps, startDate)
    })

    const accountBalance = useMutation({
        mutationFn: ({
            accountId,
            date
        }: {
            accountId: number
            date: string
        }) => getAccountBalance(accountId, date)
    })

    return {
        balanceReport,
        accountBalance
    }
}

// Keep backward compatibility
export const useGetBalanceReport = useBalance
