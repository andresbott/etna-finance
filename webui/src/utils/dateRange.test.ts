import { describe, it, expect, vi, afterEach } from 'vitest'
import { lastDaysRange, rangeToStartEnd } from './dateRange'

describe('dateRange utils', () => {
    afterEach(() => {
        vi.useRealTimers()
    })

    describe('lastDaysRange', () => {
        it('returns correct start/end for 7 days', () => {
            vi.useFakeTimers()
            vi.setSystemTime(new Date(2026, 2, 10, 14, 30, 0)) // March 10 2026, 14:30

            const result = lastDaysRange(7)
            expect(result).toEqual({
                start: '2026-03-03',
                end: '2026-03-10'
            })
        })

        it('returns correct start/end for 30 days', () => {
            vi.useFakeTimers()
            vi.setSystemTime(new Date(2026, 2, 10, 9, 0, 0))

            const result = lastDaysRange(30)
            expect(result).toEqual({
                start: '2026-02-08',
                end: '2026-03-10'
            })
        })

        it('returns correct range for 0 days (start equals end)', () => {
            vi.useFakeTimers()
            vi.setSystemTime(new Date(2026, 0, 15, 12, 0, 0)) // Jan 15

            const result = lastDaysRange(0)
            expect(result).toEqual({
                start: '2026-01-15',
                end: '2026-01-15'
            })
        })

        it('crosses year boundary correctly', () => {
            vi.useFakeTimers()
            vi.setSystemTime(new Date(2026, 0, 5, 10, 0, 0)) // Jan 5 2026

            const result = lastDaysRange(10)
            expect(result).toEqual({
                start: '2025-12-26',
                end: '2026-01-05'
            })
        })

        it('handles leap year boundary', () => {
            vi.useFakeTimers()
            vi.setSystemTime(new Date(2024, 2, 1, 10, 0, 0)) // March 1 2024 (leap year)

            const result = lastDaysRange(1)
            expect(result).toEqual({
                start: '2024-02-29',
                end: '2024-03-01'
            })
        })

        it('handles large number of days (365)', () => {
            vi.useFakeTimers()
            vi.setSystemTime(new Date(2026, 2, 10, 10, 0, 0))

            const result = lastDaysRange(365)
            expect(result).toEqual({
                start: '2025-03-10',
                end: '2026-03-10'
            })
        })
    })

    describe('rangeToStartEnd', () => {
        it('returns 6 months back for "6m" range', () => {
            vi.useFakeTimers()
            vi.setSystemTime(new Date(2026, 2, 10, 15, 0, 0)) // March 10 2026

            const result = rangeToStartEnd('6m')
            expect(result).toEqual({
                start: '2025-09-10',
                end: '2026-03-10'
            })
        })

        it('returns 10 years back for "max" range', () => {
            vi.useFakeTimers()
            vi.setSystemTime(new Date(2026, 2, 10, 15, 0, 0))

            const result = rangeToStartEnd('max')
            expect(result).toEqual({
                start: '2016-03-10',
                end: '2026-03-10'
            })
        })

        it('handles 6m crossing year boundary', () => {
            vi.useFakeTimers()
            vi.setSystemTime(new Date(2026, 3, 15, 10, 0, 0)) // April 15 2026

            const result = rangeToStartEnd('6m')
            expect(result).toEqual({
                start: '2025-10-15',
                end: '2026-04-15'
            })
        })

        it('handles 6m when start month has fewer days', () => {
            vi.useFakeTimers()
            // Aug 31 -> 6 months back = Feb 28 (or March depending on JS Date behavior)
            vi.setSystemTime(new Date(2026, 7, 31, 10, 0, 0)) // Aug 31 2026

            const result = rangeToStartEnd('6m')
            // JS Date: new Date(2026, 7, 31) then setMonth(1) -> Feb 31 overflows to March 3
            expect(result).toEqual({
                start: '2026-03-03',
                end: '2026-08-31'
            })
        })

        it('returns dates formatted as YYYY-MM-DD', () => {
            vi.useFakeTimers()
            vi.setSystemTime(new Date(2026, 0, 5, 10, 0, 0)) // Jan 5 2026

            const result = rangeToStartEnd('6m')
            expect(result.start).toMatch(/^\d{4}-\d{2}-\d{2}$/)
            expect(result.end).toMatch(/^\d{4}-\d{2}-\d{2}$/)
        })
    })
})
