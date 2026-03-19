<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import Button from 'primevue/button'
import MultiSelect from 'primevue/multiselect'
import TreeSelect from 'primevue/treeselect'
import Checkbox from 'primevue/checkbox'
import InputText from 'primevue/inputtext'
import { useCategoryTree } from '@/composables/useCategoryTree'

const categoryIds = defineModel<number[]>('categoryIds', { default: () => [] })
const types = defineModel<string[]>('types', { default: () => [] })
const hasAttachment = defineModel<boolean>('hasAttachment', { default: false })
const search = defineModel<string>('search', { default: '' })
const expanded = defineModel<boolean>('expanded', { default: false })

const props = defineProps<{ accountType?: string | null }>()

const isInvestmentAccount = computed(() =>
    props.accountType === 'investment' || props.accountType === 'unvested'
)

const hasActiveFilters = computed(() =>
    categoryIds.value.length > 0 ||
    types.value.length > 0 ||
    hasAttachment.value ||
    search.value !== ''
)

defineExpose({ hasActiveFilters })

// Type options
const defaultTypeOptions = [
    { label: 'Income', value: 'income' },
    { label: 'Expense', value: 'expense' },
    { label: 'Transfer', value: 'transfer' },
    { label: 'Investment', value: 'investment' },
    { label: 'Balance Status', value: 'balancestatus' }
]

const investmentTypeOptions = [
    { label: 'Buy', value: 'stockbuy' },
    { label: 'Sell', value: 'stocksell' },
    { label: 'Grant', value: 'stockgrant' },
    { label: 'Transfer', value: 'stocktransfer' },
]

const typeOptions = computed(() =>
    isInvestmentAccount.value ? investmentTypeOptions : defaultTypeOptions
)

// Category tree
const { IncomeTreeData, ExpenseTreeData } = useCategoryTree()

const convertTree = (nodes: any[], parentPath = ''): any[] => {
    if (!nodes || !Array.isArray(nodes)) return []
    return nodes.map((node: any) => {
        const path = parentPath ? `${parentPath} / ${node.data.name}` : node.data.name
        const converted: any = {
            key: String(node.data.id),
            label: node.data.name,
            icon: `pi ${node.data.icon || 'pi-tag'}`,
            data: { ...node.data, path }
        }
        if (node.children?.length) {
            converted.children = convertTree(node.children, path)
        }
        return converted
    })
}

const categoryTreeData = computed(() => {
    const items: any[] = []
    const expenseChildren = convertTree(ExpenseTreeData.value)
    const incomeChildren = convertTree(IncomeTreeData.value)
    if (expenseChildren.length) {
        items.push({ key: 'expense-group', label: 'Expense', selectable: false, children: expenseChildren })
    }
    if (incomeChildren.length) {
        items.push({ key: 'income-group', label: 'Income', selectable: false, children: incomeChildren })
    }
    items.push({ key: '0', label: 'Unclassified', icon: 'pi pi-question-circle' })
    return items
})

const collectAllKeys = (nodes: any[]): Record<string, boolean> => {
    const keys: Record<string, boolean> = {}
    if (!nodes) return keys
    for (const node of nodes) {
        if (node.children?.length) {
            keys[node.key] = true
            Object.assign(keys, collectAllKeys(node.children))
        }
    }
    return keys
}
const expandedKeys = computed(() => collectAllKeys(categoryTreeData.value))

// Convert between categoryIds (number[]) and TreeSelect selectionKeys
const selectionKeys = ref<Record<string, any>>({})

watch(categoryIds, (ids) => {
    const newKeys: Record<string, any> = {}
    for (const id of ids) {
        newKeys[String(id)] = { checked: true, partialChecked: false }
    }
    selectionKeys.value = newKeys
}, { immediate: true })

function onCategorySelectionChange(val: Record<string, any> | null) {
    selectionKeys.value = val || {}
    const ids: number[] = []
    if (val) {
        for (const [key, state] of Object.entries(val)) {
            if (state.checked && !isNaN(Number(key))) {
                ids.push(Number(key))
            }
        }
    }
    categoryIds.value = ids
}

const selectedCategoryLabel = computed(() => {
    const count = categoryIds.value.length
    if (count === 0) return ''
    if (count === 1) {
        // Find the name from tree
        const findLabel = (nodes: any[]): string | null => {
            for (const node of nodes) {
                if (node.key === String(categoryIds.value[0])) return node.label
                if (node.children) {
                    const found = findLabel(node.children)
                    if (found) return found
                }
            }
            return null
        }
        return findLabel(categoryTreeData.value) || '1 category'
    }
    return `${count} categories`
})

const searchInput = ref(search.value)

watch(search, (val) => {
    if (val !== searchInput.value) {
        searchInput.value = val
    }
})

function submitSearch() {
    search.value = searchInput.value
}

function clearFilters() {
    categoryIds.value = []
    types.value = []
    hasAttachment.value = false
    searchInput.value = ''
    search.value = ''
}

watch(expanded, (val) => {
    if (!val) clearFilters()
})
</script>

<template>
    <div v-if="expanded" class="filter-panel">
        <div class="filter-row">
            <InputText
                v-model="searchInput"
                placeholder="Search description/notes..."
                class="filter-input-search"
                @keydown.enter="submitSearch"
            />
            <MultiSelect
                v-model="types"
                :options="typeOptions"
                optionLabel="label"
                optionValue="value"
                placeholder="Type"
                class="filter-input"
                :showToggleAll="false"
                scrollHeight="20rem"
            />
            <TreeSelect
                v-if="!isInvestmentAccount"
                :modelValue="selectionKeys"
                @update:modelValue="onCategorySelectionChange"
                :options="categoryTreeData"
                :expandedKeys="expandedKeys"
                selectionMode="checkbox"
                placeholder="Category"
                class="filter-input-category"
                scrollHeight="400px"
                filter
                filterPlaceholder="Search categories..."
            >
                <template #value>
                    <span v-if="selectedCategoryLabel">{{ selectedCategoryLabel }}</span>
                    <span v-else>Category</span>
                </template>
            </TreeSelect>
            <div class="filter-checkbox">
                <Checkbox v-model="hasAttachment" :binary="true" inputId="hasAttachment" />
                <label for="hasAttachment">Has attachment</label>
            </div>
            <Button
                v-if="hasActiveFilters"
                label="Clear"
                severity="secondary"
                text
                size="small"
                @click="clearFilters"
            />
        </div>
    </div>
</template>

<style scoped>
.filter-panel {
    flex: 1;
}

.filter-row {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    align-items: center;
}

.filter-input {
    min-width: 10rem;
    max-width: 15rem;
}

.filter-input-category {
    min-width: 10rem;
    max-width: 18rem;
}

.filter-input-search {
    min-width: 14rem;
    max-width: 22rem;
    flex: 1;
}

.filter-checkbox {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    white-space: nowrap;
}
</style>
