import { apiClient } from '@/lib/api/client'

export interface CaseStudy<T = Record<string, unknown>> {
    id: number
    toolType: string
    name: string
    description: string
    expectedAnnualReturn: number
    params: T
    attachmentId?: number
    createdAt: string
    updatedAt: string
}

export interface PortfolioSimulatorParams {
    initialContribution: number
    growthRatePct: number
    expenseRatioPct: number
    capitalGainTaxPct: number
    taxModel: 'exit' | 'annual'
    // Deprecated — ignored if present in saved data
    monthlyContribution?: number
    durationYears?: number
    inflationPct?: number
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
    incidentalPct?: number
    cashEquity: number
    additionalEquity: Array<{ name: string; amount: number }>
    mortgages: Array<{
        name: string
        splitPct: number
        interestRate: number
        termYears: number
        amortize: boolean
    }>
    grossAnnualIncome: number
    housingPriceIncreasePct?: number
}

export interface BuyVsRentSimulatorParams {
    purchasePrice: number
    cashEquity: number
    additionalEquity: Array<{ name: string; amount: number }>
    mortgages: Array<{
        name: string
        splitPct: number
        interestRate: number
        termYears: number
        amortize: boolean
    }>
    propertyTax: number
    insurance: number
    maintenance: number
    otherCosts: number
    incidentalPct?: number
    housingPriceIncreasePct?: number
    currentMonthlyRent: number
    rentIncreasePct?: number
    etfReturnPct?: number
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

export async function uploadCaseAttachment(toolType: string, id: number, file: File): Promise<{ id: number; originalName: string; mimeType: string; fileSize: number }> {
    const formData = new FormData()
    formData.append('file', file)
    const { data } = await apiClient.post(`${toolPath(toolType)}/${id}/attachment`, formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
    })
    return data
}

export function getCaseAttachmentUrl(toolType: string, id: number): string {
    return `${apiClient.defaults.baseURL}${toolPath(toolType)}/${id}/attachment`
}

export async function deleteCaseAttachment(toolType: string, id: number): Promise<void> {
    await apiClient.delete(`${toolPath(toolType)}/${id}/attachment`)
}
