import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import {
    GetIncomeCategories,
    CreateIncomeCategory,
    deleteIncomeCategory,
    UpdateIncomeCategory,
    GetExpenseCategories,
    CreateExpenseCategory,
    UpdateExpenseCategory,
    DeleteExpenseCategory
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
        queryFn: GetIncomeCategories
    })

    // CREATE category
    const createIncomeMutation = useMutation({
        mutationFn: (payload: CreateIncomeCategoryDTO) => CreateIncomeCategory(payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['incomeCategories'] })
        }
    })

    // UPDATE category
    interface UpdateIncomeCategoryArgs {
        id: number // or string, depending on your backend
        payload: UpdateIncomeCategoryDTO
    }
    const updateIncomeMutation = useMutation({
        mutationFn: ({ id, payload }: UpdateIncomeCategoryArgs) =>
            UpdateIncomeCategory({ id: id, payload: payload }),
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
        queryFn: GetExpenseCategories
    })

    // CREATE category
    const createExpenseMutation = useMutation({
        mutationFn: (payload: CreateExpenseCategoryDTO) => CreateExpenseCategory(payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['expenseCategory'] })
        }
    })

    // UPDATE category
    interface UpdateExpenseCategoryArgs {
        id: number // or string, depending on your backend
        payload: UpdateExpenseCategoryDTO
    }

    const updateExpenseMutation = useMutation({
        mutationFn: ({ id, payload }: UpdateExpenseCategoryArgs) =>
            UpdateExpenseCategory({ id: id, payload: payload }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['expenseCategory'] })
        }
    })

    // DELETE category
    const deleteExpenseMutation = useMutation({
        mutationFn: (id: number) => DeleteExpenseCategory(id),
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
