import { apiClient } from '@/lib/api/client'
import type {
    Category,
    CreateIncomeCategoryDTO,
    UpdateIncomeCategoryDTO,
    CreateExpenseCategoryDTO,
    UpdateExpenseCategoryDTO,
    ExpenseCategory
} from '@/types/category'

//* ================  INCOME  ================*//
export const getIncomeCategories = async (): Promise<Category[]> => {
    const { data } = await apiClient.get('/fin/category/income')
    return data.items
}

export const createIncomeCategory = async (payload: CreateIncomeCategoryDTO): Promise<Category> => {
    const { data } = await apiClient.post('/fin/category/income', payload)
    return data
}

interface UpdateIncomeCategoryArgs {
    id: number // or string, depending on your backend
    payload: UpdateIncomeCategoryDTO
}

export const updateIncomeCategory = async ({
    id,
    payload
}: UpdateIncomeCategoryArgs): Promise<Category> => {
    const { data } = await apiClient.put(`/fin/category/income/${id}`, payload)
    return data
}

export const deleteIncomeCategory = async (id: number): Promise<void> => {
    await apiClient.delete(`/fin/category/income/${id}`)
}

//* ================  Expense  ================*//

export const getExpenseCategories = async (): Promise<Category[]> => {
    const { data } = await apiClient.get('/fin/category/expense')
    return data.items
}

export const createExpenseCategory = async (
    payload: CreateExpenseCategoryDTO
): Promise<ExpenseCategory> => {
    const { data } = await apiClient.post('/fin/category/expense', payload)
    return data
}

interface UpdateExpenseCategoryArgs {
    id: number // or string, depending on your backend
    payload: UpdateExpenseCategoryDTO
}

export const updateExpenseCategory = async ({
    id,
    payload
}: UpdateExpenseCategoryArgs): Promise<ExpenseCategory> => {
    const { data } = await apiClient.put(`/fin/category/expense/${id}`, payload)
    return data
}

export const deleteExpenseCategory = async (id: number): Promise<void> => {
    await apiClient.delete(`/fin/category/expense/${id}`)
}
