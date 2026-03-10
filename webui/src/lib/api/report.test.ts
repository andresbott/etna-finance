import { describe, it, expect, vi, beforeEach } from 'vitest'
import { getBalanceReport, getAccountBalance, getIncomeExpenseReport } from '@/lib/api/report'

vi.mock('./client', () => ({
    apiClient: { get: vi.fn() },
}))

import { apiClient } from './client'

const mockGet = vi.mocked(apiClient.get)

beforeEach(() => {
    mockGet.mockReset()
})

// ---------------------------------------------------------------------------
// getBalanceReport
// ---------------------------------------------------------------------------
describe('getBalanceReport', () => {
    it('builds the URL with comma-joined account IDs, steps, and startDate', async () => {
        mockGet.mockResolvedValue({ data: { accounts: {} } })

        await getBalanceReport([1, 2, 3], 12, '2025-01-01')

        expect(mockGet).toHaveBeenCalledWith(
            '/fin/report/balance?accountIds=1,2,3&steps=12&startDate=2025-01-01'
        )
    })

    it('handles a single account ID', async () => {
        mockGet.mockResolvedValue({ data: { accounts: {} } })

        await getBalanceReport([42], 6, '2024-06-01')

        expect(mockGet).toHaveBeenCalledWith(
            '/fin/report/balance?accountIds=42&steps=6&startDate=2024-06-01'
        )
    })

    it('handles an empty account IDs array', async () => {
        mockGet.mockResolvedValue({ data: {} })

        await getBalanceReport([], 1, '2025-01-01')

        expect(mockGet).toHaveBeenCalledWith(
            '/fin/report/balance?accountIds=&steps=1&startDate=2025-01-01'
        )
    })

    it('returns the data from the response', async () => {
        const payload = { accounts: { 1: [{ Sum: 100 }] } }
        mockGet.mockResolvedValue({ data: payload })

        const result = await getBalanceReport([1], 1, '2025-01-01')

        expect(result).toEqual(payload)
    })

    it('propagates network errors', async () => {
        mockGet.mockRejectedValue(new Error('Network Error'))

        await expect(getBalanceReport([1], 1, '2025-01-01')).rejects.toThrow('Network Error')
    })
})

// ---------------------------------------------------------------------------
// getAccountBalance
// ---------------------------------------------------------------------------
describe('getAccountBalance', () => {
    it('builds URL with single accountId, steps=1, and endDate', async () => {
        mockGet.mockResolvedValue({ data: { accounts: { 5: [{ Sum: 250 }] } } })

        await getAccountBalance(5, '2025-03-10')

        expect(mockGet).toHaveBeenCalledWith(
            '/fin/report/balance?accountIds=5&steps=1&endDate=2025-03-10'
        )
    })

    it('extracts the Sum from the first entry of the matching account', async () => {
        mockGet.mockResolvedValue({
            data: { accounts: { 7: [{ Sum: 1234.56 }] } },
        })

        const result = await getAccountBalance(7, '2025-01-01')

        expect(result).toBe(1234.56)
    })

    it('returns 0 when the account key is missing', async () => {
        mockGet.mockResolvedValue({ data: { accounts: {} } })

        const result = await getAccountBalance(99, '2025-01-01')

        expect(result).toBe(0)
    })

    it('returns 0 when the account array is empty', async () => {
        mockGet.mockResolvedValue({ data: { accounts: { 10: [] } } })

        const result = await getAccountBalance(10, '2025-01-01')

        expect(result).toBe(0)
    })

    it('returns 0 when Sum is missing from the first entry', async () => {
        mockGet.mockResolvedValue({
            data: { accounts: { 10: [{ Other: 500 }] } },
        })

        const result = await getAccountBalance(10, '2025-01-01')

        expect(result).toBe(0)
    })

    it('returns 0 when Sum is 0', async () => {
        mockGet.mockResolvedValue({
            data: { accounts: { 10: [{ Sum: 0 }] } },
        })

        const result = await getAccountBalance(10, '2025-01-01')

        expect(result).toBe(0)
    })

    it('returns 0 when data is null', async () => {
        mockGet.mockResolvedValue({ data: null })

        const result = await getAccountBalance(1, '2025-01-01')

        expect(result).toBe(0)
    })

    it('returns 0 when data.accounts is undefined', async () => {
        mockGet.mockResolvedValue({ data: {} })

        const result = await getAccountBalance(1, '2025-01-01')

        expect(result).toBe(0)
    })

    it('uses only the first entry even when multiple entries exist', async () => {
        mockGet.mockResolvedValue({
            data: { accounts: { 3: [{ Sum: 100 }, { Sum: 999 }] } },
        })

        const result = await getAccountBalance(3, '2025-01-01')

        expect(result).toBe(100)
    })

    it('propagates network errors', async () => {
        mockGet.mockRejectedValue(new Error('timeout'))

        await expect(getAccountBalance(1, '2025-01-01')).rejects.toThrow('timeout')
    })
})

// ---------------------------------------------------------------------------
// getIncomeExpenseReport
// ---------------------------------------------------------------------------
describe('getIncomeExpenseReport', () => {
    it('builds URL with URLSearchParams for startDate and endDate', async () => {
        mockGet.mockResolvedValue({ data: [{ category: 'Food', amount: 50 }] })

        await getIncomeExpenseReport('2025-01-01', '2025-01-31')

        expect(mockGet).toHaveBeenCalledWith(
            '/fin/report/income-expense?startDate=2025-01-01&endDate=2025-01-31'
        )
    })

    it('returns the data from the response', async () => {
        const payload = [{ category: 'Salary', amount: 3000 }]
        mockGet.mockResolvedValue({ data: payload })

        const result = await getIncomeExpenseReport('2025-01-01', '2025-01-31')

        expect(result).toEqual(payload)
    })

    it('returns an empty array when data is null', async () => {
        mockGet.mockResolvedValue({ data: null })

        const result = await getIncomeExpenseReport('2025-01-01', '2025-01-31')

        expect(result).toEqual([])
    })

    it('returns an empty array when data is undefined', async () => {
        mockGet.mockResolvedValue({ data: undefined })

        const result = await getIncomeExpenseReport('2025-01-01', '2025-01-31')

        expect(result).toEqual([])
    })

    it('returns data as-is when it is a non-empty array', async () => {
        const payload = [{ a: 1 }, { b: 2 }]
        mockGet.mockResolvedValue({ data: payload })

        const result = await getIncomeExpenseReport('2025-06-01', '2025-06-30')

        expect(result).toEqual(payload)
    })

    it('propagates network errors', async () => {
        mockGet.mockRejectedValue(new Error('500'))

        await expect(getIncomeExpenseReport('2025-01-01', '2025-01-31')).rejects.toThrow('500')
    })
})
