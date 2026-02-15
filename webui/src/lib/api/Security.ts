import { apiClient } from '@/lib/api/client'
import type { Security, CreateSecurityDTO, UpdateSecurityDTO } from '@/types/security'

const SECURITY_PATH = '/fin/security'

export const getSecurities = async (): Promise<Security[]> => {
    const { data } = await apiClient.get<{ items: Security[] }>(SECURITY_PATH)
    return data.items ?? []
}

export const createSecurity = async (payload: CreateSecurityDTO): Promise<Security> => {
    const { data } = await apiClient.post<Security>(SECURITY_PATH, payload)
    return data
}

export const updateSecurity = async (
    id: number,
    payload: UpdateSecurityDTO
): Promise<void> => {
    await apiClient.put(`${SECURITY_PATH}/${id}`, payload)
}

export const deleteSecurity = async (id: number): Promise<void> => {
    await apiClient.delete(`${SECURITY_PATH}/${id}`)
}
