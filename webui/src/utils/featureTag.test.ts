import { describe, it, expect } from 'vitest'
import { featureState, featureTag } from './featureTag'

describe('featureState', () => {
    it('is off when the feature is disabled, regardless of the auto-enabled list', () => {
        expect(featureState(false, 'rsu', [])).toBe('off')
        expect(featureState(false, 'rsu', ['rsu'])).toBe('off')
    })

    it('is auto-enabled when on and listed as auto-enabled', () => {
        expect(featureState(true, 'rsu', ['rsu', 'investmentInstruments'])).toBe('auto-enabled')
    })

    it('is on when enabled but not auto-enabled (configured on)', () => {
        expect(featureState(true, 'financialSimulator', ['rsu'])).toBe('on')
    })
})

describe('featureTag', () => {
    it('maps auto-enabled to an amber "Auto-enabled" tag', () => {
        expect(featureTag(true, 'rsu', ['rsu'])).toEqual({ value: 'Auto-enabled', severity: 'warn' })
    })

    it('maps a configured-on feature to a green "Enabled" tag', () => {
        expect(featureTag(true, 'rsu', [])).toEqual({ value: 'Enabled', severity: 'success' })
    })

    it('maps a disabled feature to a grey "Disabled" tag', () => {
        expect(featureTag(false, 'rsu', [])).toEqual({ value: 'Disabled', severity: 'secondary' })
    })
})
