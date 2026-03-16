import { useQuery, keepPreviousData } from '@tanstack/vue-query'
import { unref, computed, Ref, ref } from 'vue'
import { getEntries } from '@/lib/api/Entry'
import type { PaginatedEntriesResponse } from '@/types/entry'
import { useEntryMutations } from './useEntryMutations'

export interface UseEntriesOptions {
    startDate?: Ref<Date | null>
    endDate?: Ref<Date | null>
    accountIds?: Ref<string[] | null>
    page?: Ref<number>
    limit?: Ref<number>
    categoryIds?: Ref<number[]>
    types?: Ref<string[]>
    hasAttachment?: Ref<boolean>
    search?: Ref<string>
}

const DEFAULT_PAGE_SIZE = 25

export function useEntries(options: UseEntriesOptions = {}) {
    const {
        startDate: startDateRef = ref(null),
        endDate: endDateRef = ref(null),
        accountIds: accountIdsRef = ref(null),
        page: pageRef = ref(1),
        limit: limitRef = ref(DEFAULT_PAGE_SIZE),
        categoryIds: categoryIdsRef = ref([]),
        types: typesRef = ref([]),
        hasAttachment: hasAttachmentRef = ref(false),
        search: searchRef = ref('')
    } = options

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

        const catIds = unref(categoryIdsRef)
        if (catIds && catIds.length > 0) {
            key.push('cats', ...catIds.map(String))
        }
        const txTypes = unref(typesRef)
        if (txTypes && txTypes.length > 0) {
            key.push('types', ...txTypes)
        }
        if (unref(hasAttachmentRef)) {
            key.push('attachment')
        }
        const searchVal = unref(searchRef)
        if (searchVal) {
            key.push('search', searchVal)
        }

        return key
    })

    const emptyResponse: PaginatedEntriesResponse = {
        items: [],
        total: 0,
        page: 1,
        limit: DEFAULT_PAGE_SIZE,
        priorBalance: 0
    }

    const entriesQuery = useQuery({
        queryKey,
        enabled: computed(() => !!unref(startDateRef) && !!unref(endDateRef)),
        placeholderData: keepPreviousData,
        queryFn: () => {
            const start = unref(startDateRef)
            const end = unref(endDateRef)
            const accountIds = unref(accountIdsRef) || []
            const page = unref(pageRef)
            const limit = unref(limitRef)

            if (!start || !end) {
                return Promise.resolve(emptyResponse)
            }
            return getEntries({
                startDate: start,
                endDate: end,
                accountIds,
                page,
                limit,
                categoryIds: unref(categoryIdsRef),
                hasAttachment: unref(hasAttachmentRef) || undefined,
                types: unref(typesRef),
                search: unref(searchRef)
            })
        }
    })

    const { createEntry: createEntryFn, updateEntry: updateEntryFn, deleteEntry: deleteEntryFn,
            isCreating, isUpdating, isDeleting } = useEntryMutations()

    // Computed values for easy access to pagination data
    const entries = computed(() => entriesQuery.data.value?.items ?? [])
    const totalRecords = computed(() => entriesQuery.data.value?.total ?? 0)
    const currentPage = computed(() => entriesQuery.data.value?.page ?? 1)
    const pageSize = computed(() => entriesQuery.data.value?.limit ?? DEFAULT_PAGE_SIZE)
    const priorBalance = computed(() => entriesQuery.data.value?.priorBalance ?? 0)

    return {
        // Queries - now with proper pagination access
        entries,
        totalRecords,
        currentPage,
        pageSize,
        priorBalance,
        isLoading: entriesQuery.isLoading,
        isFetching: entriesQuery.isFetching,
        isError: entriesQuery.isError,
        error: entriesQuery.error,
        refetch: entriesQuery.refetch,

        // Mutations
        createEntry: createEntryFn,
        updateEntry: updateEntryFn,
        deleteEntry: deleteEntryFn,

        // Mutation states
        isCreating, isUpdating, isDeleting
    }
}
