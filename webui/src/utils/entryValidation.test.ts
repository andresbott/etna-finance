import { describe, it, expect } from 'vitest'
import { accountValidation } from './entryValidation'

describe('accountValidation', () => {
    it('rejects null with "Account must be selected"', () => {
        const result = accountValidation.safeParse(null)
        expect(result.success).toBe(false)
        if (!result.success) {
            expect(result.error.issues[0].message).toBe('Account must be selected')
        }
    })

    it('accepts a record with a boolean value', () => {
        const result = accountValidation.safeParse({ '123': true })
        expect(result.success).toBe(true)
    })

    it('accepts a record with multiple entries', () => {
        const result = accountValidation.safeParse({ '1': true, '2': false })
        expect(result.success).toBe(true)
    })

    it('accepts an empty record', () => {
        const result = accountValidation.safeParse({})
        expect(result.success).toBe(true)
    })

    it('rejects undefined', () => {
        const result = accountValidation.safeParse(undefined)
        expect(result.success).toBe(false)
    })

    it('rejects a string', () => {
        const result = accountValidation.safeParse('account')
        expect(result.success).toBe(false)
    })

    it('rejects a number', () => {
        const result = accountValidation.safeParse(42)
        expect(result.success).toBe(false)
    })
})
