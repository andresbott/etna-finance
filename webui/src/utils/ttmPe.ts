import type { EpsRecord } from '@/lib/api/MarketData'

/**
 * computeTtmPe returns a trailing-twelve-month P/E value per trading date, or null where it cannot
 * be computed. For each date it sums the basic EPS of the four most recent filings with
 * time <= date (TTM) and divides the close by that sum.
 *
 * Returns null for a date when: fewer than 4 filings precede it, the TTM EPS is <= 0, or the close
 * is missing/<= 0. The result is always the same length as `dates` so it stays aligned with the
 * price series for charting.
 *
 * Dates and EPS times must both be `YYYY-MM-DD` so lexicographic string comparison is correct.
 */
export function computeTtmPe(
    dates: string[],
    closes: number[],
    eps: EpsRecord[]
): (number | null)[] {
    if (dates.length === 0) return []
    if (eps.length === 0) return dates.map(() => null)

    const sorted = [...eps].sort((a, b) => a.time.localeCompare(b.time))

    return dates.map((dateStr, i) => {
        const close = closes[i]
        if (!Number.isFinite(close) || close <= 0) return null
        const applicable = sorted.filter(p => p.time <= dateStr)
        if (applicable.length < 4) return null
        const last4 = applicable.slice(-4)
        const ttmEps = last4.reduce((sum, p) => sum + p.eps_basic, 0)
        if (ttmEps <= 0) return null
        return close / ttmEps
    })
}
