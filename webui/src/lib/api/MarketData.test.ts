import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { apiClient } from './client'
import {
    getMarketDataSymbols,
    getPriceHistory,
    getLatestPrice,
    createPrice,
    createPricesBulk,
    updatePrice,
    deletePrice,
    type PriceRecord,
    type CreatePriceDTO,
    type UpdatePriceDTO,
} from './MarketData'

vi.mock('./client', () => ({
    apiClient: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}))

beforeEach(() => vi.clearAllMocks())

const BASE = '/fin/marketdata'

const mockPrice: PriceRecord = {
    id: 1,
    symbol: 'AAPL',
    time: '2025-01-15T10:00:00Z',
    price: 198.5,
}

/* ------------------------------------------------------------------ */
/*  getMarketDataSymbols                                              */
/* ------------------------------------------------------------------ */
describe('getMarketDataSymbols', () => {
    it('calls GET /fin/marketdata/symbols and returns symbols', async () => {
        const symbols = ['AAPL', 'MSFT', 'GOOG'];
        (apiClient.get as Mock).mockResolvedValue({ data: { symbols } })

        const result = await getMarketDataSymbols()

        expect(apiClient.get).toHaveBeenCalledWith(`${BASE}/symbols`)
        expect(apiClient.get).toHaveBeenCalledTimes(1)
        expect(result).toEqual(symbols)
    })

    it('returns empty array when symbols is undefined', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: {} })

        const result = await getMarketDataSymbols()

        expect(result).toEqual([])
    })

    it('returns empty array when symbols is null', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { symbols: null } })

        const result = await getMarketDataSymbols()

        expect(result).toEqual([])
    })
})

/* ------------------------------------------------------------------ */
/*  getPriceHistory                                                    */
/* ------------------------------------------------------------------ */
describe('getPriceHistory', () => {
    it('calls GET with encoded symbol and no query params when dates omitted', async () => {
        const items = [mockPrice];
        (apiClient.get as Mock).mockResolvedValue({ data: { items } })

        const result = await getPriceHistory('AAPL')

        expect(apiClient.get).toHaveBeenCalledWith(`${BASE}/AAPL/prices`)
        expect(result).toEqual(items)
    })

    it('appends start query param only', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { items: [] } })

        await getPriceHistory('AAPL', '2025-01-01')

        expect(apiClient.get).toHaveBeenCalledWith(`${BASE}/AAPL/prices?start=2025-01-01`)
    })

    it('appends end query param only', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { items: [] } })

        await getPriceHistory('AAPL', undefined, '2025-12-31')

        expect(apiClient.get).toHaveBeenCalledWith(`${BASE}/AAPL/prices?end=2025-12-31`)
    })

    it('appends both start and end query params', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { items: [] } })

        await getPriceHistory('AAPL', '2025-01-01', '2025-12-31')

        expect(apiClient.get).toHaveBeenCalledWith(
            `${BASE}/AAPL/prices?start=2025-01-01&end=2025-12-31`
        )
    })

    it('URL-encodes symbols with special characters', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { items: [] } })

        await getPriceHistory('BTC/USD')

        expect(apiClient.get).toHaveBeenCalledWith(`${BASE}/BTC%2FUSD/prices`)
    })

    it('URL-encodes symbols with spaces', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { items: [] } })

        await getPriceHistory('S&P 500')

        expect(apiClient.get).toHaveBeenCalledWith(`${BASE}/S%26P%20500/prices`)
    })

    it('returns empty array when items is undefined', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: {} })

        const result = await getPriceHistory('AAPL')

        expect(result).toEqual([])
    })

    it('returns empty array when items is null', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { items: null } })

        const result = await getPriceHistory('AAPL')

        expect(result).toEqual([])
    })
})

/* ------------------------------------------------------------------ */
/*  getLatestPrice                                                     */
/* ------------------------------------------------------------------ */
describe('getLatestPrice', () => {
    it('calls GET with encoded symbol and returns the price record', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: mockPrice })

        const result = await getLatestPrice('AAPL')

        expect(apiClient.get).toHaveBeenCalledWith(`${BASE}/AAPL/prices/latest`)
        expect(result).toEqual(mockPrice)
    })

    it('URL-encodes symbols with special characters', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: mockPrice })

        await getLatestPrice('BTC/USD')

        expect(apiClient.get).toHaveBeenCalledWith(`${BASE}/BTC%2FUSD/prices/latest`)
    })

    it('returns null when the API responds with 404', async () => {
        const error = { response: { status: 404 } };
        (apiClient.get as Mock).mockRejectedValue(error)

        const result = await getLatestPrice('UNKNOWN')

        expect(result).toBeNull()
    })

    it('re-throws non-404 errors', async () => {
        const error = { response: { status: 500 } };
        (apiClient.get as Mock).mockRejectedValue(error)

        await expect(getLatestPrice('AAPL')).rejects.toEqual(error)
    })

    it('re-throws errors without a response property', async () => {
        const error = new Error('Network error');
        (apiClient.get as Mock).mockRejectedValue(error)

        await expect(getLatestPrice('AAPL')).rejects.toThrow('Network error')
    })
})

