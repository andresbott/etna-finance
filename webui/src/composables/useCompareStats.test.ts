import { describe, it, expect } from 'vitest'
import { computeCompareStats } from './useCompareStats'
import type { CompareSeries } from './useCompareSeries'

function makeSeries(
    symbol: string,
    dates: string[],
    opens: number[],
    highs: number[],
    lows: number[],
    closes: number[],
    volumes: number[]
): CompareSeries {
    return {
        symbol,
        ohlcv: { dates, opens, highs, lows, closes, volumes, points: [] }
    }
}

const dates5 = ['2026-01-01', '2026-01-02', '2026-01-03', '2026-01-04', '2026-01-05']

describe('computeCompareStats', () => {
    it('computes open/close/change/high/low for a simple rising series', () => {
        const series = [
            makeSeries(
                'AAA',
                dates5,
                [100, 102, 104, 106, 108],
                [101, 103, 105, 107, 112],
                [99, 100, 103, 105, 107],
                [102, 104, 106, 108, 110],
                [1000, 2000, 3000, 4000, 5000]
            )
        ]
        const rows = computeCompareStats(series, '', { AAA: 'USD' })
        expect(rows).toHaveLength(1)
        const r = rows[0]
        expect(r.symbol).toBe('AAA')
        expect(r.currency).toBe('USD')
        expect(r.open).toBe(100)
        expect(r.close).toBe(110)
        expect(r.high).toBe(112)
        expect(r.low).toBe(99)
        expect(r.change).toBeCloseTo(10, 6)
        expect(r.changePct).toBeCloseTo(10, 6)
        expect(r.avgVolume).toBeCloseTo(3000, 6)
    })

    it('computes max drawdown from peak to trough on closes', () => {
        const dates = ['2026-01-01', '2026-01-02', '2026-01-03', '2026-01-04']
        const series = [
            makeSeries(
                'DD',
                dates,
                [100, 100, 100, 100],
                [100, 120, 120, 110],
                [100, 100, 90, 100],
                [100, 120, 90, 110],
                [1, 1, 1, 1]
            )
        ]
        const r = computeCompareStats(series, '', {})[0]
        expect(r.maxDrawdownPct).toBeCloseTo(-25, 6)
    })

    it('computes volatility as population stdev of daily returns * 100', () => {
        const dates = ['2026-01-01', '2026-01-02', '2026-01-03']
        const flat = [
            makeSeries('V', dates, [100, 110, 121], [100, 110, 121], [100, 110, 121], [100, 110, 121], [1, 1, 1])
        ]
        expect(computeCompareStats(flat, '', {})[0].volatilityPct).toBeCloseTo(0, 6)

        const swing = [
            makeSeries('V', dates, [100, 110, 99], [100, 110, 99], [100, 110, 99], [100, 110, 99], [1, 1, 1])
        ]
        expect(computeCompareStats(swing, '', {})[0].volatilityPct).toBeCloseTo(10, 6)
    })

    it('computes stats only over the visible range, trimming the warmup prefix', () => {
        const dates = ['2026-01-01', '2026-01-02', '2026-01-03', '2026-01-04']
        const series = [
            makeSeries(
                'W',
                dates,
                [50, 60, 100, 106],
                [55, 65, 105, 112],
                [49, 59, 99, 105],
                [52, 62, 104, 110],
                [1, 1, 1, 1]
            )
        ]
        const r = computeCompareStats(series, '2026-01-03', {})[0]
        expect(r.open).toBe(100)
        expect(r.close).toBe(110)
        expect(r.high).toBe(112)
        expect(r.low).toBe(99)
    })

    it('guards single-point visible range: drawdown and volatility are 0', () => {
        const series = [
            makeSeries('S', ['2026-01-01'], [100], [105], [95], [102], [500])
        ]
        const r = computeCompareStats(series, '', {})[0]
        expect(r.open).toBe(100)
        expect(r.close).toBe(102)
        expect(r.maxDrawdownPct).toBe(0)
        expect(r.volatilityPct).toBe(0)
        expect(r.avgVolume).toBe(500)
    })

    it('omits symbols with no data in the visible range', () => {
        const series = [
            makeSeries('GONE', ['2026-01-01'], [100], [100], [100], [100], [1]),
            makeSeries('HERE', ['2026-02-01'], [200], [200], [200], [200], [2])
        ]
        const rows = computeCompareStats(series, '2026-02-01', {})
        expect(rows.map((r) => r.symbol)).toEqual(['HERE'])
    })
})
