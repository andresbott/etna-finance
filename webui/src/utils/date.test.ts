import { describe, it, expect, vi, afterEach } from 'vitest'
import { toLocalDateString } from './date'

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

    it('formats string input "2024-03-01" correctly', () => {
        // Note: new Date("2024-03-01") is parsed as UTC midnight, which may
        // shift to the previous day in negative-offset timezones.
        // We verify the function produces the same result as manually constructing.
        const result = toLocalDateString('2024-03-01')
        const parsed = new Date('2024-03-01')
        const y = parsed.getFullYear()
        const m = String(parsed.getMonth() + 1).padStart(2, '0')
        const d = String(parsed.getDate()).padStart(2, '0')
        expect(result).toBe(`${y}-${m}-${d}`)
    })

    it('returns today (UTC) for null input', () => {
        const now = new Date()
        const expected = now.toISOString().slice(0, 10)
        // Pin time to avoid midnight race
        vi.useFakeTimers({ now })
        expect(toLocalDateString(null)).toBe(expected)
        vi.useRealTimers()
    })

    it('returns today (UTC) for undefined input', () => {
        const now = new Date()
        const expected = now.toISOString().slice(0, 10)
        vi.useFakeTimers({ now })
        expect(toLocalDateString(undefined)).toBe(expected)
        vi.useRealTimers()
    })

    it('returns today (UTC) for invalid date string', () => {
        const now = new Date()
        const expected = now.toISOString().slice(0, 10)
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
