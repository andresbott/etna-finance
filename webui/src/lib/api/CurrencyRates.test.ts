import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { apiClient } from './client'
import {
    parsePair,
    getFXPairs,
    getRateHistory,
    getLatestRate,
    createRate,
    createRatesBulk,
    updateRate,
    deleteRate,
    type RateRecord,
    type CreateRateDTO,
} from '@/lib/api/CurrencyRates'

vi.mock('./client', () => ({
    apiClient: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}))

beforeEach(() => vi.clearAllMocks())

const mockRate: RateRecord = {
    id: 1,
    main: 'USD',
    secondary: 'EUR',
    time: '2025-06-01',
    rate: 0.92,
}

describe('parsePair', () => {
    it('splits a standard "MAIN/SECONDARY" pair', () => {
        expect(parsePair('USD/EUR')).toEqual(['USD', 'EUR'])
    })

    it('handles lowercase pairs', () => {
        expect(parsePair('btc/usd')).toEqual(['btc', 'usd'])
    })

    it('returns [pair, ""] when there is no slash', () => {
        expect(parsePair('USDEUR')).toEqual(['USDEUR', ''])
    })

    it('returns ["", ""] for an empty string', () => {
        expect(parsePair('')).toEqual(['', ''])
    })

    it('splits on the first slash only', () => {
        expect(parsePair('A/B/C')).toEqual(['A', 'B/C'])
    })

    it('handles a leading slash', () => {
        expect(parsePair('/EUR')).toEqual(['', 'EUR'])
    })

    it('handles a trailing slash', () => {
        expect(parsePair('USD/')).toEqual(['USD', ''])
    })
})

describe('getFXPairs', () => {
    it('calls GET /fin/fx/pairs and returns the pairs array', async () => {
        const pairs = ['USD/EUR', 'GBP/USD'];
        (apiClient.get as Mock).mockResolvedValue({ data: { pairs } })

        const result = await getFXPairs()

        expect(apiClient.get).toHaveBeenCalledWith('/fin/fx/pairs')
        expect(apiClient.get).toHaveBeenCalledTimes(1)
        expect(result).toEqual(pairs)
    })

    it('returns empty array when pairs is undefined', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: {} })

        const result = await getFXPairs()

        expect(result).toEqual([])
    })

    it('returns empty array when pairs is null', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { pairs: null } })

        const result = await getFXPairs()

        expect(result).toEqual([])
    })
})

describe('getRateHistory', () => {
    it('calls GET with URL-encoded currency names and no query string when dates are omitted', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { items: [mockRate] } })

        const result = await getRateHistory('USD', 'EUR')

        expect(apiClient.get).toHaveBeenCalledWith('/fin/fx/USD/EUR/rates')
        expect(result).toEqual([mockRate])
    })

    it('URL-encodes currency names with special characters', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { items: [] } })

        await getRateHistory('US D', 'E/R')

        expect(apiClient.get).toHaveBeenCalledWith('/fin/fx/US%20D/E%2FR/rates')
    })

    it('appends start query param when provided', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { items: [] } })

        await getRateHistory('USD', 'EUR', '2025-01-01')

        expect(apiClient.get).toHaveBeenCalledWith('/fin/fx/USD/EUR/rates?start=2025-01-01')
    })

    it('appends end query param when provided', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { items: [] } })

        await getRateHistory('USD', 'EUR', undefined, '2025-12-31')

        expect(apiClient.get).toHaveBeenCalledWith('/fin/fx/USD/EUR/rates?end=2025-12-31')
    })

    it('appends both start and end query params when provided', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { items: [] } })

        await getRateHistory('USD', 'EUR', '2025-01-01', '2025-12-31')

        expect(apiClient.get).toHaveBeenCalledWith(
            '/fin/fx/USD/EUR/rates?start=2025-01-01&end=2025-12-31'
        )
    })

    it('returns empty array when items is undefined', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: {} })

        const result = await getRateHistory('USD', 'EUR')

        expect(result).toEqual([])
    })
})

