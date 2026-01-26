export interface Category {
    id: string
    name: string
    description?: string
    icon?: string
}

export interface CreateIncomeCategoryDTO {
    name: string
    description?: string
    parentId?: number | null
    icon?: string
}

export interface UpdateIncomeCategoryDTO {
    name?: string
    description?: string
    parentId?: number | null
    icon?: string
}

export interface ExpenseCategory {
    id: string
    name: string
    description?: string
    icon?: string
}

export interface CreateExpenseCategoryDTO {
    name: string
    description?: string
    parentId?: number | null
    icon?: string
}

export interface UpdateExpenseCategoryDTO {
    name?: string
    description?: string
    parentId?: number | null
    icon?: string
}
