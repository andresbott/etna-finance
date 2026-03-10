import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { apiClient } from './client'
import {
    getProviders,
    createAccountProvider,
    updateAccountProvider,
    deleteAccountProvider,
    createAccount,
    updateAccount,
    deleteAccount,
    type ProviderItem,
    type AccountItem,
} from './Account'

vi.mock('./client', () => ({
    apiClient: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}))

beforeEach(() => vi.clearAllMocks())

const mockProvider: ProviderItem = {
    id: 1,
    name: 'Test Bank',
    description: 'A test provider',
    icon: 'bank-icon',
}

const mockAccount: AccountItem = {
    id: 10,
    name: 'Checking',
    currency: 'USD',
    type: 'checking',
    icon: 'check-icon',
    importProfileId: 5,
}

describe('getProviders', () => {
    it('calls GET /fin/provider and returns items', async () => {
        const items = [mockProvider];
        (apiClient.get as Mock).mockResolvedValue({ data: { items } })

        const result = await getProviders()

        expect(apiClient.get).toHaveBeenCalledWith('/fin/provider')
        expect(apiClient.get).toHaveBeenCalledTimes(1)
        expect(result).toEqual(items)
    })

    it('returns empty array when items is undefined', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: {} })

        const result = await getProviders()

        expect(result).toEqual([])
    })

    it('returns empty array when items is null', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: { items: null } })

        const result = await getProviders()

        expect(result).toEqual([])
    })
})

describe('createAccountProvider', () => {
    it('calls POST /fin/provider with payload and returns created provider', async () => {
        const payload: Partial<ProviderItem> = { name: 'New Bank', description: 'desc' };
        (apiClient.post as Mock).mockResolvedValue({ data: mockProvider })

        const result = await createAccountProvider(payload)

        expect(apiClient.post).toHaveBeenCalledWith('/fin/provider', payload)
        expect(apiClient.post).toHaveBeenCalledTimes(1)
        expect(result).toEqual(mockProvider)
    })
})

describe('updateAccountProvider', () => {
    it('calls PUT /fin/provider/:id with payload and returns updated provider', async () => {
        const payload: Partial<ProviderItem> = { name: 'Updated Bank' };
        (apiClient.put as Mock).mockResolvedValue({ data: { ...mockProvider, ...payload } })

        const result = await updateAccountProvider(1, payload)

        expect(apiClient.put).toHaveBeenCalledWith('/fin/provider/1', payload)
        expect(apiClient.put).toHaveBeenCalledTimes(1)
        expect(result).toEqual({ ...mockProvider, ...payload })
    })
})

describe('deleteAccountProvider', () => {
    it('calls DELETE /fin/provider/:id', async () => {
        (apiClient.delete as Mock).mockResolvedValue({})

        await deleteAccountProvider(1)

        expect(apiClient.delete).toHaveBeenCalledWith('/fin/provider/1')
        expect(apiClient.delete).toHaveBeenCalledTimes(1)
    })

    it('returns void', async () => {
        (apiClient.delete as Mock).mockResolvedValue({})

        const result = await deleteAccountProvider(1)

        expect(result).toBeUndefined()
    })
})

describe('createAccount', () => {
    it('calls POST /fin/account with payload and returns created account', async () => {
        const payload: Partial<AccountItem> = { name: 'Savings', currency: 'EUR', type: 'savings' };
        (apiClient.post as Mock).mockResolvedValue({ data: mockAccount })

        const result = await createAccount(payload)

        expect(apiClient.post).toHaveBeenCalledWith('/fin/account', payload)
        expect(apiClient.post).toHaveBeenCalledTimes(1)
        expect(result).toEqual(mockAccount)
    })
})

describe('updateAccount', () => {
    it('calls PUT /fin/account/:id with payload and returns updated account', async () => {
        const payload: Partial<AccountItem> = { name: 'Updated Checking' };
        (apiClient.put as Mock).mockResolvedValue({ data: { ...mockAccount, ...payload } })

        const result = await updateAccount(10, payload)

        expect(apiClient.put).toHaveBeenCalledWith('/fin/account/10', payload)
        expect(apiClient.put).toHaveBeenCalledTimes(1)
        expect(result).toEqual({ ...mockAccount, ...payload })
    })
})

describe('deleteAccount', () => {
    it('calls DELETE /fin/account/:id', async () => {
        (apiClient.delete as Mock).mockResolvedValue({})

        await deleteAccount(10)

        expect(apiClient.delete).toHaveBeenCalledWith('/fin/account/10')
        expect(apiClient.delete).toHaveBeenCalledTimes(1)
    })

    it('returns void', async () => {
        (apiClient.delete as Mock).mockResolvedValue({})

        const result = await deleteAccount(10)

        expect(result).toBeUndefined()
    })
})
