import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import {
    getIncomeCategories,
    createIncomeCategory,
    deleteIncomeCategory,
    updateIncomeCategory,
    getExpenseCategories,
    createExpenseCategory,
    updateExpenseCategory,
    deleteExpenseCategory
} from '@/lib/api/Category'
import type {
    CreateIncomeCategoryDTO,
    UpdateIncomeCategoryDTO,
    CreateExpenseCategoryDTO,
    UpdateExpenseCategoryDTO
} from '@/types/category'

export function useCategories() {
    const queryClient = useQueryClient()

    //* ================  INCOME  ================*//

    const incomeCategoriesQuery = useQuery({
        queryKey: ['incomeCategories'],
        queryFn: getIncomeCategories
    })

    // CREATE category
    const createIncomeMutation = useMutation({
        mutationFn: (payload: CreateIncomeCategoryDTO) => createIncomeCategory(payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['incomeCategories'] })
        }
    })

    // UPDATE category
    interface updateIncomeCategoryArgs {
        id: number // or string, depending on your backend
        payload: UpdateIncomeCategoryDTO
    }
    const updateIncomeMutation = useMutation({
        mutationFn: ({ id, payload }: updateIncomeCategoryArgs) =>
            updateIncomeCategory({ id: id, payload: payload }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['incomeCategories'] })
        }
    })

    // DELETE category
    const deleteIncomeMutation = useMutation({
        mutationFn: (id: number) => deleteIncomeCategory(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['incomeCategories'] })
        }
    })

    //* ================  Expense  ================*//

    const expenseCategoryQuery = useQuery({
        queryKey: ['expenseCategory'],
        queryFn: getExpenseCategories
    })

    // CREATE category
    const createExpenseMutation = useMutation({
        mutationFn: (payload: CreateExpenseCategoryDTO) => createExpenseCategory(payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['expenseCategory'] })
        }
    })

    // UPDATE category
    interface updateExpenseCategoryArgs {
        id: number // or string, depending on your backend
        payload: UpdateExpenseCategoryDTO
    }

    const updateExpenseMutation = useMutation({
        mutationFn: ({ id, payload }: updateExpenseCategoryArgs) =>
            updateExpenseCategory({ id: id, payload: payload }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['expenseCategory'] })
        }
    })

    // DELETE category
    const deleteExpenseMutation = useMutation({
        mutationFn: (id: number) => deleteExpenseCategory(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['expenseCategory'] })
        }
    })

    return {
        incomeCategories: incomeCategoriesQuery,
        createIncomeCategory: createIncomeMutation,
        updateIncomeMutation: updateIncomeMutation,
        deleteIncomeMutation: deleteIncomeMutation,

        expenseCategories: expenseCategoryQuery,
        createExpenseMutation: createExpenseMutation,
        updateExpenseMutation: updateExpenseMutation,
        deleteExpenseMutation: deleteExpenseMutation
    }
}
