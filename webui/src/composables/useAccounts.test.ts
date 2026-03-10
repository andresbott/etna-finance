import { describe, it, expect, vi, afterEach, type Mock } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { renderComposable, createTestQueryClient } from '../test/helpers'
import { useAccounts } from './useAccounts'
import {
  getProviders,
  createAccountProvider,
  updateAccountProvider,
  deleteAccountProvider,
  createAccount,
  updateAccount,
  deleteAccount,
} from '../lib/api/Account'
import type { ProviderItem } from '../lib/api/Account'

vi.mock('../lib/api/Account', () => ({
  getProviders: vi.fn(),
  createAccountProvider: vi.fn(),
  updateAccountProvider: vi.fn(),
  deleteAccountProvider: vi.fn(),
  createAccount: vi.fn(),
  updateAccount: vi.fn(),
  deleteAccount: vi.fn(),
}))

const mockGetProviders = getProviders as Mock

describe('useAccounts', () => {
  let unmount: () => void

  afterEach(() => {
    unmount?.()
    vi.clearAllMocks()
  })

  function setup(providerData?: ProviderItem[]) {
    if (providerData !== undefined) {
      mockGetProviders.mockResolvedValue(providerData)
    }
    const qc = createTestQueryClient()
    const { result, unmount: u } = renderComposable(() => useAccounts(), { queryClient: qc })
    unmount = u
    return { result, queryClient: qc }
  }

  describe('query', () => {
    it('starts in loading state', () => {
      mockGetProviders.mockReturnValue(new Promise(() => {})) // never resolves
      const { result } = setup()

      expect(result.isLoading.value).toBe(true)
      expect(result.accounts.value).toBeUndefined()
    })

    it('returns providers after fetch', async () => {
      const providers: ProviderItem[] = [
        { id: 1, name: 'Bank A', accounts: [{ id: 10, name: 'Checking', currency: 'USD', type: 'checkin' }] },
      ]
      const { result } = setup(providers)

      await flushPromises()

      expect(result.isLoading.value).toBe(false)
      expect(result.accounts.value).toHaveLength(1)
      expect(result.accounts.value![0].id).toBe(1)
      expect(result.accounts.value![0].accounts).toHaveLength(1)
      expect(result.accounts.value![0].accounts[0].name).toBe('Checking')
    })

    it('normalizes null accounts to empty array', async () => {
      const providers: ProviderItem[] = [
        { id: 1, name: 'Bank B', accounts: undefined },
        { id: 2, name: 'Bank C' },
      ]
      const { result } = setup(providers)

      await flushPromises()

      expect(result.accounts.value).toHaveLength(2)
      expect(result.accounts.value![0].accounts).toEqual([])
      expect(result.accounts.value![1].accounts).toEqual([])
    })

    it('handles empty provider array', async () => {
      const { result } = setup([])

      await flushPromises()

      expect(result.isLoading.value).toBe(false)
      expect(result.accounts.value).toEqual([])
    })

    it('sets isError on fetch failure', async () => {
      mockGetProviders.mockRejectedValue(new Error('Network error'))
      const qc = createTestQueryClient()
      const { result, unmount: u } = renderComposable(() => useAccounts(), { queryClient: qc })
      unmount = u

      await flushPromises()

      expect(result.isError.value).toBe(true)
      expect(result.error.value).toBeInstanceOf(Error)
    })
  })

  describe('mutations', () => {
    it('exposes createAccountProvider that calls the API', async () => {
      const newProvider = { id: 3, name: 'New Bank' }
      ;(createAccountProvider as Mock).mockResolvedValue(newProvider)
      const { result } = setup([])

      await flushPromises()

      const returned = await result.createAccountProvider({ name: 'New Bank' })
      expect(createAccountProvider).toHaveBeenCalled()
      expect((createAccountProvider as Mock).mock.calls[0][0]).toEqual({ name: 'New Bank' })
      expect(returned).toEqual(newProvider)
    })

    it('exposes updateAccountProvider that calls the API with id and payload', async () => {
      const updated = { id: 1, name: 'Updated' }
      ;(updateAccountProvider as Mock).mockResolvedValue(updated)
      const { result } = setup([])

      await flushPromises()

      await result.updateAccountProvider({ id: 1, name: 'Updated', description: 'desc', icon: 'pi-star' })
      expect(updateAccountProvider).toHaveBeenCalledWith(1, { name: 'Updated', description: 'desc', icon: 'pi-star' })
    })

    it('exposes deleteAccountProvider that calls the API', async () => {
      ;(deleteAccountProvider as Mock).mockResolvedValue(undefined)
      const { result } = setup([])

      await flushPromises()

      await result.deleteAccountProvider(5)
      expect(deleteAccountProvider).toHaveBeenCalled()
      expect((deleteAccountProvider as Mock).mock.calls[0][0]).toBe(5)
    })

    it('exposes createAccount that calls the API', async () => {
      const newAccount = { id: 20, name: 'Savings', currency: 'EUR', type: 'savings' }
      ;(createAccount as Mock).mockResolvedValue(newAccount)
      const { result } = setup([])

      await flushPromises()

      const returned = await result.createAccount({ name: 'Savings', currency: 'EUR', type: 'savings' })
      expect(createAccount).toHaveBeenCalled()
      expect((createAccount as Mock).mock.calls[0][0]).toEqual({ name: 'Savings', currency: 'EUR', type: 'savings' })
      expect(returned).toEqual(newAccount)
    })

    it('exposes updateAccount that calls the API with id and payload', async () => {
      ;(updateAccount as Mock).mockResolvedValue({ id: 10, name: 'Renamed' })
      const { result } = setup([])

      await flushPromises()

      await result.updateAccount({ id: 10, name: 'Renamed', currency: 'USD', type: 'checkin', icon: 'pi-cc', importProfileId: 7 })
      expect(updateAccount).toHaveBeenCalledWith(10, { name: 'Renamed', currency: 'USD', type: 'checkin', icon: 'pi-cc', importProfileId: 7 })
    })

    it('exposes deleteAccount that calls the API', async () => {
      ;(deleteAccount as Mock).mockResolvedValue(undefined)
      const { result } = setup([])

      await flushPromises()

      await result.deleteAccount(10)
      expect(deleteAccount).toHaveBeenCalled()
      expect((deleteAccount as Mock).mock.calls[0][0]).toBe(10)
    })
  })

  describe('mutation pending states', () => {
    it('isCreating reflects createAccount pending state', () => {
      const { result } = setup([])
      // Before any mutation, isPending should be false
      expect(result.isCreating.value).toBe(false)
    })

    it('isUpdating reflects updateAccount pending state', () => {
      const { result } = setup([])
      expect(result.isUpdating.value).toBe(false)
    })

    it('isDeleting reflects delete pending state', () => {
      const { result } = setup([])
      expect(result.isDeleting.value).toBe(false)
    })
  })

  describe('refetch', () => {
    it('exposes refetch function that re-calls getProviders', async () => {
      const { result } = setup([{ id: 1, name: 'Bank' }])

      await flushPromises()
      expect(mockGetProviders).toHaveBeenCalledTimes(1)

      mockGetProviders.mockResolvedValue([{ id: 1, name: 'Bank' }, { id: 2, name: 'Bank 2' }])
      await result.refetch()
      await flushPromises()

      expect(mockGetProviders).toHaveBeenCalledTimes(2)
      expect(result.accounts.value).toHaveLength(2)
    })
  })
})
