export interface Entry {
    id: string
    date: string
    description?: string
    amount: number
    accountId: string
    categoryId?: number
    notes?: string
}

export interface CreateEntryDTO {
    date: string
    description?: string
    amount: number
    accountId: string
    categoryId?: number
    notes?: string
}

export interface UpdateEntryDTO {
    id: string
    date?: string
    description?: string
    amount?: number
    accountId?: string
    categoryId?: number
    notes?: string
}

export interface EntryFilters {
    startDate: Date
    endDate: Date
    accountIds?: string[]
}