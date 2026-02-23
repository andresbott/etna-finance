import { apiClient } from '@/lib/api/client'
import type { Entry, CreateEntryDTO, UpdateEntryDTO, PaginatedEntriesResponse } from '@/types/entry'
import { toLocalDateString } from '@/utils/date'

/** Format a Date for API params as local YYYY-MM-DD. */
export const formatDate = (date: Date): string => {
    if (!(date instanceof Date) || isNaN(date.getTime())) return ''
    return toLocalDateString(date)
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
export const getEntries = async (options: GetEntriesOptions): Promise<PaginatedEntriesResponse> => {
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
export const createEntry = async (payload: CreateEntryDTO): Promise<Entry> => {
    const { data } = await apiClient.post('/fin/entries', payload)
    return data
}

/**
 * Updates an existing entry
 */
export const updateEntry = async (payload: UpdateEntryDTO): Promise<Entry> => {
    const { data } = await apiClient.put(`/fin/entries/${payload.id}`, payload)
    return data
}

/**
 * Fetches a single entry by id. Use when the list response is incomplete (e.g. sell fees not in list).
 */
export const getEntry = async (id: string | number): Promise<Record<string, unknown>> => {
    const { data } = await apiClient.get(`/fin/entries/${id}`)
    return data as Record<string, unknown>
}

/**
 * Deletes an entry
 */
export const deleteEntry = async (id: string): Promise<void> => {
    await apiClient.delete(`/fin/entries/${id}`)
}

/**
 * Payload for creating a stock buy or sell transaction
 */
export interface CreateStockTransactionPayload {
    type: 'stockbuy' | 'stocksell'
    description: string
    date: string
    instrumentId: number
    quantity: number
    totalAmount: number
    investmentAccountId: number
    cashAccountId: number
}

/**
 * Creates a stock buy or sell transaction
 */
export const createStockTransaction = async (
    payload: CreateStockTransactionPayload
): Promise<unknown> => {
    const { data } = await apiClient.post('/fin/entries', payload)
    return data
}

/**
 * Payload for creating a stock grant (instruments added for free; no cash account)
 */
export interface CreateStockGrantPayload {
    type: 'stockgrant'
    description: string
    date: string
    instrumentId: number
    quantity: number
    fairMarketValue?: number
    accountId: number // Investment or Unvested account that receives the shares
}

/**
 * Creates a stock grant transaction
 */
export const createStockGrant = async (
    payload: CreateStockGrantPayload
): Promise<unknown> => {
    const { data } = await apiClient.post('/fin/entries', payload)
    return data
}

/**
 * Payload for creating a stock transfer (instruments moved between two position accounts, e.g. Unvested → Investment)
 */
export interface CreateStockTransferPayload {
    type: 'stocktransfer'
    description: string
    date: string
    instrumentId: number
    quantity: number
    originAccountId: number  // source (investment or unvested)
    targetAccountId: number // target (investment or unvested)
}

/**
 * Creates a stock transfer transaction
 */
export const createStockTransfer = async (
    payload: CreateStockTransferPayload
): Promise<unknown> => {
    const { data } = await apiClient.post('/fin/entries', payload)
    return data
}