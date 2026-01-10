import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { unref, computed, Ref, ref } from 'vue'
import { GetEntries, CreateEntry, UpdateEntry, DeleteEntry } from '@/lib/api/Entry'
import type { CreateEntryDTO, UpdateEntryDTO, PaginatedEntriesResponse } from '@/types/entry'

export interface UseEntriesOptions {
    startDate?: Ref<Date | null>
    endDate?: Ref<Date | null>
    accountIds?: Ref<string[] | null>
    page?: Ref<number>
    limit?: Ref<number>
}

const DEFAULT_PAGE_SIZE = 25

export function useEntries(options: UseEntriesOptions = {}) {
    const {
        startDate: startDateRef = ref(null),
        endDate: endDateRef = ref(null),
        accountIds: accountIdsRef = ref(null),
        page: pageRef = ref(1),
        limit: limitRef = ref(DEFAULT_PAGE_SIZE)
    } = options

    const queryClient = useQueryClient()

    const queryKey = computed(() => {
        const start = unref(startDateRef)
        const end = unref(endDateRef)
        const accountIds = unref(accountIdsRef)
        const page = unref(pageRef)
        const limit = unref(limitRef)

        const key: (string | number)[] = ['entries']

        if (start && end) {
            key.push(start.toISOString(), end.toISOString())
        } else {
            key.push('invalid') // fallback key to avoid undefined
        }

        // Add pagination to query key
        key.push('page', page, 'limit', limit)

        // Add account IDs to query key if provided
        if (accountIds && accountIds.length > 0) {
            key.push('accounts', ...accountIds)
        }

        return key
    })

    const emptyResponse: PaginatedEntriesResponse = {
        items: [],
        total: 0,
        page: 1,
        limit: DEFAULT_PAGE_SIZE
    }

    const entriesQuery = useQuery({
        queryKey,
        enabled: computed(() => !!unref(startDateRef) && !!unref(endDateRef)),
        queryFn: () => {
            const start = unref(startDateRef)
            const end = unref(endDateRef)
            const accountIds = unref(accountIdsRef) || []
            const page = unref(pageRef)
            const limit = unref(limitRef)

            if (!start || !end) {
                return Promise.resolve(emptyResponse)
            }
            return GetEntries({
                startDate: start,
                endDate: end,
                accountIds,
                page,
                limit
            })
        }
    })

    // Mutation for creating an entry
    const createEntryMutation = useMutation({
        mutationFn: (payload: CreateEntryDTO) => CreateEntry(payload),
        onSuccess: () => {
            // Invalidate all entries queries, not just the specific date range
            queryClient.invalidateQueries({ queryKey: ['entries'] })
        }
    })

    // Mutation for updating an entry
    const updateEntryMutation = useMutation({
        mutationFn: (payload: UpdateEntryDTO) => UpdateEntry(payload),
        onSuccess: () => {
            // Invalidate all entries queries, not just the specific date range
            queryClient.invalidateQueries({ queryKey: ['entries'] })
        }
    })

    // Mutation for deleting an entry
    const deleteEntryMutation = useMutation({
        mutationFn: (id: string) => DeleteEntry(id),
        onSuccess: () => {
            // Invalidate all entries queries, not just the specific date range
            queryClient.invalidateQueries({ queryKey: ['entries'] })
        }
    })

    // Computed values for easy access to pagination data
    const entries = computed(() => entriesQuery.data.value?.items || [])
    const totalRecords = computed(() => entriesQuery.data.value?.total || 0)
    const currentPage = computed(() => entriesQuery.data.value?.page || 1)
    const pageSize = computed(() => entriesQuery.data.value?.limit || DEFAULT_PAGE_SIZE)

    return {
        // Queries - now with proper pagination access
        entries,
        totalRecords,
        currentPage,
        pageSize,
        isLoading: entriesQuery.isLoading,
        isFetching: entriesQuery.isFetching,
        isError: entriesQuery.isError,
        error: entriesQuery.error,
        refetch: entriesQuery.refetch,

        // Mutations
        createEntry: createEntryMutation.mutateAsync,
        updateEntry: updateEntryMutation.mutateAsync,
        deleteEntry: deleteEntryMutation.mutateAsync,

        // Mutation states
        isCreating: createEntryMutation.isPending,
        isUpdating: updateEntryMutation.isPending,
        isDeleting: deleteEntryMutation.isPending
    }
}
