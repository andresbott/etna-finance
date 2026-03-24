/**
 * Shared helpers for displaying entry types in tables and lists.
 */

const ENTRY_TYPE_ICONS: Record<string, string> = {
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

export function getEntryTypeIcon(type: string | null | undefined): string {
    return ENTRY_TYPE_ICONS[type ?? ''] ?? 'ti ti-help-circle'
}
