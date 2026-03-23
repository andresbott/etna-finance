import { describe, it, expect } from 'vitest'
import {
    ACCOUNT_TYPES,
    ACCOUNT_TYPE_ICONS,
    ACCOUNT_TYPE_LABELS,
    ENTRY_OPERATIONS,
    ALLOWED_OPERATIONS_BY_ACCOUNT_TYPE,
    getAccountTypeLabel,
    getAccountTypeIcon,
    getAllowedOperations,
    isOperationAllowed,
} from './account'
import type { AccountType, EntryOperation } from './account'

describe('getAccountTypeLabel', () => {
    it('returns correct label for each account type', () => {
        expect(getAccountTypeLabel(ACCOUNT_TYPES.CASH)).toBe('Cash')
        expect(getAccountTypeLabel(ACCOUNT_TYPES.CHECKING)).toBe('Checking')
        expect(getAccountTypeLabel(ACCOUNT_TYPES.SAVINGS)).toBe('Savings')
        expect(getAccountTypeLabel(ACCOUNT_TYPES.INVESTMENT)).toBe('Investment')
        expect(getAccountTypeLabel(ACCOUNT_TYPES.UNVESTED)).toBe('Unvested products')
        expect(getAccountTypeLabel(ACCOUNT_TYPES.LENT)).toBe('Lent money')
    })

    it('returns "Unknown" for null', () => {
        expect(getAccountTypeLabel(null)).toBe('Unknown')
    })

    it('returns "Unknown" for undefined', () => {
        expect(getAccountTypeLabel(undefined)).toBe('Unknown')
    })

    it('returns "Unknown" for empty string', () => {
        expect(getAccountTypeLabel('')).toBe('Unknown')
    })

    it('returns the raw string for an unrecognized account type', () => {
        expect(getAccountTypeLabel('crypto')).toBe('crypto')
    })
})

describe('getAccountTypeIcon', () => {
    it('returns correct icon for each account type', () => {
        expect(getAccountTypeIcon(ACCOUNT_TYPES.CASH)).toBe('cash-banknote')
        expect(getAccountTypeIcon(ACCOUNT_TYPES.CHECKING)).toBe('credit-card')
        expect(getAccountTypeIcon(ACCOUNT_TYPES.SAVINGS)).toBe('box')
        expect(getAccountTypeIcon(ACCOUNT_TYPES.INVESTMENT)).toBe('chart-line')
        expect(getAccountTypeIcon(ACCOUNT_TYPES.UNVESTED)).toBe('gift')
        expect(getAccountTypeIcon(ACCOUNT_TYPES.LENT)).toBe('send')
    })

    it('returns fallback "wallet" for null', () => {
        expect(getAccountTypeIcon(null)).toBe('wallet')
    })

    it('returns fallback "wallet" for undefined', () => {
        expect(getAccountTypeIcon(undefined)).toBe('wallet')
    })

    it('returns fallback "wallet" for empty string', () => {
        expect(getAccountTypeIcon('')).toBe('wallet')
    })

    it('returns fallback "wallet" for an unrecognized account type', () => {
        expect(getAccountTypeIcon('crypto')).toBe('wallet')
    })
})