describe('getLatestRate', () => {
    it('calls GET latest endpoint and returns the rate record', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: mockRate })

        const result = await getLatestRate('USD', 'EUR')

        expect(apiClient.get).toHaveBeenCalledWith('/fin/fx/USD/EUR/rates/latest')
        expect(result).toEqual(mockRate)
    })

    it('URL-encodes currency names', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: mockRate })

        await getLatestRate('US D', 'E/R')

        expect(apiClient.get).toHaveBeenCalledWith('/fin/fx/US%20D/E%2FR/rates/latest')
    })

    it('returns null when the API responds with 404', async () => {
        (apiClient.get as Mock).mockRejectedValue({ response: { status: 404 } })

        const result = await getLatestRate('USD', 'EUR')

        expect(result).toBeNull()
    })

    it('re-throws non-404 errors', async () => {
        const err = { response: { status: 500 } };
        (apiClient.get as Mock).mockRejectedValue(err)

        await expect(getLatestRate('USD', 'EUR')).rejects.toBe(err)
    })
})

describe('createRate', () => {
    it('calls POST with URL-encoded currencies and payload', async () => {
        const payload: CreateRateDTO = { time: '2025-06-01', rate: 0.92 };
        (apiClient.post as Mock).mockResolvedValue({})

        await createRate('USD', 'EUR', payload)

        expect(apiClient.post).toHaveBeenCalledWith('/fin/fx/USD/EUR/rates', payload)
        expect(apiClient.post).toHaveBeenCalledTimes(1)
    })

    it('URL-encodes currency names with special characters', async () => {
        const payload: CreateRateDTO = { time: '2025-06-01', rate: 1.5 };
        (apiClient.post as Mock).mockResolvedValue({})

        await createRate('US D', 'E/R', payload)

        expect(apiClient.post).toHaveBeenCalledWith('/fin/fx/US%20D/E%2FR/rates', payload)
    })

    it('returns void', async () => {
        (apiClient.post as Mock).mockResolvedValue({})

        const result = await createRate('USD', 'EUR', { time: '2025-06-01', rate: 0.92 })

        expect(result).toBeUndefined()
    })
})

describe('createRatesBulk', () => {
    it('calls POST bulk endpoint with points payload', async () => {
        const payload = {
            points: [
                { time: '2025-06-01', rate: 0.92 },
                { time: '2025-06-02', rate: 0.93 },
            ],
        };
        (apiClient.post as Mock).mockResolvedValue({})

        await createRatesBulk('USD', 'EUR', payload)

        expect(apiClient.post).toHaveBeenCalledWith('/fin/fx/USD/EUR/rates/bulk', payload)
        expect(apiClient.post).toHaveBeenCalledTimes(1)
    })

    it('URL-encodes currency names', async () => {
        (apiClient.post as Mock).mockResolvedValue({})

        await createRatesBulk('US D', 'E/R', { points: [] })

        expect(apiClient.post).toHaveBeenCalledWith('/fin/fx/US%20D/E%2FR/rates/bulk', { points: [] })
    })

    it('returns void', async () => {
        (apiClient.post as Mock).mockResolvedValue({})

        const result = await createRatesBulk('USD', 'EUR', { points: [] })

        expect(result).toBeUndefined()
    })
})

describe('updateRate', () => {
    it('calls PUT /fin/fx/rates/:id with payload', async () => {
        const payload = { rate: 0.95 };
        (apiClient.put as Mock).mockResolvedValue({})

        await updateRate(42, payload)

        expect(apiClient.put).toHaveBeenCalledWith('/fin/fx/rates/42', payload)
        expect(apiClient.put).toHaveBeenCalledTimes(1)
    })

    it('sends partial payload with only time', async () => {
        (apiClient.put as Mock).mockResolvedValue({})

        await updateRate(7, { time: '2025-07-01' })

        expect(apiClient.put).toHaveBeenCalledWith('/fin/fx/rates/7', { time: '2025-07-01' })
    })

    it('returns void', async () => {
        (apiClient.put as Mock).mockResolvedValue({})

        const result = await updateRate(1, { rate: 1.0 })

        expect(result).toBeUndefined()
    })
})

describe('deleteRate', () => {
    it('calls DELETE /fin/fx/rates/:id', async () => {
        (apiClient.delete as Mock).mockResolvedValue({})

        await deleteRate(42)

        expect(apiClient.delete).toHaveBeenCalledWith('/fin/fx/rates/42')
        expect(apiClient.delete).toHaveBeenCalledTimes(1)
    })

    it('returns void', async () => {
        (apiClient.delete as Mock).mockResolvedValue({})

        const result = await deleteRate(1)

        expect(result).toBeUndefined()
    })
})
