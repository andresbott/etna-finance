<script setup lang="ts">
import { ref, watch } from 'vue'
import Column from 'primevue/column'
import Button from 'primevue/button'
import TreeTable from 'primevue/treetable'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { useCategories } from '@/composables/useCategories'
import { CreateIncomeCategoryDTO, UpdateIncomeCategoryDTO } from '@/types/category'
import CategoryDialog from './dialogs/CategoryDialog.vue'
import { useCategoryTree } from '@/composables/useCategoryTree'
import type { TreeTableExpandedKeys } from 'primevue/treetable'
import type { TreeNode } from 'primevue/treenode'

const {
    createIncomeCategory,
    updateIncomeMutation,
    deleteIncomeMutation
} = useCategories()

const { IncomeTreeData } = useCategoryTree()

const expandedKeys = ref<TreeTableExpandedKeys>({})

watch(
    IncomeTreeData,
    (newNodes) => {
        if (newNodes && newNodes.length > 0) {
            expandedKeys.value = expandAll(newNodes)
        }
    },
    { immediate: true }
)

function expandAll(nodes: TreeNode[]): TreeTableExpandedKeys {
    const expanded: TreeTableExpandedKeys = {}
    for (const node of nodes) {
        if (node.children && node.children.length) {
            expanded[node.key as string] = true
            Object.assign(expanded, expandAll(node.children))
        }
    }
    return expanded
}

interface Category {
    id: number | null
    parentId?: number | null
    name: string
    description: string
    type?: string
    action?: string
    icon?: string
}

const categoryDialogVisible = ref(false)
const categoryData = ref<Category>({ id: null, name: '', description: '', parentId: 0, icon: 'tag' })

const resetCategoryData = () => {
    categoryData.value = { id: null, name: '', description: '', parentId: 0, icon: 'tag' }
}

const handleAddEdit = (item: Category | null, action: string) => {
    if (action === 'add') {
        if (item != null) {
            categoryData.value.parentId = item.id
        }
        categoryData.value.action = 'add'
    } else if (action === 'edit') {
        if (item == null) {
            console.error('Something went wrong, item cannot be null for edit actions')
            return
        }
        categoryData.value.parentId = item.parentId ? item.parentId : 0
        categoryData.value.id = item.id
        categoryData.value.name = item.name
        categoryData.value.description = item.description
        categoryData.value.icon = item.icon || 'tag'
        categoryData.value.action = 'edit'
    }
    categoryData.value.type = 'income'
    categoryDialogVisible.value = true
}

const saveCategory = () => {
    if (!categoryData.value.name) return

    if (categoryData.value.action === 'add') {
        const dto: CreateIncomeCategoryDTO = {
            name: categoryData.value.name,
            description: categoryData.value.description || undefined,
            parentId: categoryData.value.parentId || undefined,
            icon: categoryData.value.icon || 'tag'
        }
        createIncomeCategory.mutate(dto, {
            onSuccess: () => { categoryDialogVisible.value = false },
            onError: (err) => console.error('Failed to add income category', err),
            onSettled: () => { resetCategoryData() }
        })
    }

    if (categoryData.value.action === 'edit') {
        if (!categoryData.value.id) {
            console.error('Something went wrong and id is null')
            return
        }
        const dto: UpdateIncomeCategoryDTO = {
            name: categoryData.value.name,
            description: categoryData.value.description || undefined,
            parentId: categoryData.value.parentId,
            icon: categoryData.value.icon || 'tag'
        }
        updateIncomeMutation.mutate(
            { id: categoryData.value.id, payload: dto },
            {
                onSuccess: () => { categoryDialogVisible.value = false },
                onError: (err) => console.error('Failed to update income category', err),
                onSettled: () => { resetCategoryData() }
            }
        )
    }
}

interface SelectDeleteCategory {
    id: number
    name: string
    type: string
}

const confirmDeleteDialog = ref(false)
const selectedItem = ref<SelectDeleteCategory | null>(null)

const confirmDelete = (item: SelectDeleteCategory) => {
    selectedItem.value = { ...item, type: 'income' }
    confirmDeleteDialog.value = true
}

const deleteCategory = () => {
    if (selectedItem.value === null) return
    deleteIncomeMutation.mutate(selectedItem.value.id, {
        onSuccess: () => {
            confirmDeleteDialog.value = false
            selectedItem.value = null
        },
        onError: (err) => { console.error('Failed to delete category:', err) }
    })
}
</script>

<template>
    <div>
        <TreeTable :value="IncomeTreeData" :expandedKeys="expandedKeys">
            <Column field="name" header="Name" expander>
                <template #body="slotProps">
                    <span class="inline-flex align-items-center gap-2">
                        <i :class="['ti', `ti-${slotProps.node.data.icon || 'tag'}`]"></i>
                        {{ slotProps.node.data.name }}
                    </span>
                </template>
            </Column>
            <Column field="description" header="Description"></Column>
            <Column>
                <template #header>
                    <div class="flex gap-1 justify-content-end w-full">
                        <Button
                            label="Add new parent category"
                            icon="ti ti-plus"
                            @click="handleAddEdit(null, 'add')"
                        />
                    </div>
                </template>
                <template #body="slotProps">
                    <div class="flex gap-1 justify-content-end w-full">
                        <Button icon="ti ti-plus" text rounded class="p-1"
                            @click="handleAddEdit(slotProps.node.data, 'add')" />
                        <Button icon="ti ti-pencil" text rounded class="p-1"
                            @click="handleAddEdit(slotProps.node.data, 'edit')" />
                        <Button icon="ti ti-trash" text rounded severity="danger" class="p-1"
                            @click="confirmDelete(slotProps.node.data)" />
                    </div>
                </template>
            </Column>
        </TreeTable>
    </div>

    <CategoryDialog
        v-model:visible="categoryDialogVisible"
        :categoryData="categoryData"
        @update:categoryData="categoryData = $event"
        @save="saveCategory"
        @reset="resetCategoryData"
    />

    <ConfirmDialog
        v-if="selectedItem"
        v-model:visible="confirmDeleteDialog"
        :name="selectedItem.name"
        title="Delete Category"
        message="Are you sure you want to delete this category?"
        @confirm="deleteCategory"
    />
</template>
