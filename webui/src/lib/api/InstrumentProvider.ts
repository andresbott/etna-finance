import { apiClient } from '@/lib/api/client'
import type {
    InstrumentProvider,
    CreateInstrumentProviderDTO,
    UpdateInstrumentProviderDTO
} from '@/types/security'

const INSTRUMENT_PROVIDER_PATH = '/fin/instrument-provider'

export const getInstrumentProviders = async (): Promise<InstrumentProvider[]> => {
    const { data } = await apiClient.get<{ items: InstrumentProvider[] }>(INSTRUMENT_PROVIDER_PATH)
    return data.items ?? []
}

export const createInstrumentProvider = async (
    payload: CreateInstrumentProviderDTO
): Promise<InstrumentProvider> => {
    const { data } = await apiClient.post<InstrumentProvider>(INSTRUMENT_PROVIDER_PATH, payload)
    return data
}

export const updateInstrumentProvider = async (
    id: number,
    payload: UpdateInstrumentProviderDTO
): Promise<void> => {
    await apiClient.put(`${INSTRUMENT_PROVIDER_PATH}/${id}`, payload)
}

export const deleteInstrumentProvider = async (id: number): Promise<void> => {
    await apiClient.delete(`${INSTRUMENT_PROVIDER_PATH}/${id}`)
}
