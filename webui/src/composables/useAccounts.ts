import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import AccountProvider from '@/models/AccountProvider'
import Account from '@/models/Account'
import type { ProviderItem, AccountItem } from '@/lib/api/Account'
import {
    getProviders,
    createAccountProvider as createProviderApi,
    updateAccountProvider as updateProviderApi,
    deleteAccountProvider as deleteProviderApi,
    createAccount as createAccountApi,
    updateAccount as updateAccountApi,
    deleteAccount as deleteAccountApi
} from '@/lib/api/Account'
import { invalidateAndRefetch } from '@/composables/queryUtils'

function transformProviders(data: ProviderItem[]): AccountProvider[] {
    return data.map(
        (item) =>
            new AccountProvider({
                id: item.id,
                name: item.name,
                description: item.description,
                icon: item.icon,
                accounts: (item.accounts ?? []).map(
                    (acc: AccountItem) =>
                        new Account({
                            id: acc.id,
                            name: acc.name,
                            currency: acc.currency,
                            type: acc.type,
                            icon: acc.icon
                        })
                )
            })
    )
}

export function useAccounts() {
    const queryClient = useQueryClient()
    const QUERY_KEY = ['accounts']

    const doInvalidateAndRefetch = () => invalidateAndRefetch(queryClient, QUERY_KEY)

    const accountsQuery = useQuery({
        queryKey: QUERY_KEY,
        queryFn: getProviders,
        select: transformProviders
    })

    const createAccountProviderMutation = useMutation({
        mutationFn: createProviderApi,
        onSuccess: doInvalidateAndRefetch
    })

    const updateAccountProviderMutation = useMutation({
        mutationFn: (data: { id: number; name?: string; description?: string; icon?: string }) =>
            updateProviderApi(data.id, { name: data.name, description: data.description, icon: data.icon }),
        onSuccess: doInvalidateAndRefetch
    })

    const deleteAccountProviderMutation = useMutation({
        mutationFn: deleteProviderApi,
        onSuccess: doInvalidateAndRefetch
    })

    const createAccountMutation = useMutation({
        mutationFn: createAccountApi,
        onSuccess: doInvalidateAndRefetch
    })

    const updateAccountMutation = useMutation({
        mutationFn: (data: { id: number; name?: string; currency?: string; type?: string; icon?: string }) =>
            updateAccountApi(data.id, { name: data.name, currency: data.currency, type: data.type, icon: data.icon }),
        onSuccess: doInvalidateAndRefetch
    })

    const deleteAccountMutation = useMutation({
        mutationFn: deleteAccountApi,
        onSuccess: doInvalidateAndRefetch
    })

    return {
        accounts: accountsQuery.data,
        isLoading: accountsQuery.isLoading,
        isError: accountsQuery.isError,
        error: accountsQuery.error,
        refetch: accountsQuery.refetch,

        createAccountProvider: createAccountProviderMutation.mutateAsync,
        updateAccountProvider: updateAccountProviderMutation.mutateAsync,
        deleteAccountProvider: deleteAccountProviderMutation.mutateAsync,

        createAccount: createAccountMutation.mutateAsync,
        updateAccount: updateAccountMutation.mutateAsync,
        deleteAccount: deleteAccountMutation.mutateAsync,

        isCreating: createAccountMutation.isPending,
        isUpdating: updateAccountMutation.isPending,
        isDeleting: deleteAccountMutation.isPending || deleteAccountProviderMutation.isPending
    }
}
