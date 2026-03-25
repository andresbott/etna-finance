/**
 * Local date formatting for API request params (YYYY-MM-DD).
 * Uses local date parts to avoid UTC shift so "today" stays today in all timezones.
 */

/**
 * Parse a date-only string ("2024-01-15") as local midnight instead of UTC midnight.
 * new Date("2024-01-15") parses as UTC, which shifts to the previous day in negative-offset
 * timezones (Americas). This function detects date-only strings and uses component integers.
 */
export function parseLocalDate(d: string): Date {
    const parts = d.match(/^(\d{4})-(\d{2})-(\d{2})$/)
    if (parts) {
        return new Date(Number(parts[1]), Number(parts[2]) - 1, Number(parts[3]))
    }
    return new Date(d)
}

/**
 * Format date as local YYYY-MM-DD for API requests.
 * Prefer this over toISOString().slice(0,10) which uses UTC and can shift the calendar day.
 */
export function toLocalDateString(d: Date | string | null | undefined): string {
    const date = d == null
        ? new Date()
        : typeof d === 'string' ? parseLocalDate(d) : d
    if (!(date instanceof Date) || Number.isNaN(date.getTime())) {
        const now = new Date()
        return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
    }
    const y = date.getFullYear()
    const m = String(date.getMonth() + 1).padStart(2, '0')
    const day = String(date.getDate()).padStart(2, '0')
    return `${y}-${m}-${day}`
}
