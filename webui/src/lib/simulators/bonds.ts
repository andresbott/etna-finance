import type { BondsSimulatorParams } from '@/lib/api/ToolsData'

const DEFAULT_YEARS = 10
const MAX_YEARS = 50
const MIN_YEARS = 1 / 12 // one month
const EPS = 1e-9

export interface BondsProjectionSeries {
    invested: number[]
    totalValue: number[]
    cumulativeCoupons: number[]
}

export interface BondsProjection {
    years: number[]
    invested: number
    totalCoupons: number          // gross coupon income over the life
    couponTaxPaid: number
    capitalGain: number           // face − purchase (negative for premium bonds)
    capitalGainTaxPaid: number
    totalReturn: number           // net gain over purchase price
    finalValue: number            // net value at maturity (incl. accumulated coupons)
    netYTM: number                // after-tax YTM (IRR of after-tax cash flows), %
    series: BondsProjectionSeries
}

interface CashFlow {
    t: number      // time in years from purchase
    amount: number
}

// IRR of the after-tax cash flows, expressed as an effective annual %.
// price is paid at t=0; each cash flow is discounted by its time in years.
function computeNetYTM(price: number, cashFlows: CashFlow[]): number {
    if (price <= 0 || cashFlows.length === 0) return 0

    const npv = (y: number): number => {
        let sum = -price
        for (const cf of cashFlows) {
            sum += cf.amount / Math.pow(1 + y, cf.t)
        }
        return sum
    }

    let lo = -0.9999
    let hi = 1
    let fLo = npv(lo)
    let fHi = npv(hi)
    let guard = 0
    while (fLo * fHi > 0 && hi < 1000 && guard < 100) {
        hi *= 2
        fHi = npv(hi)
        guard++
    }
    if (fLo * fHi > 0) return 0

    let mid = 0
    for (let k = 0; k < 200; k++) {
        mid = (lo + hi) / 2
        const fMid = npv(mid)
        if (Math.abs(fMid) < EPS) break
        if (fLo * fMid < 0) {
            hi = mid
            fHi = fMid
        } else {
            lo = mid
            fLo = fMid
        }
    }

    return mid * 100
}

export function computeBondsProjection(params: BondsSimulatorParams, durationYears?: number): BondsProjection {
    const face = params.faceValue ?? 0
    const price = params.purchasePrice ?? 0
    const couponPct = params.couponRatePct ?? 0
    const freq = params.couponFrequency === 2 ? 2 : 1
    const taxRate = (params.taxesPct ?? 0) / 100
    const T = Math.min(Math.max(MIN_YEARS, durationYears ?? DEFAULT_YEARS), MAX_YEARS)

    // Year-end grid points for the chart: whole years up to maturity, plus a final
    // partial-year point when maturity does not fall on a whole year.
    const lastWhole = Math.floor(T + EPS)
    const gridTimes: number[] = []
    for (let i = 0; i <= lastWhole; i++) gridTimes.push(i)
    if (T - lastWhole > EPS) gridTimes.push(T)

    if (face === 0 && price === 0) {
        const zeros = gridTimes.map(() => 0)
        return {
            years: gridTimes,
            invested: 0,
            totalCoupons: 0,
            couponTaxPaid: 0,
            capitalGain: 0,
            capitalGainTaxPaid: 0,
            totalReturn: 0,
            finalValue: 0,
            netYTM: 0,
            series: { invested: zeros, totalValue: zeros, cumulativeCoupons: zeros },
        }
    }

    const grossCouponPerPeriod = (face * (couponPct / 100)) / freq
    const netCouponPerPeriod = grossCouponPerPeriod * (1 - taxRate)

    // Full coupon periods plus a possible prorated stub for the partial final period.
    const fullPeriods = Math.floor(T * freq + EPS)
    const stubFraction = T * freq - fullPeriods // fraction of a coupon period (0..1)
    const hasStub = stubFraction > EPS
    const stubGrossCoupon = grossCouponPerPeriod * stubFraction
    const stubNetCoupon = stubGrossCoupon * (1 - taxRate)

    const capitalGain = face - price
    const capitalGainTaxPaid = Math.max(0, capitalGain) * taxRate
    const netRedemption = face - capitalGainTaxPaid

    // Net coupon income accumulated (as cash, no reinvestment return) by a given time.
    const cumNetCouponsBy = (t: number): number => {
        const periods = Math.min(fullPeriods, Math.floor(t * freq + EPS))
        let total = netCouponPerPeriod * periods
        if (t >= T - EPS && hasStub) total += stubNetCoupon
        return total
    }

    const investedArr = gridTimes.map(() => price)
    const cumCouponsArr = gridTimes.map((t) => cumNetCouponsBy(t))
    const totalValueArr = gridTimes.map((t, i) => {
        const principal = t >= T - EPS ? netRedemption : price
        return principal + cumCouponsArr[i]
    })

    const sumNetCoupons = netCouponPerPeriod * fullPeriods + (hasStub ? stubNetCoupon : 0)
    const finalValue = netRedemption + sumNetCoupons
    const totalReturn = finalValue - price
    const totalCoupons = grossCouponPerPeriod * fullPeriods + (hasStub ? stubGrossCoupon : 0)
    const couponTaxPaid = totalCoupons * taxRate

    // After-tax cash flows for the YTM: a coupon at each full period, then the
    // redemption (plus any prorated stub coupon) at maturity.
    const cashFlows: CashFlow[] = []
    for (let p = 1; p <= fullPeriods; p++) {
        cashFlows.push({ t: p / freq, amount: netCouponPerPeriod })
    }
    cashFlows.push({ t: T, amount: netRedemption + (hasStub ? stubNetCoupon : 0) })
    const netYTM = computeNetYTM(price, cashFlows)

    return {
        years: gridTimes,
        invested: price,
        totalCoupons,
        couponTaxPaid,
        capitalGain,
        capitalGainTaxPaid,
        totalReturn,
        finalValue,
        netYTM,
        series: { invested: investedArr, totalValue: totalValueArr, cumulativeCoupons: cumCouponsArr },
    }
}

export function computeBondsExpectedReturn(params: BondsSimulatorParams, durationYears?: number): number {
    return computeBondsProjection(params, durationYears).netYTM
}
