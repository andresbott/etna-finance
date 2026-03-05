import { computed } from 'vue'
import { z } from 'zod'
import { useSettingsStore } from '@/store/settingsStore'

/**
 * Global date formatting for the webui.
 *
 * All user-visible date representation in the app should use this composable so that
 * the format follows the global setting (Settings → dateFormat). Use `formatDate(date)`
 * for display (Date or ISO date string). Use `pickerDateFormat` for PrimeVue date
 * pickers. Do not introduce locale-specific or hardcoded date formats for display.
 */

/**
 * Convert a backend date-format string (DD/MM/YYYY etc.) to PrimeVue DatePicker format.
 * Backend tokens: YYYY, YY, MM, DD  –  separators: - / .
 * PrimeVue tokens: dd (day), mm (month), yy (4-digit year), y (2-digit year)
 */
function toPrimeVueDateFormat(backendFormat) {
    if (!backendFormat) return 'dd/mm/yy'
    return backendFormat
        .replace('YYYY', 'yy')
        .replace('YY', 'y')
        .replace('DD', 'dd')
        .replace('MM', 'mm')
}

/**
 * Format a Date (or ISO date string) for display using the given format pattern.
 */
function formatDisplayDate(date, format) {
    if (!date) return ''
    const d = new Date(date)
    if (isNaN(d.getTime())) return String(date)

    const day = String(d.getDate()).padStart(2, '0')
    const month = String(d.getMonth() + 1).padStart(2, '0')
    const yearFull = String(d.getFullYear())
    const yearShort = yearFull.slice(-2)

    if (!format) format = 'DD/MM/YYYY'

    return format
        .replace('YYYY', yearFull)
        .replace('YY', yearShort)
        .replace('DD', day)
        .replace('MM', month)
}

/**
 * Format time part as HH:mm (24h) from a Date or ISO string.
 */
function formatTime(date) {
    if (!date) return ''
    const d = new Date(date)
    if (isNaN(d.getTime())) return ''
    const h = String(d.getHours()).padStart(2, '0')
    const m = String(d.getMinutes()).padStart(2, '0')
    return `${h}:${m}`
}

/**
 * Composable that provides date formatting helpers driven by the settings store.
 *
 * - `formatDate(date)` – formats a Date / ISO-string for display (date only).
 * - `formatDateTime(date)` – date and time (HH:mm) for display.
 * - `pickerDateFormat` – computed PrimeVue DatePicker format string.
 */
export function useDateFormat() {
    const settings = useSettingsStore()

    const pickerDateFormat = computed(() => toPrimeVueDateFormat(settings.dateFormat))

    const formatDate = (date) => formatDisplayDate(date, settings.dateFormat)

    const formatDateTime = (date) => {
        const datePart = formatDisplayDate(date, settings.dateFormat)
        const timePart = formatTime(date)
        return datePart && timePart ? `${datePart} ${timePart}` : datePart || timePart || ''
    }

    const dateValidation = computed(() =>
        z.coerce.date({ invalid_type_error: `Enter a valid date (${settings.dateFormat || 'DD/MM/YYYY'})` })
    )

    return {
        pickerDateFormat,
        formatDate,
        formatDateTime,
        dateValidation
    }
}