describe('getAllowedOperations', () => {
    const allOperations = Object.values(ENTRY_OPERATIONS)

    it('returns all operations when accountType is null', () => {
        const result = getAllowedOperations(null)
        expect(result).toEqual(allOperations)
    })

    it('returns all operations when accountType is undefined', () => {
        const result = getAllowedOperations(undefined)
        expect(result).toEqual(allOperations)
    })

    it('returns cash-type operations for CASH', () => {
        const result = getAllowedOperations(ACCOUNT_TYPES.CASH)
        expect(result).toEqual([
            ENTRY_OPERATIONS.INCOME,
            ENTRY_OPERATIONS.EXPENSE,
            ENTRY_OPERATIONS.TRANSFER,
            ENTRY_OPERATIONS.BALANCE_STATUS,
            ENTRY_OPERATIONS.IMPORT_CSV,
        ])
    })

    it('returns cash-type operations for CHECKING', () => {
        const result = getAllowedOperations(ACCOUNT_TYPES.CHECKING)
        expect(result).toEqual([
            ENTRY_OPERATIONS.INCOME,
            ENTRY_OPERATIONS.EXPENSE,
            ENTRY_OPERATIONS.TRANSFER,
            ENTRY_OPERATIONS.BALANCE_STATUS,
            ENTRY_OPERATIONS.IMPORT_CSV,
        ])
    })

    it('returns cash-type operations for SAVINGS', () => {
        const result = getAllowedOperations(ACCOUNT_TYPES.SAVINGS)
        expect(result).toEqual([
            ENTRY_OPERATIONS.INCOME,
            ENTRY_OPERATIONS.EXPENSE,
            ENTRY_OPERATIONS.TRANSFER,
            ENTRY_OPERATIONS.BALANCE_STATUS,
            ENTRY_OPERATIONS.IMPORT_CSV,
        ])
    })

    it('returns cash-type operations for LENT', () => {
        const result = getAllowedOperations(ACCOUNT_TYPES.LENT)
        expect(result).toEqual([
            ENTRY_OPERATIONS.INCOME,
            ENTRY_OPERATIONS.EXPENSE,
            ENTRY_OPERATIONS.TRANSFER,
            ENTRY_OPERATIONS.BALANCE_STATUS,
            ENTRY_OPERATIONS.IMPORT_CSV,
        ])
    })

    it('returns investment operations for INVESTMENT', () => {
        const result = getAllowedOperations(ACCOUNT_TYPES.INVESTMENT)
        expect(result).toEqual([
            ENTRY_OPERATIONS.BUY_STOCK,
            ENTRY_OPERATIONS.SELL_STOCK,
            ENTRY_OPERATIONS.GRANT_STOCK,
            ENTRY_OPERATIONS.TRANSFER_INSTRUMENT,
        ])
    })

    it('returns grant, vest, and forfeit operations for UNVESTED', () => {
        const result = getAllowedOperations(ACCOUNT_TYPES.UNVESTED)
        expect(result).toEqual([
            ENTRY_OPERATIONS.GRANT_STOCK,
            ENTRY_OPERATIONS.VEST_STOCK,
            ENTRY_OPERATIONS.FORFEIT_STOCK,
        ])
    })

    it('does not include expense in investment accounts', () => {
        expect(getAllowedOperations(ACCOUNT_TYPES.INVESTMENT)).not.toContain(ENTRY_OPERATIONS.EXPENSE)
        expect(getAllowedOperations(ACCOUNT_TYPES.UNVESTED)).not.toContain(ENTRY_OPERATIONS.EXPENSE)
    })

    it('does not include buyStock in cash-like accounts', () => {
        expect(getAllowedOperations(ACCOUNT_TYPES.CASH)).not.toContain(ENTRY_OPERATIONS.BUY_STOCK)
        expect(getAllowedOperations(ACCOUNT_TYPES.CHECKING)).not.toContain(ENTRY_OPERATIONS.BUY_STOCK)
        expect(getAllowedOperations(ACCOUNT_TYPES.SAVINGS)).not.toContain(ENTRY_OPERATIONS.BUY_STOCK)
        expect(getAllowedOperations(ACCOUNT_TYPES.LENT)).not.toContain(ENTRY_OPERATIONS.BUY_STOCK)
    })
})

