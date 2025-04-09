import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import axios from 'axios'
import { unref, computed } from 'vue'

const API_BASE_URL = import.meta.env.VITE_SERVER_URL_V0
const ENTRIES_ENDPOINT = `${API_BASE_URL}/fin/entries`


function formatDate(date) {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
}


/**
 * Fetches entries from the API with date range filtering
 * @param {Date} startDate - Start date for filtering
 * @param {Date} endDate - End date for filtering
 * @returns {Promise<Entry[]>}
 */
const fetchEntries = async (startDate, endDate) => {
    const params = new URLSearchParams({
        startDate: formatDate(startDate),
        endDate: formatDate(endDate)
    })
    const { data } = await axios.get(`${ENTRIES_ENDPOINT}?${params}`)
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

export function useEntries(startDateRef, endDateRef) {
    const queryClient = useQueryClient()


    const queryKey = computed(() => [
        'entries',
        unref(startDateRef),
        unref(endDateRef)
    ])

    const entriesQuery = useQuery({
        queryKey,
        queryFn: () => fetchEntries(unref(startDateRef), unref(endDateRef))
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