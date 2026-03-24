import { ref, nextTick, watch } from 'vue'
import { getEntry } from '@/lib/api/Entry'

export function useEntryDialogs(deleteEntryFn: (id: string) => Promise<void>) {
    const selectedEntry = ref<Record<string, unknown> | null>(null)
    const isEditMode = ref(false)
    const isDuplicateMode = ref(false)

    const deleteDialogVisible = ref(false)
    const entryToDelete = ref<Record<string, unknown> | null>(null)
    const transformDeleteId = ref<number | null>(null)

    const dialogs = {
        incomeExpense: ref(false),
        expense: ref(false),
        income: ref(false),
        transfer: ref(false),
        buyStock: ref(false),
        sellStock: ref(false),
        grantStock: ref(false),
        transferInstrument: ref(false),
        vestStock: ref(false),
        forfeitStock: ref(false),
        balanceStatus: ref(false),
        revaluation: ref(false)
    }

    const ENTRY_TYPE_TO_DIALOG: Record<string, keyof typeof dialogs> = {
        income: 'incomeExpense',
        expense: 'incomeExpense',
        transfer: 'transfer',
        stockbuy: 'buyStock',
        stocksell: 'sellStock',
        stockgrant: 'grantStock',
        stocktransfer: 'transferInstrument',
        stockvest: 'vestStock',
        stockforfeit: 'forfeitStock',
        balancestatus: 'balanceStatus',
        revaluation: 'revaluation'
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

        // For sells/vests, list API does not include fees or full details; fetch full entry so dialog shows correct data
        if (entry.type === 'stocksell' || entry.type === 'stockvest' || entry.type === 'stockforfeit') {
            try {
                const full = await getEntry(entry.id as string)
                selectedEntry.value = full
            } catch (e) {
                console.error('Failed to load entry for edit', e)
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

    const openTransformToTransfer = async (data: {
        id: number
        date: Date
        description: string
        notes: string
        amount: number
        accountId: number
        entryType: string
    }) => {
        // Close income/expense dialog first
        dialogs.incomeExpense.value = false
        await nextTick()

        isEditMode.value = false
        isDuplicateMode.value = false
        transformDeleteId.value = data.id

        // Build selectedEntry shaped for TransferDialog props
        if (data.entryType === 'expense') {
            selectedEntry.value = {
                description: data.description,
                notes: data.notes,
                date: data.date,
                originAccountId: data.accountId,
                originAmount: data.amount,
                targetAccountId: null,
                targetAmount: 0
            }
        } else {
            selectedEntry.value = {
                description: data.description,
                notes: data.notes,
                date: data.date,
                targetAccountId: data.accountId,
                targetAmount: data.amount,
                originAccountId: null,
                originAmount: 0
            }
        }

        dialogs.transfer.value = true
    }

    // Clear transformDeleteId when transfer dialog closes (cancel or success)
    watch(dialogs.transfer, (open) => {
        if (!open) {
            transformDeleteId.value = null
        }
    })

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
        handleDeleteEntry,
        openTransformToTransfer,
        transformDeleteId
    }
}
