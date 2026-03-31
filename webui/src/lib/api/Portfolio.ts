import { apiClient } from '@/lib/api/client'

export interface Position {
    id: number
    accountId: number
    instrumentId: number
    quantity: number
    costBasis: number
    avgCost: number
}

export interface Lot {
    id: number
    tradeId: number
    accountId: number
    instrumentId: number
    openDate: string
    quantity: number
    originalQty: number
    costPerShare: number
    costBasis: number
    status: number
    closedDate?: string
}

export interface Trade {
    id: number
    transactionId: number
    accountId: number
    instrumentId: number
    tradeType: number
    quantity: number
    pricePerShare: number
    totalAmount: number
    currency: string
    date: string
}

export interface PositionDetail {
    position: Position
    lots: Lot[]
}

export const getPositions = async (accountId?: number): Promise<Position[]> => {
    const params = new URLSearchParams()
    if (accountId) params.set('accountId', String(accountId))
    const { data } = await apiClient.get(`/fin/portfolio/positions?${params}`)
    return data.items ?? []
}

export const getPositionDetail = async (
    accountId: number,
    instrumentId: number
): Promise<PositionDetail> => {
    const { data } = await apiClient.get(
        `/fin/portfolio/positions/${accountId}/${instrumentId}`
    )
    return data
}

export const getLots = async (
    accountId?: number,
    instrumentId?: number,
    beforeDate?: string
): Promise<Lot[]> => {
    const params = new URLSearchParams()
    if (accountId) params.set('accountId', String(accountId))
    if (instrumentId) params.set('instrumentId', String(instrumentId))
    if (beforeDate) params.set('beforeDate', beforeDate)
    const { data } = await apiClient.get(`/fin/portfolio/lots?${params}`)
    return data.items ?? []
}

export interface InstrumentReturn {
    instrumentId: number
    totalInvested: number
    realizedProceeds: number
    realizedGL: number
    currentQuantity: number
    currentCostBasis: number
    firstTradeDate: string
    lastTradeDate: string
}

export const getInstrumentReturns = async (): Promise<InstrumentReturn[]> => {
    const { data } = await apiClient.get('/fin/portfolio/returns')
    return data.items ?? []
}

export const getTrades = async (
    accountId?: number,
    instrumentId?: number
): Promise<Trade[]> => {
    const params = new URLSearchParams()
    if (accountId) params.set('accountId', String(accountId))
    if (instrumentId) params.set('instrumentId', String(instrumentId))
    const { data } = await apiClient.get(`/fin/portfolio/trades?${params}`)
    return data.items ?? []
}
