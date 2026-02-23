import { ref, type Ref } from 'vue'
import { toLocalDateString } from '@/utils/date'

/**
 * Shared form helpers for entry dialogs.
 * Use formValues ref as source of truth on submit (PrimeVue Form submit event may omit values).
 */

export type SubmitEvent = {
    valid?: boolean
    values?: Record<string, unknown>
    states?: Record<string, { value?: unknown }>
    preventDefault?: () => void
}

/**
 * Extract numeric account ID from form value (AccountSelector can use { [id]: true } or number).
 */
export function extractAccountId(formValue: unknown): number | null {
    if (formValue == null) return null
    if (typeof formValue === 'number') return Number.isNaN(formValue) ? null : formValue
    if (typeof formValue === 'object') {
        const keys = Object.keys(formValue as Record<string, unknown>)
        if (keys.length > 0) {
            const n = parseInt(keys[0], 10)
            return Number.isNaN(n) ? null : n
        }
    }
    return null
}

/**
 * Format account ID for form initialValues (AccountSelector expects { [id]: true } or null).
 */
export function getFormattedAccountId(accountId: number | null | undefined): Record<number, boolean> | null {
    if (accountId == null) return null
    return { [accountId]: true }
}

/**
 * Strip time from date for date-only display and API (YYYY-MM-DD).
 */
export function getDateOnly(date: Date | string | null | undefined): Date {
    if (!date) return new Date(new Date().setHours(0, 0, 0, 0))
    const d = new Date(date)
    return new Date(d.getFullYear(), d.getMonth(), d.getDate())
}

/**
 * Format date for API as YYYY-MM-DD (local date to avoid UTC shift).
 */
export function toDateString(date: Date | string | null | undefined): string {
    return toLocalDateString(date)
}

/**
 * Get merged form values from submit event and formValues ref.
 * Prefers event.values, then event.states, then formValues so Save works even when PrimeVue omits values.
 */
export function getSubmitValues<T extends Record<string, unknown>>(
    e: SubmitEvent,
    formValues: Ref<T> | T
): T {
    const refVal = typeof formValues === 'object' && formValues !== null && 'value' in formValues
        ? (formValues as Ref<T>).value
        : (formValues as T)

    const fromEvent =
        e.values ??
        (e.states && Object.fromEntries(
            Object.entries(e.states).map(([k, s]) => [k, (s as { value?: unknown })?.value])
        ))

    if (!fromEvent || typeof fromEvent !== 'object') return refVal
    return { ...refVal, ...fromEvent } as T
}

/**
 * Composable for entry dialog form: shared helpers + getSubmitValues bound to a formValues ref.
 */
export function useEntryDialogForm<T extends Record<string, unknown>>(formValues: Ref<T>) {
    return {
        formValues,
        extractAccountId,
        getFormattedAccountId,
        getDateOnly,
        toDateString,
        getSubmitValues: (e: SubmitEvent) => getSubmitValues(e, formValues)
    }
}
