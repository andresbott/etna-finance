export interface Category {
    id: string
    name: string
    description?: string
}

export interface CreateIncomeCategoryDTO {
    name: string
    description?: string
    parentId?: number | null
}

export interface UpdateIncomeCategoryDTO {
    name?: string
    description?: string
    parentId?: number | null
}

export interface ExpenseCategory {
    id: string
    name: string
    description?: string
}

export interface CreateExpenseCategoryDTO {
    name: string
    description?: string
    parentId?: number | null
}

export interface UpdateExpenseCategoryDTO {
    name?: string
    description?: string
    parentId?: number | null
}
