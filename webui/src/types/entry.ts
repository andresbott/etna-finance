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
    // Stock buy / sell
    instrumentId?: number
    quantity?: number
    totalAmount?: number
    StockAmount?: number
    investmentAccountId?: number
    cashAccountId?: number
    // Stock transfer
    originAccountId?: number
    targetAccountId?: number
    // Manual lot selection for stock sell
    lotAllocations?: Array<{ lotId: number; quantity: number }>
}

export interface EntryFilters {
    startDate: Date
    endDate: Date
    accountIds?: string[]
}

export interface PaginatedEntriesResponse {
    items: Entry[]
    total: number
    page: number
    limit: number
}