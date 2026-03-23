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
    categoryIds?: number[]
    hasAttachment?: boolean
    types?: string[]
    search?: string
}

/**
 * Fetches entries from the API with date range filtering, optional account filtering, and pagination
 */
export const getEntries = async (options: GetEntriesOptions): Promise<PaginatedEntriesResponse> => {
    const { startDate, endDate, accountIds = [], page = 1, limit = 25, categoryIds = [], hasAttachment, types = [], search } = options

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

    if (categoryIds && categoryIds.length > 0) {
        categoryIds.forEach((id) => params.append('categoryIds', String(id)))
    }
    if (types && types.length > 0) {
        types.forEach((t) => params.append('types', t))
    }
    if (hasAttachment) {
        params.set('hasAttachment', 'true')
    }
    if (search) {
        params.set('search', search)
    }

    const { data } = await apiClient.get(`/fin/entries?${params}`)
    
    return {
        items: data.items || [],
        total: data.total ?? 0,
        page,
        limit,
        priorBalance: data.priorBalance ?? 0
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

/**
 * Payload for creating a stock forfeit (unvested shares forfeited on departure)
 */
export interface CreateStockForfeitPayload {
    type: 'stockforfeit'
    description: string
    date: string
    instrumentId: number
    accountId: number
    lotAllocations: Array<{ lotId: number; quantity: number }>
    notes?: string
}

/**
 * Creates a stock forfeit transaction
 */
export const createStockForfeit = async (
    payload: CreateStockForfeitPayload
): Promise<unknown> => {
    const { data } = await apiClient.post('/fin/entries', payload)
    return data
}

/**
 * Payload for creating a stock vest (shares vesting from unvested to investment account)
 */
export interface CreateStockVestPayload {
    type: 'stockvest'
    description: string
    notes?: string
    date: string
    instrumentId: number
    vestingPrice: number
    categoryId: number
    originAccountId: number
    targetAccountId: number
    lotAllocations: Array<{ lotId: number; quantity: number }>
}

/**
 * Creates a stock vest transaction
 */
export const createStockVest = async (
    payload: CreateStockVestPayload
): Promise<unknown> => {
    const { data } = await apiClient.post('/fin/entries', payload)
    return data
}