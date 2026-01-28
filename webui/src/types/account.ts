/**
 * Account types and their allowed operations configuration.
 * 
 * This file provides a centralized mapping of which entry operations
 * are permitted for each account type. Update this file to modify
 * the allowed operations for any account type.
 */

/**
 * Account types available in the system
 */
export const ACCOUNT_TYPES = {
    CASH: 'cash',
    CHECKING: 'checkin',
    SAVINGS: 'savings',
    INVESTMENT: 'investment',
} as const

export type AccountType = typeof ACCOUNT_TYPES[keyof typeof ACCOUNT_TYPES]

/**
 * Icons for each account type (PrimeIcons class names without 'pi-' prefix)
 */
export const ACCOUNT_TYPE_ICONS: Record<AccountType, string> = {
    [ACCOUNT_TYPES.CASH]: 'pi-money-bill',
    [ACCOUNT_TYPES.CHECKING]: 'pi-credit-card',
    [ACCOUNT_TYPES.SAVINGS]: 'pi-box',
    [ACCOUNT_TYPES.INVESTMENT]: 'pi-chart-line',
}

/**
 * Get the icon for an account type, with fallback to default wallet icon.
 */
export function getAccountTypeIcon(accountType: AccountType | string | null | undefined): string {
    if (!accountType) return 'pi-wallet'
    return ACCOUNT_TYPE_ICONS[accountType as AccountType] || 'pi-wallet'
}

/**
 * Entry operation types available in the system
 */
export const ENTRY_OPERATIONS = {
    EXPENSE: 'expense',
    INCOME: 'income',
    TRANSFER: 'transfer',
    STOCK: 'stock',
} as const

export type EntryOperation = typeof ENTRY_OPERATIONS[keyof typeof ENTRY_OPERATIONS]

/**
 * Mapping of account types to their allowed entry operations.
 * 
 * Update this object to change which operations are available for each account type.
 * 
 * Current configuration:
 * - cash, checking, savings: income, expense, transfer
 * - investment: stock operations (placeholder for future implementation)
 */
export const ALLOWED_OPERATIONS_BY_ACCOUNT_TYPE: Record<AccountType, EntryOperation[]> = {
    [ACCOUNT_TYPES.CASH]: [
        ENTRY_OPERATIONS.INCOME,
        ENTRY_OPERATIONS.EXPENSE,
        ENTRY_OPERATIONS.TRANSFER,
    ],
    [ACCOUNT_TYPES.CHECKING]: [
        ENTRY_OPERATIONS.INCOME,
        ENTRY_OPERATIONS.EXPENSE,
        ENTRY_OPERATIONS.TRANSFER,
    ],
    [ACCOUNT_TYPES.SAVINGS]: [
        ENTRY_OPERATIONS.INCOME,
        ENTRY_OPERATIONS.EXPENSE,
        ENTRY_OPERATIONS.TRANSFER,
    ],
    [ACCOUNT_TYPES.INVESTMENT]: [
        // Stock operations only - placeholder for future implementation
        ENTRY_OPERATIONS.STOCK,
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
