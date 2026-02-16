import { apiClient } from '@/lib/api/client'
import type { Instrument, CreateInstrumentDTO, UpdateInstrumentDTO } from '@/types/instrument'

const INSTRUMENT_PATH = '/fin/instrument'

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
