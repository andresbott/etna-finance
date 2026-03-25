import { describe, it, expect, vi, afterEach } from 'vitest'
import { parseLocalDate, toLocalDateString } from './date'

describe('parseLocalDate', () => {
    it('parses date-only string as local midnight', () => {
        const d = parseLocalDate('2024-01-15')
        expect(d.getFullYear()).toBe(2024)
        expect(d.getMonth()).toBe(0) // January
        expect(d.getDate()).toBe(15)
    })

    it('preserves date in any timezone for date-only strings', () => {
        // The key invariant: getDate() should always match the input day
        const d = parseLocalDate('2024-03-01')
        expect(d.getDate()).toBe(1)
        expect(d.getMonth()).toBe(2) // March
    })

    it('falls back to new Date() for non-date-only strings', () => {
        const d = parseLocalDate('2024-01-15T10:30:00Z')
        expect(d.getTime()).toBe(new Date('2024-01-15T10:30:00Z').getTime())
    })
})

describe('toLocalDateString', () => {
    afterEach(() => {
        vi.restoreAllMocks()
    })

    it('formats a Date object to YYYY-MM-DD using local date parts', () => {
        // 2024-07-15 in local time
        const date = new Date(2024, 6, 15) // month is 0-indexed
        expect(toLocalDateString(date)).toBe('2024-07-15')
    })

    it('pads single-digit months', () => {
        const date = new Date(2023, 0, 20) // January
        expect(toLocalDateString(date)).toBe('2023-01-20')
    })

    it('pads single-digit days', () => {
        const date = new Date(2023, 11, 5) // December 5
        expect(toLocalDateString(date)).toBe('2023-12-05')
    })

    it('pads both single-digit month and day', () => {
        const date = new Date(2025, 1, 3) // February 3
        expect(toLocalDateString(date)).toBe('2025-02-03')
    })

    it('handles double-digit month and day without extra padding', () => {
        const date = new Date(2024, 10, 28) // November 28
        expect(toLocalDateString(date)).toBe('2024-11-28')
    })

    it('accepts a valid date string input', () => {
        // Using an explicit local date format to avoid timezone shifts
        const date = new Date(2022, 5, 10)
        const result = toLocalDateString(date.toISOString())
        // The result depends on local timezone; compare against constructing the same way
        const expected = toLocalDateString(new Date(date.toISOString()))
        expect(result).toBe(expected)
    })

    it('formats date-only string "2024-03-01" as local date without UTC shift', () => {
        // parseLocalDate detects date-only strings and parses as local midnight,
        // so the result should always be the same date regardless of timezone.
        expect(toLocalDateString('2024-03-01')).toBe('2024-03-01')
    })

    it('returns today (local) for null input', () => {
        const now = new Date()
        const expected = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
        vi.useFakeTimers({ now })
        expect(toLocalDateString(null)).toBe(expected)
        vi.useRealTimers()
    })

    it('returns today (local) for undefined input', () => {
        const now = new Date()
        const expected = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
        vi.useFakeTimers({ now })
        expect(toLocalDateString(undefined)).toBe(expected)
        vi.useRealTimers()
    })

    it('returns today (local) for invalid date string', () => {
        const now = new Date()
        const expected = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
        vi.useFakeTimers({ now })
        expect(toLocalDateString('not-a-date')).toBe(expected)
        vi.useRealTimers()
    })

    it('handles end-of-year date', () => {
        const date = new Date(2024, 11, 31) // December 31
        expect(toLocalDateString(date)).toBe('2024-12-31')
    })

    it('handles start-of-year date', () => {
        const date = new Date(2025, 0, 1) // January 1
        expect(toLocalDateString(date)).toBe('2025-01-01')
    })

    it('handles leap day', () => {
        const date = new Date(2024, 1, 29) // Feb 29 2024
        expect(toLocalDateString(date)).toBe('2024-02-29')
    })
})
