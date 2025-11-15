import { useMutation } from '@tanstack/vue-query'
import { getBalanceReport } from '@/lib/api/report'

export const useGetBalanceReport = () => {
    return useMutation({
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
}
