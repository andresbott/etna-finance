/** Shared fields present on all entry types returned by the API. */
interface BaseEntry {
    id: string
    date: string
    description?: string
    notes?: string
    attachmentId?: number | null
    categoryId?: number
}

export interface IncomeEntry extends BaseEntry {
    type: 'income'
    accountId: string
    amount: number
    Amount: number
    targetStockAmount?: number
}

export interface ExpenseEntry extends BaseEntry {
    type: 'expense'
    accountId: string
    amount: number
    Amount: number
    targetStockAmount?: number
}

export interface TransferEntry extends BaseEntry {
    type: 'transfer'
    originAccountId: number
    targetAccountId: number
    originAmount: number
    targetAmount: number
    originStockAmount?: number
    targetStockAmount?: number
}

export interface StockBuyEntry extends BaseEntry {
    type: 'stockbuy'
    investmentAccountId: number
    cashAccountId: number
    instrumentId: number
    quantity: number
    totalAmount: number
    StockAmount: number
}

export interface StockSellEntry extends BaseEntry {
    type: 'stocksell'
    investmentAccountId: number
    cashAccountId: number
    instrumentId: number
    quantity: number
    totalAmount: number
    StockAmount: number
    costBasis?: number
    fees?: number
    lotAllocations?: Array<{ lotId: number; quantity: number }>
}

export interface StockGrantEntry extends BaseEntry {
    type: 'stockgrant'
    accountId: string
    instrumentId: number
    quantity: number
    fairMarketValue: number
}

export interface StockTransferEntry extends BaseEntry {
    type: 'stocktransfer'
    originAccountId: number
    targetAccountId: number
    instrumentId: number
    quantity: number
}

export interface BalanceStatusEntry extends BaseEntry {
    type: 'balancestatus'
    accountId: string
    amount: number
    Amount: number
}

export type Entry =
    | IncomeEntry
    | ExpenseEntry
    | TransferEntry
    | StockBuyEntry
    | StockSellEntry
    | StockGrantEntry
    | StockTransferEntry
    | BalanceStatusEntry

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
    priorBalance: number
}