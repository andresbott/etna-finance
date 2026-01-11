/**
 * Format a number as currency with proper locale formatting
 * @param amount - The amount to format
 * @param locale - The locale to use (default: 'en-US')
 * @param minimumFractionDigits - Minimum decimal places (default: 2)
 * @param maximumFractionDigits - Maximum decimal places (default: 2)
 * @returns Formatted currency string
 */
export function formatCurrency(
    amount: number,
    locale: string = 'en-US',
    minimumFractionDigits: number = 2,
    maximumFractionDigits: number = 2
): string {
    return amount.toLocaleString(locale, {
        minimumFractionDigits,
        maximumFractionDigits
    })
}

/**
 * Format amount with default settings
 * @param amount - The amount to format
 * @returns Formatted amount string
 */
export function formatAmount(amount: number): string {
    return formatCurrency(amount, 'en-US', 2, 2)
}







