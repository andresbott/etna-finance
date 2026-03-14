import type { PortfolioSimulatorParams } from '@/lib/api/ToolsData'

export interface PortfolioProjectionSeries {
    totalInvested: number[]
    netWorth: number[]
    inflationAdjustedNetWorth: number[]
    totalGains: number[]
    taxImpact: number[]
    inflationAdjustedGains: number[]
}

export interface PortfolioProjection {
    years: number[]
    values: number[]
    totalContributions: number
    finalValue: number
    finalValueAfterTax: number
    realFinalValue: number
    totalGain: number
    taxPaid: number
    inflationImpact: number
    inflationAdjustedGains: number
    series: PortfolioProjectionSeries
}

/**
 * Compute the expected annual return after tax and inflation.
 */
export function computePortfolioExpectedReturn(params: PortfolioSimulatorParams): number {
    const growth = params.growthRatePct ?? 0
    const tax = params.capitalGainTaxPct ?? 0
    const inflation = params.inflationPct ?? 0
    return growth - tax - inflation
}

/**
 * Project portfolio value year by year with compound growth and monthly contributions.
 * Pure function — no Vue reactivity.
 */
export function computePortfolioProjection(params: PortfolioSimulatorParams): PortfolioProjection {
    const years = params.durationYears
    const initial = params.initialContribution ?? 0
    const monthly = params.monthlyContribution ?? 0
    const growthPct = params.growthRatePct ?? 0
    const taxPct = params.capitalGainTaxPct ?? 0

    if (years <= 0) {
        return {
            years: [0],
            values: [initial],
            totalContributions: initial,
            finalValue: initial,
            finalValueAfterTax: initial,
            realFinalValue: initial,
            totalGain: 0,
            taxPaid: 0,
            inflationImpact: 0,
            inflationAdjustedGains: 0,
            series: {
                totalInvested: [initial],
                netWorth: [initial],
                inflationAdjustedNetWorth: [initial],
                totalGains: [0],
                taxImpact: [0],
                inflationAdjustedGains: [0],
            },
        }
    }

    const monthlyRate = Math.pow(1 + growthPct / 100, 1 / 12) - 1
    const totalContributions = initial + monthly * 12 * years
    const taxRate = taxPct / 100

    const yearLabels = [0]
    const yearValues = [initial]
    const taxPerYear = [0]
    let balance = initial
    let cumulativeTax = 0

    for (let y = 1; y <= years; y++) {
        for (let m = 0; m < 12; m++) {
            balance = (balance + monthly) * (1 + monthlyRate)
        }
        const netWorthBeforeTax = balance
        const taxThisYear = netWorthBeforeTax * taxRate
        cumulativeTax += taxThisYear
        balance = netWorthBeforeTax - taxThisYear

        yearLabels.push(y)
        yearValues.push(balance)
        taxPerYear.push(taxThisYear)
    }

    const finalValueAfterTax = balance
    const taxPaid = cumulativeTax
    const gain = finalValueAfterTax + taxPaid - totalContributions
    const inflationPctVal = params.inflationPct ?? 0
    const inflationFactor = 1 + inflationPctVal / 100
    const realFinalValue =
        inflationPctVal > 0 ? finalValueAfterTax / Math.pow(inflationFactor, years) : finalValueAfterTax
    const inflationImpactTotal = finalValueAfterTax - realFinalValue
    const realCostBasis =
        inflationPctVal > 0 ? totalContributions / Math.pow(inflationFactor, years) : totalContributions
    const inflationAdjustedGainsFinal = realFinalValue - realCostBasis

    // Per-year series for the chart
    const totalInvestedSeries = yearLabels.map((yr) => initial + monthly * 12 * yr)
    const netWorthSeries = yearValues
    const cumulativeTaxByYear = taxPerYear.map((_, i) =>
        taxPerYear.slice(0, i + 1).reduce((a, b) => a + b, 0),
    )
    const totalGainsSeries = yearLabels.map(
        (_, i) => netWorthSeries[i] + cumulativeTaxByYear[i] - totalInvestedSeries[i],
    )
    const taxImpactSeries = cumulativeTaxByYear
    const inflationAdjustedNetWorthSeries = yearLabels.map((y, i) =>
        inflationPctVal > 0 ? netWorthSeries[i] / Math.pow(inflationFactor, y) : netWorthSeries[i],
    )
    const realCostBasisSeries = yearLabels.map((y, i) =>
        inflationPctVal > 0 ? totalInvestedSeries[i] / Math.pow(inflationFactor, y) : totalInvestedSeries[i],
    )
    const inflationAdjustedGainsSeries = yearLabels.map(
        (_, i) => inflationAdjustedNetWorthSeries[i] - realCostBasisSeries[i],
    )

    return {
        years: yearLabels,
        values: yearValues,
        totalContributions,
        finalValue: finalValueAfterTax,
        finalValueAfterTax,
        realFinalValue,
        totalGain: gain,
        taxPaid,
        inflationImpact: inflationImpactTotal,
        inflationAdjustedGains: inflationAdjustedGainsFinal,
        series: {
            totalInvested: totalInvestedSeries,
            netWorth: netWorthSeries,
            inflationAdjustedNetWorth: inflationAdjustedNetWorthSeries,
            totalGains: totalGainsSeries,
            taxImpact: taxImpactSeries,
            inflationAdjustedGains: inflationAdjustedGainsSeries,
        },
    }
}

/**
 * Convenience: always project 20 years and return the inflation-adjusted net worth series.
 */
export function computePortfolioNetWorth20Y(params: PortfolioSimulatorParams): number[] {
    const result = computePortfolioProjection({ ...params, durationYears: 20 })
    return result.series.inflationAdjustedNetWorth
}
