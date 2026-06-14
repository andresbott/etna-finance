import { describe, it, expect, vi, afterEach, type Mock } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { renderComposable, createTestQueryClient } from '../test/helpers'
import { useCsvProfileSave } from './useCsvProfileSave'
import { createProfile, updateProfile } from '../lib/api/CsvImport'
import { getProviders, updateAccount } from '../lib/api/Account'

vi.mock('../lib/api/CsvImport', () => ({
  createProfile: vi.fn(),
  updateProfile: vi.fn(),
}))

vi.mock('../lib/api/Account', () => ({
  getProviders: vi.fn(),
  createAccountProvider: vi.fn(),
  updateAccountProvider: vi.fn(),
  deleteAccountProvider: vi.fn(),
  createAccount: vi.fn(),
  updateAccount: vi.fn(),
  deleteAccount: vi.fn(),
}))

const payload = {
  name: 'Bank CSV',
  csvSeparator: ',',
  skipRows: 1,
  dateColumn: 'Date',
  dateFormat: '2006-01-02',
  descriptionColumn: 'Description',
  amountMode: 'single',
  amountColumn: 'Amount',
  creditColumn: '',
  debitColumn: '',
}

describe('useCsvProfileSave', () => {
  let unmount: () => void

  afterEach(() => {
    unmount?.()
    vi.clearAllMocks()
  })

  function setup() {
    ;(getProviders as Mock).mockResolvedValue([])
    const qc = createTestQueryClient()
    const { result, unmount: u } = renderComposable(() => useCsvProfileSave(), { queryClient: qc })
    unmount = u
    return result
  }

  it('create mode: creates the profile then binds it to the account', async () => {
    ;(createProfile as Mock).mockResolvedValue({ id: 7, ...payload })
    ;(updateAccount as Mock).mockResolvedValue({})
    const result = setup()

    await result.saveProfile({ accountId: 42, profileId: 0, payload })
    await flushPromises()

    expect(createProfile).toHaveBeenCalledWith(payload)
    expect(updateAccount).toHaveBeenCalledWith(42, { importProfileId: 7 })
    expect(updateProfile).not.toHaveBeenCalled()
  })

  it('edit mode: updates the existing profile and does not re-bind', async () => {
    ;(updateProfile as Mock).mockResolvedValue({ id: 7, ...payload })
    const result = setup()

    await result.saveProfile({ accountId: 42, profileId: 7, payload })
    await flushPromises()

    expect(updateProfile).toHaveBeenCalledWith(7, payload)
    expect(createProfile).not.toHaveBeenCalled()
    expect(updateAccount).not.toHaveBeenCalled()
  })
})
