import { describe, it, expect, vi, afterEach } from 'vitest'
import { ref } from 'vue'
import {
    extractAccountId,
    getFormattedAccountId,
    getDateOnly,
    toDateString,
    getSubmitValues,
    type SubmitEvent
} from './useEntryDialogForm'

describe('extractAccountId', () => {
    it('returns null for null', () => {
        expect(extractAccountId(null)).toBeNull()
    })

    it('returns null for undefined', () => {
        expect(extractAccountId(undefined)).toBeNull()
    })

    it('returns the number directly when given a number', () => {
        expect(extractAccountId(42)).toBe(42)
    })

    it('returns 0 when given 0', () => {
        expect(extractAccountId(0)).toBe(0)
    })

    it('returns null for NaN', () => {
        expect(extractAccountId(NaN)).toBeNull()
    })

    it('extracts ID from object with numeric key like { 5: true }', () => {
        expect(extractAccountId({ 5: true })).toBe(5)
    })

    it('extracts ID from object with string numeric key', () => {
        expect(extractAccountId({ '123': true })).toBe(123)
    })

    it('returns null for object with non-numeric key', () => {
        expect(extractAccountId({ abc: true })).toBeNull()
    })

    it('returns null for empty object', () => {
        expect(extractAccountId({})).toBeNull()
    })

    it('uses the first key when object has multiple keys', () => {
        // Object.keys order: integer keys come first in ascending order
        const result = extractAccountId({ 10: true, 20: true })
        expect(result).toBe(10)
    })

    it('returns null for non-number non-object types (string)', () => {
        expect(extractAccountId('hello')).toBeNull()
    })

    it('returns null for boolean', () => {
        expect(extractAccountId(true)).toBeNull()
    })

    it('handles negative numbers', () => {
        expect(extractAccountId(-1)).toBe(-1)
    })
})

describe('getFormattedAccountId', () => {
    it('returns null for null', () => {
        expect(getFormattedAccountId(null)).toBeNull()
    })

    it('returns null for undefined', () => {
        expect(getFormattedAccountId(undefined)).toBeNull()
    })

    it('returns { [id]: true } for a valid account ID', () => {
        expect(getFormattedAccountId(5)).toEqual({ 5: true })
    })

    it('returns { 0: true } for account ID 0', () => {
        expect(getFormattedAccountId(0)).toEqual({ 0: true })
    })

    it('round-trips with extractAccountId', () => {
        const id = 42
        const formatted = getFormattedAccountId(id)
        expect(extractAccountId(formatted)).toBe(id)
    })
})

describe('getDateOnly', () => {
    afterEach(() => {
        vi.restoreAllMocks()
    })

    it('strips time from a Date object', () => {
        const date = new Date(2024, 6, 15, 14, 30, 45, 123)
        const result = getDateOnly(date)
        expect(result.getFullYear()).toBe(2024)
        expect(result.getMonth()).toBe(6)
        expect(result.getDate()).toBe(15)
        expect(result.getHours()).toBe(0)
        expect(result.getMinutes()).toBe(0)
        expect(result.getSeconds()).toBe(0)
        expect(result.getMilliseconds()).toBe(0)
    })

    it('handles a date string input', () => {
        const result = getDateOnly('2023-12-25')
        const parsed = new Date('2023-12-25')
        expect(result.getFullYear()).toBe(parsed.getFullYear())
        expect(result.getMonth()).toBe(parsed.getMonth())
        expect(result.getDate()).toBe(parsed.getDate())
        expect(result.getHours()).toBe(0)
        expect(result.getMinutes()).toBe(0)
    })

    it('returns today at midnight for null', () => {
        const now = new Date(2025, 2, 10, 15, 30, 0)
        vi.useFakeTimers({ now })
        const result = getDateOnly(null)
        expect(result.getFullYear()).toBe(2025)
        expect(result.getMonth()).toBe(2)
        expect(result.getDate()).toBe(10)
        expect(result.getHours()).toBe(0)
        expect(result.getMinutes()).toBe(0)
        vi.useRealTimers()
    })

    it('returns today at midnight for undefined', () => {
        const now = new Date(2025, 2, 10, 15, 30, 0)
        vi.useFakeTimers({ now })
        const result = getDateOnly(undefined)
        expect(result.getFullYear()).toBe(2025)
        expect(result.getMonth()).toBe(2)
        expect(result.getDate()).toBe(10)
        expect(result.getHours()).toBe(0)
        vi.useRealTimers()
    })

    it('returns today at midnight for empty string', () => {
        const now = new Date(2025, 2, 10, 15, 30, 0)
        vi.useFakeTimers({ now })
        const result = getDateOnly('')
        expect(result.getFullYear()).toBe(2025)
        expect(result.getMonth()).toBe(2)
        expect(result.getDate()).toBe(10)
        vi.useRealTimers()
    })

    it('returns a new Date instance (not same reference)', () => {
        const input = new Date(2024, 0, 1)
        const result = getDateOnly(input)
        expect(result).not.toBe(input)
    })
})

