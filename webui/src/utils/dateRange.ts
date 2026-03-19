/**
 * Date range helpers for charts and API (start/end as local YYYY-MM-DD).
 */

import { toLocalDateString } from '@/utils/date'

export { toLocalDateString } from '@/utils/date'

export type PriceHistoryRange = '7d' | '1m' | '3m' | '6m' | '1y' | 'max'

export function lastDaysRange(days: number): { start: string; end: string } {
    const end = new Date()
    end.setHours(0, 0, 0, 0)
    const start = new Date(end)
    start.setDate(start.getDate() - days)
    return {
        start: toLocalDateString(start),
        end: toLocalDateString(end)
    }
}

export function rangeToStartEnd(range: PriceHistoryRange): { start: string; end: string } {
    const end = new Date()
    end.setHours(0, 0, 0, 0)
    const start = new Date(end)
    switch (range) {
        case '7d':
            start.setDate(start.getDate() - 7)
            break
        case '1m':
            start.setMonth(start.getMonth() - 1)
            break
        case '3m':
            start.setMonth(start.getMonth() - 3)
            break
        case '6m':
            start.setMonth(start.getMonth() - 6)
            break
        case '1y':
            start.setFullYear(start.getFullYear() - 1)
            break
        case 'max':
            start.setFullYear(start.getFullYear() - 10)
            break
    }
    return {
        start: toLocalDateString(start),
        end: toLocalDateString(end)
    }
}
