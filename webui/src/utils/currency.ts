/**
 * Format a number as currency with proper locale formatting
 * @param amount - The amount to format
 * @param minimumFractionDigits - Minimum decimal places (default: 2)
 * @param maximumFractionDigits - Maximum decimal places (default: 2)
 * @returns Formatted currency string
 */
export function formatCurrency(
    amount: number,
    minimumFractionDigits: number = 2,
    maximumFractionDigits: number = 2
): string {
    return amount.toLocaleString(undefined, {
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
    return formatCurrency(amount, 2, 2)
}
