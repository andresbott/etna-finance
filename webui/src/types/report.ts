/** One point of an account's balance history, as returned by GET /fin/report/balance. */
export interface BalancePoint {
    date: string
    /** Balance in the account's own currency. */
    nativeBalance: number
    /** Balance converted to the main currency; equals nativeBalance when no rate was available. */
    mainCurrencyBalance: number
    /** True when an exchange rate (or, for investment accounts, a price) was missing for this date. */
    hasUnconvertedAmounts: boolean
    /** Number of entries summed for this step (0 for investment accounts). */
    entryCount: number
}

export interface BalanceReport {
    accounts: Record<number, BalancePoint[]>
}
