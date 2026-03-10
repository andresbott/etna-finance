import { describe, it, expect } from 'vitest'
import { findAccountById } from './accountUtils'
import type { Account, AccountProvider } from '@/types/account'

function makeAccount(overrides: Partial<Account> & { id: number; name: string }): Account {
    return {
        currency: 'USD',
        type: 'checking',
        ...overrides,
    }
}

function makeProvider(overrides: Partial<AccountProvider> & { id: number; name: string }): AccountProvider {
    return {
        accounts: [],
        ...overrides,
    }
}

const sampleProviders: AccountProvider[] = [
    makeProvider({
        id: 1,
        name: 'Bank A',
        accounts: [
            makeAccount({ id: 10, name: 'Checking A', currency: 'USD', type: 'checking' }),
            makeAccount({ id: 20, name: 'Savings A', currency: 'EUR', type: 'savings' }),
        ],
    }),
    makeProvider({
        id: 2,
        name: 'Bank B',
        accounts: [
            makeAccount({ id: 30, name: 'Investment B', currency: 'USD', type: 'investment' }),
        ],
    }),
    makeProvider({
        id: 3,
        name: 'Empty Bank',
        accounts: [],
    }),
]

describe('findAccountById', () => {
    it('finds an account by numeric id in the first provider', () => {
        const result = findAccountById(sampleProviders, 10)
        expect(result).not.toBeNull()
        expect(result!.name).toBe('Checking A')
    })

    it('finds an account by numeric id in a subsequent provider', () => {
        const result = findAccountById(sampleProviders, 30)
        expect(result).not.toBeNull()
        expect(result!.name).toBe('Investment B')
    })

    it('returns null when string id does not match numeric account id (strict equality)', () => {
        // account.id is 20 (number), search with '20' (string) — strict equality fails
        const result = findAccountById(sampleProviders, '20')
        expect(result).toBeNull()
    })

    it('finds account by string id when account.id is also a string', () => {
        const providers: AccountProvider[] = [
            makeProvider({
                id: 1,
                name: 'Bank',
                accounts: [{ id: '20' as any, name: 'StringId', currency: 'CHF', type: 'savings' }],
            }),
        ]
        const result = findAccountById(providers, '20')
        expect(result).not.toBeNull()
        expect(result!.name).toBe('StringId')
    })

    it('returns the full account object with all fields', () => {
        const result = findAccountById(sampleProviders, 10)
        expect(result).toEqual({
            id: 10,
            name: 'Checking A',
            currency: 'USD',
            type: 'checking',
        })
    })

    it('returns null when the id does not match any account', () => {
        expect(findAccountById(sampleProviders, 999)).toBeNull()
    })

    it('returns null when the string id does not match any account', () => {
        expect(findAccountById(sampleProviders, 'nonexistent')).toBeNull()
    })

    it('returns null for an empty providers array', () => {
        expect(findAccountById([], 10)).toBeNull()
    })

    it('skips providers with no accounts array', () => {
        const providers = [
            { id: 1, name: 'No accounts' } as unknown as AccountProvider,
            makeProvider({
                id: 2,
                name: 'Has accounts',
                accounts: [makeAccount({ id: 5, name: 'Found' })],
            }),
        ]
        expect(findAccountById(providers, 5)).not.toBeNull()
        expect(findAccountById(providers, 5)!.name).toBe('Found')
    })

    it('handles providers where accounts is explicitly undefined', () => {
        const providers = [
            { id: 1, name: 'Undef', accounts: undefined } as unknown as AccountProvider,
        ]
        expect(findAccountById(providers, 1)).toBeNull()
    })

    it('returns the first match when duplicate ids exist across providers', () => {
        const providers: AccountProvider[] = [
            makeProvider({
                id: 1,
                name: 'First',
                accounts: [makeAccount({ id: 42, name: 'First Match' })],
            }),
            makeProvider({
                id: 2,
                name: 'Second',
                accounts: [makeAccount({ id: 42, name: 'Second Match' })],
            }),
        ]
        const result = findAccountById(providers, 42)
        expect(result!.name).toBe('First Match')
    })

    it('handles id of 0', () => {
        const providers: AccountProvider[] = [
            makeProvider({
                id: 1,
                name: 'P',
                accounts: [makeAccount({ id: 0, name: 'Zero Account' })],
            }),
        ]
        const result = findAccountById(providers, 0)
        expect(result).not.toBeNull()
        expect(result!.name).toBe('Zero Account')
    })

    it('skips providers with empty accounts and finds in later providers', () => {
        const result = findAccountById(sampleProviders, 30)
        // Provider at index 2 has empty accounts, account 30 is in provider at index 1
        expect(result).not.toBeNull()
        expect(result!.name).toBe('Investment B')
    })
})
