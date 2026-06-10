import { describe, it, expect, vi, afterEach, type Mock } from 'vitest'
import { apiClient } from '@/lib/api/client'
import { lookupInstrument } from './Instrument'

vi.mock('@/lib/api/client', () => ({
    apiClient: { get: vi.fn() }
}))

const mockGet = apiClient.get as Mock

describe('lookupInstrument', () => {
    afterEach(() => vi.clearAllMocks())

    it('returns the details on 200', async () => {
        mockGet.mockResolvedValue({
            status: 200,
            data: { name: 'Apple Inc.', currency: 'USD', type: 'Stock', exchange: 'NASDAQ', notes: 'n' }
        })
        const res = await lookupInstrument('AAPL')
        expect(mockGet).toHaveBeenCalledWith('/fin/instrument/lookup', { params: { symbol: 'AAPL' } })
        expect(res?.name).toBe('Apple Inc.')
    })

    it('returns null on 204', async () => {
        mockGet.mockResolvedValue({ status: 204, data: '' })
        const res = await lookupInstrument('NOPE')
        expect(res).toBeNull()
    })
})
