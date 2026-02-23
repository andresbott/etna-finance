/**
 * Shared helpers for displaying entry types in tables and lists.
 */

const ENTRY_TYPE_ICONS: Record<string, string> = {
    expense: 'pi pi-minus text-red-500',
    income: 'pi pi-plus text-green-500',
    transfer: 'pi pi-arrow-right-arrow-left text-blue-500',
    stockbuy: 'pi pi-chart-line text-yellow-500',
    stocksell: 'pi pi-chart-line text-orange-500',
    stockgrant: 'pi pi-gift text-purple-500',
    stocktransfer: 'pi pi-arrow-right-arrow-left text-indigo-500',
    'opening-balance': 'pi pi-calculator text-gray-500'
}

export function getEntryTypeIcon(type: string | null | undefined): string {
    return ENTRY_TYPE_ICONS[type ?? ''] ?? 'pi pi-question-circle'
}
