import { useAccounts } from '@/composables/useAccounts'
import { createProfile, updateProfile } from '@/lib/api/CsvImport'
import type { ImportProfile } from '@/types/csvimport'

// The form/dialog produces the profile fields without the server-assigned id.
// `amountMode` is kept as a plain string here because the field originates from
// untyped form/select inputs; it is narrowed to ImportProfile['amountMode'] at
// the API boundary below.
export type ProfileFormPayload = Omit<ImportProfile, 'id' | 'amountMode'> & {
  amountMode: string
}

export interface SaveProfileArgs {
  accountId: number
  profileId: number // 0 => create mode
  payload: ProfileFormPayload
}

export function useCsvProfileSave() {
  const { updateAccount } = useAccounts()

  async function saveProfile({ accountId, profileId, payload }: SaveProfileArgs) {
    const profile = payload as Omit<ImportProfile, 'id'>
    if (profileId > 0) {
      await updateProfile(profileId, profile)
      return
    }
    const created = await createProfile(profile)
    await updateAccount({ id: accountId, importProfileId: created.id })
  }

  return { saveProfile }
}
