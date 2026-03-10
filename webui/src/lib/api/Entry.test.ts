import { describe, it, expect, vi, beforeEach } from 'vitest'
import { formatDate } from '@/lib/api/Entry'

vi.mock('./client', () => ({
    apiClient: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}))

describe('formatDate', () => {
    it('formats a normal date as YYYY-MM-DD using local date parts', () => {
        // Use a date where UTC vs local could differ — but we test the local parts
        const d = new Date(2024, 0, 15) // Jan 15, 2024 local
        expect(formatDate(d)).toBe('2024-01-15')
    })

    it('zero-pads single-digit months and days', () => {
        const d = new Date(2023, 2, 5) // March 5
        expect(formatDate(d)).toBe('2023-03-05')
    })

    it('handles the last day of the year', () => {
        const d = new Date(2025, 11, 31) // Dec 31
        expect(formatDate(d)).toBe('2025-12-31')
    })

    it('returns empty string for an invalid date', () => {
        expect(formatDate(new Date('not-a-date'))).toBe('')
    })

    it('returns empty string when given a non-Date value', () => {
        // The runtime guard checks instanceof Date
        expect(formatDate('2024-01-01' as unknown as Date)).toBe('')
        expect(formatDate(null as unknown as Date)).toBe('')
        expect(formatDate(undefined as unknown as Date)).toBe('')
    })
})

// ---------------------------------------------------------------------------
// API wrapper tests
// ---------------------------------------------------------------------------
import {
    getEntries,
    createEntry,
    updateEntry,
    getEntry,
    deleteEntry,
    createStockTransaction,
    createStockGrant,
    createStockTransfer,
} from '@/lib/api/Entry'
import { apiClient } from '@/lib/api/client'

const mockedClient = vi.mocked(apiClient, true)

