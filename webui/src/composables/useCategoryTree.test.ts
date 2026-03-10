import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import {
    GetIncomeCategories,
    GetExpenseCategories,
} from '../lib/api/Category'
import { renderComposable, createTestQueryClient } from '../test/helpers'
import { useCategoryTree } from './useCategoryTree'
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
const mockedGetExpenseCategories = GetExpenseCategories as ReturnType<typeof vi.fn>

describe('useCategoryTree', () => {
    let unmount: () => void
    let queryClient: QueryClient

    beforeEach(() => {
        vi.clearAllMocks()
        queryClient = createTestQueryClient()
    })

    afterEach(() => {
        unmount()
    })

    describe('IncomeTreeData', () => {
        it('returns empty array when income categories have not loaded yet', () => {
            mockedGetIncomeCategories.mockReturnValue(new Promise(() => {}))
            mockedGetExpenseCategories.mockReturnValue(new Promise(() => {}))

            const rendered = renderComposable(() => useCategoryTree(), { queryClient })
            unmount = rendered.unmount

            expect(rendered.result.IncomeTreeData.value).toEqual([])
        })

        it('returns empty array when income categories data is empty', async () => {
            mockedGetIncomeCategories.mockResolvedValue([])
            mockedGetExpenseCategories.mockResolvedValue([])

            const rendered = renderComposable(() => useCategoryTree(), { queryClient })
            unmount = rendered.unmount

            await flushPromises()

            expect(rendered.result.IncomeTreeData.value).toEqual([])
        })

        it('builds tree from flat income categories', async () => {
            const categories = [
                { id: 1, name: 'Salary', parentId: null },
                { id: 2, name: 'Freelance', parentId: null },
                { id: 3, name: 'Consulting', parentId: 2 },
            ]
            mockedGetIncomeCategories.mockResolvedValue(categories)
            mockedGetExpenseCategories.mockResolvedValue([])

            const rendered = renderComposable(() => useCategoryTree(), { queryClient })
            unmount = rendered.unmount

            await flushPromises()

            const tree = rendered.result.IncomeTreeData.value
            expect(tree).toHaveLength(2)

            // First root node: Salary
            expect(tree[0].key).toBe('1')
            expect(tree[0].data.name).toBe('Salary')
            expect(tree[0].children).toBeUndefined()

            // Second root node: Freelance with child
            expect(tree[1].key).toBe('2')
            expect(tree[1].data.name).toBe('Freelance')
            expect(tree[1].children).toHaveLength(1)
            expect(tree[1].children[0].key).toBe('3')
            expect(tree[1].children[0].data.name).toBe('Consulting')
        })

        it('builds tree nodes with key and data properties for TreeTable', async () => {
            const categories = [
                { id: 10, name: 'Investments', parentId: null, description: 'Investment income' },
            ]
            mockedGetIncomeCategories.mockResolvedValue(categories)
            mockedGetExpenseCategories.mockResolvedValue([])

            const rendered = renderComposable(() => useCategoryTree(), { queryClient })
            unmount = rendered.unmount

            await flushPromises()

            const tree = rendered.result.IncomeTreeData.value
            expect(tree).toHaveLength(1)
            expect(tree[0].key).toBe('10')
            expect(tree[0].data).toEqual({
                id: 10,
                name: 'Investments',
                parentId: null,
                description: 'Investment income',
            })
        })

        it('recomputes when income categories data changes', async () => {
            mockedGetIncomeCategories.mockResolvedValue([
                { id: 1, name: 'Salary', parentId: null },
            ])
            mockedGetExpenseCategories.mockResolvedValue([])

            const rendered = renderComposable(() => useCategoryTree(), { queryClient })
            unmount = rendered.unmount

            await flushPromises()

            expect(rendered.result.IncomeTreeData.value).toHaveLength(1)

            // Simulate refetch with new data
            mockedGetIncomeCategories.mockResolvedValue([
                { id: 1, name: 'Salary', parentId: null },
                { id: 2, name: 'Bonus', parentId: null },
            ])
            await queryClient.invalidateQueries({ queryKey: ['incomeCategories'] })
            await flushPromises()

            expect(rendered.result.IncomeTreeData.value).toHaveLength(2)
        })
    })

    describe('ExpenseTreeData', () => {
        it('returns empty array when expense categories have not loaded yet', () => {
            mockedGetIncomeCategories.mockReturnValue(new Promise(() => {}))
            mockedGetExpenseCategories.mockReturnValue(new Promise(() => {}))

            const rendered = renderComposable(() => useCategoryTree(), { queryClient })
            unmount = rendered.unmount

            expect(rendered.result.ExpenseTreeData.value).toEqual([])
        })

        it('returns empty array when expense categories data is empty', async () => {
            mockedGetIncomeCategories.mockResolvedValue([])
            mockedGetExpenseCategories.mockResolvedValue([])

            const rendered = renderComposable(() => useCategoryTree(), { queryClient })
            unmount = rendered.unmount

            await flushPromises()

            expect(rendered.result.ExpenseTreeData.value).toEqual([])
        })

        it('builds tree from flat expense categories', async () => {
            const categories = [
                { id: 100, name: 'Housing', parentId: null },
                { id: 101, name: 'Rent', parentId: 100 },
                { id: 102, name: 'Utilities', parentId: 100 },
                { id: 103, name: 'Food', parentId: null },
            ]
            mockedGetIncomeCategories.mockResolvedValue([])
            mockedGetExpenseCategories.mockResolvedValue(categories)

            const rendered = renderComposable(() => useCategoryTree(), { queryClient })
            unmount = rendered.unmount

            await flushPromises()

            const tree = rendered.result.ExpenseTreeData.value
            expect(tree).toHaveLength(2)

            // Housing with 2 children
            expect(tree[0].key).toBe('100')
            expect(tree[0].data.name).toBe('Housing')
            expect(tree[0].children).toHaveLength(2)
            expect(tree[0].children[0].data.name).toBe('Rent')
            expect(tree[0].children[1].data.name).toBe('Utilities')

            // Food with no children
            expect(tree[1].key).toBe('103')
            expect(tree[1].data.name).toBe('Food')
            expect(tree[1].children).toBeUndefined()
        })

        it('recomputes when expense categories data changes', async () => {
            mockedGetIncomeCategories.mockResolvedValue([])
            mockedGetExpenseCategories.mockResolvedValue([
                { id: 100, name: 'Housing', parentId: null },
            ])

            const rendered = renderComposable(() => useCategoryTree(), { queryClient })
            unmount = rendered.unmount

            await flushPromises()

            expect(rendered.result.ExpenseTreeData.value).toHaveLength(1)

            // Simulate refetch with new data
            mockedGetExpenseCategories.mockResolvedValue([
                { id: 100, name: 'Housing', parentId: null },
                { id: 103, name: 'Food', parentId: null },
            ])
            await queryClient.invalidateQueries({ queryKey: ['expenseCategory'] })
            await flushPromises()

            expect(rendered.result.ExpenseTreeData.value).toHaveLength(2)
        })
    })

    describe('return value', () => {
        it('returns both IncomeTreeData and ExpenseTreeData', () => {
            mockedGetIncomeCategories.mockResolvedValue([])
            mockedGetExpenseCategories.mockResolvedValue([])

            const rendered = renderComposable(() => useCategoryTree(), { queryClient })
            unmount = rendered.unmount

            expect(rendered.result).toHaveProperty('IncomeTreeData')
            expect(rendered.result).toHaveProperty('ExpenseTreeData')
        })
    })
})
