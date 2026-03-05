/**
 * Local date formatting for API request params (YYYY-MM-DD).
 * Uses local date parts to avoid UTC shift so "today" stays today in all timezones.
 */

/**
 * Format date as local YYYY-MM-DD for API requests.
 * Prefer this over toISOString().slice(0,10) which uses UTC and can shift the calendar day.
 */
export function toLocalDateString(d: Date | string | null | undefined): string {
    if (d == null) return new Date().toISOString().slice(0, 10)
    const date = typeof d === 'string' ? new Date(d) : d
    if (!(date instanceof Date) || Number.isNaN(date.getTime())) return new Date().toISOString().slice(0, 10)
    const y = date.getFullYear()
    const m = String(date.getMonth() + 1).padStart(2, '0')
    const day = String(date.getDate()).padStart(2, '0')
    return `${y}-${m}-${day}`
}
