import { apiClient } from '@/lib/api/client'
import type { Entry, CreateEntryDTO, UpdateEntryDTO, PaginatedEntriesResponse } from '@/types/entry'

/**
 * Helper function to format a Date object to YYYY-MM-DD string
 */
export const formatDate = (date: Date): string => {
    if (!(date instanceof Date) || isNaN(date.getTime())) return ''

    const year = date.getFullYear()
    const month = String(date.getMonth() + 1).padStart(2, '0')
    const day = String(date.getDate()).padStart(2, '0')
    return `${year}-${month}-${day}`
}

export interface GetEntriesOptions {
    startDate: Date
    endDate: Date
    accountIds?: string[]
    page?: number
    limit?: number
}

/**
 * Fetches entries from the API with date range filtering, optional account filtering, and pagination
 */
export const GetEntries = async (options: GetEntriesOptions): Promise<PaginatedEntriesResponse> => {
    const { startDate, endDate, accountIds = [], page = 1, limit = 25 } = options

    const params = new URLSearchParams({
        startDate: formatDate(startDate),
        endDate: formatDate(endDate),
        page: String(page),
        limit: String(limit)
    })

    // Add account IDs to params if provided
    if (accountIds && accountIds.length > 0) {
        accountIds.forEach((id) => params.append('accountIds', id))
    }

    const { data } = await apiClient.get(`/fin/entries?${params}`)
    
    const items = data.items || []
    
    // Calculate total: check various common field names from API response
    let total = data.total ?? data.totalCount ?? data.count ?? data.totalItems ?? data.total_count
    
    if (total === undefined || total === null) {
        // Fallback: if we received a full page, assume there could be more
        // This enables pagination controls even if backend doesn't return total
        total = items.length >= limit ? (page * limit) + 1 : ((page - 1) * limit) + items.length
    }
    
    return {
        items,
        total,
        page: data.page ?? data.currentPage ?? page,
        limit: data.limit ?? data.pageSize ?? data.perPage ?? limit
    }
}

/**
 * Creates a new entry
 */
export const CreateEntry = async (payload: CreateEntryDTO): Promise<Entry> => {
    const { data } = await apiClient.post('/fin/entries', payload)
    return data
}

/**
 * Updates an existing entry
 */
export const UpdateEntry = async (payload: UpdateEntryDTO): Promise<Entry> => {
    const { data } = await apiClient.put(`/fin/entries/${payload.id}`, payload)
    return data
}

/**
 * Deletes an entry
 */
export const DeleteEntry = async (id: string): Promise<void> => {
    await apiClient.delete(`/fin/entries/${id}`)
}