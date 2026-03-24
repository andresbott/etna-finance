import { describe, it, expect, vi, beforeEach } from 'vitest'
import { nextTick } from 'vue'
import { useEntryDialogs } from './useEntryDialogs'

vi.mock('@/lib/api/Entry', () => ({
    getEntry: vi.fn()
}))

describe('useEntryDialogs', () => {
    const deleteEntryFn = vi.fn(() => Promise.resolve())
    let result: ReturnType<typeof useEntryDialogs>

    beforeEach(() => {
        vi.clearAllMocks()
        result = useEntryDialogs(deleteEntryFn)
    })

    describe('openTransformToTransfer', () => {
        const expenseData = {
            id: 42,
            date: new Date(2026, 2, 22),
            description: 'Credit card payment',
            notes: 'Monthly bill',
            amount: 150.50,
            accountId: 7,
            entryType: 'expense'
        }

        const incomeData = {
            id: 99,
            date: new Date(2026, 2, 15),
            description: 'Refund received',
            notes: '',
            amount: 200,
            accountId: 3,
            entryType: 'income'
        }

        it('closes the income/expense dialog', async () => {
            result.dialogs.incomeExpense.value = true
            await result.openTransformToTransfer(expenseData)
            expect(result.dialogs.incomeExpense.value).toBe(false)
        })

        it('opens the transfer dialog', async () => {
            expect(result.dialogs.transfer.value).toBe(false)
            await result.openTransformToTransfer(expenseData)
            expect(result.dialogs.transfer.value).toBe(true)
        })

        it('sets isEditMode to false', async () => {
            result.isEditMode.value = true
            await result.openTransformToTransfer(expenseData)
            expect(result.isEditMode.value).toBe(false)
        })

        it('sets isDuplicateMode to false', async () => {
            result.isDuplicateMode.value = true
            await result.openTransformToTransfer(expenseData)
            expect(result.isDuplicateMode.value).toBe(false)
        })

        it('stores the entry id in transformDeleteId', async () => {
            await result.openTransformToTransfer(expenseData)
            expect(result.transformDeleteId.value).toBe(42)
        })

        it('maps expense to origin side of transfer', async () => {
            await result.openTransformToTransfer(expenseData)
            const entry = result.selectedEntry.value!
            expect(entry.originAccountId).toBe(7)
            expect(entry.originAmount).toBe(150.50)
            expect(entry.targetAccountId).toBeNull()
            expect(entry.targetAmount).toBe(0)
        })

        it('maps income to target side of transfer', async () => {
            await result.openTransformToTransfer(incomeData)
            const entry = result.selectedEntry.value!
            expect(entry.targetAccountId).toBe(3)
            expect(entry.targetAmount).toBe(200)
            expect(entry.originAccountId).toBeNull()
            expect(entry.originAmount).toBe(0)
        })

        it('carries over description, notes and date', async () => {
            await result.openTransformToTransfer(expenseData)
            const entry = result.selectedEntry.value!
            expect(entry.description).toBe('Credit card payment')
            expect(entry.notes).toBe('Monthly bill')
            expect(entry.date).toEqual(new Date(2026, 2, 22))
        })
    })

    describe('openEditEntryDialog', () => {
        it('opens the correct dialog for stockvest', async () => {
            await result.openEditEntryDialog({ id: '1', type: 'stockvest' })
            expect(result.dialogs.vestStock.value).toBe(true)
            expect(result.isEditMode.value).toBe(true)
        })

        it('opens the correct dialog for stockforfeit', async () => {
            await result.openEditEntryDialog({ id: '2', type: 'stockforfeit' })
            expect(result.dialogs.forfeitStock.value).toBe(true)
            expect(result.isEditMode.value).toBe(true)
        })

        it('fetches full entry for stockvest edits', async () => {
            const { getEntry } = await import('@/lib/api/Entry')
            const mockEntry = { id: '1', type: 'stockvest', vestingPrice: 75, lotAllocations: [{ lotId: 1, quantity: 50 }] }
            vi.mocked(getEntry).mockResolvedValue(mockEntry)

            await result.openEditEntryDialog({ id: '1', type: 'stockvest' })
            expect(getEntry).toHaveBeenCalledWith('1')
            expect(result.selectedEntry.value).toEqual(mockEntry)
        })

        it('fetches full entry for stockforfeit edits', async () => {
            const { getEntry } = await import('@/lib/api/Entry')
            const mockEntry = { id: '2', type: 'stockforfeit', lotAllocations: [{ lotId: 3, quantity: 40 }] }
            vi.mocked(getEntry).mockResolvedValue(mockEntry)

            await result.openEditEntryDialog({ id: '2', type: 'stockforfeit' })
            expect(getEntry).toHaveBeenCalledWith('2')
            expect(result.selectedEntry.value).toEqual(mockEntry)
        })

        it('does NOT fetch full entry for income edits', async () => {
            const { getEntry } = await import('@/lib/api/Entry')
            vi.mocked(getEntry).mockClear()

            await result.openEditEntryDialog({ id: '3', type: 'income' })
            expect(getEntry).not.toHaveBeenCalled()
            expect(result.dialogs.incomeExpense.value).toBe(true)
        })

        it('maps all entry types to correct dialogs', async () => {
            const typeToDialog: Record<string, string> = {
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
                revaluation: 'revaluation',
            }
            for (const [type, dialogKey] of Object.entries(typeToDialog)) {
                // Reset all dialogs
                for (const d of Object.values(result.dialogs)) d.value = false
                await result.openEditEntryDialog({ id: '1', type })
                expect((result.dialogs as Record<string, { value: boolean }>)[dialogKey].value, `${type} should open ${dialogKey}`).toBe(true)
            }
        })
    })

    describe('openDuplicateEntryDialog', () => {
        it('opens the correct dialog for stockvest in duplicate mode', async () => {
            await result.openDuplicateEntryDialog({ id: '1', type: 'stockvest' })
            expect(result.dialogs.vestStock.value).toBe(true)
            expect(result.isDuplicateMode.value).toBe(true)
            expect(result.isEditMode.value).toBe(false)
        })

        it('opens the correct dialog for stockforfeit in duplicate mode', async () => {
            await result.openDuplicateEntryDialog({ id: '2', type: 'stockforfeit' })
            expect(result.dialogs.forfeitStock.value).toBe(true)
            expect(result.isDuplicateMode.value).toBe(true)
        })
    })

    describe('transformDeleteId watcher', () => {
        it('clears transformDeleteId when transfer dialog closes', async () => {
            await result.openTransformToTransfer({
                id: 10,
                date: new Date(),
                description: 'test',
                notes: '',
                amount: 50,
                accountId: 1,
                entryType: 'expense'
            })
            expect(result.transformDeleteId.value).toBe(10)

            result.dialogs.transfer.value = false
            await nextTick()
            expect(result.transformDeleteId.value).toBeNull()
        })

        it('does not set transformDeleteId when transfer dialog opens normally', async () => {
            result.dialogs.transfer.value = true
            await nextTick()
            expect(result.transformDeleteId.value).toBeNull()
        })
    })
})
