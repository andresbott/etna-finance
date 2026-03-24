import { describe, it, expect } from 'vitest'
import { getEntryTypeIcon } from './entryDisplay'

describe('getEntryTypeIcon', () => {
    it('returns the correct icon for each known entry type', () => {
        const expected: Record<string, string> = {
            expense: 'ti ti-minus text-red-500',
            income: 'ti ti-plus text-green-500',
            transfer: 'ti ti-arrows-left-right text-blue-500',
            stockbuy: 'ti ti-chart-line text-yellow-500',
            stocksell: 'ti ti-chart-line text-orange-500',
            stockgrant: 'ti ti-gift text-purple-500',
            stocktransfer: 'ti ti-arrows-left-right text-indigo-500',
            stockvest: 'ti ti-certificate text-teal-500',
            stockforfeit: 'ti ti-circle-x text-red-400',
            balancestatus: 'ti ti-calculator text-gray-500',
            revaluation: 'ti ti-adjustments text-cyan-500',
            'opening-balance': 'ti ti-calculator text-gray-500'
        }

        for (const [type, icon] of Object.entries(expected)) {
            expect(getEntryTypeIcon(type)).toBe(icon)
        }
    })

    it('returns fallback icon for unknown type', () => {
        expect(getEntryTypeIcon('unknown')).toBe('ti ti-help-circle')
    })

    it('returns fallback icon for empty string', () => {
        expect(getEntryTypeIcon('')).toBe('ti ti-help-circle')
    })

    it('returns fallback icon for null', () => {
        expect(getEntryTypeIcon(null)).toBe('ti ti-help-circle')
    })

    it('returns fallback icon for undefined', () => {
        expect(getEntryTypeIcon(undefined)).toBe('ti ti-help-circle')
    })
})
