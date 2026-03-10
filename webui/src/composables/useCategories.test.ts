import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import {
    GetIncomeCategories,
    CreateIncomeCategory,
    UpdateIncomeCategory,
    deleteIncomeCategory,
    GetExpenseCategories,
    CreateExpenseCategory,
    UpdateExpenseCategory,
    DeleteExpenseCategory,
} from '../lib/api/Category'
import { renderComposable, createTestQueryClient } from '../test/helpers'
import { useCategories } from './useCategories'
import type { QueryClient } from '@tanstack/vue-query'

vi.mock('../lib/api/Category', () => ({
    GetIncomeCategories: vi.fn(),
    CreateIncomeCategory: vi.fn(),
    UpdateIncomeCategory: vi.fn(),
    deleteIncomeCategory: vi.fn(),
    GetExpenseCategories: vi.fn(),
    CreateExpenseCategory: vi.fn(),
    UpdateExpenseCategory: vi.fn(),
    DeleteExpenseCategory: vi.fn(),
}))

const mockedGetIncomeCategories = GetIncomeCategories as ReturnType<typeof vi.fn>
const mockedCreateIncomeCategory = CreateIncomeCategory as ReturnType<typeof vi.fn>
const mockedUpdateIncomeCategory = UpdateIncomeCategory as ReturnType<typeof vi.fn>
const mockedDeleteIncomeCategory = deleteIncomeCategory as ReturnType<typeof vi.fn>
const mockedGetExpenseCategories = GetExpenseCategories as ReturnType<typeof vi.fn>
const mockedCreateExpenseCategory = CreateExpenseCategory as ReturnType<typeof vi.fn>
const mockedUpdateExpenseCategory = UpdateExpenseCategory as ReturnType<typeof vi.fn>
const mockedDeleteExpenseCategory = DeleteExpenseCategory as ReturnType<typeof vi.fn>

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
            mockedGetIncomeCategories.mockResolvedValue(categories)

            const { result } = renderComposable(() => useCategories(), { queryClient })
            unmount = renderComposable(() => useCategories(), { queryClient }).unmount
            // Re-render properly
            unmount()
            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount

            await flushPromises()

            expect(mockedGetIncomeCategories).toHaveBeenCalled()
            expect(rendered.result.incomeCategories.data.value).toEqual(categories)
        })

        it('exposes loading state while fetching income categories', () => {
            mockedGetIncomeCategories.mockReturnValue(new Promise(() => {})) // never resolves

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount

            expect(rendered.result.incomeCategories.isLoading.value).toBe(true)
        })

        it('exposes error when income categories fetch fails', async () => {
            mockedGetIncomeCategories.mockRejectedValue(new Error('Network error'))

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount

            await flushPromises()

            expect(rendered.result.incomeCategories.isError.value).toBe(true)
            expect(rendered.result.incomeCategories.error.value).toBeInstanceOf(Error)
        })
    })

    // ================  INCOME MUTATIONS  ================

    describe('createIncomeCategory mutation', () => {
        it('calls CreateIncomeCategory with the payload', async () => {
            mockedGetIncomeCategories.mockResolvedValue([])
            const newCategory = { id: '3', name: 'Bonus' }
            mockedCreateIncomeCategory.mockResolvedValue(newCategory)

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount
            await flushPromises()

            const payload = { name: 'Bonus' }
            rendered.result.createIncomeCategory.mutate(payload)
            await flushPromises()

            expect(mockedCreateIncomeCategory).toHaveBeenCalledWith(payload)
        })

        it('invalidates incomeCategories query on success', async () => {
            mockedGetIncomeCategories.mockResolvedValue([])
            mockedCreateIncomeCategory.mockResolvedValue({ id: '3', name: 'Bonus' })
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
        it('calls UpdateIncomeCategory with id and payload', async () => {
            mockedGetIncomeCategories.mockResolvedValue([])
            mockedUpdateIncomeCategory.mockResolvedValue({ id: '1', name: 'Updated' })

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount
            await flushPromises()

            rendered.result.updateIncomeMutation.mutate({ id: 1, payload: { name: 'Updated' } })
            await flushPromises()

            expect(mockedUpdateIncomeCategory).toHaveBeenCalledWith({ id: 1, payload: { name: 'Updated' } })
        })

        it('invalidates incomeCategories query on success', async () => {
            mockedGetIncomeCategories.mockResolvedValue([])
            mockedUpdateIncomeCategory.mockResolvedValue({ id: '1', name: 'Updated' })
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
            mockedGetIncomeCategories.mockResolvedValue([])
            mockedDeleteIncomeCategory.mockResolvedValue(undefined)

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount
            await flushPromises()

            rendered.result.deleteIncomeMutation.mutate(5)
            await flushPromises()

            expect(mockedDeleteIncomeCategory).toHaveBeenCalledWith(5)
        })

        it('invalidates incomeCategories query on success', async () => {
            mockedGetIncomeCategories.mockResolvedValue([])
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
            mockedGetExpenseCategories.mockResolvedValue(categories)
            mockedGetIncomeCategories.mockResolvedValue([])

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount

            await flushPromises()

            expect(mockedGetExpenseCategories).toHaveBeenCalled()
            expect(rendered.result.expenseCategories.data.value).toEqual(categories)
        })

        it('exposes loading state while fetching expense categories', () => {
            mockedGetExpenseCategories.mockReturnValue(new Promise(() => {}))
            mockedGetIncomeCategories.mockReturnValue(new Promise(() => {}))

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount

            expect(rendered.result.expenseCategories.isLoading.value).toBe(true)
        })

        it('exposes error when expense categories fetch fails', async () => {
            mockedGetExpenseCategories.mockRejectedValue(new Error('Server error'))
            mockedGetIncomeCategories.mockResolvedValue([])

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount

            await flushPromises()

            expect(rendered.result.expenseCategories.isError.value).toBe(true)
        })
    })

    // ================  EXPENSE MUTATIONS  ================

    describe('createExpenseMutation', () => {
        it('calls CreateExpenseCategory with the payload', async () => {
            mockedGetIncomeCategories.mockResolvedValue([])
            mockedGetExpenseCategories.mockResolvedValue([])
            mockedCreateExpenseCategory.mockResolvedValue({ id: '12', name: 'Transport' })

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount
            await flushPromises()

            const payload = { name: 'Transport' }
            rendered.result.createExpenseMutation.mutate(payload)
            await flushPromises()

            expect(mockedCreateExpenseCategory).toHaveBeenCalledWith(payload)
        })

        it('invalidates expenseCategory query on success', async () => {
            mockedGetIncomeCategories.mockResolvedValue([])
            mockedGetExpenseCategories.mockResolvedValue([])
            mockedCreateExpenseCategory.mockResolvedValue({ id: '12', name: 'Transport' })
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
        it('calls UpdateExpenseCategory with id and payload', async () => {
            mockedGetIncomeCategories.mockResolvedValue([])
            mockedGetExpenseCategories.mockResolvedValue([])
            mockedUpdateExpenseCategory.mockResolvedValue({ id: '10', name: 'Updated Rent' })

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount
            await flushPromises()

            rendered.result.updateExpenseMutation.mutate({ id: 10, payload: { name: 'Updated Rent' } })
            await flushPromises()

            expect(mockedUpdateExpenseCategory).toHaveBeenCalledWith({ id: 10, payload: { name: 'Updated Rent' } })
        })

        it('invalidates expenseCategory query on success', async () => {
            mockedGetIncomeCategories.mockResolvedValue([])
            mockedGetExpenseCategories.mockResolvedValue([])
            mockedUpdateExpenseCategory.mockResolvedValue({ id: '10', name: 'Updated' })
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
        it('calls DeleteExpenseCategory with the id', async () => {
            mockedGetIncomeCategories.mockResolvedValue([])
            mockedGetExpenseCategories.mockResolvedValue([])
            mockedDeleteExpenseCategory.mockResolvedValue(undefined)

            const rendered = renderComposable(() => useCategories(), { queryClient })
            unmount = rendered.unmount
            await flushPromises()

            rendered.result.deleteExpenseMutation.mutate(10)
            await flushPromises()

            expect(mockedDeleteExpenseCategory).toHaveBeenCalledWith(10)
        })

        it('invalidates expenseCategory query on success', async () => {
            mockedGetIncomeCategories.mockResolvedValue([])
            mockedGetExpenseCategories.mockResolvedValue([])
            mockedDeleteExpenseCategory.mockResolvedValue(undefined)
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
            mockedGetIncomeCategories.mockResolvedValue([])
            mockedGetExpenseCategories.mockResolvedValue([])

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
