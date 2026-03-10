import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import {
    getIncomeCategories,
    createIncomeCategory,
    updateIncomeCategory,
    deleteIncomeCategory,
    getExpenseCategories,
    createExpenseCategory,
    updateExpenseCategory,
    deleteExpenseCategory,
} from '../lib/api/Category'
import { renderComposable, createTestQueryClient } from '../test/helpers'
import { useCategories } from './useCategories'
import type { QueryClient } from '@tanstack/vue-query'

vi.mock('../lib/api/Category', () => ({
    getIncomeCategories: vi.fn(),
    createIncomeCategory: vi.fn(),
    updateIncomeCategory: vi.fn(),
    deleteIncomeCategory: vi.fn(),
    getExpenseCategories: vi.fn(),
    createExpenseCategory: vi.fn(),
    updateExpenseCategory: vi.fn(),
    deleteExpenseCategory: vi.fn(),
}))

const mockedgetIncomeCategories = getIncomeCategories as ReturnType<typeof vi.fn>
const mockedcreateIncomeCategory = createIncomeCategory as ReturnType<typeof vi.fn>
const mockedupdateIncomeCategory = updateIncomeCategory as ReturnType<typeof vi.fn>
const mockedDeleteIncomeCategory = deleteIncomeCategory as ReturnType<typeof vi.fn>
const mockedgetExpenseCategories = getExpenseCategories as ReturnType<typeof vi.fn>
const mockedcreateExpenseCategory = createExpenseCategory as ReturnType<typeof vi.fn>
const mockedupdateExpenseCategory = updateExpenseCategory as ReturnType<typeof vi.fn>
const mockeddeleteExpenseCategory = deleteExpenseCategory as ReturnType<typeof vi.fn>

