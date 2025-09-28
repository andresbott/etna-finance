<script setup>
import { ref, watch, computed } from 'vue'
import TreeSelect from 'primevue/treeselect'
import { useCategoryTree } from '@/composables/useCategoryTree'
import { findNodeById } from '@/utils/categoryUtils'

const props = defineProps({
    modelValue: { type: Number, default: 0 }, // parentId
    type: { type: String, default: 'expense' } // expense | income
})

const { IncomeTreeData, ExpenseTreeData } = useCategoryTree()
const emit = defineEmits(['update:modelValue'])

// Pick correct tree
const rawTreeData = computed(() => {
    return props.type === 'expense' ? ExpenseTreeData.value : IncomeTreeData.value
})

// Convert raw nodes → TreeSelect format
const convertTree = (nodes, parentKey = '0', parentPath = '') => {
    if (!nodes || !Array.isArray(nodes)) return []

    return nodes.map((node, index) => {
        const currentKey = `${parentKey}-${index}`
        const path = parentPath ? `${parentPath} / ${node.data.name}` : node.data.name

        const converted = {
            key: String(node.data.id),
            label: node.data.name,
            data: { ...node.data, path }
        }

        if (node.children?.length) {
            converted.children = convertTree(node.children, currentKey, path)
        }

        return converted
    })
}

const categoryTreeData = computed(() => {
    return [{ key: '0', label: 'Root Category', checked: true }, ...convertTree(rawTreeData.value)]
})

// SelectionKeys
const selectionKeys = ref({})

// Sync when parentId changes
watch(
    () => props.modelValue,
    (newVal) => {
        if (newVal) {
            selectionKeys.value = { [String(newVal)]: { checked: true } }
        } else {
            selectionKeys.value = { 0: { checked: true } }
        }
    },
    { immediate: true }
)

// Handle changes
const onSelectionChange = (val) => {
    const selectedKey = val ? Object.keys(val)[0] : null

    if (selectedKey === 0) {
        emit('update:modelValue', 0)
        selectionKeys.value = { 0: { checked: true } }
    } else {
        emit('update:modelValue', selectedKey ? parseInt(selectedKey, 10) : 0)
        selectionKeys.value = val
    }
}

const selectedCategoryDisplay = computed(() => {
    if (props.modelValue === 0 || !props.modelValue) return 'Root Category'
    const selectedNode = findNodeById(categoryTreeData.value, props.modelValue)

    return selectedNode ? selectedNode.data.path : ''
})
</script>

<template>
    <div class="field">
        <label for="category-parent">Parent Category</label>
        <TreeSelect
            id="category-parent"
            :options="categoryTreeData"
            v-model="selectionKeys"
            placeholder="Select parent category"
            class="w-full"
            @update:modelValue="onSelectionChange"
        >
            <template #value>
                <span v-if="selectedCategoryDisplay">{{ selectedCategoryDisplay }}</span>
                <span v-else>Select parent category</span>
            </template>
        </TreeSelect>
    </div>
</template>
