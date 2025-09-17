<script setup>
import { ref, watch, computed } from 'vue'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import TreeSelect from 'primevue/treeselect'

const props = defineProps({
    visible: Boolean,
    categoryData: Object,
    expenseTreeData: { type: Array, default: () => [] },
    incomeTreeData: { type: Array, default: () => [] }
})

const emit = defineEmits([
    'update:visible',
    'save',
    'reset',
    'update:categoryParentId',
    'update:categoryData'
])

// Local category (avoid mutating props directly)
const localCategory = ref({
    id: null,
    name: '',
    description: '',
    parentId: null,
    type: 'expense',
    action: null,
    ...props.categoryData
})

watch(
    () => props.categoryData,
    (newVal) => {
        if (newVal) {
            localCategory.value = {
                id: null,
                name: '',
                description: '',
                parentId: null,
                type: 'expense',
                action: null,
                ...newVal
            }
        }
    },
    { immediate: true, deep: true }
)

// Pick correct tree (expense or income)
const rawTreeData = computed(() => {
    return localCategory.value?.type === 'expense' ? props.expenseTreeData : props.incomeTreeData
})

// Convert raw nodes → PrimeVue TreeSelect format
const convertTree = (nodes, parentKey = '0', parentPath = '') => {
    if (!nodes || !Array.isArray(nodes)) return []

    return nodes.map((node, index) => {
        const currentKey = `${parentKey}-${index}`
        const path = parentPath ? `${parentPath} / ${node.data.name}` : node.data.name

        const converted = {
            key: String(node.data.id),
            label: `${node.data.name}`,
            data: { ...node.data, path }
        }

        if (node.children && node.children.length > 0) {
            converted.children = convertTree(node.children, currentKey, path)
        }

        return converted
    })
}

const categoryTreeData = computed(() => convertTree(rawTreeData.value))

const selectionKeys = ref({})

// Sync selection when parentId changes
watch(
    () => localCategory.value.parentId,
    (newParentId) => {
        if (newParentId) {
            selectionKeys.value = { [String(newParentId)]: { checked: true } }
        } else {
            selectionKeys.value = {}
        }
    },
    { immediate: true }
)

// Handle TreeSelect updates
const onSelectionChange = (val) => {
    const selectedKey = val ? Object.keys(val)[0] : null
    localCategory.value.parentId = selectedKey ? parseInt(selectedKey, 10) : null
    emit('update:categoryParentId', localCategory.value.parentId)
}

// Get selected category display text with path
const selectedCategoryDisplay = computed(() => {
    if (!localCategory.value.parentId || !categoryTreeData.value.length) {
        return ''
    }

    // Find the selected node recursively
    const findNodeById = (nodes, id) => {
        for (const node of nodes) {
            if (node.key === String(id)) {
                return node
            }
            if (node.children) {
                const found = findNodeById(node.children, id)
                if (found) return found
            }
        }
        return null
    }

    const selectedNode = findNodeById(categoryTreeData.value, localCategory.value.parentId)
    return selectedNode ? `${selectedNode.data.path}` : ''
})

// Get dialog headeer title
const titleMap = {
    income: 'Income Category',
    expense: 'Expense Category'
}
const dialogHeaderTitle = computed(() => {
    const action = props.categoryData.action == 'edit' ? 'Edit' : 'Add New'
    const type = titleMap[props.categoryData.type]
    return `${action} ${type}`
})

/* ---------------------------
   Actions
---------------------------- */
const save = () => {
    emit('update:categoryData', localCategory.value)
    emit('save', localCategory.value)
    emit('update:visible', false)
}

const reset = () => {
    localCategory.value = {
        id: null,
        name: '',
        description: '',
        parentId: null,
        type: 'expense',
        action: null
    }
    selectionKeys.value = {}
    emit('reset')
}

const cancel = () => {
    reset()
    emit('update:visible', false)
}

// Show parent selector condition
const showParentSelector = computed(() => {
    return (
        localCategory.value &&
        localCategory.value.parentId != undefined &&
        localCategory.value.action !== 'add'
    )
})
</script>

<template>
    <Dialog
        :visible="visible"
        modal
        :header="dialogHeaderTitle"
        :draggable="false"
        @update:visible="emit('update:visible', $event)"
        @hide="cancel"
    >
        <div>
            <!-- Name -->
            <div class="field">
                <label for="category-name">Name</label>
                <InputText id="category-name" v-model="localCategory.name" class="w-full" />
            </div>

            <!-- Description -->
            <div class="field">
                <label for="category-description">Description</label>
                <InputText
                    id="category-description"
                    v-model="localCategory.description"
                    class="w-full"
                />
            </div>

            <!-- Parent selector -->
            <div v-if="showParentSelector" class="field">
                <label for="category-parent">Parent Category</label>

                <TreeSelect
                    id="category-parent"
                    :options="categoryTreeData"
                    v-model="selectionKeys"
                    placeholder="Select parent category"
                    class="w-full"
                    @update:modelValue="onSelectionChange"
                >
                    <template #value="slotProps">
                        <span v-if="selectedCategoryDisplay">{{ selectedCategoryDisplay }}</span>
                        <span v-else>Select parent category</span>
                    </template>
                </TreeSelect>
            </div>
        </div>

        <template #footer>
            <Button label="Save" icon="pi pi-check" @click="save" :disabled="!localCategory.name" />
            <Button label="Cancel" icon="pi pi-times" text @click="cancel" />
        </template>
    </Dialog>
</template>
