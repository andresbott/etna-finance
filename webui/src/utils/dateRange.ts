/**
 * Date range helpers for charts and API (start/end as local YYYY-MM-DD).
 */

import { toLocalDateString } from '@/utils/date'

export { toLocalDateString } from '@/utils/date'

export type PriceHistoryRange = '6m' | 'max'

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
        case '6m':
            start.setMonth(start.getMonth() - 6)
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
