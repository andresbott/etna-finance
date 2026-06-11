import { describe, it, expect } from 'vitest'
import { computeTtmPe } from './ttmPe'
import type { EpsRecord } from '@/lib/api/MarketData'

const eps = (time: string, basic: number): EpsRecord => ({ time, eps_basic: basic, eps_diluted: basic })

describe('computeTtmPe', () => {
    it('returns length-matched nulls when there are no EPS filings', () => {
        expect(computeTtmPe(['2025-01-01', '2025-01-02'], [10, 11], [])).toEqual([null, null])
    })

    it('returns null for dates with fewer than 4 prior filings', () => {
        const filings = [eps('2024-02-01', 1), eps('2024-05-01', 1), eps('2024-08-01', 1)]
        const result = computeTtmPe(['2024-09-01'], [40], filings)
        expect(result).toEqual([null])
    })

    it('computes close / sum(last 4 basic EPS) once 4 filings are available', () => {
        const filings = [
            eps('2024-02-01', 1.0),
            eps('2024-05-01', 1.0),
            eps('2024-08-01', 1.0),
            eps('2024-11-01', 1.0),
        ]
        // ttm = 4.0, close = 80 => P/E = 20
        const result = computeTtmPe(['2024-12-01'], [80], filings)
        expect(result).toEqual([20])
    })

    it('only counts filings with time <= the date (uses the trailing 4)', () => {
        const filings = [
            eps('2024-02-01', 0.5),
            eps('2024-05-01', 0.5),
            eps('2024-08-01', 1.0),
            eps('2024-11-01', 1.0),
            eps('2025-02-01', 5.0), // in the future relative to the first date
        ]
        // For 2024-12-01: last 4 are 0.5+0.5+1.0+1.0 = 3.0; close 30 => 10
        // For 2025-03-01: last 4 are 0.5+1.0+1.0+5.0 = 7.5; close 75 => 10
        const result = computeTtmPe(['2024-12-01', '2025-03-01'], [30, 75], filings)
        expect(result).toEqual([10, 10])
    })

    it('returns null when ttm EPS is negative', () => {
        const filings = [eps('2024-02-01', -1), eps('2024-05-01', -1), eps('2024-08-01', 0.5), eps('2024-11-01', 0.5)]
        // ttm = -1 => null
        expect(computeTtmPe(['2024-12-01'], [40], filings)).toEqual([null])
    })

    it('returns null when ttm EPS is exactly zero', () => {
        const filings = [eps('2024-02-01', -1), eps('2024-05-01', -1), eps('2024-08-01', 1), eps('2024-11-01', 1)]
        // ttm = 0 => null (guard is ttmEps <= 0, avoids divide-by-zero)
        expect(computeTtmPe(['2024-12-01'], [40], filings)).toEqual([null])
    })

    it('returns null when close is missing or non-positive', () => {
        const filings = [eps('2024-02-01', 1), eps('2024-05-01', 1), eps('2024-08-01', 1), eps('2024-11-01', 1)]
        expect(computeTtmPe(['2024-12-01', '2024-12-02'], [0, -5], filings)).toEqual([null, null])
    })

    it('returns an empty array when there are no dates', () => {
        const filings = [eps('2024-02-01', 1), eps('2024-05-01', 1), eps('2024-08-01', 1), eps('2024-11-01', 1)]
        expect(computeTtmPe([], [], filings)).toEqual([])
    })

    it('returns null when close is NaN', () => {
        const filings = [eps('2024-02-01', 1), eps('2024-05-01', 1), eps('2024-08-01', 1), eps('2024-11-01', 1)]
        expect(computeTtmPe(['2024-12-01'], [NaN], filings)).toEqual([null])
    })
})
