import { apiClient } from '@/lib/api/client'

export interface RateRecord {
    id: number
    main: string
    secondary: string
    time: string
    rate: number
}

export interface CreateRateDTO {
    time: string
    rate: number
}

export interface UpdateRateDTO {
    time?: string
    rate?: number
}

const FX_PATH = '/fin/fx'

export async function getFXPairs(): Promise<string[]> {
    const { data } = await apiClient.get<{ pairs: string[] }>(`${FX_PATH}/pairs`)
    return data.pairs ?? []
}

/** Parse pair "MAIN/SECONDARY" into [main, secondary]. */
export function parsePair(pair: string): [string, string] {
    const i = pair.indexOf('/')
    if (i < 0) return [pair, '']
    return [pair.slice(0, i), pair.slice(i + 1)]
}

export async function getRateHistory(
    main: string,
    secondary: string,
    start?: string,
    end?: string
): Promise<RateRecord[]> {
    const params = new URLSearchParams()
    if (start) params.set('start', start)
    if (end) params.set('end', end)
    const qs = params.toString()
    const url = `${FX_PATH}/${encodeURIComponent(main)}/${encodeURIComponent(secondary)}/rates${qs ? `?${qs}` : ''}`
    const { data } = await apiClient.get<{ items: RateRecord[] }>(url)
    return data.items ?? []
}

export async function getLatestRate(main: string, secondary: string): Promise<RateRecord | null> {
    try {
        const { data } = await apiClient.get<RateRecord>(
            `${FX_PATH}/${encodeURIComponent(main)}/${encodeURIComponent(secondary)}/rates/latest`
        )
        return data
    } catch (err: unknown) {
        if (typeof err === 'object' && err !== null && 'response' in err) {
            const ax = err as { response?: { status?: number } }
            if (ax.response?.status === 404) return null
        }
        throw err
    }
}

export async function createRate(
    main: string,
    secondary: string,
    payload: CreateRateDTO
): Promise<void> {
    await apiClient.post(
        `${FX_PATH}/${encodeURIComponent(main)}/${encodeURIComponent(secondary)}/rates`,
        payload
    )
}

export async function createRatesBulk(
    main: string,
    secondary: string,
    payload: { points: CreateRateDTO[] }
): Promise<void> {
    await apiClient.post(
        `${FX_PATH}/${encodeURIComponent(main)}/${encodeURIComponent(secondary)}/rates/bulk`,
        payload
    )
}

export async function updateRate(id: number, payload: UpdateRateDTO): Promise<void> {
    await apiClient.put(`${FX_PATH}/rates/${id}`, payload)
}

export async function deleteRate(id: number): Promise<void> {
    await apiClient.delete(`${FX_PATH}/rates/${id}`)
}