describe('Entry API wrappers', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    // -- getEntries ---------------------------------------------------------

    describe('getEntries', () => {
        it('builds query string with dates and default pagination', async () => {
            const items = [{ id: 1 }]
            mockedClient.get.mockResolvedValue({
                data: { items, total: 1, priorBalance: 100 },
            })

            const result = await getEntries({
                startDate: new Date(2024, 0, 1),
                endDate: new Date(2024, 0, 31),
            })

            expect(mockedClient.get).toHaveBeenCalledOnce()
            const url: string = mockedClient.get.mock.calls[0][0]
            expect(url).toContain('/fin/entries?')
            const params = new URLSearchParams(url.split('?')[1])
            expect(params.get('startDate')).toBe('2024-01-01')
            expect(params.get('endDate')).toBe('2024-01-31')
            expect(params.get('page')).toBe('1')
            expect(params.get('limit')).toBe('25')

            expect(result).toEqual({
                items,
                total: 1,
                page: 1,
                limit: 25,
                priorBalance: 100,
            })
        })

        it('includes custom page and limit', async () => {
            mockedClient.get.mockResolvedValue({
                data: { items: [], total: 0, priorBalance: 0 },
            })

            await getEntries({
                startDate: new Date(2024, 5, 1),
                endDate: new Date(2024, 5, 30),
                page: 3,
                limit: 50,
            })

            const url: string = mockedClient.get.mock.calls[0][0]
            const params = new URLSearchParams(url.split('?')[1])
            expect(params.get('page')).toBe('3')
            expect(params.get('limit')).toBe('50')
        })

        it('appends multiple accountIds', async () => {
            mockedClient.get.mockResolvedValue({
                data: { items: [], total: 0, priorBalance: 0 },
            })

            await getEntries({
                startDate: new Date(2024, 0, 1),
                endDate: new Date(2024, 0, 31),
                accountIds: ['a1', 'a2', 'a3'],
            })

            const url: string = mockedClient.get.mock.calls[0][0]
            const params = new URLSearchParams(url.split('?')[1])
            expect(params.getAll('accountIds')).toEqual(['a1', 'a2', 'a3'])
        })

        it('defaults items to [] and totals to 0 when response has no values', async () => {
            mockedClient.get.mockResolvedValue({ data: {} })

            const result = await getEntries({
                startDate: new Date(2024, 0, 1),
                endDate: new Date(2024, 0, 31),
            })

            expect(result.items).toEqual([])
            expect(result.total).toBe(0)
            expect(result.priorBalance).toBe(0)
        })
    })

    // -- createEntry --------------------------------------------------------

    describe('createEntry', () => {
        it('posts payload to /fin/entries and returns data', async () => {
            const payload = { description: 'Test', date: '2024-01-15', amount: 100 } as any
            const created = { id: 42, ...payload }
            mockedClient.post.mockResolvedValue({ data: created })

            const result = await createEntry(payload)

            expect(mockedClient.post).toHaveBeenCalledWith('/fin/entries', payload)
            expect(result).toEqual(created)
        })
    })

    // -- updateEntry --------------------------------------------------------

    describe('updateEntry', () => {
        it('puts payload to /fin/entries/:id and returns data', async () => {
            const payload = { id: '7', description: 'Updated' } as any
            const updated = { ...payload }
            mockedClient.put.mockResolvedValue({ data: updated })

            const result = await updateEntry(payload)

            expect(mockedClient.put).toHaveBeenCalledWith('/fin/entries/7', payload)
            expect(result).toEqual(updated)
        })
    })

    // -- getEntry -----------------------------------------------------------

    describe('getEntry', () => {
        it('fetches a single entry by string id', async () => {
            const entry = { id: '5', description: 'Single' }
            mockedClient.get.mockResolvedValue({ data: entry })

            const result = await getEntry('5')

            expect(mockedClient.get).toHaveBeenCalledWith('/fin/entries/5')
            expect(result).toEqual(entry)
        })

        it('fetches a single entry by numeric id', async () => {
            mockedClient.get.mockResolvedValue({ data: { id: 10 } })

            await getEntry(10)

            expect(mockedClient.get).toHaveBeenCalledWith('/fin/entries/10')
        })
    })

    // -- deleteEntry --------------------------------------------------------

    describe('deleteEntry', () => {
        it('sends DELETE to /fin/entries/:id', async () => {
            mockedClient.delete.mockResolvedValue({})

            await deleteEntry('99')

            expect(mockedClient.delete).toHaveBeenCalledWith('/fin/entries/99')
        })
    })

    // -- createStockTransaction ---------------------------------------------

    describe('createStockTransaction', () => {
        it('posts stock buy payload and returns data', async () => {
            const payload = {
                type: 'stockbuy' as const,
                description: 'Buy AAPL',
                date: '2024-03-01',
                instrumentId: 1,
                quantity: 10,
                totalAmount: 1500,
                investmentAccountId: 2,
                cashAccountId: 3,
            }
            const response = { id: 100, ...payload }
            mockedClient.post.mockResolvedValue({ data: response })

            const result = await createStockTransaction(payload)

            expect(mockedClient.post).toHaveBeenCalledWith('/fin/entries', payload)
            expect(result).toEqual(response)
        })

        it('posts stock sell payload', async () => {
            const payload = {
                type: 'stocksell' as const,
                description: 'Sell TSLA',
                date: '2024-04-01',
                instrumentId: 5,
                quantity: 3,
                totalAmount: 900,
                investmentAccountId: 2,
                cashAccountId: 3,
            }
            mockedClient.post.mockResolvedValue({ data: { id: 101 } })

            await createStockTransaction(payload)

            expect(mockedClient.post).toHaveBeenCalledWith('/fin/entries', payload)
        })
    })

    // -- createStockGrant ---------------------------------------------------

    describe('createStockGrant', () => {
        it('posts stock grant payload and returns data', async () => {
            const payload = {
                type: 'stockgrant' as const,
                description: 'RSU vest',
                date: '2024-06-01',
                instrumentId: 2,
                quantity: 50,
                fairMarketValue: 150.5,
                accountId: 10,
            }
            const response = { id: 200, ...payload }
            mockedClient.post.mockResolvedValue({ data: response })

            const result = await createStockGrant(payload)

            expect(mockedClient.post).toHaveBeenCalledWith('/fin/entries', payload)
            expect(result).toEqual(response)
        })

        it('works without optional fairMarketValue', async () => {
            const payload = {
                type: 'stockgrant' as const,
                description: 'Grant',
                date: '2024-06-01',
                instrumentId: 2,
                quantity: 10,
                accountId: 10,
            }
            mockedClient.post.mockResolvedValue({ data: { id: 201 } })

            await createStockGrant(payload)

            expect(mockedClient.post).toHaveBeenCalledWith('/fin/entries', payload)
        })
    })

    // -- createStockTransfer ------------------------------------------------

    describe('createStockTransfer', () => {
        it('posts stock transfer payload and returns data', async () => {
            const payload = {
                type: 'stocktransfer' as const,
                description: 'Vest transfer',
                date: '2024-07-15',
                instrumentId: 3,
                quantity: 25,
                originAccountId: 10,
                targetAccountId: 11,
            }
            const response = { id: 300, ...payload }
            mockedClient.post.mockResolvedValue({ data: response })

            const result = await createStockTransfer(payload)

            expect(mockedClient.post).toHaveBeenCalledWith('/fin/entries', payload)
            expect(result).toEqual(response)
        })
    })
})
