import { apiClient } from '@/lib/api/client'
import type { Category, CreateIncomeCategoryDTO, UpdateIncomeCategoryDTO } from '@/types/category'

//* ================  INCOME  ================*//
export const getIncomeCategories = async (): Promise<Category[]> => {
    const { data } = await apiClient.get('/fin/category/income')
    return data.items
}

export const createIncomeCategory = async (payload: CreateIncomeCategoryDTO): Promise<Category> => {
    const { data } = await apiClient.post('/fin/category/income', payload)
    return data
}

export const updateIncomeCategory = async ({
    id,
    payload
}: {
    id: string
    payload: UpdateIncomeCategoryDTO
}): Promise<Category> => {
    const { data } = await apiClient.put(`/fin/category/income/${id}`, payload)
    return data
}

export const deleteIncomeCategory = async (id: string): Promise<void> => {
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

export const updateExpenseCategory = async ({
    id,
    payload
}: {
    id: string
    payload: UpdateExpenseCategoryDTO
}): Promise<ExpenseCategory> => {
    const { data } = await apiClient.put(`/fin/category/expense/${id}`, payload)
    return data
}

export const deleteExpenseCategory = async (id: string): Promise<void> => {
    await apiClient.delete(`/fin/category/expense/${id}`)
}
