import { apiClient } from '@/lib/api/client'

export interface CaseStudy<T = Record<string, unknown>> {
    id: number
    toolType: string
    name: string
    description: string
    expectedAnnualReturn: number
    params: T
    createdAt: string
    updatedAt: string
}

export interface PortfolioSimulatorParams {
    durationYears: number
    initialContribution: number
    monthlyContribution: number
    growthRatePct: number
    inflationPct: number
    capitalGainTaxPct: number
}

export interface RealEstateSimulatorParams {
    purchasePrice: number
    marketValue: number
    squareMeters: number
    monthlyRent: number
    propertyTax: number
    insurance: number
    maintenance: number
    otherCosts: number
    cashEquity: number
    additionalEquity: Array<{ name: string; amount: number }>
    mortgages: Array<{
        name: string
        principal: number
        interestRate: number
        termYears: number
        amortize: boolean
    }>
    grossMonthlyIncome: number
}

function toolPath(toolType: string): string {
    return `/tools/${toolType}/cases`
}

export async function listCases<T = Record<string, unknown>>(toolType: string): Promise<CaseStudy<T>[]> {
    const { data } = await apiClient.get<CaseStudy<T>[]>(toolPath(toolType))
    return data ?? []
}

export async function getCase<T = Record<string, unknown>>(toolType: string, id: number): Promise<CaseStudy<T>> {
    const { data } = await apiClient.get<CaseStudy<T>>(`${toolPath(toolType)}/${id}`)
    return data
}

export async function createCase<T = Record<string, unknown>>(
    toolType: string,
    payload: { name: string; description: string; expectedAnnualReturn: number; params: T }
): Promise<CaseStudy<T>> {
    const { data } = await apiClient.post<CaseStudy<T>>(toolPath(toolType), payload)
    return data
}

export async function updateCase<T = Record<string, unknown>>(
    toolType: string,
    id: number,
    payload: { name?: string; description?: string; expectedAnnualReturn?: number; params?: T }
): Promise<CaseStudy<T>> {
    const { data } = await apiClient.put<CaseStudy<T>>(`${toolPath(toolType)}/${id}`, payload)
    return data
}

export async function deleteCase(toolType: string, id: number): Promise<void> {
    await apiClient.delete(`${toolPath(toolType)}/${id}`)
}
