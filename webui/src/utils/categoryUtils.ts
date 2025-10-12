import { useCategoryTree } from '@/composables/useCategoryTree'

export interface CategoryNode {
    key: string
    label: string
    data?: {
        id: number
        parentId?: number
        name: string
        description?: string
        path: string
    }
    checked?: boolean
    children?: CategoryNode[]
}

export function findNodeById(nodes: CategoryNode[], id: number | string): CategoryNode | null {
    for (const node of nodes) {
        if (node.key === String(id) || node.data?.id === id) return node
        if (node.children) {
            const found = findNodeById(node.children, id)
            if (found) return found
        }
    }
    return null
}

export function useCategoryUtils() {
    const { IncomeTreeData, ExpenseTreeData } = useCategoryTree()

    const getCategoryName = (id: number | string, type: 'expense' | 'income') => {
        if (!id || id === 0) return 'Root'

        const nodes = type === 'expense' ? ExpenseTreeData.value : IncomeTreeData.value
        const node = findNodeById(nodes, id)

        return node ? (node.data?.name ?? node.label) : 'Unknown'
    }

    const getCategoryPath = (id: number | string, type: 'expense' | 'income') => {
        if (!id || id === 0) return 'Root'

        const nodes = type === 'expense' ? ExpenseTreeData.value : IncomeTreeData.value
        const node = findNodeById(nodes, id)

        return node ? (node.data?.path ?? node.label) : 'Unknown'
    }

    return { getCategoryName, getCategoryPath }
}
