import type { RealEstateSimulatorParams } from '@/lib/api/ToolsData'

// ── Types ────────────────────────────────────────────────────────────

export interface AmortizationYearMortgage {
    name: string
    beginningBalance: number
    interestPaid: number
    principalPaid: number
    endingBalance: number
}

export interface AmortizationYear {
    year: number
    mortgages: AmortizationYearMortgage[]
    totalBeginning: number
    totalInterest: number
    totalPrincipal: number
    totalEnding: number
}

export interface RealEstateProjection {
    yearLabels: number[]
    propertyEquity: number[]
    remainingMortgage: number[]
    cumulativeInterest: number[]
    cumulativeCashFlow: number[]
    netWorth: number[]
}

// ── Helpers (pure) ───────────────────────────────────────────────────

export function calcMonthlyPayment(
    principal: number,
    annualRate: number,
    termYears: number,
    amortize: boolean
): number {
    if (principal <= 0 || annualRate < 0 || termYears <= 0) return 0
    if (!amortize) {
        return principal * (annualRate / 100) / 12
    }
    const monthlyRate = annualRate / 100 / 12
    if (monthlyRate === 0) {
        return principal / (termYears * 12)
    }
    const months = termYears * 12
    const factor = Math.pow(1 + monthlyRate, months)
    return principal * (monthlyRate * factor) / (factor - 1)
}

export function calcTotalInterest(
    principal: number,
    annualRate: number,
    termYears: number,
    amortize: boolean
): number {
    const monthly = calcMonthlyPayment(principal, annualRate, termYears, amortize)
    if (amortize) {
        return monthly * termYears * 12 - principal
    }
    // Interest-only: return annual interest (term is irrelevant)
    return monthly * 12
}

// ── Derived intermediates from params ────────────────────────────────

function deriveTotalEquity(params: RealEstateSimulatorParams): number {
    const additional = (params.additionalEquity ?? []).reduce((sum, e) => sum + (e.amount ?? 0), 0)
    return (params.cashEquity ?? 0) + additional
}

function deriveTotalMortgageNeeded(params: RealEstateSimulatorParams): number {
    return Math.max(0, (params.purchasePrice ?? 0) - deriveTotalEquity(params))
}

function deriveMortgagePrincipal(params: RealEstateSimulatorParams, m: { splitPct: number }): number {
    return deriveTotalMortgageNeeded(params) * (m.splitPct ?? 0) / 100
}

function deriveIncidentalCost(params: RealEstateSimulatorParams): number {
    return (params.purchasePrice ?? 0) * (params.incidentalPct ?? 0) / 100
}

function deriveMaintenanceCost(params: RealEstateSimulatorParams): number {
    return (params.purchasePrice ?? 0) * (params.maintenancePct ?? 0) / 100
}

function deriveAnnualRentGross(params: RealEstateSimulatorParams): number {
    return (params.monthlyRent ?? 0) * 12
}

function deriveTotalRecurringCosts(params: RealEstateSimulatorParams): number {
    if (params.useSimpleCosts ?? true) {
        return deriveIncidentalCost(params) + (params.otherCosts ?? 0)
    }
    const grossRent = deriveAnnualRentGross(params)
    const vacancyCost = grossRent * (params.vacancyPct ?? 0) / 100
    const managementCost = grossRent * (params.managementPct ?? 0) / 100
    return (params.propertyTax ?? 0) + (params.insurance ?? 0) + deriveMaintenanceCost(params)
        + (params.renovationFund ?? 0) + vacancyCost + managementCost
}

function deriveAnnualRent(params: RealEstateSimulatorParams): number {
    return (params.monthlyRent ?? 0) * 12
}

