import { describe, it, expect } from 'vitest'
import { getEntryTypeIcon } from './entryDisplay'

describe('getEntryTypeIcon', () => {
    it('returns the correct icon for each known entry type', () => {
        const expected: Record<string, string> = {
            expense: 'pi pi-minus text-red-500',
            income: 'pi pi-plus text-green-500',
            transfer: 'pi pi-arrow-right-arrow-left text-blue-500',
            stockbuy: 'pi pi-chart-line text-yellow-500',
            stocksell: 'pi pi-chart-line text-orange-500',
            stockgrant: 'pi pi-gift text-purple-500',
            stocktransfer: 'pi pi-arrow-right-arrow-left text-indigo-500',
            balancestatus: 'pi pi-calculator text-gray-500',
            'opening-balance': 'pi pi-calculator text-gray-500'
        }

        for (const [type, icon] of Object.entries(expected)) {
            expect(getEntryTypeIcon(type)).toBe(icon)
        }
    })

    it('returns fallback icon for unknown type', () => {
        expect(getEntryTypeIcon('unknown')).toBe('pi pi-question-circle')
    })

    it('returns fallback icon for empty string', () => {
        expect(getEntryTypeIcon('')).toBe('pi pi-question-circle')
    })

    it('returns fallback icon for null', () => {
        expect(getEntryTypeIcon(null)).toBe('pi pi-question-circle')
    })

    it('returns fallback icon for undefined', () => {
        expect(getEntryTypeIcon(undefined)).toBe('pi pi-question-circle')
    })
})
