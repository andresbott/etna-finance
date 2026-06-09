import { describe, it, expect } from 'vitest'
import { computeBondsProjection, computeBondsExpectedReturn } from './bonds'
import type { BondsSimulatorParams } from '@/lib/api/ToolsData'

const BASE: BondsSimulatorParams = {
    faceValue: 1000,
    purchasePrice: 1000,
    couponRatePct: 4,
    couponFrequency: 1,
    maturityDate: '2036-06-01',
    taxesPct: 0,
}

describe('computeBondsExpectedReturn — YTM properties', () => {
    it('par bond (price = face, no tax): YTM ≈ coupon rate', () => {
        expect(computeBondsExpectedReturn(BASE)).toBeCloseTo(4, 1)
    })

    it('discount bond (price < face): YTM > coupon rate', () => {
        const ytm = computeBondsExpectedReturn({ ...BASE, purchasePrice: 900 })
        expect(ytm).toBeGreaterThan(4)
    })

    it('premium bond (price > face): YTM < coupon rate', () => {
        const ytm = computeBondsExpectedReturn({ ...BASE, purchasePrice: 1100 })
        expect(ytm).toBeLessThan(4)
    })

    it('zero-coupon discount bond: positive YTM from price gain alone', () => {
        const ytm = computeBondsExpectedReturn({ ...BASE, couponRatePct: 0, purchasePrice: 700 })
        expect(ytm).toBeGreaterThan(0)
    })

    it('taxes reduce net YTM', () => {
        const taxed = computeBondsExpectedReturn({ ...BASE, taxesPct: 25 })
        expect(taxed).toBeLessThan(computeBondsExpectedReturn(BASE))
    })
})

describe('computeBondsProjection', () => {
    it('returns years 0..maturity', () => {
        const p = computeBondsProjection(BASE)
        expect(p.years).toHaveLength(11)
        expect(p.years[0]).toBe(0)
        expect(p.years[10]).toBe(10)
    })

    it('invested series is flat at purchase price', () => {
        const p = computeBondsProjection(BASE)
        expect(p.series.invested.every(v => v === 1000)).toBe(true)
    })

    it('total value starts at purchase price and ends above it for a profitable bond', () => {
        const p = computeBondsProjection(BASE)
        expect(p.series.totalValue[0]).toBe(1000)
        expect(p.series.totalValue[10]).toBeGreaterThan(1000)
    })

    it('cumulative coupons accumulate after-tax', () => {
        const p = computeBondsProjection({ ...BASE, taxesPct: 25 })
        // gross coupon 40/yr, net 30/yr, 10 yrs = 300
        expect(p.series.cumulativeCoupons[10]).toBeCloseTo(300, 1)
    })

    it('capital gain tax is applied to a discount bond redemption gain', () => {
        const p = computeBondsProjection({ ...BASE, purchasePrice: 900, taxesPct: 25 })
        expect(p.capitalGain).toBe(100)
        expect(p.capitalGainTaxPaid).toBeCloseTo(25, 5)
    })

    it('premium bond has no capital gain tax', () => {
        const p = computeBondsProjection({ ...BASE, purchasePrice: 1100, taxesPct: 25 })
        expect(p.capitalGain).toBe(-100)
        expect(p.capitalGainTaxPaid).toBe(0)
    })

    it('fractional maturity adds a partial final-year grid point', () => {
        const p = computeBondsProjection(BASE, 9.5)
        expect(p.years[p.years.length - 1]).toBeCloseTo(9.5, 5)
        // whole-year points 0..9 plus the 9.5 point
        expect(p.years).toHaveLength(11)
    })

    it('sub-year maturity accrues a prorated final coupon', () => {
        // half a year of a 40 gross coupon, no tax = 20
        const p = computeBondsProjection(BASE, 0.5)
        expect(p.years).toEqual([0, 0.5])
        expect(p.series.cumulativeCoupons[p.series.cumulativeCoupons.length - 1]).toBeCloseTo(20, 5)
    })

    it('zero-input guard: face = price = 0 returns zeroed series, no NaN', () => {
        const p = computeBondsProjection({ ...BASE, faceValue: 0, purchasePrice: 0 })
        expect(p.netYTM).toBe(0)
        expect(p.finalValue).toBe(0)
        expect(p.series.totalValue.every(v => v === 0)).toBe(true)
    })
})
