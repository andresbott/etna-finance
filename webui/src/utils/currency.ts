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
    // Treat -0 and values that would round to "-0.00" as plain zero
    const threshold = 0.5 * Math.pow(10, -maximumFractionDigits)
    const value = Object.is(amount, -0) || (amount < 0 && amount > -threshold) ? 0 : amount
    return value.toLocaleString(undefined, {
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
