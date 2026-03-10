import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { apiClient } from './client'
import {
    getPositions,
    getPositionDetail,
    getLots,
    getTrades,
    type Position,
    type Lot,
    type Trade,
    type PositionDetail,
} from './Portfolio'

vi.mock('./client', () => ({
    apiClient: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}))

beforeEach(() => vi.clearAllMocks())

const mockPosition: Position = {
    id: 1,
    accountId: 10,
    instrumentId: 100,
    quantity: 50,
    costBasis: 5000,
    avgCost: 100,
}

const mockLot: Lot = {
    id: 1,
    tradeId: 5,
    accountId: 10,
    instrumentId: 100,
    openDate: '2025-01-15',
    quantity: 25,
    originalQty: 25,
    costPerShare: 100,
    costBasis: 2500,
    status: 1,
}

const mockTrade: Trade = {
    id: 1,
    transactionId: 20,
    accountId: 10,
    instrumentId: 100,
    tradeType: 1,
    quantity: 25,
    pricePerShare: 100,
    totalAmount: 2500,
    currency: 'USD',
    date: '2025-01-15',
}

describe('getPositions', () => {
    it('calls GET /fin/portfolio/positions without params and returns items', async () => {
        const items = [mockPosition];
        (apiClient.get as Mock).mockResolvedValue({ data: { items } })

        const result = await getPositions()

        expect(apiClient.get).toHaveBeenCalledWith('/fin/portfolio/positions?')
        expect(apiClient.get).toHaveBeenCalledTimes(1)
        expect(result).toEqual(items)
    })

    it('passes accountId as query param when provided', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { items: [mockPosition] } })

        await getPositions(10)

        expect(apiClient.get).toHaveBeenCalledWith('/fin/portfolio/positions?accountId=10')
    })

    it('returns empty array when items is undefined', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: {} })

        const result = await getPositions()

        expect(result).toEqual([])
    })

    it('returns empty array when items is null', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { items: null } })

        const result = await getPositions()

        expect(result).toEqual([])
    })
})

describe('getPositionDetail', () => {
    it('calls GET /fin/portfolio/positions/:accountId/:instrumentId and returns data', async () => {
        const detail: PositionDetail = { position: mockPosition, lots: [mockLot] };
        (apiClient.get as Mock).mockResolvedValue({ data: detail })

        const result = await getPositionDetail(10, 100)

        expect(apiClient.get).toHaveBeenCalledWith('/fin/portfolio/positions/10/100')
        expect(apiClient.get).toHaveBeenCalledTimes(1)
        expect(result).toEqual(detail)
    })

    it('returns detail with empty lots array', async () => {
        const detail: PositionDetail = { position: mockPosition, lots: [] };
        (apiClient.get as Mock).mockResolvedValue({ data: detail })

        const result = await getPositionDetail(10, 100)

        expect(result.lots).toEqual([])
    })
})

describe('getLots', () => {
    it('calls GET /fin/portfolio/lots without params and returns items', async () => {
        const items = [mockLot];
        (apiClient.get as Mock).mockResolvedValue({ data: { items } })

        const result = await getLots()

        expect(apiClient.get).toHaveBeenCalledWith('/fin/portfolio/lots?')
        expect(apiClient.get).toHaveBeenCalledTimes(1)
        expect(result).toEqual(items)
    })

    it('passes accountId as query param when provided', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { items: [] } })

        await getLots(10)

        expect(apiClient.get).toHaveBeenCalledWith('/fin/portfolio/lots?accountId=10')
    })

    it('passes both accountId and instrumentId as query params', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { items: [] } })

        await getLots(10, 100)

        expect(apiClient.get).toHaveBeenCalledWith('/fin/portfolio/lots?accountId=10&instrumentId=100')
    })

    it('passes only instrumentId when accountId is undefined', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { items: [] } })

        await getLots(undefined, 100)

        expect(apiClient.get).toHaveBeenCalledWith('/fin/portfolio/lots?instrumentId=100')
    })

    it('returns empty array when items is undefined', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: {} })

        const result = await getLots()

        expect(result).toEqual([])
    })

    it('returns empty array when items is null', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { items: null } })

        const result = await getLots()

        expect(result).toEqual([])
    })
})

describe('getTrades', () => {
    it('calls GET /fin/portfolio/trades without params and returns items', async () => {
        const items = [mockTrade];
        (apiClient.get as Mock).mockResolvedValue({ data: { items } })

        const result = await getTrades()

        expect(apiClient.get).toHaveBeenCalledWith('/fin/portfolio/trades?')
        expect(apiClient.get).toHaveBeenCalledTimes(1)
        expect(result).toEqual(items)
    })

    it('passes accountId as query param when provided', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { items: [] } })

        await getTrades(10)

        expect(apiClient.get).toHaveBeenCalledWith('/fin/portfolio/trades?accountId=10')
    })

    it('passes both accountId and instrumentId as query params', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { items: [] } })

        await getTrades(10, 100)

        expect(apiClient.get).toHaveBeenCalledWith('/fin/portfolio/trades?accountId=10&instrumentId=100')
    })

    it('passes only instrumentId when accountId is undefined', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { items: [] } })

        await getTrades(undefined, 100)

        expect(apiClient.get).toHaveBeenCalledWith('/fin/portfolio/trades?instrumentId=100')
    })

    it('returns empty array when items is undefined', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: {} })

        const result = await getTrades()

        expect(result).toEqual([])
    })

    it('returns empty array when items is null', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { items: null } })

        const result = await getTrades()

        expect(result).toEqual([])
    })
})
