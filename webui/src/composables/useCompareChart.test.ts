import { describe, it, expect } from 'vitest'
import { buildCompareChartOption, type CompareChartSeries } from './useCompareChart'

function ramp(n: number, start = 100): number[] {
    return Array.from({ length: n }, (_, i) => start + i)
}

function makeSeries(symbol: string, dates: string[], closes: number[], currency = 'USD'): CompareChartSeries {
    return { symbol, currency, dates, closes }
}

const datesA = ['2026-01-01', '2026-01-02', '2026-01-03', '2026-01-04', '2026-01-05']

describe('buildCompareChartOption', () => {
    it('creates one line series per instrument', () => {
        const input = [
            makeSeries('AAA', datesA, ramp(5)),
            makeSeries('BBB', datesA, ramp(5, 200))
        ]
        const opt: any = buildCompareChartOption(input, 'price', 50, '')
        expect(opt.series).toHaveLength(2)
        expect(opt.series.map((s: any) => s.name)).toEqual(['AAA', 'BBB'])
        expect(opt.series.every((s: any) => s.type === 'line')).toBe(true)
    })

    it('price view uses a scaled value axis without fixed bounds', () => {
        const opt: any = buildCompareChartOption([makeSeries('AAA', datesA, ramp(5))], 'price', 50, '')
        expect(opt.yAxis.scale).toBe(true)
        expect(opt.yAxis.min).toBeUndefined()
        expect(opt.yAxis.max).toBeUndefined()
    })

    it('rsi view fixes the y axis to 0..100 and adds 30/70 mark lines', () => {
        const opt: any = buildCompareChartOption([makeSeries('AAA', ramp(40).map(String), ramp(40))], 'rsi', 14, '')
        expect(opt.yAxis.min).toBe(0)
        expect(opt.yAxis.max).toBe(100)
        const markYs = opt.series[0].markLine.data.map((d: any) => d.yAxis)
        expect(markYs).toEqual([30, 70])
    })

    it('applies the period to SMA (first period-1 points are null)', () => {
        const period = 3
        const opt: any = buildCompareChartOption([makeSeries('AAA', datesA, ramp(5))], 'sma', period, '')
        const data = opt.series[0].data
        expect(data.slice(0, period - 1)).toEqual([null, null])
        expect(data[period - 1]).not.toBeNull()
    })

    it('builds a unioned, sorted x-axis and pads missing dates with null', () => {
        const a = makeSeries('AAA', ['2026-01-01', '2026-01-03'], [10, 30])
        const b = makeSeries('BBB', ['2026-01-02', '2026-01-03'], [20, 25])
        const opt: any = buildCompareChartOption([a, b], 'price', 50, '')
        expect(opt.xAxis.data).toEqual(['2026-01-01', '2026-01-02', '2026-01-03'])
        expect(opt.series[0].data).toEqual([10, null, 30])
        expect(opt.series[1].data).toEqual([null, 20, 25])
    })

    it('trims the warmup prefix before the visible start date', () => {
        const opt: any = buildCompareChartOption([makeSeries('AAA', datesA, ramp(5))], 'price', 50, '2026-01-03')
        expect(opt.xAxis.data).toEqual(['2026-01-03', '2026-01-04', '2026-01-05'])
        expect(opt.series[0].data).toEqual([102, 103, 104])
    })

    it('keeps raw ISO dates as axis keys but renders labels via the injected formatter', () => {
        const toDDMMYYYY = (iso: string) => {
            const [y, m, d] = iso.split('-')
            return `${d}/${m}/${y}`
        }
        const opt: any = buildCompareChartOption([makeSeries('AAA', datesA, ramp(5))], 'price', 50, '', toDDMMYYYY)
        // Underlying category keys stay raw ISO (series alignment / lookups depend on it)
        expect(opt.xAxis.data).toEqual(datesA)
        // Axis tick labels and the tooltip header are formatted per the setting
        expect(opt.xAxis.axisLabel.formatter('2026-01-01')).toBe('01/01/2026')
        const header = opt.tooltip.formatter([{ axisValue: '2026-01-01', seriesName: 'AAA', value: 100, marker: '' }])
        expect(header).toContain('01/01/2026')
    })
})
