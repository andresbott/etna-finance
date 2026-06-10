import { describe, it, expect, vi, afterEach, type Mock } from 'vitest'
import { AxiosError } from 'axios'
import { apiClient } from '@/lib/api/client'
import { lookupInstrument, LookupRateLimitError } from './Instrument'

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

    it('throws LookupRateLimitError on 429 with Retry-After', async () => {
        const err = new AxiosError('rate limited')
        err.response = {
            status: 429,
            headers: { 'retry-after': '30' },
            data: '',
            statusText: 'Too Many Requests',
            config: {} as never
        }
        mockGet.mockRejectedValue(err)
        await expect(lookupInstrument('ADBE')).rejects.toMatchObject({
            name: 'LookupRateLimitError',
            retryAfterSeconds: 30
        })
        await expect(lookupInstrument('ADBE')).rejects.toBeInstanceOf(LookupRateLimitError)
    })
})
