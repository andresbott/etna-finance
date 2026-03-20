<script setup>
import { ref, watch, computed } from 'vue'
import TreeSelect from 'primevue/treeselect'
import { useCategoryTree } from '@/composables/useCategoryTree'
import { findNodeById } from '@/utils/categoryUtils'

const props = defineProps({
    modelValue: { type: Number, default: 0 },
    type: { type: String, default: 'expense' }, // expense | income | all
    showRoot: { type: Boolean, default: false },
    label: { type: String, default: 'Category' },
    placeholder: { type: String, default: 'Select a category' }
})

const { IncomeTreeData, ExpenseTreeData } = useCategoryTree()
const emit = defineEmits(['update:modelValue'])

const convertTree = (nodes, parentPath = '') => {
    if (!nodes || !Array.isArray(nodes)) return []

    return nodes.map((node) => {
        const path = parentPath ? `${parentPath} / ${node.data.name}` : node.data.name

        const converted = {
            key: String(node.data.id),
            label: node.data.name,
            icon: `ti ti-${node.data.icon || 'tag'}`,
            data: { ...node.data, path }
        }

        if (node.children?.length) {
            converted.children = convertTree(node.children, path)
        }

        return converted
    })
}

const categoryTreeData = computed(() => {
    const items = []

    if (props.showRoot) {
        items.push({ key: '0', label: 'Root Category', data: { path: 'Root Category' } })
    }

    if (props.type === 'all') {
        const expenseChildren = convertTree(ExpenseTreeData.value)
        const incomeChildren = convertTree(IncomeTreeData.value)
        if (expenseChildren.length) {
            items.push({ key: 'expense-group', label: 'Expense', selectable: false, children: expenseChildren })
        }
        if (incomeChildren.length) {
            items.push({ key: 'income-group', label: 'Income', selectable: false, children: incomeChildren })
        }
    } else {
        const rawTreeData = props.type === 'expense' ? ExpenseTreeData.value : IncomeTreeData.value
        items.push(...convertTree(rawTreeData))
    }

    return items
})

// Expand all nodes by default
const collectKeys = (nodes) => {
    const keys = {}
    if (!nodes) return keys
    for (const node of nodes) {
        if (node.children?.length) {
            keys[node.key] = true
            Object.assign(keys, collectKeys(node.children))
        }
    }
    return keys
}
const expandedKeys = computed(() => collectKeys(categoryTreeData.value))

// SelectionKeys
const selectionKeys = ref({})

watch(
    () => props.modelValue,
    (newVal) => {
        if (newVal) {
            selectionKeys.value = { [String(newVal)]: { checked: true } }
        } else {
            selectionKeys.value = props.showRoot ? { 0: { checked: true } } : {}
        }
    },
    { immediate: true }
)

const onSelectionChange = (val) => {
    const selectedKey = val ? Object.keys(val)[0] : null

    if (selectedKey === '0') {
        emit('update:modelValue', 0)
        selectionKeys.value = { 0: { checked: true } }
    } else {
        emit('update:modelValue', selectedKey ? parseInt(selectedKey, 10) : 0)
        selectionKeys.value = val
    }
}

const selectedNode = computed(() => {
    if (props.modelValue === 0 || !props.modelValue) return null
    return findNodeById(categoryTreeData.value, props.modelValue)
})

const selectedCategoryDisplay = computed(() => {
    if (props.modelValue === 0 || !props.modelValue) {
        return props.showRoot ? 'Root Category' : ''
    }
    return selectedNode.value ? selectedNode.value.data.path : ''
})

const selectedCategoryIcon = computed(() => {
    return selectedNode.value?.data?.icon || 'tag'
})
</script>

<template>
    <div class="field">
        <label for="category-select">{{ label }}</label>
        <TreeSelect
            id="category-select"
            :options="categoryTreeData"
            v-model="selectionKeys"
            :expandedKeys="expandedKeys"
            :placeholder="placeholder"
            class="w-full"
            scrollHeight="400px"
            filter
            filterPlaceholder="Search categories..."
            @update:modelValue="onSelectionChange"
        >
            <template #value>
                <span v-if="selectedCategoryDisplay" class="flex items-center gap-2">
                    <i :class="['ti', `ti-${selectedCategoryIcon}`]"></i>
                    {{ selectedCategoryDisplay }}
                </span>
                <span v-else>{{ placeholder }}</span>
            </template>
        </TreeSelect>
    </div>
</template>