import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import axios from 'axios'
import AccountProvider from '@/models/AccountProvider'
import Account from '@/models/Account'

const API_BASE_URL = import.meta.env.VITE_SERVER_URL_V0
const ACCOUNTS_ENDPOINT = `${API_BASE_URL}/fin/account`
const PROVIDER_ENDPOINT = `${API_BASE_URL}/fin/provider`

/**
 * Fetches all accounts from the API
 * @returns {Promise<Account[]>}
 */
const fetchProviders = async () => {
    const { data } = await axios.get(PROVIDER_ENDPOINT)
    return data.items || []
}

const createAccountProvider = async (accountData) => {
    const { data } = await axios.post(PROVIDER_ENDPOINT, accountData)
    return data
}
const updateAccountProvider = async (accountData) => {
    const { data } = await axios.put(`${PROVIDER_ENDPOINT}/${accountData.id}`, accountData)
    return data
}

/**
 * Deletes an account provider
 * @param {number} id - Account Provider ID
 * @returns {Promise<void>}
 */
const deleteAccountProvider = async (id) => {
    await axios.delete(`${PROVIDER_ENDPOINT}/${id}`)
}

/**
 * Creates a new account
 * @param {CreateAccountDTO} accountData
 * @returns {Promise<Account>}
 */
const createAccount = async (accountData) => {
    const { data } = await axios.post(ACCOUNTS_ENDPOINT, accountData)
    return data
}

/**
 * Updates an existing account
 * @param {UpdateAccountDTO} accountData
 * @returns {Promise<Account>}
 */
const updateAccount = async (accountData) => {
    const { data } = await axios.put(`${ACCOUNTS_ENDPOINT}/${accountData.id}`, accountData)
    return data
}

/**
 * Deletes an account
 * @param {string} id - Account ID
 * @returns {Promise<void>}
 */
const deleteAccount = async (id) => {
    await axios.delete(`${ACCOUNTS_ENDPOINT}/${id}`)
}

export function useAccounts() {
    const queryClient = useQueryClient()
    const QUERY_KEY = ['accounts']

    const accountsQuery = useQuery({
        queryKey: QUERY_KEY,
        queryFn: fetchProviders,
        select: (data) => data.map(item => new AccountProvider({
            id: item.id,
            name: item.name,
            description: item.description,
            accounts: item.accounts.map(acc => new Account({
                id: acc.id,
                name: acc.name,
                currency: acc.currency,
                type: acc.type
            }))
        }))
    })



    const createAccountProviderMutation = useMutation({
        mutationFn: createAccountProvider,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: QUERY_KEY })
            queryClient.refetchQueries({ queryKey: QUERY_KEY })
        }
    })

    const updateAccountProviderMutation = useMutation({
        mutationFn: updateAccountProvider,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: QUERY_KEY })
            queryClient.refetchQueries({ queryKey: QUERY_KEY })
        }
    })

    // Mutation for deleting an account provider
    const deleteAccountProviderMutation = useMutation({
        mutationFn: deleteAccountProvider,
        onSuccess: (_, deletedId) => {
            queryClient.setQueryData(QUERY_KEY, (oldProviders = []) =>
                oldProviders.filter((provider) => provider.id !== deletedId)
            )
        }
    })

    // Mutation for creating an account
    const createAccountMutation = useMutation({
        mutationFn: createAccount,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: QUERY_KEY })
            queryClient.refetchQueries({ queryKey: QUERY_KEY })
        }
    })

    // Mutation for updating an account
    const updateAccountMutation = useMutation({
        mutationFn: updateAccount,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: QUERY_KEY })
            queryClient.refetchQueries({ queryKey: QUERY_KEY })
        }
    })

    // Mutation for deleting an account
    const deleteAccountMutation = useMutation({
        mutationFn: deleteAccount,
        onSuccess: (_, deletedId) => {
            queryClient.setQueryData(QUERY_KEY, (oldAccounts = []) =>
                oldAccounts.filter((account) => account.id !== deletedId)
            )
        }
    })


    return {
        // Queries
        accounts: accountsQuery.data,
        isLoading: accountsQuery.isLoading,
        isError: accountsQuery.isError,
        error: accountsQuery.error,
        refetch: accountsQuery.refetch,


        // Mutations
        createAccountProvider: createAccountProviderMutation.mutateAsync,
        updateAccountProvider: updateAccountProviderMutation.mutateAsync,
        deleteAccountProvider: deleteAccountProviderMutation.mutateAsync,

        createAccount: createAccountMutation.mutateAsync,
        updateAccount: updateAccountMutation.mutateAsync,
        deleteAccount: deleteAccountMutation.mutateAsync,

        // Mutation states
        isCreating: createAccountMutation.isLoading,
        isUpdating: updateAccountMutation.isLoading,
        isDeleting: deleteAccountMutation.isLoading || deleteAccountProviderMutation.isLoading
    }
}
