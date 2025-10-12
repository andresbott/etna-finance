import { computed } from 'vue'
import { useCategories } from '@/composables/useCategories'
import { buildTreeForTable } from '@/utils/convertToTree'

export function useCategoryTree() {
    const { incomeCategories, expenseCategories } = useCategories()

    const IncomeTreeData = computed(() => {
        if (!incomeCategories.data) return []
        return buildTreeForTable(incomeCategories.data.value)
    })

    const ExpenseTreeData = computed(() => {
        if (!expenseCategories.data) return []
        return buildTreeForTable(expenseCategories.data.value)
    })

    return {
        IncomeTreeData,
        ExpenseTreeData
    }
}
