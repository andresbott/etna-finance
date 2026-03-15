import type { BuyVsRentSimulatorParams } from '@/lib/api/ToolsData'

// ── Types ────────────────────────────────────────────────────────────

export interface BuyVsRentProjection {
    yearLabels: number[]
    buyNetWorth: number[]
    rentNetWorth: number[]
}

// ── Helpers ──────────────────────────────────────────────────────────

function calcMonthlyMortgagePayment(
    principal: number,
    annualRate: number,
    termYears: number,
    amortize: boolean
): number {
    if (principal <= 0 || annualRate < 0 || termYears <= 0) return 0
    if (!amortize) return principal * (annualRate / 100) / 12
    const monthlyRate = annualRate / 100 / 12
    if (monthlyRate === 0) return principal / (termYears * 12)
    const months = termYears * 12
    const factor = Math.pow(1 + monthlyRate, months)
    return principal * (monthlyRate * factor) / (factor - 1)
}

// ── Main computation ─────────────────────────────────────────────────

export function computeBuyVsRentProjection(params: BuyVsRentSimulatorParams): BuyVsRentProjection {
    const years = 20
    const purchasePrice = params.purchasePrice ?? 0
    const cashEquity = params.cashEquity ?? 0
    const additionalEquityTotal = (params.additionalEquity ?? []).reduce((s, e) => s + (e.amount ?? 0), 0)
    const totalEquity = cashEquity + additionalEquityTotal
    const totalMortgageNeeded = Math.max(0, purchasePrice - totalEquity)
    const appreciation = (params.housingPriceIncreasePct ?? 0) / 100
    const currentRent = params.currentMonthlyRent ?? 0
    const rentIncrease = (params.rentIncreasePct ?? 0) / 100
    const etfReturn = (params.etfReturnPct ?? 0) / 100

    // Property recurring costs (annual)
    const incidentalCost = purchasePrice * (params.incidentalPct ?? 0) / 100
    const maintenanceCost = purchasePrice * (params.maintenancePct ?? 0) / 100
    const annualRent = currentRent * 12
    const vacancyCost = annualRent * (params.vacancyPct ?? 0) / 100
    const managementCost = annualRent * (params.managementPct ?? 0) / 100

    const detailedCosts = (params.propertyTax ?? 0) + (params.insurance ?? 0)
        + maintenanceCost + (params.renovationFund ?? 0)
        + vacancyCost + managementCost
    const simpleCosts = incidentalCost + (params.otherCosts ?? 0)
    const annualRecurringCosts = (params.useSimpleCosts ?? true) ? simpleCosts : detailedCosts

    // Mortgage details
    const mortgages = params.mortgages ?? []
    const mortgageInfos = mortgages.map(m => {
        const principal = totalMortgageNeeded * (m.splitPct ?? 0) / 100
        const monthly = calcMonthlyMortgagePayment(principal, m.interestRate, m.termYears, m.amortize)
        return { principal, monthly, rate: m.interestRate / 100 / 12, amortize: m.amortize, termYears: m.termYears }
    })

    const yearLabels: number[] = [0]
    const buyNetWorth: number[] = [totalEquity]
    const rentNetWorth: number[] = [totalEquity]

    // Track mortgage balances for buy scenario
    const balances = mortgageInfos.map(m => m.principal)

    // Track ETF balance for rent scenario
    const monthlyEtfRate = Math.pow(1 + etfReturn, 1 / 12) - 1
    let etfBalance = totalEquity

    for (let y = 1; y <= years; y++) {
        // ── BUY SCENARIO (year y) ──
        // Amortization: compute interest/principal for each mortgage this year
        let yearMortgagePayment = 0
        for (let mi = 0; mi < mortgageInfos.length; mi++) {
            const m = mortgageInfos[mi]
            if (balances[mi] <= 0 || (m.amortize && y > m.termYears)) continue
            for (let month = 0; month < 12; month++) {
                if (balances[mi] <= 0) break
                const interest = balances[mi] * m.rate
                let principalPayment = 0
                if (m.amortize) {
                    principalPayment = Math.min(m.monthly - interest, balances[mi])
                }
                yearMortgagePayment += m.monthly
                balances[mi] -= principalPayment
            }
        }

        const propertyValue = purchasePrice * Math.pow(1 + appreciation, y)
        const remainingMortgage = balances.reduce((s, b) => s + Math.max(0, b), 0)
        const propertyEquity = propertyValue - remainingMortgage

        // Cumulative cash flow for buy: rent income is 0 (owner-occupied), costs go out
        // We track net worth = property equity + cumulative cash savings
        // But for buy, the "cost" each year is mortgage + recurring, and the "benefit" is not paying rent
        // So buy cash flow = -(mortgage + recurring) + saved rent (which is 0 for owner-occupied, but we account for it differently)

        // ── RENT SCENARIO (year y) ──
        // Monthly: pay rent, invest the rest
        const monthlyBuyCost = yearMortgagePayment / 12 + annualRecurringCosts / 12
        const yearRent = currentRent * 12 * Math.pow(1 + rentIncrease, y - 1)
        const monthlyRentThisYear = yearRent / 12

        for (let month = 0; month < 12; month++) {
            // Rent scenario: pay rent, invest savings (difference between buy cost and rent)
            const monthlySavings = monthlyBuyCost - monthlyRentThisYear
            etfBalance = (etfBalance + monthlySavings) * (1 + monthlyEtfRate)
        }

        yearLabels.push(y)

        // Buy net worth: property equity (no cumulative cash flow tracking needed since
        // both scenarios spend money — we compare final wealth positions)
        buyNetWorth.push(propertyEquity)

        // Rent net worth: ETF balance
        rentNetWorth.push(etfBalance)
    }

    return { yearLabels, buyNetWorth, rentNetWorth }
}

export function computeBuyVsRentNetWorth20Y(params: BuyVsRentSimulatorParams): number[] {
    const projection = computeBuyVsRentProjection(params)
    return projection.buyNetWorth
}
