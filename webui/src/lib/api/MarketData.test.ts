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
} from './MarketData'

vi.mock('./client', () => ({
    apiClient: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}))

beforeEach(() => vi.clearAllMocks())

const BASE = '/fin/marketdata'

const mockPrice: PriceRecord = {
    symbol: 'AAPL',
    time: '2025-01-15T10:00:00Z',
    open: 195.0,
    high: 200.0,
    low: 194.0,
    close: 198.5,
    volume: 1234567,
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
        const payload: CreatePriceDTO = {
            time: '2025-01-15T10:00:00Z',
            open: 195.0, high: 200.0, low: 194.0, close: 198.5, volume: 1234567
        };
        (apiClient.post as Mock).mockResolvedValue({})

        await createPrice('AAPL', payload)

        expect(apiClient.post).toHaveBeenCalledWith(`${BASE}/AAPL/prices`, payload)
        expect(apiClient.post).toHaveBeenCalledTimes(1)
    })

    it('URL-encodes symbols with special characters', async () => {
        const payload: CreatePriceDTO = {
            time: '2025-01-15T10:00:00Z',
            open: 40000, high: 42500, low: 39500, close: 42000, volume: 500
        };
        (apiClient.post as Mock).mockResolvedValue({})

        await createPrice('BTC/USD', payload)

        expect(apiClient.post).toHaveBeenCalledWith(`${BASE}/BTC%2FUSD/prices`, payload)
    })

    it('returns void', async () => {
        (apiClient.post as Mock).mockResolvedValue({})

        const result = await createPrice('AAPL', {
            time: '2025-01-15T10:00:00Z',
            open: 195.0, high: 200.0, low: 194.0, close: 198.5, volume: 1234567
        })

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
                { time: '2025-01-15T10:00:00Z', open: 195.0, high: 200.0, low: 194.0, close: 198.5, volume: 1000 },
                { time: '2025-01-16T10:00:00Z', open: 198.0, high: 202.0, low: 197.0, close: 200.0, volume: 1200 },
            ],
        };
        (apiClient.post as Mock).mockResolvedValue({})

        await createPricesBulk('AAPL', payload)

        expect(apiClient.post).toHaveBeenCalledWith(`${BASE}/AAPL/prices/bulk`, payload)
        expect(apiClient.post).toHaveBeenCalledTimes(1)
    })

    it('URL-encodes symbols with special characters', async () => {
        const payload = { points: [{ time: '2025-01-15T10:00:00Z', open: 40000, high: 42500, low: 39500, close: 42000, volume: 500 }] };
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
    it('calls PUT /fin/marketdata/{symbol}/prices/{date} with payload', async () => {
        const payload: CreatePriceDTO = {
            time: '2025-01-15', open: 195.0, high: 200.0, low: 194.0, close: 205.0, volume: 1000
        };
        (apiClient.put as Mock).mockResolvedValue({})

        await updatePrice('AAPL', '2025-01-15', payload)

        expect(apiClient.put).toHaveBeenCalledWith(`${BASE}/AAPL/prices/2025-01-15`, payload)
        expect(apiClient.put).toHaveBeenCalledTimes(1)
    })

    it('URL-encodes symbol with special characters', async () => {
        const payload: CreatePriceDTO = {
            time: '2025-02-01', open: 40000, high: 42500, low: 39500, close: 41000, volume: 500
        };
        (apiClient.put as Mock).mockResolvedValue({})

        await updatePrice('BTC/USD', '2025-02-01', payload)

        expect(apiClient.put).toHaveBeenCalledWith(`${BASE}/BTC%2FUSD/prices/2025-02-01`, payload)
    })

    it('returns void', async () => {
        (apiClient.put as Mock).mockResolvedValue({})

        const result = await updatePrice('AAPL', '2025-01-15', {
            time: '2025-01-15', open: 195.0, high: 200.0, low: 194.0, close: 205.0, volume: 1000
        })

        expect(result).toBeUndefined()
    })
})

/* ------------------------------------------------------------------ */
/*  deletePrice                                                        */
/* ------------------------------------------------------------------ */
describe('deletePrice', () => {
    it('calls DELETE /fin/marketdata/{symbol}/prices/{date}', async () => {
        (apiClient.delete as Mock).mockResolvedValue({})

        await deletePrice('AAPL', '2025-01-15')

        expect(apiClient.delete).toHaveBeenCalledWith(`${BASE}/AAPL/prices/2025-01-15`)
        expect(apiClient.delete).toHaveBeenCalledTimes(1)
    })

    it('URL-encodes symbol with special characters', async () => {
        (apiClient.delete as Mock).mockResolvedValue({})

        await deletePrice('BTC/USD', '2025-01-15')

        expect(apiClient.delete).toHaveBeenCalledWith(`${BASE}/BTC%2FUSD/prices/2025-01-15`)
    })

    it('returns void', async () => {
        (apiClient.delete as Mock).mockResolvedValue({})

        const result = await deletePrice('AAPL', '2025-01-15')

        expect(result).toBeUndefined()
    })
})
