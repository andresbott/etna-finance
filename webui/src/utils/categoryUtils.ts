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

export function buildCategoryPath(nodes: CategoryNode[], id: number | string): string {
    const node = findNodeById(nodes, id)
    if (!node) return 'Unknown'
    
    // If path exists in data, use it
    if (node.data?.path) return node.data.path
    
    // Otherwise build the path by traversing up
    const path: string[] = []
    let currentNode: CategoryNode | null = node
    
    // Build path from current node up to root
    const visited = new Set<string>()
    while (currentNode && !visited.has(currentNode.key)) {
        visited.add(currentNode.key)
        path.unshift(currentNode.data?.name || currentNode.label)
        
        // Find parent
        if (currentNode.data?.parentId) {
            currentNode = findNodeById(nodes, currentNode.data.parentId)
        } else {
            break
        }
    }
    
    return path.join(' > ')
}

export function useCategoryUtils() {
    const { IncomeTreeData, ExpenseTreeData } = useCategoryTree()

    const getCategoryName = (id: number | string, type: 'expense' | 'income') => {
        if (id === 0) return 'Root'
        if (!id) return '-'


        const nodes = type === 'expense' ? ExpenseTreeData.value : IncomeTreeData.value
        const node = findNodeById(nodes, id)

        return node ? (node.data?.name ?? node.label) : 'Unknown'
    }

    const getCategoryPath = (id: number | string, type: 'expense' | 'income') => {
        if (!id || id === 0) return 'Root'

        const nodes = type === 'expense' ? ExpenseTreeData.value : IncomeTreeData.value
        return buildCategoryPath(nodes, id)
    }

    return { getCategoryName, getCategoryPath }
}
