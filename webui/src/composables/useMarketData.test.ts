import { describe, it, expect } from 'vitest'
import {
    filterMarketInstruments,
    type MarketInstrument,
    type MarketInstrumentFilters
} from './useMarketData'

function makeInstrument(overrides: Partial<MarketInstrument>): MarketInstrument {
    return {
        id: 1,
        symbol: 'AAPL',
        name: 'Apple Inc.',
        notes: '',
        currency: 'USD',
        type: 'Stock',
        exchange: 'NASDAQ',
        lastPrice: 100,
        change: 0,
        changePct: 0,
        volume: 0,
        peRatio: null,
        dividendYield: 0,
        week52High: 0,
        week52Low: 0,
        lastUpdate: '2026-06-10',
        ...overrides
    }
}

const noFilters: MarketInstrumentFilters = { search: '', types: [], exchanges: [] }

describe('filterMarketInstruments', () => {
    const apple = makeInstrument({ id: 1, symbol: 'AAPL', name: 'Apple Inc.', type: 'Stock', exchange: 'NASDAQ' })
    const vwrl = makeInstrument({ id: 2, symbol: 'VWRL', name: 'Vanguard FTSE All-World', type: 'ETF', exchange: 'LSE' })
    const reit = makeInstrument({ id: 3, symbol: 'O', name: 'Realty Income', type: 'REIT', exchange: 'NYSE' })
    const all = [apple, vwrl, reit]

    it('returns all instruments when no filters are active', () => {
        expect(filterMarketInstruments(all, noFilters)).toEqual(all)
    })

    it('matches search on symbol case-insensitively', () => {
        expect(filterMarketInstruments(all, { ...noFilters, search: 'aapl' })).toEqual([apple])
    })

    it('matches search on name case-insensitively', () => {
        expect(filterMarketInstruments(all, { ...noFilters, search: 'vanguard' })).toEqual([vwrl])
    })

    it('returns empty array when search matches nothing', () => {
        expect(filterMarketInstruments(all, { ...noFilters, search: 'tesla' })).toEqual([])
    })

    it('filters by type (OR within types)', () => {
        expect(filterMarketInstruments(all, { ...noFilters, types: ['ETF', 'REIT'] })).toEqual([vwrl, reit])
    })

    it('filters by exchange (OR within exchanges)', () => {
        expect(filterMarketInstruments(all, { ...noFilters, exchanges: ['NASDAQ'] })).toEqual([apple])
    })

    it('ANDs across search, types, and exchanges', () => {
        const result = filterMarketInstruments(all, { search: 'realty', types: ['REIT'], exchanges: ['NYSE'] })
        expect(result).toEqual([reit])
    })

    it('ANDs to empty when groups disagree', () => {
        const result = filterMarketInstruments(all, { ...noFilters, types: ['ETF'], exchanges: ['NASDAQ'] })
        expect(result).toEqual([])
    })

    it('treats a whitespace-only search as no search filter', () => {
        expect(filterMarketInstruments(all, { ...noFilters, search: '   ' })).toEqual(all)
    })

    it('returns an empty array when given no instruments', () => {
        expect(filterMarketInstruments([], { ...noFilters, search: 'aapl', types: ['ETF'] })).toEqual([])
    })
})
