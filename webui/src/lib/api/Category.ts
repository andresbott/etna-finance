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
export const GetIncomeCategories = async (): Promise<Category[]> => {
    const { data } = await apiClient.get('/fin/category/income')
    return data.items
}

export const CreateIncomeCategory = async (payload: CreateIncomeCategoryDTO): Promise<Category> => {
    const { data } = await apiClient.post('/fin/category/income', payload)
    return data
}

interface UpdateIncomeCategoryArgs {
    id: number // or string, depending on your backend
    payload: UpdateIncomeCategoryDTO
}

export const UpdateIncomeCategory = async ({
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

export const GetExpenseCategories = async (): Promise<Category[]> => {
    const { data } = await apiClient.get('/fin/category/expense')
    return data.items
}

export const CreateExpenseCategory = async (
    payload: CreateExpenseCategoryDTO
): Promise<ExpenseCategory> => {
    const { data } = await apiClient.post('/fin/category/expense', payload)
    return data
}

interface UpdateExpenseCategoryArgs {
    id: number // or string, depending on your backend
    payload: UpdateExpenseCategoryDTO
}

export const UpdateExpenseCategory = async ({
    id,
    payload
}: UpdateExpenseCategoryArgs): Promise<ExpenseCategory> => {
    const { data } = await apiClient.put(`/fin/category/expense/${id}`, payload)
    return data
}

export const DeleteExpenseCategory = async (id: number): Promise<void> => {
    await apiClient.delete(`/fin/category/expense/${id}`)
}
