import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { unref, computed, Ref } from 'vue'
import { GetEntries, CreateEntry, UpdateEntry, DeleteEntry } from '@/lib/api/Entry'
import type { Entry, CreateEntryDTO, UpdateEntryDTO } from '@/types/entry'

export function useEntries(
    startDateRef: Ref<Date | null>,
    endDateRef: Ref<Date | null>,
    accountIdsRef: Ref<string[] | null> = null
) {
    const queryClient = useQueryClient()

    const queryKey = computed(() => {
        const start = unref(startDateRef)
        const end = unref(endDateRef)
        const accountIds = unref(accountIdsRef)

        const key = ['entries']

        if (start && end) {
            key.push(start.toISOString(), end.toISOString())
        } else {
            key.push('invalid') // fallback key to avoid undefined
        }

        // Add account IDs to query key if provided
        if (accountIds && accountIds.length > 0) {
            key.push('accounts', ...accountIds)
        }

        return key
    })

    const entriesQuery = useQuery({
        queryKey,
        enabled: computed(() => !!unref(startDateRef) && !!unref(endDateRef)),
        queryFn: () => {
            const start = unref(startDateRef)
            const end = unref(endDateRef)
            const accountIds = unref(accountIdsRef) || []
            
            if (!start || !end) {
                return Promise.resolve([])
            }
            
            return GetEntries(start, end, accountIds)
        }
    })

    // Mutation for creating an entry
    const createEntryMutation = useMutation({
        mutationFn: (payload: CreateEntryDTO) => CreateEntry(payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: queryKey.value })
        }
    })

    // Mutation for updating an entry
    const updateEntryMutation = useMutation({
        mutationFn: (payload: UpdateEntryDTO) => UpdateEntry(payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: queryKey.value })
        }
    })

    // Mutation for deleting an entry
    const deleteEntryMutation = useMutation({
        mutationFn: (id: string) => DeleteEntry(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: queryKey.value })
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