import { describe, it, expect } from 'vitest'
import { computePortfolioProjection, computePortfolioExpectedReturn, computePortfolioNetWorth20Y } from './portfolio'
import type { PortfolioSimulatorParams } from '@/lib/api/ToolsData'

const BASE_PARAMS: PortfolioSimulatorParams = {
    initialContribution: 10000,
    growthRatePct: 7,
    expenseRatioPct: 0.2,
    capitalGainTaxPct: 19,
    taxModel: 'exit',
}

describe('computePortfolioProjection — exit tax', () => {
    const result = computePortfolioProjection(BASE_PARAMS)

    it('returns 21 data points (years 0–20)', () => {
        expect(result.years).toHaveLength(21)
        expect(result.years[0]).toBe(0)
        expect(result.years[20]).toBe(20)
    })

    it('starts at initial contribution', () => {
        expect(result.series.netWorth[0]).toBe(10000)
        expect(result.series.totalInvested[0]).toBe(10000)
    })

    it('total invested is flat (no monthly contributions)', () => {
        const allSame = result.series.totalInvested.every(v => v === 10000)
        expect(allSame).toBe(true)
    })

    it('net worth grows over time (before exit tax applied at year 20)', () => {
        expect(result.series.netWorth[10]).toBeGreaterThan(10000)
        expect(result.series.netWorth[20]).toBeGreaterThan(10000)
    })

    it('tax impact is 0 for years 0-19, positive at year 20', () => {
        for (let i = 0; i <= 19; i++) {
            expect(result.series.taxImpact[i]).toBe(0)
        }
        expect(result.series.taxImpact[20]).toBeGreaterThan(0)
    })

    it('exit tax is applied only to gains, not total balance', () => {
        const preTaxBalance = result.finalValueAfterTax + result.taxPaid
        const gains = preTaxBalance - 10000
        expect(result.taxPaid).toBeCloseTo(gains * 0.19, 2)
    })

    it('uses effective return (growth minus expense ratio)', () => {
        // 7% growth - 0.2% TER = 6.8% effective
        const preTaxBalance = result.finalValueAfterTax + result.taxPaid
        expect(preTaxBalance).toBeCloseTo(10000 * Math.pow(1.068, 20), 0)
    })

    it('finalValueBeforeTax equals balance before exit tax', () => {
        expect(result.finalValueBeforeTax).toBeCloseTo(
            result.finalValueAfterTax + result.taxPaid, 2
        )
    })
})

describe('computePortfolioProjection — annual tax', () => {
    const params: PortfolioSimulatorParams = { ...BASE_PARAMS, taxModel: 'annual' }
    const result = computePortfolioProjection(params)

    it('returns 21 data points', () => {
        expect(result.years).toHaveLength(21)
    })

    it('tax impact grows each year', () => {
        expect(result.series.taxImpact[1]).toBeGreaterThan(0)
        expect(result.series.taxImpact[10]).toBeGreaterThan(result.series.taxImpact[5])
        expect(result.series.taxImpact[20]).toBeGreaterThan(result.series.taxImpact[10])
    })

    it('annual tax net worth is lower than exit tax net worth (tax drag)', () => {
        const exitResult = computePortfolioProjection(BASE_PARAMS)
        expect(result.finalValueAfterTax).toBeLessThan(exitResult.finalValueAfterTax)
    })

    it('taxes only year-over-year gains, not full balance', () => {
        const effectiveReturn = 7 - 0.2
        const balanceAfterYear1 = 10000 * Math.pow(1 + effectiveReturn / 100, 1)
        const year1Gain = balanceAfterYear1 - 10000
        const year1Tax = year1Gain * 0.19
        expect(result.series.taxImpact[1]).toBeCloseTo(year1Tax, 0)
    })

    it('finalValueBeforeTax equals net worth plus cumulative tax', () => {
        expect(result.finalValueBeforeTax).toBeCloseTo(
            result.finalValueAfterTax + result.taxPaid, 2
        )
    })
})

describe('computePortfolioExpectedReturn', () => {
    it('computes exit tax expected return', () => {
        const result = computePortfolioExpectedReturn(BASE_PARAMS)
        // effectiveReturn = 7 - 0.2 = 6.8
        // estimatedTaxDrag = (19/100) * 6.8 / 20 = 0.0646
        // expected = 6.8 - 0.0646 = 6.7354
        expect(result).toBeCloseTo(6.7354, 2)
    })

    it('computes annual tax expected return', () => {
        const params = { ...BASE_PARAMS, taxModel: 'annual' as const }
        const result = computePortfolioExpectedReturn(params)
        // estimatedTaxDrag = (19/100) * 6.8 = 1.292
        // expected = 6.8 - 1.292 = 5.508
        expect(result).toBeCloseTo(5.508, 2)
    })
})

describe('computePortfolioNetWorth20Y', () => {
    it('returns 21 net worth values', () => {
        const values = computePortfolioNetWorth20Y(BASE_PARAMS)
        expect(values).toHaveLength(21)
        expect(values[0]).toBe(BASE_PARAMS.initialContribution)
        expect(values[20]).toBeGreaterThan(0)
    })
})

describe('edge cases', () => {
    it('handles zero growth', () => {
        const params = { ...BASE_PARAMS, growthRatePct: 0, expenseRatioPct: 0 }
        const result = computePortfolioProjection(params)
        expect(result.finalValueAfterTax).toBe(10000)
        expect(result.taxPaid).toBe(0)
    })

    it('handles negative effective return (expense > growth)', () => {
        const params = { ...BASE_PARAMS, growthRatePct: 0.1, expenseRatioPct: 0.5 }
        const result = computePortfolioProjection(params)
        expect(result.finalValueAfterTax).toBeLessThan(10000)
        expect(result.taxPaid).toBe(0)
    })

    it('handles zero initial contribution', () => {
        const params = { ...BASE_PARAMS, initialContribution: 0 }
        const result = computePortfolioProjection(params)
        expect(result.finalValueAfterTax).toBe(0)
    })

    it('defaults missing expenseRatioPct to 0', () => {
        const params = { ...BASE_PARAMS }
        delete (params as any).expenseRatioPct
        const result = computePortfolioProjection(params)
        const withTER = computePortfolioProjection(BASE_PARAMS)
        expect(result.finalValueAfterTax).toBeGreaterThan(withTER.finalValueAfterTax)
    })

    it('totalGain equals finalValueAfterTax + taxPaid - initial', () => {
        const result = computePortfolioProjection(BASE_PARAMS)
        expect(result.totalGain).toBeCloseTo(
            result.finalValueAfterTax + result.taxPaid - BASE_PARAMS.initialContribution, 2
        )
    })

    it('defaults missing taxModel to exit', () => {
        const params = { ...BASE_PARAMS }
        delete (params as any).taxModel
        const result = computePortfolioProjection(params)
        const exitResult = computePortfolioProjection({ ...BASE_PARAMS, taxModel: 'exit' })
        expect(result.finalValueAfterTax).toBeCloseTo(exitResult.finalValueAfterTax, 2)
    })
})