/* ------------------------------------------------------------------ */
/*  createPrice                                                        */
/* ------------------------------------------------------------------ */
describe('createPrice', () => {
    it('calls POST with encoded symbol and payload', async () => {
        const payload: CreatePriceDTO = { time: '2025-01-15T10:00:00Z', price: 198.5 };
        (apiClient.post as Mock).mockResolvedValue({})

        await createPrice('AAPL', payload)

        expect(apiClient.post).toHaveBeenCalledWith(`${BASE}/AAPL/prices`, payload)
        expect(apiClient.post).toHaveBeenCalledTimes(1)
    })

    it('URL-encodes symbols with special characters', async () => {
        const payload: CreatePriceDTO = { time: '2025-01-15T10:00:00Z', price: 42000 };
        (apiClient.post as Mock).mockResolvedValue({})

        await createPrice('BTC/USD', payload)

        expect(apiClient.post).toHaveBeenCalledWith(`${BASE}/BTC%2FUSD/prices`, payload)
    })

    it('returns void', async () => {
        (apiClient.post as Mock).mockResolvedValue({})

        const result = await createPrice('AAPL', { time: '2025-01-15T10:00:00Z', price: 198.5 })

        expect(result).toBeUndefined()
    })
})

/* ------------------------------------------------------------------ */
/*  createPricesBulk                                                   */
/* ------------------------------------------------------------------ */
describe('createPricesBulk', () => {
    it('calls POST with encoded symbol and bulk payload', async () => {
        const payload = {
            points: [
                { time: '2025-01-15T10:00:00Z', price: 198.5 },
                { time: '2025-01-16T10:00:00Z', price: 200.0 },
            ],
        };
        (apiClient.post as Mock).mockResolvedValue({})

        await createPricesBulk('AAPL', payload)

        expect(apiClient.post).toHaveBeenCalledWith(`${BASE}/AAPL/prices/bulk`, payload)
        expect(apiClient.post).toHaveBeenCalledTimes(1)
    })

    it('URL-encodes symbols with special characters', async () => {
        const payload = { points: [{ time: '2025-01-15T10:00:00Z', price: 42000 }] };
        (apiClient.post as Mock).mockResolvedValue({})

        await createPricesBulk('BTC/USD', payload)

        expect(apiClient.post).toHaveBeenCalledWith(`${BASE}/BTC%2FUSD/prices/bulk`, payload)
    })

    it('returns void', async () => {
        (apiClient.post as Mock).mockResolvedValue({})

        const result = await createPricesBulk('AAPL', { points: [] })

        expect(result).toBeUndefined()
    })
})

/* ------------------------------------------------------------------ */
/*  updatePrice                                                        */
/* ------------------------------------------------------------------ */
describe('updatePrice', () => {
    it('calls PUT /fin/marketdata/prices/:id with payload', async () => {
        const payload: UpdatePriceDTO = { price: 205.0 };
        (apiClient.put as Mock).mockResolvedValue({})

        await updatePrice(1, payload)

        expect(apiClient.put).toHaveBeenCalledWith(`${BASE}/prices/1`, payload)
        expect(apiClient.put).toHaveBeenCalledTimes(1)
    })

    it('sends partial payload with only time', async () => {
        const payload: UpdatePriceDTO = { time: '2025-02-01T00:00:00Z' };
        (apiClient.put as Mock).mockResolvedValue({})

        await updatePrice(42, payload)

        expect(apiClient.put).toHaveBeenCalledWith(`${BASE}/prices/42`, payload)
    })

    it('returns void', async () => {
        (apiClient.put as Mock).mockResolvedValue({})

        const result = await updatePrice(1, { price: 205.0 })

        expect(result).toBeUndefined()
    })
})

/* ------------------------------------------------------------------ */
/*  deletePrice                                                        */
/* ------------------------------------------------------------------ */
describe('deletePrice', () => {
    it('calls DELETE /fin/marketdata/prices/:id', async () => {
        (apiClient.delete as Mock).mockResolvedValue({})

        await deletePrice(1)

        expect(apiClient.delete).toHaveBeenCalledWith(`${BASE}/prices/1`)
        expect(apiClient.delete).toHaveBeenCalledTimes(1)
    })

    it('returns void', async () => {
        (apiClient.delete as Mock).mockResolvedValue({})

        const result = await deletePrice(99)

        expect(result).toBeUndefined()
    })
})
