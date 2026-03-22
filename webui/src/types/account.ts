/**
 * Account types and their allowed operations configuration.
 *
 * This file provides a centralized mapping of which entry operations
 * are permitted for each account type. Update this file to modify
 * the allowed operations for any account type.
 */

/**
 * Account domain type (API response shape).
 */
export interface Account {
    id: number
    name: string
    currency: string
    type: string
    icon?: string
    notes?: string
    importProfileId?: number
}

/**
 * Account provider with nested accounts (API response shape).
 */
export interface AccountProvider {
    id: number
    name: string
    description?: string
    icon?: string
    accounts: Account[]
}

/**
 * Account types available in the system
 */
export const ACCOUNT_TYPES = {
    CASH: 'cash',
    CHECKING: 'checkin',
    SAVINGS: 'savings',
    INVESTMENT: 'investment',
    UNVESTED: 'unvested', // not yet accessible (e.g. unvested RSUs); can transfer to Investment
    LENT: 'lent', // money lent to others; owned but not in any account
} as const

export type AccountType = typeof ACCOUNT_TYPES[keyof typeof ACCOUNT_TYPES]

/**
 * Icons for each account type (Tabler icon names, prefix-free)
 */
export const ACCOUNT_TYPE_ICONS: Record<AccountType, string> = {
    [ACCOUNT_TYPES.CASH]: 'cash-banknote',
    [ACCOUNT_TYPES.CHECKING]: 'credit-card',
    [ACCOUNT_TYPES.SAVINGS]: 'box',
    [ACCOUNT_TYPES.INVESTMENT]: 'chart-line',
    [ACCOUNT_TYPES.UNVESTED]: 'gift',
    [ACCOUNT_TYPES.LENT]: 'send',
}

/**
 * Display labels for each account type (for UI).
 */
export const ACCOUNT_TYPE_LABELS: Record<AccountType, string> = {
    [ACCOUNT_TYPES.CASH]: 'Cash',
    [ACCOUNT_TYPES.CHECKING]: 'Checking',
    [ACCOUNT_TYPES.SAVINGS]: 'Savings',
    [ACCOUNT_TYPES.INVESTMENT]: 'Investment',
    [ACCOUNT_TYPES.UNVESTED]: 'Unvested products',
    [ACCOUNT_TYPES.LENT]: 'Lent money',
}

/**
 * Get the display label for an account type.
 */
export function getAccountTypeLabel(accountType: AccountType | string | null | undefined): string {
    if (!accountType) return 'Unknown'
    return ACCOUNT_TYPE_LABELS[accountType as AccountType] ?? accountType
}

/**
 * Get the icon for an account type, with fallback to default wallet icon.
 */
export function getAccountTypeIcon(accountType: AccountType | string | null | undefined): string {
    if (!accountType) return 'wallet'
    return ACCOUNT_TYPE_ICONS[accountType as AccountType] || 'wallet'
}

/**
 * Entry operation types available in the system
 */
export const ENTRY_OPERATIONS = {
    EXPENSE: 'expense',
    INCOME: 'income',
    TRANSFER: 'transfer',
    BUY_STOCK: 'buyStock',
    SELL_STOCK: 'sellStock',
    GRANT_STOCK: 'grantStock',
    TRANSFER_INSTRUMENT: 'transferInstrument',
    BALANCE_STATUS: 'balanceStatus',
    IMPORT_CSV: 'importCsv',
} as const

export type EntryOperation = typeof ENTRY_OPERATIONS[keyof typeof ENTRY_OPERATIONS]

/**
 * Mapping of account types to their allowed entry operations.
 * 
 * Update this object to change which operations are available for each account type.
 * 
 * Current configuration:
 * - cash, checking, savings: income, expense, transfer
 * - investment, unvested: buy/sell stock (unvested = not yet accessible, e.g. RSUs)
 */
export const ALLOWED_OPERATIONS_BY_ACCOUNT_TYPE: Record<AccountType, EntryOperation[]> = {
    [ACCOUNT_TYPES.CASH]: [
        ENTRY_OPERATIONS.INCOME,
        ENTRY_OPERATIONS.EXPENSE,
        ENTRY_OPERATIONS.TRANSFER,
        ENTRY_OPERATIONS.BALANCE_STATUS,
        ENTRY_OPERATIONS.IMPORT_CSV,
    ],
    [ACCOUNT_TYPES.CHECKING]: [
        ENTRY_OPERATIONS.INCOME,
        ENTRY_OPERATIONS.EXPENSE,
        ENTRY_OPERATIONS.TRANSFER,
        ENTRY_OPERATIONS.BALANCE_STATUS,
        ENTRY_OPERATIONS.IMPORT_CSV,
    ],
    [ACCOUNT_TYPES.SAVINGS]: [
        ENTRY_OPERATIONS.INCOME,
        ENTRY_OPERATIONS.EXPENSE,
        ENTRY_OPERATIONS.TRANSFER,
        ENTRY_OPERATIONS.BALANCE_STATUS,
        ENTRY_OPERATIONS.IMPORT_CSV,
    ],
    [ACCOUNT_TYPES.INVESTMENT]: [
        ENTRY_OPERATIONS.BUY_STOCK,
        ENTRY_OPERATIONS.SELL_STOCK,
        ENTRY_OPERATIONS.GRANT_STOCK,
        ENTRY_OPERATIONS.TRANSFER_INSTRUMENT,
    ],
    [ACCOUNT_TYPES.UNVESTED]: [
        ENTRY_OPERATIONS.BUY_STOCK,
        ENTRY_OPERATIONS.SELL_STOCK,
        ENTRY_OPERATIONS.GRANT_STOCK,
        ENTRY_OPERATIONS.TRANSFER_INSTRUMENT,
    ],
    [ACCOUNT_TYPES.LENT]: [
        ENTRY_OPERATIONS.INCOME,
        ENTRY_OPERATIONS.EXPENSE,
        ENTRY_OPERATIONS.TRANSFER,
        ENTRY_OPERATIONS.BALANCE_STATUS,
        ENTRY_OPERATIONS.IMPORT_CSV,
    ],
}

/**
 * Get allowed operations for a given account type.
 * Returns all operations if accountType is null/undefined (for "all transactions" view).
 * 
 * @param accountType - The account type to get allowed operations for
 * @returns Array of allowed entry operations
 */
export function getAllowedOperations(accountType: AccountType | null | undefined): EntryOperation[] {
    if (!accountType) {
        // Return all operations when no specific account is selected
        return Object.values(ENTRY_OPERATIONS)
    }
    
    return ALLOWED_OPERATIONS_BY_ACCOUNT_TYPE[accountType] || Object.values(ENTRY_OPERATIONS)
}

/**
 * Check if an operation is allowed for a given account type.
 * 
 * @param operation - The operation to check
 * @param accountType - The account type to check against
 * @returns true if the operation is allowed, false otherwise
 */
export function isOperationAllowed(operation: EntryOperation, accountType: AccountType | null | undefined): boolean {
    const allowedOps = getAllowedOperations(accountType)
    return allowedOps.includes(operation)
}
