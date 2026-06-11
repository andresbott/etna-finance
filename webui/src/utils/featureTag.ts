// Display helpers for optional feature settings on the configuration page.
//
// The backend keeps a feature's *effective* state as a boolean (used for gating
// throughout the app) and separately reports which features it auto-enabled at
// startup (config said off, but data on disk required them). This module derives a
// display-only ternary from those two inputs — it does not affect any gating logic.

export type FeatureState = 'on' | 'auto-enabled' | 'off'

/** Derive the display state of a feature from its effective flag and the auto-enabled list. */
export function featureState(enabled: boolean, key: string, autoEnabled: string[]): FeatureState {
    if (!enabled) return 'off'
    return autoEnabled.includes(key) ? 'auto-enabled' : 'on'
}

export interface FeatureTag {
    value: string
    severity: 'success' | 'warn' | 'secondary'
}

/** Map a feature's display state to a PrimeVue Tag's label and severity. */
export function featureTag(enabled: boolean, key: string, autoEnabled: string[]): FeatureTag {
    switch (featureState(enabled, key, autoEnabled)) {
        case 'auto-enabled':
            return { value: 'Auto-enabled', severity: 'warn' }
        case 'on':
            return { value: 'Enabled', severity: 'success' }
        case 'off':
            return { value: 'Disabled', severity: 'secondary' }
    }
}
