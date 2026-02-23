/**
 * Shared formatters for market data and change indicators (percent, severity).
 */

export function formatPct(value: number | null | undefined): string {
    if (value == null) return '-'
    const sign = value >= 0 ? '+' : ''
    return sign + value.toFixed(2) + '%'
}

/**
 * PrimeVue severity for change value: success (positive), danger (negative), secondary (zero/null).
 */
export function getChangeSeverity(value: number | null | undefined): string {
    if (value == null) return 'secondary'
    if (value > 0) return 'success'
    if (value < 0) return 'danger'
    return 'secondary'
}