describe('toDateString', () => {
    afterEach(() => {
        vi.restoreAllMocks()
    })

    it('formats a Date to YYYY-MM-DD using local parts', () => {
        const date = new Date(2024, 6, 15)
        expect(toDateString(date)).toBe('2024-07-15')
    })

    it('pads single-digit months and days', () => {
        const date = new Date(2023, 0, 5) // Jan 5
        expect(toDateString(date)).toBe('2023-01-05')
    })

    it('handles null by returning today in UTC ISO format', () => {
        const now = new Date()
        const expected = now.toISOString().slice(0, 10)
        vi.useFakeTimers({ now })
        expect(toDateString(null)).toBe(expected)
        vi.useRealTimers()
    })

    it('handles undefined by returning today', () => {
        const now = new Date()
        const expected = now.toISOString().slice(0, 10)
        vi.useFakeTimers({ now })
        expect(toDateString(undefined)).toBe(expected)
        vi.useRealTimers()
    })

    it('accepts a date string', () => {
        const date = new Date(2022, 5, 10)
        const result = toDateString(date.toISOString())
        const parsed = new Date(date.toISOString())
        const y = parsed.getFullYear()
        const m = String(parsed.getMonth() + 1).padStart(2, '0')
        const d = String(parsed.getDate()).padStart(2, '0')
        expect(result).toBe(`${y}-${m}-${d}`)
    })

    it('handles leap day', () => {
        const date = new Date(2024, 1, 29)
        expect(toDateString(date)).toBe('2024-02-29')
    })
})

describe('getSubmitValues', () => {
    it('returns event values merged over formValues ref', () => {
        const formValues = ref({ name: 'old', amount: 100 })
        const event: SubmitEvent = {
            valid: true,
            values: { name: 'new' }
        }
        const result = getSubmitValues(event, formValues)
        expect(result).toEqual({ name: 'new', amount: 100 })
    })

    it('prefers event.values over event.states', () => {
        const formValues = ref({ name: 'base' })
        const event: SubmitEvent = {
            values: { name: 'from-values' },
            states: { name: { value: 'from-states' } }
        }
        const result = getSubmitValues(event, formValues)
        expect(result.name).toBe('from-values')
    })

    it('falls back to event.states when values is undefined', () => {
        const formValues = ref({ name: 'base', amount: 50 })
        const event: SubmitEvent = {
            states: { name: { value: 'from-states' } }
        }
        const result = getSubmitValues(event, formValues)
        expect(result).toEqual({ name: 'from-states', amount: 50 })
    })

    it('returns formValues ref value when event has no values or states', () => {
        const formValues = ref({ name: 'original', amount: 200 })
        const event: SubmitEvent = { valid: true }
        const result = getSubmitValues(event, formValues)
        expect(result).toEqual({ name: 'original', amount: 200 })
    })

    it('returns formValues ref value for empty event', () => {
        const formValues = ref({ x: 1 })
        const event: SubmitEvent = {}
        const result = getSubmitValues(event, formValues)
        expect(result).toEqual({ x: 1 })
    })

    it('accepts a plain object instead of a ref', () => {
        const formValues = { name: 'plain', count: 10 }
        const event: SubmitEvent = { values: { count: 99 } }
        const result = getSubmitValues(event, formValues)
        expect(result).toEqual({ name: 'plain', count: 99 })
    })

    it('event values override all matching keys from formValues', () => {
        const formValues = ref({ a: 1, b: 2, c: 3 })
        const event: SubmitEvent = { values: { a: 10, b: 20 } }
        const result = getSubmitValues(event, formValues)
        expect(result).toEqual({ a: 10, b: 20, c: 3 })
    })

    it('handles states with undefined value property', () => {
        const formValues = ref({ name: 'fallback' })
        const event: SubmitEvent = {
            states: { name: {} }
        }
        const result = getSubmitValues(event, formValues)
        expect(result).toEqual({ name: undefined })
    })

    it('event can add new keys not present in formValues', () => {
        const formValues = ref({ a: 1 })
        const event: SubmitEvent = { values: { a: 2, b: 3 } }
        const result = getSubmitValues(event, formValues as any)
        expect(result).toEqual({ a: 2, b: 3 })
    })
})
