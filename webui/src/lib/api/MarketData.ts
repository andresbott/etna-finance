import { apiClient } from '@/lib/api/client'
import { with404Null } from '@/lib/api/helpers'

export interface PriceRecord {
    id: number
    symbol: string
    time: string
    price: number
}

export interface CreatePriceDTO {
    time: string
    price: number
}

export interface UpdatePriceDTO {
    time?: string
    price?: number
}

const MARKET_DATA_PATH = '/fin/marketdata'

export async function getMarketDataSymbols(): Promise<string[]> {
    const { data } = await apiClient.get<{ symbols: string[] }>(`${MARKET_DATA_PATH}/symbols`)
    return data.symbols ?? []
}

export async function getPriceHistory(
    symbol: string,
    start?: string,
    end?: string
): Promise<PriceRecord[]> {
    const params = new URLSearchParams()
    if (start) params.set('start', start)
    if (end) params.set('end', end)
    const qs = params.toString()
    const url = `${MARKET_DATA_PATH}/${encodeURIComponent(symbol)}/prices${qs ? `?${qs}` : ''}`
    const { data } = await apiClient.get<{ items: PriceRecord[] }>(url)
    return data.items ?? []
}

export async function getLatestPrice(symbol: string): Promise<PriceRecord | null> {
    return with404Null(async () => {
        const { data } = await apiClient.get<PriceRecord>(
            `${MARKET_DATA_PATH}/${encodeURIComponent(symbol)}/prices/latest`
        )
        return data
    })
}

export async function createPrice(
    symbol: string,
    payload: CreatePriceDTO
): Promise<void> {
    await apiClient.post(
        `${MARKET_DATA_PATH}/${encodeURIComponent(symbol)}/prices`,
        payload
    )
}

export async function createPricesBulk(
    symbol: string,
    payload: { points: CreatePriceDTO[] }
): Promise<void> {
    await apiClient.post(
        `${MARKET_DATA_PATH}/${encodeURIComponent(symbol)}/prices/bulk`,
        payload
    )
}

export async function updatePrice(
    id: number,
    payload: UpdatePriceDTO
): Promise<void> {
    await apiClient.put(`${MARKET_DATA_PATH}/prices/${id}`, payload)
}

export async function deletePrice(id: number): Promise<void> {
    await apiClient.delete(`${MARKET_DATA_PATH}/prices/${id}`)
}