function deriveMaxTerm(params: RealEstateSimulatorParams): number {
    const mortgages = params.mortgages ?? []
    if (mortgages.length === 0) return 1
    const hasNonAmortizing = mortgages.some(m => !m.amortize)
    const amortizing = mortgages.filter(m => m.amortize)
    const baseTerm = amortizing.length > 0
        ? Math.max(...amortizing.map(m => m.termYears), 1)
        : Math.max(...mortgages.map(m => m.termYears), 1)
    return hasNonAmortizing ? baseTerm + 5 : baseTerm
}

// ── Main computation functions ───────────────────────────────────────

function computeAmortizationScheduleForYears(params: RealEstateSimulatorParams, minYears: number): AmortizationYear[] {
    const maxTerm = Math.max(deriveMaxTerm(params), minYears)
    return computeAmortizationScheduleInternal(params, maxTerm)
}

export function computeAmortizationSchedule(params: RealEstateSimulatorParams): AmortizationYear[] {
    return computeAmortizationScheduleInternal(params, deriveMaxTerm(params))
}

function computeAmortizationScheduleInternal(params: RealEstateSimulatorParams, maxTerm: number): AmortizationYear[] {
    const mortgages = params.mortgages ?? []
    const years: AmortizationYear[] = []

    const balances = mortgages.map(m => deriveMortgagePrincipal(params, m))

    for (let y = 1; y <= maxTerm; y++) {
        const yearData = mortgages.map((m, i) => {
            const principal = deriveMortgagePrincipal(params, m)
            const bal = balances[i]
            if (bal <= 0 || (m.amortize && y > m.termYears)) {
                return {
                    name: m.name,
                    beginningBalance: Math.max(0, bal),
                    interestPaid: 0,
                    principalPaid: 0,
                    endingBalance: Math.max(0, bal)
                }
            }

            const monthlyRate = m.interestRate / 100 / 12
            let yearInterest = 0
            let yearPrincipal = 0
            let currentBal = bal

            const monthlyPayment = calcMonthlyPayment(principal, m.interestRate, m.termYears, m.amortize)

            for (let month = 0; month < 12; month++) {
                if (currentBal <= 0) break
                const interestThisMonth = currentBal * monthlyRate
                let principalThisMonth: number
                if (m.amortize) {
                    principalThisMonth = Math.min(monthlyPayment - interestThisMonth, currentBal)
                } else {
                    principalThisMonth = 0
                }
                yearInterest += interestThisMonth
                yearPrincipal += principalThisMonth
                currentBal -= principalThisMonth
            }

            balances[i] = currentBal

            return {
                name: m.name,
                beginningBalance: bal,
                interestPaid: yearInterest,
                principalPaid: yearPrincipal,
                endingBalance: currentBal
            }
        })

        years.push({
            year: y,
            mortgages: yearData,
            totalBeginning: yearData.reduce((s, m) => s + m.beginningBalance, 0),
            totalInterest: yearData.reduce((s, m) => s + m.interestPaid, 0),
            totalPrincipal: yearData.reduce((s, m) => s + m.principalPaid, 0),
            totalEnding: yearData.reduce((s, m) => s + m.endingBalance, 0)
        })
    }

    return years
}

