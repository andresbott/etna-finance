import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_SERVER_URL_V0
const ACCOUNTS_ENDPOINT = `${API_BASE_URL}/fin/accounts`

/**
 * Fetches all accounts from the API
 * @returns {Promise<Account[]>}
 */
const fetchAccounts = async () => {
    const { data } = await axios.get(ACCOUNTS_ENDPOINT)
    return data.items || []
}

/**
 * Fetches a single account by ID
 * @param {string} id - Account ID
 * @returns {Promise<Account>}
 */
const fetchAccountById = async (id) => {
    const { data } = await axios.get(`${ACCOUNTS_ENDPOINT}/${id}`)
    return data
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
        queryFn: fetchAccounts
    })

    // Query for fetching a single account
    const useAccountById = (id) => {
        return useQuery({
            queryKey: [...QUERY_KEY, id],
            queryFn: () => fetchAccountById(id),
            enabled: !!id
        })
    }

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
        useAccountById,

        // Mutations
        createAccount: createAccountMutation.mutateAsync,
        updateAccount: updateAccountMutation.mutateAsync,
        deleteAccount: deleteAccountMutation.mutateAsync,

        // Mutation states
        isCreating: createAccountMutation.isLoading,
        isUpdating: updateAccountMutation.isLoading,
        isDeleting: deleteAccountMutation.isLoading
    }
}
