import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_SERVER_URL_V0
const ENTRIES_ENDPOINT = `${API_BASE_URL}/fin/entries`

/**
 * Fetches all entries from the API
 * @returns {Promise<Entry[]>}
 */
const fetchEntries = async () => {
    const { data } = await axios.get(ENTRIES_ENDPOINT)
    return data.items || []
}


/**
 * Creates a new entry
 * @param {CreateEntryDTO} entryData
 * @returns {Promise<Entry>}
 */
const createEntry = async (entryData) => {
    const { data } = await axios.post(ENTRIES_ENDPOINT, entryData)
    return data
}

/**
 * Updates an existing entry
 * @param {UpdateEntryDTO} entryData
 * @returns {Promise<Entry>}
 */
const updateEntry = async (entryData) => {
    const { data } = await axios.put(`${ENTRIES_ENDPOINT}/${entryData.id}`, entryData)
    return data
}

/**
 * Deletes an entry
 * @param {string} id - Entry ID
 * @returns {Promise<void>}
 */
const deleteEntry = async (id) => {
    await axios.delete(`${ENTRIES_ENDPOINT}/${id}`)
}

export function useEntries() {
    const queryClient = useQueryClient()
    const QUERY_KEY = ['entries']

    const entriesQuery = useQuery({
        queryKey: QUERY_KEY,
        queryFn: fetchEntries
    })

    // Mutation for creating an entry
    const createEntryMutation = useMutation({
        mutationFn: createEntry,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: QUERY_KEY })
            queryClient.refetchQueries({ queryKey: QUERY_KEY })
        }
    })

    // Mutation for updating an entry
    const updateEntryMutation = useMutation({
        mutationFn: updateEntry,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: QUERY_KEY })
            queryClient.refetchQueries({ queryKey: QUERY_KEY })
        }
    })

    // Mutation for deleting an entry
    const deleteEntryMutation = useMutation({
        mutationFn: deleteEntry,
        onSuccess: (_, deletedId) => {
            queryClient.setQueryData(QUERY_KEY, (oldEntries = []) =>
                oldEntries.filter((entry) => entry.id !== deletedId)
            )
        }
    })

    return {
        // Queries
        entries: entriesQuery.data,
        isLoading: entriesQuery.isLoading,
        isError: entriesQuery.isError,
        error: entriesQuery.error,
        refetch: entriesQuery.refetch,

        // Mutations
        createEntry: createEntryMutation.mutateAsync,
        updateEntry: updateEntryMutation.mutateAsync,
        deleteEntry: deleteEntryMutation.mutateAsync,

        // Mutation states
        isCreating: createEntryMutation.isLoading,
        isUpdating: updateEntryMutation.isLoading,
        isDeleting: deleteEntryMutation.isLoading
    }
} 