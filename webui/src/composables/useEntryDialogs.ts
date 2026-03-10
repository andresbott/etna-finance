import { ref } from 'vue'
import { getEntry } from '@/lib/api/Entry'

export function useEntryDialogs(deleteEntryFn: (id: string) => Promise<void>) {
    const selectedEntry = ref<Record<string, unknown> | null>(null)
    const isEditMode = ref(false)
    const isDuplicateMode = ref(false)

    const deleteDialogVisible = ref(false)
    const entryToDelete = ref<Record<string, unknown> | null>(null)

    const dialogs = {
        incomeExpense: ref(false),
        expense: ref(false),
        income: ref(false),
        transfer: ref(false),
        buyStock: ref(false),
        sellStock: ref(false),
        grantStock: ref(false),
        transferInstrument: ref(false),
        balanceStatus: ref(false)
    }

    const ENTRY_TYPE_TO_DIALOG: Record<string, keyof typeof dialogs> = {
        income: 'incomeExpense',
        expense: 'incomeExpense',
        transfer: 'transfer',
        stockbuy: 'buyStock',
        stocksell: 'sellStock',
        stockgrant: 'grantStock',
        stocktransfer: 'transferInstrument',
        balancestatus: 'balanceStatus'
    }

    function openDialogForType(type: string) {
        const dialogKey = ENTRY_TYPE_TO_DIALOG[type]
        if (dialogKey) {
            dialogs[dialogKey].value = true
        }
    }

    const openEditEntryDialog = async (entry: Record<string, unknown>) => {
        isEditMode.value = true
        isDuplicateMode.value = false

        // For sells, list API does not include fees; fetch full entry so dialog shows correct net + fees
        if (entry.type === 'stocksell') {
            try {
                const full = await getEntry(entry.id as string)
                selectedEntry.value = full
            } catch (e) {
                console.error('Failed to load sell entry for edit', e)
                selectedEntry.value = entry
            }
        } else {
            selectedEntry.value = entry
        }

        openDialogForType(entry.type as string)
    }

    const openDuplicateEntryDialog = (entry: Record<string, unknown>) => {
        isEditMode.value = false
        isDuplicateMode.value = true
        selectedEntry.value = entry
        openDialogForType(entry.type as string)
    }

    const openDeleteDialog = (entry: Record<string, unknown>) => {
        entryToDelete.value = entry
        deleteDialogVisible.value = true
    }

    const handleDeleteEntry = async () => {
        try {
            await deleteEntryFn(entryToDelete.value?.id as string)
            deleteDialogVisible.value = false
        } catch (error) {
            console.error('Failed to delete entry:', error)
        }
    }

    return {
        selectedEntry,
        isEditMode,
        isDuplicateMode,
        dialogs,
        deleteDialogVisible,
        entryToDelete,
        openEditEntryDialog,
        openDuplicateEntryDialog,
        openDeleteDialog,
        handleDeleteEntry
    }
}