export function computeRealEstateProjection(params: RealEstateSimulatorParams): RealEstateProjection {
    const schedule = computeAmortizationSchedule(params)
    const totalMortgagePrincipal = (params.mortgages ?? []).reduce(
        (sum, m) => sum + deriveMortgagePrincipal(params, m), 0
    )
    const yearLabels = [0, ...schedule.map(s => s.year)]

    const initialMortgageBalance = totalMortgagePrincipal
    const remainingMortgage = [initialMortgageBalance, ...schedule.map(s => s.totalEnding)]
    const mv = params.marketValue ?? 0
    const annualGrowth = (params.housingPriceIncreasePct ?? 0) / 100
    const propertyEquity = yearLabels.map((_, i) => {
        const projectedValue = mv * Math.pow(1 + annualGrowth, yearLabels[i])
        return projectedValue - remainingMortgage[i]
    })

    let cumulativeInterest = 0
    const cumulativeInterestSeries = [0]
    for (const yr of schedule) {
        cumulativeInterest += yr.totalInterest
        cumulativeInterestSeries.push(cumulativeInterest)
    }

    const annualRent = deriveAnnualRent(params)
    const totalRecurringCosts = deriveTotalRecurringCosts(params)

    let cumCashFlow = 0
    const cumulativeCashFlow = [0]
    for (const yr of schedule) {
        const yearMortgagePayments = yr.totalInterest + yr.totalPrincipal
        cumCashFlow += annualRent - totalRecurringCosts - yearMortgagePayments
        cumulativeCashFlow.push(cumCashFlow)
    }

    const netWorth = yearLabels.map((_, i) => propertyEquity[i] + cumulativeCashFlow[i])

    return {
        yearLabels,
        propertyEquity,
        remainingMortgage,
        cumulativeInterest: cumulativeInterestSeries,
        cumulativeCashFlow,
        netWorth
    }
}

export function computeRealEstateNetWorth20Y(params: RealEstateSimulatorParams): number[] {
    const projection = computeRealEstateProjection(params)
    const scheduleLen = projection.yearLabels.length // includes year 0

    const mv = params.marketValue ?? 0
    const annualGrowth = (params.housingPriceIncreasePct ?? 0) / 100
    const annualRent = deriveAnnualRent(params)
    const totalRecurringCosts = deriveTotalRecurringCosts(params)

    const result: number[] = []
    const TARGET = 21 // years 0-20

    for (let i = 0; i < TARGET; i++) {
        if (i < scheduleLen) {
            // Within schedule range: netWorth = propertyEquity + cumulativeCashFlow
            result.push(projection.propertyEquity[i] + projection.cumulativeCashFlow[i])
        } else {
            // Beyond schedule: mortgage is paid off, continue projecting
            const year = i
            const projectedValue = mv * Math.pow(1 + annualGrowth, year)
            // Remaining mortgage is 0 (all paid off)
            const propertyEquity = projectedValue

            // Cash flow beyond schedule: no mortgage payments, just rent - recurring costs
            const lastScheduleCashFlow = projection.cumulativeCashFlow[scheduleLen - 1]
            const extraYears = i - (scheduleLen - 1)
            const annualNetCashFlow = annualRent - totalRecurringCosts
            const cumCashFlow = lastScheduleCashFlow + annualNetCashFlow * extraYears

            result.push(propertyEquity + cumCashFlow)
        }
    }

    return result
}

export function computeRealEstateExpectedReturn(params: RealEstateSimulatorParams): number {
    const annualRent = deriveAnnualRent(params)
    const totalRecurringCosts = deriveTotalRecurringCosts(params)
    const noi = annualRent - totalRecurringCosts
    const mv = params.marketValue ?? 0
    return mv > 0 ? (noi / mv) * 100 : 0
}

export function computeRealEstateAnnualYield(params: RealEstateSimulatorParams, years: number): number[] {
    const schedule = computeAmortizationScheduleForYears(params, years)
    const totalEquity = deriveTotalEquity(params)
    if (totalEquity <= 0) return Array(years).fill(0)

    const annualRent = deriveAnnualRent(params)
    const totalRecurringCosts = deriveTotalRecurringCosts(params)
    const noi = annualRent - totalRecurringCosts
    const annualAppreciation = (params.marketValue ?? 0) * (params.housingPriceIncreasePct ?? 0) / 100

    const result: number[] = []
    for (let y = 1; y <= years; y++) {
        const yr = schedule[y - 1]
        const yearMortgagePayments = yr.totalInterest + yr.totalPrincipal
        const leveragedCashFlow = noi - yearMortgagePayments
        const equityBuildup = yr.totalPrincipal
        result.push(((leveragedCashFlow + equityBuildup + annualAppreciation) / totalEquity) * 100)
    }
    return result
}
