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
