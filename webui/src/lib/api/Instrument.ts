import axios from 'axios'
import { apiClient } from '@/lib/api/client'
import type { Instrument, CreateInstrumentDTO, UpdateInstrumentDTO, InstrumentLookup } from '@/types/instrument'

const INSTRUMENT_PATH = '/fin/instrument'

// Thrown when the reference provider rate-limits a lookup (HTTP 429), so callers can
// distinguish "no match" (null) from "try again later".
export class LookupRateLimitError extends Error {
    retryAfterSeconds?: number
    constructor(retryAfterSeconds?: number) {
        super('rate limited by reference provider')
        this.name = 'LookupRateLimitError'
        this.retryAfterSeconds = retryAfterSeconds
    }
}

export const getInstruments = async (): Promise<Instrument[]> => {
    const { data } = await apiClient.get<{ items: Instrument[] }>(INSTRUMENT_PATH)
    return data.items
}

export const createInstrument = async (payload: CreateInstrumentDTO): Promise<Instrument> => {
    const { data } = await apiClient.post<Instrument>(INSTRUMENT_PATH, payload)
    return data
}

export const updateInstrument = async (
    id: number,
    payload: UpdateInstrumentDTO
): Promise<void> => {
    await apiClient.put(`${INSTRUMENT_PATH}/${id}`, payload)
}

export const deleteInstrument = async (id: number): Promise<void> => {
    await apiClient.delete(`${INSTRUMENT_PATH}/${id}`)
}

export const lookupInstrument = async (symbol: string): Promise<InstrumentLookup | null> => {
    try {
        const res = await apiClient.get<InstrumentLookup>(`${INSTRUMENT_PATH}/lookup`, {
            params: { symbol }
        })
        if (res.status === 204 || !res.data) {
            return null
        }
        return res.data
    } catch (e) {
        if (axios.isAxiosError(e) && e.response?.status === 429) {
            const retryAfter = Number(e.response.headers['retry-after'])
            throw new LookupRateLimitError(Number.isFinite(retryAfter) ? retryAfter : undefined)
        }
        throw e
    }
}