describe('useCategories', () => {
    let unmount: () => void
    let queryClient: QueryClient

    beforeEach(() => {
        vi.clearAllMocks()
        queryClient = createTestQueryClient()
    })

    afterEach(() => {
        unmount()
    })

    // ================  INCOME QUERIES  ================

    describe('income categories query', () => {
        it('fetches income categories on mount', async () => {
            const categories = [
                { id: '1', name: 'Salary' },
                { id: '2', name: 'Freelance' },
            ]
            mockedgetIncomeCategories.mockResolvedValue(categories)

            const { result } = renderComposable(() => useCategories(), { queryClient })
            unmount = renderComposable(() => useCategories(), { queryClient }).unmount
            // Re-render properly
            unmount()
            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount

            await flushPromises()

            expect(mockedgetIncomeCategories).toHaveBeenCalled()
            expect(rendered.result.incomeCategories.data.value).toEqual(categories)
        })

        it('exposes loading state while fetching income categories', () => {
            mockedgetIncomeCategories.mockReturnValue(new Promise(() => {})) // never resolves

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount

            expect(rendered.result.incomeCategories.isLoading.value).toBe(true)
        })

        it('exposes error when income categories fetch fails', async () => {
            mockedgetIncomeCategories.mockRejectedValue(new Error('Network error'))

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount

            await flushPromises()

            expect(rendered.result.incomeCategories.isError.value).toBe(true)
            expect(rendered.result.incomeCategories.error.value).toBeInstanceOf(Error)
        })
    })

    // ================  INCOME MUTATIONS  ================

    describe('createIncomeCategory mutation', () => {
        it('calls createIncomeCategory with the payload', async () => {
            mockedgetIncomeCategories.mockResolvedValue([])
            const newCategory = { id: '3', name: 'Bonus' }
            mockedcreateIncomeCategory.mockResolvedValue(newCategory)

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount
            await flushPromises()

            const payload = { name: 'Bonus' }
            rendered.result.createIncomeCategory.mutate(payload)
            await flushPromises()

            expect(mockedcreateIncomeCategory).toHaveBeenCalledWith(payload)
        })

        it('invalidates incomeCategories query on success', async () => {
            mockedgetIncomeCategories.mockResolvedValue([])
            mockedcreateIncomeCategory.mockResolvedValue({ id: '3', name: 'Bonus' })
            const spy = vi.spyOn(queryClient, 'invalidateQueries')

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount
            await flushPromises()

            rendered.result.createIncomeCategory.mutate({ name: 'Bonus' })
            await flushPromises()

            expect(spy).toHaveBeenCalledWith({ queryKey: ['incomeCategories'] })
        })
    })

    describe('updateIncomeMutation', () => {
        it('calls updateIncomeCategory with id and payload', async () => {
            mockedgetIncomeCategories.mockResolvedValue([])
            mockedupdateIncomeCategory.mockResolvedValue({ id: '1', name: 'Updated' })

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount
            await flushPromises()

            rendered.result.updateIncomeMutation.mutate({ id: 1, payload: { name: 'Updated' } })
            await flushPromises()

            expect(mockedupdateIncomeCategory).toHaveBeenCalledWith({ id: 1, payload: { name: 'Updated' } })
        })

        it('invalidates incomeCategories query on success', async () => {
            mockedgetIncomeCategories.mockResolvedValue([])
            mockedupdateIncomeCategory.mockResolvedValue({ id: '1', name: 'Updated' })
            const spy = vi.spyOn(queryClient, 'invalidateQueries')

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount
            await flushPromises()

            rendered.result.updateIncomeMutation.mutate({ id: 1, payload: { name: 'Updated' } })
            await flushPromises()

            expect(spy).toHaveBeenCalledWith({ queryKey: ['incomeCategories'] })
        })
    })

    describe('deleteIncomeMutation', () => {
        it('calls deleteIncomeCategory with the id', async () => {
            mockedgetIncomeCategories.mockResolvedValue([])
            mockedDeleteIncomeCategory.mockResolvedValue(undefined)

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount
            await flushPromises()

            rendered.result.deleteIncomeMutation.mutate(5)
            await flushPromises()

            expect(mockedDeleteIncomeCategory).toHaveBeenCalledWith(5)
        })

        it('invalidates incomeCategories query on success', async () => {
            mockedgetIncomeCategories.mockResolvedValue([])
            mockedDeleteIncomeCategory.mockResolvedValue(undefined)
            const spy = vi.spyOn(queryClient, 'invalidateQueries')

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount
            await flushPromises()

            rendered.result.deleteIncomeMutation.mutate(5)
            await flushPromises()

            expect(spy).toHaveBeenCalledWith({ queryKey: ['incomeCategories'] })
        })
    })

    // ================  EXPENSE QUERIES  ================

    describe('expense categories query', () => {
        it('fetches expense categories on mount', async () => {
            const categories = [
                { id: '10', name: 'Rent' },
                { id: '11', name: 'Food' },
            ]
            mockedgetExpenseCategories.mockResolvedValue(categories)
            mockedgetIncomeCategories.mockResolvedValue([])

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount

            await flushPromises()

            expect(mockedgetExpenseCategories).toHaveBeenCalled()
            expect(rendered.result.expenseCategories.data.value).toEqual(categories)
        })

        it('exposes loading state while fetching expense categories', () => {
            mockedgetExpenseCategories.mockReturnValue(new Promise(() => {}))
            mockedgetIncomeCategories.mockReturnValue(new Promise(() => {}))

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount

            expect(rendered.result.expenseCategories.isLoading.value).toBe(true)
        })

        it('exposes error when expense categories fetch fails', async () => {
            mockedgetExpenseCategories.mockRejectedValue(new Error('Server error'))
            mockedgetIncomeCategories.mockResolvedValue([])

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount

            await flushPromises()

            expect(rendered.result.expenseCategories.isError.value).toBe(true)
        })
    })

    // ================  EXPENSE MUTATIONS  ================

    describe('createExpenseMutation', () => {
        it('calls createExpenseCategory with the payload', async () => {
            mockedgetIncomeCategories.mockResolvedValue([])
            mockedgetExpenseCategories.mockResolvedValue([])
            mockedcreateExpenseCategory.mockResolvedValue({ id: '12', name: 'Transport' })

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount
            await flushPromises()

            const payload = { name: 'Transport' }
            rendered.result.createExpenseMutation.mutate(payload)
            await flushPromises()

            expect(mockedcreateExpenseCategory).toHaveBeenCalledWith(payload)
        })

        it('invalidates expenseCategory query on success', async () => {
            mockedgetIncomeCategories.mockResolvedValue([])
            mockedgetExpenseCategories.mockResolvedValue([])
            mockedcreateExpenseCategory.mockResolvedValue({ id: '12', name: 'Transport' })
            const spy = vi.spyOn(queryClient, 'invalidateQueries')

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount
            await flushPromises()

            rendered.result.createExpenseMutation.mutate({ name: 'Transport' })
            await flushPromises()

            expect(spy).toHaveBeenCalledWith({ queryKey: ['expenseCategory'] })
        })
    })

    describe('updateExpenseMutation', () => {
        it('calls updateExpenseCategory with id and payload', async () => {
            mockedgetIncomeCategories.mockResolvedValue([])
            mockedgetExpenseCategories.mockResolvedValue([])
            mockedupdateExpenseCategory.mockResolvedValue({ id: '10', name: 'Updated Rent' })

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount
            await flushPromises()

            rendered.result.updateExpenseMutation.mutate({ id: 10, payload: { name: 'Updated Rent' } })
            await flushPromises()

            expect(mockedupdateExpenseCategory).toHaveBeenCalledWith({ id: 10, payload: { name: 'Updated Rent' } })
        })

        it('invalidates expenseCategory query on success', async () => {
            mockedgetIncomeCategories.mockResolvedValue([])
            mockedgetExpenseCategories.mockResolvedValue([])
            mockedupdateExpenseCategory.mockResolvedValue({ id: '10', name: 'Updated' })
            const spy = vi.spyOn(queryClient, 'invalidateQueries')

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount
            await flushPromises()

            rendered.result.updateExpenseMutation.mutate({ id: 10, payload: { name: 'Updated' } })
            await flushPromises()

            expect(spy).toHaveBeenCalledWith({ queryKey: ['expenseCategory'] })
        })
    })

    describe('deleteExpenseMutation', () => {
        it('calls deleteExpenseCategory with the id', async () => {
            mockedgetIncomeCategories.mockResolvedValue([])
            mockedgetExpenseCategories.mockResolvedValue([])
            mockeddeleteExpenseCategory.mockResolvedValue(undefined)

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount
            await flushPromises()

            rendered.result.deleteExpenseMutation.mutate(10)
            await flushPromises()

            expect(mockeddeleteExpenseCategory).toHaveBeenCalledWith(10)
        })

        it('invalidates expenseCategory query on success', async () => {
            mockedgetIncomeCategories.mockResolvedValue([])
            mockedgetExpenseCategories.mockResolvedValue([])
            mockeddeleteExpenseCategory.mockResolvedValue(undefined)
            const spy = vi.spyOn(queryClient, 'invalidateQueries')

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount
            await flushPromises()

            rendered.result.deleteExpenseMutation.mutate(10)
            await flushPromises()

            expect(spy).toHaveBeenCalledWith({ queryKey: ['expenseCategory'] })
        })
    })

    // ================  RETURN SHAPE  ================

    describe('return value', () => {
        it('returns all expected keys', () => {
            mockedgetIncomeCategories.mockResolvedValue([])
            mockedgetExpenseCategories.mockResolvedValue([])

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount

            const keys = Object.keys(rendered.result)
            expect(keys).toContain('incomeCategories')
            expect(keys).toContain('createIncomeCategory')
            expect(keys).toContain('updateIncomeMutation')
            expect(keys).toContain('deleteIncomeMutation')
            expect(keys).toContain('expenseCategories')
            expect(keys).toContain('createExpenseMutation')
            expect(keys).toContain('updateExpenseMutation')
            expect(keys).toContain('deleteExpenseMutation')
        })
    })
})
