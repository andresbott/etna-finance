import { apiClient } from '@/lib/api/client'

const PROVIDER_PATH = '/fin/provider'
const ACCOUNT_PATH = '/fin/account'

export interface ProviderItem {
    id: number
    name: string
    description?: string
    icon?: string
    accounts?: AccountItem[]
}

export interface AccountItem {
    id: number
    name: string
    currency: string
    type: string
    icon?: string
    notes?: string
    providerId?: number
    importProfileId?: number
    favorite?: boolean
}

export async function getProviders(): Promise<ProviderItem[]> {
    const { data } = await apiClient.get<{ items: ProviderItem[] }>(PROVIDER_PATH)
    return data.items ?? []
}

export async function createAccountProvider(payload: Partial<ProviderItem>): Promise<ProviderItem> {
    const { data } = await apiClient.post<ProviderItem>(PROVIDER_PATH, payload)
    return data
}

export async function updateAccountProvider(id: number, payload: Partial<ProviderItem>): Promise<ProviderItem> {
    const { data } = await apiClient.put<ProviderItem>(`${PROVIDER_PATH}/${id}`, payload)
    return data
}

export async function deleteAccountProvider(id: number): Promise<void> {
    await apiClient.delete(`${PROVIDER_PATH}/${id}`)
}

export async function createAccount(payload: Partial<AccountItem>): Promise<AccountItem> {
    const { data } = await apiClient.post<AccountItem>(ACCOUNT_PATH, payload)
    return data
}

export async function updateAccount(id: number, payload: Partial<AccountItem>): Promise<AccountItem> {
    const { data } = await apiClient.put<AccountItem>(`${ACCOUNT_PATH}/${id}`, payload)
    return data
}

export async function deleteAccount(id: number): Promise<void> {
    await apiClient.delete(`${ACCOUNT_PATH}/${id}`)
}
