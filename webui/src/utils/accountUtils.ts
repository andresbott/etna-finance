import { useAccounts } from '@/composables/useAccounts'
import type Account from '@/models/Account'
import type AccountProvider from '@/models/AccountProvider'

export interface AccountData {
    id: string | number
    name: string
    currency: string
    type: string
}

export function findAccountById(providers: AccountProvider[], id: number | string): Account | null {
    for (const provider of providers) {
        if (provider.accounts) {
            for (const account of provider.accounts) {
                if (account.id === String(id) || account.id === id) {
                    return account
                }
            }
        }
    }
    return null
}

export function useAccountUtils() {
    const { accounts } = useAccounts()

    const getAccountName = (id: number | string): string => {
        if (!id || id === 0) return 'Unknown Account'
        if (!accounts?.value) return 'Loading...'

        const account = findAccountById(accounts.value, id)
        return account ? account.name : 'Unknown Account'
    }

    const getAccountCurrency = (id: number | string): string => {
        if (!id || id === 0) return 'Unknown'
        if (!accounts?.value) return 'Loading...'

        const account = findAccountById(accounts.value, id)
        return account ? account.currency : 'Unknown'
    }

    const getAccountType = (id: number | string): string => {
        if (!id || id === 0) return 'Unknown'
        if (!accounts?.value) return 'Loading...'

        const account = findAccountById(accounts.value, id)
        return account ? account.type : 'Unknown'
    }

    const getAccount = (id: number | string): Account | null => {
        if (!id || id === 0) return null
        if (!accounts?.value) return null

        return findAccountById(accounts.value, id)
    }

    return { 
        getAccountName, 
        getAccountCurrency, 
        getAccountType, 
        getAccount 
    }
}
