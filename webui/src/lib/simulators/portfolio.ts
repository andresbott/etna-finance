import type { PortfolioSimulatorParams } from '@/lib/api/ToolsData'

const DEFAULT_DURATION_YEARS = 20
const MAX_DURATION_YEARS = 50

export interface PortfolioProjectionSeries {
    totalInvested: number[]
    netWorth: number[]
    totalGains: number[]
    taxImpact: number[]
}

export interface PortfolioProjection {
    years: number[]
    totalContributions: number
    finalValueBeforeTax: number
    finalValueAfterTax: number
    totalGain: number
    taxPaid: number
    series: PortfolioProjectionSeries
}

export function computePortfolioExpectedReturn(params: PortfolioSimulatorParams): number {
    const effectiveReturn = (params.growthRatePct ?? 0) - (params.expenseRatioPct ?? 0)
    const taxRate = (params.capitalGainTaxPct ?? 0) / 100
    const taxModel = params.taxModel ?? 'exit'

    const estimatedTaxDrag = taxModel === 'exit'
        ? taxRate * effectiveReturn / DEFAULT_DURATION_YEARS
        : taxRate * effectiveReturn

    return effectiveReturn - estimatedTaxDrag
}

export function computePortfolioProjection(params: PortfolioSimulatorParams, durationYears: number = DEFAULT_DURATION_YEARS): PortfolioProjection {
    const duration = Math.min(Math.max(1, durationYears), MAX_DURATION_YEARS)
    const initial = params.initialContribution ?? 0
    const growthPct = params.growthRatePct ?? 0
    const expensePct = params.expenseRatioPct ?? 0
    const taxPct = params.capitalGainTaxPct ?? 0
    const taxModel = params.taxModel ?? 'exit'

    const effectiveReturn = growthPct - expensePct
    const taxRate = taxPct / 100

    if (initial === 0) {
        const zeros = Array(duration + 1).fill(0)
        return {
            years: Array.from({ length: duration + 1 }, (_, i) => i),
            totalContributions: 0,
            finalValueBeforeTax: 0,
            finalValueAfterTax: 0,
            totalGain: 0,
            taxPaid: 0,
            series: {
                totalInvested: zeros,
                netWorth: zeros,
                totalGains: zeros,
                taxImpact: zeros,
            },
        }
    }

    const monthlyRate = Math.pow(1 + effectiveReturn / 100, 1 / 12) - 1

    const yearLabels: number[] = [0]
    const balances: number[] = [initial]
    const cumulativeTaxes: number[] = [0]

    let balance = initial
    let cumulativeTax = 0

    for (let y = 1; y <= duration; y++) {
        const balanceStartOfYear = balance

        for (let m = 0; m < 12; m++) {
            balance = balance * (1 + monthlyRate)
        }

        if (taxModel === 'annual') {
            const yearGain = balance - balanceStartOfYear
            if (yearGain > 0) {
                const tax = yearGain * taxRate
                cumulativeTax += tax
                balance -= tax
            }
        }

        yearLabels.push(y)
        balances.push(balance)
        cumulativeTaxes.push(cumulativeTax)
    }

    let finalValueBeforeTax = balance
    let finalValueAfterTax = balance
    if (taxModel === 'exit') {
        const totalGains = balance - initial
        if (totalGains > 0) {
            const exitTax = totalGains * taxRate
            cumulativeTax = exitTax
            finalValueAfterTax = balance - exitTax
            balances[duration] = finalValueAfterTax
            cumulativeTaxes[duration] = exitTax
        }
    } else {
        finalValueBeforeTax = balance + cumulativeTax
        finalValueAfterTax = balance
    }

    const totalTaxPaid = cumulativeTax

    const totalInvestedSeries = Array(duration + 1).fill(initial)
    const netWorthSeries = balances
    const taxImpactSeries = cumulativeTaxes
    const totalGainsSeries = yearLabels.map((_, i) =>
        netWorthSeries[i] + cumulativeTaxes[i] - initial
    )

    return {
        years: yearLabels,
        totalContributions: initial,
        finalValueBeforeTax,
        finalValueAfterTax,
        totalGain: finalValueAfterTax + totalTaxPaid - initial,
        taxPaid: totalTaxPaid,
        series: {
            totalInvested: totalInvestedSeries,
            netWorth: netWorthSeries,
            totalGains: totalGainsSeries,
            taxImpact: taxImpactSeries,
        },
    }
}

export function computePortfolioNetWorth20Y(params: PortfolioSimulatorParams, durationYears?: number): number[] {
    const result = computePortfolioProjection(params, durationYears)
    return result.series.netWorth
}
