/**
 * Reads chart colors from CSS custom properties at runtime.
 * Use this in JS chart configurations (echarts, etc.) where CSS `var()` cannot be used directly.
 */

function getCssVar(name: string): string {
    return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}

/** Green color for positive values (up candles, gains, etc.) */
export function getGreenColor(): string {
    return getCssVar('--c-green-500') || '#22c55e'
}

/** Red color for negative values (down candles, losses, etc.) */
export function getRedColor(): string {
    return getCssVar('--c-red-500') || '#ef4444'
}

/** Semi-transparent green for area fills */
export function getGreenAreaColor(): string {
    return getCssVar('--c-green-500')
        ? `rgba(${hexToRgb(getCssVar('--c-green-500'))}, 0.15)`
        : 'rgba(34,197,94,0.15)'
}

/** Semi-transparent red for area fills */
export function getRedAreaColor(): string {
    return getCssVar('--c-red-500')
        ? `rgba(${hexToRgb(getCssVar('--c-red-500'))}, 0.15)`
        : 'rgba(239,68,68,0.15)'
}

/** Neutral/muted color for grid lines, marks, etc. */
export function getNeutralColor(): string {
    return getCssVar('--c-surface-400') || '#9ca3af'
}

/** Text color for axis labels */
export function getTextColor(): string {
    return getCssVar('--c-text-color') || '#495057'
}

/** Border/grid line color */
export function getSurfaceBorderColor(): string {
    return getCssVar('--c-content-border-color') || '#dfe7ef'
}

/** Convert a hex color string to "r, g, b" for use in rgba() */
function hexToRgb(hex: string): string {
    const h = hex.replace('#', '')
    const r = parseInt(h.substring(0, 2), 16)
    const g = parseInt(h.substring(2, 4), 16)
    const b = parseInt(h.substring(4, 6), 16)
    return `${r}, ${g}, ${b}`
}

/**
 * Interpolate between neutral gray and a target color based on intensity.
 * Used for treemap-style heatmaps where color saturation indicates magnitude.
 */
export function interpolateColor(
    changePct: number | null,
    maxIntensity: number = 10
): string {
    if (changePct === null || changePct === 0) return getNeutralColor()

    const ratio = Math.min(Math.abs(changePct) / maxIntensity, 1)
    const green = getGreenColor()
    const red = getRedColor()
    const target = changePct > 0 ? green : red

    const [tr, tg, tb] = hexToRgbArray(target)
    // Neutral gray base: 153
    const r = Math.round(153 + ratio * (tr - 153))
    const g = Math.round(153 + ratio * (tg - 153))
    const b = Math.round(153 + ratio * (tb - 153))
    return `rgb(${r},${g},${b})`
}

function hexToRgbArray(hex: string): [number, number, number] {
    const h = hex.replace('#', '')
    return [
        parseInt(h.substring(0, 2), 16),
        parseInt(h.substring(2, 4), 16),
        parseInt(h.substring(4, 6), 16)
    ]
}
