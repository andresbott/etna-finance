import { apiClient } from '@/lib/api/client'
import type { Entry, CreateEntryDTO, UpdateEntryDTO } from '@/types/entry'

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

/**
 * Fetches entries from the API with date range filtering and optional account filtering
 */
export const GetEntries = async (
    startDate: Date,
    endDate: Date,
    accountIds: string[] = []
): Promise<Entry[]> => {
    const params = new URLSearchParams({
        startDate: formatDate(startDate),
        endDate: formatDate(endDate)
    })

    // Add account IDs to params if provided
    if (accountIds && accountIds.length > 0) {
        accountIds.forEach((id) => params.append('accountIds', id))
    }

    const { data } = await apiClient.get(`/fin/entries?${params}`)
    return data.items || []
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