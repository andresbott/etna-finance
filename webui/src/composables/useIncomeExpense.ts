import { getIncomeExpenseReport } from '@/lib/api/report'

/**
 * Fetches income/expense report for the given date range.
 * @param startDate - YYYY-MM-DD
 * @param endDate - YYYY-MM-DD
 */
export async function fetchIncomeExpense(
    startDate: string,
    endDate: string
): Promise<unknown> {
    const result = await getIncomeExpenseReport(startDate, endDate)
    return result ?? []
}