describe('isOperationAllowed', () => {
    it('returns true for allowed operations on cash-like accounts', () => {
        const cashLike: AccountType[] = [
            ACCOUNT_TYPES.CASH,
            ACCOUNT_TYPES.CHECKING,
            ACCOUNT_TYPES.SAVINGS,
            ACCOUNT_TYPES.LENT,
        ]
        const allowedOps: EntryOperation[] = [
            ENTRY_OPERATIONS.INCOME,
            ENTRY_OPERATIONS.EXPENSE,
            ENTRY_OPERATIONS.TRANSFER,
            ENTRY_OPERATIONS.BALANCE_STATUS,
            ENTRY_OPERATIONS.IMPORT_CSV,
        ]

        for (const acctType of cashLike) {
            for (const op of allowedOps) {
                expect(isOperationAllowed(op, acctType)).toBe(true)
            }
        }
    })

    it('returns false for disallowed operations on cash-like accounts', () => {
        const cashLike: AccountType[] = [
            ACCOUNT_TYPES.CASH,
            ACCOUNT_TYPES.CHECKING,
            ACCOUNT_TYPES.SAVINGS,
            ACCOUNT_TYPES.LENT,
        ]
        const disallowedOps: EntryOperation[] = [
            ENTRY_OPERATIONS.BUY_STOCK,
            ENTRY_OPERATIONS.SELL_STOCK,
            ENTRY_OPERATIONS.GRANT_STOCK,
            ENTRY_OPERATIONS.TRANSFER_INSTRUMENT,
        ]

        for (const acctType of cashLike) {
            for (const op of disallowedOps) {
                expect(isOperationAllowed(op, acctType)).toBe(false)
            }
        }
    })

    it('returns true for allowed operations on investment accounts', () => {
        const investmentOps: EntryOperation[] = [
            ENTRY_OPERATIONS.BUY_STOCK,
            ENTRY_OPERATIONS.SELL_STOCK,
            ENTRY_OPERATIONS.GRANT_STOCK,
            ENTRY_OPERATIONS.TRANSFER_INSTRUMENT,
        ]
        for (const op of investmentOps) {
            expect(isOperationAllowed(op, ACCOUNT_TYPES.INVESTMENT)).toBe(true)
        }

        // Unvested accounts only allow grant, vest, and forfeit
        const unvestedOps: EntryOperation[] = [
            ENTRY_OPERATIONS.GRANT_STOCK,
            ENTRY_OPERATIONS.VEST_STOCK,
            ENTRY_OPERATIONS.FORFEIT_STOCK,
        ]
        for (const op of unvestedOps) {
            expect(isOperationAllowed(op, ACCOUNT_TYPES.UNVESTED)).toBe(true)
        }
        expect(isOperationAllowed(ENTRY_OPERATIONS.BUY_STOCK, ACCOUNT_TYPES.UNVESTED)).toBe(false)
        expect(isOperationAllowed(ENTRY_OPERATIONS.SELL_STOCK, ACCOUNT_TYPES.UNVESTED)).toBe(false)
        expect(isOperationAllowed(ENTRY_OPERATIONS.TRANSFER_INSTRUMENT, ACCOUNT_TYPES.UNVESTED)).toBe(false)
    })

    it('returns false for disallowed operations on investment accounts', () => {
        const investmentLike: AccountType[] = [
            ACCOUNT_TYPES.INVESTMENT,
            ACCOUNT_TYPES.UNVESTED,
        ]
        const disallowedOps: EntryOperation[] = [
            ENTRY_OPERATIONS.INCOME,
            ENTRY_OPERATIONS.EXPENSE,
            ENTRY_OPERATIONS.TRANSFER,
            ENTRY_OPERATIONS.BALANCE_STATUS,
            ENTRY_OPERATIONS.IMPORT_CSV,
        ]

        for (const acctType of investmentLike) {
            for (const op of disallowedOps) {
                expect(isOperationAllowed(op, acctType)).toBe(false)
            }
        }
    })

    it('returns true for any operation when accountType is null', () => {
        for (const op of Object.values(ENTRY_OPERATIONS)) {
            expect(isOperationAllowed(op, null)).toBe(true)
        }
    })

    it('returns true for any operation when accountType is undefined', () => {
        for (const op of Object.values(ENTRY_OPERATIONS)) {
            expect(isOperationAllowed(op, undefined)).toBe(true)
        }
    })
})

describe('constants integrity', () => {
    it('every account type has a label', () => {
        for (const type of Object.values(ACCOUNT_TYPES)) {
            expect(ACCOUNT_TYPE_LABELS[type]).toBeDefined()
            expect(typeof ACCOUNT_TYPE_LABELS[type]).toBe('string')
        }
    })

    it('every account type has an icon', () => {
        for (const type of Object.values(ACCOUNT_TYPES)) {
            expect(ACCOUNT_TYPE_ICONS[type]).toBeDefined()
            expect(typeof ACCOUNT_TYPE_ICONS[type]).toBe('string')
        }
    })

    it('every account type has allowed operations defined', () => {
        for (const type of Object.values(ACCOUNT_TYPES)) {
            expect(ALLOWED_OPERATIONS_BY_ACCOUNT_TYPE[type]).toBeDefined()
            expect(Array.isArray(ALLOWED_OPERATIONS_BY_ACCOUNT_TYPE[type])).toBe(true)
            expect(ALLOWED_OPERATIONS_BY_ACCOUNT_TYPE[type].length).toBeGreaterThan(0)
        }
    })
})
